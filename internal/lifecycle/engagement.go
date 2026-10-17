package lifecycle

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// sourceLocation is a physical place a scenario's source code can live.
type sourceLocation string

const (
	workingTree      sourceLocation = "working-tree"
	restorePointCopy sourceLocation = "restore-point-copy"
)

type sourceVariant string

const (
	liveVariant   sourceVariant = "live"
	shadowVariant sourceVariant = "shadow"
)

// locationForVariant is the lifecycle-owned directionality policy. With no
// engagement every variant runs from the working tree. Under the current
// policy an engaged live instance runs from the frozen copy while the shadow
// instance continues to run from the working tree.
func locationForVariant(v sourceVariant, engaged bool) sourceLocation {
	if !engaged {
		return workingTree
	}
	if v == liveVariant {
		return restorePointCopy
	}
	return workingTree
}

// EngagementResolver is the lifecycle's seam onto Baseline Modes engagement
// state. It answers a single FACT — "is scenario S currently engaged, and where
// are its code locations" — and deliberately does NOT decide which directory an
// instance should run from. That decision belongs to the directionality policy
// composed here by effectiveSourceDir. Keeping the two apart is what makes a
// layout flip a single-definition change: this seam reports reality, the policy
// owns the directionality.
//
// The interface is owned by the consumer (the lifecycle), per the
// seam-discovery contract. The production implementation is backed by
// internal/baselinefloor and injected at Runner construction so the lifecycle
// package never imports the floor directly; tests inject a fake.
type EngagementResolver interface {
	// Engagement reports whether scenario is in a SOURCE-DIR SPLIT and, if so, the
	// manifest facts needed to resolve its locations. engaged=true means there is
	// an active split where the serving (Baseline) instance must run from a frozen
	// copy while the working tree holds the Candidate — i.e. a SHADOW-mode
	// engagement. A live-mode engagement edits the working tree in place (its
	// restore point is a cold rollback, not a running source), so it creates no
	// split and is reported engaged=false. A scenario with no open engagement also
	// returns engaged=false with a zero EngagementInfo and no error.
	//
	// An error means the engagement state could not be determined (e.g. an
	// unreadable floor); callers must treat that as fail-closed for the serving
	// instance rather than guessing.
	Engagement(scenario string) (info EngagementInfo, engaged bool, err error)
}

// EngagementInfo carries the floor-manifest facts the lifecycle needs to map a
// role's Location to a concrete path. It holds no policy.
type EngagementInfo struct {
	// RestorePointDir is the absolute path to the frozen restore-point copy.
	RestorePointDir string
	// WorkingTreeDir is the absolute path to the canonical repo checkout. It is
	// informational/cross-check only; the lifecycle already holds the working
	// tree as item.Path.
	WorkingTreeDir string
	// Mode is the recorded engagement mode ("shadow" / "live").
	Mode string
	// Slug is the engagement slug (baseline-<slug>).
	Slug string
	// Generation identifies the restore-point capture: the capture ID, or the
	// engagement creation time for an engagement captured before capture IDs
	// existed. A serving tree materialized from another generation is stale.
	Generation string
	// SharedSourcesFrozen reports that the capture froze the shared sources
	// the build reaches outside the scenario (Go replace closure, pnpm file:
	// dependencies, bundler aliases) together with their generated outputs.
	// A serving instance built from such a tree must not regenerate the
	// repository's shared outputs: it does not read them.
	SharedSourcesFrozen bool
	// CaptureID and ProtoArtifactID name the frozen capture in lifecycle logs,
	// so a frozen build failure points at the capture it compiled against.
	CaptureID       string
	ProtoArtifactID string
}

// ServingSourceMaterializer is an optional capability of an
// EngagementResolver. The restore point is a bare copy of the scenario
// directory, so the relative references in its sources (Go replace
// directives, pnpm file: dependencies, bundler aliases) do not resolve from
// it. A materializer prepares a repo-shaped serving tree around the frozen
// sources and returns the scenario directory in it. When the resolver does
// not implement it, the serving instance runs from the restore point itself.
type ServingSourceMaterializer interface {
	MaterializeServingSource(scenario string, info EngagementInfo, repoRoot string) (string, error)
}

// defaultEngagementResolver is the process-wide engagement resolver applied to
// every Runner built by NewRunner. It is nil by default — so test binaries,
// which never call SetDefaultEngagementResolver, get the pre-Baseline-Modes
// "always run from the working tree" behavior and stay hermetic. The production
// binary wires the baselinefloor-backed resolver once at startup. This is the
// single construction-edge injection point the plan describes: lifecycle still
// depends only on the interface, the concrete (which imports the floor) is
// supplied from outside.
var defaultEngagementResolver EngagementResolver

