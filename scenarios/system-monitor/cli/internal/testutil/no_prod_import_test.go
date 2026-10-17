package testutil_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNoProductionImports keeps test-only helpers out of the production CLI
// dependency graph. The compiler's package graph is the source of truth here,
// so this guard remains stable as the CLI gains subcommands.
func TestNoProductionImports(t *testing.T) {
	t.Helper()
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
	command.Dir = "../.."
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
