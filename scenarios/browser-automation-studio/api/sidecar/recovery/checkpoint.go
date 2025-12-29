package recovery

import (
	"encoding/json"
	"time"
)

// SessionCheckpoint represents a saved state of a recording session.
// This is stored in the database and can be used to resume after a crash.
type SessionCheckpoint struct {
	// ID is the unique identifier for this checkpoint.
	ID string `json:"id" db:"id"`

	// SessionID is the recording session this checkpoint belongs to.
	SessionID string `json:"sessionId" db:"session_id"`

	// WorkflowID is the workflow being recorded, if any.
	WorkflowID string `json:"workflowId,omitempty" db:"workflow_id"`

	// RecordedActions are the actions recorded so far.
	RecordedActions []RecordedAction `json:"recordedActions"`

	// CurrentURL is the current page URL at checkpoint time.
	CurrentURL string `json:"currentUrl" db:"current_url"`

	// BrowserConfig is the browser configuration for the session.
	BrowserConfig BrowserConfig `json:"browserConfig"`

	// CreatedAt is when the checkpoint was first created.
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// UpdatedAt is when the checkpoint was last updated.
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// RecordedAction represents a single recorded user action.
type RecordedAction struct {
	// Type is the action type (e.g., "click", "type", "navigate").
	Type string `json:"type"`

	// Selector is the CSS selector for the target element.
	Selector string `json:"selector,omitempty"`

	// Value is the value for the action (e.g., text for type, URL for navigate).
	Value string `json:"value,omitempty"`

	// URL is the page URL where the action occurred.
	URL string `json:"url,omitempty"`

	// Timestamp is when the action was recorded.
	Timestamp time.Time `json:"timestamp"`

	// Metadata is additional action-specific data.
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// BrowserConfig holds browser configuration for a session.
type BrowserConfig struct {
	// ViewportWidth is the browser viewport width.
	ViewportWidth int `json:"viewportWidth"`

	// ViewportHeight is the browser viewport height.
	ViewportHeight int `json:"viewportHeight"`

	// UserAgent is the custom user agent, if set.
	UserAgent string `json:"userAgent,omitempty"`
}

// ActionCount returns the number of recorded actions.
func (c *SessionCheckpoint) ActionCount() int {
	return len(c.RecordedActions)
}

// IsEmpty returns true if no actions have been recorded.
func (c *SessionCheckpoint) IsEmpty() bool {
	return len(c.RecordedActions) == 0
}

// LastAction returns the most recent action, or nil if empty.
func (c *SessionCheckpoint) LastAction() *RecordedAction {
	if len(c.RecordedActions) == 0 {
		return nil
	}
	return &c.RecordedActions[len(c.RecordedActions)-1]
}

// MarshalActions serializes the recorded actions to JSON.
func (c *SessionCheckpoint) MarshalActions() ([]byte, error) {
	return json.Marshal(c.RecordedActions)
}

// UnmarshalActions deserializes recorded actions from JSON.
func (c *SessionCheckpoint) UnmarshalActions(data []byte) error {
	return json.Unmarshal(data, &c.RecordedActions)
}

// MarshalBrowserConfig serializes the browser config to JSON.
func (c *SessionCheckpoint) MarshalBrowserConfig() ([]byte, error) {
	return json.Marshal(c.BrowserConfig)
}

// UnmarshalBrowserConfig deserializes browser config from JSON.
func (c *SessionCheckpoint) UnmarshalBrowserConfig(data []byte) error {
	return json.Unmarshal(data, &c.BrowserConfig)
}
