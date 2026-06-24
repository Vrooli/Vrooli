package models

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// HFSnapshotFetcher is the production RepoFetcher: it fetches a HuggingFace repo
// snapshot at a pinned revision via huggingface_hub.snapshot_download (the
// maintained, resumable, LFS-aware fetcher — we do NOT reimplement repo/LFS
// fetch in Go). huggingface_hub is a governed scenario dependency (see
// docs/package-governance.md); Python is the same interpreter the sidecar uses.
type HFSnapshotFetcher struct {
	// Python is the interpreter to invoke (defaults to "python3").
	Python string
	// Env, when set, replaces the child environment (else os.Environ is used).
	// Lets callers pin HF_HOME / HF_HUB_OFFLINE in tests or sandboxed runs.
	Env []string
}

// snapshotScript downloads repo@revision into dest using snapshot_download. It is
// kept tiny and argument-free (all inputs via argv) so there is no shell-quoting
// surface. local_dir gives a real file tree (not a symlinked cache) under dest.
const snapshotScript = `import sys
from huggingface_hub import snapshot_download
repo, revision, dest = sys.argv[1], sys.argv[2], sys.argv[3]
allow = sys.argv[4:] or None
snapshot_download(repo_id=repo, revision=revision, local_dir=dest, allow_patterns=allow)
`

// Snapshot fetches repo into destDir. Progress is reported coarsely (huggingface
// streams its own progress to stderr); emit is called once at completion so the
// job layer advances. A non-zero exit surfaces the captured stderr.
func (f *HFSnapshotFetcher) Snapshot(ctx context.Context, repo RepoSource, destDir string, emit func(done, total int64)) error {
	if strings.TrimSpace(repo.RepoID) == "" {
		return fmt.Errorf("hf snapshot: empty repo_id")
	}
	if strings.TrimSpace(repo.Revision) == "" {
		return fmt.Errorf("hf snapshot: empty revision (must pin a commit SHA)")
	}
	python := f.Python
	if python == "" {
		python = "python3"
	}
	args := append([]string{"-c", snapshotScript, repo.RepoID, repo.Revision, destDir}, repo.AllowPatterns...)
	cmd := exec.CommandContext(ctx, python, args...)
	if f.Env != nil {
		cmd.Env = f.Env
	} else {
		cmd.Env = os.Environ()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hf snapshot_download %s@%s: %w: %s", repo.RepoID, shortRev(repo.Revision), err, strings.TrimSpace(string(out)))
	}
	if emit != nil {
		emit(1, 1)
	}
	return nil
}
