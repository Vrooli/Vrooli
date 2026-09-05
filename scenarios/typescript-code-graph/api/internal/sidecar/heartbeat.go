package sidecar

import "time"

// Heartbeat defaults per plan §7 Phase 3.
const (
	defaultHeartbeatInterval = 10 * time.Second
	defaultHeartbeatTimeout  = 5 * time.Second
)

// heartbeatRequest / heartbeatResponse mirror plan §8.4 wire shapes.
type heartbeatRequest struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
}

//nolint:unused // documented §8.4 response wire shape; intentionally retained for protocol reference.
type heartbeatResponse struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
}
