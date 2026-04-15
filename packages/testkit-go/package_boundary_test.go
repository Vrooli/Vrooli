package testkitgo

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootPackageProductionFilesDoNotImportInternalPackages(t *testing.T) {
	root := filepath.Join(ProjectRoot(t), "packages", "testkit-go")
	matches, err := filepath.Glob(filepath.Join(root, "*.go"))
	if err != nil {
		t.Fatalf("glob root package files: %v", err)
	}

	fset := token.NewFileSet()
	for _, path := range matches {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if strings.HasPrefix(importPath, "github.com/vrooli/vrooli/internal/") {
				t.Fatalf("%s imports forbidden internal package %q", filepath.Base(path), importPath)
			}
		}
	}
}
