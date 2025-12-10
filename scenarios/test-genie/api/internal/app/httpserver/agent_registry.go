// Package httpserver provides HTTP API types and handlers.
// This file contains Data Transfer Objects (DTOs) for the agent API endpoints.
// The actual agent management logic is in the agents package (internal/agents/).
package httpserver

import (
	"time"

	"test-genie/internal/agents"
)

// AgentStatus is an alias for agents.AgentStatus to simplify JSON serialization.
// Use agents.AgentStatus as the source of truth for status values.
type AgentStatus = agents.AgentStatus

// ActiveAgent represents a spawned agent in API responses.
// This is an API DTO that provides the JSON structure for agent endpoints.
// The actual agent model is agents.SpawnedAgent in the agents package.
type ActiveAgent struct {
	ID          string      `json:"id"`
	SessionID   string      `json:"sessionId,omitempty"`
	Scenario    string      `json:"scenario"`
	Scope       []string    `json:"scope"`
	Phases      []string    `json:"phases,omitempty"`
	Model       string      `json:"model"`
	Status      AgentStatus `json:"status"`
	StartedAt   time.Time   `json:"startedAt"`
	CompletedAt *time.Time  `json:"completedAt,omitempty"`
	PromptHash  string      `json:"promptHash"`
	PromptIndex int         `json:"promptIndex"`
	PromptText  string      `json:"promptText,omitempty"`
	Output      string      `json:"output,omitempty"`
	Error       string      `json:"error,omitempty"`
	PID         int         `json:"pid,omitempty"`
	Hostname    string      `json:"hostname,omitempty"`
}
