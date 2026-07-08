package cleanupfakes

import (
	"context"
	"path/filepath"
	"testing"

	"cleanup-manager/internal/cleanup"
)

func TestFileSystemBlocksRemovalOutsideFakeRoot(t *testing.T) {
	t.Parallel()

	fsys := &FileSystem{
		Root:        t.TempDir(),
		Files:       map[string]cleanup.FileInfo{},
		AllowRemove: true,
	}

	if err := fsys.RemoveAll(context.Background(), "/tmp/not-owned"); err == nil {
		t.Fatal("RemoveAll() expected outside-root error")
	}
}

func TestFileSystemRecordsRemovalInsideFakeRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "cache")
	fsys := &FileSystem{
		Root: root,
		Files: map[string]cleanup.FileInfo{
			target: {Path: target, IsDir: true},
		},
		AllowRemove: true,
	}

	if err := fsys.RemoveAll(context.Background(), target); err != nil {
		t.Fatalf("RemoveAll() unexpected error: %v", err)
	}
	if len(fsys.Removed) != 1 || fsys.Removed[0] != target {
		t.Fatalf("removed paths = %#v, want %q", fsys.Removed, target)
	}
}

func TestProcessRunnerBlocksForbiddenCleanupCommands(t *testing.T) {
	t.Parallel()

	runner := &ProcessRunner{Forbidden: []string{"docker system prune", "journalctl --vacuum"}}
	_, err := runner.Run(context.Background(), cleanup.ProcessCommand{Name: "docker", Args: []string{"system", "prune"}})
	if err == nil {
		t.Fatal("Run() expected forbidden command error")
	}
}

func TestDockerClientBlocksVolumePrune(t *testing.T) {
	t.Parallel()

	client := &DockerClient{}
	_, err := client.Prune(context.Background(), cleanup.DockerPruneRequest{Volumes: true})
	if err == nil {
		t.Fatal("Prune() expected volume prune error")
	}
}
