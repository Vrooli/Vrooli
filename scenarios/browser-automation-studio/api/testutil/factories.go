// Package testutil provides testing utilities and domain object factories.
package testutil

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/domain"
	recordingpersistence "github.com/vrooli/browser-automation-studio/services/recording/persistence"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// RecordingSessionOption modifies a RecordingSession during construction.
type RecordingSessionOption func(*domain.RecordingSession)

// WithSessionID sets the session ID.
func WithSessionID(id string) RecordingSessionOption {
	return func(s *domain.RecordingSession) {
		s.ID = id
	}
}

// WithSessionProfileID sets the profile ID.
func WithSessionProfileID(id string) RecordingSessionOption {
	return func(s *domain.RecordingSession) {
		s.ProfileID = id
	}
}

// WithSessionStatus sets the session status.
func WithSessionStatus(status domain.SessionStatus) RecordingSessionOption {
	return func(s *domain.RecordingSession) {
		s.Status = status
	}
}

// WithSessionViewport sets the viewport dimensions.
func WithSessionViewport(width, height int) RecordingSessionOption {
	return func(s *domain.RecordingSession) {
		s.ViewportWidth = width
		s.ViewportHeight = height
	}
}

// WithSessionCreatedAt sets the creation time.
func WithSessionCreatedAt(t time.Time) RecordingSessionOption {
	return func(s *domain.RecordingSession) {
		s.CreatedAt = t
	}
}

