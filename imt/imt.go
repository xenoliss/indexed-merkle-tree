package imt

import (
	"encoding/binary"
	"errors"
	"fmt"
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

// Set sets the `value` for the given `key` in the tree. It returns a `MutateProof` corresponding
// to the state update.
func (t *Tree) Set(key, value *big.Int) (*MutateProof, error) {
	// Check if the key already exists and if not set the `isInsertion` flag to true.
	nodeBytes, err := t.db.Get(t.nodeKey(key))

	// Return on unexpected errors.
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}

	// If no error is returned, `nodeBytes` must not be nil.
	// NOTE: This is a safe guard but it should never happen due to how `db.Database.Get` is implemented.
	isInsertion := nodeBytes == nil
	if err == nil && isInsertion {
		return nil, fmt.Errorf("unexpected nil value for %v", key)
	}

	// Lookup the low nullifier node.
	lnKey, lnNode, lnPreUpdateProof, err := t.lowNullifierNode(key)
	if err != nil {
		return nil, err
	}

	// Build the node to set.
	// NOTE: For insertion leave the default `Index` as it will be set to the updated tree size in `setNode`.
	var node *Node
	if isInsertion {
		node = &Node{Value: value, NextKey: lnNode.NextKey}
	} else {
		node, err = nodeFromBytes(nodeBytes)
		if err != nil {
			return nil, err
		}

		node.Value = value
	}

	// Set the node.
	nodeProof, err := t.setNode(key, node, isInsertion)
	if err != nil {
		return nil, err
	}

	// Update the low nullifier node and save it in the database.
	lnNode.NextKey = key
	lnPostUpdateProof, err := t.setNode(lnKey, lnNode, false)
	if err != nil {
		return nil, err
	}

	return &MutateProof{LnPreUpdateProof: lnPreUpdateProof, NodeProof: nodeProof, LnPostUpdateProof: lnPostUpdateProof}, nil
}

// Return the tree root hash.
func (t *Tree) root() (*big.Int, error) {
	// Get the root hash from the database.
	rootHash, err := t.db.Get(t.hashKey(0, 0))

	// If the root hash does not exist default to an empty node hash.
	if errors.Is(err, db.ErrNotFound) {
		rootHashBn, err := emptyNode().hash(t.hashFn)
		if err != nil {
			return nil, err
		}
		rootHash = rootHashBn.Bytes()
	} else if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(rootHash), nil
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

// size returns the size of the tree from the database.
// Returns 0 if the `sizeKey` is not yet registered.
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

// lowNullifierNode fecths the low nuliffier node for the given `key`.
// It return the low nuliffier key and node and the `Proof` for it.
func (t *Tree) lowNullifierNode(key *big.Int) (*big.Int, *Node, *Proof, error) {
	// Fetch the tree size from the database.
	size, err := t.size()
	if err != nil {
		return nil, nil, nil, err
	}

	// If the size of the tree is empty return the 0 index node.
	if size == 0 {
		lnKey := new(big.Int).SetBytes(t.nodeKey(big.NewInt(0)))
		lnNode := emptyNode()

		// Build the ln node proof.
		proof, err := t.nodeProof(lnNode)
		if err != nil {
			return nil, nil, nil, err
		}

		return lnKey, lnNode, proof, nil
	}

	// Ensure no error and non-empty node bytes.
	lnKeyBytes, lnNodeBytes, err := t.db.GetLT(t.nodeKey(key))
	if err != nil {
		return nil, nil, nil, err
	}

	if lnNodeBytes == nil {
		return nil, nil, nil, errors.New("unexpected nil low nullifier")
	}

	// Return the low nullifier key and node.
	lnKey := new(big.Int).SetBytes(t.nodeKey(new(big.Int).SetBytes(lnKeyBytes)))
	lnNode, err := nodeFromBytes(lnNodeBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	// Build the ln node proof.
	proof, err := t.nodeProof(lnNode)
	if err != nil {
		return nil, nil, nil, err
	}

	return lnKey, lnNode, proof, nil
}

// Sets a node in the tree.
// Returns a `Proof` fof the given node.
func (t *Tree) setNode(key *big.Int, node *Node, isInstertion bool) (*Proof, error) {
	size, err := t.size()
	if err != nil {
		return nil, err
	}

	// In case of insertion, increase the size of the tree and set the node `Index` to the it.
	if isInstertion {
		size += 1
		if err := t.setSize(size); err != nil {
			return nil, err
		}

		node.Index = size
	}

	// Register the new node in the database
	if err := t.db.Set(t.nodeKey(key), node.bytes()); err != nil {
		return nil, err
	}

	// Hash the node.
	h, err := node.hash(t.hashFn)
	if err != nil {
		return nil, err
	}

	// Register the node's hash.
	if err := t.db.Set(t.hashKey(t.leafLevel, node.Index), h.Bytes()); err != nil {
		return nil, err
	}

	// Update the hashes up to the root.
	siblingHashes := make([]*big.Int, t.leafLevel)
	index := node.Index
	for level := t.leafLevel; level > 0; {
		siblingIndex := index + 1 - (index%2)*2

		// Fetch the sibling node hash from the database
		siblingHashBytes, err := t.db.Get(t.hashKey(level, siblingIndex))
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}

		siblingHash := new(big.Int).SetBytes(siblingHashBytes)

		// Save the sibling hash.
		siblingHashes[level-1] = siblingHash

		// Compute the parent hash.
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

		if level == 0 && index != 0 {
			return nil, errors.New("tree is over capacity")
		}

		// Register the parent hash
		if err := t.db.Set(t.hashKey(level, index), h.Bytes()); err != nil {
			return nil, err
		}
	}

	// Return the sibling hashes and the final root hash.
	p := &Proof{Root: h, Size: size, Node: node.deepCopy(), SiblingHashes: siblingHashes}
	return p, nil
}

// Store the tree size in the database.
func (t *Tree) setSize(s uint64) error {
	return t.db.Set(sizeKey, new(big.Int).SetUint64(s).Bytes())
}

// nodeProof generates an inclusion `Proof` for the given `node`.
func (t *Tree) nodeProof(node *Node) (*Proof, error) {
	// Fetch the treesize.
	size, err := t.size()
	if err != nil {
		return nil, err
	}

	// Fetch the tree root.
	root, err := t.root()
	if err != nil {
		return nil, err
	}

	// Climb up the tree and compute the parent hashes using the provided sibling hashes.
	siblingHashes := make([]*big.Int, t.leafLevel)
	index := node.Index
	for level := t.leafLevel; level > 0; level-- {
		siblingIndex := index + 1 - (index%2)*2

		// Fetch the sibling node hash from the database.
		siblingHashBytes, err := t.db.Get(t.hashKey(level, siblingIndex))
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
		siblingHash := new(big.Int).SetBytes(siblingHashBytes)

		// Save the sibling hash.
		siblingHashes[level-1] = siblingHash
	}

	// Return the node `Proof`.
	return &Proof{Root: root, Size: size, Node: node.deepCopy(), SiblingHashes: siblingHashes}, nil
}
