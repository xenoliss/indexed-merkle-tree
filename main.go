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

	p1, err := t.Set(big.NewInt(1), big.NewInt(5))
	if err != nil {
		panic(err)
	}

	valid, err := p1.IsValid(t)
	if err != nil {
		panic(err)
	}

	fmt.Printf("p1 valid: %v\n", valid)
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
