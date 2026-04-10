// Sentinel errors for the backlog package. These enable callers to use
// errors.Is for programmatic error handling instead of string matching.
package backlog

import "errors"

var (
	// ErrNotFound indicates the requested backlog item does not exist.
	ErrNotFound = errors.New("backlog item not found")

	// ErrAlreadyExists indicates a backlog item with the given kind/name
	// already exists on disk.
	ErrAlreadyExists = errors.New("backlog item already exists")

	// ErrInvalidKind indicates the provided kind string is not a recognized
	// backlog kind (idea, research, fix, execute, chore).
	ErrInvalidKind = errors.New("invalid backlog kind")
)
