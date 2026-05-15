package agentactivity

import (
	"errors"
	"fmt"
	"strings"
)

// Lane is one of the four canonical concurrency lanes. Lane names are
// identical to `operatingmode.PhaseKind` values — Lane is the runtime
// projection of the phase classification, used here as a string to avoid
// importing operatingmode (which already imports agentactivity).
type Lane string

// Lane constants mirror operatingmode.PhaseKind. Keep these in lock-step.
const (
	LaneInvestigate Lane = "investigate"
	LaneExecute     Lane = "execute"
	LaneReview      Lane = "review"
	LaneReconcile   Lane = "reconcile"
)

// Lanes returns the canonical four-lane ordering used by Operations Center
// columns, GovernanceStatusResponse.Lanes, and per-lane stats aggregation.
// Returning a fresh slice keeps callers from mutating shared state.
func Lanes() []Lane {
	return []Lane{LaneInvestigate, LaneExecute, LaneReview, LaneReconcile}
}

// IsValidLane reports whether l is one of the four canonical lanes.
func IsValidLane(l Lane) bool {
	switch l {
	case LaneInvestigate, LaneExecute, LaneReview, LaneReconcile:
		return true
	default:
		return false
	}
}

// ErrLaneSaturated is returned when a tracked spawn would exceed the lane's
// configured concurrency cap. Backlog-process callers translate this into a
// pending-state enqueue (see execution.QueueBacklog); ad-hoc spawn paths
// (workshop / clarify / finalize / classify / operating-mode phase) surface
// the error so the caller can decide.
var ErrLaneSaturated = errors.New("lane concurrency cap reached")

// LanePolicy is the seam through which a Service learns each lane's cap.
// The settings store provides the canonical implementation; tests can
// supply a fixed-map stub.
type LanePolicy interface {
	// LimitFor returns the concurrency cap for the lane. A return value of
	// zero or less is treated as "no cap" so tests and partial settings
	// loads do not accidentally throttle work.
	LimitFor(lane Lane) int
}

// purposeLane is the canonical Purpose → default Lane map used when a Spec
// has no explicit PhaseKind. Every Purpose constant declared in types.go
// MUST appear here; init() panics on first import otherwise so a forgotten
// lane assignment cannot ship.
var purposeLane = map[Purpose]Lane{
	PurposeInitialize:             LaneInvestigate,
	PurposeWorkshop:               LaneInvestigate,
	PurposeFinalize:               LaneReview,
	PurposeResearch:               LaneInvestigate,
	PurposeProcess:                LaneExecute,
	PurposeFixup:                  LaneExecute,
	PurposeFollowUp:               LaneExecute,
	PurposeSpecSync:               LaneExecute,
	PurposeClassify:               LaneInvestigate,
	PurposeClarify:                LaneInvestigate,
	PurposeReview:                 LaneReview,
	PurposeFeedback:               LaneInvestigate,
	PurposeFeedbackContinue:       LaneInvestigate,
	PurposeInitiativeReview:       LaneReview,
	PurposeMetaOrchestration:      LaneInvestigate,
	PurposeOperatingModeAuthoring: LaneInvestigate,
	PurposeSwarmOperations:        LaneInvestigate,
}

// allRegisteredPurposes mirrors the Purpose constants in types.go. It is
// the source list init() walks to assert lane coverage. New Purpose
// constants added without updating this list (and purposeLane) cause an
// init-time panic — that is intentional: the lane-derivation seam is the
// single source of truth and silent drift would mask capacity issues.
var allRegisteredPurposes = []Purpose{
	PurposeInitialize,
	PurposeWorkshop,
	PurposeFinalize,
	PurposeResearch,
	PurposeProcess,
	PurposeFixup,
	PurposeFollowUp,
	PurposeSpecSync,
	PurposeClassify,
	PurposeClarify,
	PurposeReview,
	PurposeFeedback,
	PurposeFeedbackContinue,
	PurposeInitiativeReview,
	PurposeMetaOrchestration,
	PurposeOperatingModeAuthoring,
	PurposeSwarmOperations,
}

