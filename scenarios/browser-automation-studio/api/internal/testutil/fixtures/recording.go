// Package fixtures contains deterministic domain-object builders for API tests.
package fixtures

import (
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/domain"
	recordingpersistence "github.com/vrooli/browser-automation-studio/services/recording/persistence"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// RecordingSessionOption changes the default recording session fixture.
type RecordingSessionOption func(*domain.RecordingSession)

// RecordingSession returns an active recording session with useful defaults.
func RecordingSession(opts ...RecordingSessionOption) *domain.RecordingSession {
	session := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now().UTC(),
	}
	for _, opt := range opts {
		opt(session)
	}
	return session
}

// WithRecordingSessionID sets the recording session ID.
func WithRecordingSessionID(id string) RecordingSessionOption {
	return func(session *domain.RecordingSession) {
		session.ID = id
	}
}

// WithRecordingSessionProfileID sets the recording session profile ID.
func WithRecordingSessionProfileID(id string) RecordingSessionOption {
	return func(session *domain.RecordingSession) {
		session.ProfileID = id
	}
}

// WithRecordingSessionStatus sets the recording session status.
func WithRecordingSessionStatus(status domain.SessionStatus) RecordingSessionOption {
	return func(session *domain.RecordingSession) {
		session.Status = status
	}
}

// WithRecordingSessionViewport sets the recording session viewport.
func WithRecordingSessionViewport(width, height int) RecordingSessionOption {
	return func(session *domain.RecordingSession) {
		session.ViewportWidth = width
		session.ViewportHeight = height
	}
}

// WithRecordingSessionCreatedAt sets the recording session creation time.
func WithRecordingSessionCreatedAt(createdAt time.Time) RecordingSessionOption {
	return func(session *domain.RecordingSession) {
		session.CreatedAt = createdAt
	}
}

// TimelineEntryOption changes the default unified timeline entry fixture.
type TimelineEntryOption func(*recordingpersistence.UnifiedTimelineEntry)

// TimelineEntry returns an action timeline entry with useful defaults.
func TimelineEntry(opts ...TimelineEntryOption) *recordingpersistence.UnifiedTimelineEntry {
	entry := &recordingpersistence.UnifiedTimelineEntry{
		ID:        uuid.New(),
		Type:      recordingpersistence.TimelineEntryTypeAction,
		Timestamp: time.Now().UTC(),
		SessionID: uuid.New().String(),
		PageID:    uuid.New(),
		Sequence:  1,
	}
	for _, opt := range opts {
		opt(entry)
	}
	return entry
}

// WithTimelineEntryID sets the timeline entry ID.
func WithTimelineEntryID(id uuid.UUID) TimelineEntryOption {
	return func(entry *recordingpersistence.UnifiedTimelineEntry) {
		entry.ID = id
	}
}

// WithTimelineEntryType sets the timeline entry type.
func WithTimelineEntryType(entryType recordingpersistence.TimelineEntryType) TimelineEntryOption {
	return func(entry *recordingpersistence.UnifiedTimelineEntry) {
		entry.Type = entryType
	}
}

// WithTimelineEntryTimestamp sets the timeline entry timestamp.
func WithTimelineEntryTimestamp(timestamp time.Time) TimelineEntryOption {
	return func(entry *recordingpersistence.UnifiedTimelineEntry) {
		entry.Timestamp = timestamp
	}
}

// WithTimelineEntrySessionID sets the timeline entry session ID.
func WithTimelineEntrySessionID(sessionID string) TimelineEntryOption {
	return func(entry *recordingpersistence.UnifiedTimelineEntry) {
		entry.SessionID = sessionID
	}
}

// WithTimelineEntryPageID sets the timeline entry page ID.
func WithTimelineEntryPageID(pageID uuid.UUID) TimelineEntryOption {
	return func(entry *recordingpersistence.UnifiedTimelineEntry) {
		entry.PageID = pageID
	}
}

// WithTimelineEntrySequence sets the timeline entry sequence number.
func WithTimelineEntrySequence(sequence int) TimelineEntryOption {
	return func(entry *recordingpersistence.UnifiedTimelineEntry) {
		entry.Sequence = sequence
	}
}

// WithTimelineEntryAction attaches a recording action and marks the entry as action typed.
func WithTimelineEntryAction(action *domain.RecordingAction) TimelineEntryOption {
	return func(entry *recordingpersistence.UnifiedTimelineEntry) {
		entry.Action = action
		entry.Type = recordingpersistence.TimelineEntryTypeAction
	}
}

// WithTimelineEntryPageEvent attaches a page event and marks the entry as page-event typed.
func WithTimelineEntryPageEvent(event *domain.PageEvent) TimelineEntryOption {
	return func(entry *recordingpersistence.UnifiedTimelineEntry) {
		entry.PageEvent = event
		entry.Type = recordingpersistence.TimelineEntryTypePageEvent
	}
}

