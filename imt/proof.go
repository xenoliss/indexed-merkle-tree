package imt

import (
	"math/big"
)

type Proof struct {
	Root          *big.Int
	Size          uint64
	Node          *Node
	SiblingHashes []*big.Int
}

var zeroBn = new(big.Int)

// isValid returns `true` if the `Proof` is a valid proof for the given tree.
func (p *Proof) IsValid(t *Tree) (bool, error) {
	// Start with the node hash.
	h, err := p.Node.hash(t.hashFn)
	if err != nil {
		return false, err
	}

	// If the tree is empty, the root must be the hash of the initial state node.
	if p.Size == 0 {
		return p.Node.Index == 0 &&
			p.Node.Value.Cmp(zeroBn) == 0 &&
			p.Node.NextKey.Cmp(zeroBn) == 0 &&
			h.Cmp(p.Root) == 0, nil
	}

	// Climb up the tree and compute the parent hashes using the provided sibling hashes.
	index := p.Node.Index
	for level := t.leafLevel; level > 0; level-- {
		siblingHash := p.SiblingHashes[level-1]

		if index%2 == 0 {
			h, err = t.hashFn([]*big.Int{h, siblingHash})
		} else {
			h, err = t.hashFn([]*big.Int{siblingHash, h})
		}

		if err != nil {
			return false, err
		}
	}

	// Compare it with the root hash.
	return h.Cmp(p.Root) == 0, nil
}

type MutateProof struct {
	LnPreUpdateProof  *Proof
	NodeProof         *Proof
	LnPostUpdateProof *Proof
}
