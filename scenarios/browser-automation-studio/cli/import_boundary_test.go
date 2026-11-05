package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The CLI is an API client. Keeping it out of API handler and test-helper
// packages prevents command additions from acquiring an in-process server
// dependency that cannot exist in a deployed CLI binary.
func TestProductionCLIImportsOnlyPublicAPIContracts(t *testing.T) {
	forbidden := []string{
		"github.com/vrooli/browser-automation-studio/handlers",
		"github.com/vrooli/browser-automation-studio/internal/testutil",
		"github.com/vrooli/browser-automation-studio/testutil",
	}
	var violations []string
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, blocked := range forbidden {
				if path == blocked || strings.HasPrefix(path, blocked+"/") {
					violations = append(violations, path+" imports "+blocked)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan CLI imports: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("CLI must depend on public API contracts only:\n%s", strings.Join(violations, "\n"))
	}
}
