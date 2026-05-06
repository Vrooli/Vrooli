// Package hostreqrun bundles the host-requirement resolve + ensure machinery
// behind a single entry point so that lifecycle hooks (scenario start, resource
// install/start, and `vrooli setup`) can all enforce declared tools and
// safeguards through the same code path.
package hostreqrun

import (
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/hostreq"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

// Options describes a single enforcement call.
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

// Deps allows tests to stub the underlying resolve/ensure calls.
type Deps struct {
	Resolve func(root, home string, opts hostreq.ResolveOptions) (hostreq.Resolution, error)
	Ensure  func(opts vrooliruntime.EnsureOptions, resolution hostreq.Resolution) (vrooliruntime.Report, error)
}

// DefaultDeps wires the real resolver and runtime.
func DefaultDeps() Deps {
	return Deps{
		Resolve: hostreq.Resolve,
		Ensure:  vrooliruntime.EnsureRequirements,
	}
}

// Enforce resolves host requirements for the configured scope and ensures them
// via the runtime registry. When no requirements are declared for the scope
// the function is a no-op. When auto-install is true, missing required handlers
// are applied; failures bubble up as errors unless DryRun is set.
func Enforce(opts Options) (vrooliruntime.Report, error) {
	return EnforceWithDeps(DefaultDeps(), opts)
}

// EnforceWithDeps lets callers inject Resolve/Ensure implementations (used by
// tests).
func EnforceWithDeps(deps Deps, opts Options) (vrooliruntime.Report, error) {
	if deps.Resolve == nil {
		deps.Resolve = hostreq.Resolve
	}
	if deps.Ensure == nil {
		deps.Ensure = vrooliruntime.EnsureRequirements
	}

	platform := opts.Platform
	if platform == "" {
		platform = hostreq.CurrentPlatform()
	}

	resolution, err := deps.Resolve(opts.Root, opts.Home, hostreq.ResolveOptions{
		Environment:   opts.Environment,
		When:          opts.When,
		Resources:     opts.Resources,
		Scenarios:     opts.Scenarios,
		ScenarioPaths: opts.ScenarioPaths,
		Platform:      platform,
	})
	if err != nil {
		return vrooliruntime.Report{}, fmt.Errorf("resolve host requirements (%s): %w", describe(opts.Label), err)
	}
	if len(resolution.Tools) == 0 && len(resolution.Safeguards) == 0 {
		return vrooliruntime.Report{}, nil
	}

	report, ensureErr := deps.Ensure(vrooliruntime.EnsureOptions{
		Environment: opts.Environment,
		SudoMode:    opts.SudoMode,
		DryRun:      opts.DryRun,
		AutoInstall: opts.AutoInstall,
		Stdout:      opts.Stdout,
		Stderr:      opts.Stderr,
	}, resolution)
	if ensureErr != nil && !opts.DryRun {
		return report, fmt.Errorf("ensure host requirements (%s): %w", describe(opts.Label), ensureErr)
	}
	return report, nil
}

func describe(label string) string {
	if label == "" {
		return "unscoped"
	}
	return label
}
