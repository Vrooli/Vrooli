package recording

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/scheduletest"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	recordingschema "github.com/vrooli/browser-automation-studio/internal/recording"
	"github.com/vrooli/browser-automation-studio/internal/testutil/fixtures"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
	_ "modernc.org/sqlite"
)

func TestNewService(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewService_WithClock(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	mockClock := scheduletest.New(time.Time{})

	svc := NewService(repo, ServiceConfig{
		Clock: mockClock,
	})

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	// Verify clock is used
	ctx := context.Background()
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if !session.CreatedAt.Equal(mockClock.Now()) {
		t.Errorf("expected CreatedAt to use mock clock time, got %v", session.CreatedAt)
	}
}

func TestService_CreateSession(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.ID == "" {
		t.Error("expected non-empty session ID")
	}

	if session.ViewportWidth != 1920 {
		t.Errorf("expected viewport width 1920, got %d", session.ViewportWidth)
	}

	if session.Status != domain.SessionStatusActive {
		t.Errorf("expected status active, got %s", session.Status)
	}
}

func TestService_RecordAction(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	// Create a session first
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Record an action
	action := &driver.RecordedAction{
		ID:         uuid.New().String(),
		SessionID:  session.ID,
		ActionType: "click",
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		Confidence: 0.95,
		Selector: &driver.SelectorSet{
			Primary: "#submit-button",
		},
	}

	pageID := uuid.New()
	err = svc.RecordAction(ctx, session.ID, action, pageID, ActionSourceManual)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}

	// Verify the action was recorded
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: session.ID,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	if timeline.TotalCount != 1 {
		t.Errorf("expected 1 timeline entry, got %d", timeline.TotalCount)
	}

	if len(timeline.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(timeline.Entries))
	}

	entry := timeline.Entries[0]
	if entry.Type != persistence.TimelineEntryTypeAction {
		t.Errorf("expected entry type action, got %s", entry.Type)
	}

	if entry.Action == nil {
		t.Fatal("expected action to be non-nil")
	}

	if entry.Action.ActionType != "click" {
		t.Errorf("expected action type click, got %s", entry.Action.ActionType)
	}
}

func TestService_RecordPageEvent(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	// Create a session first
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Record a page event using domain.PageEvent
	pageID := uuid.New()
	event := &domain.PageEvent{
		ID:        uuid.New(),
		Type:      domain.PageEventCreated,
		PageID:    pageID,
		URL:       "https://example.com",
		Title:     "Example",
		Timestamp: time.Now(),
	}

	err = svc.RecordPageEvent(ctx, session.ID, event)
	if err != nil {
		t.Fatalf("RecordPageEvent failed: %v", err)
	}

	// Verify the event was recorded
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: session.ID,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	if timeline.TotalCount != 1 {
		t.Errorf("expected 1 timeline entry, got %d", timeline.TotalCount)
	}

	if len(timeline.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(timeline.Entries))
	}

	entry := timeline.Entries[0]
	if entry.Type != persistence.TimelineEntryTypePageEvent {
		t.Errorf("expected entry type page_event, got %s", entry.Type)
	}

	if entry.PageEvent == nil {
		t.Fatal("expected page event to be non-nil")
	}

	if entry.PageEvent.URL != "https://example.com" {
		t.Errorf("expected URL https://example.com, got %s", entry.PageEvent.URL)
	}
}

func TestService_CloseSession(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Close the session
	err = svc.CloseSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("CloseSession failed: %v", err)
	}

	// Verify the session is closed
	closedSession, err := svc.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if closedSession.Status != domain.SessionStatusClosed {
		t.Errorf("expected status closed, got %s", closedSession.Status)
	}

	if closedSession.ClosedAt == nil {
		t.Error("expected ClosedAt to be non-nil")
	}
}

