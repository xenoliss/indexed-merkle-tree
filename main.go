package main

import (
	"crypto/sha256"
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

	tree := imt.NewTree(db, 32, 4, hash)

	err = tree.Insert(big.NewInt(1), big.NewInt(5))
	if err != nil {
		panic(err)
	}

	err = tree.Insert(big.NewInt(4), big.NewInt(5))
	if err != nil {
		panic(err)
	}
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