// NewRecordingSession creates a RecordingSession with sensible defaults.
func NewRecordingSession(opts ...RecordingSessionOption) *domain.RecordingSession {
	s := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// TimelineEntryOption modifies a UnifiedTimelineEntry during construction.
type TimelineEntryOption func(*recordingpersistence.UnifiedTimelineEntry)

// WithEntryID sets the entry ID.
func WithEntryID(id uuid.UUID) TimelineEntryOption {
	return func(e *recordingpersistence.UnifiedTimelineEntry) {
		e.ID = id
	}
}

// WithEntryType sets the entry type.
func WithEntryType(t recordingpersistence.TimelineEntryType) TimelineEntryOption {
	return func(e *recordingpersistence.UnifiedTimelineEntry) {
		e.Type = t
	}
}

// WithEntryTimestamp sets the timestamp.
func WithEntryTimestamp(t time.Time) TimelineEntryOption {
	return func(e *recordingpersistence.UnifiedTimelineEntry) {
		e.Timestamp = t
	}
}

// WithEntrySessionID sets the session ID.
func WithEntrySessionID(id string) TimelineEntryOption {
	return func(e *recordingpersistence.UnifiedTimelineEntry) {
		e.SessionID = id
	}
}

// WithEntryPageID sets the page ID.
func WithEntryPageID(id uuid.UUID) TimelineEntryOption {
	return func(e *recordingpersistence.UnifiedTimelineEntry) {
		e.PageID = id
	}
}

// WithEntrySequence sets the sequence number.
func WithEntrySequence(seq int) TimelineEntryOption {
	return func(e *recordingpersistence.UnifiedTimelineEntry) {
		e.Sequence = seq
	}
}

// WithEntryAction sets the action.
func WithEntryAction(a *domain.RecordingAction) TimelineEntryOption {
	return func(e *recordingpersistence.UnifiedTimelineEntry) {
		e.Action = a
		e.Type = recordingpersistence.TimelineEntryTypeAction
	}
}

// WithEntryPageEvent sets the page event.
func WithEntryPageEvent(pe *domain.PageEvent) TimelineEntryOption {
	return func(e *recordingpersistence.UnifiedTimelineEntry) {
		e.PageEvent = pe
		e.Type = recordingpersistence.TimelineEntryTypePageEvent
	}
}

// NewTimelineEntry creates a UnifiedTimelineEntry with sensible defaults.
func NewTimelineEntry(opts ...TimelineEntryOption) *recordingpersistence.UnifiedTimelineEntry {
	e := &recordingpersistence.UnifiedTimelineEntry{
		ID:        uuid.New(),
		Type:      recordingpersistence.TimelineEntryTypeAction,
		Timestamp: time.Now(),
		SessionID: uuid.New().String(),
		PageID:    uuid.New(),
		Sequence:  1,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// RecordingActionOption modifies a RecordingAction during construction.
type RecordingActionOption func(*domain.RecordingAction)

// WithActionID sets the action ID.
func WithActionID(id uuid.UUID) RecordingActionOption {
	return func(a *domain.RecordingAction) {
		a.ID = id
	}
}

// WithActionType sets the action type.
func WithActionType(t string) RecordingActionOption {
	return func(a *domain.RecordingAction) {
		a.ActionType = t
	}
}

// WithActionURL sets the URL.
func WithActionURL(url string) RecordingActionOption {
	return func(a *domain.RecordingAction) {
		a.URL = url
	}
}

// WithActionSelector sets the selector.
func WithActionSelector(primary string) RecordingActionOption {
	return func(a *domain.RecordingAction) {
		a.Selector = &domain.SelectorSet{Primary: primary}
	}
}

// NewRecordingAction creates a RecordingAction with sensible defaults.
func NewRecordingAction(opts ...RecordingActionOption) *domain.RecordingAction {
	a := &domain.RecordingAction{
		ID:          uuid.New(),
		SessionID:   uuid.New().String(),
		PageID:      uuid.New(),
		SequenceNum: 1,
		ActionType:  "click",
		Timestamp:   time.Now(),
		Confidence:  0.95,
		Source:      domain.ActionSourceManual,
		CreatedAt:   time.Now(),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// PageEventOption modifies a PageEvent during construction.
type PageEventOption func(*domain.PageEvent)

// WithPageEventID sets the event ID.
func WithPageEventID(id uuid.UUID) PageEventOption {
	return func(e *domain.PageEvent) {
		e.ID = id
	}
}

// WithPageEventType sets the event type.
func WithPageEventType(t domain.PageEventType) PageEventOption {
	return func(e *domain.PageEvent) {
		e.Type = t
	}
}

// WithPageEventURL sets the URL.
func WithPageEventURL(url string) PageEventOption {
	return func(e *domain.PageEvent) {
		e.URL = url
	}
}

// WithPageEventPageID sets the page ID.
func WithPageEventPageID(id uuid.UUID) PageEventOption {
	return func(e *domain.PageEvent) {
		e.PageID = id
	}
}

// NewPageEvent creates a PageEvent with sensible defaults.
func NewPageEvent(opts ...PageEventOption) *domain.PageEvent {
	e := &domain.PageEvent{
		ID:        uuid.New(),
		Type:      domain.PageEventCreated,
		PageID:    uuid.New(),
		URL:       "https://example.com",
		Title:     "Example Page",
		Timestamp: time.Now(),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// SessionProfileOption modifies a SessionProfile during construction.
type SessionProfileOption func(*sessionprofilepersistence.SessionProfile)

// WithProfileID sets the profile ID.
func WithProfileID(id sessionprofilepersistence.ProfileID) SessionProfileOption {
	return func(p *sessionprofilepersistence.SessionProfile) {
		p.ID = id
	}
}

// WithProfileName sets the profile name.
func WithProfileName(name string) SessionProfileOption {
	return func(p *sessionprofilepersistence.SessionProfile) {
		p.Name = name
	}
}

// WithProfileCreatedAt sets the creation time.
func WithProfileCreatedAt(t time.Time) SessionProfileOption {
	return func(p *sessionprofilepersistence.SessionProfile) {
		p.CreatedAt = t
	}
}

// WithProfileUpdatedAt sets the update time.
func WithProfileUpdatedAt(t time.Time) SessionProfileOption {
	return func(p *sessionprofilepersistence.SessionProfile) {
		p.UpdatedAt = t
	}
}

// WithProfileLastUsedAt sets the last used time.
func WithProfileLastUsedAt(t time.Time) SessionProfileOption {
	return func(p *sessionprofilepersistence.SessionProfile) {
		p.LastUsedAt = t
	}
}

// WithProfileStorageState sets the storage state.
func WithProfileStorageState(state json.RawMessage) SessionProfileOption {
	return func(p *sessionprofilepersistence.SessionProfile) {
		p.StorageState = state
	}
}

// WithProfileHistory sets the history entries.
func WithProfileHistory(entries []sessionprofilepersistence.HistoryEntry) SessionProfileOption {
	return func(p *sessionprofilepersistence.SessionProfile) {
		p.History = entries
	}
}

// WithProfileOpenTabs sets the open tabs.
func WithProfileOpenTabs(tabs []sessionprofilepersistence.TabState) SessionProfileOption {
	return func(p *sessionprofilepersistence.SessionProfile) {
		p.OpenTabs = tabs
	}
}

// NewSessionProfile creates a SessionProfile with sensible defaults.
func NewSessionProfile(opts ...SessionProfileOption) *sessionprofilepersistence.SessionProfile {
	now := time.Now()
	p := &sessionprofilepersistence.SessionProfile{
		ID:         sessionprofilepersistence.ProfileID(uuid.New().String()),
		Name:       "Test Session",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// HistoryEntryOption modifies a HistoryEntry during construction.
type HistoryEntryOption func(*sessionprofilepersistence.HistoryEntry)

// WithHistoryEntryID sets the entry ID.
func WithHistoryEntryID(id string) HistoryEntryOption {
	return func(e *sessionprofilepersistence.HistoryEntry) {
		e.ID = id
	}
}

// WithHistoryEntryURL sets the URL.
func WithHistoryEntryURL(url string) HistoryEntryOption {
	return func(e *sessionprofilepersistence.HistoryEntry) {
		e.URL = url
	}
}

// WithHistoryEntryTitle sets the title.
func WithHistoryEntryTitle(title string) HistoryEntryOption {
	return func(e *sessionprofilepersistence.HistoryEntry) {
		e.Title = title
	}
}

// WithHistoryEntryTimestamp sets the timestamp.
func WithHistoryEntryTimestamp(ts string) HistoryEntryOption {
	return func(e *sessionprofilepersistence.HistoryEntry) {
		e.Timestamp = ts
	}
}

// NewHistoryEntry creates a HistoryEntry with sensible defaults.
func NewHistoryEntry(opts ...HistoryEntryOption) *sessionprofilepersistence.HistoryEntry {
	e := &sessionprofilepersistence.HistoryEntry{
		ID:        uuid.New().String(),
		URL:       "https://example.com",
		Title:     "Example Page",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// TabStateOption modifies a TabState during construction.
type TabStateOption func(*sessionprofilepersistence.TabState)

// WithTabStateURL sets the URL.
func WithTabStateURL(url string) TabStateOption {
	return func(t *sessionprofilepersistence.TabState) {
		t.URL = url
	}
}

// WithTabStateTitle sets the title.
func WithTabStateTitle(title string) TabStateOption {
	return func(t *sessionprofilepersistence.TabState) {
		t.Title = title
	}
}

// WithTabStateActive sets whether the tab is active.
func WithTabStateActive(active bool) TabStateOption {
	return func(t *sessionprofilepersistence.TabState) {
		t.IsActive = active
	}
}

// WithTabStateOrder sets the order.
func WithTabStateOrder(order int) TabStateOption {
	return func(t *sessionprofilepersistence.TabState) {
		t.Order = order
	}
}

// NewTabState creates a TabState with sensible defaults.
func NewTabState(opts ...TabStateOption) *sessionprofilepersistence.TabState {
	t := &sessionprofilepersistence.TabState{
		URL:      "https://example.com",
		Title:    "Example Page",
		IsActive: false,
		Order:    0,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}
