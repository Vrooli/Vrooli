package testutil_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNoProductionImports keeps the shared fixture package test-only. The
// compiled production import graph is the source of truth, so this guard does
// not depend on a second filesystem scanner that could drift from Go's build.
func TestNoProductionImports(t *testing.T) {
	module := goCommand(t, "list", "-m", "-f", "{{.Path}}")
	prefix := strings.TrimSpace(module) + "/internal/testutil"

	packages := goCommand(t, "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./...")
	for _, line := range strings.Split(strings.TrimSpace(packages), "\n") {
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
