// Package evidence is Swarm Manager's owner-neutral, durable evidence ledger.
// It records normalized source observations and links them to exactly one
// agent-session or operating-mode execution only after exhaustive ownership
// resolution. Producer domains retain authority for their own mutations.
package evidence

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type OwnerKind string

const (
	OwnerAgentSession           OwnerKind = "agent_session"
	OwnerOperatingModeExecution OwnerKind = "operating_mode_execution"
)

type Owner struct {
	Kind  OwnerKind `json:"kind"`
	ID    string    `json:"id"`
	Round int       `json:"round,omitempty"`
}

func (o Owner) Validate() error {
	if o.Kind != OwnerAgentSession && o.Kind != OwnerOperatingModeExecution {
		return fmt.Errorf("evidence owner kind %q is invalid", o.Kind)
	}
	if strings.TrimSpace(o.ID) == "" {
		return fmt.Errorf("evidence owner id is required")
	}
	if o.Round < 0 || (o.Kind == OwnerAgentSession && o.Round != 0) {
		return fmt.Errorf("evidence owner round is invalid")
	}
	return nil
}

type Subject struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func (s Subject) Validate() error {
	if strings.TrimSpace(s.Kind) == "" || strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("evidence subject kind and id are required")
	}
	return nil
}

type Confidence string

const (
	ConfidenceAuthoritative Confidence = "authoritative"
	ConfidenceObserved      Confidence = "observed"
	ConfidenceReported      Confidence = "reported"
	ConfidenceOperator      Confidence = "operator_verified"
)

type Verification string

const (
	VerificationVerified    Verification = "verified"
	VerificationUnverified  Verification = "unverified"
	VerificationUnavailable Verification = "unavailable"
)

type Observation struct {
	SourceSystem  string            `json:"source_system"`
	SourceEventID string            `json:"source_event_id"`
	RunID         string            `json:"run_id"`
	Subject       Subject           `json:"subject"`
	Action        string            `json:"action"`
	Confidence    Confidence        `json:"confidence"`
	Verification  Verification      `json:"verification"`
	ContentDigest string            `json:"content_digest,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	ObservedAt    time.Time         `json:"observed_at"`
}

func (o Observation) Validate() error {
	if strings.TrimSpace(o.SourceSystem) == "" || strings.TrimSpace(o.SourceEventID) == "" {
		return fmt.Errorf("evidence source system and event id are required")
	}
	if strings.TrimSpace(o.RunID) == "" {
		return fmt.Errorf("evidence run id is required")
	}
	if err := o.Subject.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(o.Action) == "" {
		return fmt.Errorf("evidence action is required")
	}
	switch o.Confidence {
	case ConfidenceAuthoritative, ConfidenceObserved, ConfidenceReported, ConfidenceOperator:
	default:
		return fmt.Errorf("evidence confidence %q is invalid", o.Confidence)
	}
	switch o.Verification {
	case VerificationVerified, VerificationUnverified, VerificationUnavailable:
	default:
		return fmt.Errorf("evidence verification %q is invalid", o.Verification)
	}
	if len(o.Metadata) > 24 {
		return fmt.Errorf("evidence metadata has too many fields")
	}
	for key, value := range o.Metadata {
		if len(key) > 96 || len(value) > 512 {
			return fmt.Errorf("evidence metadata exceeds bounded field size")
		}
	}
	return nil
}

func (o Observation) normalized() Observation {
	o.SourceSystem = strings.TrimSpace(o.SourceSystem)
	o.SourceEventID = strings.TrimSpace(o.SourceEventID)
	o.RunID = strings.TrimSpace(o.RunID)
	o.Subject.Kind = strings.TrimSpace(o.Subject.Kind)
	o.Subject.ID = strings.TrimSpace(o.Subject.ID)
	o.Action = strings.TrimSpace(o.Action)
	o.ContentDigest = strings.TrimSpace(o.ContentDigest)
	if o.ObservedAt.IsZero() {
		o.ObservedAt = time.Now().UTC()
	} else {
		o.ObservedAt = o.ObservedAt.UTC()
	}
	return o
}

type OwnershipStatus string

const (
	OwnershipResolved    OwnershipStatus = "resolved"
	OwnershipUnresolved  OwnershipStatus = "unresolved"
	OwnershipAmbiguous   OwnershipStatus = "ambiguous"
	OwnershipUnavailable OwnershipStatus = "unavailable"
)

type IngestResult struct {
	ObservationID   int64           `json:"observation_id"`
	Owner           *Owner          `json:"owner,omitempty"`
	OwnershipStatus OwnershipStatus `json:"ownership_status"`
	Duplicate       bool            `json:"duplicate"`
}

// Record is the owner-facing projection of one immutable source observation.
// The fact remains owned by its producer; this record only represents its
// canonical evidence link and contains no raw transcript or token material.
type Record struct {
	Observation Observation `json:"observation"`
	Owner       Owner       `json:"owner"`
	LinkedAt    time.Time   `json:"linked_at"`
}

type Checkpoint struct {
	ProducerID string    `json:"producer_id"`
	RunID      string    `json:"run_id"`
	FactKind   string    `json:"fact_kind"`
	Cursor     string    `json:"cursor"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Watermark struct {
	ProducerID  string    `json:"producer_id"`
	RunID       string    `json:"run_id"`
	FactKind    string    `json:"fact_kind"`
	Coverage    string    `json:"coverage"`
	CompletedAt time.Time `json:"completed_at"`
}

// MigrationAudit is a durable parity receipt for a bounded compatibility
// migration. Source and projected digests are derived from stable owner/source
// identities, never artifact content, so the receipt proves coverage without
// duplicating sensitive metadata.
type MigrationAudit struct {
	MigrationKey    string    `json:"migration_key"`
	SourceCount     int       `json:"source_count"`
	ProjectedCount  int       `json:"projected_count"`
	SourceDigest    string    `json:"source_digest"`
	ProjectedDigest string    `json:"projected_digest"`
	CompletedAt     time.Time `json:"completed_at"`
}

func sortedMetadata(metadata map[string]string) []string {
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
