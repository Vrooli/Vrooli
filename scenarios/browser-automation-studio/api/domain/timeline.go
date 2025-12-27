// Package domain provides core domain types for browser-automation-studio.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// TimelineType represents the type of entry in the recording timeline.
type TimelineType string

const (
	TimelineTypeAction    TimelineType = "action"
	TimelineTypePageEvent TimelineType = "page_event"
)

// TimelineEntry is a union type for the recording timeline.
// Each entry represents either a user action or a page lifecycle event,
// ordered chronologically to form a complete recording history.
type TimelineEntry struct {
	ID        uuid.UUID    `json:"id"`
	Type      TimelineType `json:"type"` // "action" | "page_event"
	Timestamp time.Time    `json:"timestamp"`
	PageID    uuid.UUID    `json:"pageId"`

	// One of these will be set based on Type
	Action    *RecordedActionEntry `json:"action,omitempty"`
	PageEvent *PageEvent           `json:"pageEvent,omitempty"`
}

// RecordedActionEntry is a simplified action representation for timeline storage.
// It contains the essential fields from the full RecordedAction for timeline display.
type RecordedActionEntry struct {
	ID          string                 `json:"id"`
	ActionType  string                 `json:"actionType"`
	URL         string                 `json:"url,omitempty"`
	SequenceNum int                    `json:"sequenceNum"`
	Timestamp   string                 `json:"timestamp"`
	Selector    *SelectorInfo          `json:"selector,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Confidence  float64                `json:"confidence"`
	PageTitle   string                 `json:"pageTitle,omitempty"`
}

// SelectorInfo contains selector information for timeline entries.
type SelectorInfo struct {
	Primary string `json:"primary"`
}

// TimelineResponse is the response for getting the unified timeline.
type TimelineResponse struct {
	Entries      []TimelineEntry `json:"entries"`
	HasMore      bool            `json:"hasMore"`
	TotalEntries int             `json:"totalEntries"`
}
