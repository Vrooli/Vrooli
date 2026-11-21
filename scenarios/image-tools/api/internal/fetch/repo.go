package fetch

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	// FileSHA256 pins selected high-value files independently of the computed
	// installed-tree receipt. Paths are relative to the repository root.
	FileSHA256 map[string]string `json:"file_sha256,omitempty"`
}

// RepoFetcher fetches a whole multi-file model repository (a HuggingFace
// snapshot) at a pinned revision into destDir. It is a seam (like Downloader) so
// a catalog installer is testable without network/python; the production
// implementation (HFSnapshotFetcher) shells huggingface_hub.snapshot_download.
// emit reports byte progress (total may be -1 if unknown).
type RepoFetcher interface {
	Snapshot(ctx context.Context, repo RepoSpec, destDir string, emit func(done, total int64)) error
}

type ObservedRepoFetcher interface {
	SnapshotObserved(ctx context.Context, repo RepoSpec, destDir string, observe func(done, total int64) error) error
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
	return f.SnapshotObserved(ctx, repo, destDir, func(done, total int64) error {
		if emit != nil {
			emit(done, total)
		}
		return nil
	})
}

func (f *HFSnapshotFetcher) SnapshotObserved(ctx context.Context, repo RepoSpec, destDir string, observe func(done, total int64) error) error {
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
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(workCtx, python, args...)
	if f.Env != nil {
		cmd.Env = f.Env
	} else {
		cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, nil)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	watchErr := make(chan error, 1)
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-workCtx.Done():
				watchErr <- nil
				return
			case <-ticker.C:
				size, sizeErr := directoryBytes(destDir)
				if sizeErr != nil {
					watchErr <- sizeErr
					cancel()
					return
				}
				if observe != nil {
					if err := observe(size, -1); err != nil {
						watchErr <- err
						cancel()
						return
					}
				}
			}
		}
	}()
	output := &boundedCommandOutput{limit: 1 << 20}
	cmd.Stdout, cmd.Stderr = output, output
	waitErr := cmd.Wait()
	cancel()
	<-watchDone
	observedErr := <-watchErr
	if observedErr != nil {
		return fmt.Errorf("disk progress observer: %w", observedErr)
	}
	if waitErr != nil {
		return fmt.Errorf("hf snapshot_download %s@%s: %w: %s", repo.RepoID, ShortRev(repo.Revision), waitErr, strings.TrimSpace(output.String()))
	}
	if observe != nil {
		if err := observe(1, 1); err != nil {
			return err
		}
	}
	return nil
}

type boundedCommandOutput struct {
	mu     sync.Mutex
	buffer bytes.Buffer
	limit  int
}

func (w *boundedCommandOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	remaining := w.limit - w.buffer.Len()
	if remaining > len(p) {
		remaining = len(p)
	}
	if remaining > 0 {
		_, _ = w.buffer.Write(p[:remaining])
	}
	return len(p), nil
}

func (w *boundedCommandOutput) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

func directoryBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			info, err = os.Stat(path) // HF local_dir can point at cache blobs with file symlinks.
			if err != nil {
				return err
			}
		}
		if info.Mode().IsRegular() {
			if total > math.MaxInt64-info.Size() {
				return fmt.Errorf("snapshot size overflow")
			}
			total += info.Size()
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return total, err
}

// ShortRev truncates a commit SHA for human-readable progress messages.
func ShortRev(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}
