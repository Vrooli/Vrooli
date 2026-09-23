package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/domain"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
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
	lastPageEvent       map[string]*domain.PageEvent
	logger              *logrus.Logger
}

func NewTestRecordingHub(logger *logrus.Logger) *TestRecordingHub {
	return &TestRecordingHub{
		clients:             make(map[string]chan *bastimeline.TimelineEntry),
		broadcastCounts:     make(map[string]int),
		pageBroadcastCounts: make(map[string]int),
		lastEntry:           make(map[string]*bastimeline.TimelineEntry),
		lastPageEvent:       make(map[string]*domain.PageEvent),
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

func (h *TestRecordingHub) BroadcastBinaryFrame(sessionID string, jpegData []byte) {}
func (h *TestRecordingHub) HasRecordingFrameSubscribers(sessionID string) bool {
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
	if page, ok := event.(*domain.PageEvent); ok {
		h.lastPageEvent[sessionID] = page
	}
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
	session     *autosession.Session
	journal     *recording.Service
	afterCommit func()
}

func (s *journalIngressService) GetSession(string) (*autosession.Session, bool) {
	return s.session, s.session != nil
}

func (s *journalIngressService) AddTimelineAction(ctx context.Context, id string, action *driver.RecordedAction, page uuid.UUID) error {
	err := s.journal.RecordAction(ctx, id, action, page, recording.ActionSourceManual)
	if err == nil && s.afterCommit != nil {
		s.afterCommit()
	}
	return err
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

// createOwnedNavigationSession exercises driver wire ownership for API controls.
func createOwnedNavigationSession(t *testing.T, sessionID string, response any, observe func(*http.Request, map[string]any), statuses ...int) *autosession.Session {
	t.Helper()
	owner := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/start" {
			_ = json.NewEncoder(w).Encode(map[string]string{"session_id": sessionID, "lease_id": "navigation-lease"})
			return
		}
		body := map[string]any{}
		if r.Method == http.MethodGet {
			for key, values := range r.URL.Query() {
				body[key] = values[0]
			}
		} else {
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
				w.WriteHeader(400)
				return
			}
			assert.Equal(t, http.MethodPost, r.Method)
		}
		assert.Equal(t, owner.String(), body["execution_id"])
		assert.Equal(t, "navigation-lease", body["lease_id"])
		if observe != nil {
			observe(r, body)
		}
		if len(statuses) > 0 {
			w.WriteHeader(statuses[0])
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	session, err := autosession.NewManagerWithClient(client).Create(context.Background(), autosession.Spec{ExecutionID: owner, Mode: autosession.ModeRecording})
	require.NoError(t, err)
	return session
}

func TestHistoryNavigationRequiresRecordedOutcome(t *testing.T) {
	operations := []struct {
		name              string
		handle            func(*Handler, http.ResponseWriter, *http.Request)
		pageNotifications int
	}{
		{"reload", (*Handler).ReloadRecordingSession, 1},
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
		{name: "missing session", missing: true, status: 404},
	}
	for _, operation := range operations {
		for _, tc := range cases {
			t.Run(operation.name+"/"+tc.name, func(t *testing.T) {
				log := logrus.New()
				repo := persistence.NewMockRepository()
				repo.AppendTimelineEntryErr = tc.saveErr
				journal := recording.NewService(repo, recording.ServiceConfig{})
				require.NoError(t, journal.RegisterSession(context.Background(), "session", recording.SessionConfig{}))
				var calls atomic.Int32
				sess := createOwnedNavigationSession(t, "session", &driver.HistoryNavigationResponse{DriverPageID: "initial-driver-page", URL: "https://after.test", Title: "after", CanGoBack: true}, func(r *http.Request, body map[string]any) {
					calls.Add(1)
					endpoint := map[string]string{"reload": "reload", "goBack": "go-back", "goForward": "go-forward"}[operation.name]
					assert.Equal(t, "/session/session/record/"+endpoint, r.URL.Path)
				})
				sess.InitializePageTracking("https://before.test")
				sess.Pages().SetInitialPageDriverID("initial-driver-page")
				if tc.missing {
					sess = nil
				}
				service := &journalIngressService{MockRecordModeService: NewMockRecordModeService(), session: sess, journal: journal}
				hub := NewTestRecordingHub(log)
				h := &Handler{recordModeService: service, wsHub: hub, log: log}
				req := httptest.NewRequest(http.MethodPost, "/session/"+operation.name, strings.NewReader(`{}`))
				route := chi.NewRouteContext()
				route.URLParams.Add("sessionId", "session")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
				response := httptest.NewRecorder()
				operation.handle(h, response, req)
				require.Equal(t, tc.status, response.Code, response.Body.String())
				if tc.missing {
					require.Zero(t, calls.Load())
				} else {
					require.Equal(t, int32(1), calls.Load())
				}
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
	for _, failure := range []string{"none", "commit", "ack", "handoff"} {
		t.Run(failure, func(t *testing.T) {
			ctx := context.Background()
			repo := persistence.NewMockRepository()
			journal := recording.NewService(repo, recording.ServiceConfig{})
			require.NoError(t, journal.RegisterSession(ctx, "session", recording.SessionConfig{}))
			mock := NewMockRecordModeService()
			id := uuid.NewString()
			owner := uuid.New()
			acks := make(chan []string, 2)
			var failAck atomic.Bool
			var currentOwner atomic.Value
			failAck.Store(failure == "ack")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/session/start":
					var body struct {
						ExecutionID string `json:"execution_id"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					currentOwner.Store(body.ExecutionID)
					_, _ = w.Write([]byte(`{"session_id":"session","lease_id":"recording-lease"}`))
				case "/session/session/record/actions/ack":
					var body struct {
						ExecutionID string   `json:"execution_id"`
						LeaseID     string   `json:"lease_id"`
						EntryIDs    []string `json:"entry_ids"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, owner.String(), body.ExecutionID)
					assert.Equal(t, "recording-lease", body.LeaseID)
					assert.Equal(t, []string{id}, body.EntryIDs)
					timeline, err := journal.GetTimeline(ctx, persistence.TimelineQuery{SessionID: "session"})
					if assert.NoError(t, err) {
						assert.Len(t, timeline.Entries, 1, "commit must precede HTTP acknowledgement")
					}
					acks <- body.EntryIDs
					if body.ExecutionID != currentOwner.Load().(string) {
						http.Error(w, "stale lease", http.StatusNotFound)
						return
					}
					if failAck.Load() {
						http.Error(w, "lost acknowledgement", http.StatusServiceUnavailable)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"entry_ids": body.EntryIDs})
				default:
					t.Errorf("unexpected driver request %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			manager := autosession.NewManagerWithClient(client)
			sess, err := manager.Create(ctx, autosession.Spec{ExecutionID: owner, Mode: autosession.ModeRecording})
			require.NoError(t, err)
			sess.InitializePageTracking("https://fixture.test")
			mock.MockClient().RecordedActionsResponse = &driver.GetActionsResponse{SessionID: "session", Actions: []driver.RecordedAction{{ID: id, ActionType: "click", Timestamp: "2026-09-22T00:00:00Z"}}}
			if failure == "commit" {
				repo.AppendTimelineEntryErr = errors.New("disk full")
			}
			service := &journalIngressService{MockRecordModeService: mock, session: sess, journal: journal}
			if failure == "handoff" {
				service.afterCommit = func() {
					replacement, err := manager.Create(ctx, autosession.Spec{ExecutionID: uuid.New(), Mode: autosession.ModeRecording})
					require.NoError(t, err)
					service.session = replacement
				}
			}
			h := &Handler{recordModeService: service, log: logrus.New()}
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
				require.Empty(t, acks)
			} else {
				require.Len(t, acks, 1)
				require.Equal(t, []string{id}, <-acks)
			}
			if failure == "handoff" {
				timeline, err := journal.GetTimeline(ctx, persistence.TimelineQuery{SessionID: "session"})
				require.NoError(t, err)
				require.Len(t, timeline.Entries, 1, "lease handoff must retain the committed entry")
				return
			}
			repo.AppendTimelineEntryErr = nil
			failAck.Store(false)
			require.Equal(t, 200, invoke().Code)
			require.Len(t, acks, 1)
			require.Equal(t, []string{id}, <-acks)
			timeline, err := journal.GetTimeline(ctx, persistence.TimelineQuery{SessionID: "session"})
			require.NoError(t, err)
			require.Len(t, timeline.Entries, 1, "retry must not append a second observation")
		})
	}
}

type navigationCompletionService struct {
	*journalIngressService
	current atomic.Pointer[autosession.Session]
}

func (s *navigationCompletionService) GetSession(string) (*autosession.Session, bool) {
	current := s.current.Load()
	return current, current != nil
}

// [REQ:BAS-RH-J03] [REQ:BAS-RH-J17] A completed browser effect belongs to its original page and Session.
func TestNavigationCompletionKeepsOriginalOwnership(t *testing.T) {
	operations := []struct {
		name    string
		handle  func(*Handler, http.ResponseWriter, *http.Request)
		journal bool
	}{
		{"navigate", (*Handler).NavigateRecordingSession, false},
		{"reload", (*Handler).ReloadRecordingSession, true},
		{"go-back", (*Handler).GoBackRecordingSession, true},
		{"go-forward", (*Handler).GoForwardRecordingSession, true},
	}
	for _, op := range operations {
		for _, change := range []string{"none", "active tab", "session", "missing page receipt", "unknown page receipt", "session after commit"} {
			if !op.journal && change == "session after commit" {
				continue
			}
			t.Run(op.name+"/"+change, func(t *testing.T) {
				repo := persistence.NewMockRepository()
				journal := recording.NewService(repo, recording.ServiceConfig{})
				require.NoError(t, journal.RegisterSession(context.Background(), "session", recording.SessionConfig{}))
				service := &navigationCompletionService{journalIngressService: &journalIngressService{MockRecordModeService: NewMockRecordModeService(), journal: journal}}
				replacement := &autosession.Session{}
				replacement.InitializePageTracking("https://replacement.test")
				replacement.Pages().SetInitialPageDriverID("original-driver-page")
				receiptID := "original-driver-page"
				if change == "missing page receipt" {
					receiptID = ""
				}
				if change == "unknown page receipt" {
					receiptID = "foreign-page"
				}
				var original *autosession.Session
				otherID := uuid.New()
				original = createOwnedNavigationSession(t, "session", map[string]any{"url": "https://completed.test", "title": "completed", "driver_page_id": receiptID, "favicon_url": "https://completed.test/icon.svg"}, func(r *http.Request, _ map[string]any) {
					assert.Equal(t, "/session/session/record/"+op.name, r.URL.Path)
					switch change {
					case "active tab":
						assert.NoError(t, original.Pages().SetActivePage(otherID))
					case "session":
						service.current.Store(replacement)
					}
				})
				original.InitializePageTracking("https://original.test")
				original.Pages().SetInitialPageDriverID("original-driver-page")
				originalID := original.Pages().GetInitialPageID()
				original.Pages().AddPage(&domain.Page{ID: otherID, URL: "https://other.test", Title: "other", Status: domain.PageStatusActive})
				service.current.Store(original)
				if change == "session after commit" {
					service.afterCommit = func() { service.current.Store(replacement) }
				}
				log := logrus.New()
				hub := NewTestRecordingHub(log)
				h := &Handler{recordModeService: service, wsHub: hub, log: log}
				req := httptest.NewRequest(http.MethodPost, "/session/"+op.name, strings.NewReader(`{"url":"https://completed.test"}`))
				route := chi.NewRouteContext()
				route.URLParams.Add("sessionId", "session")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
				response := httptest.NewRecorder()
				op.handle(h, response, req)
				allowed := change == "none" || change == "active tab"
				status := http.StatusServiceUnavailable
				if allowed {
					status = http.StatusOK
				}
				require.Equal(t, status, response.Code, response.Body.String())
				other, ok := original.Pages().GetPage(otherID)
				require.True(t, ok)
				assert.Equal(t, "https://other.test", other.URL, "late completion modified the newly selected tab")
				assert.Equal(t, "https://replacement.test", replacement.Pages().GetActivePage().URL, "late completion modified a replacement Session")
				timeline, err := journal.GetTimeline(context.Background(), persistence.TimelineQuery{SessionID: "session"})
				require.NoError(t, err)
				if op.journal && (allowed || change == "session after commit") {
					require.Len(t, timeline.Entries, 1)
					assert.Equal(t, originalID, timeline.Entries[0].Action.PageID)
				} else {
					assert.Empty(t, timeline.Entries)
				}
				if allowed {
					originalPage, ok := original.Pages().GetPage(originalID)
					require.True(t, ok)
					assert.Equal(t, "https://completed.test", originalPage.URL)
					assert.Equal(t, "https://completed.test/icon.svg", originalPage.FaviconURL)
					require.NotNil(t, hub.lastPageEvent["session"])
					assert.Equal(t, originalID, hub.lastPageEvent["session"].PageID)
					require.NotNil(t, hub.lastPageEvent["session"].FaviconURL)
					assert.Equal(t, "https://completed.test/icon.svg", *hub.lastPageEvent["session"].FaviconURL)
				} else {
					assert.Zero(t, hub.GetBroadcastCount("session"), "late completion leaked into the replacement subscriber stream")
					assert.Zero(t, hub.pageBroadcastCounts["session"])
				}
			})
		}
	}
}

// [REQ:BAS-RH-J03] Recording activation cannot relabel the initial tab or duplicate a registered page.
func TestPageCallbacksPreserveRegisteredPageIdentity(t *testing.T) {
	for _, eventType := range []string{"initial", "created"} {
		t.Run(eventType, func(t *testing.T) {
			sess := &autosession.Session{}
			sess.InitializePageTracking("https://original.test")
			pages := sess.Pages()
			pages.SetInitialPageDriverID("initial-driver")
			initialID := pages.GetInitialPageID()
			second := pages.AddPage(&domain.Page{DriverPageID: "second-driver", URL: "https://second.test"})
			service := NewMockRecordModeService()
			service.OwnedSessions = map[string]*autosession.Session{"session": sess}
			log := logrus.New()
			hub := NewTestRecordingHub(log)
			h := &Handler{recordModeService: service, wsHub: hub, log: log}
			invoke := func(driverID string) *httptest.ResponseRecorder {
				body, err := json.Marshal(domain.DriverPageEvent{EventType: eventType, DriverPageID: driverID, URL: "https://second.test", Title: "second"})
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, "/session/page-event", strings.NewReader(string(body)))
				route := chi.NewRouteContext()
				route.URLParams.Add("sessionId", "session")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
				response := httptest.NewRecorder()
				h.ReceivePageEvent(response, req)
				return response
			}
			for i := 0; i < 2; i++ {
				response := invoke("second-driver")
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			}
			require.Equal(t, 2, pages.PageCount())
			require.Equal(t, initialID, *pages.GetPageIDByDriverID("initial-driver"))
			require.Equal(t, second.ID, *pages.GetPageIDByDriverID("second-driver"))
			initial, _ := pages.GetPage(initialID)
			require.Equal(t, "https://original.test", initial.URL)
			if eventType == "initial" {
				require.Equal(t, second.ID, pages.GetActivePageID())
			} else {
				require.Equal(t, second.ID, hub.lastPageEvent["session"].PageID)
			}
			missing := invoke("")
			require.Equal(t, http.StatusBadRequest, missing.Code)
			require.Equal(t, 2, pages.PageCount())
		})
	}
}

// [REQ:BAS-RH-J03] Close success reflects the browser, and repeated observations commit once.
func TestCloseRecordingPageBrowserReceipt(t *testing.T) {
	for _, fault := range []string{"none", "driver", "wrong receipt", "missing selection", "journal"} {
		t.Run(fault, func(t *testing.T) {
			ctx := context.Background()
			var closeCalls atomic.Int32
			executionID := uuid.New()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/session/start" {
					_, _ = w.Write([]byte(`{"session_id":"tab-close","lease_id":"close-lease","active_page_id":"initial"}`))
					return
				}
				assert.Equal(t, "/session/tab-close/record/close-page", r.URL.Path)
				closeCalls.Add(1)
				var body map[string]string
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, map[string]string{"execution_id": executionID.String(), "lease_id": "close-lease", "page_id": "target"}, body)
				if fault == "driver" {
					w.WriteHeader(503)
					_, _ = w.Write([]byte(`{"error":"close failed"}`))
					return
				}
				closed := "target"
				if fault == "wrong receipt" {
					closed = "unrelated"
				}
				selected := "initial"
				if fault == "missing selection" {
					selected = ""
				}
				_ = json.NewEncoder(w).Encode(map[string]string{"closed_page_id": closed, "active_page_id": selected})
			}))
			defer server.Close()
			client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
			require.NoError(t, err)
			manager := autosession.NewManagerWithClient(client)
			owner, err := manager.Create(ctx, autosession.Spec{ExecutionID: executionID, Mode: autosession.ModeRecording})
			require.NoError(t, err)
			owner.InitializePageTracking("https://initial.test")
			page := owner.Pages().AddPage(&domain.Page{DriverPageID: "target", URL: "https://target.test"})
			require.NoError(t, owner.Pages().SetActivePage(page.ID))
			repo := persistence.NewMockRepository()
			journal := recording.NewService(repo, recording.ServiceConfig{})
			require.NoError(t, journal.RegisterSession(ctx, "tab-close", recording.SessionConfig{}))
			if fault == "journal" {
				repo.AppendTimelineEntryErr = errors.New("journal unavailable")
			}
			hub := NewMockHub()
			h := &Handler{recordModeService: livecapture.NewServiceWithManager(manager, logrus.New(), journal), wsHub: hub, log: logrus.New()}
			router := chi.NewRouter()
			router.Post("/{sessionId}/pages/{pageId}/close", h.CloseRecordingPage)
			router.Post("/{sessionId}/page-event", h.ReceivePageEvent)
			closePage := func() *httptest.ResponseRecorder {
				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, httptest.NewRequest("POST", "/tab-close/pages/"+page.ID.String()+"/close", nil))
				return rr
			}
			rr := closePage()
			assert.Equal(t, int32(1), closeCalls.Load(), "browser must receive the owned close")
			current, _ := owner.Pages().GetPage(page.ID)
			if fault == "driver" || fault == "wrong receipt" || fault == "missing selection" {
				assert.Equal(t, 503, rr.Code, rr.Body.String())
				assert.Equal(t, domain.PageStatusActive, current.Status)
				assert.Equal(t, page.ID, owner.Pages().GetActivePageID())
				return
			}
			assert.Equal(t, domain.PageStatusClosed, current.Status)
			if fault == "journal" {
				assert.Equal(t, 503, rr.Code)
				repo.AppendTimelineEntryErr = nil
				rr = closePage()
			}
			require.Equal(t, 200, rr.Code, rr.Body.String())
			var reply map[string]string
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &reply))
			assert.Equal(t, owner.Pages().GetInitialPageID().String(), reply["activePageId"])
			callback := httptest.NewRecorder()
			router.ServeHTTP(callback, httptest.NewRequest("POST", "/tab-close/page-event", strings.NewReader(`{"driverPageId":"target","eventType":"closed"}`)))
			require.Equal(t, 200, callback.Code, callback.Body.String())
			require.Equal(t, 200, closePage().Code)
			assert.Equal(t, int32(1), closeCalls.Load(), "a committed closure is not a new browser command")
			entries, err := journal.GetTimeline(ctx, persistence.TimelineQuery{SessionID: "tab-close"})
			require.NoError(t, err)
			require.Len(t, entries.Entries, 1, "callback and retry share one committed closure")
			assert.Equal(t, page.ID, entries.Entries[0].PageID)
		})
	}
}

