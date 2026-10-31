package rewrite_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestRewriteNoExternalCommand enforces PRD non-goal §12: the rewrite
// domain never invokes git, tsc, pnpm, or any other external command.
// File mutation is the sidecar's job; this package is pure
// orchestration.
//
// The gate checks two things:
//
//  1. No import of os/exec (the only stdlib way to spawn).
//  2. No source token matches the regex (?i)(\bgit\b|\btsc\b|\bpnpm\b)
//     outside of comments — so an accidentally introduced literal
//     command string fails the test even if exec is not imported.
func TestRewriteNoExternalCommand(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir .: %v", err)
	}

	fset := token.NewFileSet()
	disallowedToken := regexp.MustCompile(`(?i)\b(git|tsc|pnpm)\b`)

	var violations []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(".", name)

		// Import check.
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse imports %s: %v", name, err)
			continue
		}
		for _, imp := range file.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			if ip == "os/exec" {
				violations = append(violations, name+" imports os/exec")
			}
		}

		// Source token scan — strip line comments first so the
		// substrate-boundary comment in types.go does not trip us.
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		stripped := stripLineComments(string(raw))
		if disallowedToken.MatchString(stripped) {
			violations = append(violations, name+" mentions git/tsc/pnpm in source")
		}
	}

	if len(violations) > 0 {
		t.Errorf("internal/rewrite must not spawn external commands or name git/tsc/pnpm")
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}

func stripLineComments(src string) string {
	out := make([]byte, 0, len(src))
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		out = append(out, line...)
		out = append(out, '\n')
	}
	return string(out)
}
