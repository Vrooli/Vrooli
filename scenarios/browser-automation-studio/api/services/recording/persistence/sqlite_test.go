package persistence

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/domain"
	_ "modernc.org/sqlite"
)

// newTestDB creates an in-memory SQLite database for testing.
// Uses shared cache mode to allow concurrent access from multiple connections.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use file::memory:?cache=shared for concurrent access support
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	// Set connection pool to single connection to avoid table creation race
	db.SetMaxOpenConns(1)

	// Create tables
	schema := `
		CREATE TABLE IF NOT EXISTS recording_sessions (
			id TEXT PRIMARY KEY,
			profile_id TEXT,
			status TEXT NOT NULL,
			viewport_width INTEGER NOT NULL,
			viewport_height INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			closed_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS timeline_entries (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			session_id TEXT NOT NULL,
			page_id TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			action_json TEXT,
			page_event_json TEXT,
			FOREIGN KEY (session_id) REFERENCES recording_sessions(id)
		);

		CREATE INDEX IF NOT EXISTS idx_timeline_session_sequence ON timeline_entries(session_id, sequence);
		CREATE INDEX IF NOT EXISTS idx_timeline_session_page ON timeline_entries(session_id, page_id);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

func newTestRepo(t *testing.T) (*SQLiteRepository, *sql.DB) {
	t.Helper()
	db := newTestDB(t)
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	return NewSQLiteRepository(db, log), db
}

func TestSQLiteRepository_CreateSession(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	session := &domain.RecordingSession{
		ID:             uuid.New().String(),
		ProfileID:      "profile-123",
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      now,
	}

	err := repo.CreateSession(ctx, session)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify it was created
	retrieved, err := repo.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected session to exist")
	}
	if retrieved.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, retrieved.ID)
	}
	if retrieved.ProfileID != session.ProfileID {
		t.Errorf("expected ProfileID %s, got %s", session.ProfileID, retrieved.ProfileID)
	}
	if retrieved.Status != domain.SessionStatusActive {
		t.Errorf("expected status active, got %s", retrieved.Status)
	}
	if retrieved.ViewportWidth != 1920 {
		t.Errorf("expected viewport width 1920, got %d", retrieved.ViewportWidth)
	}
}

func TestSQLiteRepository_GetSession_NotFound(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()
	session, err := repo.GetSession(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session != nil {
		t.Errorf("expected nil session for nonexistent ID")
	}
}

func TestSQLiteRepository_CloseSession(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()

	// Create a session first
	session := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Close it
	closedAt := time.Now().Truncate(time.Millisecond)
	if err := repo.CloseSession(ctx, session.ID, closedAt); err != nil {
		t.Fatalf("CloseSession failed: %v", err)
	}

	// Verify it's closed
	retrieved, err := repo.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved.Status != domain.SessionStatusClosed {
		t.Errorf("expected status closed, got %s", retrieved.Status)
	}
	if retrieved.ClosedAt == nil {
		t.Error("expected ClosedAt to be set")
	}
}

func TestSQLiteRepository_DeleteSession(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()

	// Create session and entry
	session := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	entry := &UnifiedTimelineEntry{
		ID:        uuid.New(),
		Type:      TimelineEntryTypeAction,
		Timestamp: time.Now(),
		SessionID: session.ID,
		PageID:    uuid.New(),
		Sequence:  1,
	}
	if err := repo.SaveTimelineEntry(ctx, entry); err != nil {
		t.Fatalf("SaveTimelineEntry failed: %v", err)
	}

	// Delete the session
	if err := repo.DeleteSession(ctx, session.ID); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify session is gone
	retrieved, err := repo.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved != nil {
		t.Error("expected session to be deleted")
	}

	// Verify entries are gone
	count, err := repo.CountTimelineEntries(ctx, session.ID)
	if err != nil {
		t.Fatalf("CountTimelineEntries failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 entries, got %d", count)
	}
}

func TestSQLiteRepository_ListSessions(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()
	profileID := "test-profile"

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		session := &domain.RecordingSession{
			ID:             uuid.New().String(),
			ProfileID:      profileID,
			Status:         domain.SessionStatusActive,
			ViewportWidth:  1920,
			ViewportHeight: 1080,
			CreatedAt:      time.Now().Add(time.Duration(i) * time.Minute),
		}
		if err := repo.CreateSession(ctx, session); err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
	}

	// Create one with different profile
	other := &domain.RecordingSession{
		ID:             uuid.New().String(),
		ProfileID:      "other-profile",
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	if err := repo.CreateSession(ctx, other); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// List all
	all, err := repo.ListSessions(ctx, nil, 100, 0)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(all) != 6 {
		t.Errorf("expected 6 sessions, got %d", len(all))
	}

	// List by profile
	filtered, err := repo.ListSessions(ctx, &profileID, 100, 0)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(filtered) != 5 {
		t.Errorf("expected 5 sessions for profile, got %d", len(filtered))
	}

	// Test limit
	limited, err := repo.ListSessions(ctx, nil, 3, 0)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(limited) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(limited))
	}
}

func TestSQLiteRepository_SaveTimelineEntry(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()

	// Create session
	session := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Create action entry
	action := &domain.RecordingAction{
		ID:          uuid.New(),
		SessionID:   session.ID,
		ActionType:  "click",
		Confidence:  0.95,
		Selector:    &domain.SelectorSet{Primary: "#submit"},
	}

	entry := &UnifiedTimelineEntry{
		ID:        uuid.New(),
		Type:      TimelineEntryTypeAction,
		Timestamp: time.Now(),
		SessionID: session.ID,
		PageID:    uuid.New(),
		Sequence:  1,
		Action:    action,
	}

	if err := repo.SaveTimelineEntry(ctx, entry); err != nil {
		t.Fatalf("SaveTimelineEntry failed: %v", err)
	}

	// Retrieve it
	retrieved, err := repo.GetTimelineEntry(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetTimelineEntry failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected entry to exist")
	}
	if retrieved.ID != entry.ID {
		t.Errorf("expected ID %s, got %s", entry.ID, retrieved.ID)
	}
	if retrieved.Type != TimelineEntryTypeAction {
		t.Errorf("expected type action, got %s", retrieved.Type)
	}
	if retrieved.Action == nil {
		t.Fatal("expected action to be non-nil")
	}
	if retrieved.Action.ActionType != "click" {
		t.Errorf("expected action type click, got %s", retrieved.Action.ActionType)
	}
}

func TestSQLiteRepository_SaveTimelineEntries_Batch(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()

	// Create session
	session := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Create batch of entries
	entries := make([]*UnifiedTimelineEntry, 10)
	for i := 0; i < 10; i++ {
		entries[i] = &UnifiedTimelineEntry{
			ID:        uuid.New(),
			Type:      TimelineEntryTypeAction,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			SessionID: session.ID,
			PageID:    uuid.New(),
			Sequence:  i + 1,
		}
	}

	if err := repo.SaveTimelineEntries(ctx, entries); err != nil {
		t.Fatalf("SaveTimelineEntries failed: %v", err)
	}

	// Verify count
	count, err := repo.CountTimelineEntries(ctx, session.ID)
	if err != nil {
		t.Fatalf("CountTimelineEntries failed: %v", err)
	}
	if count != 10 {
		t.Errorf("expected 10 entries, got %d", count)
	}
}

func TestSQLiteRepository_GetTimeline_Filtering(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()

	// Create session
	session := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	pageID := uuid.New()
	baseTime := time.Now().Add(-time.Hour)

	// Create mixed entries
	entries := []*UnifiedTimelineEntry{
		{ID: uuid.New(), Type: TimelineEntryTypeAction, Timestamp: baseTime, SessionID: session.ID, PageID: pageID, Sequence: 1},
		{ID: uuid.New(), Type: TimelineEntryTypePageEvent, Timestamp: baseTime.Add(time.Minute), SessionID: session.ID, PageID: pageID, Sequence: 2},
		{ID: uuid.New(), Type: TimelineEntryTypeAction, Timestamp: baseTime.Add(2 * time.Minute), SessionID: session.ID, PageID: uuid.New(), Sequence: 3},
	}

	for _, e := range entries {
		if err := repo.SaveTimelineEntry(ctx, e); err != nil {
			t.Fatalf("SaveTimelineEntry failed: %v", err)
		}
	}

	// Test filter by page ID
	t.Run("filter by page ID", func(t *testing.T) {
		resp, err := repo.GetTimeline(ctx, TimelineQuery{
			SessionID: session.ID,
			PageID:    &pageID,
		})
		if err != nil {
			t.Fatalf("GetTimeline failed: %v", err)
		}
		if len(resp.Entries) != 2 {
			t.Errorf("expected 2 entries for page, got %d", len(resp.Entries))
		}
	})

	// Test filter by entry type
	t.Run("filter by entry type", func(t *testing.T) {
		resp, err := repo.GetTimeline(ctx, TimelineQuery{
			SessionID:  session.ID,
			EntryTypes: []TimelineEntryType{TimelineEntryTypeAction},
		})
		if err != nil {
			t.Fatalf("GetTimeline failed: %v", err)
		}
		if len(resp.Entries) != 2 {
			t.Errorf("expected 2 action entries, got %d", len(resp.Entries))
		}
	})

	// Test filter by timestamp
	t.Run("filter by timestamp", func(t *testing.T) {
		since := baseTime.Add(30 * time.Second)
		resp, err := repo.GetTimeline(ctx, TimelineQuery{
			SessionID: session.ID,
			Since:     &since,
		})
		if err != nil {
			t.Fatalf("GetTimeline failed: %v", err)
		}
		if len(resp.Entries) != 2 {
			t.Errorf("expected 2 entries after timestamp, got %d", len(resp.Entries))
		}
	})
}

func TestSQLiteRepository_PruneOldSessions(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()

	// Create old session
	old := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusClosed,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      now.Add(-30 * 24 * time.Hour), // 30 days old
	}
	if err := repo.CreateSession(ctx, old); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Add entry to old session
	entry := &UnifiedTimelineEntry{
		ID:        uuid.New(),
		Type:      TimelineEntryTypeAction,
		Timestamp: old.CreatedAt,
		SessionID: old.ID,
		PageID:    uuid.New(),
		Sequence:  1,
	}
	if err := repo.SaveTimelineEntry(ctx, entry); err != nil {
		t.Fatalf("SaveTimelineEntry failed: %v", err)
	}

	// Create recent session
	recent := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      now.Add(-time.Hour), // 1 hour old
	}
	if err := repo.CreateSession(ctx, recent); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Prune sessions older than 7 days
	pruned, err := repo.PruneOldSessions(ctx, now.Add(-7*24*time.Hour))
	if err != nil {
		t.Fatalf("PruneOldSessions failed: %v", err)
	}
	if pruned != 1 {
		t.Errorf("expected 1 pruned session, got %d", pruned)
	}

	// Verify old session is gone
	session, err := repo.GetSession(ctx, old.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session != nil {
		t.Error("expected old session to be pruned")
	}

	// Verify recent session still exists
	session, err = repo.GetSession(ctx, recent.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if session == nil {
		t.Error("expected recent session to still exist")
	}
}

func TestSQLiteRepository_ConcurrentAccess(t *testing.T) {
	repo, db := newTestRepo(t)
	defer db.Close()

	ctx := context.Background()

	// Create session
	session := &domain.RecordingSession{
		ID:             uuid.New().String(),
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Concurrent writes
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(seq int) {
			defer wg.Done()
			entry := &UnifiedTimelineEntry{
				ID:        uuid.New(),
				Type:      TimelineEntryTypeAction,
				Timestamp: time.Now(),
				SessionID: session.ID,
				PageID:    uuid.New(),
				Sequence:  seq,
			}
			if err := repo.SaveTimelineEntry(ctx, entry); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent write error: %v", err)
	}

	// Verify all entries were saved
	count, err := repo.CountTimelineEntries(ctx, session.ID)
	if err != nil {
		t.Fatalf("CountTimelineEntries failed: %v", err)
	}
	if count != 10 {
		t.Errorf("expected 10 entries, got %d", count)
	}
}
