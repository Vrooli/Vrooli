package technique

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"image-tools/internal/adapters"
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
			name: "diffusers txt2img",
			args: mustArgs(t, DiffusersText2Image, req("text_to_image", nil, map[string]string{
				"prompt": "a castle", "width": "512", "height": "768", "steps": "28", "cfg_scale": "6.0", "negative_prompt": "blurry", "seed": "42",
			})),
			want: []string{"-m", "image_tools_sidecar.text_to_image", "--model", "/models/m1", "--architecture", "sd15", "--prompt", "a castle", "--out", "/out.png", "--width", "512", "--height", "768", "--steps", "28", "--guidance", "6.0", "--negative-prompt", "blurry", "--seed", "42"},
		},
		{
			name: "diffusers txt2img needs a model architecture",
			err: func() bool {
				r := req("text_to_image", nil, map[string]string{"prompt": "x"})
				r.Model.Architecture = ""
				return argErr(DiffusersText2Image, r)
			}(),
		},
		{
			name: "diffusers img2img threads strength",
			args: mustArgs(t, DiffusersImg2Img, req("image_to_image", []string{"/in.png"}, map[string]string{
				"prompt": "make it autumn", "strength": "0.5", "seed": "7",
			})),
			want: []string{"-m", "image_tools_sidecar.image_to_image", "--model", "/models/m1", "--architecture", "sd15", "--image", "/in.png", "--prompt", "make it autumn", "--out", "/out.png", "--strength", "0.5", "--seed", "7"},
		},
		{
			name: "diffusers img2img needs input",
			err:  argErr(DiffusersImg2Img, req("image_to_image", nil, map[string]string{"prompt": "x"})),
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

// TestLoRAArgs covers the Phase 4 LoRA flag assembly: repeatable --lora
// <path>:<scale> from the request's conditioning stack, skipping non-LoRA
// adapters, and failing closed when a LoRA declares no weight file.
func TestLoRAArgs(t *testing.T) {
	dir := t.TempDir()
	weight := filepath.Join(dir, "lcm.safetensors")
	if err := os.WriteFile(weight, []byte("x"), 0o644); err != nil {
		t.Fatalf("write weight: %v", err)
	}

	req := backends.Request{
		Operation: "text_to_image",
		Adapters: []adapters.ResolvedAdapter{
			{ID: "lcm", Kind: adapters.KindLoRA, Scale: 0.8, Dir: dir},
			{ID: "cn", Kind: adapters.KindControlNet, Scale: 1.0, Dir: dir}, // skipped (not lora)
		},
	}
	args, err := LoRAArgs(req)
	if err != nil {
		t.Fatalf("LoRAArgs: %v", err)
	}
	want := []string{"--lora", weight + ":0.8"}
	if !slices.Equal(args, want) {
		t.Fatalf("LoRAArgs = %v, want %v", args, want)
	}

	// Fail closed when a LoRA adapter has no weight file in its dir.
	empty := t.TempDir()
	_, err = LoRAArgs(backends.Request{Adapters: []adapters.ResolvedAdapter{{ID: "x", Kind: adapters.KindLoRA, Dir: empty}}})
	if err == nil {
		t.Fatal("expected error for a LoRA with no weight file")
	}
}

// TestControlNetArgs covers the Phase 5 ControlNet flag assembly: repeatable
// --controlnet <dir>:<scale>:<image>, skipping non-ControlNet adapters, and
// failing closed on a missing dir or conditioning image.
func TestControlNetArgs(t *testing.T) {
	dir := t.TempDir()
	req := backends.Request{
		Operation: "text_to_image",
		Adapters: []adapters.ResolvedAdapter{
			{ID: "cn", Kind: adapters.KindControlNet, Scale: 0.9, Dir: dir, ConditioningImageKey: "/tmp/cond-0.png"},
			{ID: "lcm", Kind: adapters.KindLoRA, Scale: 1.0, Dir: dir}, // skipped (not controlnet)
		},
	}
	args, err := ControlNetArgs(req)
	if err != nil {
		t.Fatalf("ControlNetArgs: %v", err)
	}
	want := []string{"--controlnet", dir + ":0.9:/tmp/cond-0.png"}
	if !slices.Equal(args, want) {
		t.Fatalf("ControlNetArgs = %v, want %v", args, want)
	}
	// Fail closed when a ControlNet has no conditioning image.
	_, err = ControlNetArgs(backends.Request{Adapters: []adapters.ResolvedAdapter{{ID: "cn", Kind: adapters.KindControlNet, Dir: dir}}})
	if err == nil {
		t.Fatal("expected error for a controlnet with no conditioning image")
	}
	// Fail closed when a ControlNet has no installed dir.
	_, err = ControlNetArgs(backends.Request{Adapters: []adapters.ResolvedAdapter{{ID: "cn", Kind: adapters.KindControlNet, ConditioningImageKey: "/x.png"}}})
	if err == nil {
		t.Fatal("expected error for a controlnet with no installed dir")
	}
}

// TestIPAdapterArgs covers the Phase 6 IP-Adapter flag assembly: repeatable
// --ip-adapter <weightfile>:<scale>:<ref>, skipping other kinds, failing closed on
// a missing weight file or reference image.
func TestIPAdapterArgs(t *testing.T) {
	dir := t.TempDir()
	weight := filepath.Join(dir, "ip.safetensors")
	if err := os.WriteFile(weight, []byte("x"), 0o644); err != nil {
		t.Fatalf("write weight: %v", err)
	}
	req := backends.Request{
		Adapters: []adapters.ResolvedAdapter{
			{ID: "ip", Kind: adapters.KindIPAdapter, Scale: 0.6, Dir: dir, ConditioningImageKey: "/tmp/ref.png"},
		},
	}
	args, err := IPAdapterArgs(req)
	if err != nil {
		t.Fatalf("IPAdapterArgs: %v", err)
	}
	want := []string{"--ip-adapter", weight + ":0.6:/tmp/ref.png"}
	if !slices.Equal(args, want) {
		t.Fatalf("IPAdapterArgs = %v, want %v", args, want)
	}
	// Fail closed when the reference image is missing.
	_, err = IPAdapterArgs(backends.Request{Adapters: []adapters.ResolvedAdapter{{ID: "ip", Kind: adapters.KindIPAdapter, Dir: dir}}})
	if err == nil {
		t.Fatal("expected error for an ip-adapter with no reference image")
	}
}

// TestDiffusersBuildersEmitConditioning asserts the txt2img/img2img builders
// thread ControlNet + IP-Adapter flags through alongside LoRA.
func TestDiffusersBuildersEmitConditioning(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ip.safetensors"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write weight: %v", err)
	}
	req := backends.Request{
		Operation: "text_to_image",
		Model:     models.Model{ID: "m", Architecture: "sd15"},
		Params:    map[string]string{"prompt": "p"},
		Output:    storage.OutputTarget{LocalPath: "/out.png"},
		Adapters: []adapters.ResolvedAdapter{
			{ID: "cn", Kind: adapters.KindControlNet, Scale: 1, Dir: dir, ConditioningImageKey: "/cond.png"},
			{ID: "ip", Kind: adapters.KindIPAdapter, Scale: 0.6, Dir: dir, ConditioningImageKey: "/ref.png"},
		},
	}
	args, err := DiffusersText2Image(req, "/models/m")
	if err != nil {
		t.Fatalf("t2i: %v", err)
	}
	if !slices.Contains(args, "--controlnet") || !slices.Contains(args, "--ip-adapter") {
		t.Fatalf("text2image missing conditioning flags: %v", args)
	}
}