// [REQ:BAS-RH-J03] Preview tab creation returns a canonical page before callbacks exist.
func TestCreateRecordingPageCanonicalReceipt(t *testing.T) {
	executionID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/start" {
			_, _ = w.Write([]byte(`{"session_id":"admission","lease_id":"lease","active_page_id":"initial"}`))
			return
		}
		assert.Equal(t, "/session/admission/record/new-page", r.URL.Path)
		var body map[string]string
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]string{"execution_id": executionID.String(), "lease_id": "lease", "url": "https://requested.test"}, body)
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"driver_page_id":"created","url":"https://actual.test","title":"Actual"}`))
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	manager := autosession.NewManagerWithClient(client)
	owner, err := manager.Create(context.Background(), autosession.Spec{ExecutionID: executionID, Mode: autosession.ModeRecording})
	require.NoError(t, err)
	owner.InitializePageTracking("https://original.test")
	h := &Handler{recordModeService: livecapture.NewServiceWithManager(manager, logrus.New(), nil), log: logrus.New()}
	router := chi.NewRouter()
	router.Post("/{sessionId}/pages", h.CreateRecordingPage)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest("POST", "/admission/pages", strings.NewReader(`{"url":"https://requested.test"}`)))
	require.Equal(t, 201, rr.Code, rr.Body.String())
	var receipt struct {
		Page         *domain.Page `json:"page"`
		ActivePageID string       `json:"activePageId"`
		DriverPageID string       `json:"driverPageId"`
		URL          string       `json:"url"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &receipt))
	require.NotNil(t, receipt.Page)
	assert.Equal(t, owner.Pages().GetActivePageID().String(), receipt.ActivePageID)
	assert.Equal(t, receipt.ActivePageID, receipt.Page.ID.String())
	assert.Equal(t, "admission", receipt.Page.SessionID)
	assert.Equal(t, "https://actual.test", receipt.Page.URL)
	assert.Equal(t, "Actual", receipt.Page.Title)
	assert.Equal(t, domain.PageStatusActive, receipt.Page.Status)
	assert.False(t, receipt.Page.CreatedAt.IsZero())
	assert.Equal(t, "created", receipt.DriverPageID)
	assert.Equal(t, receipt.Page.URL, receipt.URL)
}

