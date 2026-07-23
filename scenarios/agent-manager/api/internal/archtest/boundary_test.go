// Package archtest protects the import boundaries that keep agent-manager's
// runtime recoverable and its read-side exceptions intentional.
package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWorkflowRuntimeDoesNotImportOrchestration(t *testing.T) {
	violations := importsUnder(t, "api/internal/workflowruntime", "agent-manager/internal/orchestration")
	if len(violations) != 0 {
		t.Fatalf("workflowruntime must remain a leaf package: %v", violations)
	}
}

func TestProductionOrchestrationDoesNotImportDatabase(t *testing.T) {
	violations := importsUnder(t, "api/internal/orchestration", "agent-manager/internal/database")
	if len(violations) != 0 {
		t.Fatalf("orchestration must depend on repository interfaces, not database: %v", violations)
	}
}

func TestHandlersDirectPersistenceImportsAreReadSideExceptions(t *testing.T) {
	allowed := map[string]bool{
		"events.go":            true,
		"operational_stats.go": true,
		"pricing.go":           true,
	}
	for _, path := range goFiles(t, filepath.Join(scenarioRoot(t), "api/internal/handlers")) {
		imports, err := importsFromFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range imports {
			if imported != "agent-manager/internal/eventlog" && imported != "agent-manager/internal/stats" {
				continue
			}
			if !allowed[filepath.Base(path)] {
				t.Fatalf("%s imports read-side persistence %q without a documented exception", filepath.Base(path), imported)
			}
		}
	}
}

func TestHandlersUseCapabilityBoundaries(t *testing.T) {
	path := filepath.Join(scenarioRoot(t), "api/internal/handlers/handlers.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"orchestration.Service", "*orchestration.Orchestrator"} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("handlers must depend on narrow capability interfaces, found %q", forbidden)
		}
	}
	if !strings.Contains(string(source), "orchestration.HandlerServices") {
		t.Fatal("handlers must receive the composition-root capability bundle")
	}
}

func TestOnlyWiringConstructsProductionOrchestrator(t *testing.T) {
	apiRoot := filepath.Join(scenarioRoot(t), "api")
	for _, path := range goFiles(t, apiRoot) {
		rel, err := filepath.Rel(apiRoot, path)
		if err != nil {
			t.Fatal(err)
		}
		normalized := filepath.ToSlash(rel)
		if strings.Contains(normalized, "/testutil/") || strings.HasPrefix(normalized, "internal/wiring/") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(source), "orchestration.New(") {
			t.Fatalf("%s constructs the orchestrator outside internal/wiring", normalized)
		}
	}
}

func TestCapabilityBoundaryDetectorRejectsLegacyType(t *testing.T) {
	fixture := `package handlers; type Handler struct { svc orchestration.Service }`
	if !strings.Contains(fixture, "orchestration.Service") {
		t.Fatal("legacy handler fixture did not trigger the capability detector")
	}
}

func TestBoundaryDetectorRejectsViolatingFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source, forbidden string
	}{
		{"workflow runtime orchestration import", `package workflowruntime; import "agent-manager/internal/orchestration"`, "agent-manager/internal/orchestration"},
		{"orchestration database import", `package orchestration; import "agent-manager/internal/database"`, "agent-manager/internal/database"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			imports, err := importsFromSource(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			if !contains(imports, tc.forbidden) {
				t.Fatalf("fixture did not trigger %q: %v", tc.forbidden, imports)
			}
		})
	}
}

func importsUnder(t *testing.T, relativeDir, forbidden string) []string {
	t.Helper()
	var violations []string
	for _, path := range goFiles(t, filepath.Join(scenarioRoot(t), relativeDir)) {
		// Helpers under testutil are compiled so cross-package integration tests
		// can import them; they are test infrastructure, not runtime code.
		if strings.Contains(filepath.ToSlash(path), "/testutil/") {
			continue
		}
		imports, err := importsFromFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if contains(imports, forbidden) {
			violations = append(violations, filepath.ToSlash(path))
		}
	}
	return violations
}

func goFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func importsFromFile(path string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return importPaths(file), nil
}

func importsFromSource(source string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	return importPaths(file), nil
}

func importPaths(file *ast.File) []string {
	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		imports = append(imports, strings.Trim(spec.Path.Value, `"`))
	}
	return imports
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func scenarioRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve archtest source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
