package db

import (
	"errors"

	"github.com/cockroachdb/pebble"
)

type PebbleDb struct {
	db *pebble.DB
	wo *pebble.WriteOptions
}

func NewPebbleDb(db *pebble.DB) *PebbleDb {
	return &PebbleDb{
		db: db,
		wo: pebble.Sync,
	}
}

// Get implements Database.
func (p *PebbleDb) Get(key []byte) ([]byte, error) {
	b, closer, err := p.db.Get(key)
	if errors.Is(err, pebble.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	ret := make([]byte, len(b))
	copy(ret, b)
	closer.Close()

	return ret, nil
}

// GetLT implements Database.
func (p *PebbleDb) GetLT(ltKey []byte) ([]byte, []byte, error) {
	iter, err := p.db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return nil, nil, err
	}
	defer iter.Close()

	if !iter.SeekLT(ltKey) {
		return nil, nil, nil
	}

	k := iter.Key()
	v, err := iter.ValueAndErr()
	if err != nil {
		return nil, nil, err
	}

	return k, v, nil
}

// Set implements Database.
func (p *PebbleDb) Set(key []byte, value []byte) error {
	return p.db.Set(key, value, p.wo)
}

var _ Database = (*PebbleDb)(nil)
