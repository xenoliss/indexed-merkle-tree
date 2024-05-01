package imt

import (
	"math/big"
)

type Proof struct {
	// The root of the tree.
	Root *big.Int

	// The size of the tree.
	Size uint64

	// The node being proved.
	Node *Node

	// The merkle path of sibling hashes.
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

	// If the tree is empty, the root must be the hash of the 0 index node.
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
	// The low nullifier inclusion proof before inserting the new key in the tree.
	// Nil if the mutation is not an instertion.
	LnPreInsertProof *Proof

	// The low nullifier inclusion proof before inserting the new key in the tree.
	// Nil if the mutation is not an instertion.
	LnPostInsertProof *Proof

	// The inserted/updated node proof.
	NodeProof *Proof
}

func (p *MutateProof) IsValid(t *Tree) (bool, error) {
	if p.LnPreInsertProof != nil {
		valid, err := p.LnPreInsertProof.IsValid(t)
		if !valid || err != nil {
			return false, err
		}
	}

	if p.LnPostInsertProof != nil {
		valid, err := p.LnPostInsertProof.IsValid(t)
		if !valid || err != nil {
			return false, err
		}
	}

	return p.NodeProof.IsValid(t)
}
