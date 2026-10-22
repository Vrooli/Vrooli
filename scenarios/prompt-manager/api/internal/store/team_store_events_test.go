package store

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"prompt-manager/internal/sourceledger"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestHeartbeatAttemptsUseVrooliEventsInProductionStore(t *testing.T) {
	var captured *domain.EventEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			captured = &domain.EventEnvelope{}
			if err := protojson.Unmarshal(body, captured); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			if captured == nil {
				_, _ = w.Write([]byte("[]"))
				return
			}
			body, err := protojson.Marshal(captured)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte("["))
			_, _ = w.Write(body)
			_, _ = w.Write([]byte("]"))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "infra-health"), 0o755); err != nil {
		t.Fatal(err)
	}
	team := newIndependentTestTeam("infra-health", "Infrastructure health")
	team.Kind = KindTeam
	team.SchemaVersion = CurrentSchemaVersion
	if err := SaveJSON(filepath.Join(root, "teams", "infra-health", "team.json"), team); err != nil {
		t.Fatal(err)
	}
	teamStore := NewFileTeamStore(root, root, nil)
	teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
	teamStore.SetEventsEndpoint(server.URL)
	entry := &HeartbeatAttempt{ID: "attempt-1", TeamID: "infra-health", AgentID: "auditor", ProfileKey: "auditor", Status: "failed", Phase: "pre_run_failure", StartedAt: "2026-08-08T20:00:00Z", Error: "agent unavailable"}
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), entry.TeamID, entry); err != nil {
		t.Fatalf("append heartbeat attempt: %v", err)
	}
	if captured == nil || captured.GetEventType() != heartbeatAttemptEventType || captured.GetSource().GetScenario() != "prompt-manager" || captured.GetEventId() != heartbeatAttemptFactEventID(entry.ID, entry.Phase) {
		t.Fatalf("unexpected event envelope: %#v", captured)
	}
	if err := teamStore.SetHeartbeatConfig(t.Context(), entry.TeamID, entry.AgentID, &HeartbeatConfig{
		TeamID: entry.TeamID, AgentID: entry.AgentID,
		LastExecution: &HeartbeatExecResult{StartedAt: entry.StartedAt, EndedAt: entry.EndedAt, Status: entry.Status, Error: entry.Error},
	}); err != nil {
		t.Fatalf("set matching legacy projection: %v", err)
	}
	got, total, err := teamStore.ListHeartbeatAttempts(t.Context(), entry.TeamID, "", "", "", 10, 0)
	if err != nil {
		t.Fatalf("list heartbeat attempts: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].ID != entry.ID || got[0].Error != entry.Error {
		t.Fatalf("unexpected attempts: total=%d got=%+v", total, got)
	}
}

