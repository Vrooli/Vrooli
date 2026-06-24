package models

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// sidecarPYPath returns the absolute path to the embedded sidecar Python package
// root (internal/sidecar/py), relative to this test's package dir.
func sidecarPYPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd() // .../api/internal/models
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	p := filepath.Clean(filepath.Join(wd, "..", "sidecar", "py"))
	if _, err := os.Stat(filepath.Join(p, "image_tools_sidecar", "_diffusers.py")); err != nil {
		t.Skipf("sidecar python package not found at %s: %v", p, err)
	}
	return p
}

func python3(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping diffusers sidecar conformance")
	}
	return bin
}

// TestSeedRuntimeFamiliesAreRegistered asserts every diffusers model in the seed
// that declares a runtime family resolves to a registered Go adapter whose
// pipeline class matches — the seed↔families.go drift guard (no python needed).
func TestSeedRuntimeFamiliesAreRegistered(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	declared := 0
	for _, m := range r.Models() {
		fam := m.Runtime.Family
		if fam == "" {
			continue
		}
		declared++
		reg, ok := DiffusersFamilyByName(fam)
		if !ok {
			t.Errorf("model %s declares unregistered runtime.family %q", m.ID, fam)
			continue
		}
		if m.Runtime.PipelineClass != "" && m.Runtime.PipelineClass != reg.PipelineClass {
			t.Errorf("model %s pipeline_class %q != family %q (%s)", m.ID, m.Runtime.PipelineClass, fam, reg.PipelineClass)
		}
	}
	if declared == 0 {
		t.Fatal("expected at least one seed model to declare a runtime family")
	}
	// The sorted accessor returns every registered adapter.
	if got := DiffusersFamilies(); len(got) != len(diffusersFamilies) {
		t.Fatalf("DiffusersFamilies returned %d, want %d", len(got), len(diffusersFamilies))
	}
}

// TestDiffusersSidecarContract runs the pure (torch-free) Python adapter unit
// checks inside `go test` so the param→kwarg mapping is covered in CI without a
// separate pytest harness. Skips (not fails) where python3 is unavailable.
func TestDiffusersSidecarContract(t *testing.T) {
	py := python3(t)
	root := sidecarPYPath(t)
	cmd := exec.Command(py, "-m", "image_tools_sidecar.test_diffusers_adapters")
	cmd.Env = append(os.Environ(), "PYTHONPATH="+root)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python adapter checks failed: %v\n%s", err, out)
	}
}

// TestDiffusersFamilyAdaptersMirrorPython asserts the Go family registry
// (families.go) and the Python adapter table (_diffusers.FAMILIES) agree on the
// set of families, their pipeline classes, and ready state — so the registry,
// the Go catalog doctor, and the runner cannot drift apart.
func TestDiffusersFamilyAdaptersMirrorPython(t *testing.T) {
	py := python3(t)
	root := sidecarPYPath(t)
	cmd := exec.Command(py, "-c", "import json,image_tools_sidecar._diffusers as d; print(json.dumps(d.FAMILIES))")
	cmd.Env = append(os.Environ(), "PYTHONPATH="+root)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("dump python FAMILIES: %v", err)
	}
	var pyFams map[string]struct {
		PipelineClass string `json:"pipeline_class"`
		Ready         bool   `json:"ready"`
	}
	if err := json.Unmarshal(out, &pyFams); err != nil {
		t.Fatalf("decode python FAMILIES %q: %v", out, err)
	}

	goFams := diffusersFamilies
	if len(pyFams) != len(goFams) {
		t.Fatalf("family count mismatch: go=%d python=%d", len(goFams), len(pyFams))
	}
	for name, g := range goFams {
		p, ok := pyFams[name]
		if !ok {
			t.Errorf("family %q registered in Go but missing in Python", name)
			continue
		}
		if p.PipelineClass != g.PipelineClass {
			t.Errorf("family %q pipeline_class: go=%q python=%q", name, g.PipelineClass, p.PipelineClass)
		}
		if p.Ready != g.Ready {
			t.Errorf("family %q ready: go=%v python=%v", name, g.Ready, p.Ready)
		}
	}
	for name := range pyFams {
		if _, ok := goFams[name]; !ok {
			t.Errorf("family %q registered in Python but missing in Go", name)
		}
	}
}
