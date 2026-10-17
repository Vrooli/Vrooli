package execution

import (
	"device-control/internal/evidence"
	"device-control/internal/sessions"
)

type Step struct {
	ID                   string         `json:"id"`
	Kind                 string         `json:"kind"`
	RequiredCapabilities []string       `json:"required_capabilities"`
	Target               string         `json:"target"`
	TimeoutMS            int64          `json:"timeout_ms"`
	Arguments            map[string]any `json:"arguments"`
	Preconditions        []Condition    `json:"preconditions,omitempty"`
	Postconditions       []Condition    `json:"postconditions,omitempty"`
	ObservationRequired  bool           `json:"observation_required,omitempty"`
	RetryBudget          int            `json:"retry_budget,omitempty"`
	IdempotencyKey       string         `json:"idempotency_key,omitempty"`
}

type Condition struct {
	Kind     string `json:"kind"`
	Target   string `json:"target,omitempty"`
	Expected any    `json:"expected,omitempty"`
}

type Flow struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Steps                  []Step `json:"steps"`
	Transport              string `json:"transport,omitempty"`
	RequireUnlocked        bool   `json:"require_unlocked,omitempty"`
	AuthProfileID          string `json:"auth_profile_id,omitempty"`
	AllowUnredactedCapture bool   `json:"allow_unredacted_capture"`
	SuppressActuation      bool   `json:"suppress_actuation,omitempty"`
	MaxDurationMS          int64  `json:"max_duration_ms,omitempty"`
	RetryBudget            int    `json:"retry_budget,omitempty"`
	ApplicationID          string `json:"application_id,omitempty"`
	ApplicationRevision    string `json:"application_revision,omitempty"`
}
type GapReport struct {
	Runnable bool     `json:"runnable"`
	Gaps     []string `json:"gaps"`
	Warnings []string `json:"warnings"`
}
type Chapter struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Disposition  string   `json:"disposition"`
	Message      string   `json:"message"`
	EvidenceIDs  []string `json:"evidence_ids,omitempty"`
	FailureClass string   `json:"failure_class,omitempty"`
	Attempts     int      `json:"attempts,omitempty"`
}
type Resolution struct {
	Target     string  `json:"target"`
	Rung       string  `json:"rung"`
	Confidence float64 `json:"confidence"`
}
type RunResult struct {
	RunID            string                      `json:"run_id"`
	Disposition      string                      `json:"disposition"`
	Chapters         []Chapter                   `json:"chapters"`
	Resolutions      []Resolution                `json:"resolutions"`
	Evidence         []evidence.Reference        `json:"evidence"`
	Restoration      []sessions.RestorationEvent `json:"restoration,omitempty"`
	Incomplete       bool                        `json:"incomplete,omitempty"`
	DisconnectReason string                      `json:"disconnect_reason,omitempty"`
	DisconnectStep   string                      `json:"disconnect_step,omitempty"`
	Binding          RunBinding                  `json:"binding"`
}

type RunBinding struct {
	DeviceID            string `json:"device_id"`
	Transport           string `json:"transport"`
	LeaseID             string `json:"lease_id,omitempty"`
	LeaseEpoch          uint64 `json:"lease_epoch,omitempty"`
	ApplicationID       string `json:"application_id,omitempty"`
	ApplicationRevision string `json:"application_revision,omitempty"`
}
