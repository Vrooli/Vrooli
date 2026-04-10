package repository

import "errors"

// ErrNotFound is the sentinel error returned by repository implementations
// when a requested entity does not exist.
var ErrNotFound = errors.New("not found")
