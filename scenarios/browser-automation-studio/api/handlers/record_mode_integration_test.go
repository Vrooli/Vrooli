package handlers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// TestRecordingHub is a test hub that captures broadcasts for verification.
type TestRecordingHub struct {
	mu              sync.RWMutex
	clients         map[string]chan *wsHub.UnifiedTimelineEntry
	broadcastCounts map[string]int
	lastEntry       map[string]*wsHub.UnifiedTimelineEntry
	logger          *logrus.Logger
}

func NewTestRecordingHub(logger *logrus.Logger) *TestRecordingHub {
	return &TestRecordingHub{
		clients:         make(map[string]chan *wsHub.UnifiedTimelineEntry),
		broadcastCounts: make(map[string]int),
		lastEntry:       make(map[string]*wsHub.UnifiedTimelineEntry),
		logger:          logger,
	}
}

// Subscribe adds a test client for a session.
func (h *TestRecordingHub) Subscribe(sessionID string) chan *wsHub.UnifiedTimelineEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan *wsHub.UnifiedTimelineEntry, 100)
	h.clients[sessionID] = ch
	return ch
}

// Unsubscribe removes a test client.
func (h *TestRecordingHub) Unsubscribe(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ch, ok := h.clients[sessionID]; ok {
		close(ch)
		delete(h.clients, sessionID)
	}
}

// GetBroadcastCount returns the number of broadcasts for a session.
func (h *TestRecordingHub) GetBroadcastCount(sessionID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.broadcastCounts[sessionID]
}

// GetLastEntry returns the last broadcast entry for a session.
func (h *TestRecordingHub) GetLastEntry(sessionID string) *wsHub.UnifiedTimelineEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastEntry[sessionID]
}

// Implement wsHub.HubInterface

func (h *TestRecordingHub) ServeWS(conn *websocket.Conn, executionID *uuid.UUID) {}

func (h *TestRecordingHub) BroadcastEnvelope(event any) {}

func (h *TestRecordingHub) BroadcastRecordingEntry(sessionID string, entry *wsHub.UnifiedTimelineEntry) wsHub.BroadcastResult {
	h.mu.Lock()
	defer h.mu.Unlock()

	result := wsHub.BroadcastResult{}

	if entry == nil {
		h.logger.WithField("session_id", sessionID).Warn("TestRecordingHub: nil entry")
		return result
	}

	h.broadcastCounts[sessionID]++
	h.lastEntry[sessionID] = entry

	if ch, ok := h.clients[sessionID]; ok {
		result.SubscriberCount = 1
		select {
		case ch <- entry:
			result.SentCount = 1
		default:
			result.DroppedCount = 1
		}
	}

	return result
}

