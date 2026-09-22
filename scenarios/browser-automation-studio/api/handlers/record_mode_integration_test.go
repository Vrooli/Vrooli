package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// TestRecordingHub is a test hub that captures broadcasts for verification.
type TestRecordingHub struct {
	mu                  sync.RWMutex
	clients             map[string]chan *bastimeline.TimelineEntry
	broadcastCounts     map[string]int
	pageBroadcastCounts map[string]int
	lastEntry           map[string]*bastimeline.TimelineEntry
	logger              *logrus.Logger
}

func NewTestRecordingHub(logger *logrus.Logger) *TestRecordingHub {
	return &TestRecordingHub{
		clients:             make(map[string]chan *bastimeline.TimelineEntry),
		broadcastCounts:     make(map[string]int),
		pageBroadcastCounts: make(map[string]int),
		lastEntry:           make(map[string]*bastimeline.TimelineEntry),
		logger:              logger,
	}
}

// Subscribe adds a test client for a session.
func (h *TestRecordingHub) Subscribe(sessionID string) chan *bastimeline.TimelineEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan *bastimeline.TimelineEntry, 100)
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
func (h *TestRecordingHub) GetLastEntry(sessionID string) *bastimeline.TimelineEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastEntry[sessionID]
}

// Implement wsHub.HubInterface

func (h *TestRecordingHub) ServeWS(conn *websocket.Conn, executionID *uuid.UUID) {}

func (h *TestRecordingHub) BroadcastEnvelope(event any) {}

func (h *TestRecordingHub) BroadcastTimelineEntry(sessionID string, entry *bastimeline.TimelineEntry) wsHub.BroadcastResult {
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
func (h *TestRecordingHub) BroadcastBinaryFrame(sessionID string, jpegData []byte)                {}
func (h *TestRecordingHub) HasRecordingSubscribers(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[sessionID]
	return ok
}
func (h *TestRecordingHub) BroadcastPerfStats(sessionID string, stats any) {}
func (h *TestRecordingHub) BroadcastPageEvent(sessionID string, event any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pageBroadcastCounts[sessionID]++
}
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
func (h *TestRecordingHub) Run()                       {}
func (h *TestRecordingHub) CloseExecution(_ uuid.UUID) {}

// Compile-time interface check
var _ wsHub.HubInterface = (*TestRecordingHub)(nil)

// TestRecordingPipeline_EndToEnd tests the complete recording pipeline.

// TestRecordingPipeline_NoSubscribers tests recording when no UI is connected.

// TestRecordingPipeline_CorrelationIDGeneration tests automatic correlation ID generation.

// TestRecordingPipeline_PageEvent tests page event recording.

// TestRecordingPipeline_NilAction tests handling of nil action.

// TestRecordingPipeline_DuplicateNavigate tests deduplication of navigate actions.

// TestBroadcastResult_Metrics tests that BroadcastResult contains correct metrics.
func TestBroadcastResult_Metrics(t *testing.T) {
	logger := logrus.New()
	hub := NewTestRecordingHub(logger)

	sessionID := "test-session"
	entry := &bastimeline.TimelineEntry{Id: uuid.NewString()}

	// Test with no subscribers
	result1 := hub.BroadcastTimelineEntry(sessionID, entry)
	assert.Equal(t, 0, result1.SubscriberCount)
	assert.Equal(t, 0, result1.SentCount)
	assert.Equal(t, 0, result1.DroppedCount)

	// Add a subscriber
	_ = hub.Subscribe(sessionID)

	// Test with subscriber
	result2 := hub.BroadcastTimelineEntry(sessionID, entry)
	assert.Equal(t, 1, result2.SubscriberCount)
	assert.Equal(t, 1, result2.SentCount)
	assert.Equal(t, 0, result2.DroppedCount)

	// Test with nil entry
	result3 := hub.BroadcastTimelineEntry(sessionID, nil)
	assert.Equal(t, 0, result3.SubscriberCount)
	assert.Equal(t, 0, result3.SentCount)
}

type journalIngressService struct {
	*MockRecordModeService
	session *autosession.Session
	journal *recording.Service
}

func (s *journalIngressService) GetSession(string) (*autosession.Session, bool) {
	return s.session, s.session != nil
}

func (s *journalIngressService) AddTimelineAction(ctx context.Context, id string, action *driver.RecordedAction, page uuid.UUID) error {
	return s.journal.RecordAction(ctx, id, action, page, recording.ActionSourceManual)
}

// [REQ:BAS-RH-J06] Exercise the real HTTP ingress, not an alternate recorder API.
func TestReceiveRecordingActionRequiresCommit(t *testing.T) {
	for _, tc := range []struct {
		name               string
		saveErr            error
		missing            bool
		status, broadcasts int
	}{
		{name: "committed", status: 200, broadcasts: 1},
		{name: "storage failure", saveErr: errors.New("synthetic disk failure"), status: 503},
		{name: "unknown session", missing: true, status: 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log := logrus.New()
			repo := persistence.NewMockRepository()
			repo.AppendTimelineEntryErr = tc.saveErr
			svc := recording.NewService(repo, recording.ServiceConfig{})
			if err := svc.RegisterSession(context.Background(), "session", recording.SessionConfig{}); err != nil {
				t.Fatal(err)
			}
			sess := &autosession.Session{}
			sess.InitializePageTracking("https://example.test")
			if tc.missing {
				sess = nil
			}
			hub := NewTestRecordingHub(log)
			h := &Handler{recordModeService: &journalIngressService{MockRecordModeService: NewMockRecordModeService(), session: sess, journal: svc}, wsHub: hub, log: log}
			req := httptest.NewRequest(http.MethodPost, "/recordings/live/session/action", strings.NewReader(`{"id":"event-123","actionType":"input","timestamp":"2026-09-22T00:00:00Z","payload":{"text":"preserve"}}`))
			route := chi.NewRouteContext()
			route.URLParams.Add("sessionId", "session")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
			response := httptest.NewRecorder()
			h.ReceiveRecordingAction(response, req)
			if response.Code != tc.status {
				t.Errorf("status%d want%d: %s", response.Code, tc.status, response.Body.String())
			}
			if response.Code == http.StatusOK {
				var receipt map[string]string
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &receipt))
				require.Equal(t, "event-123", receipt["entry_id"])
			}
			if n := hub.GetBroadcastCount("session"); n != tc.broadcasts {
				t.Errorf("broadcasts%d want%d", n, tc.broadcasts)
			}
		})
	}
}

