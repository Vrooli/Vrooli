package coverage

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func infrastructureManagerRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate this test")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func TestCoverageHasNoPersistedDenominatorSchema(t *testing.T) {
	root := infrastructureManagerRoot(t)
	coverageDir := filepath.Join(root, "api", "internal", "coverage")
	err := filepath.WalkDir(coverageDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".sql") {
			t.Errorf("coverage owns a schema file: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if schema := Schema(); strings.TrimSpace(schema) != "" {
		t.Fatalf("coverage schema must stay empty, got %q", schema)
	}
}

func TestCoverageEndpointsRemainReadOnly(t *testing.T) {
	if len(Endpoints) == 0 {
		t.Fatal("coverage endpoints are unexpectedly empty")
	}
	for _, endpoint := range Endpoints {
		name := endpoint.Path[strings.LastIndex(endpoint.Path, "/")+1:]
		for _, verb := range []string{"Create", "Update", "Delete", "Set", "Write", "Actuate", "Mutate", "Apply"} {
			if strings.HasPrefix(name, verb) {
				t.Errorf("coverage endpoint %q exposes mutating verb %q", endpoint.Path, verb)
			}
		}
	}
}

func dependencyMethodIsReadOnly(name string) bool {
	for _, verb := range []string{"Create", "Update", "Delete", "Set", "Write", "Actuate", "Mutate", "Restart", "Stop", "Start", "Execute", "Trigger", "Send", "Reconcile", "Remove", "Patch", "Cancel", "Bind", "Shelve", "Resolve", "Install", "Uninstall", "Repair", "Run"} {
		if strings.HasPrefix(name, verb) {
			return false
		}
	}
	return true
}

func TestDependencyReadersExposeNoMutatingVerb(t *testing.T) {
	if dependencyMethodIsReadOnly("SetPolicy") {
		t.Fatal("negative control: mutating dependency method was allowed")
	}
	if !dependencyMethodIsReadOnly("ListChecks") {
		t.Fatal("read-only dependency method was rejected")
	}
	root := infrastructureManagerRoot(t)
	sourcesDir := filepath.Join(root, "api", "internal", "sources")
	err := filepath.WalkDir(sourcesDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			method, ok := node.(*ast.FuncDecl)
			if ok && method.Recv != nil && method.Name.IsExported() && !dependencyMethodIsReadOnly(method.Name.Name) {
				t.Errorf("dependency reader %s exposes mutating method %s", path, method.Name.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
