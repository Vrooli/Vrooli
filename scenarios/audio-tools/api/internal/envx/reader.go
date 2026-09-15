// Package envx is the canonical seam for process-environment reads.
//
// Production wires envx.OS{} once in main.go (composition root) and threads
// it through every constructor that needs to read an environment variable.
// Tests substitute mocks.FakeEnv to control values deterministically without
// mutating process env via t.Setenv from domain tests.
//
// # Why a separate seam from clock / httpc
//
// Domain code that reads os.Getenv at request time forces tests to either
// (a) call t.Setenv (mutates process state — races under t.Parallel) or
// (b) skip the branch. Both are L3 violations. envx.Reader.Get gives a
// fake-substitutable one-method surface that costs nothing in production
// (OS.Get is a one-line shim) but enables deterministic tests.
//
// The seam exposes Get(string) string deliberately. The two-value form
// (string, bool) is not used because consumers default on empty-string;
// adding a bool would just propagate a second branch everywhere.
package envx

import "os"

// seam: Reader is the process-env seam (SEAMS.md row "envx.Reader").
// Production wires envx.OS{}; tests wire mocks.FakeEnv.
//
// Reader returns process environment values by key. Empty string indicates
// "unset" — matching os.Getenv semantics — so callers default on empty.
type Reader interface {
	Get(key string) string
}

// OS is the production Reader; delegates to os.Getenv. Constructed once
// in main.go / bootstrap and passed to every consumer.
type OS struct{}

// Get returns the value of the named environment variable, or empty
// string if unset.
func (OS) Get(key string) string { return os.Getenv(key) }

// Compile-time guarantee that OS satisfies Reader.
var _ Reader = OS{}
