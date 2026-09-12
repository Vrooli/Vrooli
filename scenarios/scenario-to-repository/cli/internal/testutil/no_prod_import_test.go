package testutil_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNoProductionImports verifies the CLI's compiled package graph never
// imports its test utility package. Asking Go for production imports avoids a
// second filesystem walker that can drift from the compiler's own view.
func TestNoProductionImports(t *testing.T) {
	module := goCommand(t, "list", "-m", "-f", "{{.Path}}")
	prefix := strings.TrimSpace(module) + "/internal/testutil"

	output := goCommand(t, "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./...")
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		for _, imported := range fields[1:] {
			if strings.HasPrefix(imported, prefix) {
				t.Errorf("production package %s imports %s", fields[0], imported)
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
