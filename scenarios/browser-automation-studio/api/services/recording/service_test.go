package recording

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/internal/clock"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
)

func TestNewService(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewService_WithClock(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	mockClock := clock.NewMock(time.Time{})

	svc := NewService(repo, nil, log, ServiceConfig{
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

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	svc := NewService(repo, nil, log, ServiceConfig{})
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

func TestService_DuplicateNavigateDetection(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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
	now := time.Now()

	// Record a navigate action
	action1 := &driver.RecordedAction{
		ID:         uuid.New().String(),
		SessionID:  session.ID,
		ActionType: "navigate",
		URL:        "https://example.com",
		Timestamp:  now.Format(time.RFC3339Nano),
		Confidence: 1.0,
	}

	err = svc.RecordAction(ctx, session.ID, action1, pageID, ActionSourceAuto)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}

	// Record a duplicate navigate action (same URL, within threshold)
	action2 := &driver.RecordedAction{
		ID:         uuid.New().String(),
		SessionID:  session.ID,
		ActionType: "navigate",
		URL:        "https://example.com",
		Timestamp:  now.Add(100 * time.Millisecond).Format(time.RFC3339Nano),
		Confidence: 1.0,
	}

	err = svc.RecordAction(ctx, session.ID, action2, pageID, ActionSourceAuto)
	if err != nil {
		t.Fatalf("RecordAction failed: %v", err)
	}

	// Verify only one action was recorded (duplicate was skipped)
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: session.ID,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	if timeline.TotalCount != 1 {
		t.Errorf("expected 1 timeline entry (duplicate skipped), got %d", timeline.TotalCount)
	}
}

func TestService_GetTimelineCount(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Check initial count
	if count := svc.GetTimelineCount(session.ID); count != 0 {
		t.Errorf("expected initial count 0, got %d", count)
	}

	// Add some actions
	pageID := uuid.New()
	for i := 0; i < 5; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			SessionID:  session.ID,
			ActionType: "click",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 0.95,
		}
		_ = svc.RecordAction(ctx, session.ID, action, pageID, ActionSourceManual)
	}

	// Check count after adding actions
	if count := svc.GetTimelineCount(session.ID); count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
}

// =============================================================================
// Unified API Tests
// =============================================================================

func TestService_RecordActionUnified_Success(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	result, err := svc.RecordActionUnified(ctx, RecordActionRequest{
		SessionID:     session.ID,
		Action:        action,
		PageID:        pageID,
		Source:        ActionSourceManual,
		CorrelationID: "test-correlation-123",
	})
	if err != nil {
		t.Fatalf("RecordActionUnified failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.CorrelationID != "test-correlation-123" {
		t.Errorf("expected correlation ID 'test-correlation-123', got '%s'", result.CorrelationID)
	}

	if result.ActionID == uuid.Nil {
		t.Error("expected non-nil action ID")
	}

	if result.SequenceNum != 1 {
		t.Errorf("expected sequence number 1, got %d", result.SequenceNum)
	}

	if !result.Persisted {
		t.Error("expected action to be persisted")
	}

	if result.HasErrors() {
		t.Errorf("expected no errors, got %v", result.Errors)
	}

	// Verify action was recorded in timeline
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: session.ID,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	if timeline.TotalCount != 1 {
		t.Errorf("expected 1 timeline entry, got %d", timeline.TotalCount)
	}
}

func TestService_RecordActionUnified_NilAction(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	result, err := svc.RecordActionUnified(ctx, RecordActionRequest{
		SessionID: session.ID,
		Action:    nil, // nil action
		PageID:    pageID,
		Source:    ActionSourceManual,
	})
	if err != nil {
		t.Fatalf("RecordActionUnified should not return error for nil action: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.HasErrors() {
		t.Error("expected errors for nil action")
	}

	// Check for validation error
	foundValidationError := false
	for _, e := range result.Errors {
		if e.Stage == "validation" && e.Message == "action is nil" {
			foundValidationError = true
			break
		}
	}
	if !foundValidationError {
		t.Errorf("expected validation error for nil action, got %v", result.Errors)
	}
}

func TestService_RecordActionUnified_PersistenceError(t *testing.T) {
	repo := persistence.NewMockRepository()
	repo.SaveTimelineEntryErr = errors.New("database connection failed")

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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
	action := &driver.RecordedAction{
		ID:         uuid.New().String(),
		SessionID:  session.ID,
		ActionType: "click",
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		Confidence: 0.95,
	}

	result, err := svc.RecordActionUnified(ctx, RecordActionRequest{
		SessionID: session.ID,
		Action:    action,
		PageID:    pageID,
		Source:    ActionSourceManual,
	})
	if err != nil {
		t.Fatalf("RecordActionUnified should not return error on persistence failure: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Action should still be in cache (partial success)
	if result.SequenceNum != 1 {
		t.Errorf("expected sequence number 1, got %d", result.SequenceNum)
	}

	// Persisted should be false
	if result.Persisted {
		t.Error("expected Persisted to be false on persistence error")
	}

	// Should have persistence error
	if !result.HasErrors() {
		t.Error("expected errors for persistence failure")
	}

	foundPersistenceError := false
	for _, e := range result.Errors {
		if e.Stage == "persistence" {
			foundPersistenceError = true
			break
		}
	}
	if !foundPersistenceError {
		t.Errorf("expected persistence error, got %v", result.Errors)
	}

	// Verify action was still cached despite persistence error
	count := svc.GetTimelineCount(session.ID)
	if count != 1 {
		t.Errorf("expected 1 entry in cache despite persistence error, got %d", count)
	}
}

func TestService_RecordPageEventUnified_Success(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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
	event := &domain.PageEvent{
		ID:        uuid.New(),
		Type:      domain.PageEventCreated,
		PageID:    pageID,
		URL:       "https://example.com",
		Title:     "Example",
		Timestamp: time.Now(),
	}

	result, err := svc.RecordPageEventUnified(ctx, RecordPageEventRequest{
		SessionID:     session.ID,
		Event:         event,
		CorrelationID: "test-page-event-123",
	})
	if err != nil {
		t.Fatalf("RecordPageEventUnified failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.CorrelationID != "test-page-event-123" {
		t.Errorf("expected correlation ID 'test-page-event-123', got '%s'", result.CorrelationID)
	}

	if result.ActionID != event.ID {
		t.Errorf("expected action ID %s, got %s", event.ID, result.ActionID)
	}

	if result.SequenceNum != 1 {
		t.Errorf("expected sequence number 1, got %d", result.SequenceNum)
	}

	if !result.Persisted {
		t.Error("expected event to be persisted")
	}

	if result.HasErrors() {
		t.Errorf("expected no errors, got %v", result.Errors)
	}

	// Verify event was recorded in timeline
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: session.ID,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	if timeline.TotalCount != 1 {
		t.Errorf("expected 1 timeline entry, got %d", timeline.TotalCount)
	}

	if len(timeline.Entries) > 0 && timeline.Entries[0].Type != persistence.TimelineEntryTypePageEvent {
		t.Errorf("expected page_event entry type, got %s", timeline.Entries[0].Type)
	}
}

func TestService_RecordPageEventUnified_NilEvent(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	result, err := svc.RecordPageEventUnified(ctx, RecordPageEventRequest{
		SessionID: session.ID,
		Event:     nil, // nil event
	})
	if err != nil {
		t.Fatalf("RecordPageEventUnified should not return error for nil event: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.HasErrors() {
		t.Error("expected errors for nil event")
	}

	// Check for validation error
	foundValidationError := false
	for _, e := range result.Errors {
		if e.Stage == "validation" && e.Message == "event is nil" {
			foundValidationError = true
			break
		}
	}
	if !foundValidationError {
		t.Errorf("expected validation error for nil event, got %v", result.Errors)
	}
}

// =============================================================================
// Cache Eviction Tests
// =============================================================================

func TestService_CacheEviction_TruncatesAt1000(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	// Add 1001 actions to trigger cache eviction
	for i := 0; i < 1001; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			SessionID:  session.ID,
			ActionType: "click",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 0.95,
		}
		_ = svc.RecordAction(ctx, session.ID, action, pageID, ActionSourceManual)
	}

	// Cache should have been truncated from 1001 to ~501 entries
	count := svc.GetTimelineCount(session.ID)
	if count > 600 {
		t.Errorf("expected cache to be truncated, got %d entries (should be around 501)", count)
	}
	if count < 500 {
		t.Errorf("expected at least 500 entries after truncation, got %d", count)
	}
}

func TestService_CacheEviction_PreservesNewestEntries(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	// Add 1001 actions - each with a unique URL to identify it
	for i := 0; i < 1001; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			SessionID:  session.ID,
			ActionType: "click",
			URL:        fmt.Sprintf("https://example.com/action-%d", i),
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 0.95,
		}
		_ = svc.RecordAction(ctx, session.ID, action, pageID, ActionSourceManual)
	}

	// Get timeline and verify newest entries are preserved
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: session.ID,
		Limit:     1000,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	// The last action (action-1000) should still be in the cache
	foundLatest := false
	for _, entry := range timeline.Entries {
		if entry.Action != nil && entry.Action.URL == "https://example.com/action-1000" {
			foundLatest = true
			break
		}
	}
	if !foundLatest {
		t.Error("expected newest entry (action-1000) to be preserved after cache eviction")
	}

	// The first action (action-0) should NOT be in the cache (evicted)
	foundOldest := false
	for _, entry := range timeline.Entries {
		if entry.Action != nil && entry.Action.URL == "https://example.com/action-0" {
			foundOldest = true
			break
		}
	}
	if foundOldest {
		t.Error("expected oldest entry (action-0) to be evicted from cache")
	}
}

// =============================================================================
// GetTimelineForPage Tests
// =============================================================================

func TestService_GetTimelineForPage_Success(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	svc := NewService(repo, nil, log, ServiceConfig{})
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
// ClearSession Tests
// =============================================================================

func TestService_ClearSession_RemovesFromCache(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
	ctx := context.Background()

	// Create a session
	session, err := svc.CreateSession(ctx, SessionConfig{
		ViewportWidth:  1920,
		ViewportHeight: 1080,
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Record some actions
	pageID := uuid.New()
	for i := 0; i < 5; i++ {
		action := &driver.RecordedAction{
			ID:         uuid.New().String(),
			SessionID:  session.ID,
			ActionType: "click",
			Timestamp:  time.Now().Format(time.RFC3339Nano),
			Confidence: 0.95,
		}
		_ = svc.RecordAction(ctx, session.ID, action, pageID, ActionSourceManual)
	}

	// Verify entries are in cache
	if count := svc.GetTimelineCount(session.ID); count != 5 {
		t.Fatalf("expected 5 entries before clear, got %d", count)
	}

	// Clear the session
	svc.ClearSession(session.ID)

	// Verify cache is empty
	if count := svc.GetTimelineCount(session.ID); count != 0 {
		t.Errorf("expected 0 entries after clear, got %d", count)
	}
}

func TestService_ClearSession_NonExistentSession(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})

	// Should not panic when clearing non-existent session
	svc.ClearSession("non-existent-session-id")

	// Verify count is 0 for non-existent session
	count := svc.GetTimelineCount("non-existent-session-id")
	if count != 0 {
		t.Errorf("expected 0 for non-existent session, got %d", count)
	}
}

// =============================================================================
// WarmCache Tests
// =============================================================================

func TestService_WarmCache_LoadsFromRepository(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
	ctx := context.Background()

	sessionID := "test-session-for-warming"

	// Create session directly in repository (simulating existing data)
	session := &domain.RecordingSession{
		ID:             sessionID,
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	_ = repo.CreateSession(ctx, session)

	// Add entries directly to repository
	pageID := uuid.New()
	for i := 0; i < 5; i++ {
		entry := &persistence.UnifiedTimelineEntry{
			ID:        uuid.New(),
			Type:      persistence.TimelineEntryTypeAction,
			Timestamp: time.Now(),
			SessionID: sessionID,
			PageID:    pageID,
			Sequence:  i + 1,
			Action: &domain.RecordingAction{
				ID:         uuid.New(),
				SessionID:  sessionID,
				ActionType: "click",
				Timestamp:  time.Now(),
			},
		}
		_ = repo.SaveTimelineEntry(ctx, entry)
	}

	// Cache should be empty initially
	if count := svc.GetTimelineCount(sessionID); count != 0 {
		t.Fatalf("expected empty cache initially, got %d", count)
	}

	// Warm the cache
	err := svc.WarmCache(ctx, sessionID)
	if err != nil {
		t.Fatalf("WarmCache failed: %v", err)
	}

	// Cache should now have entries
	count := svc.GetTimelineCount(sessionID)
	if count != 5 {
		t.Errorf("expected 5 entries after warming, got %d", count)
	}
}

func TestService_WarmCache_NilRepository(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Create service with nil repository
	svc := NewService(nil, nil, log, ServiceConfig{})
	ctx := context.Background()

	// Should not error when repository is nil
	err := svc.WarmCache(ctx, "any-session-id")
	if err != nil {
		t.Errorf("WarmCache should not error with nil repository: %v", err)
	}
}

func TestService_WarmCache_SetsCorrectSequence(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
	ctx := context.Background()

	sessionID := "test-session-sequence"

	// Create session
	session := &domain.RecordingSession{
		ID:             sessionID,
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1920,
		ViewportHeight: 1080,
		CreatedAt:      time.Now(),
	}
	_ = repo.CreateSession(ctx, session)

	// Add entries with specific sequence numbers
	pageID := uuid.New()
	maxSeq := 42
	for i := 0; i < 5; i++ {
		entry := &persistence.UnifiedTimelineEntry{
			ID:        uuid.New(),
			Type:      persistence.TimelineEntryTypeAction,
			Timestamp: time.Now(),
			SessionID: sessionID,
			PageID:    pageID,
			Sequence:  maxSeq - 4 + i, // 38, 39, 40, 41, 42
			Action: &domain.RecordingAction{
				ID:         uuid.New(),
				SessionID:  sessionID,
				ActionType: "click",
				Timestamp:  time.Now(),
			},
		}
		_ = repo.SaveTimelineEntry(ctx, entry)
	}

	// Warm the cache
	err := svc.WarmCache(ctx, sessionID)
	if err != nil {
		t.Fatalf("WarmCache failed: %v", err)
	}

	// Record a new action - it should get sequence 43 (maxSeq + 1)
	action := &driver.RecordedAction{
		ID:         uuid.New().String(),
		SessionID:  sessionID,
		ActionType: "click",
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		Confidence: 0.95,
	}
	_ = svc.RecordAction(ctx, sessionID, action, pageID, ActionSourceManual)

	// Get timeline and check sequence
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	// Find the newest entry (should have sequence 43)
	maxFoundSeq := 0
	for _, entry := range timeline.Entries {
		if entry.Sequence > maxFoundSeq {
			maxFoundSeq = entry.Sequence
		}
	}

	if maxFoundSeq != maxSeq+1 {
		t.Errorf("expected max sequence %d after warming and adding, got %d", maxSeq+1, maxFoundSeq)
	}
}

// =============================================================================
// filterEntries Tests
// =============================================================================

func TestService_FilterEntries_ByPageID(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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

func TestService_FilterEntries_BySince(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	mockClock := clock.NewMock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	svc := NewService(repo, nil, log, ServiceConfig{Clock: mockClock})
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

func TestService_FilterEntries_ByEntryType(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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
		event := &domain.PageEvent{
			ID:        uuid.New(),
			Type:      domain.PageEventCreated,
			PageID:    pageID,
			Timestamp: time.Now(),
		}
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

func TestService_FilterEntries_SortsChronologically(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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

	// Record actions with interleaved timestamps on different pages
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	action1 := &driver.RecordedAction{ID: uuid.New().String(), ActionType: "click", Timestamp: baseTime.Format(time.RFC3339Nano)}
	action2 := &driver.RecordedAction{ID: uuid.New().String(), ActionType: "click", Timestamp: baseTime.Add(2 * time.Second).Format(time.RFC3339Nano)}
	action3 := &driver.RecordedAction{ID: uuid.New().String(), ActionType: "click", Timestamp: baseTime.Add(1 * time.Second).Format(time.RFC3339Nano)}

	_ = svc.RecordAction(ctx, session.ID, action1, page1, ActionSourceManual)
	_ = svc.RecordAction(ctx, session.ID, action2, page2, ActionSourceManual)
	_ = svc.RecordAction(ctx, session.ID, action3, page1, ActionSourceManual)

	// Filter by page1 (should sort chronologically)
	timeline, err := svc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: session.ID,
		PageID:    &page1,
	})
	if err != nil {
		t.Fatalf("GetTimeline failed: %v", err)
	}

	if len(timeline.Entries) != 2 {
		t.Fatalf("expected 2 entries for page1, got %d", len(timeline.Entries))
	}

	// Verify chronological order
	if !timeline.Entries[0].Timestamp.Before(timeline.Entries[1].Timestamp) {
		t.Error("expected entries to be sorted chronologically")
	}
}

// =============================================================================
// SetOnAction Tests
// =============================================================================

func TestService_SetOnAction_InvokesCallback(t *testing.T) {
	repo := persistence.NewMockRepository()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	svc := NewService(repo, nil, log, ServiceConfig{})
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
	svc := NewService(nil, nil, log, ServiceConfig{})
	ctx := context.Background()

	// Should return nil, nil when repository is nil
	session, err := svc.GetSession(ctx, "any-id")
	if err != nil {
		t.Errorf("expected no error with nil repository, got: %v", err)
	}
	if session != nil {
		t.Error("expected nil session with nil repository")
	}
}
