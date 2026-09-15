package ai

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/storage"
	"image-tools/internal/technique"
)

type fakeWarmRunner struct {
	calls int
	err   error
}

func (f *fakeWarmRunner) Run(context.Context, string, []string) error {
	f.calls++
	return f.err
}

func (f *fakeWarmRunner) Close() error { return nil }

// TestRegisterProviders_SatisfiesBootInvariant proves the real provider set
// covers every Phase-3 AI op with a standalone (non-ComfyUI) backend, so the
// boot-time backends.Validate invariant holds.
func TestRegisterProviders_SatisfiesBootInvariant(t *testing.T) {
	reg := backends.New()
	if err := RegisterProviders(reg, func(string) (string, error) { return "/bin/x", nil }, nil, ""); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := reg.Validate(); err != nil {
		t.Fatalf("boot invariant failed: %v", err)
	}
	for _, op := range Names() {
		if len(reg.Providers(op)) == 0 {
			t.Errorf("op %q has no registered provider", op)
		}
	}
}

// TestProgramName_PythonUsesVenvInterpreter proves the isolation seam: a Python
// backend invokes the scenario's absolute venv interpreter when one is set,
// resolves to NO program (empty — never a bare host "python3") when it is not,
// and a non-Python backend is never rewritten to the interpreter (its alias/PATH
// resolution is untouched).
func TestProgramName_PythonUsesVenvInterpreter(t *testing.T) {
	const venv = "/data/image-tools/pyenv/bin/python"

	py := &execProvider{name: "diffusers", program: "python3", pythonInterpreter: venv}
	if got := py.programName(); got != venv {
		t.Errorf("python backend with venv: programName() = %q, want %q", got, venv)
	}

	// No venv → no fallback to the shared host python3 (that is the contamination
	// the isolation seam exists to prevent). programName() is empty; Availability/
	// Execute then report the backend unavailable.
	pyNoVenv := &execProvider{name: "diffusers", program: "python3"}
	if got := pyNoVenv.programName(); got != "" {
		t.Errorf("python backend without venv must NOT fall back to PATH python3, got %q", got)
	}

	// A non-python backend must ignore the interpreter entirely.
	sd := &execProvider{name: "sd", program: "sd", pythonInterpreter: venv}
	if got := sd.programName(); got != "sd" {
		t.Errorf("non-python backend must not use the venv interpreter, got %q", got)
	}

	// GPU-alias resolution for non-python backends is unaffected by the seam.
	sdGPU := &execProvider{
		name: "sd", program: "sd", programAliases: []string{"sd-gpu"}, pythonInterpreter: venv,
		lookPath: func(f string) (string, error) {
			if f == "sd-gpu" {
				return "/usr/bin/sd-gpu", nil
			}
			return "", errors.New("not found")
		},
	}
	if got := sdGPU.programName(); got != "sd-gpu" {
		t.Errorf("alias resolution broken by seam: got %q, want sd-gpu", got)
	}
}

// TestPythonBackendUnavailableWithoutVenv proves the isolation contract end to
// end: with no provisioned interpreter a Python backend is reported UNAVAILABLE
// (with remediation) and refuses to Execute — it never probes or runs a bare host
// python3. The lookPath is wired to "succeed" for any program so a regression that
// reintroduced a PATH fallback would wrongly flip the backend available and fail.
func TestPythonBackendUnavailableWithoutVenv(t *testing.T) {
	p := &execProvider{
		name:      "diffusers",
		program:   "python3",
		imports:   []string{"diffusers", "torch", "PIL"},
		provision: deriveProvision("uv"),
		lookPath:  func(string) (string, error) { return "/usr/bin/python3", nil },
		checkPy:   func(context.Context, string, []string) error { return nil },
	}

	a := p.Availability(context.Background())
	if a.Available {
		t.Fatalf("python backend must be unavailable without a provisioned venv: %+v", a)
	}
	if !strings.Contains(a.Detail, "not provisioned") {
		t.Errorf("detail should explain the venv is not provisioned: %q", a.Detail)
	}
	if !strings.Contains(a.Provision, "vrooli host install uv") {
		t.Errorf("provision should point at uv: %q", a.Provision)
	}

	req := backends.Request{
		Operation: "edit_instruct",
		Model:     models.Model{ID: "m1"},
		ModelDir:  "/models/m1",
		InputKeys: []string{"/in.png"},
		Output:    storage.OutputTarget{LocalPath: "/out.png"},
	}
	if _, err := p.Execute(context.Background(), req); err == nil || !strings.Contains(err.Error(), "not provisioned") {
		t.Fatalf("Execute must refuse when the venv is unprovisioned, got %v", err)
	}
}

