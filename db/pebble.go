package db

import (
	"errors"

	"github.com/cockroachdb/pebble"
)

type PebbleDb struct {
	batch *pebble.Batch
	wo    *pebble.WriteOptions
}

func NewPebbleDb(db *pebble.DB) *PebbleDb {
	return &PebbleDb{
		batch: db.NewIndexedBatch(),
		wo:    pebble.Sync,
	}
}

// Get implements Database.
func (p *PebbleDb) Get(key []byte) ([]byte, error) {
	b, closer, err := p.batch.Get(key)
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
	iter, err := p.batch.NewIter(&pebble.IterOptions{})
	if err != nil {
		return nil, nil, err
	}
	defer iter.Close()

	if !iter.SeekLT(ltKey) {
		return nil, nil, ErrNotFound
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
	return p.batch.Set(key, value, p.wo)
}

// Close implements Database.
func (p *PebbleDb) Close() error {
	return p.batch.Close()
}

// Commit implements Database.
func (p *PebbleDb) Commit() error {
	err := p.batch.Commit(p.wo)
	if err != nil {
		return err
	}

	p.batch.Reset()
	return nil
}

// Discard implements Database.
func (p *PebbleDb) Discard() {
	p.batch.Reset()
}

var _ Database = (*PebbleDb)(nil)
