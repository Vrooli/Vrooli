package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"prompt-manager/internal/store"

	amapi "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestTeamRunAccountingJoinsRecordedIDsAndDeduplicatesResumes(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team-a", "Team A")); err != nil {
		t.Fatal(err)
	}
	for i, row := range []struct{ team, member, run string }{
		{"team-a", "lead", "resumed"}, {"team-a", "lead", "resumed"},
		{"team-a", "worker", "imported"}, {"team-a", "worker", "missing"},
		{"team-b", "lead", "foreign"}, {"team-a", "lead", ""},
	} {
		if err := teams.AppendHeartbeatAttempt(ctx, row.team, &store.HeartbeatAttempt{
			ID: time.Unix(int64(i), 0).String(), TeamID: row.team, AgentID: row.member, RunID: row.run,
			StartedAt: "2026-09-12T10:00:00Z", Status: "completed",
		}); err != nil {
			t.Fatal(err)
		}
	}
	h.runRegistry = NewRunRegistry(t.TempDir())
	h.runRegistry.Register("team-a", "lead", "resumed", time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), nil)
	owner := map[string]*ampb.Run{
		"resumed": {Id: "resumed", Status: ampb.RunStatus_RUN_STATUS_COMPLETE, HarnessKind: "codex", SessionId: "session",
			RequestedModel: "requested", ActualModel: "actual", TerminalClass: "verdict", StopReason: "blocked",
			Summary: &ampb.RunSummary{TokensUsed: 120}},
		"imported": {Id: "imported", Status: ampb.RunStatus_RUN_STATUS_COMPLETE, ImportSourceHarness: "codex", ImportSourceSessionId: "session",
			ActualModel: "actual", TerminalClass: "verdict", StopReason: "blocked", Summary: &ampb.RunSummary{TokensUsed: 120}},
	}
	reads := make(chan string, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/api/v1/runs/") {
			t.Errorf("expected exact owner run read, got %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		id := r.URL.Path[len("/api/v1/runs/"):]
		reads <- id
		if owner[id] == nil {
			http.Error(w, "owner unavailable", http.StatusNotFound)
			return
		}
		wire, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(&amapi.GetRunResponse{Run: owner[id]})
		if err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(wire)
	}))
	defer server.Close()
	h.agentClient = newTestClient(t, server)
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team-a&start=2026-09-12T00:00:00Z&end=2026-09-13T00:00:00Z", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]float64{"knownRuns": 3, "observedRuns": 2, "unavailableRuns": 1, "unqueriedRuns": 0, "observedExecutions": 1, "duplicateExecutions": 1} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v", key, got[key], want)
		}
	}
	if _, ok := got["actualModels"]; !ok {
		t.Fatalf("missing accounting projection: %s", w.Body.String())
	}
	if got["actualModels"].(map[string]any)["actual"] != float64(1) {
		t.Fatal("models not deduplicated")
	}
	if got["terminalReasons"].(map[string]any)["blocked"] != float64(1) {
		t.Fatal("blocked verdict became accepted completion")
	}
	usage := got["usage"].(map[string]any)
	if usage["tokens"] != nil || usage["costUSD"] != nil || usage["qualifiedRuns"] != float64(0) || usage["reportedTokenRuns"] != float64(2) {
		t.Fatalf("unqualified summary became qualified usage: %v", usage)
	}
	if len(reads) != 3 {
		t.Fatalf("expected one read per exact owner id, got %d", len(reads))
	}
}

func TestTeamRunAccountingUsesRecordAfterActiveRegistryRetires(t *testing.T) {
	h, teams, agents, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	if err := teams.SetHeartbeatConfig(ctx, "team", "lead", &store.HeartbeatConfig{TeamID: "team", AgentID: "lead"}); err != nil {
		t.Fatal(err)
	}
	registry := NewRunRegistry(t.TempDir())
	e := newTestExecutor(t, teams, agents, nil, t.TempDir(), registry, nil)
	wake := &SupervisionWake{ID: "wake", CreatedAt: time.Now().UTC().Add(-time.Hour)}
	old := &Run{ID: "original", Status: "running"}
	if err := e.recordSupervisionExecution(ctx, "team", "lead", wake, old); err != nil {
		t.Fatal(err)
	}
	old.Status = "failed"
	if err := e.recordSupervisionExecution(ctx, "team", "lead", wake, old); err != nil {
		t.Fatal(err)
	}
	if registry.Count() != 0 {
		t.Fatal("fixture has not retired active run")
	}
	// The current owner has resumed that same identity. Local terminal state
	// and repeated declaration rows must neither win nor count another run.
	h.runRegistry = registry
	h.agentClient = newMockAgentClient().WithGetRunResponse("original", &Run{ID: "original", Status: "RUN_STATUS_RUNNING", ActualModel: "actual"})
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team", nil))
	var got teamRunAccounting
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KnownRuns != 1 || got.ObservedExecutions != 1 || got.RuntimeStates["running"] != 1 || got.RuntimeStates["failed"] != 0 {
		t.Fatalf("local history overrode current resumed owner: %+v", got)
	}
}