func TestHistoryNavigationRequiresRecordedOutcome(t *testing.T) {
	operations := []struct {
		name              string
		handle            func(*Handler, http.ResponseWriter, *http.Request)
		pageNotifications int
	}{
		{"reload", (*Handler).ReloadRecordingSession, 0},
		{"goBack", (*Handler).GoBackRecordingSession, 1},
		{"goForward", (*Handler).GoForwardRecordingSession, 1},
	}
	cases := []struct {
		name               string
		saveErr            error
		missing            bool
		status, broadcasts int
	}{
		{name: "committed", status: 200, broadcasts: 1},
		{name: "storage failure", saveErr: errors.New("synthetic disk failure"), status: 503},
		{name: "missing session", missing: true, status: 503},
	}
	for _, operation := range operations {
		for _, tc := range cases {
			t.Run(operation.name+"/"+tc.name, func(t *testing.T) {
				log := logrus.New()
				repo := persistence.NewMockRepository()
				repo.AppendTimelineEntryErr = tc.saveErr
				journal := recording.NewService(repo, recording.ServiceConfig{})
				require.NoError(t, journal.RegisterSession(context.Background(), "session", recording.SessionConfig{}))
				sess := &autosession.Session{}
				sess.InitializePageTracking("https://before.test")
				if tc.missing {
					sess = nil
				}
				service := &journalIngressService{MockRecordModeService: NewMockRecordModeService(), session: sess, journal: journal}
				client := service.MockClient()
				client.ReloadResponse = &driver.ReloadResponse{URL: "https://after.test", Title: "after", CanGoBack: true}
				client.GoBackResponse = &driver.GoBackResponse{URL: "https://after.test", Title: "after", CanGoBack: true}
				client.GoForwardResponse = &driver.GoForwardResponse{URL: "https://after.test", Title: "after", CanGoBack: true}
				hub := NewTestRecordingHub(log)
				h := &Handler{recordModeService: service, wsHub: hub, log: log}
				req := httptest.NewRequest(http.MethodPost, "/session/"+operation.name, strings.NewReader(`{}`))
				route := chi.NewRouteContext()
				route.URLParams.Add("sessionId", "session")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
				response := httptest.NewRecorder()
				operation.handle(h, response, req)
				require.Equal(t, tc.status, response.Code, response.Body.String())
				assert.Equal(t, tc.broadcasts, hub.GetBroadcastCount("session"))
				assert.Equal(t, tc.broadcasts*operation.pageNotifications, hub.pageBroadcastCounts["session"])
				if tc.status == 200 {
					var result map[string]interface{}
					require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
					assert.Equal(t, map[string]interface{}{"session_id": "session", "url": "https://after.test", "title": "after", "can_go_back": true, "can_go_forward": false}, result)
					entries, err := journal.GetTimeline(req.Context(), persistence.TimelineQuery{SessionID: "session"})
					require.NoError(t, err)
					require.Equal(t, 1, entries.TotalCount)
					require.Len(t, entries.Entries, 1)
					assert.Equal(t, operation.name, entries.Entries[0].Action.ActionType)
					assert.Equal(t, "https://after.test", sess.Pages().GetActivePage().URL)
				}
			})
		}
	}
}