// RecordingActionOption changes the default recording action fixture.
type RecordingActionOption func(*domain.RecordingAction)

// RecordingAction returns a click action with useful defaults.
func RecordingAction(opts ...RecordingActionOption) *domain.RecordingAction {
	action := &domain.RecordingAction{
		ID:          uuid.New(),
		SessionID:   uuid.New().String(),
		PageID:      uuid.New(),
		SequenceNum: 1,
		ActionType:  "click",
		Timestamp:   time.Now().UTC(),
		Confidence:  0.95,
		Source:      domain.ActionSourceManual,
		CreatedAt:   time.Now().UTC(),
	}
	for _, opt := range opts {
		opt(action)
	}
	return action
}

// WithRecordingActionID sets the recording action ID.
func WithRecordingActionID(id uuid.UUID) RecordingActionOption {
	return func(action *domain.RecordingAction) {
		action.ID = id
	}
}

// WithRecordingActionSessionID sets the recording action session ID.
func WithRecordingActionSessionID(sessionID string) RecordingActionOption {
	return func(action *domain.RecordingAction) {
		action.SessionID = sessionID
	}
}

// WithRecordingActionPageID sets the recording action page ID.
func WithRecordingActionPageID(pageID uuid.UUID) RecordingActionOption {
	return func(action *domain.RecordingAction) {
		action.PageID = pageID
	}
}

// WithRecordingActionType sets the recording action type.
func WithRecordingActionType(actionType string) RecordingActionOption {
	return func(action *domain.RecordingAction) {
		action.ActionType = actionType
	}
}

// WithRecordingActionTimestamp sets the recording action timestamp.
func WithRecordingActionTimestamp(timestamp time.Time) RecordingActionOption {
	return func(action *domain.RecordingAction) {
		action.Timestamp = timestamp
	}
}

// WithRecordingActionSelector sets the primary selector for the recording action.
func WithRecordingActionSelector(primary string) RecordingActionOption {
	return func(action *domain.RecordingAction) {
		action.Selector = &domain.SelectorSet{Primary: primary}
	}
}

// PageEventOption changes the default page event fixture.
type PageEventOption func(*domain.PageEvent)

// PageEvent returns a page-created event with useful defaults.
func PageEvent(opts ...PageEventOption) *domain.PageEvent {
	event := &domain.PageEvent{
		ID:        uuid.New(),
		Type:      domain.PageEventCreated,
		PageID:    uuid.New(),
		URL:       "https://example.com",
		Title:     "Example Page",
		Timestamp: time.Now().UTC(),
	}
	for _, opt := range opts {
		opt(event)
	}
	return event
}

// WithPageEventID sets the page event ID.
func WithPageEventID(id uuid.UUID) PageEventOption {
	return func(event *domain.PageEvent) {
		event.ID = id
	}
}

// WithPageEventType sets the page event type.
func WithPageEventType(eventType domain.PageEventType) PageEventOption {
	return func(event *domain.PageEvent) {
		event.Type = eventType
	}
}

// WithPageEventPageID sets the page event page ID.
func WithPageEventPageID(pageID uuid.UUID) PageEventOption {
	return func(event *domain.PageEvent) {
		event.PageID = pageID
	}
}

// WithPageEventURL sets the page event URL.
func WithPageEventURL(url string) PageEventOption {
	return func(event *domain.PageEvent) {
		event.URL = url
	}
}

// WithPageEventTimestamp sets the page event timestamp.
func WithPageEventTimestamp(timestamp time.Time) PageEventOption {
	return func(event *domain.PageEvent) {
		event.Timestamp = timestamp
	}
}

// SessionProfileOption changes the default session profile fixture.
type SessionProfileOption func(*sessionprofilepersistence.SessionProfile)

// SessionProfile returns a session profile with useful defaults.
func SessionProfile(opts ...SessionProfileOption) *sessionprofilepersistence.SessionProfile {
	now := time.Now().UTC()
	profile := &sessionprofilepersistence.SessionProfile{
		ID:         sessionprofilepersistence.ProfileID(uuid.New().String()),
		Name:       "Test Session",
		CreatedAt:  now,
		UpdatedAt:  now,
		LastUsedAt: now,
	}
	for _, opt := range opts {
		opt(profile)
	}
	return profile
}

// WithSessionProfileID sets the session profile ID.
func WithSessionProfileID(id sessionprofilepersistence.ProfileID) SessionProfileOption {
	return func(profile *sessionprofilepersistence.SessionProfile) {
		profile.ID = id
	}
}

// WithSessionProfileName sets the session profile name.
func WithSessionProfileName(name string) SessionProfileOption {
	return func(profile *sessionprofilepersistence.SessionProfile) {
		profile.Name = name
	}
}