func init() {
	for _, p := range allRegisteredPurposes {
		if _, ok := purposeLane[p]; !ok {
			panic(fmt.Sprintf("agentactivity: purpose %q has no lane assignment in purposeLane (lanes.go); add one before this package can be loaded", p))
		}
	}
}

// laneFromPhaseKind maps an operatingmode.PhaseKind value (carried as a
// string to avoid an import cycle) to its lane. Inputs are normalized to
// lower-case + trimmed to mirror Spec.normalized().
func laneFromPhaseKind(phaseKind string) (Lane, bool) {
	switch strings.ToLower(strings.TrimSpace(phaseKind)) {
	case string(LaneInvestigate):
		return LaneInvestigate, true
	case string(LaneExecute):
		return LaneExecute, true
	case string(LaneReview):
		return LaneReview, true
	case string(LaneReconcile):
		return LaneReconcile, true
	default:
		return "", false
	}
}

// LaneOf returns the lane for a (purpose, phaseKind) pair.
//
// PhaseKind, when non-empty and valid, takes precedence over the Purpose
// default. Operating-mode phase runs explicitly classify themselves with a
// kind, and a single dynamic mode-defined purpose (e.g.
// "holistic_loop_review" vs "holistic_loop_reconcile") would otherwise
// share a lane.
//
// When phaseKind is empty, the per-Purpose default lane is used. This is
// the path taken by ad-hoc spawn paths (workshop / clarify / finalize /
// classify / backlog process) before they are wired through with explicit
// phaseKind values.
//
// Returns an error when:
//   - phaseKind is set but unrecognized — typo-protection for new lanes.
//   - phaseKind is empty AND purpose has no default lane (typically an
//     operating-mode dynamic purpose that did not propagate phaseKind).
//
// LaneOf is the ONLY place lane derivation lives. Call sites that need the
// lane for bookkeeping must call this function — never reimplement the map
// or duplicate the precedence rule.
func LaneOf(purpose Purpose, phaseKind string) (Lane, error) {
	if strings.TrimSpace(phaseKind) != "" {
		if lane, ok := laneFromPhaseKind(phaseKind); ok {
			return lane, nil
		}
		return "", fmt.Errorf("unknown phase_kind %q (expected one of investigate|execute|review|reconcile)", phaseKind)
	}
	if lane, ok := purposeLane[purpose]; ok {
		return lane, nil
	}
	return "", fmt.Errorf("no lane registered for purpose %q (and no phase_kind set)", purpose)
}

// LaneActiveCount returns the number of currently-active records in the
// given lane. Records whose lane cannot be derived (e.g. an operating-mode
// dynamic purpose written before P2 wired phaseKind) are skipped — they do
// not count against any lane. The Operations Center surfaces such records
// in its activity list so operators can see them, but lane-cap math
// excludes them to avoid false saturation.
func LaneActiveCount(records []Record, lane Lane) int {
	count := 0
	for _, rec := range records {
		if !isActiveStatus(rec.Status) {
			continue
		}
		recLane, err := LaneOf(rec.Purpose, rec.PhaseKind)
		if err != nil {
			continue
		}
		if recLane == lane {
			count++
		}
	}
	return count
}

// LaneActiveCounts returns active counts for every canonical lane in the
// order returned by Lanes(). Used by Operations Center / governance status
// to render utilization without N+1 store reads.
func LaneActiveCounts(records []Record) map[Lane]int {
	counts := make(map[Lane]int, 4)
	for _, lane := range Lanes() {
		counts[lane] = 0
	}
	for _, rec := range records {
		if !isActiveStatus(rec.Status) {
			continue
		}
		recLane, err := LaneOf(rec.Purpose, rec.PhaseKind)
		if err != nil {
			continue
		}
		counts[recLane]++
	}
	return counts
}