// TestRegisterProviders_ThreadsInterpreterToPythonBackendsOnly proves the venv
// interpreter reaches every Python backend and no binary backend.
func TestRegisterProviders_ThreadsInterpreterToPythonBackendsOnly(t *testing.T) {
	reg := backends.New()
	const venv = "/data/pyenv/bin/python"
	if err := RegisterProviders(reg, func(string) (string, error) { return venv, nil }, nil, venv); err != nil {
		t.Fatalf("register: %v", err)
	}
	for _, s := range providerSpecs() {
		for _, p := range reg.Providers(s.ops()[0]) {
			ep, ok := p.(*execProvider)
			if !ok || ep.name != s.name {
				continue
			}
			if s.isPython() {
				if ep.pythonInterpreter != venv {
					t.Errorf("python backend %q did not receive venv interpreter (got %q)", s.name, ep.pythonInterpreter)
				}
				if ep.programName() != venv {
					t.Errorf("python backend %q programName = %q, want venv", s.name, ep.programName())
				}
			} else if ep.pythonInterpreter != "" {
				t.Errorf("binary backend %q must not carry a venv interpreter (got %q)", s.name, ep.pythonInterpreter)
			}
		}
	}
}

// TestExecProvider_Availability proves availability tracks the program on PATH.
func TestExecProvider_Availability(t *testing.T) {
	reg := backends.New()
	missing := func(string) (string, error) { return "", errors.New("not found") }
	if err := RegisterProviders(reg, missing, nil, ""); err != nil {
		t.Fatalf("register: %v", err)
	}
	for _, p := range reg.Providers("upscale") {
		if p.Available(context.Background()) {
			t.Errorf("provider %q should be unavailable when its program is missing", p.Name())
		}
	}
}

func TestExecProvider_AvailabilityChecksPythonImports(t *testing.T) {
	p := &execProvider{
		name:              "onnxruntime",
		program:           "python3",
		pythonInterpreter: "/data/pyenv/bin/python",
		imports:           []string{"onnxruntime", "PIL", "numpy"},
		lookPath: func(string) (string, error) {
			return "/usr/bin/python3", nil
		},
		checkPy: func(_ context.Context, _ string, modules []string) error {
			if !slices.Equal(modules, []string{"onnxruntime", "PIL", "numpy"}) {
				t.Fatalf("modules = %v", modules)
			}
			return errors.New("missing module")
		},
	}

	a := p.Availability(context.Background())
	if a.Available {
		t.Fatalf("provider should be unavailable when imports fail: %+v", a)
	}
	if !strings.Contains(a.Detail, "imports failed") {
		t.Fatalf("detail should explain import failure: %q", a.Detail)
	}
}

func TestExecProvider_ONNXRuntimeAvailabilityReportsExecutionProviders(t *testing.T) {
	p := &execProvider{
		name:              "onnxruntime",
		program:           "python3",
		pythonInterpreter: "/data/pyenv/bin/python",
		imports:           []string{"onnxruntime", "PIL", "numpy"},
		lookPath: func(string) (string, error) {
			return "/usr/bin/python3", nil
		},
		checkPy: func(context.Context, string, []string) error { return nil },
		checkONNX: func(context.Context, string) ([]string, error) {
			return []string{"AzureExecutionProvider", "CPUExecutionProvider"}, nil
		},
	}

	a := p.Availability(context.Background())
	if !a.Available {
		t.Fatalf("CPUExecutionProvider should make ONNX sidecar available: %+v", a)
	}
	if !strings.Contains(a.Detail, "AzureExecutionProvider,CPUExecutionProvider") {
		t.Fatalf("detail should list ONNX Runtime providers: %q", a.Detail)
	}
	if !strings.Contains(a.Detail, "CUDAExecutionProvider unavailable") {
		t.Fatalf("detail should explain CPU-only CUDA state: %q", a.Detail)
	}
}

func TestExecProvider_ONNXRuntimeAvailabilityRequiresCPUProvider(t *testing.T) {
	p := &execProvider{
		name:              "onnxruntime",
		program:           "python3",
		pythonInterpreter: "/data/pyenv/bin/python",
		imports:           []string{"onnxruntime", "PIL", "numpy"},
		lookPath: func(string) (string, error) {
			return "/usr/bin/python3", nil
		},
		checkPy: func(context.Context, string, []string) error { return nil },
		checkONNX: func(context.Context, string) ([]string, error) {
			return []string{"CUDAExecutionProvider"}, nil
		},
	}

	a := p.Availability(context.Background())
	if a.Available {
		t.Fatalf("ONNX sidecar should be unavailable without CPUExecutionProvider: %+v", a)
	}
	if !strings.Contains(a.Detail, "CPUExecutionProvider missing") {
		t.Fatalf("detail should explain missing CPU provider: %q", a.Detail)
	}
}