func TestTeamRunAccountingBoundsOwnerReadsAndKeepsMissingIdentity(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if err := teams.AppendHeartbeatAttempt(ctx, "team", &store.HeartbeatAttempt{ID: fmt.Sprint(i), TeamID: "team", AgentID: "lead",
			RunID: fmt.Sprintf("run-%d", i), StartedAt: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)}); err != nil {
			t.Fatal(err)
		}
	}
	client := newMockAgentClient().WithGetRunResponse("run-0", &Run{ID: "wrong-id", Status: "complete"})
	h.agentClient = client
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team&limit=1", nil))
	var got teamRunAccounting
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KnownRuns != 4 || got.ObservedRuns != 0 || got.UnavailableRuns != 1 || got.UnqueriedRuns != 3 || len(client.getRunCalls) != 1 || !got.Coverage.Partial {
		t.Fatalf("bounded query lost known identities or promoted wrong owner: %+v", got)
	}
	if got.Usage.Tokens != nil || got.Usage.CostUSD != nil {
		t.Fatal("outage became zero usage")
	}
}

func TestTeamRunAccountingWindowMemberAndMissingMetricCoverage(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	for i, row := range []struct{ member, id, started string }{
		{"lead", "wanted", "2026-09-12T10:00:00Z"}, {"worker", "wrong-member", "2026-09-12T10:00:00Z"},
		{"lead", "old", "2026-09-11T10:00:00Z"}, {"lead", "upper-bound", "2026-09-13T00:00:00Z"},
		{"lead", "invalid-time", "missing"}, {"lead", "", "2026-09-12T10:00:00Z"},
	} {
		if err := teams.AppendHeartbeatAttempt(ctx, "team", &store.HeartbeatAttempt{ID: fmt.Sprint(i), TeamID: "team", AgentID: row.member,
			RunID: row.id, StartedAt: row.started}); err != nil {
			t.Fatal(err)
		}
	}
	h.agentClient = newMockAgentClient().WithGetRunResponse("wanted", &Run{ID: "wanted", Status: "RUN_STATUS_RUNNING", RequestedModel: "not-actual", Summary: json.RawMessage(`{"cost_estimate":0}`)})
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team&agent_id=lead&start=2026-09-12T00:00:00Z&end=2026-09-13T00:00:00Z", nil))
	var got teamRunAccounting
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KnownRuns != 1 || got.Coverage.InvalidTimestamps != 1 || got.UnknownModelExecutions != 1 || len(got.ActualModels) != 0 || got.Usage.ReportedCostRuns != 1 || got.Usage.ReportedTokenRuns != 0 {
		t.Fatalf("window/member scope or metric presence lost: %+v", got)
	}
	if got.Usage.CostUSD != nil {
		t.Fatal("reported estimate became actual charge")
	}
}

func TestTeamRunAccountingDoesNotMergeDifferentOrUnknownRunners(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	ctx := context.Background()
	if err := teams.Create(ctx, newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	client := newMockAgentClient()
	for i, harness := range []string{"codex", "claude-code", "", ""} {
		id := fmt.Sprintf("run-%d", i)
		if err := teams.AppendHeartbeatAttempt(ctx, "team", &store.HeartbeatAttempt{ID: id, TeamID: "team", AgentID: "lead", RunID: id,
			StartedAt: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)}); err != nil {
			t.Fatal(err)
		}
		client.WithGetRunResponse(id, &Run{ID: id, HarnessKind: harness, SessionID: "same-session-text", Status: "RUN_STATUS_RUNNING"})
	}
	h.agentClient = client
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?accounting=true&team_id=team", nil))
	var got teamRunAccounting
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.KnownRuns != 4 || got.ObservedExecutions != 4 || got.DuplicateExecutions != 0 || got.UnknownModelExecutions != 4 {
		t.Fatalf("unqualified session identity merged unrelated runs: %+v", got)
	}
}

func TestTeamRunAccountingRejectsUnboundedOrInvalidQueries(t *testing.T) {
	h, teams, _, _, _ := setupTeamLogsTestHandlers(t)
	if err := teams.Create(context.Background(), newIndependentTestTeam("team", "Team")); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"", "&limit=0", "&limit=101", "&start=invalid", "&start=2026-01-01T00:00:00Z&end=2026-09-12T00:00:00Z"} {
		url := "/runs?accounting=true&team_id=team" + query
		if query == "" {
			url = "/runs?accounting=true"
		}
		w := httptest.NewRecorder()
		h.ListRuns(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("query %q: status %d", query, w.Code)
		}
	}
}

