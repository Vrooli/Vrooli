// Package persistence provides data access for the unified recording service.
package persistence

import (
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/domain"
)

// TimelineEntryType identifies the type of entry in the unified timeline.
type TimelineEntryType string

const (
	// TimelineEntryTypeAction indicates a user action entry.
	TimelineEntryTypeAction TimelineEntryType = "action"
	// TimelineEntryTypePageEvent indicates a page lifecycle event entry.
	TimelineEntryTypePageEvent TimelineEntryType = "page_event"
)

// UnifiedTimelineEntry represents a single entry in the recording timeline.
// It can be either a recorded action or a page lifecycle event.
type UnifiedTimelineEntry struct {
	// ID uniquely identifies this timeline entry.
	ID uuid.UUID `json:"id"`

	// Type indicates whether this is an action or page event.
	Type TimelineEntryType `json:"type"`

	// Timestamp is when this entry occurred.
	Timestamp time.Time `json:"timestamp"`

	// SessionID links this entry to its recording session.
	SessionID string `json:"sessionId"`

	// PageID identifies which page this entry belongs to.
	PageID uuid.UUID `json:"pageId"`

	// Sequence is the ordering number within the session.
	Sequence int `json:"sequence"`

	// Action contains action details (set when Type == TimelineEntryTypeAction).
	Action *domain.RecordingAction `json:"action,omitempty"`

	// PageEvent contains page event details (set when Type == TimelineEntryTypePageEvent).
	PageEvent *domain.PageEvent `json:"pageEvent,omitempty"`
}

// TimelineQuery specifies criteria for querying the timeline.
type TimelineQuery struct {
	// SessionID is required - identifies which session to query.
	SessionID string

	// PageID optionally filters to a specific page.
	PageID *uuid.UUID

	// Since optionally filters to entries after this timestamp.
	Since *time.Time

	// EntryTypes optionally filters to specific entry types.
	EntryTypes []TimelineEntryType

	// Limit caps the number of results (default 100, max 1000).
	Limit int

	// Offset skips the first N results for pagination.
	Offset int
}

// ApplyDefaults sets default values for unset query fields.
func (q *TimelineQuery) ApplyDefaults() {
	if q.Limit <= 0 {
		q.Limit = 100
	}
	if q.Limit > 1000 {
		q.Limit = 1000
	}
}

// TimelineResponse contains the result of a timeline query.
type TimelineResponse struct {
	// Entries contains the timeline entries matching the query.
	Entries []UnifiedTimelineEntry `json:"entries"`

	// HasMore indicates if there are more entries beyond the limit.
	HasMore bool `json:"hasMore"`

	// TotalCount is the total number of entries in the session.
	TotalCount int `json:"totalCount"`
}
