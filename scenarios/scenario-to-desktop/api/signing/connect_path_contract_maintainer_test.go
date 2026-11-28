package signing

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

func TestConnectDeleteSigningConfigRejectsTraversalWithoutTouchingSibling(t *testing.T) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(scenariosRoot, 0o755); err != nil {
		t.Fatalf("create scenarios root: %v", err)
	}

	const sentinelContents = "keep this signing config intact"
	siblingDir := filepath.Join(root, "sibling")
	if err := os.MkdirAll(siblingDir, 0o755); err != nil {
		t.Fatalf("create sibling directory: %v", err)
	}
	sentinelPath := filepath.Join(siblingDir, SigningConfigFilename)
	if err := os.WriteFile(sentinelPath, []byte(sentinelContents), 0o600); err != nil {
		t.Fatalf("write sibling sentinel: %v", err)
	}

	repo := NewFileRepository(WithScenarioLocator(&DefaultScenarioLocator{vrooliRoot: root}))
	service := NewConnectService(&Handler{repo: repo})
	_, err := service.DeleteSigningConfig(context.Background(), connect.NewRequest(
		&domainv1.DeleteSigningConfigRequest{ScenarioName: "../sibling"},
	))
	if err == nil {
		t.Fatal("DeleteSigningConfig succeeded for traversal scenario name")
	}

	got, readErr := os.ReadFile(sentinelPath)
	if readErr != nil {
		t.Fatalf("read sibling sentinel after delete: %v", readErr)
	}
	if string(got) != sentinelContents {
		t.Fatalf("sibling signing config changed: got %q, want %q", got, sentinelContents)
	}
}