func TestHeartbeatAttemptLifecycleFactsHaveDistinctEventIDsAndProjectLatest(t *testing.T) {
	var mu sync.Mutex
	var captured []*domain.EventEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			env := &domain.EventEnvelope{}
			if err := protojson.Unmarshal(body, env); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			for _, existing := range captured {
				if existing.GetEventId() == env.GetEventId() {
					mu.Unlock()
					w.WriteHeader(http.StatusConflict)
					return
				}
			}
			captured = append(captured, env)
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			mu.Lock()
			events := append([]*domain.EventEnvelope(nil), captured...)
			mu.Unlock()
			if r.URL.Query().Get("order") != "desc" {
				http.Error(w, "expected newest-first query", http.StatusBadRequest)
				return
			}
			before := r.URL.Query().Get("before_event_id")
			if before != "" {
				cursor := -1
				for i, env := range events {
					if env.GetEventId() == before {
						cursor = i
						break
					}
				}
				if cursor < 0 {
					http.Error(w, "unknown cursor", http.StatusBadRequest)
					return
				}
				events = events[:cursor]
			}
			for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
				events[i], events[j] = events[j], events[i]
			}
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit > 0 && len(events) > limit {
				events = events[:limit]
			}
			w.Write([]byte("["))
			for i, env := range events {
				if i > 0 {
					w.Write([]byte(","))
				}
				body, err := protojson.Marshal(env)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(body)
			}
			w.Write([]byte("]"))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "infra-health"), 0o755); err != nil {
		t.Fatal(err)
	}
	teamStore := NewFileTeamStore(root, root, nil)
	teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
	teamStore.SetEventsEndpoint(server.URL)
	start := "2026-10-17T01:00:00Z"
	logicalID := "heartbeat-logical-attempt-1"
	created := &HeartbeatAttempt{ID: logicalID, TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusRunning, Phase: "run_created", StartedAt: start, RunID: "run-1"}
	terminal := *created
	terminal.Status = HeartbeatStatusFailed
	terminal.Phase = "run_terminal"
	terminal.EndedAt = "2026-10-17T01:01:00Z"
	terminal.Error = "owner run failed"
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), created.TeamID, created); err != nil {
		t.Fatalf("append run-created fact: %v", err)
	}
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), terminal.TeamID, &terminal); err != nil {
		t.Fatalf("append terminal fact: %v", err)
	}
	mu.Lock()
	if len(captured) != 2 || captured[0].GetEventId() == captured[1].GetEventId() {
		mu.Unlock()
		t.Fatalf("lifecycle facts need distinct EventIds: %#v", captured)
	}
	mu.Unlock()

	latest, total, err := teamStore.ListHeartbeatAttempts(t.Context(), "infra-health", "auditor", "", "", 10, 0)
	if err != nil || total != 1 || len(latest) != 1 || latest[0].ID != logicalID || latest[0].Status != HeartbeatStatusFailed || latest[0].Phase != "run_terminal" {
		t.Fatalf("latest logical attempt = total %d %+v err=%v", total, latest, err)
	}
	running, total, err := teamStore.ListHeartbeatAttempts(t.Context(), "infra-health", "auditor", HeartbeatStatusRunning, "", 10, 0)
	if err != nil || total != 0 || len(running) != 0 {
		t.Fatalf("running filter returned a stale lifecycle fact: total %d %+v err=%v", total, running, err)
	}
}

func TestHeartbeatAttemptPageReturnsNewestBeyondOldEventWindow(t *testing.T) {
	var mu sync.Mutex
	var captured []*domain.EventEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			env := &domain.EventEnvelope{}
			if err := protojson.Unmarshal(body, env); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			captured = append(captured, env)
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			q := r.URL.Query()
			if q.Get("type") != heartbeatAttemptEventType || q.Get("source") != "prompt-manager" || q.Get("order") != "desc" {
				http.Error(w, "incorrect event query filters", http.StatusBadRequest)
				return
			}
			mu.Lock()
			events := append([]*domain.EventEnvelope(nil), captured...)
			mu.Unlock()
			if before := q.Get("before_event_id"); before != "" {
				cursor := -1
				for i, env := range events {
					if env.GetEventId() == before {
						cursor = i
						break
					}
				}
				if cursor < 0 {
					http.Error(w, "unknown cursor", http.StatusBadRequest)
					return
				}
				events = events[:cursor]
			}
			for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
				events[i], events[j] = events[j], events[i]
			}
			limit, _ := strconv.Atoi(q.Get("limit"))
			if limit > 0 && len(events) > limit {
				events = events[:limit]
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("["))
			for i, env := range events {
				if i > 0 {
					w.Write([]byte(","))
				}
				body, err := protojson.Marshal(env)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(body)
			}
			w.Write([]byte("]"))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	teamStore := NewFileTeamStore(root, root, nil)
	teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
	teamStore.SetEventsEndpoint(server.URL)
	for i := 0; i < 120; i++ {
		entry := &HeartbeatAttempt{ID: fmt.Sprintf("old-%03d", i), TeamID: "target", AgentID: "member", Status: HeartbeatStatusCompleted, Phase: "run_terminal", StartedAt: fmt.Sprintf("2026-10-19T00:%02d:00Z", i%60)}
		if err := teamStore.AppendHeartbeatAttempt(t.Context(), entry.TeamID, entry); err != nil {
			t.Fatalf("append old attempt %d: %v", i, err)
		}
	}
	latest := &HeartbeatAttempt{ID: "current", TeamID: "target", AgentID: "member", Status: HeartbeatStatusRunning, Phase: "run_created", StartedAt: "2026-10-20T15:59:00Z", RunID: "run-current"}
	terminal := *latest
	terminal.Status = HeartbeatStatusFailed
	terminal.Phase = "run_terminal"
	terminal.EndedAt = "2026-10-20T16:00:00Z"
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), latest.TeamID, latest); err != nil {
		t.Fatal(err)
	}
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), terminal.TeamID, &terminal); err != nil {
		t.Fatal(err)
	}

	page, err := teamStore.ListHeartbeatAttemptPage(t.Context(), "target", "member", "", "", 1, 0)
	if err != nil {
		t.Fatalf("latest attempt page: %v", err)
	}
	if len(page.Attempts) != 1 || page.Attempts[0].ID != "current" || page.Attempts[0].Status != HeartbeatStatusFailed || page.Attempts[0].Phase != "run_terminal" {
		t.Fatalf("page did not project the newest lifecycle fact: %+v", page)
	}
	if !page.TotalExact || !page.HasMore || page.Total < 2 || page.ScannedEvents != 122 {
		t.Fatalf("bounded window metadata = %+v", page)
	}

	older, err := teamStore.ListHeartbeatAttemptPage(t.Context(), "target", "member", "", "", 1, 1)
	if err != nil || len(older.Attempts) != 1 || older.Attempts[0].ID == "current" {
		t.Fatalf("offset page in bounded window = %+v err=%v", older, err)
	}
}