// [REQ:BAS-RH-J06] A pull clear is a commit followed by explicit acknowledgement.
func TestPullRecordingActionsCommitBeforeAcknowledgement(t *testing.T) {
	for _, failure := range []string{"none", "commit", "ack"} {
		t.Run(failure, func(t *testing.T) {
			ctx := context.Background()
			repo := persistence.NewMockRepository()
			journal := recording.NewService(repo, recording.ServiceConfig{})
			require.NoError(t, journal.RegisterSession(ctx, "session", recording.SessionConfig{}))
			sess := &autosession.Session{}
			sess.InitializePageTracking("https://fixture.test")
			mock := NewMockRecordModeService()
			id := uuid.NewString()
			mock.MockClient().RecordedActionsResponse = &driver.GetActionsResponse{SessionID: "session", Actions: []driver.RecordedAction{{ID: id, ActionType: "click", Timestamp: "2026-09-22T00:00:00Z"}}}
			if failure == "commit" {
				repo.AppendTimelineEntryErr = errors.New("disk full")
			}
			if failure == "ack" {
				mock.MockClient().AcknowledgeRecordedActionsError = errors.New("lost acknowledgement")
			}
			h := &Handler{recordModeService: &journalIngressService{MockRecordModeService: mock, session: sess, journal: journal}, log: logrus.New()}
			invoke := func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, "/recordings/live/session/actions?clear=true", nil)
				route := chi.NewRouteContext()
				route.URLParams.Add("sessionId", "session")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
				response := httptest.NewRecorder()
				h.GetRecordedActions(response, req)
				return response
			}
			response := invoke()
			if failure == "none" {
				require.Equal(t, 200, response.Code, response.Body.String())
			} else {
				require.Equal(t, 503, response.Code, response.Body.String())
			}
			if failure == "commit" {
				require.Empty(t, mock.MockClient().AcknowledgedEntryIDs)
			} else {
				require.Equal(t, []string{id}, mock.MockClient().AcknowledgedEntryIDs)
			}
			repo.AppendTimelineEntryErr = nil
			mock.MockClient().AcknowledgeRecordedActionsError = nil
			require.Equal(t, 200, invoke().Code)
			require.Equal(t, []string{id}, mock.MockClient().AcknowledgedEntryIDs)
			timeline, err := journal.GetTimeline(ctx, persistence.TimelineQuery{SessionID: "session"})
			require.NoError(t, err)
			require.Len(t, timeline.Entries, 1, "retry must not append a second observation")
		})
	}
}
