//go:build e2e_lang_graph

// Package conflicts_test contains the live end-to-end check that
// runs cartographer against real `go-code-graph` / `typescript-code-graph`
// instances. Tagged `e2e_lang_graph` so it stays opt-in until those
// dependency scenarios ship (tracked in docs/internal/PROBLEMS.md).
//
// Run via: go test -tags e2e_lang_graph ./internal/conflicts/...
package conflicts_test

import (
	"testing"
)

// TestE2E_LiveGoCodeGraph is the unblock-marker: when go-code-graph
// reaches a usable state, fill this test in to drive an Extract call
// through the live adapter, then through the conflicts detector chain.
// Until then, the build tag keeps the test out of `go test ./...`.
func TestE2E_LiveGoCodeGraph(t *testing.T) {
	t.Skip("e2e_lang_graph: go-code-graph scenario not implemented yet; see docs/internal/PROBLEMS.md")
}
