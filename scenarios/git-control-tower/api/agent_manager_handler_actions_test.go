package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	httpx "github.com/vrooli/api-core/servertest"
)

func actionTestServer(t *testing.T, status CapabilityStatus, handler http.Handler) (*Server, func()) {
	t.Helper()
	upstream := httpx.NewHandlerServer(t, handler)
	client := &AgentManagerClient{BaseClient: BaseClient{httpClient: httpx.TestClient(), resolver: discovery.NewStaticResolver(upstream.URL), serviceName: "agent-manager"}, maxRetries: 0, retryBaseDelay: time.Millisecond}
	registry := NewCapabilityRegistry([]CapabilityDef{{ID: "agent-manager"}}, map[string]StatusChecker{"agent-manager": &FakeStatusChecker{Status: status}}, time.Minute)
	return &Server{git: NewFakeGitRunner(), agentManagerClient: client, capabilities: registry}, upstream.Close
}

func actionRequest(t *testing.T, s *Server, method, path, body string, id string, call func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r = mux.SetURLVars(r, map[string]string{"id": id})
	w := httptest.NewRecorder()
	call(w, r)
	return w
}

func TestAgentRunEventsBoundsAndProjection(t *testing.T) {
	var gotAfter, gotLimit int
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAfter = 0
		gotLimit = 0
		_, _ = fmt.Sscanf(r.URL.Query().Get("afterSequence"), "%d", &gotAfter)
		_, _ = fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &gotLimit)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events":[{"id":"evt-1","run_id":"run-a","event_type":"message","sequence":"7","message":{"text":"hello"}}]}`))
	})
	s, closeUpstream := actionTestServer(t, StatusAvailable, upstream)
	defer closeUpstream()
	w := actionRequest(t, s, http.MethodGet, "/events?afterSequence=0&limit=900", "", "run-a", s.handleAgentRunEvents)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body)
	}
	if gotAfter != 0 || gotLimit != 500 {
		t.Fatalf("forwarded bounds after=%d limit=%d", gotAfter, gotLimit)
	}
	var got AgentRunEventsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Events) != 1 || got.Events[0].ID != "evt-1" || got.Events[0].Sequence != 7 || got.Events[0].EventType != "message" {
		t.Fatalf("projection=%+v", got.Events)
	}
}

