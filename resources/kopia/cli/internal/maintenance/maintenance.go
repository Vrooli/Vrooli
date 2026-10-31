// Package maintenance implements the maintenance commands of resource-kopia:
// run, set, and info. Maintenance prunes unreferenced content and applies retention,
// enforcing the policies set via the policy command group.
package maintenance

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/cmdutil"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repoctx"
)

// Service wires the dependencies the maintenance commands need.
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

// Run executes a maintenance cycle (quick by default, full with --full).
func (s Service) Run(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("maintenance run")
	repo := fs.String("repo", "", "Repository name (required)")
	full := fs.Bool("full", false, "Run full maintenance (prunes per retention)")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "maintenance", "run")
	if *full {
		argv = append(argv, "--full")
	}
	return s.run(ctx, target, argv)
}

// Set configures the automatic maintenance schedule for a repository.
func (s Service) Set(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("maintenance set")
	repo := fs.String("repo", "", "Repository name (required)")
	enableFull := fs.String("enable-full", "", "Enable full maintenance: true|false")
	enableQuick := fs.String("enable-quick", "", "Enable quick maintenance: true|false")
	fullInterval := fs.String("full-interval", "", "Full maintenance interval (e.g. 24h)")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "maintenance", "set")
	if strings.TrimSpace(*enableFull) != "" {
		argv = append(argv, "--enable-full", *enableFull)
	}
	if strings.TrimSpace(*enableQuick) != "" {
		argv = append(argv, "--enable-quick", *enableQuick)
	}
	if strings.TrimSpace(*fullInterval) != "" {
		argv = append(argv, "--full-interval", *fullInterval)
	}
	return s.run(ctx, target, argv)
}

// Info prints the repository's maintenance parameters: the owner (user@host
// allowed to run it), whether quick/full cycles are enabled, and when they last
// ran. Only the owner may run maintenance without --force, so a repo whose
// owner is another identity is never maintained automatically from here.
func (s Service) Info(ctx context.Context, args []string) error {
	fs := cmdutil.NewFlagSet("maintenance info")
	repo := fs.String("repo", "", "Repository name (required)")
	jsonOut := fs.Bool("json", false, "Emit kopia's native JSON")
	if err := cmdutil.Parse(fs, args); err != nil {
		return err
	}
	target, err := s.resolve(ctx, *repo)
	if err != nil {
		return err
	}
	argv := append(target.GlobalArgs(), "maintenance", "info")
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
	if err := target.RetireLegacyPassphrase(); err != nil {
		return err
	}
	_, err = s.out().Write(cmdutil.EnsureTrailingNewline(out))
	return err
}
