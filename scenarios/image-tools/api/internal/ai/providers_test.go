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
	if err := RegisterProviders(reg, func(string) (string, error) { return "/bin/x", nil }, nil); err != nil {
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

// TestExecProvider_Availability proves availability tracks the program on PATH.
func TestExecProvider_Availability(t *testing.T) {
	reg := backends.New()
	missing := func(string) (string, error) { return "", errors.New("not found") }
	if err := RegisterProviders(reg, missing, nil); err != nil {
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
		name:    "onnxruntime",
		program: "python3",
		imports: []string{"onnxruntime", "PIL", "numpy"},
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
		name:    "onnxruntime",
		program: "python3",
		imports: []string{"onnxruntime", "PIL", "numpy"},
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
		name:    "onnxruntime",
		program: "python3",
		imports: []string{"onnxruntime", "PIL", "numpy"},
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
		name:    "onnxruntime",
		program: "python3",
		imports: []string{"onnxruntime", "PIL", "numpy"},
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
		name:    "onnxruntime",
		program: "python3",
		build:   buildOnnxSidecar,
		warm:    warm,
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
		name:    "onnxruntime",
		program: "python3",
		build:   buildOnnxSidecar,
		warm:    warm,
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
	p := &execProvider{name: "onnxruntime", program: "python3", build: buildOnnxSidecar}
	if _, err := p.Execute(context.Background(), backends.Request{Operation: "depth_map", Model: models.Model{ID: "m1"}, InputKeys: []string{"/in.png"}}); err == nil {
		t.Fatal("expected missing output path error")
	}
	p.run = func(context.Context, string, []string) error { return errors.New("boom") }
	if _, err := p.Execute(context.Background(), req); err == nil || !strings.Contains(err.Error(), "execution failed") {
		t.Fatalf("expected execution failure, got %v", err)
	}
	p.build = func(backends.Request, string) ([]string, error) { return nil, errors.New("bad args") }
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

// TestArgBuilders pins each backend's argv assembly (the part testable without
// the real binary).
func TestArgBuilders(t *testing.T) {
	req := func(op string, in []string, params map[string]string) backends.Request {
		return backends.Request{
			Operation: op,
			Model:     models.Model{ID: "m1"},
			ModelDir:  "/models/m1",
			InputKeys: in,
			Output:    storage.OutputTarget{LocalPath: "/out.png"},
			Params:    params,
		}
	}
	cases := []struct {
		name string
		args []string
		err  bool
		want []string // substrings that must appear in argv
	}{
		{
			name: "sd text",
			args: mustArgs(t, buildStableDiffusionCpp, req("text_to_image", nil, map[string]string{"prompt": "cat", "steps": "30"})),
			want: []string{"-m", "/models/m1", "-p", "cat", "--steps", "30", "-o", "/out.png"},
		},
		{
			name: "sd img2img needs input",
			err:  argErr(buildStableDiffusionCpp, req("image_to_image", nil, map[string]string{"prompt": "x"})),
		},
		{
			name: "iopaint needs mask",
			err:  argErr(buildIopaint, req("object_removal", []string{"/in.png"}, nil)),
		},
		{
			name: "realesrgan scale",
			// Uses the release's bundled model name (resolved relative to the
			// binary), not a -m dir or the image-tools model id.
			args: mustArgs(t, buildRealesrgan, req("upscale", []string{"/in.png"}, map[string]string{"scale": "2"})),
			want: []string{"-i", "/in.png", "-o", "/out.png", "-s", "2", "-n", "realesr-animevideov3"},
		},
		{
			name: "realesrgan denoise",
			args: mustArgs(t, buildRealesrgan, req("denoise", []string{"/in.png"}, map[string]string{"denoise": "3"})),
			want: []string{"-i", "/in.png", "-o", "/out.png", "-s", "4", "-n", "realesr-animevideov3"},
		},
		{
			name: "rembg",
			args: mustArgs(t, buildRembg, req("background_removal", []string{"/in.png"}, nil)),
			want: []string{"i", "-m", "m1", "/in.png", "/out.png"},
		},
		{
			name: "rembg background_replace",
			args: mustArgs(t, buildRembg, req("background_replace", []string{"/in.png"}, map[string]string{"background_color": "240,240,240,255"})),
			want: []string{"i", "-m", "m1", "--bgcolor", "240,240,240,255", "/in.png", "/out.png"},
		},
		{
			name: "rembg background_replace needs color",
			err:  argErr(buildRembg, req("background_replace", []string{"/in.png"}, nil)),
		},
		{
			name: "llama.cpp caption",
			args: mustArgs(t, buildLlamaCppCaption, func() backends.Request {
				r := req("caption", []string{"/in.png"}, map[string]string{"prompt": "alt text", "max_tokens": "48", "temperature": "0.2"})
				r.Model.Source.Assets = []models.Asset{
					{Filename: "model.gguf", Kind: models.ArtifactGGUF},
					{Filename: "mmproj-model.gguf", Kind: models.ArtifactGGUF},
				}
				return r
			}()),
			want: []string{"-m", "/models/m1/model.gguf", "--mmproj", "/models/m1/mmproj-model.gguf", "--image", "/in.png", "-p", "alt text", "-n", "48", "--temp", "0.2"},
		},
		{
			name: "llama.cpp caption needs mmproj",
			err: func() bool {
				r := req("caption", []string{"/in.png"}, nil)
				r.Model.Source.Assets = []models.Asset{{Filename: "model.gguf", Kind: models.ArtifactGGUF}}
				return argErr(buildLlamaCppCaption, r)
			}(),
		},
		{
			name: "diffusers edit_instruct",
			args: mustArgs(t, buildDiffusers, func() backends.Request {
				r := req("edit_instruct", []string{"/in.png"}, map[string]string{"prompt": "make it winter", "cfg_scale": "7.5", "strength": "1.5", "seed": "42", "steps": "30", "negative_prompt": "blurry"})
				r.Model.Runtime.Family = "instruct-pix2pix"
				return r
			}()),
			// The pipeline class is selected by --family, not hardcoded. --steps and
			// --negative-prompt are passed only when explicitly requested.
			want: []string{"-m", "image_tools_sidecar.edit_instruct", "--model", "/models/m1", "--family", "instruct-pix2pix", "--image", "/in.png", "--prompt", "make it winter", "--out", "/out.png", "--steps", "30", "--guidance", "7.5", "--image-guidance", "1.5", "--negative-prompt", "blurry", "--seed", "42"},
		},
		{
			name: "diffusers edit_instruct needs input",
			err: func() bool {
				r := req("edit_instruct", nil, map[string]string{"prompt": "x"})
				r.Model.Runtime.Family = "instruct-pix2pix"
				return argErr(buildDiffusers, r)
			}(),
		},
		{
			name: "diffusers edit_instruct needs a runtime family",
			err:  argErr(buildDiffusers, req("edit_instruct", []string{"/in.png"}, map[string]string{"prompt": "x"})),
		},
		{
			name: "diffusers inpaint via dispatcher",
			args: mustArgs(t, buildDiffusers, req("inpaint", []string{"/in.png", "/mask.png"}, map[string]string{"prompt": "sky"})),
			want: []string{"-m", "image_tools_sidecar.inpaint", "--mask", "/mask.png", "--prompt", "sky"},
		},
		{
			name: "diffusers outpaint reuses the inpaint sidecar",
			args: mustArgs(t, buildDiffusers, req("outpaint", []string{"/in.png", "/mask.png"}, map[string]string{"prompt": "extend the beach"})),
			want: []string{"-m", "image_tools_sidecar.inpaint", "--mask", "/mask.png", "--prompt", "extend the beach"},
		},
		{
			name: "diffusers background_replace reuses the inpaint sidecar",
			args: mustArgs(t, buildDiffusers, req("background_replace", []string{"/in.png", "/mask.png"}, map[string]string{"prompt": "a studio backdrop"})),
			want: []string{"-m", "image_tools_sidecar.inpaint", "--mask", "/mask.png", "--prompt", "a studio backdrop"},
		},
		{
			name: "diffusers outpaint needs a mask",
			err:  argErr(buildDiffusers, req("outpaint", []string{"/in.png"}, map[string]string{"prompt": "x"})),
		},
		{
			name: "onnx colorize sidecar dispatch",
			args: mustArgs(t, buildOnnxSidecar, req("colorize", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.colorize", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx depth sidecar dispatch",
			args: mustArgs(t, buildOnnxSidecar, req("depth_map", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.depth", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx deblur sidecar dispatch",
			args: mustArgs(t, buildOnnxSidecar, req("deblur", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.deblur", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx detection sidecar dispatch",
			args: mustArgs(t, buildOnnxSidecar, req("object_detection", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.detect", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx segment sidecar dispatch",
			args: mustArgs(t, buildOnnxSidecar, req("segment", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.segment", "--model", "/models/m1", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx tagging sidecar dispatch",
			args: mustArgs(t, buildOnnxSidecar, req("tagging", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.tagging", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx nsfw sidecar dispatch",
			args: mustArgs(t, buildOnnxSidecar, req("nsfw_classify", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.nsfw", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx embedding sidecar dispatch",
			args: mustArgs(t, buildOnnxSidecar, req("embedding", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.embedding", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "python-sidecar colorize dispatch",
			args: mustArgs(t, buildPythonSidecar, req("colorize", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.colorize", "--model", "/models/m1", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "python-sidecar face restore dispatch",
			args: mustArgs(t, buildPythonSidecar, req("face_restore", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.restore", "--model", "/models/m1", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "python-sidecar old photo restore dispatch",
			args: mustArgs(t, buildPythonSidecar, req("old_photo_restore", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.restore", "--model", "/models/m1", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx rejects unknown op",
			err:  argErr(buildOnnxSidecar, req("text_to_image", []string{"/in.png"}, nil)),
		},
		{
			name: "python-sidecar rejects unknown op",
			err:  argErr(buildPythonSidecar, req("text_to_image", []string{"/in.png"}, nil)),
		},
		{
			name: "diffusers rejects unknown op",
			err:  argErr(buildDiffusers, req("text_to_image", []string{"/in.png"}, nil)),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err {
				return // error path asserted by argErr
			}
			for _, w := range tc.want {
				if !slices.Contains(tc.args, w) {
					t.Errorf("argv %v missing %q", tc.args, w)
				}
			}
		})
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

func TestOnnxModelPathResolution(t *testing.T) {
	req := backends.Request{Model: models.Model{Source: models.Source{Assets: []models.Asset{
		{Filename: "fallback.bin", Kind: models.ArtifactGeneric},
		{Filename: "model.onnx", Kind: models.ArtifactONNX},
	}}}}
	if got := onnxModelPath(req, "/models/m1"); got != filepath.Join("/models/m1", "model.onnx") {
		t.Fatalf("onnx path = %q", got)
	}
	req.Model.Source.Assets = []models.Asset{{Filename: "first.bin"}}
	if got := onnxModelPath(req, "/models/m1"); got != filepath.Join("/models/m1", "first.bin") {
		t.Fatalf("fallback asset path = %q", got)
	}
	req.Model.Source.Assets = nil
	if got := onnxModelPath(req, "/models/m1"); got != "/models/m1" {
		t.Fatalf("empty assets path = %q", got)
	}
}

func mustArgs(t *testing.T, b argBuilder, req backends.Request) []string {
	t.Helper()
	args, err := b(req, req.ModelDir)
	if err != nil {
		t.Fatalf("build args: %v", err)
	}
	return args
}

func argErr(b argBuilder, req backends.Request) bool {
	_, err := b(req, req.ModelDir)
	return err != nil
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
