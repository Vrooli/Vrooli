package support

import (
	"encoding/json"
	"time"
)

// Event mirrors the shape returned by /api/v1/events and /api/v1/events/{id}.
type Event struct {
	ID               string                 `json:"id"`
	UserID           string                 `json:"user_id,omitempty"`
	Title            string                 `json:"title"`
	Description      string                 `json:"description,omitempty"`
	StartTime        string                 `json:"start_time,omitempty"`
	EndTime          string                 `json:"end_time,omitempty"`
	Timezone         string                 `json:"timezone,omitempty"`
	Location         string                 `json:"location,omitempty"`
	EventType        string                 `json:"event_type,omitempty"`
	Status           string                 `json:"status,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	AutomationConfig map[string]interface{} `json:"automation_config,omitempty"`
	CreatedAt        *time.Time             `json:"created_at,omitempty"`
	UpdatedAt        *time.Time             `json:"updated_at,omitempty"`
}

// EventListResponse is the envelope returned by /api/v1/events (list).
type EventListResponse struct {
	Events     []Event `json:"events"`
	TotalCount int     `json:"total_count"`
	HasMore    bool    `json:"has_more"`
	Timezone   string  `json:"timezone,omitempty"`
}

// EventCreateResponse is the shape returned by POST /api/v1/events.
type EventCreateResponse struct {
	Success            bool                   `json:"success"`
	Event              map[string]interface{} `json:"event"`
	RemindersScheduled int                    `json:"reminders_scheduled"`
}

// SuggestedAction describes one action proposed by the chat assistant.
type SuggestedAction struct {
	Action     string                 `json:"action"`
	Confidence float64                `json:"confidence"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ChatResponse mirrors the payload from POST /api/v1/schedule/chat.
type ChatResponse struct {
	Response             string                 `json:"response"`
	SuggestedActions     []SuggestedAction      `json:"suggested_actions"`
	RequiresConfirmation bool                   `json:"requires_confirmation"`
	Context              map[string]interface{} `json:"context,omitempty"`
}

// OAuthInitiateResponse is returned by GET /api/v1/external-sync/oauth/{provider}.
type OAuthInitiateResponse struct {
	AuthURL  string `json:"auth_url"`
	Provider string `json:"provider"`
	State    string `json:"state"`
}

// SyncConnection describes one external calendar connection.
type SyncConnection struct {
	Provider      string          `json:"provider"`
	Connected     bool            `json:"connected"`
	SyncEnabled   bool            `json:"sync_enabled"`
	LastSync      json.RawMessage `json:"last_sync,omitempty"`
	SyncDirection string          `json:"sync_direction,omitempty"`
}

// SyncStatusResponse is returned by GET /api/v1/external-sync/status.
type SyncStatusResponse struct {
	Connections []SyncConnection `json:"connections"`
}

// SyncResult describes one provider's sync outcome.
type SyncResult struct {
	Provider      string   `json:"provider"`
	Status        string   `json:"status"`
	EventsCreated int      `json:"events_created,omitempty"`
	EventsUpdated int      `json:"events_updated,omitempty"`
	EventsSynced  int      `json:"events_synced,omitempty"`
	Errors        []string `json:"errors,omitempty"`
}

// SyncRunResponse is returned by POST /api/v1/external-sync/sync.
type SyncRunResponse struct {
	Success bool         `json:"success"`
	Results []SyncResult `json:"results"`
}

// DisconnectResponse is returned by DELETE /api/v1/external-sync/disconnect/{provider}.
type DisconnectResponse struct {
	Success      bool   `json:"success"`
	Provider     string `json:"provider"`
	Disconnected bool   `json:"disconnected"`
}
