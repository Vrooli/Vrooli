package skills

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
	"prompt-manager/internal/projection"
)

func TestProjectionConnectPreviewApplyAndConflict(t *testing.T) {
	source, target := t.TempDir(), filepath.Join(t.TempDir(), "native")
	if err := os.Mkdir(filepath.Join(source, "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "alpha", "SKILL.md"), []byte("---\nname: alpha\ndescription: test\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	projector := &projection.Service{SourceRoot: source, Targets: []projection.Target{{Runtime: "fixture", Path: target}}, LoadPack: func() (projection.BasePack, error) {
		return projection.BasePack{Skills: []string{"alpha"}, MaxSkills: 1, MaxTokens: 100}, nil
	}}
	_, handler := NewConnectMountWithProjection(nil, nil, nil, projector)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := skillsconnect.NewSkillsServiceClient(server.Client(), server.URL)
	req := &skillsv1.RefreshProjectionRequest{Runtime: "fixture", Skills: []string{"alpha"}}
	preview, err := client.RefreshProjection(context.Background(), connect.NewRequest(req))
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Msg.Rows) != 1 || preview.Msg.Rows[0].Status != "missing" {
		t.Fatalf("preview: %+v", preview.Msg)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("preview wrote target")
	}
	req.Apply, req.ExpectedDigest = true, preview.Msg.Digest
	result, err := client.RefreshProjection(context.Background(), connect.NewRequest(req))
	if err != nil || !result.Msg.Rows[0].Applied {
		t.Fatalf("apply: %v %v", result, err)
	}
	// Reusing a review after state changes must not silently write.
	if _, err := client.RefreshProjection(context.Background(), connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale preview: %v", err)
	}
}
