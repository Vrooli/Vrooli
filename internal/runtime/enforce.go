package runtime

import (
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// Options describes a single host-requirement enforcement call.
type Options struct {
	Root          string
	Home          string
	Environment   string
	When          string
	Resources     string
	Scenarios     string
	ScenarioPaths []string
	Platform      string
	SudoMode      string
	DryRun        bool
	AutoInstall   bool
	Stdout        io.Writer
	Stderr        io.Writer
	// Label is a short context string (e.g. "scenario:foo", "resource:bar")
	// used to scope error messages so callers know which invocation failed.
	Label string
}

// enforceDeps allows the enforcement policy to be tested without invoking
// host probes or installers. Production callers use Enforce.
type enforceDeps struct {
	resolve func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error)
	ensure  func(opts EnsureOptions, resolution hostreq.Resolution) (Report, error)
}

func Enforce(opts Options) (Report, error) {
	return enforceWithDeps(enforceDeps{
		resolve: hostreq.Resolve,
		ensure:  EnsureRequirements,
	}, opts)
}

func enforceWithDeps(deps enforceDeps, opts Options) (Report, error) {
	if deps.resolve == nil {
		deps.resolve = hostreq.Resolve
	}
	if deps.ensure == nil {
		deps.ensure = EnsureRequirements
	}

	platform := opts.Platform
	if platform == "" {
		platform = hostreqspec.CurrentPlatform()
	}

	resolution, err := deps.resolve(opts.Root, opts.Home, hostreq.ResolveOptions{
		Environment:   opts.Environment,
		When:          opts.When,
		Resources:     opts.Resources,
		Scenarios:     opts.Scenarios,
		ScenarioPaths: opts.ScenarioPaths,
		Platform:      platform,
	})
	if err != nil {
		return Report{}, fmt.Errorf("resolve host requirements (%s): %w", describeEnforcementLabel(opts.Label), err)
	}
	if len(resolution.Tools) == 0 && len(resolution.Safeguards) == 0 {
		return Report{}, nil
	}

	report, ensureErr := deps.ensure(EnsureOptions{
		Environment: opts.Environment,
		SudoMode:    opts.SudoMode,
		DryRun:      opts.DryRun,
		AutoInstall: opts.AutoInstall,
		Stdout:      opts.Stdout,
		Stderr:      opts.Stderr,
	}, resolution)
	if ensureErr != nil && !opts.DryRun {
		return report, fmt.Errorf("ensure host requirements (%s): %w", describeEnforcementLabel(opts.Label), ensureErr)
	}
	return report, nil
}

func describeEnforcementLabel(label string) string {
	if label == "" {
		return "unscoped"
	}
	return label
}
