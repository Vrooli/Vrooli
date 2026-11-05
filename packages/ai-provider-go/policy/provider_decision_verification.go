package policy

import (
	"errors"
	"strings"
	"time"
)

const ProviderDecisionEventType = "ai_gateway.provider_decision.decision_requested.v1"

// ProviderDecisionReceipt is the read-only Hub response used to verify the
// exact decision attached to a provider rule or agreement.
type ProviderDecisionReceipt struct {
	Source                   string `json:"source"`
	EventType                string `json:"event_type"`
	Requester                string `json:"requester"`
	EventID                  string `json:"event_id"`
	CanonicalTermsSHA256     string `json:"canonical_terms_sha256"`
	Scope                    string `json:"scope"`
	State                    string `json:"state"`
	Answer                   string `json:"answer"`
	AnsweredByIdentityKind   string `json:"answered_by_identity_kind"`
	AnsweredByIdentitySource string `json:"answered_by_identity_source"`
	CreatedAt                string `json:"created_at"`
	AnsweredAt               string `json:"answered_at"`
	Deadline                 string `json:"deadline"`
}

// VerifyProviderDecisionReceipt binds a Hub read to the exact immutable
// request and accepts only a timely explicit approval by a verified human.
func VerifyProviderDecisionReceipt(expected ProviderDecisionReceipt, actual ProviderDecisionReceipt, now time.Time) error {
	expectedSource := strings.TrimSpace(expected.Source)
	expectedEvent := strings.TrimSpace(expected.EventType)
	expectedRequester := strings.TrimSpace(expected.Requester)
	expectedDigest := strings.ToLower(strings.TrimSpace(expected.CanonicalTermsSHA256))
	expectedScope := strings.TrimSpace(expected.Scope)
	if expectedSource != "ai-gateway" || expectedEvent != ProviderDecisionEventType || expectedRequester == "" || !validProviderDecisionDigest(expectedDigest) || (expectedScope != ProviderDecisionScopeRule && expectedScope != ProviderDecisionScopeAgreement) {
		return errors.New("provider decision expectation is incomplete")
	}
	if actual.Source != expectedSource || actual.EventType != expectedEvent || actual.Requester != expectedRequester || actual.EventID != expectedRequester || actual.CanonicalTermsSHA256 != expectedDigest || actual.Scope != expectedScope {
		return errors.New("provider decision identity does not match the expected request")
	}
	if actual.State != "answered" || actual.Answer != "approve" || actual.AnsweredByIdentityKind != "verified_human" || strings.TrimSpace(actual.AnsweredByIdentitySource) == "" {
		return errors.New("provider decision is not an explicit verified-human approval")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, actual.CreatedAt)
	if err != nil {
		return errors.New("provider decision creation time is invalid")
	}
	answeredAt, err := time.Parse(time.RFC3339Nano, actual.AnsweredAt)
	if err != nil {
		return errors.New("provider decision answer time is invalid")
	}
	deadline, err := time.Parse(time.RFC3339Nano, actual.Deadline)
	if err != nil {
		return errors.New("provider decision deadline is invalid")
	}
	now = now.UTC()
	if now.IsZero() || createdAt.After(answeredAt) || answeredAt.After(deadline) || now.Before(createdAt) || now.After(deadline) {
		return errors.New("provider decision falls outside its verified time window")
	}
	return nil
}

func validProviderDecisionDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
