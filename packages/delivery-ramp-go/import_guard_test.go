package deliveryramp

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSpineHasNoScenarioImports(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filename)
	entries, err := filepath.Glob(filepath.Join(root, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range entries {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, spec := range file.Imports {
			value := strings.Trim(spec.Path.Value, `"`)
			if strings.Contains(value, "/scenarios/") || strings.Contains(value, "scenario-to-desktop-api") {
				t.Fatalf("spine file %s imports scenario module %q", path, value)
			}
		}
	}
}
