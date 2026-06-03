// Package registry is the domain-scoped home for the search-hub provider
// registry — the persisted set of ProviderDescriptors the router routes on.
//
// Layering mirrors the canonical Vrooli pattern:
//
//	Connect handler → Store (validates via Validate, persists)
//	                     ↑
//	                     FakeStore (handler tests) / real SQLite (store tests)
//
// Unlike the notes reference domain, the registry does not re-model its proto
// contract as a separate domain struct: a ProviderDescriptor is pure wire data
// (no business behavior), so the store persists the generated proto message
// directly (as a protojson blob plus projected filter columns). Validation and
// default-normalization are the only domain logic and live in validate.go.
package registry

import "fmt"

// ListFilter narrows ListProviders. A zero value (UNSPECIFIED enums, empty
// type) means "no filter" — every field is optional and ANDed together.
type ListFilter struct {
	Bucket int32  // registryv1.Bucket; 0 = unspecified = no filter
	Type   string // empty = no filter
	State  int32  // registryv1.ProviderState; 0 = unspecified = no filter
}

// ErrInvalidDescriptor is the typed sentinel returned by Validate (and
// propagated by Store.Upsert) when a descriptor fails validation. Field names
// the offending field; Reason is a human-safe explanation. Handlers translate
// via errors.As into a 400 carrying connect.CodeInvalidArgument with message
// "<field>: <reason>".
type ErrInvalidDescriptor struct {
	Field  string
	Reason string
}

func (e ErrInvalidDescriptor) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ErrProviderNotFound is the typed sentinel returned by Store.Get when no row
// matches the given provider_id. Handlers translate via errors.As into a 404
// carrying connect.CodeNotFound. (Deregister reports absence via its boolean
// result rather than this error, mirroring an idempotent delete.)
type ErrProviderNotFound struct {
	ProviderID string
}

func (e ErrProviderNotFound) Error() string {
	return fmt.Sprintf("provider %q not found", e.ProviderID)
}