func TestActionSource_Values(t *testing.T) {
	// Verify the constants exist and have expected values
	if ActionSourceAuto != "auto" {
		t.Errorf("expected ActionSourceAuto to be 'auto', got '%s'", ActionSourceAuto)
	}

	if ActionSourceManual != "manual" {
		t.Errorf("expected ActionSourceManual to be 'manual', got '%s'", ActionSourceManual)
	}

	if ActionSourceAI != "ai" {
		t.Errorf("expected ActionSourceAI to be 'ai', got '%s'", ActionSourceAI)
	}
}

// =============================================================================
// Unified API Tests
// =============================================================================

// =============================================================================
// Cache Eviction Tests
// =============================================================================

// =============================================================================
// GetTimelineForPage Tests
// =============================================================================

func TestService_GetTimelineForPage_Success(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Create two different page IDs
	page1 := uuid.New()
	page2 := uuid.New()

	// Record actions on page 1
	for i := 0; i < 3; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			SessionID:  session.ID,
			ActionType: "click",
			URL:        "https://page1.example.com",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 0.95,
		}
		_ = svc.RecordAction(ctx, session.ID, action, page1, ActionSourceManual)
	}

	// Record actions on page 2
	for i := 0; i < 2; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			SessionID:  session.ID,
			ActionType: "type",
			URL:        "https://page2.example.com",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 0.90,
		}
		_ = svc.RecordAction(ctx, session.ID, action, page2, ActionSourceManual)
	}

	// Get timeline for page 1 only
	entries, err := svc.GetTimelineForPage(ctx, session.ID, page1, 100)
	if err != nil {
		t.Fatalf("GetTimelineForPage failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries for page1, got %d", len(entries))
	}

	// Verify all entries belong to page1
	for _, entry := range entries {
		if entry.PageID != page1 {
			t.Errorf("expected all entries to have page ID %s, got %s", page1, entry.PageID)
		}
	}
}

func TestService_GetTimelineForPage_EmptyResult(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Query for a page that has no entries
	nonExistentPage := uuid.New()
	entries, err := svc.GetTimelineForPage(ctx, session.ID, nonExistentPage, 100)
	if err != nil {
		t.Fatalf("GetTimelineForPage failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries for non-existent page, got %d", len(entries))
	}
}

func TestService_GetTimelineForPage_RespectsLimit(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	pageID := uuid.New()

	// Record 10 actions
	for i := 0; i < 10; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			SessionID:  session.ID,
			ActionType: "click",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 0.95,
		}
		_ = svc.RecordAction(ctx, session.ID, action, pageID, ActionSourceManual)
	}

	// Get timeline with limit of 5
	entries, err := svc.GetTimelineForPage(ctx, session.ID, pageID, 5)
	if err != nil {
		t.Fatalf("GetTimelineForPage failed: %v", err)
	}

	if len(entries) > 5 {
		t.Errorf("expected at most 5 entries with limit, got %d", len(entries))
	}
}

// =============================================================================
// filterEntries Tests
// =============================================================================

func TestService_Query_ByPageID(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	page1 := uuid.New()
	page2 := uuid.New()

	// Record actions on both pages
	for i := 0; i < 3; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			ActionType: "click",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
		}
		_ = svc.RecordAction(ctx, session.ID, action, page1, ActionSourceManual)
	}
	for i := 0; i < 2; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			ActionType: "type",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
		}
		_ = svc.RecordAction(ctx, session.ID, action, page2, ActionSourceManual)
	}

	// Filter by page1
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: session.ID,
		PageID:    &page1,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	if len(timeline.Entries) != 3 {
		t.Errorf("expected 3 entries for page1, got %d", len(timeline.Entries))
	}
}

