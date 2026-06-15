// Package repokit provides generic in-memory CRUD repository fakes.
//
// Per-domain Repository fakes follow a structurally identical shape: an
// in-memory slice, atomic call counters, per-method error injection, a
// typed-sentinel constructor for Get-misses, and ID assignment on Create
// when the input ID is empty. repokit hoists that shape into a generic
// substrate so each new domain stops re-implementing ~100 lines of mock
// + ~100 lines of mock-self-tests.
//
// Per-domain wiring is a 5-15 line type alias + extractor plumbing:
//
//	type FakeRepository = repokit.SliceRepo[audit.Run]
//
//	func NewFakeRepository() *FakeRepository {
//	    return &FakeRepository{
//	        GetID:    func(r audit.Run) string { return r.ID },
//	        SetID:    func(r *audit.Run, id string) { r.ID = id },
//	        NotFound: func(id string) error { return audit.ErrRunNotFound{ID: id} },
//	    }
//	}
//
// Test-only — production code must not import this package. The contract
// is enforced by api/internal/testutil/no_prod_import_test.go (the AST
// guardrail walks every non-test file under api/ and fails if any import
// references a path containing /testutil/).
//
// # Scope
//
// SliceRepo covers the standard "Create(T) → T / Get(id) → T / List(limit)
// → []T" shape. Other domain-specific shapes stay hand-written
// for now; lift to repokit only when a second consumer surfaces.
//
// A RecordingService generic for the FakeService shape (records inputs,
// returns canned outputs) is a natural follow-up; deferred until a second
// service-fake consumer exists.
package repokit
