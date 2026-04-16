package support

import (
	"encoding/json"
)

// Session mirrors SessionResponse from the web-console API
// (/api/v1/sessions[/{id}]).
type Session struct {
	ID              string          `json:"id"`
	Shell           string          `json:"shell,omitempty"`
	CreatedAt       string          `json:"created_at,omitempty"`
	Cols            int             `json:"cols,omitempty"`
	Rows            int             `json:"rows,omitempty"`
	Backend         string          `json:"backend,omitempty"`
	SurvivesRestart bool            `json:"survives_restart,omitempty"`
	Busy            bool            `json:"busy,omitempty"`
	Recovered       bool            `json:"recovered,omitempty"`
	Policy          json.RawMessage `json:"policy,omitempty"`
}

// PolicyResponse mirrors the /sessions/{id}/policy shape.
type PolicyResponse struct {
	SessionID string          `json:"session_id"`
	Policy    json.RawMessage `json:"policy,omitempty"`
	ExpiresAt string          `json:"expires_at,omitempty"`
	TTL       *float64        `json:"ttl_seconds,omitempty"`
}

// EventRecord is one entry from /api/v1/events.
type EventRecord struct {
	ID         string          `json:"id,omitempty"`
	Type       string          `json:"type,omitempty"`
	SessionID  string          `json:"session_id,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
	OccurredAt string          `json:"occurred_at,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// ShortcutProfile mirrors a shortcut profile entry.
type ShortcutProfile struct {
	ID          string          `json:"id"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Enabled     bool            `json:"enabled,omitempty"`
	Layout      string          `json:"layout,omitempty"`
	OS          string          `json:"os,omitempty"`
	Bindings    json.RawMessage `json:"bindings,omitempty"`
	CreatedAt   string          `json:"created_at,omitempty"`
	UpdatedAt   string          `json:"updated_at,omitempty"`
}

// VoiceProfile summarises a speaker verification profile.
type VoiceProfile struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Speaker string `json:"speaker,omitempty"`
	Status  string `json:"status,omitempty"`
}
