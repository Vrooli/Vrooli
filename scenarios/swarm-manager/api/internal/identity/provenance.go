// Package identity provides middleware and helpers for extracting agent
// identity provenance from incoming HTTP requests. Provenance flows through
// request context and is consumed by downstream handlers for attribution.
package identity

import (
	"context"
	"strings"
)

// ProvenanceType distinguishes operator from agent attribution.
const (
	TypeOperator = "operator"
	TypeAgent    = "agent"
)

// Provenance identifies who initiated a request.
type Provenance struct {
	Type        string `json:"type"`
	RunID       string `json:"run_id,omitempty"`
	TaskID      string `json:"task_id,omitempty"`
	ProfileKey  string `json:"profile_key,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	SessionKind string `json:"session_kind,omitempty"`
	Source      string `json:"source,omitempty"`
}

// IsAgent returns true if this provenance represents an agent identity.
func (p Provenance) IsAgent() bool {
	return p.Type == TypeAgent
}

// WithSession returns provenance enriched with the owning Swarm Manager
// session. Session data is derived server-side from the verified run ID.
func (p Provenance) WithSession(ref SessionReference) Provenance {
	p.SessionID = strings.TrimSpace(ref.SessionID)
	p.SessionKind = strings.TrimSpace(ref.SessionKind)
	p.Source = strings.TrimSpace(ref.Source)
	return p
}

// FormatStartedBy returns a string suitable for the execution started_by field.
// Operator: "operator", Agent: "agent:<profile_key>/<run_id>".
func (p Provenance) FormatStartedBy() string {
	if p.IsAgent() {
		return "agent:" + p.ProfileKey + "/" + p.RunID
	}
	return TypeOperator
}

type provenanceKey struct{}

// NewContext stores provenance in the context.
func NewContext(ctx context.Context, p Provenance) context.Context {
	return context.WithValue(ctx, provenanceKey{}, p)
}

// FromContext extracts provenance from the context.
// Returns operator provenance if none is set.
func FromContext(ctx context.Context) Provenance {
	p, ok := ctx.Value(provenanceKey{}).(Provenance)
	if !ok {
		return Provenance{Type: TypeOperator}
	}
	return p
}
