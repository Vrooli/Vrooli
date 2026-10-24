package planlog

import "fmt"

// ErrInvalidEntry is a client-input validation failure (missing title, unknown
// type, etc.). The handler maps it to an InvalidArgument Connect code.
type ErrInvalidEntry struct{ Reason string }

func (e ErrInvalidEntry) Error() string { return "invalid log entry: " + e.Reason }

// ErrEntryNotFound is returned when an entry id does not resolve.
type ErrEntryNotFound struct{ ID string }

func (e ErrEntryNotFound) Error() string { return fmt.Sprintf("log entry %q not found", e.ID) }

// ErrNotPromotable is returned when PromoteEntry targets a non-finding entry or
// an invalid target type.
type ErrNotPromotable struct{ Reason string }

func (e ErrNotPromotable) Error() string { return "entry not promotable: " + e.Reason }

// ErrDownstreamUnavailable is the typed, non-fatal signal that a downstream sink
// could not be reached. The service records it as a degraded sync state on the
// durable entry rather than failing the operation.
type ErrDownstreamUnavailable struct {
	System string
	Reason string
}

func (e ErrDownstreamUnavailable) Error() string {
	return fmt.Sprintf("downstream %s unavailable: %s", e.System, e.Reason)
}
