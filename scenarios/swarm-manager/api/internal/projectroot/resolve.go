// Package projectroot resolves the (ProjectRoot, ScopePath) pair that should
// be passed to agent-manager when spawning a sandbox for a backlog item.
//
// The resolver derives the target scenario from the item's acceptance_allow
// globs (which by convention start with "scenarios/<name>/...") and uses
// repo-contract to locate the monorepo root on disk. The intended workspace-
// sandbox model is:
//
//	ProjectRoot = monorepo root        (full repo for path-context)
//	ScopePath   = scenarios/<scenario> (narrow working area for the lowerdir)
//
// Acceptance globs are matched project-relative, so this layout lets globs
// like "scenarios/foo/cli/**" match changes that are scope-relative as
// "cli/foo.go". See workspace-sandbox/api/internal/sandbox/service.go's
// projectRelativePath for the matching contract.
//
// DOC: docs/internal/SEAMS.md
package projectroot

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	repocontract "github.com/vrooli/repo-contract-go"

	"swarm-manager/internal/pathutil"
)

// Resolution is the absolute ProjectRoot and ProjectRoot-relative ScopePath
// for a backlog item's spawn request.
type Resolution struct {
	// ProjectRoot is the absolute filesystem path the sandbox should treat
	// as the project (monorepo) root. Always absolute.
	ProjectRoot string

	// ScopePath is the working area, relative to ProjectRoot. For items that
	// target a single scenario this is "scenarios/<name>"; for items that do
	// not unambiguously target one scenario it is ".".
	ScopePath string

	// TargetScenario is the scenario name the resolver locked onto, or "" if
	// no single scenario could be determined. Useful for logging and tests.
	TargetScenario string
}

// Options controls Resolve. Today only AcceptanceAllow is consulted; the
// struct exists so per-item or per-initiative overrides can be added without
// breaking call sites.
type Options struct {
	// AcceptanceAllow is the backlog item's acceptance_allow glob list.
	AcceptanceAllow []string
}

// Errors returned by Resolve. Callers should use errors.Is for matching.
var (
	// ErrAmbiguousScenarios is returned when AcceptanceAllow names more than
	// one distinct scenario. Cross-scenario items must declare their target
	// explicitly (future: BacklogItem.TargetProjectRoot).
	ErrAmbiguousScenarios = errors.New("acceptance_allow targets multiple scenarios; explicit target_project_root required")

	// ErrRepoRootUnresolvable is returned when repo-contract cannot locate
	// the monorepo root from the environment or working directory.
	ErrRepoRootUnresolvable = errors.New("monorepo root could not be resolved via repo-contract")
)

// Resolve produces the (ProjectRoot, ScopePath) pair for a backlog item.
//
// Behavior:
//   - Exactly one scenario in AcceptanceAllow → narrow scope to that scenario.
//   - Zero scenarios (empty list, or globs that do not match scenarios/<name>/…)
//     → wide scope (ProjectRoot=monorepo, ScopePath="."). A warning is logged;
//     this case is rare and indicates an item that touches monorepo-shared code.
//   - More than one distinct scenario → ErrAmbiguousScenarios.
//
// ProjectRoot is always absolute. ScopePath is always ProjectRoot-relative
// using forward slashes (workspace-sandbox normalizes via filepath.Join).
func Resolve(opts Options) (Resolution, error) {
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return Resolution{}, fmt.Errorf("%w: %v", ErrRepoRootUnresolvable, err)
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return Resolution{}, fmt.Errorf("%w: absolutize %q: %v", ErrRepoRootUnresolvable, repoRoot, err)
	}

	scenarios := pathutil.ScenariosFromGlobs(opts.AcceptanceAllow)
	switch len(scenarios) {
	case 1:
		return Resolution{
			ProjectRoot:    absRoot,
			ScopePath:      "scenarios/" + scenarios[0],
			TargetScenario: scenarios[0],
		}, nil
	case 0:
		slog.Warn("projectroot: no target scenario inferable from acceptance_allow; falling back to monorepo-wide scope",
			"acceptance_allow", opts.AcceptanceAllow,
			"project_root", absRoot,
		)
		return Resolution{
			ProjectRoot: absRoot,
			ScopePath:   ".",
		}, nil
	default:
		return Resolution{}, fmt.Errorf("%w: scenarios=%v", ErrAmbiguousScenarios, scenarios)
	}
}
