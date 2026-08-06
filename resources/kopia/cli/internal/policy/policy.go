// Package policy implements the policy commands of resource-kopia: set, show,
// list. Policies attach GFS retention + compression to a (host, user, path)
// source within a repository. The wrapper translates Vrooli flags to kopia
// policy flags and passes through kopia's native policy JSON on show/list.
package policy

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"resource-kopia/cli/internal/cmdutil"
	"resource-kopia/cli/internal/kexec"
	"resource-kopia/cli/internal/repoctx"
)

// Service wires the dependencies the policy commands need.
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

// keepFlag pairs a Vrooli/kopia retention flag with its parsed value. A
// negative value means the operator did not set it, so it is not emitted.
type keepFlag struct {
	name  string
	value *int
}

// Set applies GFS retention + compression to a source path's policy.
func (s Service) Set(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("policy set")
	repo := fs.String("repo", "", "Repository name (required)")
	path := fs.String("path", "", "Source path the policy applies to (required)")
	keepLatest := fs.Int("keep-latest", -1, "Number of most recent snapshots to keep")
	keepHourly := fs.Int("keep-hourly", -1, "Hourly snapshots to keep")
	keepDaily := fs.Int("keep-daily", -1, "Daily snapshots to keep")
	keepWeekly := fs.Int("keep-weekly", -1, "Weekly snapshots to keep")
	keepMonthly := fs.Int("keep-monthly", -1, "Monthly snapshots to keep")
	keepAnnual := fs.Int("keep-annual", -1, "Annual snapshots to keep")
	compression := fs.String("compression", "", "Compression algorithm (e.g. zstd)")
	frequency := fs.String("snapshot-interval", "", "Snapshot frequency hint (kopia interval, e.g. 1h)")
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

	argv := append(target.GlobalArgs(), "policy", "set", *path)
	keeps := []keepFlag{
		{"--keep-latest", keepLatest},
		{"--keep-hourly", keepHourly},
		{"--keep-daily", keepDaily},
		{"--keep-weekly", keepWeekly},
		{"--keep-monthly", keepMonthly},
		{"--keep-annual", keepAnnual},
	}
	for _, k := range keeps {
		if *k.value >= 0 {
			argv = append(argv, k.name, strconv.Itoa(*k.value))
		}
	}
	if strings.TrimSpace(*compression) != "" {
		argv = append(argv, "--compression", *compression)
	}
	if strings.TrimSpace(*frequency) != "" {
		argv = append(argv, "--snapshot-interval", *frequency)
	}
	return s.run(ctx, target, argv)
}

// Show prints the resolved policy for a source path.
func (s Service) Show(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("policy show")
	repo := fs.String("repo", "", "Repository name (required)")
	path := fs.String("path", "", "Source path (omit for --global)")
	global := fs.Bool("global", false, "Show the global policy")
	jsonOut := fs.Bool("json", false, "Emit kopia's native JSON")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "policy", "show")
	if *global {
		argv = append(argv, "--global")
	} else if strings.TrimSpace(*path) != "" {
		argv = append(argv, *path)
	}
	if *jsonOut {
		argv = append(argv, "--json")
	}
	return s.run(ctx, target, argv)
}

// List prints all policies in a repository.
func (s Service) List(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("policy list")
	repo := fs.String("repo", "", "Repository name (required)")
	jsonOut := fs.Bool("json", false, "Emit kopia's native JSON")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "policy", "list")
	if *jsonOut {
		argv = append(argv, "--json")
	}
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
