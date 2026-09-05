package scenarios

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

type HealthSource interface {
	Snapshot(context.Context, string) ScenarioHealthSnapshot
}

// HealthEvidenceState describes whether a provider-owned health projection can
// safely support remediation. It is intentionally not a health verdict.
type HealthEvidenceState string

const (
	HealthEvidenceFresh       HealthEvidenceState = "fresh"
	HealthEvidenceStale       HealthEvidenceState = "stale"
	HealthEvidenceDegraded    HealthEvidenceState = "degraded"
	HealthEvidenceUnavailable HealthEvidenceState = "unavailable"
	HealthEvidenceNone        HealthEvidenceState = "no_evidence"
)

// ScenarioHealthSnapshot is the Swarm boundary for Test Genie evidence. The
// adapter copies supported provider fields into this projection; consumers must
// not infer ordering, severity, or maturity beyond what is presented here.
type ScenarioHealthSnapshot struct {
	EvidenceState HealthEvidenceState        `json:"evidence_state"`
	Reason        string                     `json:"reason,omitempty"`
	SourceRunID   string                     `json:"source_run_id,omitempty"`
	ObservedAt    string                     `json:"observed_at,omitempty"`
	Freshness     string                     `json:"freshness,omitempty"`
	Verdict       string                     `json:"verdict,omitempty"`
	Phases        []ScenarioHealthPhase      `json:"phases,omitempty"`
	Remediation   []ScenarioRemediationState `json:"remediation,omitempty"`
}

type ScenarioHealthPhase struct {
	Phase                   string   `json:"phase"`
	Label                   string   `json:"label,omitempty"`
	Verdict                 string   `json:"verdict,omitempty"`
	CurrentRung             string   `json:"current_rung,omitempty"`
	NextRung                string   `json:"next_rung,omitempty"`
	PriorityCapabilityID    string   `json:"priority_capability_id,omitempty"`
	PriorityCapabilityLabel string   `json:"priority_capability_label,omitempty"`
	BlockingCodes           []string `json:"blocking_codes,omitempty"`
	RemediationTopics       []string `json:"remediation_topics,omitempty"`
}

type ScenarioRemediationState struct {
	Fingerprint string `json:"fingerprint"`
	State       string `json:"state"`
	WorkRef     string `json:"work_ref,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// RemediationTarget is the canonical routing identity for one bounded phase
// remediation. It never accepts copied findings, scores, or run counts.
type RemediationTarget struct {
	Scenario      string
	ProviderPhase string
	CapabilityID  string
}

var (
	ErrRemediationScenarioRequired   = errors.New("scenario remediation target requires a scenario")
	ErrRemediationPhaseRequired      = errors.New("scenario remediation target requires a provider phase")
	ErrRemediationCapabilityRequired = errors.New("scenario remediation target requires a capability target")
)

func (t RemediationTarget) Normalize() (RemediationTarget, error) {
	t.Scenario = strings.TrimSpace(t.Scenario)
	t.ProviderPhase = strings.TrimSpace(t.ProviderPhase)
	t.CapabilityID = strings.TrimSpace(t.CapabilityID)
	if t.Scenario == "" {
		return RemediationTarget{}, ErrRemediationScenarioRequired
	}
	if t.ProviderPhase == "" {
		return RemediationTarget{}, ErrRemediationPhaseRequired
	}
	if t.CapabilityID == "" {
		return RemediationTarget{}, ErrRemediationCapabilityRequired
	}
	return t, nil
}

// Fingerprint is stable for equivalent target inputs across manual and
// automatic callers. It is deliberately independent of transient run IDs.
func (t RemediationTarget) Fingerprint() (string, error) {
	normalized, err := t.Normalize()
	if err != nil {
		return "", err
	}
	identity := strings.Join([]string{
		"scenario-remediation/v1",
		strings.ToLower(normalized.Scenario),
		strings.ToLower(normalized.ProviderPhase),
		strings.ToLower(normalized.CapabilityID),
	}, "\x00")
	digest := sha256.Sum256([]byte(identity))
	return "srh:" + hex.EncodeToString(digest[:]), nil
}

func (s ScenarioHealthSnapshot) IsActionable() bool {
	return s.EvidenceState == HealthEvidenceFresh
}
