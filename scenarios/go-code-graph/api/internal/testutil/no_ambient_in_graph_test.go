package testutil_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoAmbientInGraphDomain is the Phase-2 drift gate: production
// code inside internal/graph/ must inject ambient dependencies
// (wall-clock, env, default loggers, default HTTP clients) via
// constructors, not reach for the well-known package-level globals.
//
// Equivalent of feedback_design_to_ideal_fix_substrate at the test
// level — substrate is "seams everywhere", and this test catches
// drift the first time someone reflex-reaches for time.Now() inside
// the graph domain.
//
// Forbidden references inside non-test .go files under
// internal/graph/ (and its sub-packages):
//
//   - time.Now (use a clock.Clock seam)
//   - os.Getenv (configuration arrives via constructors)
//   - http.DefaultClient (use httpc.Doer)
//   - log.Default (loggers thread in explicitly)
//
// Also forbids any re-introduction of the deleted notes domain — the
// template-residue cleanup is permanent.
func TestNoAmbientInGraphDomain(t *testing.T) {
	root := filepath.FromSlash("../graph")
	if _, err := os.Stat(root); err != nil {
		// internal/graph/ doesn't exist yet (this test predates Phase
		// 2) — nothing to enforce.
		t.Skipf("internal/graph not present: %v", err)
		return
	}

	forbidden := map[string]string{
		"time.Now":           "use clock.Clock seam, not time.Now (drift gate, see plan §12)",
		"os.Getenv":          "thread config in via constructor, not os.Getenv",
		"http.DefaultClient": "use httpc.Doer seam, not http.DefaultClient",
		"log.Default":        "thread *log.Logger in via constructor, not log.Default()",
	}

	violations := []string{}
	fset := token.NewFileSet()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if filepath.Base(path) == "mocks" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Errorf("parse %s: %v", path, err)
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			return nil
		}
		text := string(src)
		for needle, reason := range forbidden {
			if strings.Contains(text, needle) {
				violations = append(violations, path+": forbidden token "+needle+" ("+reason+")")
			}
		}
		// Disallow re-import of the deleted notes domain.
		for _, imp := range file.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(ip, "/v1/notes") || strings.HasSuffix(ip, "/internal/notes") {
				violations = append(violations, path+" imports deleted notes package: "+ip)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("internal/graph drift gate violations:")
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}
