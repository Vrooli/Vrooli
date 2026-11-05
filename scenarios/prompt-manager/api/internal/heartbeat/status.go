package heartbeat

import "time"

func ownerRunObligationState(status string) string {
	if IsNeedsReviewStatus(status) {
		return ObligationNeedsReview
	}
	if IsParkedStatus(status) {
		return ObligationPaused
	}
	return ObligationRunning
}

type ownerRunObservation struct {
	state          string
	status         string
	terminal       bool
	startedAt      time.Time
	startedAtKnown bool
}

// observeOwnerRun is the shared interpretation of an exact owner lookup.
// Queue and registry recovery keep their own persistence and admission rules,
// but must classify identity, uncertainty, lifecycle status, and original
// start time consistently.
func observeOwnerRun(expectedRunID string, run *Run, err error) ownerRunObservation {
	if err != nil {
		return ownerRunObservation{state: ObligationOwnerUnreachable, status: ObligationOwnerUnreachable}
	}
	if run == nil || expectedRunID == "" || run.ID != expectedRunID {
		return ownerRunObservation{state: ObligationDispatchUncertain, status: ObligationDispatchUncertain}
	}
	observation := ownerRunObservation{status: run.Status, terminal: IsTerminalStatus(run.Status)}
	if observation.startedAt, observation.startedAtKnown = parseOwnerRunStartedAt(run.StartedAt); !observation.startedAtKnown {
		observation.startedAt = time.Time{}
	}
	if observation.terminal {
		return observation
	}
	observation.state = ownerRunObligationState(run.Status)
	return observation
}

// Terminal run statuses returned by agent-manager.
//
// agent-manager serializes via protojson with UseProtoNames=true, so
// responses contain proto enum names (e.g. "RUN_STATUS_COMPLETE").
// Lowercase variants are included as a safety net for domain-style values.
//
// Source of truth: packages/proto/schemas/agent-manager/v1/domain/types.proto
// Domain values: scenarios/agent-manager/api/internal/domain/types.go
var terminalStatuses = map[string]bool{
	"RUN_STATUS_COMPLETE":  true,
	"RUN_STATUS_FAILED":    true,
	"RUN_STATUS_CANCELLED": true,
	"complete":             true,
	"failed":               true,
	"cancelled":            true,
}

// IsTerminalStatus reports whether s represents a finished run.
func IsTerminalStatus(s string) bool { return terminalStatuses[s] }

// IsFailedStatus reports whether s represents a failed run.
func IsFailedStatus(s string) bool {
	return s == "RUN_STATUS_FAILED" || s == "failed"
}

// IsCancelledStatus reports whether s represents a cancelled run.
func IsCancelledStatus(s string) bool {
	return s == "RUN_STATUS_CANCELLED" || s == "cancelled"
}

// parkedStatuses are nonterminal run statuses where the owner has suspended the
// run (for example, waiting on externally-owned async work or a quota pause).
// A parked run is retained ownership: it must never be mistaken for a missing
// or dead run, and scheduling must not start a duplicate.
var parkedStatuses = map[string]bool{
	"RUN_STATUS_PARKED": true,
	"parked":            true,
}

// IsParkedStatus reports whether s represents a suspended (nonterminal) run.
func IsParkedStatus(s string) bool { return parkedStatuses[s] }

// IsNeedsReviewStatus reports a run that stopped executing and is waiting on
// an explicit operator decision. It remains nonterminal retained ownership.
func IsNeedsReviewStatus(s string) bool {
	return s == "RUN_STATUS_NEEDS_REVIEW" || s == "needs_review"
}
