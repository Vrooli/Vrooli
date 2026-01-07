// Package httpserver provides HTTP API types and handlers.
// This file contains Data Transfer Objects (DTOs) for the agent API endpoints.
// Agent execution is managed by agent-manager service.
package httpserver

import (
	"time"
)

// AgentStatus represents the status of an agent.
type AgentStatus string

// Agent status constants
const (
	AgentStatusPending   AgentStatus = "pending"
	AgentStatusRunning   AgentStatus = "running"
	AgentStatusCompleted AgentStatus = "completed"
	AgentStatusFailed    AgentStatus = "failed"
	AgentStatusStopped   AgentStatus = "stopped"
	AgentStatusTimeout   AgentStatus = "timeout"
)

// ActiveAgent represents a spawned agent in API responses.
// This is an API DTO that provides the JSON structure for agent endpoints.
// Agent state is managed by agent-manager.
type ActiveAgent struct {
	ID           string      `json:"id"`
	RunID        string      `json:"runId,omitempty"`
	SessionID    string      `json:"sessionId,omitempty"`
	Scenario     string      `json:"scenario,omitempty"`
	Scope        []string    `json:"scope,omitempty"`
	Phases       []string    `json:"phases,omitempty"`
	Model        string      `json:"model,omitempty"`
	Status       AgentStatus `json:"status"`
	StartedAt    time.Time   `json:"startedAt,omitempty"`
	CompletedAt  time.Time   `json:"completedAt,omitempty"`
	Output       string      `json:"output,omitempty"`
	Error        string      `json:"error,omitempty"`
	TokensUsed   int32       `json:"tokensUsed,omitempty"`
	CostEstimate float64     `json:"costEstimate,omitempty"`
}
