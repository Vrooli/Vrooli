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
			args: mustArgs(t, buildRealesrgan, req("upscale", []string{"/in.png"}, map[string]string{"scale": "2"})),
			want: []string{"-i", "/in.png", "-o", "/out.png", "-s", "2", "-n", "m1"},
		},
		{
			name: "rembg",
			args: mustArgs(t, buildRembg, req("background_removal", []string{"/in.png"}, nil)),
			want: []string{"i", "-m", "m1", "/in.png", "/out.png"},
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
			args: mustArgs(t, buildDiffusers, req("edit_instruct", []string{"/in.png"}, map[string]string{"prompt": "make it winter", "cfg_scale": "7.5", "strength": "1.5", "seed": "42"})),
			want: []string{"-m", "image_tools_sidecar.edit_instruct", "--model", "/models/m1", "--image", "/in.png", "--prompt", "make it winter", "--out", "/out.png", "--guidance", "7.5", "--image-guidance", "1.5", "--seed", "42"},
		},
		{
			name: "diffusers edit_instruct needs input",
			err:  argErr(buildDiffusers, req("edit_instruct", nil, map[string]string{"prompt": "x"})),
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
			name: "onnx rejects unknown op",
			err:  argErr(buildOnnxSidecar, req("text_to_image", []string{"/in.png"}, nil)),
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
	if !strings.Contains(a.Detail, "llama-mtmd-cli/llama-cli") || !strings.Contains(a.Provision, "Scenario Dependency Analyzer") {
		t.Fatalf("availability should include binary and provisioning detail: %+v", a)
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
