// Package types contains shared domain types for requirements synchronization.
package types

import "strings"

// DeclaredStatus is user-specified requirement status.
type DeclaredStatus string

const (
	StatusPending        DeclaredStatus = "pending"
	StatusPlanned        DeclaredStatus = "planned"
	StatusInProgress     DeclaredStatus = "in_progress"
	StatusComplete       DeclaredStatus = "complete"
	StatusNotImplemented DeclaredStatus = "not_implemented"
)

// LiveStatus is derived from test evidence.
type LiveStatus string

const (
	LivePassed  LiveStatus = "passed"
	LiveFailed  LiveStatus = "failed"
	LiveSkipped LiveStatus = "skipped"
	LiveNotRun  LiveStatus = "not_run"
	LiveUnknown LiveStatus = "unknown"
)

// ValidationStatus is derived from live test results.
type ValidationStatus string

const (
	ValStatusNotImplemented ValidationStatus = "not_implemented"
	ValStatusPlanned        ValidationStatus = "planned"
	ValStatusImplemented    ValidationStatus = "implemented"
	ValStatusFailing        ValidationStatus = "failing"
)

// Criticality levels from PRD.
type Criticality string

const (
	CriticalityP0 Criticality = "P0"
	CriticalityP1 Criticality = "P1"
	CriticalityP2 Criticality = "P2"
)

// ValidationType categorizes validation sources.
type ValidationType string

const (
	ValTypeTest       ValidationType = "test"
	ValTypeAutomation ValidationType = "automation"
	ValTypeManual     ValidationType = "manual"
	ValTypeLighthouse ValidationType = "lighthouse"
)

// statusPriority determines which status wins when aggregating.
// Higher values take precedence.
var statusPriority = map[LiveStatus]int{
	LiveFailed:  4,
	LiveSkipped: 3,
	LivePassed:  2,
	LiveNotRun:  1,
	LiveUnknown: 0,
}

// NormalizeLiveStatus standardizes raw status strings to LiveStatus.
func NormalizeLiveStatus(s string) LiveStatus {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "passed", "pass", "success", "ok":
		return LivePassed
	case "failed", "fail", "failure", "error":
		return LiveFailed
	case "skipped", "skip", "pending":
		return LiveSkipped
	case "not_run", "notrun", "not-run":
		return LiveNotRun
	default:
		return LiveUnknown
	}
}

// NormalizeDeclaredStatus standardizes raw status strings to DeclaredStatus.
func NormalizeDeclaredStatus(s string) DeclaredStatus {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "pending", "":
		return StatusPending
	case "planned":
		return StatusPlanned
	case "in_progress", "inprogress", "in-progress", "wip":
		return StatusInProgress
	case "complete", "completed", "done", "implemented":
		return StatusComplete
	case "not_implemented", "notimplemented", "not-implemented":
		return StatusNotImplemented
	default:
		return StatusPending
	}
}

// NormalizeValidationStatus standardizes raw status strings to ValidationStatus.
func NormalizeValidationStatus(s string) ValidationStatus {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "not_implemented", "notimplemented", "not-implemented", "":
		return ValStatusNotImplemented
	case "planned":
		return ValStatusPlanned
	case "implemented", "passing", "passed":
		return ValStatusImplemented
	case "failing", "failed":
		return ValStatusFailing
	default:
		return ValStatusNotImplemented
	}
}

// NormalizeCriticality standardizes criticality strings.
func NormalizeCriticality(s string) Criticality {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "P0", "CRITICAL", "HIGH":
		return CriticalityP0
	case "P1", "MAJOR", "MEDIUM":
		return CriticalityP1
	case "P2", "MINOR", "LOW":
		return CriticalityP2
	default:
		return CriticalityP2
	}
}

// NormalizeValidationType standardizes validation type strings.
func NormalizeValidationType(s string) ValidationType {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "test", "unit", "integration":
		return ValTypeTest
	case "automation", "workflow", "playbook":
		return ValTypeAutomation
	case "manual":
		return ValTypeManual
	case "lighthouse", "perf", "performance":
		return ValTypeLighthouse
	default:
		return ValTypeTest
	}
}

// DeriveValidationStatus converts LiveStatus to ValidationStatus.
func DeriveValidationStatus(live LiveStatus) ValidationStatus {
	switch live {
	case LivePassed:
		return ValStatusImplemented
	case LiveFailed:
		return ValStatusFailing
	case LiveSkipped, LiveNotRun:
		return ValStatusPlanned
	default:
		return ValStatusNotImplemented
	}
}

// DeriveRequirementStatus updates a declared status based on validation results.
// Returns the new status if validations warrant a change, otherwise current.
func DeriveRequirementStatus(current DeclaredStatus, validations []ValidationStatus) DeclaredStatus {
	if len(validations) == 0 {
		return current
	}

	allImplemented := true
	anyFailing := false
	anyPlanned := false

	for _, vs := range validations {
		switch vs {
		case ValStatusFailing:
			anyFailing = true
			allImplemented = false
		case ValStatusPlanned, ValStatusNotImplemented:
			allImplemented = false
			if vs == ValStatusPlanned {
				anyPlanned = true
			}
		}
	}

	// If any validation is failing, requirement cannot be complete
	if anyFailing {
		if current == StatusComplete {
			return StatusInProgress
		}
		return current
	}

	// All validations passing
	if allImplemented {
		if current == StatusInProgress || current == StatusPlanned {
			return StatusComplete
		}
	}

	// Some planned validations
	if anyPlanned && current == StatusPending {
		return StatusPlanned
	}

	return current
}

// DeriveLiveRollup aggregates multiple LiveStatus values into one.
// Follows status priority: failed > skipped > passed > not_run > unknown.
func DeriveLiveRollup(statuses []LiveStatus) LiveStatus {
	if len(statuses) == 0 {
		return LiveUnknown
	}

	maxPriority := -1
	result := LiveUnknown

	for _, s := range statuses {
		priority := statusPriority[s]
		if priority > maxPriority {
			maxPriority = priority
			result = s
		}
	}

	return result
}

// DeriveDeclaredRollup aggregates multiple DeclaredStatus values for parent requirements.
func DeriveDeclaredRollup(statuses []DeclaredStatus) DeclaredStatus {
	if len(statuses) == 0 {
		return StatusPending
	}

	// Priority: not_implemented > pending > planned > in_progress > complete
	// A parent is only complete if ALL children are complete
	allComplete := true
	anyInProgress := false
	anyPlanned := false
	anyPending := false
	anyNotImplemented := false

	for _, s := range statuses {
		switch s {
		case StatusComplete:
			// continue
		case StatusInProgress:
			allComplete = false
			anyInProgress = true
		case StatusPlanned:
			allComplete = false
			anyPlanned = true
		case StatusPending:
			allComplete = false
			anyPending = true
		case StatusNotImplemented:
			allComplete = false
			anyNotImplemented = true
		}
	}

	if allComplete {
		return StatusComplete
	}
	if anyNotImplemented {
		return StatusNotImplemented
	}
	if anyPending {
		return StatusPending
	}
	if anyPlanned {
		return StatusPlanned
	}
	if anyInProgress {
		return StatusInProgress
	}

	return StatusPending
}

// IsTerminalStatus returns true if the status represents a final state.
func IsTerminalStatus(s DeclaredStatus) bool {
	return s == StatusComplete || s == StatusNotImplemented
}

// IsCritical returns true for P0 or P1 criticality.
func IsCritical(c Criticality) bool {
	return c == CriticalityP0 || c == CriticalityP1
}
