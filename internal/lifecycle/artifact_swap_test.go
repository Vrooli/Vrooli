package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageArtifactDirectorySwapsWithoutMutatingCurrentTree(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "dist")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "index.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, staged, cleanup, err := stageArtifact(target, true)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := os.WriteFile(filepath.Join(staged, "index.html"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := swapArtifact(staged, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(target, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("target = %q, want new", got)
	}
}

func TestStageArtifactReapsOnlyAbandonedLifecycleStages(t *testing.T) {
	root := t.TempDir()
	stale := filepath.Join(root, ".vrooli-artifact-stage-abandoned")
	ordinary := filepath.Join(root, "ordinary-directory")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ordinary, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, cleanup, err := stageArtifact(filepath.Join(root, "dist"), true); err != nil {
		t.Fatal(err)
	} else {
		cleanup()
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("abandoned stage still exists: %v", err)
	}
	if _, err := os.Stat(ordinary); err != nil {
		t.Fatalf("ordinary sibling was changed: %v", err)
	}
}

func TestStageArtifactFileLeavesCurrentArtifactWhenBuildDoesNotPublish(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "app")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, staged, cleanup, err := stageArtifact(target, false)
	if err != nil {
		t.Fatal(err)
	}
	cleanup()
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatalf("staged path still exists after cleanup: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("target changed before publish: %q", got)
	}
}

func TestComponentPublishTargetUsesArtifactDirectoryForDirectoryBuilders(t *testing.T) {
	if got, want := componentPublishTarget("ui/dist/index.html", true), "ui/dist"; got != want {
		t.Fatalf("directory publish target = %q, want %q", got, want)
	}
	if got, want := componentPublishTarget("api/demo-api", false), "api/demo-api"; got != want {
		t.Fatalf("file publish target = %q, want %q", got, want)
	}
}