// [REQ:BAS-RH-J05] Exercise the actual HTTP driver wire, not a typed mock.
func TestLiveFrameBridgePreservesDriverPixels(t *testing.T) {
	executionID := uuid.New()
	frame := map[string]any{
		"source":     map[string]string{"session_id": "preview", "execution_id": executionID.String(), "lease_id": "preview-lease", "page_id": "driver-page"},
		"session_id": "preview", "mime": "image/jpeg", "image": "data:image/jpeg;base64,/9j/2Q==",
		"width": float64(640), "height": float64(480), "captured_at": "2026-09-23T03:00:00Z",
		"content_hash": "fixture-hash", "page_title": "Independent fixture", "page_url": "https://fixture.test",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session/start" {
			_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "preview", "lease_id": "preview-lease"})
			return
		}
		assert.Equal(t, "driver-page", r.URL.Query().Get("page_id"))
		assert.Equal(t, "/session/preview/record/frame", r.URL.Path)
		assert.Equal(t, "55", r.URL.Query().Get("quality"))
		_ = json.NewEncoder(w).Encode(frame)
	}))
	defer server.Close()
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	manager := autosession.NewManagerWithClient(client)
	owner, err := manager.Create(context.Background(), autosession.Spec{ExecutionID: executionID, Mode: autosession.ModeRecording})
	require.NoError(t, err)
	owner.InitializePageTracking("https://fixture.test")
	owner.Pages().SetInitialPageDriverID("driver-page")
	h := &Handler{recordModeService: livecapture.NewServiceWithManager(manager, logrus.New(), nil), log: logrus.New()}
	router := chi.NewRouter()
	router.Get("/{sessionId}/frame", h.GetRecordingFrame)
	for _, etag := range []string{"", `"fixture-hash"`} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/preview/frame?quality=55", nil)
		req.Header.Set("If-None-Match", etag)
		router.ServeHTTP(rr, req)
		assert.Equal(t, `"fixture-hash"`, rr.Header().Get("ETag"))
		if etag != "" {
			assert.Equal(t, http.StatusNotModified, rr.Code)
			assert.Empty(t, rr.Body.String())
			continue
		}
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
		var actual map[string]any
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &actual))
		expected := map[string]any{}
		for key, value := range frame {
			if key != "source" {
				expected[key] = value
			}
		}
		expected["page_id"] = owner.Pages().GetActivePageID().String()
		assert.Equal(t, expected, actual)
	}
}

