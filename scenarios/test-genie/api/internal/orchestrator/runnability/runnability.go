// Package runnability is the single decider for whether a given phase may run
// against a given target in the current environment. It replaces the scattered
// lifecycle/eligibility/skip logic that used to live across the orchestrator
// (the runtimeNeeds switch, the EnsureRunning start-if-absent clobber, the
// playbooks routed-vs-fallback PathDecision, and the requirements_decision
// skip vocabulary) with one pure policy seam:
//
//	PhaseCapabilities × RunContext → Verdict{Run | RunDegraded | Skip}
//
// The package is intentionally a dependency-light leaf: it imports nothing from
// the orchestrator, phases, or eligibility packages, so the decision matrix is
// exhaustively unit-testable and the seam is promotable to a shared package
// later without dragging test-genie internals along.
package runnability

import (
	"strings"
)

// Resource names recognized by the runnability gate. They key RunContext.Resources.
const (
	// ResourceBAS is the Browser Automation Studio workflow engine, required by
	// phases that drive browser workflows or capture visual artifacts.
	ResourceBAS = "browser-automation-studio"
)

// DBIsolation classifies how a phase obtains an isolated database for its run.
type DBIsolation int

const (
	// DBIsolationNone — the phase does not need an isolated database. It reads
	// the live target (or no database at all). The vast majority of phases.
	DBIsolationNone DBIsolation = iota
	// DBIsolationRouted — the phase needs an isolated test database, obtained
	// only by the in-place routed path: a test pool is installed on the live
	// process via RoutingService (no scenario restart) when the target is
	// statically proven routed-eligible. When isolation cannot be proven the
	// phase is refused (there is no restart-based fallback — it was deleted in
	// favour of the storage-manager fail-closed gate). The playbooks phase is
	// the canonical (currently only) holder of this capability.
	DBIsolationRouted
)

// String renders the isolation mode for log/decision messaging.
func (d DBIsolation) String() string {
	switch d {
	case DBIsolationRouted:
		return "routed"
	default:
		return "none"
	}
}

// PhaseCapabilities is the declared, environment-independent contract of a
// phase: what runtime surfaces and resources it requires and how it mutates the
// target. It lives next to the phase in the catalog SSOT and is the only input
// to the resolver that varies per phase.
type PhaseCapabilities struct {
	// Phase is the canonical phase name, carried purely for human-readable
	// verdict reasons (kept as a plain string so this package stays a leaf and
	// does not import phases).
	Phase string
	// NeedsUI / NeedsAPI declare which live target surfaces the phase drives.
	// A phase that needs a surface cannot run unless that surface is live or
	// can be brought up without violating the self-host guard.
	NeedsUI  bool
	NeedsAPI bool
	// MutatesLifecycle reports that running the phase starts/replaces the target
	// process as part of its own execution (beyond merely needing the surface
	// up) — e.g. the playbooks phase may start the target on demand. Combined
	// with the self-host guard to forbid mutating our own live process.
	MutatesLifecycle bool
	// LifecycleDecisionDeferred declares that whether the phase ACTUALLY mutates
	// the lifecycle this run depends on runtime data the static manifest cannot
	// see (e.g. the playbooks registry's observer execution_mode, or an empty
	// registry, both of which run with no restart). When set, the resolver
	// gates only the surface and resource requirements and leaves the final
	// lifecycle/DB-isolation runnability call to the phase itself — which
	// enforces it through the same Verdict vocabulary and the selfidentity SSOT.
	// Without this flag a phase that declares MutatesLifecycle on a self-target
	// would be force-skipped even when its runtime path needs no restart.
	LifecycleDecisionDeferred bool
	// DBIsolation declares the phase's database-isolation requirement.
	DBIsolation DBIsolation
	// RequiredResources names local resources (e.g. "postgres",
	// "browser-automation-studio") that must be available for the phase to run.
	RequiredResources []string
	// Optional mirrors the catalog's Optional flag. The resolver does not gate
	// on it, but it travels with the capabilities so the suite layer can decide
	// whether a Skip of this phase degrades the overall verdict to PARTIAL.
	Optional bool
}

// Surfaces records which target runtime surfaces are currently live.
type Surfaces struct {
	UI  bool
	API bool
}

// RunContext is the resolved, phase-independent description of the environment
// a run executes in. It is computed once per suite execution and reused for
// every phase.
type RunContext struct {
	// TargetIsSelf reports that the scenario under test is this orchestrator's
	// own scenario. When true, any lifecycle mutation (start/restart) would
	// terminate the live process running the suite, so the self-host guard
	// forbids it.
	TargetIsSelf bool
	// LiveSurfaces records which target surfaces are already running and can be
	// reused without a lifecycle start.
	LiveSurfaces Surfaces
	// Resources is the set of locally available resource names.
	Resources map[string]bool
}

// HasResource reports whether the named resource is available in this context.
func (rc RunContext) HasResource(name string) bool {
	if rc.Resources == nil {
		return false
	}
	return rc.Resources[name]
}

// VerdictKind is the tri-state outcome of a runnability decision.
type VerdictKind int

const (
	// VerdictRun — the phase runs normally.
	VerdictRun VerdictKind = iota
	// VerdictRunDegraded — the phase runs, but on a less-preferred path. The run
	// still produces a real result; the reason explains the degradation.
	VerdictRunDegraded
	// VerdictSkip — the phase cannot run in this environment and is skipped
	// with a reason and (where actionable) a remediation.
	VerdictSkip
)

// String renders the verdict kind for logs.
func (k VerdictKind) String() string {
	switch k {
	case VerdictRun:
		return "run"
	case VerdictRunDegraded:
		return "run_degraded"
	case VerdictSkip:
		return "skip"
	default:
		return "unknown"
	}
}

// Verdict is the resolver's decision for one phase.
type Verdict struct {
	Kind        VerdictKind
	Reason      string
	Remediation string
}

// IsRun reports whether the phase should execute (normally or degraded).
func (v Verdict) IsRun() bool { return v.Kind == VerdictRun || v.Kind == VerdictRunDegraded }

// IsSkip reports whether the phase is skipped.
func (v Verdict) IsSkip() bool { return v.Kind == VerdictSkip }

// Resolver decides, for one phase, whether it may run in the given context.
//
// seam: Resolver is the runnability policy/decision seam. Production wires
// StandardResolver (resolver.go); tests wire mocks.FakeResolver
// (mocks/resolver.go). It is pure: same inputs always yield the same verdict.
type Resolver interface {
	Resolve(caps PhaseCapabilities, rc RunContext) Verdict
}

// phaseLabel returns a stable label for messaging.
func phaseLabel(caps PhaseCapabilities) string {
	name := strings.TrimSpace(caps.Phase)
	if name == "" {
		return "phase"
	}
	return "phase " + name
}

// missingSurfaces returns the human-readable list of required-but-not-live
// surfaces for a phase in a context.
func missingSurfaces(caps PhaseCapabilities, rc RunContext) []string {
	var missing []string
	if caps.NeedsUI && !rc.LiveSurfaces.UI {
		missing = append(missing, "UI")
	}
	if caps.NeedsAPI && !rc.LiveSurfaces.API {
		missing = append(missing, "API")
	}
	return missing
}

func joinReason(parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, "; ")
}
