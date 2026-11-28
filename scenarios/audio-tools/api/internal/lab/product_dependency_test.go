package lab_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestProductBuildOmitsLabExecutionPackages checks the compiler-selected
// product graph, not just the source import graph. Lab implementation files
// can exist in the repository without becoming part of the shipped product
// binary.
func TestProductBuildOmitsLabExecutionPackages(t *testing.T) {
	root := moduleRoot(t)
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", ".")
	cmd.Dir = root
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "GOFLAGS=") {
			cmd.Env = append(cmd.Env, item)
		}
	}
	cmd.Env = append(cmd.Env, "GOFLAGS=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list product dependency graph: %v\n%s", err, output)
	}
	packages := make(map[string]struct{})
	for _, line := range strings.Split(string(output), "\n") {
		if line != "" {
			packages[line] = struct{}{}
		}
	}
	for pkg := range packages {
		for _, forbidden := range labImportPrefixes {
			if pkg == forbidden || strings.HasPrefix(pkg, forbidden+"/") {
				t.Errorf("product build links lab package %s; lab packages must be behind the audio_lab build tag", pkg)
			}
		}
	}
}
