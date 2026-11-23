package models

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// TestLoRASpecParityPython asserts the Go LoRA arg emitter (technique.LoRAArgs
// format `<path>:<scale>`) and the Python parser (_adapters.parse_lora_spec) agree
// on the wire contract: a spec the Go side emits parses back to the same path +
// scale in Python. Skips where python3 is unavailable.
func TestLoRASpecParityPython(t *testing.T) {
	py := python3(t)
	root := sidecarPYPath(t)
	cases := []struct {
		path  string
		scale string // as Go's strconv.FormatFloat(scale, 'g', -1, 64) would emit
	}{
		{"/models/adapters/lcm/lcm.safetensors", "0.8"},
		{"/a/b/weights.safetensors", "1"},
		{"/p/q.safetensors", "1.25"},
	}
	for _, tc := range cases {
		spec := tc.path + ":" + tc.scale
		cmd := exec.Command(py, "-c",
			"import json,sys,image_tools_sidecar._adapters as a; s=a.parse_lora_spec(sys.argv[1]); print(json.dumps([s.path, s.scale]))",
			spec)
		cmd.Env = append(os.Environ(), "PYTHONPATH="+root)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("python parse_lora_spec(%q): %v", spec, err)
		}
		var got []any
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("decode parse output %q: %v", out, err)
		}
		if len(got) != 2 || got[0] != tc.path {
			t.Fatalf("spec %q: python path = %v, want %q", spec, got[0], tc.path)
		}
		wantScale, _ := strconv.ParseFloat(tc.scale, 64)
		gotScale, ok := got[1].(float64)
		if !ok || gotScale != wantScale {
			t.Fatalf("spec %q: python scale = %v, want %v", spec, got[1], wantScale)
		}
	}
}

// TestConditioningSpecParityPython asserts the Go ControlNet / IP-Adapter arg
// emitters (technique.ControlNetArgs / IPAdapterArgs, both `<path>:<scale>:<image>`)
// and the Python parsers (_adapters.parse_controlnet_spec / parse_ip_adapter_spec)
// agree on the wire contract. Skips where python3 is unavailable.
func TestConditioningSpecParityPython(t *testing.T) {
	py := python3(t)
	root := sidecarPYPath(t)
	cases := []struct {
		fn    string // python parser
		path  string
		scale string
		image string
	}{
		{"parse_controlnet_spec", "/models/adapters/cn-canny-sd15", "0.9", "/tmp/imgtools-ai-1/cond-0.png"},
		{"parse_controlnet_spec", "/a/cn", "1", "/b/map.png"},
		{"parse_ip_adapter_spec", "/models/adapters/ip/ip.safetensors", "0.6", "/tmp/ref.png"},
		{"parse_ip_adapter_spec", "/x/y.safetensors", "1.25", "/z/r.png"},
	}
	for _, tc := range cases {
		spec := tc.path + ":" + tc.scale + ":" + tc.image
		cmd := exec.Command(py, "-c",
			"import json,sys,image_tools_sidecar._adapters as a; s=getattr(a,sys.argv[1])(sys.argv[2]); print(json.dumps([s.path, s.scale, s.image]))",
			tc.fn, spec)
		cmd.Env = append(os.Environ(), "PYTHONPATH="+root)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("python %s(%q): %v", tc.fn, spec, err)
		}
		var got []any
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("decode parse output %q: %v", out, err)
		}
		if len(got) != 3 || got[0] != tc.path || got[2] != tc.image {
			t.Fatalf("spec %q via %s: python = %v, want path=%q image=%q", spec, tc.fn, got, tc.path, tc.image)
		}
		wantScale, _ := strconv.ParseFloat(tc.scale, 64)
		if gotScale, ok := got[1].(float64); !ok || gotScale != wantScale {
			t.Fatalf("spec %q: python scale = %v, want %v", spec, got[1], wantScale)
		}
	}
}

// TestDiffusersFamilyAdaptersMirrorPython asserts the Go family registry
// (families.go) and the Python adapter table (_diffusers.FAMILIES) agree on the
// set of families, their pipeline classes, ready state and whether the family
// generates from text. Each failure says which file is missing which half.
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
		Generates     bool   `json:"generates"`
	}
	if err := json.Unmarshal(out, &pyFams); err != nil {
		t.Fatalf("decode python FAMILIES %q: %v", out, err)
	}

	for name, g := range diffusersFamilies {
		p, ok := pyFams[name]
		if !ok {
			t.Errorf("family %q is registered in Go (internal/models/families.go) but has no Python adapter: add an Adapter(family=%q, ...) to _ADAPTERS in internal/sidecar/py/image_tools_sidecar/_diffusers.py", name, name)
			continue
		}
		if p.PipelineClass != g.PipelineClass {
			t.Errorf("family %q loads %q in Python but %q in Go: make families.go and _diffusers.py name the same pipeline class", name, p.PipelineClass, g.PipelineClass)
		}
		if p.Ready != g.Ready {
			t.Errorf("family %q is ready=%v in Go but ready=%v in Python: flip both together once an attended run proves it", name, g.Ready, p.Ready)
		}
		servesGeneration := false
		for _, op := range g.Operations {
			if op == "text_to_image" {
				servesGeneration = true
			}
		}
		if servesGeneration != p.Generates {
			t.Errorf("family %q lists text_to_image in Go operations=%v, Python adapter generates=%v: list text_to_image exactly when the adapter can generate without an input image", name, servesGeneration, p.Generates)
		}
	}
	for name := range pyFams {
		if _, ok := diffusersFamilies[name]; !ok {
			t.Errorf("family %q has a Python adapter (_diffusers.py) but no row in internal/models/families.go: add the DiffusersFamily row so models can name it", name)
		}
	}
}

// TestDiffusersFamiliesAreCompleteEnoughToImport asserts every family declares
// what `models import` copies onto an imported entry, so a repo of a known
// family becomes a runnable model with no operator-supplied fields.
func TestDiffusersFamiliesAreCompleteEnoughToImport(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range DiffusersFamilies() {
		if len(f.Operations) == 0 {
			t.Errorf("family %q declares no operations: list the ops its adapter serves in families.go", f.Name)
		}
		for _, op := range f.Operations {
			if !r.IsOperation(op) {
				t.Errorf("family %q lists %q, which is not an operation in the capability manifests", f.Name, op)
			}
		}
		if f.MinRuntime == "" {
			t.Errorf("family %q declares no min_runtime: say which diffusers version it needs", f.Name)
		}
		if !f.Ready && f.Pending == "" {
			t.Errorf("family %q is not ready but says nothing about what blocks it: fill Pending", f.Name)
		}
		got, ok := DiffusersFamilyForPipelineClass(f.PipelineClass)
		if !ok || got.Name != f.Name {
			t.Errorf("pipeline class %q does not resolve back to family %q (two families share a pipeline class?)", f.PipelineClass, f.Name)
		}
	}
	if _, ok := DiffusersFamilyForPipelineClass("NoSuchPipeline"); ok {
		t.Error("an unknown pipeline class must not resolve to a family")
	}
}
