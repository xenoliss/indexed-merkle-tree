package db

import (
	"errors"
)

var ErrNotFound = errors.New("not found")

type ExecFn func() error

type Database interface {
	// Get retrieves the value for the given `key`. If the `key` does not
	// exist, returns the error `ErrNotFound`.
	Get(key []byte) ([]byte, error)

	// GetLT retrieves the key and value less than the given key.
	GetLT(ltKey []byte) ([]byte, []byte, error)

	// Set sets the `value` for the given `key`.
	Set(key []byte, value []byte) error

	// Close the database and release its resources.
	Close() error

	// Execut the given `fn` and atomically commit or discard all database changes.
	ExecAtomicCommitOrDiscard(fn ExecFn) error
}
