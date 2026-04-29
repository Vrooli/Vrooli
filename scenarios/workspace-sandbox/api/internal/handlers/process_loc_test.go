// Tests that pin handler-file size limits.
//
// process.go has historically accreted unrelated logic — profile
// resolution, SSE wiring, git allowlist enforcement, resource limit
// defaulting — that belonged in domain packages. The Round-3
// refactor (2026-04-29) moved those out and split the remaining
// handlers across process_*.go files. This test fails loudly if a
// future change re-balloons process.go past 600 LOC.
//
// 600 was chosen so the file has room to grow naturally; if the test
// trips, the right move is to extract a helper into runtime/ or to
// split the offending handler into its own process_*.go file, NOT
// to bump the bound.

package handlers

import (
	"bytes"
	"os"
	"testing"
)

func TestProcessHandlerFileBound(t *testing.T) {
	const max = 600
	data, err := os.ReadFile("process.go")
	if err != nil {
		t.Fatalf("read process.go: %v", err)
	}
	lines := bytes.Count(data, []byte("\n"))
	if lines > max {
		t.Fatalf("process.go has %d lines; max is %d. Extract handlers into a sibling process_*.go file or move shared helpers into internal/runtime/", lines, max)
	}
}
