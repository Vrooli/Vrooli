package lifecycle

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/engagementlayout"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenario"
)

// EngagementResolver is the lifecycle's seam onto Baseline Modes engagement
// state. It answers a single FACT — "is scenario S currently engaged, and where
// are its code locations" — and deliberately does NOT decide which directory an
// instance should run from. That decision belongs to the directionality policy
// (internal/engagementlayout), composed here by effectiveSourceDir. Keeping the
// two apart is what makes a layout flip a single-definition change: this seam
// reports reality, the layout owns the policy.
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

// layoutVariant maps a scenario descriptor's variant onto the engagement-layout
// variant: the canonical empty-variant instance is Live; any non-empty variant
// (conventionally "@shadow") is a Shadow instance.
func layoutVariant(item scenario.Scenario) engagementlayout.Variant {
	if item.Variant == "" {
		return engagementlayout.Live
	}
	return engagementlayout.Shadow
}

// effectiveSourceDir is the single named decision for "which directory does this
// instance run and build from". It composes the engagement-state fact (from the
// resolver seam) with the directionality policy (from engagementlayout) — it
// holds no `if live`/`if shadow` policy of its own. All three run-CWD sinks
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
	loc := engagementlayout.Default().LocationForVariant(layoutVariant(item), engaged)
	switch loc {
	case engagementlayout.RestorePointCopy:
		if info.RestorePointDir == "" {
			return "", fmt.Errorf("engagement for %q routes %s to the restore-point copy but the manifest has no restore-point path",
				item.Slug, layoutVariant(item))
		}
		r.logDebug("Resolving instance source dir to restore-point copy",
			logx.AttrScenario, item.Slug,
			"variant", string(layoutVariant(item)),
			"engagement_slug", info.Slug,
			"source_dir", info.RestorePointDir,
		)
		return info.RestorePointDir, nil
	default:
		return item.Path, nil
	}
}
