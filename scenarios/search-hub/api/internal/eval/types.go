// Package eval is the domain-scoped home for the search-hub baseline eval
// harness — provider-owned golden suites and the immutable, tagged runs taken
// against them.
//
// Layering mirrors the registry domain:
//
//	Connect handler → Store (validates via Validate, persists)
//	                → Runner (calls a provider, labels cases) → Store.AppendRun
//	                     ↑
//	                     fakes (handler/runner tests) / real SQLite (store tests)
//
// Like the registry, eval does not re-model its proto contract as a separate
// domain struct: an EvalSuite/EvalRun is pure wire data, so the store persists
// the generated proto message directly (a protojson blob plus projected filter
// columns). Validation/normalization (validate.go) and the soft outcome-labeling
// math (runner.go) are the only domain logic.
//
// This domain is a SIBLING of the registry, not a fork of it: a suite references
// a registry provider_id and the runner reuses that provider's Endpoint +
// ResultMapping, so there is zero provider-specific code here either.
package eval

import "fmt"

// ListSuitesFilter narrows ListSuites. A zero value means "no filter".
type ListSuitesFilter struct {
	ProviderID string // empty = no filter
}

// ListRunsFilter narrows ListRuns for the history view. SuiteID is required by
// the handler; Tag and Limit are optional. Limit <= 0 means "all".
type ListRunsFilter struct {
	SuiteID string
	Tag     string // empty = all tags
	Tier    string // empty = all tiers
	Limit   int    // <= 0 = all
}

// ErrInvalidSuite is the typed sentinel returned by Validate (and propagated by
// Store.UpsertSuite) when a suite fails validation. Handlers translate via
// errors.As into a 400 carrying connect.CodeInvalidArgument with message
// "<field>: <reason>".
type ErrInvalidSuite struct {
	Field  string
	Reason string
}

func (e ErrInvalidSuite) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ErrSuiteNotFound is the typed sentinel returned by Store.GetSuite and the
// runner when no suite matches the given suite_id. Handlers translate via
// errors.As into a 404 carrying connect.CodeNotFound.
type ErrSuiteNotFound struct {
	SuiteID string
}

func (e ErrSuiteNotFound) Error() string {
	return fmt.Sprintf("eval suite %q not found", e.SuiteID)
}

// ErrRunNotFound is the typed sentinel returned by Store.GetRun when no row
// matches the given run_id.
type ErrRunNotFound struct {
	RunID string
}

func (e ErrRunNotFound) Error() string {
	return fmt.Sprintf("eval run %q not found", e.RunID)
}
