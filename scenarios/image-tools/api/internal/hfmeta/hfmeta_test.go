package hfmeta

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLayout(t *testing.T) {
	cases := []struct {
		name  string
		files []FileInfo
		want  Layout
	}{
		{"diffusers repo", []FileInfo{{Path: "model_index.json"}, {Path: "vae/config.json"}, {Path: "transformer/diffusion_pytorch_model.safetensors"}}, LayoutDiffusersRepo},
		{"single file", []FileInfo{{Path: "spicyRealism_v10.safetensors"}, {Path: "README.md"}}, LayoutSingleFile},
		{"single ckpt", []FileInfo{{Path: "model.ckpt"}}, LayoutSingleFile},
		{"shard without index is not single-file", []FileInfo{{Path: "transformer/diffusion_pytorch_model.safetensors"}}, LayoutUnknown},
		{"empty", nil, LayoutUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectLayout(tc.files); got != tc.want {
				t.Fatalf("DetectLayout = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestHFClientRecordedFixture replays captured HuggingFace JSON through the
// Runner seam so the parse + layout + size mapping is exercised offline.
func TestHFClientRecordedFixture(t *testing.T) {
	const fixture = `{
		"repo_id": "stabilityai/stable-diffusion-xl-base-1.0",
		"revision": "462165984030d82259a11f4367a4eed129e94a7b",
		"pipeline_class": "StableDiffusionXLPipeline",
		"tags": ["text-to-image", "stable-diffusion-xl", "diffusers"],
		"base_model": "",
		"license": "openrail++",
		"nsfw": false,
		"files": [
			{"path": "model_index.json", "size": 600},
			{"path": "unet/diffusion_pytorch_model.safetensors", "size": 5000000000},
			{"path": "vae/diffusion_pytorch_model.safetensors", "size": 167000000}
		]
	}`
	c := &HFClient{Runner: func(_ context.Context, repoID string) ([]byte, error) {
		if repoID != "stabilityai/stable-diffusion-xl-base-1.0" {
			t.Fatalf("unexpected repo id %q", repoID)
		}
		return []byte(fixture), nil
	}}
	m, err := c.Inspect(context.Background(), "stabilityai/stable-diffusion-xl-base-1.0")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if m.Layout != LayoutDiffusersRepo {
		t.Errorf("layout = %q, want diffusers-repo", m.Layout)
	}
	if m.PipelineClass != "StableDiffusionXLPipeline" {
		t.Errorf("pipeline class = %q", m.PipelineClass)
	}
	if m.License != "openrail++" || m.NSFW {
		t.Errorf("license/nsfw = %q/%v", m.License, m.NSFW)
	}
	if m.Revision == "" {
		t.Error("expected a pinned revision")
	}
	if want := int64(600 + 5000000000 + 167000000); m.SizeBytes != want {
		t.Errorf("size = %d, want %d (sum of files)", m.SizeBytes, want)
	}
}

func TestHFClientInspectsLocalDir(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "model_index.json"), `{"_class_name":"StableDiffusionPipeline"}`)
	mustWrite(t, filepath.Join(root, "unet", "diffusion_pytorch_model.safetensors"), "shard")
	m, err := (&HFClient{}).Inspect(context.Background(), root)
	if err != nil {
		t.Fatalf("inspect local: %v", err)
	}
	if m.Layout != LayoutDiffusersRepo {
		t.Errorf("layout = %q, want diffusers-repo", m.Layout)
	}
	if m.PipelineClass != "StableDiffusionPipeline" {
		t.Errorf("local pipeline class = %q", m.PipelineClass)
	}
}

func TestHFClientInspectsURLAsSingleFile(t *testing.T) {
	m, err := (&HFClient{}).Inspect(context.Background(), "https://example.test/weights/spicy.safetensors?download=true")
	if err != nil {
		t.Fatalf("inspect url: %v", err)
	}
	if m.Layout != LayoutSingleFile {
		t.Errorf("layout = %q, want single-file", m.Layout)
	}
	if len(m.Files) != 1 || m.Files[0].Path != "spicy.safetensors" {
		t.Errorf("expected single file spicy.safetensors, got %+v", m.Files)
	}
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
