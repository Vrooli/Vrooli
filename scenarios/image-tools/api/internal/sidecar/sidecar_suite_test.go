package sidecar

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestPythonSidecarTestModules is the one gate for the sidecar's own Python tests.
// It discovers every test_*.py beside the sidecar sources and runs each under
// pytest (which also runs the unittest-style modules), so a new test module is
// gated the moment it exists: there is no per-module wrapper to forget. A module
// that collects no tests fails, because an empty module is a gate that gates
// nothing (pytest exits 5).
//
// The host python3 needs numpy, Pillow and pytest. Missing numpy or Pillow skips,
// like the golden-contract test above; a missing pytest fails with the fix, since
// skipping would let the plain-function modules (test_common, test_geometry,
// test_image_to_object, ...) fall out of the gate unnoticed.
func TestPythonSidecarTestModules(t *testing.T) {
	probe := exec.Command("python3", "-c", "import numpy, PIL")
	probe.Env = pythonSidecarEnv(t)
	if out, err := probe.CombinedOutput(); err != nil {
		t.Skipf("sidecar test modules need the host python3 with numpy and Pillow: %s", strings.TrimSpace(string(out)))
	}
	pytestProbe := exec.Command("python3", "-m", "pytest", "--version")
	pytestProbe.Env = pythonSidecarEnv(t)
	if out, err := pytestProbe.CombinedOutput(); err != nil {
		t.Fatalf("sidecar test modules need pytest on the host python3 (pip-installed outside the scenario venv): %v\n%s", err, out)
	}

	dir := filepath.Join("py", "image_tools_sidecar")
	modules, err := filepath.Glob(filepath.Join(dir, "test_*.py"))
	if err != nil {
		t.Fatalf("discover sidecar test modules: %v", err)
	}
	sort.Strings(modules)
	if len(modules) < 10 {
		t.Fatalf("discovered only %d sidecar test modules under %s; discovery is broken", len(modules), dir)
	}
	for _, module := range modules {
		module := module
		t.Run(strings.TrimSuffix(filepath.Base(module), ".py"), func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command("python3", "-m", "pytest", "-q", "-p", "no:cacheprovider", filepath.Base(module))
			cmd.Dir = dir
			cmd.Env = pythonSidecarEnv(t)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s failed: %v\n%s", filepath.Base(module), err, out)
			}
		})
	}
}
