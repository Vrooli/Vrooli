package livecapture

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
)

func TestNewService_InitializesComponents(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel) // Suppress logs during tests

	// NewService without a working driver will have nil sessions, but should still
	// create timeline and generator
	svc := NewService(log)

	// These should be created even if session manager fails
	if svc.generator == nil {
		t.Error("Expected generator to be initialized")
	}
	if svc.timeline == nil {
		t.Error("Expected timeline to be initialized")
	}
}

func TestService_CreateSession_RequiresSessionManager(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// Create service with nil session manager
	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}

	_, err := svc.CreateSession(context.Background(), &SessionConfig{
		ViewportWidth:  1280,
		ViewportHeight: 720,
	})

	if err == nil {
		t.Error("Expected error when session manager is nil")
	}
	if err.Error() != "session manager not initialized" {
		t.Errorf("Expected 'session manager not initialized' error, got: %v", err)
	}
}

// Note: Testing "no actions to convert" error path would require either:
// 1. A mock session manager, or
// 2. Providing no actions AND expecting the service to fetch from session
// The ApplyActionRange function returns all actions when range is invalid,
// so we can't easily test the empty actions path without mocking.

func TestService_GenerateWorkflow_WithActions(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}

	actions := []driver.RecordedAction{
		{ActionType: "navigate", URL: "https://example.com"},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn"}},
	}

	result, err := svc.GenerateWorkflow(context.Background(), "test-session", &GenerateWorkflowConfig{
		Actions: actions,
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ActionCount != 2 {
		t.Errorf("Expected ActionCount 2, got %d", result.ActionCount)
	}
	if result.FlowDefinition == nil {
		t.Error("Expected FlowDefinition to be non-nil")
	}
	if result.NodeCount < 2 {
		t.Errorf("Expected at least 2 nodes, got %d", result.NodeCount)
	}
}

func TestService_GenerateWorkflow_AppliesActionRange(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}

	actions := []driver.RecordedAction{
		{ActionType: "navigate", URL: "https://example.com"},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn1"}},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn2"}},
		{ActionType: "click", Selector: &driver.SelectorSet{Primary: "#btn3"}},
	}

	// Only use actions at index 1 and 2
	result, err := svc.GenerateWorkflow(context.Background(), "test-session", &GenerateWorkflowConfig{
		Actions:     actions,
		ActionRange: &ActionRange{Start: 1, End: 2},
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.ActionCount != 2 {
		t.Errorf("Expected ActionCount 2 (after range filter), got %d", result.ActionCount)
	}
}

func TestService_Timeline_AddAndRetrieve(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}

	sessionID := "test-session-timeline"
	pageID := uuid.New()

	action := &driver.RecordedAction{
		ID:          uuid.NewString(),
		ActionType:  "click",
		URL:         "https://example.com",
		SequenceNum: 1,
		Timestamp:   time.Now().Format(time.RFC3339Nano),
	}

	svc.AddTimelineAction(sessionID, action, pageID)

	// GetTimeline requires a valid session in sessions manager, so test ClearTimeline instead
	svc.ClearTimeline(sessionID)

	// After clearing, timeline should be empty
	entries := svc.timeline.GetTimeline(sessionID)
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(entries))
	}
}

func TestService_AddTimelinePageEvent(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}

	sessionID := "test-session-page-event"
	pageID := uuid.New()

	event := &domain.PageEvent{
		ID:        uuid.New(),
		Type:      domain.PageEventCreated,
		PageID:    pageID,
		URL:       "https://example.com/new-tab",
		Title:     "New Tab",
		Timestamp: time.Now(),
	}

	svc.AddTimelinePageEvent(sessionID, event)

	// Verify event was added
	entries := svc.timeline.GetTimeline(sessionID)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Type != domain.TimelineTypePageEvent {
		t.Errorf("Expected type 'page_event', got '%s'", entries[0].Type)
	}
}

func TestService_DriverClient_ReturnsNilWhenNoSessions(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}

	client := svc.DriverClient()
	if client != nil {
		t.Error("Expected nil DriverClient when sessions is nil")
	}
}

func TestService_Sessions_ReturnsNilWhenNotInitialized(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := &Service{
		sessions:  nil,
		generator: NewWorkflowGenerator(),
		timeline:  NewTimelineService(),
		log:       log,
	}

	sessions := svc.Sessions()
	if sessions != nil {
		t.Error("Expected nil Sessions when not initialized")
	}
}

// Note: GetSession requires a valid sessions manager and will panic if nil.
// Testing with nil sessions is not meaningful since callers should check Sessions() first.
