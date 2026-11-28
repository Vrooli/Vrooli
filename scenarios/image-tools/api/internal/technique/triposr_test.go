package technique

import (
	"strings"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/storage"
)

func TestTripoSRRequiresEngineResolvedSourceProfile(t *testing.T) {
	_, err := TripoSRSidecar(backends.Request{Operation: "image_to_object", Model: models.Model{ID: "triposr"}, InputKeys: []string{"/in.png"}}, "/weights")
	if err == nil || !strings.Contains(err.Error(), "engine-resolved managed TripoSR source profile") {
		t.Fatalf("expected source profile refusal, got %v", err)
	}
}

func TestTripoSRPassesOnlyResolvedSourceProfile(t *testing.T) {
	req := backends.Request{Operation: "image_to_object", Model: models.Model{ID: "triposr"}, ModelDir: "/weights", SourceProfileDir: "/managed/triposr", InputKeys: []string{"/in.png"}, Output: storage.OutputTarget{LocalPath: "/out.zip"}}
	args, err := TripoSRSidecar(req, "/weights")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--source-dir /managed/triposr") {
		t.Fatalf("resolved source profile missing: %v", args)
	}
}
