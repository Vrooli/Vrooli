package worktree

import (
	"os"
	"path/filepath"
	"testing"

	worktreev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/worktree"

	"github.com/vrooli/cli-core/cliapp"
)

func TestWorktreeManifestCoversWorktreeService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, worktreev1.File_git_control_tower_v1_worktree_worktree_proto, "WorktreeService")
}