func TestService_Query_BySince(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	mockClock := scheduletest.New(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	svc := NewService(repo, ServiceConfig{Clock: mockClock})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	pageID := uuid.New()

	// Record actions at different times
	times := []time.Time{
		time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 12, 2, 0, 0, time.UTC),
		time.Date(2024, 1, 1, 12, 3, 0, 0, time.UTC),
	}

	for _, ts := range times {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			ActionType: "click",
			Timestamp:  ts.Format(time.RFC3339Nano),
		}
		_ = svc.RecordAction(ctx, session.ID, action, pageID, ActionSourceManual)
	}

	// Filter to only get entries after 12:01:30
	since := time.Date(2024, 1, 1, 12, 1, 30, 0, time.UTC)
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: session.ID,
		Since:     &since,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	// Should only get entries at 12:02 and 12:03
	if len(timeline.Entries) != 2 {
		t.Errorf("expected 2 entries after 12:01:30, got %d", len(timeline.Entries))
	}
}

func TestService_Query_ByEntryType(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	pageID := uuid.New()

	// Record some actions
	for i := 0; i < 3; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			ActionType: "click",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
		}
		_ = svc.RecordAction(ctx, session.ID, action, pageID, ActionSourceManual)
	}

	// Record some page events
	for i := 0; i < 2; i++ {
		event := fixtures.PageEvent(fixtures.WithPageEventPageID(pageID))
		_ = svc.RecordPageEvent(ctx, session.ID, event)
	}

	// Filter by action type only
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID:  session.ID,
		EntryTypes: []persistence.TimelineEntryType{persistence.TimelineEntryTypeAction},
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	if len(timeline.Entries) != 3 {
		t.Errorf("expected 3 action entries, got %d", len(timeline.Entries))
	}

	// Verify all are actions
	for _, entry := range timeline.Entries {
		if entry.Type != persistence.TimelineEntryTypeAction {
			t.Errorf("expected action type, got %s", entry.Type)
		}
	}
}

// =============================================================================
// SetOnAction Tests
// =============================================================================

func TestService_SetOnAction_InvokesCallback(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, ServiceConfig{})
	ctx := context.Background()

	// Track callback invocations
	var callbackSessionID string
	var callbackEntry *persistence.UnifiedTimelineEntry
	callbackCount := 0

	svc.SetOnAction(func(sessionID string, entry *persistence.UnifiedTimelineEntry) {
		callbackSessionID = sessionID
		callbackEntry = entry
		callbackCount++
	})

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Record an action
	pageID := uuid.New()
	action := &driver.RecordedAction{
		ID:         uuid.New().String(),
		ActionType: "click",
		Timestamp:  time.Now().Format(time.RFC3339Nano),
	}
	_ = svc.RecordAction(ctx, session.ID, action, pageID, ActionSourceManual)

	// Verify callback was invoked
	if callbackCount != 1 {
		t.Errorf("expected callback to be invoked once, got %d", callbackCount)
	}

	if callbackSessionID != session.ID {
		t.Errorf("expected session ID %s, got %s", session.ID, callbackSessionID)
	}

	if callbackEntry == nil {
		t.Error("expected non-nil entry in callback")
	}
}

// =============================================================================
// GetSession Tests
// =============================================================================

func TestService_GetSession_NilRepository(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Create service with nil repository
	svc := NewService(nil, ServiceConfig{})
	ctx := context.Background()

	// Missing storage must remain explicit.
	session, err := svc.GetSession(ctx, "any-id")
	if !errors.Is(err, ErrRepositoryUnavailable) {
		t.Errorf("expected unavailable repository, got: %v", err)
	}
	if session != nil {
		t.Error("expected nil session with nil repository")
	}
}

