package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// PlanAcceptanceContract is the common authored projection used when backlog
// records approval and when execution checks it. Keep lifecycle metadata out
// of this identity; every field here changes the operator's work contract.
type PlanAcceptanceContract struct {
	Kind              string                   `json:"kind"`
	Name              string                   `json:"name"`
	Title             string                   `json:"title"`
	Description       string                   `json:"description"`
	AcceptanceAllow   []string                 `json:"acceptance_allow,omitempty"`
	AcceptanceDeny    []string                 `json:"acceptance_deny,omitempty"`
	Creates           []string                 `json:"creates,omitempty"`
	PlanRef           *PlanAcceptanceReference `json:"plan_ref,omitempty"`
	ExecutionStrategy string                   `json:"execution_strategy,omitempty"`
	ExecutionLimits   *ExecutionLimits         `json:"execution_limits,omitempty"`
	Continuation      string                   `json:"continuation,omitempty"`
	ScopePolicy       string                   `json:"scope_policy,omitempty"`
}

// Retain empty reference fields consistently. Backlog's approval projection
// has always retained them, while an execution wire struct may omit them.
type PlanAcceptanceReference struct {
	Provider string `json:"provider"`
	PlanID   string `json:"plan_id"`
	Slug     string `json:"slug"`
	Role     string `json:"role"`
}

func (contract PlanAcceptanceContract) Digest() string {
	raw, _ := json.Marshal(contract)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
