package imt

import (
	"encoding/binary"
	"errors"
	"imt/db"
	"math/big"
)

const (
	prefixNodeKey = byte(0)
	prefixHashKey = byte(1)
	prefixSizeKey = byte(2)
)

var sizeKey = []byte{prefixSizeKey}

type HashFn func([]*big.Int) (*big.Int, error)

type Tree struct {
	// The database interface handle to store the tree state.
	db db.Database

	// The field element length.
	feLen uint8

	// The last tree level that stores leafs
	leafLevel uint8

	// The hash function to use to hash siblings.
	hashFn HashFn
}

// Instanciate a new tree.
func NewTree(db db.Database, feLen, leafLevel uint8, hashFn HashFn) *Tree {
	return &Tree{
		db:        db,
		feLen:     feLen,
		leafLevel: leafLevel,
		hashFn:    hashFn,
	}
}

// Return the tree root hash.
// The returned hash is the hash of the root hash and the tree size.
func (t *Tree) Root() (*big.Int, error) {
	// Get the root hash from the database.
	rootHash, err := t.db.Get(t.hashKey(0, 0))

	// If the root hash does not exist default to an empty node hash.
	if errors.Is(err, db.ErrNotFound) {
		rootHashBn, err := emptyNode().hash(t.hashFn)
		if err != nil {
			return nil, err
		}
		rootHash = rootHashBn.Bytes()
	}

	if err != nil {
		return nil, err
	}

	// Return the hash of the root hash and the tree size.
	size, err := t.size()
	if err != nil {
		return nil, err
	}

	return t.hashFn([]*big.Int{new(big.Int).SetBytes(rootHash), new(big.Int).SetUint64(size)})
}

// Inserts a new `value` in the tree at the given `key`.
func (t *Tree) Insert(key, value *big.Int) error {
	// Ensure the key does not already exist.
	_, err := t.db.Get(t.nodeKey(key))
	if err == nil {
		return errors.New("key already exists")
	} else if !errors.Is(err, db.ErrNotFound) {
		return err
	}

	// Lookup the low nullifier node.
	lnKey, lnNode, err := t.lowNullifierNode(key)
	if err != nil {
		return err
	}

	// Update the tree size and register it in the database.
	size, err := t.size()
	if err != nil {
		return err
	}
	size += 1

	if err := t.setSize(size); err != nil {
		return err
	}

	// Create the new node and register it in the database.
	// NOTE: `size` is incremented first so the 1st real node is at index 1 as expected.
	// 		 The 0 index is reserved.
	newNode := &Node{Index: size, Value: value, NextKey: lnNode.NextKey}
	if _, err := t.setNode(key, newNode); err != nil {
		return err
	}

	// Update the low nullifier node and save it in the database.
	lnNode.NextKey = key
	if _, err := t.setNode(lnKey, lnNode); err != nil {
		return err
	}

	return nil
}

// Returns the key to store a node.
func (t *Tree) nodeKey(key *big.Int) []byte {
	b := key.Bytes()
	prefix := make([]byte, 1+int(t.feLen)-len(b))
	prefix[0] = prefixNodeKey
	return append(prefix, b...)
}

// Returns the key to store a hash for the `level` and `index` pair.
// NOTE: The `index` correspond to the hash index within the level.
func (t *Tree) hashKey(level uint8, index uint64) []byte {
	prefix := make([]byte, 1+8)
	prefix[0] = prefixHashKey
	prefix[1] = level
	return binary.BigEndian.AppendUint64(prefix, index)
}

// Read the size of the tree from the database. Returns 0 if the `sizeKey` is
// not yet registered.
func (t *Tree) size() (uint64, error) {
	size, err := t.db.Get(sizeKey)
	if errors.Is(err, db.ErrNotFound) {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	return new(big.Int).SetBytes(size).Uint64(), nil
}

// Fecth the low nuliffier node for the given `key`.
func (t *Tree) lowNullifierNode(key *big.Int) (*big.Int, *Node, error) {
	// Fetch the tree size from the database.
	size, err := t.size()
	if err != nil {
		return nil, nil, err
	}

	// If the size of the tree is empty return the 0 index node.
	if size == 0 {
		lnKey := new(big.Int).SetBytes(t.nodeKey(big.NewInt(0)))
		return lnKey, emptyNode(), nil
	}

	// Ensure no error and non-empty node bytes.
	lnKeyBytes, lnNodeBytes, err := t.db.GetLT(t.nodeKey(key))
	if err != nil {
		return nil, nil, err
	}

	if lnNodeBytes == nil {
		return nil, nil, errors.New("unexpected nil low nullifier")
	}

	// Return the low nullifier key and node.
	lnKey := new(big.Int).SetBytes(t.nodeKey(new(big.Int).SetBytes(lnKeyBytes)))
	lnNode := &Node{}
	lnNode.fromBytes(lnNodeBytes)

	return lnKey, lnNode, nil
}

// Sets a node in the tree.
func (t *Tree) setNode(key *big.Int, n *Node) ([]*big.Int, error) {
	// Register the new node in the database
	if err := t.db.Set(t.nodeKey(key), n.bytes()); err != nil {
		return nil, err
	}

	// Hash the node.
	h, err := n.hash(t.hashFn)
	if err != nil {
		return nil, err
	}

	// Register the node's hash.
	if err := t.db.Set(t.hashKey(t.leafLevel, n.Index), h.Bytes()); err != nil {
		return nil, err
	}

	// Update the hashes up to the root.
	siblingHashes := make([]*big.Int, t.leafLevel)
	index := n.Index
	for level := t.leafLevel; level > 0; {
		siblingIndex := index + 1 - (index%2)*2

		// Fetch the sibling node hash from the Database
		siblingHashBytes, err := t.db.Get(t.hashKey(level, siblingIndex))
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}

		siblingHash := new(big.Int).SetBytes(siblingHashBytes)

		// Compute the merged hash.
		if index%2 == 0 {
			h, err = t.hashFn([]*big.Int{h, siblingHash})
		} else {
			h, err = t.hashFn([]*big.Int{siblingHash, h})
		}
		if err != nil {
			return nil, err
		}

		// Climb up in the tree.
		level--
		index = index / 2

		// Save the sibling hash.
		siblingHashes[level] = siblingHash

		if level == 0 && index != 0 {
			return nil, errors.New("tree is over capacity")
		}

		// Register the merged hash
		if err := t.db.Set(t.hashKey(level, index), h.Bytes()); err != nil {
			return nil, err
		}

	}

	return siblingHashes, nil
}

// Store the tree size in the database.
func (t *Tree) setSize(s uint64) error {
	return t.db.Set(sizeKey, new(big.Int).SetUint64(s).Bytes())
}
