package livecapture

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
)

func TestTimelineService_AddAction(t *testing.T) {
	svc := NewTimelineService()
	sessionID := "test-session-123"
	pageID := uuid.New()

	action := &driver.RecordedAction{
		ID:          uuid.NewString(),
		ActionType:  "click",
		URL:         "https://example.com",
		SequenceNum: 1,
		Timestamp:   time.Now().Format(time.RFC3339Nano),
		Confidence:  0.95,
	}

	svc.AddAction(sessionID, action, pageID)

	entries := svc.GetTimeline(sessionID)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Type != domain.TimelineTypeAction {
		t.Errorf("expected type 'action', got '%s'", entries[0].Type)
	}

	if entries[0].Action == nil {
		t.Error("expected action to be non-nil")
	}

	if entries[0].Action.ActionType != "click" {
		t.Errorf("expected action type 'click', got '%s'", entries[0].Action.ActionType)
	}
}

func TestTimelineService_AddPageEvent(t *testing.T) {
	svc := NewTimelineService()
	sessionID := "test-session-123"
	pageID := uuid.New()

	event := &domain.PageEvent{
		ID:        uuid.New(),
		Type:      domain.PageEventCreated,
		PageID:    pageID,
		URL:       "https://example.com/new-tab",
		Title:     "New Tab",
		Timestamp: time.Now(),
	}

	svc.AddPageEvent(sessionID, event)

	entries := svc.GetTimeline(sessionID)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Type != domain.TimelineTypePageEvent {
		t.Errorf("expected type 'page_event', got '%s'", entries[0].Type)
	}

	if entries[0].PageEvent == nil {
		t.Error("expected page event to be non-nil")
	}

	if entries[0].PageEvent.Type != domain.PageEventCreated {
		t.Errorf("expected event type 'created', got '%s'", entries[0].PageEvent.Type)
	}
}

func TestTimelineService_ChronologicalOrdering(t *testing.T) {
	svc := NewTimelineService()
	sessionID := "test-session-123"
	pageID := uuid.New()

	// Add events in non-chronological order
	now := time.Now()

	// Add action at T+2
	action2 := &driver.RecordedAction{
		ID:          uuid.NewString(),
		ActionType:  "type",
		SequenceNum: 2,
		Timestamp:   now.Add(200 * time.Millisecond).Format(time.RFC3339Nano),
	}
	svc.AddAction(sessionID, action2, pageID)

	// Add page event at T+1
	pageEvent := &domain.PageEvent{
		ID:        uuid.New(),
		Type:      domain.PageEventNavigated,
		PageID:    pageID,
		Timestamp: now.Add(100 * time.Millisecond),
	}
	svc.AddPageEvent(sessionID, pageEvent)

	// Add action at T+0
	action1 := &driver.RecordedAction{
		ID:          uuid.NewString(),
		ActionType:  "click",
		SequenceNum: 1,
		Timestamp:   now.Format(time.RFC3339Nano),
	}
	svc.AddAction(sessionID, action1, pageID)

	// Get timeline and verify chronological order
	entries := svc.GetTimeline(sessionID)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// First entry should be the click (T+0)
	if entries[0].Type != domain.TimelineTypeAction || entries[0].Action.ActionType != "click" {
		t.Error("first entry should be click action")
	}

	// Second entry should be the page event (T+1)
	if entries[1].Type != domain.TimelineTypePageEvent {
		t.Error("second entry should be page event")
	}

	// Third entry should be the type action (T+2)
	if entries[2].Type != domain.TimelineTypeAction || entries[2].Action.ActionType != "type" {
		t.Error("third entry should be type action")
	}

	// Verify timestamps are in order
	for i := 1; i < len(entries); i++ {
		if !entries[i].Timestamp.After(entries[i-1].Timestamp) && entries[i].Timestamp != entries[i-1].Timestamp {
			t.Errorf("entry %d timestamp should be >= entry %d timestamp", i, i-1)
		}
	}
}

