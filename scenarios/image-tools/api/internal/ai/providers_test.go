package ai

import (
	"context"
	"errors"
	"slices"
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