func TestAgentRunDiffProjectsFiles(t *testing.T) {
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/runs/r1/diff") {
			t.Errorf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"diff":{"content":"patch","files":[{"path":"main.go","change_type":"modified","additions":3,"deletions":1}]}}`))
	})
	s, closeUpstream := actionTestServer(t, StatusAvailable, upstream)
	defer closeUpstream()
	w := actionRequest(t, s, http.MethodGet, "/runs/r1/diff", "", "r1", s.handleAgentRunDiff)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body)
	}
	var got AgentRunDiffResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.RunID != "r1" || got.Content != "patch" || len(got.Files) != 1 || got.Files[0].Path != "main.go" || got.Files[0].Additions != 3 || got.Files[0].Deletions != 1 {
		t.Fatalf("projection=%+v", got)
	}
}

func TestAgentRunEventsRejectInvalidQueryBounds(t *testing.T) {
	for _, query := range []string{"?afterSequence=-2", "?limit=0", "?limit=nope"} {
		t.Run(query, func(t *testing.T) {
			s, closeUpstream := actionTestServer(t, StatusAvailable, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid query reached upstream") }))
			defer closeUpstream()
			w := actionRequest(t, s, http.MethodGet, "/runs/r1/events"+query, "", "r1", s.handleAgentRunEvents)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
		})
	}
}

func TestAgentRunActionsRejectUnavailableAndMissingRunID(t *testing.T) {
	handlers := []struct {
		name, path string
		call       func(*Server, http.ResponseWriter, *http.Request)
	}{
		{"events", "/runs//events", (*Server).handleAgentRunEvents},
		{"diff", "/runs//diff", (*Server).handleAgentRunDiff},
		{"continue", "/runs//continue", (*Server).handleAgentRunContinue},
		{"approve", "/runs//approve", (*Server).handleAgentRunApprove},
		{"reject", "/runs//reject", (*Server).handleAgentRunReject},
		{"stop", "/runs//stop", (*Server).handleAgentRunStop},
	}
	for _, tc := range handlers {
		for _, statusCase := range []struct {
			name      string
			available CapabilityStatus
			id        string
			want      int
		}{{"unavailable", StatusUnavailable, "r1", http.StatusServiceUnavailable}, {"missing run id", StatusAvailable, "  ", http.StatusBadRequest}} {
			t.Run(tc.name+"/"+statusCase.name, func(t *testing.T) {
				s, closeUpstream := actionTestServer(t, statusCase.available, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Errorf("unexpected upstream call") }))
				defer closeUpstream()
				w := actionRequest(t, s, http.MethodPost, tc.path, "{}", statusCase.id, func(w http.ResponseWriter, r *http.Request) { tc.call(s, w, r) })
				if w.Code != statusCase.want {
					t.Fatalf("status=%d body=%s want=%d", w.Code, w.Body, statusCase.want)
				}
			})
		}
	}
}

func TestAgentRunActionResponseProjections(t *testing.T) {
	tests := []struct {
		name, path, body, response string
		call                       func(*Server, http.ResponseWriter, *http.Request)
		check                      func(*testing.T, *httptest.ResponseRecorder)
	}{
		{"continue", "/runs/r1/continue", `{"message":"go"}`, `{"success":true,"run":{"id":"r1","status":"RUN_STATUS_RUNNING"}}`, (*Server).handleAgentRunContinue, func(t *testing.T, w *httptest.ResponseRecorder) {
			var got AgentContinueResponse
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || !got.Success || got.Run == nil || got.Run.ID != "r1" {
				t.Fatalf("projection=%s err=%v", w.Body, err)
			}
		}},
		{"approve", "/runs/r1/approve", `{}`, `{"result":{"success":true,"files_applied":2,"commit_hash":"abc","message":"done"}}`, (*Server).handleAgentRunApprove, func(t *testing.T, w *httptest.ResponseRecorder) {
			var got AgentApproveResponse
			_ = json.Unmarshal(w.Body.Bytes(), &got)
			if !got.Success || got.FilesApplied != 2 || got.CommitHash != "abc" || got.Message != "done" {
				t.Fatalf("projection=%+v", got)
			}
		}},
		{"reject", "/runs/r1/reject", `{}`, `{"status":"rejected"}`, (*Server).handleAgentRunReject, func(t *testing.T, w *httptest.ResponseRecorder) {
			var got AgentRejectResponse
			_ = json.Unmarshal(w.Body.Bytes(), &got)
			if got.Status != "rejected" {
				t.Fatalf("projection=%+v", got)
			}
		}},
		{"stop", "/runs/r1/stop", ``, `{"status":"stopped"}`, (*Server).handleAgentRunStop, func(t *testing.T, w *httptest.ResponseRecorder) {
			var got AgentStopResponse
			_ = json.Unmarshal(w.Body.Bytes(), &got)
			if got.Status != "stopped" {
				t.Fatalf("projection=%+v", got)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, tc.path) {
					t.Errorf("path=%s want suffix %s", r.URL.Path, tc.path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.response))
			})
			s, closeUpstream := actionTestServer(t, StatusAvailable, upstream)
			defer closeUpstream()
			w := actionRequest(t, s, http.MethodPost, tc.path, tc.body, "r1", func(w http.ResponseWriter, r *http.Request) { tc.call(s, w, r) })
			if w.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
			tc.check(t, w)
		})
	}
}

func TestAgentRunActionsMapOwnerErrors(t *testing.T) {
	methods := []struct {
		name, path string
		call       func(*Server, http.ResponseWriter, *http.Request)
	}{
		{"events", "/runs/r1/events", (*Server).handleAgentRunEvents}, {"diff", "/runs/r1/diff", (*Server).handleAgentRunDiff}, {"continue", "/runs/r1/continue", (*Server).handleAgentRunContinue}, {"approve", "/runs/r1/approve", (*Server).handleAgentRunApprove}, {"reject", "/runs/r1/reject", (*Server).handleAgentRunReject}, {"stop", "/runs/r1/stop", (*Server).handleAgentRunStop},
	}
	for _, tc := range methods {
		t.Run(tc.name, func(t *testing.T) {
			upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "owner failed", http.StatusInternalServerError)
			})
			s, closeUpstream := actionTestServer(t, StatusAvailable, upstream)
			defer closeUpstream()
			body := "{}"
			if tc.name == "events" || tc.name == "diff" {
				body = ""
			}
			w := actionRequest(t, s, http.MethodPost, tc.path, body, "r1", func(w http.ResponseWriter, r *http.Request) { tc.call(s, w, r) })
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
		})
	}
}
