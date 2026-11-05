package testutil_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoAmbientInRewriteDomain mirrors no_ambient_in_graph_test.go for
// the rewrite domain. Production code under internal/rewrite/ must
// inject ambient dependencies via constructors, not reach for the
// well-known package-level globals.
//
// Forbidden references inside non-test .go files under
// internal/rewrite/ (and its sub-packages):
//
//   - time.Now (use a clock.Clock seam)
//   - os.Getenv (configuration arrives via constructors)
//   - http.DefaultClient (use httpc.Doer)
//   - log.Default (loggers thread in explicitly)
//
// Also forbids any re-introduction of the deleted notes domain.
func TestNoAmbientInRewriteDomain(t *testing.T) {
	root := filepath.FromSlash("../rewrite")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("internal/rewrite not present: %v", err)
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
		t.Errorf("internal/rewrite drift gate violations:")
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}
