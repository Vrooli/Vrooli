package fetch

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"strings"

	"github.com/vrooli/envkit-go"
)

// RepoSpec is a HuggingFace-style multi-file model repository — the install shape
// diffusers models (and many adapters) use (a directory of model_index.json +
// sharded safetensors across subdirs), which cannot be expressed as a handful of
// discrete Asset URLs without drifting every upstream revision. The installer
// fetches the whole snapshot, pinned to an immutable Revision, and integrity is a
// tree-manifest hash over the fetched files. internal/models re-exports this as
// RepoSource (its catalog vocabulary) so the seed schema is unchanged.
type RepoSpec struct {
	// RepoID is the HuggingFace repo (e.g. "Qwen/Qwen-Image-Edit-2509").
	RepoID string `json:"repo_id"`
	// Revision pins an IMMUTABLE commit SHA (never a moving branch/tag) so the
	// fetched tree — and thus the tree-manifest checksum — is reproducible.
	Revision string `json:"revision"`
	// AllowPatterns optionally restricts which files are fetched (e.g. skip the
	// original/ fp32 mirror). Empty fetches the whole repo.
	AllowPatterns []string `json:"allow_patterns,omitempty"`
}

// RepoFetcher fetches a whole multi-file model repository (a HuggingFace
// snapshot) at a pinned revision into destDir. It is a seam (like Downloader) so
// a catalog installer is testable without network/python; the production
// implementation (HFSnapshotFetcher) shells huggingface_hub.snapshot_download.
// emit reports byte progress (total may be -1 if unknown).
type RepoFetcher interface {
	Snapshot(ctx context.Context, repo RepoSpec, destDir string, emit func(done, total int64)) error
}

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
func (f *HFSnapshotFetcher) Snapshot(ctx context.Context, repo RepoSpec, destDir string, emit func(done, total int64)) error {
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
		cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hf snapshot_download %s@%s: %w: %s", repo.RepoID, ShortRev(repo.Revision), err, strings.TrimSpace(string(out)))
	}
	if emit != nil {
		emit(1, 1)
	}
	return nil
}

// ShortRev truncates a commit SHA for human-readable progress messages.
func ShortRev(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}
