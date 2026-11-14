// Package testutil_test guards the test-helper root required by the unit
// policy profile (.vrooli/testing.json): no production package may import
// internal/testutil.
package testutil_test

import (
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"testing"
)

// TestNoProductionImports asks Go for the compiled package graph and fails if
// any production package imports internal/testutil.
func TestNoProductionImports(t *testing.T) {
	module := goCommand(t, "list", "-m", "-f", "{{.Path}}")
	prefix := strings.TrimSpace(module) + "/internal/testutil"

	decoder := json.NewDecoder(strings.NewReader(goCommand(t, "list", "-json", "./...")))
	for {
		var pkg struct {
			ImportPath string
			Imports    []string
		}
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("decode go list package: %v", err)
		}
		for _, imported := range pkg.Imports {
			if strings.HasPrefix(imported, prefix) {
				t.Errorf("production package %s imports %s", pkg.ImportPath, imported)
			}
		}
	}
}

func goCommand(t *testing.T, args ...string) string {
	t.Helper()
	command := exec.Command("go", args...)
	command.Dir = "../../"
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
