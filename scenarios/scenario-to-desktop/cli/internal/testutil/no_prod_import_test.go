package testutil_test

import (
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"testing"
)

// TestNoProductionImports prevents test-only utilities from leaking into the
// CLI production package graph.
func TestNoProductionImports(t *testing.T) {
	module := goCommand(t, "list", "-m", "-f", "{{.Path}}")
	prefix := strings.TrimSpace(module) + "/internal/testutil"
	output := goCommand(t, "list", "-json", "./...")
	decoder := json.NewDecoder(strings.NewReader(output))
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