// [REQ:BAS-RH-J06] Use production schema and a real disk-backed journal.
func openJournalFixture(t *testing.T, path string) (*Service, *sql.DB) {
	t.Helper()
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{TimeFormat: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	routed, err := coredb.Open(context.Background(), coredb.Config{Driver: coredb.DriverSQLite, DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	db := routed.Primary()
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(recordingschema.Schema()); err != nil {
		t.Fatal(err)
	}
	return NewService(persistence.NewSQLiteRepository(db), ServiceConfig{}), db
}

func journalAction(id string) *driver.RecordedAction {
	return &driver.RecordedAction{ID: id, ActionType: "input", Timestamp: "2026-09-22T00:00:00Z", Payload: map[string]interface{}{"text": "original"}}
}

func TestJournalFailedCommitIsNotAcknowledgedOrPublished(t *testing.T) {
	svc, db := openJournalFixture(t, filepath.Join(t.TempDir(), "journal.db"))
	ctx := context.Background()
	sess, err := svc.CreateSession(ctx, SessionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TRIGGER reject_journal BEFORE INSERT ON timeline_entries BEGIN SELECT RAISE(ABORT, 'synthetic disk failure'); END"); err != nil {
		t.Fatal(err)
	}
	broadcasts := 0
	svc.SetOnAction(func(string, *persistence.UnifiedTimelineEntry) { broadcasts++ })
	if err := svc.RecordAction(ctx, sess.ID, journalAction(uuid.NewString()), uuid.Nil, ActionSourceAuto); err == nil {
		t.Error("failed action commit was acknowledged")
	}
	if err := svc.RecordPageEvent(ctx, sess.ID, &domain.PageEvent{ID: uuid.New(), Timestamp: time.Now()}); err == nil {
		t.Error("failed page-event commit was acknowledged")
	}
	if broadcasts != 0 {
		t.Errorf("published %d uncommitted entries", broadcasts)
	}
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{SessionID: sess.ID})
	if err != nil || timeline.TotalCount != 0 || len(timeline.Entries) != 0 {
		t.Errorf("failed entries leaked into history: %+v %v", timeline, err)
	}
}

func TestJournalHistorySurvivesPaginationReopenAndConcurrentWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.db")
	svc, db := openJournalFixture(t, path)
	ctx := context.Background()
	sess, err := svc.CreateSession(ctx, SessionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1001; i++ {
		if err := svc.RecordAction(ctx, sess.ID, journalAction(uuid.NewString()), uuid.Nil, ActionSourceAuto); err != nil {
			t.Fatal(err)
		}
	}
	assertPage := func(s *Service, offset int) {
		t.Helper()
		result, err := s.GetTimeline(ctx, persistence.TimelineQuery{SessionID: sess.ID, Limit: 100, Offset: offset})
		if err != nil {
			t.Fatal(err)
		}
		if result.TotalCount != 1001 || len(result.Entries) != 100 || result.Entries[0].Sequence != offset+1 || !result.HasMore {
			t.Fatalf("page offset%d: total%d entries%d start%d more%v", offset, result.TotalCount, len(result.Entries), result.Entries[0].Sequence, result.HasMore)
		}
	}
	assertPage(svc, 0)
	assertPage(svc, 100)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	first, _ := openJournalFixture(t, path)
	second, _ := openJournalFixture(t, path)
	assertPage(first, 0)
	assertPage(first, 100)
	var wg sync.WaitGroup
	for _, s := range []*Service{first, second} {
		wg.Add(1)
		go func(s *Service) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				if err := s.RecordAction(ctx, sess.ID, journalAction(uuid.NewString()), uuid.Nil, ActionSourceAuto); err != nil {
					t.Error(err)
					return
				}
			}
		}(s)
	}
	wg.Wait()
	result, err := first.GetTimeline(ctx, persistence.TimelineQuery{SessionID: sess.ID, Offset: 1000, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCount != 1041 || len(result.Entries) != 41 || result.HasMore {
		t.Fatalf("complete history: total%d entries%d more%v", result.TotalCount, len(result.Entries), result.HasMore)
	}
	for i, e := range result.Entries {
		if e.Sequence != 1001+i {
			t.Fatalf("journal sequence %d: want%d got%d", i, 1001+i, e.Sequence)
		}
	}
}

func TestJournalRetryIdentityPreservesDistinctNavigations(t *testing.T) {
	svc, _ := openJournalFixture(t, filepath.Join(t.TempDir(), "journal.db"))
	ctx := context.Background()
	sess, err := svc.CreateSession(ctx, SessionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	action := journalAction(uuid.NewString())
	action.ActionType = "navigate"
	action.URL = "https://example.test/same"
	for i := 0; i < 2; i++ {
		if err := svc.RecordAction(ctx, sess.ID, action, uuid.Nil, ActionSourceAuto); err != nil {
			t.Fatal(err)
		}
	}
	distinct := *action
	distinct.ID = uuid.NewString()
	if err := svc.RecordAction(ctx, sess.ID, &distinct, uuid.Nil, ActionSourceAuto); err != nil {
		t.Fatal(err)
	}
	result, err := svc.GetTimeline(ctx, persistence.TimelineQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCount != 2 || len(result.Entries) != 2 || result.Entries[0].ID == result.Entries[1].ID {
		t.Fatalf("retry must deduplicate by identity and retain distinct observations: %+v", result)
	}
	action.Payload = map[string]interface{}{"text": "conflicting payload"}
	if err := svc.RecordAction(ctx, sess.ID, action, uuid.Nil, ActionSourceAuto); err == nil {
		t.Fatal("conflicting identity reuse was acknowledged")
	}
}

func TestJournalCorruptCommittedDataFailsRead(t *testing.T) {
	svc, db := openJournalFixture(t, filepath.Join(t.TempDir(), "journal.db"))
	ctx := context.Background()
	sess, err := svc.CreateSession(ctx, SessionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	action := journalAction(uuid.NewString())
	if err := svc.RecordAction(ctx, sess.ID, action, uuid.Nil, ActionSourceAuto); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE timeline_entries SET action_json='not JSON'"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetTimeline(ctx, persistence.TimelineQuery{SessionID: sess.ID}); err == nil {
		t.Error("corrupt history was silently accepted")
	}
	if err := svc.RecordAction(ctx, sess.ID, action, uuid.Nil, ActionSourceAuto); err == nil {
		t.Error("retry hid corrupt committed payload")
	}
}

func TestJournalRetryPreservesLargeJSONNumbersAndNotifiesOnce(t *testing.T) {
	svc, _ := openJournalFixture(t, filepath.Join(t.TempDir(), "journal.db"))
	ctx := context.Background()
	sess, err := svc.CreateSession(ctx, SessionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	action := journalAction("opaque-event-1")
	action.Payload = map[string]interface{}{"exact": json.Number("9007199254740993")}
	notifications := 0
	svc.SetOnAction(func(string, *persistence.UnifiedTimelineEntry) { notifications++ })
	for i := 0; i < 2; i++ {
		if err := svc.RecordAction(ctx, sess.ID, action, uuid.Nil, ActionSourceAuto); err != nil {
			t.Fatal(err)
		}
	}
	result, err := svc.GetTimeline(ctx, persistence.TimelineQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatal(err)
	}
	if notifications != 1 || result.TotalCount != 1 || result.Entries[0].Action.Payload["exact"] != json.Number("9007199254740993") {
		t.Fatalf("retry or precision lost: notifications%d history%+v", notifications, result)
	}
}

func TestJournalRejectsTrailingCorruptJSON(t *testing.T) {
	svc, db := openJournalFixture(t, filepath.Join(t.TempDir(), "journal.db"))
	ctx := context.Background()
	sess, err := svc.CreateSession(ctx, SessionConfig{})
	if err != nil {
		t.Fatal(err)
	}
	action := journalAction(uuid.NewString())
	if err := svc.RecordAction(ctx, sess.ID, action, uuid.Nil, ActionSourceAuto); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE timeline_entries SET action_json=action_json || ' trailing corruption'"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetTimeline(ctx, persistence.TimelineQuery{SessionID: sess.ID}); err == nil {
		t.Error("valid prefix hid trailing journal corruption")
	}
	if err := svc.RecordAction(ctx, sess.ID, action, uuid.Nil, ActionSourceAuto); err == nil {
		t.Error("retry acknowledged corrupt committed JSON")
	}
}