// [REQ:BAS-RH-J03] UI page intent survives both transport boundaries.
func TestNavigationPagePrecondition(t *testing.T) {
	operations := []struct {
		name   string
		handle func(*Handler, http.ResponseWriter, *http.Request)
	}{
		{"navigate", (*Handler).NavigateRecordingSession},
		{"reload", (*Handler).ReloadRecordingSession},
		{"go-back", (*Handler).GoBackRecordingSession},
		{"go-forward", (*Handler).GoForwardRecordingSession},
	}
	for _, op := range operations {
		for _, change := range []string{"active explicit", "active implicit", "inactive", "unknown", "malformed", "closed", "missing mapping", "driver selection race"} {
			t.Run(op.name+"/"+change, func(t *testing.T) {
				repo := persistence.NewMockRepository()
				journal := recording.NewService(repo, recording.ServiceConfig{})
				require.NoError(t, journal.RegisterSession(context.Background(), "session", recording.SessionConfig{}))
				service := &navigationCompletionService{journalIngressService: &journalIngressService{MockRecordModeService: NewMockRecordModeService(), journal: journal}}
				var calls atomic.Int32
				driverStatus := http.StatusOK
				if change == "driver selection race" {
					driverStatus = http.StatusConflict
				}
				owner := createOwnedNavigationSession(t, "session", map[string]any{"driver_page_id": "red-driver-page", "url": "https://destination.test", "title": "Destination"}, func(r *http.Request, body map[string]any) {
					calls.Add(1)
					assert.Equal(t, "/session/session/record/"+op.name, r.URL.Path)
					if change == "active implicit" {
						assert.NotContains(t, body, "expected_page_id")
					} else {
						assert.Equal(t, "red-driver-page", body["expected_page_id"])
					}
				}, driverStatus)
				owner.InitializePageTracking("https://red.test")
				if change != "missing mapping" {
					owner.Pages().SetInitialPageDriverID("red-driver-page")
				}
				pageID := owner.Pages().GetInitialPageID()
				requested := pageID.String()
				blue := owner.Pages().AddPage(&domain.Page{URL: "https://blue.test", Title: "Blue", DriverPageID: "blue-driver-page"})
				switch change {
				case "inactive":
					require.NoError(t, owner.Pages().SetActivePage(blue.ID))
				case "unknown":
					requested = uuid.NewString()
				case "malformed":
					requested = "not-a-page-id"
				case "closed":
					_, err := owner.Pages().ClosePage(pageID)
					require.NoError(t, err)
				}
				service.current.Store(owner)
				log := logrus.New()
				h := &Handler{recordModeService: service, wsHub: NewTestRecordingHub(log), log: log}
				body := map[string]string{"url": "https://destination.test"}
				if change != "active implicit" {
					body["page_id"] = requested
				}
				encoded, err := json.Marshal(body)
				require.NoError(t, err)
				req := httptest.NewRequest(http.MethodPost, "/session/"+op.name, strings.NewReader(string(encoded)))
				route := chi.NewRouteContext()
				route.URLParams.Add("sessionId", "session")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
				response := httptest.NewRecorder()
				op.handle(h, response, req)
				expectedStatus := http.StatusConflict
				expectedCalls := 0
				switch change {
				case "active explicit", "active implicit":
					expectedStatus = http.StatusOK
					expectedCalls = 1
				case "malformed":
					expectedStatus = http.StatusBadRequest
				case "missing mapping":
					expectedStatus = http.StatusServiceUnavailable
				case "driver selection race":
					expectedCalls = 1
				}
				assert.Equal(t, expectedStatus, response.Code, response.Body.String())
				assert.Equal(t, int32(expectedCalls), calls.Load(), "rejected intent must not reach the driver")
				remaining, ok := owner.Pages().GetPage(blue.ID)
				require.True(t, ok)
				assert.Equal(t, "https://blue.test", remaining.URL)
			})
		}
	}
}

