package testutil_test

import (
	"encoding/json"
	"io"
	"os/exec"
	"strings"
	"testing"
)

// TestNoProductionImports verifies the API's compiled package graph never
// imports its test utility package. Asking Go for production imports avoids a
// filesystem walker that can drift from the compiler's own view.
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
		// flow-verifier emits replay helpers beside the generated formal
		// contract. They deliberately import modeltest and are invoked only by
		// the hand-authored *_test.go delegation; the generator's required
		// filename is replay.go rather than replay_test.go. Keep the boundary
		// strict for every ordinary production package while allowing that
		// generated test-only seam.
		if strings.Contains(pkg.ImportPath, "/flow/generated") {
			continue
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