// Exercise the actual AM proto JSON -> PM client -> PM handler boundary.
// A local-log fixture cannot establish what survives this projection.
func TestListRunsPreservesOwnerAccountingEvidence(t *testing.T) {
	owner := &amapi.ListRunsResponse{Runs: []*ampb.Run{{
		Id: "owner-run", Status: ampb.RunStatus_RUN_STATUS_COMPLETE,
		RequestedModel: "requested", ActualModel: "actual", HarnessKind: "codex",
		SessionId: "resumed-session", ImportSourceHarness: "codex", ImportSourceSessionId: "resumed-session",
		TerminalClass: "verdict", StopReason: "blocked",
		WorkReferences: []*eventpb.WorkReference{{Kind: "effort", Id: "effort-a", Relationship: "supervisor",
			Visibility: eventpb.WorkReferenceVisibility_WORK_REFERENCE_VISIBILITY_PUBLIC,
			State:      eventpb.WorkReferenceState_WORK_REFERENCE_STATE_ACTIVE, Verified: true}},
		Summary: &ampb.RunSummary{TokensUsed: 123, CostEstimate: 0.2},
	}, {Id: "missing-metrics", Status: ampb.RunStatus_RUN_STATUS_RUNNING}}, Total: 2, HasMore: true}
	wire, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(owner)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/runs" {
			t.Errorf("unexpected owner request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(wire)
	}))
	defer server.Close()
	h := NewHandlers(HandlersDeps{AgentClient: newTestClient(t, server)})
	w := httptest.NewRecorder()
	h.ListRuns(w, httptest.NewRequest(http.MethodGet, "/runs?limit=2", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var got, want map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wire, &want); err != nil {
		t.Fatal(err)
	}
	run := got["runs"].([]any)[0].(map[string]any)
	expected := want["runs"].([]any)[0].(map[string]any)
	for _, key := range []string{"requested_model", "actual_model", "harness_kind", "session_id", "import_source_harness", "import_source_session_id", "terminal_class", "stop_reason", "summary", "work_references"} {
		a, _ := json.Marshal(run[key])
		b, _ := json.Marshal(expected[key])
		if string(a) != string(b) {
			t.Errorf("lost owner %s: got %s want %s", key, a, b)
		}
	}
	missing := got["runs"].([]any)[1].(map[string]any)
	for _, key := range []string{"actual_model", "summary", "terminal_class", "stop_reason"} {
		if _, exists := missing[key]; exists {
			t.Errorf("missing %s became fabricated evidence: %v", key, missing[key])
		}
	}
	if got["has_more"] != true {
		t.Fatal("partial owner page became complete")
	}
}

func TestListRuns_ResolvesProfileKeyToAgentProfileID(t *testing.T) {
	mockClient := newMockAgentClient().
		WithEnsureProfileResponse(&EnsureProfileResponse{
			Profile: &AgentProfile{
				ID:         "6a89415d-9f95-4ee0-a534-00619ce6fd5a",
				ProfileKey: "prompt-manager-heartbeat",
			},
		}).
		WithListRunsResponse(&ListRunsResponse{
			Runs:  []*Run{{ID: "run-1", Status: "RUN_STATUS_COMPLETE"}},
			Total: 1,
		})

	handlers := NewHandlers(HandlersDeps{
		AgentClient: mockClient,
	})

	req := httptest.NewRequest(http.MethodGet, "/runs?profile_key=prompt-manager-heartbeat&limit=5", nil)
	w := httptest.NewRecorder()
	handlers.ListRuns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()

	if len(mockClient.ensureProfileCalls) != 1 {
		t.Fatalf("expected 1 EnsureProfile call, got %d", len(mockClient.ensureProfileCalls))
	}
	if got := mockClient.ensureProfileCalls[0]; got == nil || got.ProfileKey != "prompt-manager-heartbeat" {
		t.Fatalf("expected EnsureProfile(profile_key=prompt-manager-heartbeat), got %+v", got)
	}
	if len(mockClient.listRunsCalls) != 1 {
		t.Fatalf("expected 1 ListRuns call, got %d", len(mockClient.listRunsCalls))
	}
	if mockClient.listRunsCalls[0].AgentProfileID != "6a89415d-9f95-4ee0-a534-00619ce6fd5a" {
		t.Fatalf("expected AgentProfileID to be resolved UUID, got %q", mockClient.listRunsCalls[0].AgentProfileID)
	}
}

func TestListRuns_ProfileKeyResolutionFailureReturnsBadGateway(t *testing.T) {
	mockClient := newMockAgentClient().WithEnsureProfileResponse(&EnsureProfileResponse{})
	handlers := NewHandlers(HandlersDeps{
		AgentClient: mockClient,
	})

	req := httptest.NewRequest(http.MethodGet, "/runs?profile_key=prompt-manager-heartbeat", nil)
	w := httptest.NewRecorder()
	handlers.ListRuns(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}
