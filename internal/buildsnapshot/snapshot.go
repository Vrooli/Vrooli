// Package buildsnapshot captures the compiler inputs of a worktree into an
// immutable directory and builds from that directory.
//
// Builds used to compile from the live checkout and only notice afterwards
// that the checkout had moved (`withStableBuildInputs`). Under concurrent
// agents that is a scan-then-compile race: the artifact could contain code from
// one moment and a descriptor, fingerprint or provenance record from another.
// Compiling from a snapshot removes the race instead of detecting it, and the
// snapshot is captured from exactly the fingerprint's input set, so the
// artifact and its fingerprint always describe the same bytes.
//
// The snapshot is a plain directory copy of the input files, taken with their
// bytes as they are on disk. It deliberately does not use a git checkout:
// checkout and `git add` apply `.gitattributes` filters, which can silently
// change the bytes a fingerprint hashes (a PDF or a CRLF source file would no
// longer match the file the live build compiles). It also keeps the snapshot
// independent of any index, worktree or temporary commit, so nothing outside
// the snapshot directory can be left behind.
package buildsnapshot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

// Options describes the snapshot to capture.
type Options struct {
	// Root is the checkout the build was asked for.
	Root string
	// Targets are the relative paths that participate, matching the build
	// cohort. They resolve to the exact compiler inputs the fingerprint uses.
	Targets []string
	// ParentDir is where the snapshot directory is created. Empty means the
	// operating system's temporary directory.
	ParentDir string
}

// Snapshot is an immutable copy of one moment's compiler inputs.
type Snapshot struct {
	// Dir is the directory to compile in.
	Dir string
	// Files is the number of captured compiler inputs.
	Files int
}

// Create captures the compiler inputs of opts.Root into a new directory.
//
// It only reads the checkout: the snapshot is a copy, so concurrent edits,
// staging, commits or rebases cannot change what the build sees. Callers must
// call Remove when the build is done.
func Create(ctx context.Context, opts Options) (*Snapshot, error) {
	root := strings.TrimSpace(opts.Root)
	if root == "" {
		return nil, errors.New("build snapshot root is required")
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve build snapshot root: %w", err)
	}
	paths, err := buildinfo.CompilerInputPaths(absoluteRoot, opts.Targets...)
	if err != nil {
		return nil, fmt.Errorf("enumerate compiler inputs: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no compiler inputs under %s", strings.Join(opts.Targets, ", "))
	}
	dir, err := os.MkdirTemp(opts.ParentDir, "vrooli-build-*")
	if err != nil {
		return nil, fmt.Errorf("create the build snapshot directory: %w", err)
	}
	snapshot := &Snapshot{Dir: dir}
	for _, relative := range paths {
		if err := ctx.Err(); err != nil {
			_ = snapshot.Remove(context.WithoutCancel(ctx))
			return nil, err
		}
		if err := copyInput(absoluteRoot, dir, relative); err != nil {
			_ = snapshot.Remove(context.WithoutCancel(ctx))
			return nil, err
		}
		snapshot.Files++
	}
	return snapshot, nil
}

// Remove discards the snapshot. It is safe to call twice.
func (s *Snapshot) Remove(ctx context.Context) error {
	if s == nil || s.Dir == "" {
		return nil
	}
	return os.RemoveAll(s.Dir)
}

func copyInput(root, snapshotDir, relative string) error {
	if !filepath.IsLocal(relative) {
		return fmt.Errorf("compiler input %q escapes the build root", relative)
	}
	source := filepath.Join(root, relative)
	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("inspect compiler input %s: %w", relative, err)
	}
	if info.IsDir() {
		return fmt.Errorf("compiler input %s is a directory", relative)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		// The build reads the link target, so capture that content. A copy
		// cannot dangle if the link changes afterwards.
		resolved, err := os.Stat(source)
		if err != nil {
			return fmt.Errorf("resolve compiler input %s: %w", relative, err)
		}
		if resolved.IsDir() {
			return fmt.Errorf("compiler input %s is a directory link", relative)
		}
	}
	target := filepath.Join(snapshotDir, relative)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("prepare %s in the build snapshot: %w", relative, err)
	}
	mode := os.FileMode(0o644)
	if info.Mode().Perm()&0o111 != 0 {
		mode = 0o755
	}
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("read compiler input %s: %w", relative, err)
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("write %s into the build snapshot: %w", relative, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy compiler input %s: %w", relative, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close the snapshot copy of %s: %w", relative, err)
	}
	return nil
}
