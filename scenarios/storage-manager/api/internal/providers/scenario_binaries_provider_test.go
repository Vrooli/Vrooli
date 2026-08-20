package providers

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

type fakeLiveness struct {
	running map[string]bool
}

func (f *fakeLiveness) IsRunning(_ context.Context, path string) (bool, error) {
	return f.running[path], nil
}

func scenarioBinaryFixture(t *testing.T, running bool) (*ScenarioBinariesProvider, *cleanupfakes.FileSystem, string) {
	t.Helper()
	root := "/fake/.vrooli/bin"
	binary := filepath.Join(root, "rcl-fixture-positive")
	metadata := binary + ".build.meta"
	manifest := binary + ".manifest.json"
	module := "/fake/scenarios/rcl-fixture-positive/cli"
	old := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	files := map[string]cleanup.FileInfo{
		root:     {Path: root, IsDir: true},
		binary:   {Path: binary, Size: 100, ModTime: old},
		metadata: {Path: metadata, Size: 50, ModTime: old},
		manifest: {Path: manifest, Size: 25, ModTime: old},
	}
	contents, _ := json.Marshal(map[string]string{"kind": "scenario", "module_path": module})
	fsys := &cleanupfakes.FileSystem{Root: root, Files: files, Contents: map[string][]byte{metadata: contents}, AllowRemove: true}
	provider := NewScenarioBinariesProvider(fsys, &fakeLiveness{running: map[string]bool{binary: running}}, cleanupfakes.Clock{Time: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)}, ScenarioBinariesProviderConfig{Root: root})
	return provider, fsys, binary
}

func scenarioBinaryPolicy() cleanup.ProviderPolicy {
	return cleanup.ProviderPolicy{Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOwner}
}

func TestScenarioBinariesProviderReportsOrphanedTriple(t *testing.T) {
	provider, _, binary := scenarioBinaryFixture(t, false)
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: scenarioBinaryPolicy()})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(preview.Items) != 1 || preview.Items[0].Path != binary {
		t.Fatalf("preview items = %#v, want one triple rooted at %q", preview.Items, binary)
	}
	if preview.Items[0].Bytes != 175 {
		t.Fatalf("preview bytes = %d, want 175", preview.Items[0].Bytes)
	}
}

func TestScenarioBinariesProviderSkipsRunningBinaryWithWarning(t *testing.T) {
	provider, _, _ := scenarioBinaryFixture(t, true)
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: scenarioBinaryPolicy()})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("running binary was reclaimable: %#v", preview.Items)
	}
	if len(preview.Warnings) != 1 || preview.Warnings[0] == "" {
		t.Fatalf("running binary warning = %#v", preview.Warnings)
	}
}

func TestScenarioBinariesProviderApplyRemovesTriple(t *testing.T) {
	provider, fsys, binary := scenarioBinaryFixture(t, false)
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: scenarioBinaryPolicy()})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: provider.Metadata().Version,
		ApprovalMode:    cleanup.ApprovalModeOwner,
		IdempotencyKey:  "apply-1",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.ReclaimedBytes != 175 || len(fsys.Removed) != 3 {
		t.Fatalf("apply result = %#v removed=%#v, want 175 bytes and 3 artifacts", result, fsys.Removed)
	}
	if fsys.Removed[0] != binary {
		t.Fatalf("first removed path = %q, want binary %q", fsys.Removed[0], binary)
	}
}

func TestScenarioBinariesProviderIgnoresNonScenarioMetadata(t *testing.T) {
	provider, fsys, binary := scenarioBinaryFixture(t, false)
	metadata := binary + ".build.meta"
	contents, _ := json.Marshal(map[string]string{"kind": "resource", "module_path": "/missing/resource/cli"})
	fsys.Contents[metadata] = contents

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: scenarioBinaryPolicy()})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("non-scenario metadata was reported: %#v", preview.Items)
	}
}

func TestScenarioBinariesProviderIgnoresMissingModulePath(t *testing.T) {
	provider, fsys, binary := scenarioBinaryFixture(t, false)
	metadata := binary + ".build.meta"
	contents, _ := json.Marshal(map[string]string{"kind": "scenario"})
	fsys.Contents[metadata] = contents

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: scenarioBinaryPolicy()})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("missing module_path was reported: %#v", preview.Items)
	}
}
