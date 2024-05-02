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

	t := imt.NewTree(db, 32, 4, hash)

	setAndValidate(t, big.NewInt(1), big.NewInt(5))
	setAndValidate(t, big.NewInt(5), big.NewInt(5))
	setAndValidate(t, big.NewInt(3), big.NewInt(5))
	setAndValidate(t, big.NewInt(4), big.NewInt(5))
	setAndValidate(t, big.NewInt(10), big.NewInt(5))
}

func setAndValidate(t *imt.Tree, key, value *big.Int) {
	proof, err := t.Set(key, value)
	if err != nil {
		panic(err)
	}

	valid, err := proof.IsValid(t)
	if err != nil {
		panic(err)
	}

	if !valid {
		panic("Validation failed")
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
