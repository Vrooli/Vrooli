package initiativereview

import (
	"fmt"
	"strings"
)

// Verdict is the user's terminal decision for an initiative review round.
type Verdict string

const (
	VerdictAccept   Verdict = "accept"
	VerdictFail     Verdict = "fail"
	VerdictFollowup Verdict = "followup"
)

// NormalizeVerdict lower-cases and trims the input, returning an error when
// the value is not one of the three supported verdicts.
func NormalizeVerdict(raw string) (Verdict, error) {
	v := Verdict(strings.ToLower(strings.TrimSpace(raw)))
	switch v {
	case VerdictAccept, VerdictFail, VerdictFollowup:
		return v, nil
	}
	return "", fmt.Errorf("invalid verdict %q: must be accept, fail, or followup", raw)
}

// TargetStatus maps a verdict to the initiative status the user is requesting.
// Kept here (not in initiatives) so the verdict→status rule is localized to
// this package alongside the rest of the review-decide flow.
func (v Verdict) TargetStatus() string {
	switch v {
	case VerdictAccept:
		return "completed"
	case VerdictFail:
		return "failed"
	case VerdictFollowup:
		return "needs_followup"
	}
	return ""
}

// DecisionRecord is the audit entry persisted per initiative review decision.
type DecisionRecord struct {
	Verdict     string `json:"verdict"`
	Status      string `json:"status"`
	Rationale   string `json:"rationale,omitempty"`
	DecidedBy   string `json:"decided_by,omitempty"`
	DecidedAt   string `json:"decided_at"`
	PriorStatus string `json:"prior_status"`
	Round       int    `json:"round,omitempty"`
}

// DecideRequest is the JSON body accepted by the decide endpoint.
type DecideRequest struct {
	Verdict   string `json:"verdict"`
	Rationale string `json:"rationale,omitempty"`
	DecidedBy string `json:"decided_by,omitempty"`
}

// DecideResponse echoes the outcome of a decide call.
type DecideResponse struct {
	Initiative string `json:"initiative"`
	Verdict    string `json:"verdict"`
	Status     string `json:"status"`
	Rationale  string `json:"rationale,omitempty"`
	DecidedAt  string `json:"decided_at"`
}

// TriggerResult reports whether StartReview actually started a new round, or
// declined (with a reason). Returned so callers that only want the side effect
// don't have to probe status afterwards.
type TriggerResult struct {
	Started bool   `json:"started"`
	Reason  string `json:"reason,omitempty"`
	Round   int    `json:"round,omitempty"`
	RunID   string `json:"run_id,omitempty"`
}
