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
	temp, _ := os.MkdirTemp("", "*")
	defer os.RemoveAll(temp)

	pebbleDb, err := pebble.Open(temp, &pebble.Options{})
	if err != nil {
		panic(err)
	}

	db := db.NewPebbleDb(pebbleDb)

	tree := imt.NewTree(db, 32, 32, hash)
	err = tree.Insert(big.NewInt(5), big.NewInt(5))
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