func TestTimelineService_FilterByPage(t *testing.T) {
	svc := NewTimelineService()
	sessionID := "test-session-123"
	page1 := uuid.New()
	page2 := uuid.New()

	// Add actions to different pages
	action1 := &driver.RecordedAction{
		ID:          uuid.NewString(),
		ActionType:  "click",
		SequenceNum: 1,
		Timestamp:   time.Now().Format(time.RFC3339Nano),
	}
	svc.AddAction(sessionID, action1, page1)

	action2 := &driver.RecordedAction{
		ID:          uuid.NewString(),
		ActionType:  "type",
		SequenceNum: 2,
		Timestamp:   time.Now().Add(100 * time.Millisecond).Format(time.RFC3339Nano),
	}
	svc.AddAction(sessionID, action2, page2)

	action3 := &driver.RecordedAction{
		ID:          uuid.NewString(),
		ActionType:  "scroll",
		SequenceNum: 3,
		Timestamp:   time.Now().Add(200 * time.Millisecond).Format(time.RFC3339Nano),
	}
	svc.AddAction(sessionID, action3, page1)

	// Get all entries
	allEntries := svc.GetTimeline(sessionID)
	if len(allEntries) != 3 {
		t.Errorf("expected 3 total entries, got %d", len(allEntries))
	}

	// Filter by page1
	page1Entries := svc.GetTimelineForPage(sessionID, page1)
	if len(page1Entries) != 2 {
		t.Errorf("expected 2 entries for page1, got %d", len(page1Entries))
	}

	// Filter by page2
	page2Entries := svc.GetTimelineForPage(sessionID, page2)
	if len(page2Entries) != 1 {
		t.Errorf("expected 1 entry for page2, got %d", len(page2Entries))
	}
}

func TestTimelineService_Pagination(t *testing.T) {
	svc := NewTimelineService()
	sessionID := "test-session-123"
	pageID := uuid.New()

	// Add 10 entries
	for i := 0; i < 10; i++ {
		action := &driver.RecordedAction{
			ID:          uuid.NewString(),
			ActionType:  "click",
			SequenceNum: i + 1,
			Timestamp:   time.Now().Add(time.Duration(i) * time.Millisecond).Format(time.RFC3339Nano),
		}
		svc.AddAction(sessionID, action, pageID)
	}

	// Get first 5 entries
	entries, hasMore := svc.GetTimelinePaginated(sessionID, nil, nil, 5)
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
	if !hasMore {
		t.Error("expected hasMore to be true")
	}

	// Get all entries
	allEntries, hasMoreAll := svc.GetTimelinePaginated(sessionID, nil, nil, 100)
	if len(allEntries) != 10 {
		t.Errorf("expected 10 entries, got %d", len(allEntries))
	}
	if hasMoreAll {
		t.Error("expected hasMore to be false")
	}
}

func TestTimelineService_GetTimelineCount(t *testing.T) {
	svc := NewTimelineService()
	sessionID := "test-session-123"
	pageID := uuid.New()

	if count := svc.GetTimelineCount(sessionID); count != 0 {
		t.Errorf("expected 0 count for empty session, got %d", count)
	}

	// Add some entries
	for i := 0; i < 5; i++ {
		action := &driver.RecordedAction{
			ID:          uuid.NewString(),
			ActionType:  "click",
			SequenceNum: i + 1,
			Timestamp:   time.Now().Format(time.RFC3339Nano),
		}
		svc.AddAction(sessionID, action, pageID)
	}

	if count := svc.GetTimelineCount(sessionID); count != 5 {
		t.Errorf("expected 5 count, got %d", count)
	}
}

func TestTimelineService_ClearSession(t *testing.T) {
	svc := NewTimelineService()
	sessionID := "test-session-123"
	pageID := uuid.New()

	// Add some entries
	action := &driver.RecordedAction{
		ID:          uuid.NewString(),
		ActionType:  "click",
		SequenceNum: 1,
		Timestamp:   time.Now().Format(time.RFC3339Nano),
	}
	svc.AddAction(sessionID, action, pageID)

	if count := svc.GetTimelineCount(sessionID); count != 1 {
		t.Errorf("expected 1 count, got %d", count)
	}

	// Clear session
	svc.ClearSession(sessionID)

	if count := svc.GetTimelineCount(sessionID); count != 0 {
		t.Errorf("expected 0 count after clear, got %d", count)
	}

	entries := svc.GetTimeline(sessionID)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(entries))
	}
}

func TestTimelineService_EmptySession(t *testing.T) {
	svc := NewTimelineService()
	sessionID := "nonexistent-session"

	entries := svc.GetTimeline(sessionID)
	if entries == nil {
		t.Error("expected non-nil slice for empty session")
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}

	pageID := uuid.New()
	filtered := svc.GetTimelineForPage(sessionID, pageID)
	if filtered == nil {
		t.Error("expected non-nil slice for filtered empty session")
	}
	if len(filtered) != 0 {
		t.Errorf("expected 0 filtered entries, got %d", len(filtered))
	}
}
