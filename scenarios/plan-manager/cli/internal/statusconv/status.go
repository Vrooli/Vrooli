package statusconv

import (
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// PlanStatusFlag parses a user-facing plan status flag.
func PlanStatusFlag(s string) sharedv1.PlanStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "draft":
		return sharedv1.PlanStatus_PLAN_STATUS_DRAFT
	case "active":
		return sharedv1.PlanStatus_PLAN_STATUS_ACTIVE
	case "complete":
		return sharedv1.PlanStatus_PLAN_STATUS_COMPLETE
	case "archived":
		return sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED
	default:
		return sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED
	}
}

// PhaseStatusFlag parses a user-facing phase status flag.
func PhaseStatusFlag(s string) sharedv1.PhaseStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "todo":
		return sharedv1.PhaseStatus_PHASE_STATUS_TODO
	case "active":
		return sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE
	case "done":
		return sharedv1.PhaseStatus_PHASE_STATUS_DONE
	case "blocked":
		return sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED
	default:
		return sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED
	}
}

// PlanStatusLabel formats a plan status for CLI output.
func PlanStatusLabel(s sharedv1.PlanStatus) string {
	switch s {
	case sharedv1.PlanStatus_PLAN_STATUS_DRAFT:
		return "draft"
	case sharedv1.PlanStatus_PLAN_STATUS_ACTIVE:
		return "active"
	case sharedv1.PlanStatus_PLAN_STATUS_COMPLETE:
		return "complete"
	case sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED:
		return "archived"
	default:
		return "unspecified"
	}
}

// PhaseStatusLabel formats a phase status for CLI output.
func PhaseStatusLabel(s sharedv1.PhaseStatus) string {
	switch s {
	case sharedv1.PhaseStatus_PHASE_STATUS_TODO:
		return "todo"
	case sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE:
		return "active"
	case sharedv1.PhaseStatus_PHASE_STATUS_DONE:
		return "done"
	case sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED:
		return "blocked"
	default:
		return "unspecified"
	}
}

// PlanPhaseStatusLabel formats a phase status in plan displays, where an
// unspecified phase is treated as not-yet-started.
func PlanPhaseStatusLabel(s sharedv1.PhaseStatus) string {
	if s == sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED {
		return "todo"
	}
	return PhaseStatusLabel(s)
}
