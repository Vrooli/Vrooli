// Tests that pin handler-file size limits.
//
// process.go has historically accreted unrelated logic — profile
// resolution, SSE wiring, git allowlist enforcement, resource limit
// defaulting — that belonged in domain packages. The Round-3
// refactor (2026-04-29) moved those out and split the remaining
// handlers across process_*.go files. process_logs.go added an SSE
// writer seam in Round-4 Phase 5 that pulled the wire-format details
// out into internal/sse, leaving only orchestration here.
//
// These tests fail loudly if a future change re-balloons either file.
// 600 / 500 were chosen so the files have room to grow naturally; if a
// test trips, the right move is to extract a helper into runtime/, or
// to push wire-format / domain logic into a dedicated package, NOT to
// bump the bound.

package handlers

import (
	"bytes"
	"os"
	"testing"
)

func TestProcessHandlerFileBound(t *testing.T) {
	cases := []struct {
		file string
		max  int
		hint string
	}{
		{
			file: "process.go",
			max:  600,
			hint: "Extract handlers into a sibling process_*.go file or move shared helpers into internal/runtime/",
		},
		{
			file: "process_logs.go",
			max:  500,
			hint: "Push wire-format / encoding logic into internal/sse and keep this file orchestration-only.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatalf("read %s: %v", tc.file, err)
			}
			lines := bytes.Count(data, []byte("\n"))
			if lines > tc.max {
				t.Fatalf("%s has %d lines; max is %d. %s", tc.file, lines, tc.max, tc.hint)
			}
		})
	}
}
