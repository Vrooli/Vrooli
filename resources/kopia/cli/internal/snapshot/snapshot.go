// Package snapshot implements the snapshot lifecycle commands of resource-kopia:
// create, list, restore, verify, delete. Each resolves a repository by --repo
// (sourcing its passphrase + S3 creds from vault) and routes through the single
// kexec seam. Secrets never appear in argv.
package snapshot

import (
	"context"
	"fmt"
	"io"
	"os"
	"resource-kopia/cli/internal/cmdutil"
	"resource-kopia/cli/internal/kexec"
	"resource-kopia/cli/internal/repoctx"
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
