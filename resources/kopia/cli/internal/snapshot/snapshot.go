// Package snapshot implements the snapshot lifecycle commands of resource-kopia:
// create, list, restore, verify, delete. Each resolves a repository by --repo
// (sourcing its passphrase + S3 creds from vault) and routes through the single
// kexec seam. Secrets never appear in argv.
package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"resource-kopia/cli/internal/cmdutil"
	"resource-kopia/cli/internal/kexec"
	"resource-kopia/cli/internal/repoctx"
	"sort"
	"strconv"
	"strings"
)

// Service wires the dependencies the snapshot commands need.
type Service struct {
	Runner   kexec.Runner
	Resolver repoctx.Resolver
	Out      io.Writer
}

func (s Service) out() io.Writer {
	if s.Out != nil {
		return s.Out
	}
	return os.Stdout
}

// Create takes a snapshot of a source path into a repository.
func (s Service) Create(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("snapshot create")
	repo := fs.String("repo", "", "Repository name (required)")
	path := fs.String("path", "", "Source path to snapshot (required)")
	jsonOut := fs.Bool("json", false, "Emit kopia's native JSON (includes the snapshot id)")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*path) == "" {
		return fmt.Errorf("--path is required")
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "snapshot", "create", *path)
	if *jsonOut {
		argv = append(argv, "--json")
	}
	return s.run(ctx, target, argv)
}

// List lists snapshots in a repository, optionally filtered by source path.
func (s Service) List(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("snapshot list")
	repo := fs.String("repo", "", "Repository name (required)")
	path := fs.String("path", "", "Optional source path filter")
	jsonOut := fs.Bool("json", false, "Emit kopia's native JSON")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "snapshot", "list")
	if strings.TrimSpace(*path) != "" {
		argv = append(argv, *path)
	}
	if *jsonOut {
		argv = append(argv, "--json")
	}
	return s.run(ctx, target, argv)
}

// Restore restores a snapshot to a target directory.
func (s Service) Restore(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("snapshot restore")
	repo := fs.String("repo", "", "Repository name (required)")
	snap := fs.String("snapshot", "", "Snapshot id to restore (required)")
	dest := fs.String("target", "", "Target directory to restore into (required)")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*snap) == "" {
		return fmt.Errorf("--snapshot is required")
	}
	if strings.TrimSpace(*dest) == "" {
		return fmt.Errorf("--target is required")
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "snapshot", "restore", *snap, *dest)
	return s.run(ctx, target, argv)
}

// Browse restores a snapshot into a temporary directory and lists one
// directory level as JSON. Kopia does not expose a stable native browse command,
// so this wrapper keeps the resource surface honest while preserving DBM's lazy
// snapshot browser contract.
func (s Service) Browse(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("snapshot browse")
	repo := fs.String("repo", "", "Repository name (required)")
	snap := fs.String("snapshot", "", "Snapshot id to browse (required)")
	path := fs.String("path", "", "Optional snapshot-relative directory path")
	jsonOut := fs.Bool("json", false, "Emit JSON entries")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*snap) == "" {
		return fmt.Errorf("--snapshot is required")
	}
	if !*jsonOut {
		return fmt.Errorf("--json is required")
	}
	relPath, err := cleanBrowsePath(*path)
	if err != nil {
		return err
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "resource-kopia-browse-")
	if err != nil {
		return fmt.Errorf("create browse temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	argv := append(target.GlobalArgs(), "snapshot", "restore", *snap, tmp)
	if err := s.runSilent(ctx, target, argv); err != nil {
		return err
	}
	entries, err := listEntries(tmp, relPath)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(s.out())
	return enc.Encode(entries)
}

// Verify checks snapshot/content integrity. With --snapshot it verifies a
// single snapshot; otherwise it verifies the repository.
func (s Service) Verify(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("snapshot verify")
	repo := fs.String("repo", "", "Repository name (required)")
	snap := fs.String("snapshot", "", "Optional snapshot id to verify")
	percent := fs.Float64("verify-files-percent", 0, "Percentage of files to fully read and verify")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "snapshot", "verify")
	if *percent > 0 {
		argv = append(argv, "--verify-files-percent", strconv.FormatFloat(*percent, 'f', -1, 64))
	}
	if strings.TrimSpace(*snap) != "" {
		argv = append(argv, *snap)
	}
	return s.run(ctx, target, argv)
}

// Delete deletes a snapshot. kopia requires explicit confirmation, so --delete
// is always passed.
func (s Service) Delete(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("snapshot delete")
	repo := fs.String("repo", "", "Repository name (required)")
	snap := fs.String("snapshot", "", "Snapshot id to delete (required)")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*snap) == "" {
		return fmt.Errorf("--snapshot is required")
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "snapshot", "delete", *snap, "--delete")
	return s.run(ctx, target, argv)
}

func (s Service) resolve(ctx context.Context, repo string) (repoctx.Target, error) {
	if strings.TrimSpace(repo) == "" {
		return repoctx.Target{}, fmt.Errorf("--repo is required")
	}
	return s.Resolver.Resolve(ctx, repo)
}

func (s Service) run(ctx context.Context, target repoctx.Target, argv []string) error {
	out, err := s.Runner.Run(ctx, kexec.Call{Args: argv, Env: target.Env})
	if err != nil {
		return err
	}
	_, err = s.out().Write(cmdutil.EnsureTrailingNewline(out))
	return err
}

func (s Service) runSilent(ctx context.Context, target repoctx.Target, argv []string) error {
	_, err := s.Runner.Run(ctx, kexec.Call{Args: argv, Env: target.Env})
	return err
}

type browseEntry struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"sizeBytes"`
	Type      string `json:"type"`
	IsDir     bool   `json:"isDir"`
}

func cleanBrowsePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "." || path == "/" {
		return "", nil
	}
	path = strings.TrimPrefix(path, "/")
	clean := filepath.Clean(path)
	if clean == "." {
		return "", nil
	}
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("--path must be snapshot-relative")
	}
	return filepath.ToSlash(clean), nil
}

func listEntries(root, relPath string) ([]browseEntry, error) {
	dir := root
	if relPath != "" {
		dir = filepath.Join(root, filepath.FromSlash(relPath))
	}
	children, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("browse path %q not found", relPath)
		}
		return nil, fmt.Errorf("browse path %q: %w", relPath, err)
	}
	out := make([]browseEntry, 0, len(children))
	for _, child := range children {
		info, err := child.Info()
		if err != nil {
			return nil, fmt.Errorf("browse entry %q: %w", child.Name(), err)
		}
		entryRel := child.Name()
		if relPath != "" {
			entryRel = filepath.ToSlash(filepath.Join(relPath, child.Name()))
		}
		typ := "f"
		if child.IsDir() {
			typ = "d"
		}
		size := info.Size()
		if child.IsDir() || info.Mode().Type()&fs.ModeSymlink != 0 {
			size = 0
		}
		out = append(out, browseEntry{
			Path:      entryRel,
			Name:      entryRel,
			SizeBytes: size,
			Type:      typ,
			IsDir:     child.IsDir(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}
