// Package domain provides core domain types for browser-automation-studio.
package domain

import (
	"encoding/json"

	"github.com/google/uuid"
)

// TimelineEntry is a union type for the recording timeline.
// Each entry represents either a user action or a page lifecycle event,
// ordered chronologically to form a complete recording history.
type TimelineEntry struct {
	// Type distinguishes proto action entries from the separate lifecycle events.
	Type string `json:"type"`

	// Entry is the generated proto JSON for an action. Lifecycle events remain
	// separate because the stable TimelineEntry proto intentionally models actions.
	Entry     json.RawMessage `json:"entry,omitempty"`
	PageID    uuid.UUID       `json:"pageId"`
	PageEvent *PageEvent      `json:"pageEvent,omitempty"`
}

// TimelineResponse is the response for getting the unified timeline.
type TimelineResponse struct {
	Entries      []TimelineEntry `json:"entries"`
	HasMore      bool            `json:"hasMore"`
	TotalEntries int             `json:"totalEntries"`
}
