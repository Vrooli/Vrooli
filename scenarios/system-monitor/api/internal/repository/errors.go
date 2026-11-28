package repository

import "errors"

// ErrNotFound is the sentinel error returned by repository implementations
// when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// ErrNotSupported is returned by maintenance operations that a particular
// repository backend cannot perform (for example, compaction on an in-memory
// repository).
var ErrNotSupported = errors.New("operation not supported by this repository")

// ErrMaintenance is returned by reads that would otherwise wait behind an
// in-progress owner compaction. Callers fall back to in-memory state.
var ErrMaintenance = errors.New("database compaction in progress")
