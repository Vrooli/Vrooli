package repository

import "errors"

// ErrNotFound is the sentinel error returned by repository implementations
// when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrNotSupported is returned by maintenance operations that a particular
// repository backend cannot perform (for example, compaction on an in-memory
// repository).
var ErrNotSupported = errors.New("operation not supported by this repository")