// [REQ:BAS-RH-J04] Tab metadata comes from the browser, with explicit absence semantics.
func TestBrowserPageFaviconMetadata(t *testing.T) {
	sess := &autosession.Session{}
	sess.InitializePageTracking("https://fixture.test/one")
	pages := sess.Pages()
	pages.SetInitialPageDriverID("driver")
	service := NewMockRecordModeService()
	service.OwnedSessions = map[string]*autosession.Session{"session": sess}
	log := logrus.New()
	hub := NewTestRecordingHub(log)
	h := &Handler{recordModeService: service, wsHub: hub, log: log}
	observe := func(event map[string]any) map[string]any {
		event["driverPageId"] = "driver"
		body, err := json.Marshal(event)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/session/page-event", strings.NewReader(string(body)))
		route := chi.NewRouteContext()
		route.URLParams.Add("sessionId", "session")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
		response := httptest.NewRecorder()
		h.ReceivePageEvent(response, req)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		page, ok := pages.GetPage(pages.GetInitialPageID())
		require.True(t, ok)
		encoded, err := json.Marshal(page)
		require.NoError(t, err)
		var snapshot map[string]any
		require.NoError(t, json.Unmarshal(encoded, &snapshot))
		return snapshot
	}
	icon := "https://fixture.test/custom.svg"
	initial := observe(map[string]any{"eventType": "initial", "url": "https://fixture.test/one", "faviconUrl": icon})
	assert.Equal(t, icon, initial["faviconUrl"])
	same := observe(map[string]any{"eventType": "navigated", "url": "https://fixture.test/one", "title": "New title"})
	assert.Equal(t, icon, same["faviconUrl"])
	changed := observe(map[string]any{"eventType": "navigated", "url": "https://fixture.test/two"})
	assert.Empty(t, changed["faviconUrl"])
	updated := observe(map[string]any{"eventType": "navigated", "url": "https://fixture.test/two", "faviconUrl": icon})
	assert.Equal(t, icon, updated["faviconUrl"])
	cleared := observe(map[string]any{"eventType": "navigated", "url": "https://fixture.test/two", "faviconUrl": ""})
	assert.Empty(t, cleared["faviconUrl"])
	event, err := json.Marshal(hub.lastPageEvent["session"])
	require.NoError(t, err)
	assert.Contains(t, string(event), `"faviconUrl":""`)
}

