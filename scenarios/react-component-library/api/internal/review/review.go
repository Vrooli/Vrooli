// Package review owns the server-side findings queue and its durable triage.
//
// Findings are a projection: gate evidence, component tests, provenance and
// capture state remain owned by their producing domains. This package records
// the projected rows, the human disposition history, and the accountable
// owner. The workbench renders Review; it must not rebuild findings from
// coverage rows.
package review

import (
	"context"
	"errors"
	"time"
)

// timestampLayout is fixed-width UTC so lexical ordering matches time order
// even when a timestamp has no fractional seconds.
const timestampLayout = "2006-01-02T15:04:05.000000000Z"

const (
	defaultListLimit = 100
	maxListLimit     = 500

	defaultProjectionTTL = 30 * time.Second
	dispositionIDBytes   = 8
)

// Severity values persisted in findings.severity.
const (
	SeverityBlocker = "blocker"
	SeverityHigh    = "high"
	SeverityMedium  = "medium"
	SeverityLow     = "low"
	SeverityInfo    = "info"
)

// Status values persisted in findings.status.
const (
	StatusOpen         = "open"
	StatusAcknowledged = "acknowledged"
	StatusAccepted     = "accepted"
	StatusFixed        = "fixed"
	StatusRetired      = "retired"
)

// Decision values persisted in finding_dispositions.decision.
const (
	DecisionAcknowledge = "acknowledge"
	DecisionAcceptRisk  = "accept_risk"
	DecisionFix         = "fix"
	DecisionRetire      = "retire"
)

// ErrNotFound means the requested finding does not exist (or was removed
// between a check and an update).
var ErrNotFound = errors.New("finding not found")

// ErrInvalid marks a caller-supplied value this domain cannot accept.
var ErrInvalid = errors.New("invalid finding request")

// Finding is one projected issue plus its durable triage state.
type Finding struct {
	ID           string
	SubjectKind  string
	SubjectID    string
	Gate         string
	Severity     string
	Status       string
	Message      string
	Remediation  string
	EvidenceJSON string
	DocsRef      string
	Owner        string
	CreatedAt    string
	Dispositions []Disposition
}

// Page is one window into the filtered queue plus the filtered total. Total
// counts every row the filter matches, not just the rows in this window, so a
// client can size its pager without a second request.
type Page struct {
	Findings []Finding
	Total    int
}

// Disposition is one explicit triage decision with its reason and actor.
type Disposition struct {
	ID        string
	FindingID string
	Decision  string
	Reason    string
	Actor     string
	At        string
}

// Filter narrows List. Empty fields mean no constraint.
type Filter struct {
	Status      string
	Severity    string
	SubjectKind string
	Query       string
}

// Summary is the Review queue rollup. Blockers and high count unresolved
// findings (status open or acknowledged); acknowledged is reported separately
// so a triaged item is not mistaken for an untouched one.
type Summary struct {
	Open         int
	Blockers     int
	High         int
	Acknowledged int
	Accepted     int
	Fixed        int
}

// Projector is the server-owned projection seam. A projector reads evidence
// owners and returns the findings they currently imply; it never writes
// findings or dispositions.
type Projector interface {
	Project(ctx context.Context) ([]Finding, error)
}

// ValidSeverity reports whether value is a persisted severity.
func ValidSeverity(value string) bool {
	switch value {
	case SeverityBlocker, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
		return true
	default:
		return false
	}
}

// ValidStatus reports whether value is a persisted finding status.
func ValidStatus(value string) bool {
	switch value {
	case StatusOpen, StatusAcknowledged, StatusAccepted, StatusFixed, StatusRetired:
		return true
	default:
		return false
	}
}

// ValidDecision reports whether value is a persisted disposition decision.
func ValidDecision(value string) bool {
	switch value {
	case DecisionAcknowledge, DecisionAcceptRisk, DecisionFix, DecisionRetire:
		return true
	default:
		return false
	}
}

// StatusForDecision maps a decision onto the finding status it establishes.
// "fix" deliberately reopens the finding: the decision records intent to
// repair, not completion, so the item stays in the queue until evidence clears
// it.
func StatusForDecision(decision string) string {
	switch decision {
	case DecisionAcknowledge:
		return StatusAcknowledged
	case DecisionAcceptRisk:
		return StatusAccepted
	case DecisionFix:
		return StatusOpen
	case DecisionRetire:
		return StatusRetired
	default:
		return ""
	}
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}