func TestHeartbeatAttemptPageRejectsMalformedMatchingEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"eventId":"malformed","eventType":"prompt-manager.heartbeat.attempt.v1","sourceScenario":"prompt-manager"}]`)
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	teamStore := NewFileTeamStore(root, root, nil)
	teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
	teamStore.SetEventsEndpoint(server.URL)

	page, err := teamStore.ListHeartbeatAttemptPage(t.Context(), "target", "member", "", "", 1, 0)
	if err == nil || page != nil {
		t.Fatalf("malformed matching event produced a completeness claim: page=%+v err=%v", page, err)
	}
}

func TestHeartbeatAttemptPageFoldsByOccurredAtBeforeStatusFilter(t *testing.T) {
	older := heartbeatAttemptEventForTest(t, "evt-2", time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC), HeartbeatAttempt{
		ID: "attempt-out-of-order", TeamID: "target", AgentID: "member", Status: HeartbeatStatusRunning, Phase: "run_created", StartedAt: "2026-10-20T10:00:00Z",
	})
	newer := heartbeatAttemptEventForTest(t, "evt-1", time.Date(2026, 10, 20, 11, 0, 0, 0, time.UTC), HeartbeatAttempt{
		ID: "attempt-out-of-order", TeamID: "target", AgentID: "member", Status: HeartbeatStatusCompleted, Phase: "run_terminal", StartedAt: "2026-10-20T10:00:00Z", EndedAt: "2026-10-20T11:00:00Z",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("order") != "desc" || r.URL.Query().Get("type") != heartbeatAttemptEventType {
			http.Error(w, "query was not descending", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		olderJSON, _ := protojson.Marshal(older)
		newerJSON, _ := protojson.Marshal(newer)
		_, _ = fmt.Fprintf(w, "[%s,%s]", olderJSON, newerJSON)
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	teamStore := NewFileTeamStore(root, root, nil)
	teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
	teamStore.SetEventsEndpoint(server.URL)

	page, err := teamStore.ListHeartbeatAttemptPage(t.Context(), "target", "member", HeartbeatStatusCompleted, "", 10, 0)
	if err != nil {
		t.Fatalf("list completed attempts: %v", err)
	}
	if len(page.Attempts) != 1 || page.Attempts[0].ID != "attempt-out-of-order" || page.Attempts[0].Phase != "run_terminal" || !page.TotalExact {
		t.Fatalf("latest occurrence was not folded before status filtering: %+v", page)
	}
}

func heartbeatAttemptEventForTest(t *testing.T, eventID string, occurredAt time.Time, attempt HeartbeatAttempt) *domain.EventEnvelope {
	t.Helper()
	fields := map[string]any{
		"id": attempt.ID, "teamId": attempt.TeamID, "agentId": attempt.AgentID, "status": attempt.Status,
		"phase": attempt.Phase, "startedAt": attempt.StartedAt, "endedAt": attempt.EndedAt,
	}
	payload, err := structpb.NewStruct(fields)
	if err != nil {
		t.Fatal(err)
	}
	data, err := anypb.New(payload)
	if err != nil {
		t.Fatal(err)
	}
	return &domain.EventEnvelope{
		EventId: eventID, EventType: heartbeatAttemptEventType, OccurredAt: timestamppb.New(occurredAt),
		Source: &domain.EventSource{Scenario: "prompt-manager", ActorKind: "system"}, Data: data,
	}
}

func TestHeartbeatAttemptPageReportsBoundedWindowAsLowerBound(t *testing.T) {
	var mu sync.Mutex
	var captured []*domain.EventEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			env := &domain.EventEnvelope{}
			if err := protojson.Unmarshal(body, env); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			captured = append(captured, env)
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			q := r.URL.Query()
			if q.Get("order") != "desc" || q.Get("type") != heartbeatAttemptEventType || q.Get("source") != "prompt-manager" {
				http.Error(w, "incorrect event query filters", http.StatusBadRequest)
				return
			}
			mu.Lock()
			events := append([]*domain.EventEnvelope(nil), captured...)
			mu.Unlock()
			if before := q.Get("before_event_id"); before != "" {
				boundary := -1
				for i, event := range events {
					if event.GetEventId() == before {
						boundary = i
						break
					}
				}
				if boundary < 0 {
					http.Error(w, "unknown cursor", http.StatusBadRequest)
					return
				}
				events = events[:boundary]
			}
			for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
				events[i], events[j] = events[j], events[i]
			}
			limit, _ := strconv.Atoi(q.Get("limit"))
			if limit > 0 && len(events) > limit {
				events = events[:limit]
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, "[")
			for i, event := range events {
				if i > 0 {
					_, _ = io.WriteString(w, ",")
				}
				body, err := protojson.Marshal(event)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				_, _ = w.Write(body)
			}
			_, _ = io.WriteString(w, "]")
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	teamStore := NewFileTeamStore(root, root, nil)
	teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
	teamStore.SetEventsEndpoint(server.URL)
	oldMatching := &HeartbeatAttempt{ID: "old-completed", TeamID: "target", AgentID: "member", Status: HeartbeatStatusCompleted, Phase: "run_terminal", StartedAt: "2026-10-19T00:00:00Z"}
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), oldMatching.TeamID, oldMatching); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < heartbeatAttemptEventScanCap; i++ {
		entry := &HeartbeatAttempt{ID: fmt.Sprintf("new-running-%04d", i), TeamID: "target", AgentID: "member", Status: HeartbeatStatusRunning, Phase: "run_created", StartedAt: fmt.Sprintf("2026-10-20T00:%02d:00Z", i%60)}
		if err := teamStore.AppendHeartbeatAttempt(t.Context(), entry.TeamID, entry); err != nil {
			t.Fatalf("append running event %d: %v", i, err)
		}
	}

	page, err := teamStore.ListHeartbeatAttemptPage(t.Context(), "target", "member", HeartbeatStatusCompleted, "", 10, 0)
	if err != nil {
		t.Fatalf("bounded status page: %v", err)
	}
	if len(page.Attempts) != 0 || page.Total != 0 || page.TotalExact || !page.HasMore || page.ScannedEvents != heartbeatAttemptEventScanCap {
		t.Fatalf("bounded recent-window metadata = %+v; want incomplete lower bound after %d events", page, heartbeatAttemptEventScanCap)
	}
}

func TestHeartbeatAttemptsByRunUsesExactCorrelationBeyondRecentWindow(t *testing.T) {
	var mu sync.Mutex
	var events []*domain.EventEnvelope
	runID := "run-owner"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			env := &domain.EventEnvelope{}
			if err := protojson.Unmarshal(body, env); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			events = append(events, env)
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			q := r.URL.Query()
			if q.Get("type") != heartbeatAttemptEventType || q.Get("source") != "prompt-manager" || q.Get("agent_run_id") != runID || q.Get("order") != "desc" {
				http.Error(w, "exact run correlation filter was not used", http.StatusBadRequest)
				return
			}
			mu.Lock()
			matching := make([]*domain.EventEnvelope, 0, 2)
			for _, event := range events {
				if event.GetCorrelation() != nil && event.GetCorrelation().GetAgentRunId() == runID {
					matching = append(matching, event)
				}
			}
			mu.Unlock()
			for i, j := 0, len(matching)-1; i < j; i, j = i+1, j-1 {
				matching[i], matching[j] = matching[j], matching[i]
			}
			limit, _ := strconv.Atoi(q.Get("limit"))
			if limit > 0 && len(matching) > limit {
				matching = matching[:limit]
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("["))
			for i, event := range matching {
				if i > 0 {
					w.Write([]byte(","))
				}
				body, err := protojson.Marshal(event)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(body)
			}
			w.Write([]byte("]"))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	teamStore := NewFileTeamStore(root, root, nil)
	teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
	teamStore.SetEventsEndpoint(server.URL)
	for i := 0; i < 120; i++ {
		entry := &HeartbeatAttempt{ID: fmt.Sprintf("unrelated-%03d", i), TeamID: "other", AgentID: "other", RunID: fmt.Sprintf("other-run-%03d", i), Phase: "run_created"}
		if err := teamStore.AppendHeartbeatAttempt(t.Context(), entry.TeamID, entry); err != nil {
			t.Fatalf("append unrelated event %d: %v", i, err)
		}
	}
	owner := &HeartbeatAttempt{ID: "logical-owner", TeamID: "target", AgentID: "member", RunID: runID, TaskID: "task-owner", Tag: "tag-owner", ProfileKey: "profile-owner", Status: HeartbeatStatusRunning, Phase: "run_created"}
	terminal := *owner
	terminal.TaskID = "conflicting-task-owner"
	terminal.Status = HeartbeatStatusCompleted
	terminal.Phase = "run_terminal"
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), owner.TeamID, owner); err != nil {
		t.Fatal(err)
	}
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), terminal.TeamID, &terminal); err != nil {
		t.Fatal(err)
	}

	got, err := teamStore.ListHeartbeatAttemptsByRun(t.Context(), runID)
	if err != nil {
		t.Fatalf("exact run attempt lookup: %v", err)
	}
	if len(got) != 2 || got[0].ID != owner.ID || got[1].ID != owner.ID || got[0].Tag != owner.Tag || got[1].Tag != owner.Tag || got[0].ProfileKey != owner.ProfileKey || got[1].ProfileKey != owner.ProfileKey ||
		!((got[0].TaskID == owner.TaskID && got[1].TaskID == terminal.TaskID) || (got[1].TaskID == owner.TaskID && got[0].TaskID == terminal.TaskID)) {
		t.Fatalf("exact run owner facts = %+v", got)
	}

	for _, payloadRunID := range []string{"wrong-run", ""} {
		bad := &HeartbeatAttempt{ID: "mismatched-payload-" + payloadRunID, TeamID: "target", AgentID: "member", RunID: payloadRunID, Phase: "run_created"}
		if err := teamStore.AppendHeartbeatAttempt(t.Context(), bad.TeamID, bad); err != nil {
			t.Fatalf("append mismatched payload event: %v", err)
		}
		mu.Lock()
		for _, event := range events {
			if event.GetEventId() == heartbeatAttemptFactEventID(bad.ID, bad.Phase) {
				event.Correlation = &domain.EventCorrelation{AgentRunId: runID}
			}
		}
		mu.Unlock()
		if _, err := teamStore.ListHeartbeatAttemptsByRun(t.Context(), runID); err == nil {
			t.Fatalf("exact run lookup accepted envelope correlation %q with payload run_id %q", runID, payloadRunID)
		}
	}
}
