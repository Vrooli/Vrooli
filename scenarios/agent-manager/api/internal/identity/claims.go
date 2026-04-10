package identity

import "github.com/google/uuid"

// Claims represents the identity claims embedded in an agent token.
type Claims struct {
	RunID      uuid.UUID         `json:"run_id"`
	TaskID     uuid.UUID         `json:"task_id"`
	ProfileKey string            `json:"profile_key"`
	ScopePath  string            `json:"scope_path"`
	IssuedAt   int64             `json:"iat"`
	ExpiresAt  int64             `json:"exp"`
	Meta       map[string]string `json:"meta"`
}
