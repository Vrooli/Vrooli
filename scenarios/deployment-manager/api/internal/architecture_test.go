package internal_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDomainPackagesDoNotImportTransport keeps the domain seam enforceable as
// packages are moved under internal/<domain>. Transport belongs at the edge.
func TestDomainPackagesDoNotImportTransport(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve architecture test location")
	}
	root := filepath.Dir(file)
	entries, err := filepath.Glob(filepath.Join(root, "*", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, filename := range entries {
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}
		// The internal transport package is the explicit edge adapter; only
		// persisted domain packages are subject to this import ban.
		if filepath.Base(filepath.Dir(filename)) == "transport" {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "net/http" || strings.Contains(path, "connectrpc.com/connect") || strings.Contains(path, "gorilla/mux") {
				t.Errorf("domain package %s imports transport package %q", filename, path)
			}
		}
	}
}