// TestStableDiffusionCppRejectsAdapters asserts sd.cpp fails closed on a
// conditioned request (adapters run on the diffusers sidecar).
func TestStableDiffusionCppRejectsAdapters(t *testing.T) {
	req := backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "x"},
		Adapters:  []adapters.ResolvedAdapter{{ID: "lcm", Kind: adapters.KindLoRA}},
	}
	if _, err := StableDiffusionCpp(req, "/models/m"); err == nil {
		t.Fatal("expected sd.cpp to reject conditioning adapters")
	}
}

// TestDiffusersBuildersEmitLoRA asserts the diffusers generate/img2img/inpaint
// builders thread the --lora flag through.
func TestDiffusersBuildersEmitLoRA(t *testing.T) {
	dir := t.TempDir()
	weight := filepath.Join(dir, "l.safetensors")
	if err := os.WriteFile(weight, []byte("x"), 0o644); err != nil {
		t.Fatalf("write weight: %v", err)
	}
	mk := func(op string, in []string) backends.Request {
		return backends.Request{
			Operation: op,
			Model:     models.Model{ID: "m", Architecture: "sd15"},
			InputKeys: in,
			Params:    map[string]string{"prompt": "p"},
			Output:    storage.OutputTarget{LocalPath: "/out.png"},
			Adapters:  []adapters.ResolvedAdapter{{ID: "l", Kind: adapters.KindLoRA, Scale: 1, Dir: dir}},
		}
	}
	t2i, err := DiffusersText2Image(mk("text_to_image", nil), "/models/m")
	if err != nil {
		t.Fatalf("t2i: %v", err)
	}
	if !slices.Contains(t2i, "--lora") {
		t.Fatalf("text2image missing --lora: %v", t2i)
	}
	i2i, err := DiffusersImg2Img(mk("image_to_image", []string{"/in.png"}), "/models/m")
	if err != nil {
		t.Fatalf("i2i: %v", err)
	}
	if !slices.Contains(i2i, "--lora") {
		t.Fatalf("img2img missing --lora: %v", i2i)
	}
	inp, err := DiffusersInpaint(mk("inpaint", []string{"/in.png", "/mask.png"}), "/models/m")
	if err != nil {
		t.Fatalf("inpaint: %v", err)
	}
	if !slices.Contains(inp, "--lora") {
		t.Fatalf("inpaint missing --lora: %v", inp)
	}
}
