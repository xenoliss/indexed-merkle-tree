package main

import (
	"crypto/sha256"
	"fmt"
	"imt/db"
	"imt/imt"
	"math/big"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	tmp, _ := os.MkdirTemp("", "*")
	defer os.RemoveAll(tmp)

	pebbleDb, err := pebble.Open(tmp, &pebble.Options{})
	if err != nil {
		panic(err)
	}
	defer pebbleDb.Close()

	db := db.NewPebbleDb(pebbleDb)
	defer db.Close()

	t := imt.NewTree(db, 32, 4, hash)

	mutateProof, err := t.Set(big.NewInt(1), big.NewInt(5))
	if err != nil {
		panic(err)
	}

	v, err := mutateProof.LnPreUpdateProof.IsValid(t)
	if err != nil {
		panic(err)
	}

	fmt.Printf("LnPreUpdateProof is valid: %v\n", v)

	v, err = mutateProof.NodeProof.IsValid(t)
	if err != nil {
		panic(err)
	}

	fmt.Printf("NodeProof is valid: %v\n", v)

	v, err = mutateProof.LnPostUpdateProof.IsValid(t)
	if err != nil {
		panic(err)
	}

	fmt.Printf("LnPostUpdateProof is valid: %v\n", v)

}

func hash(v []*big.Int) (*big.Int, error) {
	h := sha256.New()

	for i := 0; i < len(v); i++ {
		bn := v[i]
		h.Write(bn.Bytes())
	}

	s := h.Sum(nil)
	return new(big.Int).SetBytes(s), nil
}