func (h *TestRecordingHub) BroadcastRecordingFrame(sessionID string, frame *wsHub.RecordingFrame) {}
func (h *TestRecordingHub) BroadcastBinaryFrame(sessionID string, jpegData []byte)               {}
func (h *TestRecordingHub) HasRecordingSubscribers(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[sessionID]
	return ok
}
func (h *TestRecordingHub) BroadcastPerfStats(sessionID string, stats any)     {}
func (h *TestRecordingHub) BroadcastPageEvent(sessionID string, event any)     {}
func (h *TestRecordingHub) BroadcastPageSwitch(sessionID, activePageID string) {}
func (h *TestRecordingHub) HasExecutionFrameSubscribers(executionID string) bool {
	return false
}
func (h *TestRecordingHub) BroadcastExecutionFrame(executionID string, frame *wsHub.ExecutionFrame) {
}
func (h *TestRecordingHub) BroadcastExportProgress(progress *wsHub.ExportProgress) {}
func (h *TestRecordingHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
func (h *TestRecordingHub) Run()                        {}
func (h *TestRecordingHub) CloseExecution(_ uuid.UUID)  {}

// Compile-time interface check
var _ wsHub.HubInterface = (*TestRecordingHub)(nil)

// TestRecordingPipeline_EndToEnd tests the complete recording pipeline.
func TestRecordingPipeline_EndToEnd(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create mock repository (in-memory)
	repo := persistence.NewMockRepository()

	// Create test hub
	hub := NewTestRecordingHub(logger)

	// Create recording service
	recordingSvc := recording.NewService(repo, hub, logger, recording.ServiceConfig{})

	// Create a session
	sessionID := "test-session-" + uuid.NewString()[:8]

	// Subscribe a test client
	clientCh := hub.Subscribe(sessionID)
	defer hub.Unsubscribe(sessionID)

	// Record an action using the unified method
	ctx := context.Background()
	action := &driver.RecordedAction{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		ActionType: "click",
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		Confidence: 0.95,
		URL:        "https://example.com",
	}

	result, err := recordingSvc.RecordActionUnified(ctx, recording.RecordActionRequest{
		SessionID:     sessionID,
		Action:        action,
		PageID:        uuid.New(),
		Source:        recording.ActionSourceManual,
		CorrelationID: "test-corr-123",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify result
	assert.Equal(t, "test-corr-123", result.CorrelationID)
	assert.True(t, result.Persisted, "Action should be persisted")
	assert.True(t, result.BroadcastSent, "Broadcast should be sent")
	assert.Equal(t, 1, result.SubscriberCount, "Should have 1 subscriber")
	assert.Equal(t, 1, result.SentCount, "Should have sent to 1 client")
	assert.Equal(t, 0, result.DroppedCount, "No messages should be dropped")
	assert.False(t, result.HasErrors(), "Should have no errors")

	// Verify action appeared in WebSocket
	select {
	case entry := <-clientCh:
		assert.NotNil(t, entry, "Should receive entry")
		assert.Equal(t, "action", entry.Type)
		assert.Equal(t, "click", entry.Action.ActionType)
	case <-time.After(time.Second):
		t.Fatal("Action did not appear in WebSocket within timeout")
	}

	// Verify persisted in timeline
	timeline, err := recordingSvc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: sessionID,
		Limit:     100,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, timeline.TotalCount, "Should have 1 entry in timeline")
}

// TestRecordingPipeline_NoSubscribers tests recording when no UI is connected.
func TestRecordingPipeline_NoSubscribers(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	repo := persistence.NewMockRepository()
	hub := NewTestRecordingHub(logger)
	recordingSvc := recording.NewService(repo, hub, logger, recording.ServiceConfig{})

	sessionID := "test-session-" + uuid.NewString()[:8]
	// Note: NOT subscribing any client

	ctx := context.Background()
	action := &driver.RecordedAction{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		ActionType: "type",
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		Confidence: 0.9,
	}

	result, err := recordingSvc.RecordActionUnified(ctx, recording.RecordActionRequest{
		SessionID: sessionID,
		Action:    action,
		PageID:    uuid.New(),
		Source:    recording.ActionSourceAuto,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Action should still be persisted
	assert.True(t, result.Persisted, "Action should be persisted even without subscribers")

	// No subscribers, so no broadcast sent
	assert.Equal(t, 0, result.SubscriberCount, "Should have 0 subscribers")
	assert.False(t, result.BroadcastSent, "Broadcast should not be sent")

	// But errors should be empty (no subscribers is not an error)
	assert.False(t, result.HasErrors(), "No subscribers should not cause errors")

	// Verify still in timeline
	timeline, err := recordingSvc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: sessionID,
		Limit:     100,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, timeline.TotalCount, "Action should still be in timeline")
}

// TestRecordingPipeline_CorrelationIDGeneration tests automatic correlation ID generation.
func TestRecordingPipeline_CorrelationIDGeneration(t *testing.T) {
	logger := logrus.New()
	repo := persistence.NewMockRepository()
	hub := NewTestRecordingHub(logger)
	recordingSvc := recording.NewService(repo, hub, logger, recording.ServiceConfig{})

	sessionID := "test-session-" + uuid.NewString()[:8]

	ctx := context.Background()
	action := &driver.RecordedAction{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		ActionType: "click",
		Timestamp:  time.Now().Format(time.RFC3339Nano),
	}

	// Don't provide correlation ID - should be auto-generated
	result, err := recordingSvc.RecordActionUnified(ctx, recording.RecordActionRequest{
		SessionID: sessionID,
		Action:    action,
		PageID:    uuid.New(),
		Source:    recording.ActionSourceManual,
		// CorrelationID intentionally omitted
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have auto-generated correlation ID
	assert.NotEmpty(t, result.CorrelationID)
	assert.Contains(t, result.CorrelationID, "rec-")
	assert.Contains(t, result.CorrelationID, sessionID[:8])
}

// TestRecordingPipeline_PageEvent tests page event recording.
func TestRecordingPipeline_PageEvent(t *testing.T) {
	logger := logrus.New()
	repo := persistence.NewMockRepository()
	hub := NewTestRecordingHub(logger)
	recordingSvc := recording.NewService(repo, hub, logger, recording.ServiceConfig{})

	sessionID := "test-session-" + uuid.NewString()[:8]
	pageID := uuid.New()

	ctx := context.Background()
	event := &domain.PageEvent{
		ID:        uuid.New(),
		Type:      domain.PageEventCreated,
		PageID:    pageID,
		URL:       "https://example.com/new",
		Title:     "New Page",
		Timestamp: time.Now(),
	}

	result, err := recordingSvc.RecordPageEventUnified(ctx, recording.RecordPageEventRequest{
		SessionID:     sessionID,
		Event:         event,
		CorrelationID: "page-event-123",
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "page-event-123", result.CorrelationID)
	assert.True(t, result.Persisted)
	assert.Equal(t, event.ID, result.ActionID)
}

// TestRecordingPipeline_NilAction tests handling of nil action.
func TestRecordingPipeline_NilAction(t *testing.T) {
	logger := logrus.New()
	repo := persistence.NewMockRepository()
	hub := NewTestRecordingHub(logger)
	recordingSvc := recording.NewService(repo, hub, logger, recording.ServiceConfig{})

	sessionID := "test-session-" + uuid.NewString()[:8]

	ctx := context.Background()

	result, err := recordingSvc.RecordActionUnified(ctx, recording.RecordActionRequest{
		SessionID: sessionID,
		Action:    nil, // nil action
		PageID:    uuid.New(),
		Source:    recording.ActionSourceManual,
	})

	require.NoError(t, err) // Should not return error, but record it in result
	require.NotNil(t, result)

	// Should have validation error
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "validation", result.Errors[0].Stage)
}

// TestRecordingPipeline_DuplicateNavigate tests deduplication of navigate actions.
func TestRecordingPipeline_DuplicateNavigate(t *testing.T) {
	logger := logrus.New()
	repo := persistence.NewMockRepository()
	hub := NewTestRecordingHub(logger)
	recordingSvc := recording.NewService(repo, hub, logger, recording.ServiceConfig{})

	sessionID := "test-session-" + uuid.NewString()[:8]
	pageID := uuid.New()

	ctx := context.Background()
	now := time.Now()

	// First navigate action
	action1 := &driver.RecordedAction{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		ActionType: "navigate",
		Timestamp:  now.Format(time.RFC3339Nano),
		URL:        "https://example.com/page1",
	}

	result1, err := recordingSvc.RecordActionUnified(ctx, recording.RecordActionRequest{
		SessionID: sessionID,
		Action:    action1,
		PageID:    pageID,
		Source:    recording.ActionSourceAuto,
	})
	require.NoError(t, err)
	assert.True(t, result1.Persisted)

	// Duplicate navigate action (same URL within threshold)
	action2 := &driver.RecordedAction{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		ActionType: "navigate",
		Timestamp:  now.Add(100 * time.Millisecond).Format(time.RFC3339Nano), // Within 500ms threshold
		URL:        "https://example.com/page1",                              // Same URL
	}

	result2, err := recordingSvc.RecordActionUnified(ctx, recording.RecordActionRequest{
		SessionID: sessionID,
		Action:    action2,
		PageID:    pageID,
		Source:    recording.ActionSourceAuto,
	})
	require.NoError(t, err)
	// Duplicate should be silently skipped (no persistence)
	assert.False(t, result2.Persisted)

	// Verify only one entry in timeline
	timeline, err := recordingSvc.GetTimeline(ctx, persistence.TimelineQuery{
		SessionID: sessionID,
		Limit:     100,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, timeline.TotalCount, "Duplicate should be deduplicated")
}

// TestBroadcastResult_Metrics tests that BroadcastResult contains correct metrics.
func TestBroadcastResult_Metrics(t *testing.T) {
	logger := logrus.New()
	hub := NewTestRecordingHub(logger)

	sessionID := "test-session"
	entry := &wsHub.UnifiedTimelineEntry{
		ID:        uuid.NewString(),
		Type:      "action",
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}

	// Test with no subscribers
	result1 := hub.BroadcastRecordingEntry(sessionID, entry)
	assert.Equal(t, 0, result1.SubscriberCount)
	assert.Equal(t, 0, result1.SentCount)
	assert.Equal(t, 0, result1.DroppedCount)

	// Add a subscriber
	_ = hub.Subscribe(sessionID)

	// Test with subscriber
	result2 := hub.BroadcastRecordingEntry(sessionID, entry)
	assert.Equal(t, 1, result2.SubscriberCount)
	assert.Equal(t, 1, result2.SentCount)
	assert.Equal(t, 0, result2.DroppedCount)

	// Test with nil entry
	result3 := hub.BroadcastRecordingEntry(sessionID, nil)
	assert.Equal(t, 0, result3.SubscriberCount)
	assert.Equal(t, 0, result3.SentCount)
}