func TestExecProvider_ONNXRuntimeAvailabilityReportsProviderProbeFailure(t *testing.T) {
	p := &execProvider{
		name:              "onnxruntime",
		program:           "python3",
		pythonInterpreter: "/data/pyenv/bin/python",
		imports:           []string{"onnxruntime", "PIL", "numpy"},
		lookPath: func(string) (string, error) {
			return "/usr/bin/python3", nil
		},
		checkPy:   func(context.Context, string, []string) error { return nil },
		checkONNX: func(context.Context, string) ([]string, error) { return nil, errors.New("probe failed") },
	}

	a := p.Availability(context.Background())
	if a.Available {
		t.Fatalf("ONNX sidecar should be unavailable when provider probe fails: %+v", a)
	}
	if !strings.Contains(a.Detail, "provider probe failed") {
		t.Fatalf("detail should explain provider probe failure: %q", a.Detail)
	}
}

func TestExecProvider_ONNXRuntimeProviderDetailReportsCUDAAvailable(t *testing.T) {
	p := &execProvider{program: "python3", imports: []string{"onnxruntime", "PIL", "numpy"}}
	detail := p.onnxRuntimeProviderDetail("/usr/bin/python3", []string{"CUDAExecutionProvider", "CPUExecutionProvider"})
	if !strings.Contains(detail, "CUDAExecutionProvider available") {
		t.Fatalf("detail should report CUDA availability: %q", detail)
	}
}

func TestDefaultCheckONNXRuntimeProviders(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-python")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nprintf '[\"CPUExecutionProvider\"]\\n'\n"), 0o755); err != nil {
		t.Fatalf("write fake python: %v", err)
	}

	providers, err := defaultCheckONNXRuntimeProviders(context.Background(), script)
	if err != nil {
		t.Fatalf("provider probe: %v", err)
	}
	if !slices.Equal(providers, []string{"CPUExecutionProvider"}) {
		t.Fatalf("providers = %v", providers)
	}
}

func TestDefaultCheckONNXRuntimeProvidersErrors(t *testing.T) {
	dir := t.TempDir()
	badJSON := filepath.Join(dir, "bad-json")
	if err := os.WriteFile(badJSON, []byte("#!/bin/sh\nprintf 'not-json\\n'\n"), 0o755); err != nil {
		t.Fatalf("write bad-json script: %v", err)
	}
	if _, err := defaultCheckONNXRuntimeProviders(context.Background(), badJSON); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}

	failing := filepath.Join(dir, "failing")
	if err := os.WriteFile(failing, []byte("#!/bin/sh\nprintf 'boom' >&2\nexit 7\n"), 0o755); err != nil {
		t.Fatalf("write failing script: %v", err)
	}
	if _, err := defaultCheckONNXRuntimeProviders(context.Background(), failing); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected command output in error, got %v", err)
	}
}

func TestExecProvider_UsesWarmRunnerBeforeOneShot(t *testing.T) {
	warm := &fakeWarmRunner{}
	var oneShotCalls int
	p := &execProvider{
		name:              "onnxruntime",
		program:           "python3",
		pythonInterpreter: "/data/pyenv/bin/python",
		techniques:        technique.Single("depth_map", technique.OnnxSidecar),
		warm:              warm,
		run: func(context.Context, string, []string) error {
			oneShotCalls++
			return nil
		},
	}
	req := backends.Request{
		Operation: "depth_map",
		Model:     models.Model{ID: "m1"},
		ModelDir:  "/models/m1",
		InputKeys: []string{"/in.png"},
		Output:    storage.OutputTarget{LocalPath: "/out.png"},
	}
	res, err := p.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if warm.calls != 1 || oneShotCalls != 0 {
		t.Fatalf("warm calls=%d one-shot calls=%d", warm.calls, oneShotCalls)
	}
	if res.Meta["runner"] != "warm" {
		t.Fatalf("expected warm runner metadata, got %+v", res.Meta)
	}
}

