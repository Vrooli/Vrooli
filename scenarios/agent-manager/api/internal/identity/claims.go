package identity

import "github.com/google/uuid"

// Claims represents the identity claims embedded in an agent token.
type Claims struct {
	RunID  uuid.UUID `json:"run_id"`
	TaskID uuid.UUID `json:"task_id"`
	// Subject is the verified owner account that requested the run. It is
	// distinct from ProfileKey, which names the agent configuration.
	Subject string `json:"subject,omitempty"`
	// Scopes is an explicit, attenuated capability list. An empty list means
	// no delegated scopes; authority is never inferred from its absence.
	Scopes     []string          `json:"scopes"`
	ProfileKey string            `json:"profile_key"`
	ScopePath  string            `json:"scope_path"`
	IssuedAt   int64             `json:"iat"`
	ExpiresAt  int64             `json:"exp"`
	Meta       map[string]string `json:"meta"`
}
