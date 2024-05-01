package db

import (
	"errors"
)

var ErrNotFound = errors.New("not found")

type Database interface {
	// Get retrieves the value for the given `key`. If the `key` does not
	// exist, returns the error `ErrNotFound`.
	Get(key []byte) ([]byte, error)

	// GetLT retrieves the key and value less than the given key.
	GetLT(ltKey []byte) ([]byte, []byte, error)

	// Set sets the `value` for the given `key`.
	Set(key []byte, value []byte) error

	// Commit commits the pending batch changes.
	Commit() error

	// Discard discards the pending batch changes.
	Discard()

	// Close the database and release its resources.
	Close() error
}