func TestExecProvider_FallsBackWhenWarmRunnerFails(t *testing.T) {
	warm := &fakeWarmRunner{err: errors.New("worker exited")}
	var oneShotCalls int
	p := &execProvider{
		name:              "onnxruntime",
		program:           "python3",
		pythonInterpreter: "/data/pyenv/bin/python",
		techniques:        technique.Single("depth_map", technique.OnnxSidecar),
		warm:              warm,
		run: func(context.Context, string, []string) error {
			oneShotCalls++
			return nil
		},
	}
	req := backends.Request{
		Operation: "depth_map",
		Model:     models.Model{ID: "m1"},
		ModelDir:  "/models/m1",
		InputKeys: []string{"/in.png"},
		Output:    storage.OutputTarget{LocalPath: "/out.png"},
	}
	res, err := p.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if warm.calls != 1 || oneShotCalls != 1 {
		t.Fatalf("warm calls=%d one-shot calls=%d", warm.calls, oneShotCalls)
	}
	if res.Meta["runner"] == "warm" {
		t.Fatalf("fallback result should not claim warm runner: %+v", res.Meta)
	}
}

func TestExecProvider_ExecutionErrors(t *testing.T) {
	req := backends.Request{
		Operation: "depth_map",
		Model:     models.Model{ID: "m1"},
		InputKeys: []string{"/in.png"},
		Output:    storage.OutputTarget{LocalPath: "/out.png"},
	}
	p := &execProvider{name: "onnxruntime", program: "python3", pythonInterpreter: "/data/pyenv/bin/python", techniques: technique.Single("depth_map", technique.OnnxSidecar)}
	if _, err := p.Execute(context.Background(), backends.Request{Operation: "depth_map", Model: models.Model{ID: "m1"}, InputKeys: []string{"/in.png"}}); err == nil {
		t.Fatal("expected missing output path error")
	}
	p.run = func(context.Context, string, []string) error { return errors.New("boom") }
	if _, err := p.Execute(context.Background(), req); err == nil || !strings.Contains(err.Error(), "execution failed") {
		t.Fatalf("expected execution failure, got %v", err)
	}
	p.techniques = technique.Single("depth_map", func(backends.Request, string) ([]string, error) { return nil, errors.New("bad args") })
	if _, err := p.Execute(context.Background(), req); err == nil || !strings.Contains(err.Error(), "build args") {
		t.Fatalf("expected build error, got %v", err)
	}
}

func TestDefaultRunHelpers(t *testing.T) {
	if err := defaultRun(context.Background(), "sh", []string{"-c", "printf ok"}); err != nil {
		t.Fatalf("defaultRun success: %v", err)
	}
	if err := defaultRun(context.Background(), "sh", []string{"-c", "printf bad; exit 7"}); err == nil || !strings.Contains(err.Error(), "bad") {
		t.Fatalf("defaultRun should include command output on failure, got %v", err)
	}
	out, err := defaultRunOutput(context.Background(), "sh", []string{"-c", "printf caption"})
	if err != nil || string(out) != "caption" {
		t.Fatalf("defaultRunOutput = %q, %v", out, err)
	}
	if _, err := defaultRunOutput(context.Background(), "sh", []string{"-c", "printf bad; exit 8"}); err == nil || !strings.Contains(err.Error(), "bad") {
		t.Fatalf("defaultRunOutput should include command output on failure, got %v", err)
	}
}

func TestLlamaCppProvider_AvailabilityAndExecute(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "caption.json")
	provider := &llamaCppProvider{
		lookPath: func(file string) (string, error) {
			if file == "llama-mtmd-cli" {
				return "/usr/bin/llama-mtmd-cli", nil
			}
			return "", errors.New("not found")
		},
		run: func(_ context.Context, name string, args []string) ([]byte, error) {
			if name != "/usr/bin/llama-mtmd-cli" {
				t.Fatalf("program = %q", name)
			}
			for _, want := range []string{"--image", "/in.png", "--mmproj", "/models/m1/mmproj.gguf"} {
				if !slices.Contains(args, want) {
					t.Fatalf("argv %v missing %q", args, want)
				}
			}
			return []byte("a small red square\n"), nil
		},
	}
	if !provider.Available(context.Background()) {
		t.Fatalf("provider should be available when llama-mtmd-cli resolves")
	}
	req := backends.Request{
		Operation: "caption",
		Model: models.Model{
			ID: "m1",
			Source: models.Source{Assets: []models.Asset{
				{Filename: "model.gguf", Kind: models.ArtifactGGUF},
				{Filename: "mmproj.gguf", Kind: models.ArtifactGGUF},
			}},
		},
		ModelDir:  "/models/m1",
		InputKeys: []string{"/in.png"},
		Output:    storage.OutputTarget{LocalPath: outPath},
	}
	res, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.OutputRef != outPath {
		t.Fatalf("output ref = %q", res.OutputRef)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if got["caption"] != "a small red square" || got["backend"] != "llama.cpp" || got["model"] != "m1" {
		t.Fatalf("bad caption payload: %+v", got)
	}
}