// [REQ:BAS-RH-J03] History observations must belong to the requested selected page.
func TestHistoryReadPageOwnership(t *testing.T) {
	for _, stack := range []bool{false, true} {
		for _, fault := range []string{"explicit", "implicit", "stale", "missing session", "handoff", "selection during read"} {
			t.Run(fmt.Sprintf("stack=%t/%s", stack, fault), func(t *testing.T) {
				service := &navigationCompletionService{journalIngressService: &journalIngressService{MockRecordModeService: NewMockRecordModeService()}}
				var calls atomic.Int32
				var owner *autosession.Session
				var blueID uuid.UUID
				owner = createOwnedNavigationSession(t, "read-session", map[string]any{"session_id": "read-session", "url": "https://red.test", "can_go_back": true, "back_stack": []any{}}, func(r *http.Request, body map[string]any) {
					calls.Add(1)
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, "red-driver", body["expected_page_id"])
					if fault == "handoff" {
						service.current.Store(nil)
					}
					if fault == "selection during read" {
						require.NoError(t, owner.Pages().SetActivePage(blueID))
					}
				})
				owner.InitializePageTracking("https://red.test")
				owner.Pages().SetInitialPageDriverID("red-driver")
				pageID := owner.Pages().GetActivePageID()
				blueID = owner.Pages().AddPage(&domain.Page{URL: "https://blue.test", DriverPageID: "blue-driver"}).ID
				service.current.Store(owner)
				if fault == "stale" {
					require.NoError(t, owner.Pages().SetActivePage(blueID))
				}
				if fault == "missing session" {
					service.current.Store(nil)
				}
				query := "?page_id=" + pageID.String()
				if fault == "implicit" {
					query = ""
				}
				req := httptest.NewRequest(http.MethodGet, "/read"+query, nil)
				route := chi.NewRouteContext()
				route.URLParams.Add("sessionId", "read-session")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
				out := httptest.NewRecorder()
				h := &Handler{recordModeService: service, log: logrus.New()}
				if stack {
					h.GetNavigationStack(out, req)
				} else {
					h.GetNavigationState(out, req)
				}
				want, wantCalls := http.StatusConflict, int32(1)
				switch fault {
				case "explicit", "implicit":
					want = http.StatusOK
				case "stale":
					wantCalls = 0
				case "missing session":
					want = http.StatusNotFound
					wantCalls = 0
				}
				assert.Equal(t, want, out.Code, out.Body.String())
				assert.Equal(t, wantCalls, calls.Load())
			})
		}
	}
}

