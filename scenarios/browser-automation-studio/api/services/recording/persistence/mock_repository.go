// Package persistence provides data access for the unified recording service.
package persistence

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/domain"
)

// CallCounts tracks the number of times each repository method was called.
// This enables tests to verify call patterns and frequencies.
type CallCounts struct {
	CreateSession        int
	GetSession           int
	CloseSession         int
	ListSessions         int
	DeleteSession        int
	SaveTimelineEntry    int
	SaveTimelineEntries  int
	GetTimelineEntry     int
	GetTimeline          int
	CountTimelineEntries int
	DeleteSessionEntries int
	PruneOldSessions     int
}

// MockRepository implements Repository for testing.
// It stores data in memory and is safe for concurrent use.
type MockRepository struct {
	mu       sync.RWMutex
	sessions map[string]*domain.RecordingSession
	entries  map[string][]*UnifiedTimelineEntry

	// Call tracking for test verification
	callCounts CallCounts

	// Error injection for testing error paths
	CreateSessionErr        error
	GetSessionErr           error
	CloseSessionErr         error
	ListSessionsErr         error
	DeleteSessionErr        error
	SaveTimelineEntryErr    error
	SaveTimelineEntriesErr  error
	GetTimelineEntryErr     error
	GetTimelineErr          error
	CountTimelineErr        error
	DeleteSessionEntriesErr error
	PruneOldSessionsErr     error
}

// NewMockRepository creates a new mock repository for testing.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		sessions: make(map[string]*domain.RecordingSession),
		entries:  make(map[string][]*UnifiedTimelineEntry),
	}
}

