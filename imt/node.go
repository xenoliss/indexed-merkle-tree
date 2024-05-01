package imt

import (
	"encoding/binary"
	"errors"
	"math/big"
)

type Node struct {
	Index   uint64
	Value   *big.Int
	NextKey *big.Int
}

// emptyNode returns an empty node.
func emptyNode() *Node {
	return &Node{Index: 0, Value: new(big.Int), NextKey: new(big.Int)}
}

// nodeFromBytes deserializes a node from bytes.
func nodeFromBytes(b []byte) (*Node, error) {
	i := binary.BigEndian.Uint64(b[0:8])

	b = b[8:]
	if len(b) < 1 {
		return nil, errors.New("invalid bytes")
	}
	v := new(big.Int).SetBytes(b[1 : 1+b[0]])

	b = b[1+b[0]:]
	if len(b) < 1 {
		return nil, errors.New("invalid bytes")
	}
	nK := new(big.Int).SetBytes(b[1 : 1+b[0]])

	return &Node{Index: i, Value: v, NextKey: nK}, nil
}

// clone returns a deep copy of the node.
func (n *Node) clone() *Node {
	cp := &Node{}
	cp.Index = n.Index
	cp.Value = new(big.Int).Set(n.Value)
	cp.NextKey = new(big.Int).Set(n.NextKey)

	return cp
}

// bytes serializes the node into bytes.
func (n *Node) bytes() []byte {
	b := binary.BigEndian.AppendUint64([]byte{}, n.Index)

	valueBytes := n.Value.Bytes()
	b = append(b, byte(len(valueBytes)))
	b = append(b, valueBytes...)

	nextKeyBytes := n.NextKey.Bytes()
	b = append(b, byte(len(nextKeyBytes)))
	b = append(b, nextKeyBytes...)

	return b
}

// hash returns the node's hash.
func (n *Node) hash(h HashFn) (*big.Int, error) {
	return h([]*big.Int{new(big.Int).SetUint64(n.Index), n.Value, n.NextKey})
}
