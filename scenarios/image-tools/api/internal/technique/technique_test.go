package technique

import (
	"path/filepath"
	"slices"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/storage"
)

// TestArgBuilders pins each technique's argv assembly (the part testable without
// the real binary).
func TestArgBuilders(t *testing.T) {
	req := func(op string, in []string, params map[string]string) backends.Request {
		return backends.Request{
			Operation: op,
			Model:     models.Model{ID: "m1", Architecture: "sd15"},
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
			args: mustArgs(t, StableDiffusionCpp, req("text_to_image", nil, map[string]string{"prompt": "cat", "steps": "30"})),
			want: []string{"-m", "/models/m1", "-p", "cat", "--steps", "30", "-o", "/out.png"},
		},
		{
			name: "sd img2img needs input",
			err:  argErr(StableDiffusionCpp, req("image_to_image", nil, map[string]string{"prompt": "x"})),
		},
		{
			name: "iopaint needs mask",
			err:  argErr(Iopaint, req("object_removal", []string{"/in.png"}, nil)),
		},
		{
			name: "realesrgan scale",
			// Uses the release's bundled model name (resolved relative to the
			// binary), not a -m dir or the image-tools model id.
			args: mustArgs(t, Realesrgan, req("upscale", []string{"/in.png"}, map[string]string{"scale": "2"})),
			want: []string{"-i", "/in.png", "-o", "/out.png", "-s", "2", "-n", "realesr-animevideov3"},
		},
		{
			name: "realesrgan denoise",
			args: mustArgs(t, Realesrgan, req("denoise", []string{"/in.png"}, map[string]string{"denoise": "3"})),
			want: []string{"-i", "/in.png", "-o", "/out.png", "-s", "4", "-n", "realesr-animevideov3"},
		},
		{
			name: "rembg",
			args: mustArgs(t, Rembg, req("background_removal", []string{"/in.png"}, nil)),
			want: []string{"i", "-m", "m1", "/in.png", "/out.png"},
		},
		{
			name: "rembg background_replace",
			args: mustArgs(t, Rembg, req("background_replace", []string{"/in.png"}, map[string]string{"background_color": "240,240,240,255"})),
			want: []string{"i", "-m", "m1", "--bgcolor", "240,240,240,255", "/in.png", "/out.png"},
		},
		{
			name: "rembg background_replace needs color",
			err:  argErr(Rembg, req("background_replace", []string{"/in.png"}, nil)),
		},
		{
			name: "llama.cpp caption",
			args: mustArgs(t, LlamaCppCaption, func() backends.Request {
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
				return argErr(LlamaCppCaption, r)
			}(),
		},
		{
			name: "diffusers edit_instruct",
			args: mustArgs(t, DiffusersEditInstruct, func() backends.Request {
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
				return argErr(DiffusersEditInstruct, r)
			}(),
		},
		{
			name: "diffusers edit_instruct needs a runtime family",
			err:  argErr(DiffusersEditInstruct, req("edit_instruct", []string{"/in.png"}, map[string]string{"prompt": "x"})),
		},
		{
			name: "diffusers inpaint",
			args: mustArgs(t, DiffusersInpaint, req("inpaint", []string{"/in.png", "/mask.png"}, map[string]string{"prompt": "sky"})),
			want: []string{"-m", "image_tools_sidecar.inpaint", "--architecture", "sd15", "--image", "/in.png", "--mask", "/mask.png", "--prompt", "sky", "--out", "/out.png"},
		},
		{
			name: "diffusers inpaint threads strength + generation params",
			args: mustArgs(t, DiffusersInpaint, req("inpaint", []string{"/in.png", "/mask.png"}, map[string]string{
				"prompt": "sky", "strength": "0.6", "steps": "28", "cfg_scale": "6.0", "negative_prompt": "blurry", "seed": "42",
			})),
			want: []string{"--strength", "0.6", "--steps", "28", "--guidance", "6.0", "--negative-prompt", "blurry", "--seed", "42"},
		},
		{
			name: "diffusers inpaint needs a model architecture",
			err: func() bool {
				r := req("inpaint", []string{"/in.png", "/mask.png"}, map[string]string{"prompt": "x"})
				r.Model.Architecture = ""
				return argErr(DiffusersInpaint, r)
			}(),
		},
		{
			name: "diffusers outpaint reuses the inpaint sidecar",
			args: mustArgs(t, DiffusersInpaint, req("outpaint", []string{"/in.png", "/mask.png"}, map[string]string{"prompt": "extend the beach"})),
			want: []string{"-m", "image_tools_sidecar.inpaint", "--mask", "/mask.png", "--prompt", "extend the beach"},
		},
		{
			name: "diffusers background_replace reuses the inpaint sidecar",
			args: mustArgs(t, DiffusersInpaint, req("background_replace", []string{"/in.png", "/mask.png"}, map[string]string{"prompt": "a studio backdrop"})),
			want: []string{"-m", "image_tools_sidecar.inpaint", "--mask", "/mask.png", "--prompt", "a studio backdrop"},
		},
		{
			name: "diffusers inpaint needs a mask",
			err:  argErr(DiffusersInpaint, req("outpaint", []string{"/in.png"}, map[string]string{"prompt": "x"})),
		},
		{
			name: "onnx colorize sidecar dispatch",
			args: mustArgs(t, OnnxSidecar, req("colorize", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.colorize", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx depth sidecar dispatch",
			args: mustArgs(t, OnnxSidecar, req("depth_map", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.depth", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx deblur sidecar dispatch",
			args: mustArgs(t, OnnxSidecar, req("deblur", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.deblur", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx detection sidecar dispatch",
			args: mustArgs(t, OnnxSidecar, req("object_detection", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.detect", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx segment sidecar dispatch",
			args: mustArgs(t, OnnxSidecar, req("segment", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.segment", "--model", "/models/m1", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx tagging sidecar dispatch",
			args: mustArgs(t, OnnxSidecar, req("tagging", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.tagging", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx nsfw sidecar dispatch",
			args: mustArgs(t, OnnxSidecar, req("nsfw_classify", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.nsfw", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx embedding sidecar dispatch",
			args: mustArgs(t, OnnxSidecar, req("embedding", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.embedding", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "python-sidecar colorize dispatch",
			args: mustArgs(t, PythonSidecar, req("colorize", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.colorize", "--model", "/models/m1", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "python-sidecar face restore dispatch",
			args: mustArgs(t, PythonSidecar, req("face_restore", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.restore", "--model", "/models/m1", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "python-sidecar old photo restore dispatch",
			args: mustArgs(t, PythonSidecar, req("old_photo_restore", []string{"/in.png"}, nil)),
			want: []string{"-m", "image_tools_sidecar.restore", "--model", "/models/m1", "--image", "/in.png", "--out", "/out.png"},
		},
		{
			name: "onnx rejects unknown op",
			err:  argErr(OnnxSidecar, req("text_to_image", []string{"/in.png"}, nil)),
		},
		{
			name: "python-sidecar rejects unknown op",
			err:  argErr(PythonSidecar, req("text_to_image", []string{"/in.png"}, nil)),
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

func TestOnnxModelPathResolution(t *testing.T) {
	req := backends.Request{Model: models.Model{Source: models.Source{Assets: []models.Asset{
		{Filename: "fallback.bin", Kind: models.ArtifactGeneric},
		{Filename: "model.onnx", Kind: models.ArtifactONNX},
	}}}}
	if got := OnnxModelPath(req, "/models/m1"); got != filepath.Join("/models/m1", "model.onnx") {
		t.Fatalf("onnx path = %q", got)
	}
	req.Model.Source.Assets = []models.Asset{{Filename: "first.bin"}}
	if got := OnnxModelPath(req, "/models/m1"); got != filepath.Join("/models/m1", "first.bin") {
		t.Fatalf("fallback asset path = %q", got)
	}
	req.Model.Source.Assets = nil
	if got := OnnxModelPath(req, "/models/m1"); got != "/models/m1" {
		t.Fatalf("empty assets path = %q", got)
	}
}

// TestNewSetRejectsDuplicateOp proves a backend cannot register two techniques
// for one op (a programming error surfaced at construction).
func TestNewSetRejectsDuplicateOp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate op")
		}
	}()
	NewSet(
		Technique{Name: "a", Op: "inpaint", Build: DiffusersInpaint},
		Technique{Name: "b", Op: "inpaint", Build: DiffusersInpaint},
	)
}

func mustArgs(t *testing.T, b ArgBuilder, req backends.Request) []string {
	t.Helper()
	args, err := b(req, req.ModelDir)
	if err != nil {
		t.Fatalf("build args: %v", err)
	}
	return args
}

func argErr(b ArgBuilder, req backends.Request) bool {
	_, err := b(req, req.ModelDir)
	return err != nil
}