// [REQ:BAS-RH-J03] Resize reports actual dimensions without retargeting stale page intent.
func TestViewportPageOwnershipAndReceipt(t *testing.T) {
	for _, fault := range []string{"explicit", "implicit", "stale", "handoff"} {
		t.Run(fault, func(t *testing.T) {
			service := &navigationCompletionService{journalIngressService: &journalIngressService{MockRecordModeService: NewMockRecordModeService()}}
			var calls atomic.Int32
			owner := createOwnedNavigationSession(t, "viewport-session", map[string]any{"session_id": "viewport-session", "driver_page_id": "red-driver", "width": 800, "height": 600}, func(r *http.Request, body map[string]any) {
				calls.Add(1)
				assert.Equal(t, "/session/viewport-session/record/viewport", r.URL.Path)
				assert.Equal(t, "red-driver", body["expected_page_id"])
				if fault == "handoff" {
					service.current.Store(nil)
				}
			})
			owner.InitializePageTracking("https://red.test")
			owner.Pages().SetInitialPageDriverID("red-driver")
			pageID := owner.Pages().GetActivePageID()
			blue := owner.Pages().AddPage(&domain.Page{URL: "https://blue.test", DriverPageID: "blue-driver"})
			if fault == "stale" {
				require.NoError(t, owner.Pages().SetActivePage(blue.ID))
			}
			service.current.Store(owner)
			body := map[string]any{"width": 800, "height": 600}
			if fault != "implicit" {
				body["page_id"] = pageID.String()
			}
			encoded, err := json.Marshal(body)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/viewport", strings.NewReader(string(encoded)))
			route := chi.NewRouteContext()
			route.URLParams.Add("sessionId", "viewport-session")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
			out := httptest.NewRecorder()
			h := &Handler{recordModeService: service, log: logrus.New()}
			h.UpdateRecordingViewport(out, req)
			if fault == "explicit" || fault == "implicit" {
				require.Equal(t, 200, out.Code, out.Body.String())
				var got map[string]any
				require.NoError(t, json.Unmarshal(out.Body.Bytes(), &got))
				assert.Equal(t, float64(800), got["width"])
				assert.Equal(t, float64(600), got["height"])
			} else {
				assert.Equal(t, 409, out.Code, out.Body.String())
			}
			expectedCalls := int32(1)
			if fault == "stale" {
				expectedCalls = 0
			}
			assert.Equal(t, expectedCalls, calls.Load())
		})
	}
}
