// Package engagementlayout is the single source of truth for Baseline Modes
// directionality: which engagement ROLE runs from which physical LOCATION, and
// which instance VARIANT serves which role.
//
// This is the one design axis that must never be restated in more than one
// place. Capture, the lifecycle source-dir resolver, promote/abandon, and the
// safety invariant test all DERIVE from the Layout declared here — none of them
// hardcode "live runs from the copy" or "shadow runs from the working tree".
// Flipping that policy (e.g. the user's "shadow-at-temp / live-at-repo"
// arrangement) is a single-definition edit: point Default at InvertedLayout, or
// swap the one role→location mapping. Every derivation recompiles unchanged.
//
// The package is a leaf: it imports only the standard library, so both
// internal/lifecycle and internal/baselinefloor can depend on it without a
// cycle.
package engagementlayout

import "fmt"

// Role is what a code location represents during an open engagement.
type Role string

const (
	// Baseline is the known-good code the SERVING instance must keep running.
	// Touching the location holding the Baseline role while an engagement is
	// open would expose the serving instance to unvalidated code — the property
	// the invariant test guards.
	Baseline Role = "baseline"
	// Candidate is the in-progress code being validated. The agent's merge lands
	// in the Candidate's location (see EditedLocation).
	Candidate Role = "candidate"
)

// Location is a physical place a scenario's source code can live.
type Location string

const (
	// WorkingTree is the canonical repo checkout: scenarios/<name>/.
	WorkingTree Location = "working-tree"
	// RestorePointCopy is the frozen pre-edit copy under the floor cache:
	// <cache>/vrooli/<scenario>/baseline-<slug>/restore-point/.
	RestorePointCopy Location = "restore-point-copy"
)

// Variant is which instance of a scenario is being addressed. The lifecycle
// maps its descriptor variant onto these: the canonical empty-variant instance
// is Live; any non-empty variant (conventionally "@shadow") is Shadow.
type Variant string

const (
	// Live is the canonical, externally-addressable instance.
	Live Variant = "live"
	// Shadow is a second, isolated instance run alongside live for validation.
	Shadow Variant = "shadow"
)

// Layout is the declared directionality mapping. It is expressed as two small
// total functions — variant→role and role→location — from which all three
// public derivations are computed. A Layout value is immutable; construct the
// standard and inverted arrangements as package-level vars and select one via
// Default.
type Layout struct {
	// Name identifies the arrangement in logs/diagnostics.
	Name string
	// variantRole records which role each variant serves. Unexported so the
	// mapping can only be read through the derivations below.
	variantRole map[Variant]Role
	// roleLocation records where each role's code lives.
	roleLocation map[Role]Location
}

// StandardLayout is the default arrangement: the in-progress Candidate stays in
// the working tree (served by the shadow instance) while the known-good
// Baseline is frozen in the restore-point copy (served by live). This is what
// makes "open a shadow without restarting live" safe: live keeps serving the
// Baseline, and a later live restart resolves to the copy rather than the
// working tree the agent is editing.
var StandardLayout = Layout{
	Name:         "standard",
	variantRole:  map[Variant]Role{Live: Baseline, Shadow: Candidate},
	roleLocation: map[Role]Location{Baseline: RestorePointCopy, Candidate: WorkingTree},
}

// InvertedLayout is the "shadow-at-temp / live-at-repo" arrangement: the
// Baseline (served by live) stays in the working tree and the Candidate (served
// by shadow) is the restore-point copy. It exists to demonstrate — and keep
// honest — that flipping directionality is a single-definition change: every
// derivation, capture, promote/abandon, and the invariant test follow this
// value with no further edits. The one coupling that does NOT follow silently —
// where the sandbox merges the agent's overlay — is surfaced by EditedLocation
// and asserted by the invariant test, not left to corrupt state.
var InvertedLayout = Layout{
	Name:         "inverted",
	variantRole:  map[Variant]Role{Live: Baseline, Shadow: Candidate},
	roleLocation: map[Role]Location{Baseline: WorkingTree, Candidate: RestorePointCopy},
}

// Default returns the active layout. Changing the deployed directionality is a
// one-line edit here.
func Default() Layout {
	return StandardLayout
}

// RoleForVariant reports which role the given variant serves. This is the ONLY
// reader of the variant→role mapping.
func (l Layout) RoleForVariant(v Variant) Role {
	return l.variantRole[v]
}

// LocationForVariant reports where the given variant's code should run from.
//
// When no engagement is open for the scenario, there is no restore-point copy
// and every variant runs from the canonical working tree. When an engagement IS
// open, the variant runs from wherever its role lives under the layout. This is
// the ONLY place the variant→role→location chain is composed for runtime
// dir-resolution; the lifecycle's effectiveSourceDir calls it and nothing else.
func (l Layout) LocationForVariant(v Variant, engaged bool) Location {
	if !engaged {
		return WorkingTree
	}
	return l.roleLocation[l.RoleForVariant(v)]
}

// LocationForRole reports where the given role's code lives under the layout.
func (l Layout) LocationForRole(r Role) Location {
	return l.roleLocation[r]
}

// EditedLocation reports the location that receives the agent's merge — the
// Candidate's location. It names the single physical coupling that cannot
// silently follow a layout flip (the sandbox merge target must agree with it),
// so a flip surfaces at exactly one boundary instead of corrupting the serving
// instance. The invariant test asserts the serving (Baseline) instance never
// runs from here.
func (l Layout) EditedLocation() Location {
	return l.roleLocation[Candidate]
}

// Validate reports whether the layout's mappings are total and internally
// consistent: both variants and both roles are mapped, and Baseline/Candidate
// occupy distinct locations (a layout that put both roles in one location would
// make isolation meaningless). It is called by tests and may be called at
// wiring time as a cheap guard.
func (l Layout) Validate() error {
	for _, v := range []Variant{Live, Shadow} {
		role, ok := l.variantRole[v]
		if !ok {
			return fmt.Errorf("engagementlayout %q: variant %q has no role", l.Name, v)
		}
		if role != Baseline && role != Candidate {
			return fmt.Errorf("engagementlayout %q: variant %q maps to unknown role %q", l.Name, v, role)
		}
	}
	for _, role := range []Role{Baseline, Candidate} {
		loc, ok := l.roleLocation[role]
		if !ok {
			return fmt.Errorf("engagementlayout %q: role %q has no location", l.Name, role)
		}
		if loc != WorkingTree && loc != RestorePointCopy {
			return fmt.Errorf("engagementlayout %q: role %q maps to unknown location %q", l.Name, role, loc)
		}
	}
	if l.roleLocation[Baseline] == l.roleLocation[Candidate] {
		return fmt.Errorf("engagementlayout %q: baseline and candidate share location %q", l.Name, l.roleLocation[Baseline])
	}
	return nil
}

// ServingInstanceIsolated reports the core safety property as a pure predicate
// over the layout: while an engagement is open, the instance serving the
// Baseline role must NOT run from the EditedLocation (the location receiving the
// agent's merge). The invariant test asserts this for every supported layout, so
// a directionality flip that would expose the serving instance fails loudly at
// one place rather than silently.
func (l Layout) ServingInstanceIsolated() bool {
	return l.LocationForRole(Baseline) != l.EditedLocation()
}