func TestLlamaCppProvider_AvailabilityMissingBinary(t *testing.T) {
	provider := &llamaCppProvider{lookPath: func(string) (string, error) { return "", errors.New("not found") }}
	a := provider.Availability(context.Background())
	if a.Available {
		t.Fatalf("provider should be unavailable when llama.cpp binaries are absent")
	}
	if !strings.Contains(a.Detail, "llama-mtmd-cli/llama-cli") || !strings.Contains(a.Provision, "vrooli host install llama-cpp") {
		t.Fatalf("availability should include binary and derived host-tool remediation: %+v", a)
	}
}

func TestLlamaCppProvider_FallsBackToLlamaCli(t *testing.T) {
	provider := &llamaCppProvider{lookPath: func(file string) (string, error) {
		if file == "llama-cli" {
			return "/usr/bin/llama-cli", nil
		}
		return "", errors.New("not found")
	}}
	a := provider.Availability(context.Background())
	if !a.Available || !strings.Contains(a.Detail, "/usr/bin/llama-cli") {
		t.Fatalf("expected llama-cli fallback availability, got %+v", a)
	}
}

func TestLlamaCppProvider_ExecuteErrors(t *testing.T) {
	baseReq := backends.Request{
		Operation: "caption",
		Model: models.Model{
			ID: "m1",
			Source: models.Source{Assets: []models.Asset{
				{Filename: "model.gguf", Kind: models.ArtifactGGUF},
				{Filename: "mmproj.gguf", Kind: models.ArtifactGGUF},
			}},
		},
		ModelDir:  "/models/m1",
		InputKeys: []string{"/in.png"},
		Output:    storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "caption.json")},
	}
	provider := &llamaCppProvider{
		lookPath: func(string) (string, error) { return "/usr/bin/llama-mtmd-cli", nil },
		run:      func(context.Context, string, []string) ([]byte, error) { return nil, errors.New("run failed") },
	}
	if _, err := provider.Execute(context.Background(), baseReq); err == nil || !strings.Contains(err.Error(), "execution failed") {
		t.Fatalf("expected run failure, got %v", err)
	}
	provider.run = func(context.Context, string, []string) ([]byte, error) { return []byte(" \n\t"), nil }
	if _, err := provider.Execute(context.Background(), baseReq); err == nil || !strings.Contains(err.Error(), "empty caption") {
		t.Fatalf("expected empty caption error, got %v", err)
	}
	if _, err := provider.Execute(context.Background(), backends.Request{Operation: "caption"}); err == nil || !strings.Contains(err.Error(), "requires a local output path") {
		t.Fatalf("expected missing output error, got %v", err)
	}
	badReq := baseReq
	badReq.Model.Source.Assets = nil
	if _, err := provider.Execute(context.Background(), badReq); err == nil || !strings.Contains(err.Error(), "build args") {
		t.Fatalf("expected build args error, got %v", err)
	}
	missing := &llamaCppProvider{lookPath: func(string) (string, error) { return "", errors.New("not found") }}
	if _, err := missing.Execute(context.Background(), baseReq); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

// TestExecProvider_PrefersGPUAlias verifies the stable-diffusion.cpp provider
// resolves its GPU-build alias ("sd-gpu") ahead of the base "sd" command when
// the alias is on PATH, and falls back to "sd" otherwise. This is what lets
// `vrooli host install sd-gpu` flip the backend to the GPU build (and the tier
// to local-gpu) without a restart or a launcher collision.
func TestExecProvider_PrefersGPUAlias(t *testing.T) {
	tests := []struct {
		name    string
		present map[string]bool
		want    string
	}{
		{name: "alias present wins", present: map[string]bool{"sd-gpu": true, "sd": true}, want: "sd-gpu"},
		{name: "only base present", present: map[string]bool{"sd": true}, want: "sd"},
		{name: "alias only", present: map[string]bool{"sd-gpu": true}, want: "sd-gpu"},
		{name: "neither present falls back to base", present: map[string]bool{}, want: "sd"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &execProvider{
				program:        "sd",
				programAliases: []string{"sd-gpu"},
				lookPath: func(name string) (string, error) {
					if tc.present[name] {
						return "/fake/bin/" + name, nil
					}
					return "", errors.New("not found")
				},
			}
			if got := p.programName(); got != tc.want {
				t.Errorf("programName() = %q, want %q", got, tc.want)
			}
		})
	}
}
