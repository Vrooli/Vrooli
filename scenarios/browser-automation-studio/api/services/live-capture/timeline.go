package livecapture

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/domain"
)

// TimelineService manages the unified timeline for recording sessions.
// It stores both actions and page events in chronological order.
type TimelineService struct {
	entries map[string][]domain.TimelineEntry // sessionID -> entries
	mu      sync.RWMutex
}

// duplicateNavigateThreshold is the time window for detecting duplicate navigate actions.
// If two navigate actions to the same URL occur within this window, the second is skipped.
const duplicateNavigateThreshold = 500 * time.Millisecond

// NewTimelineService creates a new timeline service.
func NewTimelineService() *TimelineService {
	return &TimelineService{
		entries: make(map[string][]domain.TimelineEntry),
	}
}

// AddAction adds a recorded action to the timeline.
// It includes deduplication logic for navigate actions to prevent double-navigation
// flickering when both the API and browser script capture the same navigation.
func (s *TimelineService) AddAction(sessionID string, action *RecordedAction, pageID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Parse timestamp from action
	ts, err := time.Parse(time.RFC3339Nano, action.Timestamp)
	if err != nil {
		ts = time.Now()
	}

	// Deduplication: Skip duplicate navigate actions to the same URL within threshold.
	// This prevents double-navigation entries when both the Go API handler and browser
	// script capture the same navigation event.
	if action.ActionType == "navigate" && action.URL != "" {
		entries := s.entries[sessionID]
		// Check recent entries (last 5) for duplicate navigate to same URL
		checkCount := 5
		if len(entries) < checkCount {
			checkCount = len(entries)
		}
		for i := len(entries) - 1; i >= len(entries)-checkCount && i >= 0; i-- {
			existing := entries[i]
			if existing.Action != nil &&
				existing.Action.ActionType == "navigate" &&
				existing.Action.URL == action.URL &&
				ts.Sub(existing.Timestamp) < duplicateNavigateThreshold {
				// Duplicate navigate action detected, skip it
				return
			}
		}
	}

	// Create simplified action entry for timeline storage
	actionEntry := &domain.RecordedActionEntry{
		ID:          action.ID,
		ActionType:  action.ActionType,
		URL:         action.URL,
		SequenceNum: action.SequenceNum,
		Timestamp:   action.Timestamp,
		Confidence:  action.Confidence,
		PageTitle:   action.PageTitle,
	}

	// Add selector info if available
	if action.Selector != nil {
		actionEntry.Selector = &domain.SelectorInfo{
			Primary: action.Selector.Primary,
		}
	}

	// Copy payload if present
	if action.Payload != nil {
		actionEntry.Payload = action.Payload
	}

	entry := domain.TimelineEntry{
		ID:        uuid.New(),
		Type:      domain.TimelineTypeAction,
		Timestamp: ts,
		PageID:    pageID,
		Action:    actionEntry,
	}

	s.entries[sessionID] = append(s.entries[sessionID], entry)
}

// AddPageEvent adds a page lifecycle event to the timeline.
func (s *TimelineService) AddPageEvent(sessionID string, event *domain.PageEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := domain.TimelineEntry{
		ID:        uuid.New(),
		Type:      domain.TimelineTypePageEvent,
		Timestamp: event.Timestamp,
		PageID:    event.PageID,
		PageEvent: event,
	}

	s.entries[sessionID] = append(s.entries[sessionID], entry)
}

// GetTimeline returns all timeline entries for a session, sorted by timestamp.
func (s *TimelineService) GetTimeline(sessionID string) []domain.TimelineEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.entries[sessionID]
	if entries == nil {
		return []domain.TimelineEntry{}
	}

	// Make a copy to avoid modifying the original
	result := make([]domain.TimelineEntry, len(entries))
	copy(result, entries)

	// Sort by timestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

// GetTimelineForPage returns timeline entries filtered by page ID.
func (s *TimelineService) GetTimelineForPage(sessionID string, pageID uuid.UUID) []domain.TimelineEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.entries[sessionID]
	if entries == nil {
		return []domain.TimelineEntry{}
	}

	var filtered []domain.TimelineEntry
	for _, entry := range entries {
		if entry.PageID == pageID {
			filtered = append(filtered, entry)
		}
	}

	// Sort by timestamp
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	return filtered
}

// GetTimelinePaginated returns timeline entries with pagination support.
func (s *TimelineService) GetTimelinePaginated(sessionID string, pageID *uuid.UUID, since *time.Time, limit int) ([]domain.TimelineEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.entries[sessionID]
	if entries == nil {
		return []domain.TimelineEntry{}, false
	}

	// Filter entries
	var filtered []domain.TimelineEntry
	for _, entry := range entries {
		// Filter by page if specified
		if pageID != nil && entry.PageID != *pageID {
			continue
		}
		// Filter by timestamp if specified
		if since != nil && !entry.Timestamp.After(*since) {
			continue
		}
		filtered = append(filtered, entry)
	}

	// Sort by timestamp
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	// Apply limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	hasMore := len(filtered) > limit
	if hasMore {
		filtered = filtered[:limit]
	}

	return filtered, hasMore
}

// GetTimelineCount returns the total number of entries for a session.
func (s *TimelineService) GetTimelineCount(sessionID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries[sessionID])
}

// ClearSession removes all timeline entries for a session.
func (s *TimelineService) ClearSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, sessionID)
}