// SetDefaultEngagementResolver installs the process-wide engagement resolver.
// Call it exactly once, early in the production binary's startup, before any
// Runner is constructed. Passing nil restores the no-engagement behavior.
func SetDefaultEngagementResolver(r EngagementResolver) {
	defaultEngagementResolver = r
}

// layoutVariant maps a scenario descriptor's variant onto the lifecycle-owned
// variant: the canonical live instance is live; any other variant
// (conventionally "@shadow") is a shadow instance. The variant is normalized
// through InstanceKey first, because the start path stamps the normalized
// "live" onto the descriptor while other callers leave it empty — reading the
// raw string treated an engaged LIVE instance as a shadow and left it running
// from the working tree the candidate owns.
func layoutVariant(item scenario.Scenario) sourceVariant {
	key := scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}.Normalize()
	if key.IsLive() {
		return liveVariant
	}
	return shadowVariant
}

// effectiveSourceDir is the single named decision for "which directory does this
// instance run and build from". It composes the engagement-state fact (from the
// resolver seam) with the directionality policy — it holds no
// `if live`/`if shadow` policy of its own. All three run-CWD sinks
// (background step Dir, foreground step Dir, process/registry WorkingDir) call
// this instead of reading item.Path directly, so the rule lives in exactly one
// place.
//
// Manifest/port/DB/storage resolution is unaffected: only the run/build CWD
// diverges. item.Path remains the working tree (the manifest origin); when the
// layout routes this instance to the restore-point copy, that copy is returned
// instead.
func (r *Runner) effectiveSourceDir(item scenario.Scenario) (string, error) {
	if r.Engagements == nil {
		return item.Path, nil
	}
	info, engaged, err := r.Engagements.Engagement(item.Slug)
	if err != nil {
		// Fail closed: we could not determine engagement state, so we cannot
		// safely route the serving instance. Surface it rather than defaulting to
		// the working tree (which could expose live to candidate code).
		return "", fmt.Errorf("resolve engagement for %q: %w", item.Slug, err)
	}
	loc := locationForVariant(layoutVariant(item), engaged)
	switch loc {
	case restorePointCopy:
		if info.RestorePointDir == "" {
			return "", fmt.Errorf("engagement for %q routes %s to the restore-point copy but the manifest has no restore-point path",
				item.Slug, layoutVariant(item))
		}
		sourceDir := info.RestorePointDir
		if materializer, ok := r.Engagements.(ServingSourceMaterializer); ok {
			sourceDir, err = materializer.MaterializeServingSource(item.Slug, info, r.Root)
			if err != nil {
				// Fail closed: never fall back to the working tree the candidate owns.
				return "", fmt.Errorf("materialize serving source for %q: %w", item.Slug, err)
			}
		}
		r.logDebug("Resolving instance source dir to restore-point copy",
			logx.AttrScenario, item.Slug,
			"variant", string(layoutVariant(item)),
			"engagement_slug", info.Slug,
			"restore_point", info.RestorePointDir,
			"source_dir", sourceDir,
		)
		return sourceDir, nil
	default:
		return item.Path, nil
	}
}

// withSourceLayout stamps the directory this instance builds and runs from onto
// the descriptor. Path stays the manifest origin: data directories, locks, and
// records are keyed to it. Only build inputs, build outputs, and process
// working directories follow the layout, which is what makes an engaged live
// instance serve the frozen copy while the working tree holds the candidate.
func (r *Runner) withSourceLayout(item scenario.Scenario) (scenario.Scenario, error) {
	dir, err := r.effectiveSourceDir(item)
	if err != nil {
		return item, err
	}
	if dir != "" && dir != item.Path {
		item.SourceDir = dir
	}
	return item, nil
}

// frozenServingSources reports whether item builds from a serving tree whose
// capture froze the shared sources. Only an instance routed away from the
// working tree (SourceDir set) qualifies; the answer comes from the same
// engagement fact the layout decision used.
func (r *Runner) frozenServingSources(item scenario.Scenario) (EngagementInfo, bool) {
	if r.Engagements == nil || item.SourceDir == "" || item.SourceDir == item.Path {
		return EngagementInfo{}, false
	}
	info, engaged, err := r.Engagements.Engagement(item.Slug)
	if err != nil || !engaged || !info.SharedSourcesFrozen {
		return EngagementInfo{}, false
	}
	return info, true
}
