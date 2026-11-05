package graph_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGraphPackageNoSubstrateImports enforces the substrate boundary:
// internal/graph/ owns pure data shaping. Time, ambient filesystem,
// and HTTP all belong to substrate packages — sidecar (which owns the
// Node child process), the handler (which owns transport-layer
// timing), or main (which owns wiring). If a domain file reaches for
// one of these, the layering has already broken.
//
// The sidecar package owns os/exec and time; the handler owns time
// (for ExtractionMs). The graph domain owns neither.
func TestGraphPackageNoSubstrateImports(t *testing.T) {
	forbidden := map[string]bool{
		"time":     true,
		"os":       true,
		"net/http": true,
		"os/exec":  true,
	}

	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir .: %v", err)
	}

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

		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", name, err)
			continue
		}
		for _, imp := range file.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			if forbidden[ip] {
				violations = append(violations, name+" imports "+ip)
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("internal/graph must not import substrate packages (time, os, net/http, os/exec)")
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}