// CreateSession stores a new recording session.
func (r *MockRepository) CreateSession(ctx context.Context, session *domain.RecordingSession) error {
	r.mu.Lock()
	r.callCounts.CreateSession++
	r.mu.Unlock()

	if r.CreateSessionErr != nil {
		return r.CreateSessionErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	r.entries[session.ID] = []*UnifiedTimelineEntry{}
	return nil
}

// GetSession retrieves a session by ID.
func (r *MockRepository) GetSession(ctx context.Context, sessionID string) (*domain.RecordingSession, error) {
	r.mu.Lock()
	r.callCounts.GetSession++
	r.mu.Unlock()

	if r.GetSessionErr != nil {
		return nil, r.GetSessionErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessions[sessionID], nil
}

// CloseSession marks a session as closed.
func (r *MockRepository) CloseSession(ctx context.Context, sessionID string, closedAt time.Time) error {
	r.mu.Lock()
	r.callCounts.CloseSession++
	r.mu.Unlock()

	if r.CloseSessionErr != nil {
		return r.CloseSessionErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.sessions[sessionID]; ok {
		s.ClosedAt = &closedAt
		s.Status = domain.SessionStatusClosed
	}
	return nil
}

// ListSessions returns sessions with optional filtering.
func (r *MockRepository) ListSessions(ctx context.Context, profileID *string, limit, offset int) ([]*domain.RecordingSession, error) {
	r.mu.Lock()
	r.callCounts.ListSessions++
	r.mu.Unlock()

	if r.ListSessionsErr != nil {
		return nil, r.ListSessionsErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*domain.RecordingSession
	for _, s := range r.sessions {
		if profileID != nil && s.ProfileID != *profileID {
			continue
		}
		result = append(result, s)
	}
	// Apply offset and limit
	if offset >= len(result) {
		return []*domain.RecordingSession{}, nil
	}
	result = result[offset:]
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// DeleteSession removes a session and its entries.
func (r *MockRepository) DeleteSession(ctx context.Context, sessionID string) error {
	r.mu.Lock()
	r.callCounts.DeleteSession++
	r.mu.Unlock()

	if r.DeleteSessionErr != nil {
		return r.DeleteSessionErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionID)
	delete(r.entries, sessionID)
	return nil
}

// SaveTimelineEntry stores a single timeline entry.
func (r *MockRepository) SaveTimelineEntry(ctx context.Context, entry *UnifiedTimelineEntry) error {
	r.mu.Lock()
	r.callCounts.SaveTimelineEntry++
	r.mu.Unlock()

	if r.SaveTimelineEntryErr != nil {
		return r.SaveTimelineEntryErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[entry.SessionID] = append(r.entries[entry.SessionID], entry)
	return nil
}

// SaveTimelineEntries stores multiple entries in a batch.
func (r *MockRepository) SaveTimelineEntries(ctx context.Context, entries []*UnifiedTimelineEntry) error {
	r.mu.Lock()
	r.callCounts.SaveTimelineEntries++
	r.mu.Unlock()

	if r.SaveTimelineEntriesErr != nil {
		return r.SaveTimelineEntriesErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range entries {
		r.entries[e.SessionID] = append(r.entries[e.SessionID], e)
	}
	return nil
}

// GetTimelineEntry retrieves a single entry by ID.
func (r *MockRepository) GetTimelineEntry(ctx context.Context, entryID uuid.UUID) (*UnifiedTimelineEntry, error) {
	r.mu.Lock()
	r.callCounts.GetTimelineEntry++
	r.mu.Unlock()

	if r.GetTimelineEntryErr != nil {
		return nil, r.GetTimelineEntryErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, entries := range r.entries {
		for _, e := range entries {
			if e.ID == entryID {
				return e, nil
			}
		}
	}
	return nil, nil
}

// GetTimeline returns timeline entries matching the query.
func (r *MockRepository) GetTimeline(ctx context.Context, query TimelineQuery) (*TimelineResponse, error) {
	r.mu.Lock()
	r.callCounts.GetTimeline++
	r.mu.Unlock()

	if r.GetTimelineErr != nil {
		return nil, r.GetTimelineErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	entries := r.entries[query.SessionID]
	result := make([]UnifiedTimelineEntry, 0, len(entries))
	for _, e := range entries {
		// Apply filters
		if query.PageID != nil && e.PageID != *query.PageID {
			continue
		}
		if query.Since != nil && !e.Timestamp.After(*query.Since) {
			continue
		}
		if len(query.EntryTypes) > 0 {
			found := false
			for _, t := range query.EntryTypes {
				if e.Type == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, *e)
	}
	hasMore := len(result) > query.Limit && query.Limit > 0
	if hasMore {
		result = result[:query.Limit]
	}
	return &TimelineResponse{
		Entries:    result,
		HasMore:    hasMore,
		TotalCount: len(entries),
	}, nil
}

// CountTimelineEntries returns the total entry count for a session.
func (r *MockRepository) CountTimelineEntries(ctx context.Context, sessionID string) (int, error) {
	r.mu.Lock()
	r.callCounts.CountTimelineEntries++
	r.mu.Unlock()

	if r.CountTimelineErr != nil {
		return 0, r.CountTimelineErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries[sessionID]), nil
}

// DeleteSessionEntries removes all timeline entries for a session.
func (r *MockRepository) DeleteSessionEntries(ctx context.Context, sessionID string) error {
	r.mu.Lock()
	r.callCounts.DeleteSessionEntries++
	r.mu.Unlock()

	if r.DeleteSessionEntriesErr != nil {
		return r.DeleteSessionEntriesErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[sessionID] = nil
	return nil
}

// PruneOldSessions removes sessions older than the given time.
func (r *MockRepository) PruneOldSessions(ctx context.Context, olderThan time.Time) (int, error) {
	r.mu.Lock()
	r.callCounts.PruneOldSessions++
	r.mu.Unlock()

	if r.PruneOldSessionsErr != nil {
		return 0, r.PruneOldSessionsErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	pruned := 0
	for id, s := range r.sessions {
		if s.CreatedAt.Before(olderThan) {
			delete(r.sessions, id)
			delete(r.entries, id)
			pruned++
		}
	}
	return pruned, nil
}

// GetAllSessions returns all sessions (test helper).
func (r *MockRepository) GetAllSessions() []*domain.RecordingSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*domain.RecordingSession, 0, len(r.sessions))
	for _, s := range r.sessions {
		result = append(result, s)
	}
	return result
}

// GetAllEntries returns all entries for a session (test helper).
func (r *MockRepository) GetAllEntries(sessionID string) []*UnifiedTimelineEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries[sessionID]
}

// Clear removes all data (test helper).
func (r *MockRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions = make(map[string]*domain.RecordingSession)
	r.entries = make(map[string][]*UnifiedTimelineEntry)
}

// GetCallCounts returns a snapshot of the call counts for all repository methods.
// This is useful for verifying test expectations about call patterns.
func (r *MockRepository) GetCallCounts() CallCounts {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.callCounts
}

// ResetCallCounts resets all call counters to zero.
// This is useful for resetting state between test cases.
func (r *MockRepository) ResetCallCounts() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callCounts = CallCounts{}
}

// Ensure MockRepository implements Repository at compile time.
var _ Repository = (*MockRepository)(nil)
