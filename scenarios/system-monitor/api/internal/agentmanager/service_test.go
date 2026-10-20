package agentmanager

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/scenario"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fakeRequest is what the fake agent-manager observed for one call.
type fakeRequest struct {
	Method string
	Path   string
	Query  string
	Auth   string
	Body   []byte
}

// fakeAgentManager is an httptest agent-manager that answers by "METHOD path"
// with handlers that speak the proto JSON wire shapes, and records every call.
type fakeAgentManager struct {
	t        *testing.T
	server   *httptest.Server
	mu       sync.Mutex
	requests []fakeRequest
	routes   map[string]func(w http.ResponseWriter, req fakeRequest)
}

func newFakeAgentManager(t *testing.T) *fakeAgentManager {
	t.Helper()
	f := &fakeAgentManager{t: t, routes: map[string]func(http.ResponseWriter, fakeRequest){}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		req := fakeRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Auth: r.Header.Get("Authorization"), Body: body}
		f.mu.Lock()
		f.requests = append(f.requests, req)
		handler, ok := f.routes[r.Method+" "+r.URL.Path]
		f.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"no route"}`)
			return
		}
		handler(w, req)
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeAgentManager) handle(method, path string, h func(http.ResponseWriter, fakeRequest)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.routes[method+" "+path] = h
}

func (f *fakeAgentManager) reply(method, path string, status int, msg proto.Message) {
	f.handle(method, path, func(w http.ResponseWriter, _ fakeRequest) {
		writeProto(f.t, w, status, msg)
	})
}

func (f *fakeAgentManager) fail(method, path string, status int, body string) {
	f.handle(method, path, func(w http.ResponseWriter, _ fakeRequest) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	})
}

func (f *fakeAgentManager) calls() []fakeRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakeRequest(nil), f.requests...)
}

func (f *fakeAgentManager) callsTo(method, path string) []fakeRequest {
	var out []fakeRequest
	for _, c := range f.calls() {
		if c.Method == method && c.Path == path {
			out = append(out, c)
		}
	}
	return out
}

func writeProto(t *testing.T, w http.ResponseWriter, status int, msg proto.Message) {
	t.Helper()
	data, err := protojson.Marshal(msg)
	if err != nil {
		t.Errorf("marshal fake response: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func decodeBody(t *testing.T, req fakeRequest, msg proto.Message) {
	t.Helper()
	if err := protojson.Unmarshal(req.Body, msg); err != nil {
		t.Fatalf("decode %s %s body %q: %v", req.Method, req.Path, req.Body, err)
	}
}

// newTestService builds an enabled service whose client targets the fake.
func newTestService(t *testing.T, f *fakeAgentManager, profileKey string) *AgentService {
	t.Helper()
	svc := NewAgentService(AgentServiceConfig{ProfileName: "System Monitor", ProfileKey: profileKey, Timeout: 5 * time.Second, Enabled: true})
	svc.client.baseURLResolver = func(context.Context) (string, error) { return f.server.URL, nil }
	svc.pollInterval = time.Millisecond
	return svc
}

func TestNewAgentServiceAppliesConfig(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{ProfileName: "name", ProfileKey: "key", Timeout: 7 * time.Second, Enabled: true})
	if !svc.IsEnabled() {
		t.Fatal("enabled config should report enabled")
	}
	if svc.client.httpClient.Timeout != 7*time.Second {
		t.Fatalf("client timeout = %v, want 7s", svc.client.httpClient.Timeout)
	}
	if svc.GetProfileID() != "" {
		t.Fatalf("profile id before Initialize = %q, want empty", svc.GetProfileID())
	}
	if NewAgentService(AgentServiceConfig{}).IsEnabled() {
		t.Fatal("zero config should be disabled")
	}
}

func TestDisabledServiceRefusesEveryOperationWithoutCallingAgentManager(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	svc.enabled = false
	ctx := auth01FixtureCaller(t, context.Background())

	if svc.IsAvailable(ctx) {
		t.Error("disabled service reported available")
	}
	if err := svc.Initialize(ctx); err != nil {
		t.Errorf("Initialize on disabled service = %v, want nil", err)
	}
	notEnabled := map[string]error{}
	_, notEnabled["ResolveURL"] = svc.ResolveURL(ctx)
	_, notEnabled["Execute"] = svc.Execute(ctx, ExecuteRequest{InvestigationID: "x"})
	_, notEnabled["ExecuteAsync"] = svc.ExecuteAsync(ctx, ExecuteRequest{InvestigationID: "x"})
	_, notEnabled["GetRunStatus"] = svc.GetRunStatus(ctx, "r")
	_, notEnabled["GetRunByTag"] = svc.GetRunByTag(ctx, "t")
	_, notEnabled["ListActiveRuns"] = svc.ListActiveRuns(ctx)
	notEnabled["StopRun"] = svc.StopRun(ctx, "r")
	_, notEnabled["StopAllRuns"] = svc.StopAllRuns(ctx)
	_, notEnabled["GetRunEvents"] = svc.GetRunEvents(ctx, "r")
	for op, err := range notEnabled {
		if err == nil || !strings.Contains(err.Error(), "not enabled") {
			t.Errorf("%s on disabled service = %v, want not-enabled error", op, err)
		}
	}
	if n := len(f.calls()); n != 0 {
		t.Fatalf("disabled service made %d agent-manager calls, want 0", n)
	}
}

func TestIsAvailableReflectsAgentManagerHealth(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")

	f.fail(http.MethodGet, "/health", http.StatusOK, `{"status":"healthy"}`)
	if !svc.IsAvailable(context.Background()) {
		t.Error("healthy agent-manager reported unavailable")
	}

	f.fail(http.MethodGet, "/health", http.StatusServiceUnavailable, `{"status":"down"}`)
	if svc.IsAvailable(context.Background()) {
		t.Error("unhealthy agent-manager reported available")
	}

	f.server.Close()
	if svc.IsAvailable(context.Background()) {
		t.Error("unreachable agent-manager reported available")
	}
}

func TestIsAvailableFalseWhenURLCannotBeResolved(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	svc.client.baseURLResolver = func(context.Context) (string, error) { return "", errors.New("agent-manager not running") }
	if svc.IsAvailable(context.Background()) {
		t.Fatal("unresolvable agent-manager reported available")
	}
}

func TestResolveURLReturnsConfiguredURLOrResolutionError(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")

	got, err := svc.ResolveURL(context.Background())
	if err != nil || got != f.server.URL {
		t.Fatalf("ResolveURL = %q, %v; want %q", got, err, f.server.URL)
	}

	missing := errors.New("agent-manager not registered")
	svc.client.baseURLResolver = func(context.Context) (string, error) { return "", missing }
	if _, err := svc.ResolveURL(context.Background()); !errors.Is(err, missing) {
		t.Fatalf("ResolveURL with missing agent-manager = %v, want %v", err, missing)
	}
}

func TestInitializeReconcilesScenarioProfilesAndAdoptsConfiguredProfile(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "system-monitor-investigator")
	f.reply(http.MethodPost, "/api/v1/profiles/reconcile-scenario", http.StatusOK, &apipb.ReconcileScenarioProfilesResponse{
		Scenario: "system-monitor",
		Created:  1,
		Results: []*apipb.ProfileReconcileResult{
			{ProfileKey: "other-profile", ProfileId: "profile-other"},
			{ProfileKey: "system-monitor-investigator", ProfileId: "profile-123"},
		},
	})

	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if got := svc.GetProfileID(); got != "profile-123" {
		t.Fatalf("profile id = %q, want profile-123", got)
	}
	calls := f.callsTo(http.MethodPost, "/api/v1/profiles/reconcile-scenario")
	if len(calls) != 1 {
		t.Fatalf("reconcile calls = %d, want 1", len(calls))
	}
	var sent apipb.ReconcileScenarioProfilesRequest
	decodeBody(t, calls[0], &sent)
	if sent.Scenario == "" || sent.Scenario != scenario.Name() {
		t.Fatalf("reconciled scenario = %q, want this scenario %q", sent.Scenario, scenario.Name())
	}
}

func TestInitializeLeavesProfileUnsetWhenConfiguredProfileIsAbsent(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "missing-key")
	f.reply(http.MethodPost, "/api/v1/profiles/reconcile-scenario", http.StatusOK, &apipb.ReconcileScenarioProfilesResponse{
		Results: []*apipb.ProfileReconcileResult{{ProfileKey: "other", ProfileId: "profile-other"}},
	})
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if got := svc.GetProfileID(); got != "" {
		t.Fatalf("profile id = %q, want empty", got)
	}
}

func TestInitializeSurfacesReconcileFailure(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	f.fail(http.MethodPost, "/api/v1/profiles/reconcile-scenario", http.StatusInternalServerError, `{"error":"profile source invalid"}`)

	err := svc.Initialize(context.Background())
	if err == nil || !strings.Contains(err.Error(), "reconcile scenario profiles") || !strings.Contains(err.Error(), "profile source invalid") {
		t.Fatalf("Initialize = %v, want wrapped agent-manager error", err)
	}
	if svc.GetProfileID() != "" {
		t.Fatal("failed reconcile must not set a profile id")
	}
}

func TestExecuteRefusesUnverifiedCallerBeforeAnyRequest(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")

	if _, err := svc.Execute(context.Background(), ExecuteRequest{InvestigationID: "inv-1", Prompt: "p"}); !errors.Is(err, owneridentity.ErrCreateRunCaller) {
		t.Fatalf("Execute without caller = %v, want ErrCreateRunCaller", err)
	}
	if _, err := svc.ExecuteAsync(context.Background(), ExecuteRequest{InvestigationID: "inv-1", Prompt: "p"}); !errors.Is(err, owneridentity.ErrCreateRunCaller) {
		t.Fatalf("ExecuteAsync without caller = %v, want ErrCreateRunCaller", err)
	}
	if n := len(f.calls()); n != 0 {
		t.Fatalf("unverified caller produced %d agent-manager calls, want 0", n)
	}
}

// serveInvestigationRun wires task creation, run creation, and a run that
// reports running once before reaching the given terminal state.
func serveInvestigationRun(f *fakeAgentManager, terminal *domainpb.Run) {
	f.reply(http.MethodPost, "/api/v1/tasks", http.StatusCreated, &apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-7"}})
	f.reply(http.MethodPost, "/api/v1/runs", http.StatusCreated, &apipb.CreateRunResponse{Run: &domainpb.Run{Id: terminal.Id, Status: domainpb.RunStatus_RUN_STATUS_PENDING}})
	var mu sync.Mutex
	polls := 0
	f.handle(http.MethodGet, "/api/v1/runs/"+terminal.Id, func(w http.ResponseWriter, _ fakeRequest) {
		mu.Lock()
		polls++
		first := polls == 1
		mu.Unlock()
		if first {
			writeProto(f.t, w, http.StatusOK, &apipb.GetRunResponse{Run: &domainpb.Run{Id: terminal.Id, Status: domainpb.RunStatus_RUN_STATUS_RUNNING}})
			return
		}
		writeProto(f.t, w, http.StatusOK, &apipb.GetRunResponse{Run: terminal})
	})
}

func TestExecuteCreatesTaggedInvestigationRunAndReturnsCompletedResult(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "system-monitor-investigator")
	started := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	serveInvestigationRun(f, &domainpb.Run{
		Id:        "run-42",
		Status:    domainpb.RunStatus_RUN_STATUS_COMPLETE,
		Summary:   &domainpb.RunSummary{Description: "disk pressure from logs", TokensUsed: 1234, CostEstimate: 0.25},
		StartedAt: timestamppb.New(started),
		EndedAt:   timestamppb.New(started.Add(95 * time.Second)),
	})

	res, err := svc.Execute(auth01FixtureCaller(t, context.Background()), ExecuteRequest{InvestigationID: "inv-9", Prompt: "investigate cpu", WorkingDir: "/work/dir"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	want := ExecuteResult{RunID: "run-42", Success: true, Output: "disk pressure from logs", TokensUsed: 1234, CostEstimate: 0.25, DurationSeconds: 95}
	if *res != want {
		t.Fatalf("result = %+v, want %+v", *res, want)
	}

	tasks := f.callsTo(http.MethodPost, "/api/v1/tasks")
	if len(tasks) != 1 {
		t.Fatalf("task creations = %d, want 1", len(tasks))
	}
	var taskReq apipb.CreateTaskRequest
	decodeBody(t, tasks[0], &taskReq)
	task := taskReq.GetTask()
	if task.GetTitle() != "System Investigation inv-9" || task.GetDescription() != "investigate cpu" ||
		task.GetScopePath() != "/work/dir" || task.GetProjectRoot() != "/work/dir" || task.GetCreatedBy() != "system-monitor" {
		t.Fatalf("task sent = %+v", task)
	}

	runs := f.callsTo(http.MethodPost, "/api/v1/runs")
	if len(runs) != 1 {
		t.Fatalf("run creations = %d, want 1", len(runs))
	}
	if runs[0].Auth != "Bearer fixture-owner" {
		t.Fatalf("run creation authorization = %q, want verified caller proof", runs[0].Auth)
	}
	var runReq apipb.CreateRunRequest
	decodeBody(t, runs[0], &runReq)
	if runReq.GetTaskId() != "task-7" || runReq.GetProfileRef().GetProfileKey() != "system-monitor-investigator" ||
		runReq.GetTag() != "system-monitor-inv-9" || runReq.GetRunMode() != domainpb.RunMode_RUN_MODE_IN_PLACE || !runReq.GetForce() {
		t.Fatalf("run request sent = %+v", &runReq)
	}
	if polls := len(f.callsTo(http.MethodGet, "/api/v1/runs/run-42")); polls < 2 {
		t.Fatalf("run polls = %d, want to keep polling until terminal", polls)
	}
}

func TestExecuteReportsFailedRunWithRateLimitAndTimeoutSignals(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	serveInvestigationRun(f, &domainpb.Run{
		Id:       "run-f",
		Status:   domainpb.RunStatus_RUN_STATUS_FAILED,
		ErrorMsg: "Rate Limit reached after request Timeout",
	})

	res, err := svc.Execute(auth01FixtureCaller(t, context.Background()), ExecuteRequest{InvestigationID: "inv-f"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Success || res.ErrorMessage != "Rate Limit reached after request Timeout" || !res.RateLimited || !res.Timeout || res.DurationSeconds != 0 {
		t.Fatalf("result = %+v, want failed rate-limited timeout without duration", *res)
	}
}

func TestExecuteSurfacesAgentManagerFailures(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(f *fakeAgentManager)
		wantErr []string
	}{
		{
			name: "task creation rejected",
			setup: func(f *fakeAgentManager) {
				f.fail(http.MethodPost, "/api/v1/tasks", http.StatusBadRequest, `{"error":"title required"}`)
			},
			wantErr: []string{"create task", "title required"},
		},
		{
			name: "run creation rejected",
			setup: func(f *fakeAgentManager) {
				f.reply(http.MethodPost, "/api/v1/tasks", http.StatusCreated, &apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}})
				f.fail(http.MethodPost, "/api/v1/runs", http.StatusForbidden, `{"message":"caller lacks scope"}`)
			},
			wantErr: []string{"create run", "caller lacks scope"},
		},
		{
			name: "run disappears while waiting",
			setup: func(f *fakeAgentManager) {
				f.reply(http.MethodPost, "/api/v1/tasks", http.StatusCreated, &apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}})
				f.reply(http.MethodPost, "/api/v1/runs", http.StatusCreated, &apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-gone"}})
			},
			wantErr: []string{"wait for run", "run-gone not found"},
		},
		{
			name: "run status lookup fails while waiting",
			setup: func(f *fakeAgentManager) {
				f.reply(http.MethodPost, "/api/v1/tasks", http.StatusCreated, &apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}})
				f.reply(http.MethodPost, "/api/v1/runs", http.StatusCreated, &apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-x"}})
				f.fail(http.MethodGet, "/api/v1/runs/run-x", http.StatusInternalServerError, "database locked")
			},
			wantErr: []string{"wait for run", "status 500", "database locked"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeAgentManager(t)
			svc := newTestService(t, f, "key")
			tc.setup(f)
			res, err := svc.Execute(auth01FixtureCaller(t, context.Background()), ExecuteRequest{InvestigationID: "inv"})
			if err == nil || res != nil {
				t.Fatalf("Execute = %+v, %v; want error", res, err)
			}
			for _, part := range tc.wantErr {
				if !strings.Contains(err.Error(), part) {
					t.Fatalf("error %q does not mention %q", err, part)
				}
			}
		})
	}
}

func TestExecuteStopsWaitingWhenContextEnds(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	ctx, cancel := context.WithCancel(auth01FixtureCaller(t, context.Background()))
	defer cancel()
	f.reply(http.MethodPost, "/api/v1/tasks", http.StatusCreated, &apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}})
	f.reply(http.MethodPost, "/api/v1/runs", http.StatusCreated, &apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-slow"}})
	f.handle(http.MethodGet, "/api/v1/runs/run-slow", func(w http.ResponseWriter, _ fakeRequest) {
		cancel() // caller gives up while the run is still running
		writeProto(f.t, w, http.StatusOK, &apipb.GetRunResponse{Run: &domainpb.Run{Id: "run-slow", Status: domainpb.RunStatus_RUN_STATUS_RUNNING}})
	})

	if _, err := svc.Execute(ctx, ExecuteRequest{InvestigationID: "inv"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute after cancel = %v, want context.Canceled", err)
	}
}

func TestExecuteAsyncStartsTaggedRunAndReturnsItsID(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "profile-key")
	f.reply(http.MethodPost, "/api/v1/tasks", http.StatusOK, &apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-a"}})
	f.reply(http.MethodPost, "/api/v1/runs", http.StatusOK, &apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-a"}})

	runID, err := svc.ExecuteAsync(auth01FixtureCaller(t, context.Background()), ExecuteRequest{InvestigationID: "inv-a", Prompt: "check memory", WorkingDir: "/w"})
	if err != nil || runID != "run-a" {
		t.Fatalf("ExecuteAsync = %q, %v; want run-a", runID, err)
	}
	var runReq apipb.CreateRunRequest
	decodeBody(t, f.callsTo(http.MethodPost, "/api/v1/runs")[0], &runReq)
	if runReq.GetTaskId() != "task-a" || runReq.GetTag() != "system-monitor-inv-a" || runReq.GetProfileRef().GetProfileKey() != "profile-key" || !runReq.GetForce() {
		t.Fatalf("run request sent = %+v", &runReq)
	}
	if n := len(f.callsTo(http.MethodGet, "/api/v1/runs/run-a")); n != 0 {
		t.Fatalf("ExecuteAsync polled the run %d times, want 0", n)
	}
}

func TestExecuteAsyncSurfacesAgentManagerFailures(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	ctx := auth01FixtureCaller(t, context.Background())

	f.fail(http.MethodPost, "/api/v1/tasks", http.StatusInternalServerError, `{"error":"store offline"}`)
	if _, err := svc.ExecuteAsync(ctx, ExecuteRequest{InvestigationID: "i"}); err == nil || !strings.Contains(err.Error(), "create task") || !strings.Contains(err.Error(), "store offline") {
		t.Fatalf("ExecuteAsync task failure = %v", err)
	}

	f.reply(http.MethodPost, "/api/v1/tasks", http.StatusOK, &apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-a"}})
	f.fail(http.MethodPost, "/api/v1/runs", http.StatusConflict, `{"error":"capacity"}`)
	if _, err := svc.ExecuteAsync(ctx, ExecuteRequest{InvestigationID: "i"}); err == nil || !strings.Contains(err.Error(), "create run") || !strings.Contains(err.Error(), "capacity") {
		t.Fatalf("ExecuteAsync run failure = %v", err)
	}
}

func TestGetRunStatusMapsFoundMissingAndFailure(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	f.reply(http.MethodGet, "/api/v1/runs/run-1", http.StatusOK, &apipb.GetRunResponse{Run: &domainpb.Run{Id: "run-1", Status: domainpb.RunStatus_RUN_STATUS_RUNNING}})
	f.fail(http.MethodGet, "/api/v1/runs/run-err", http.StatusBadGateway, `{"error":"upstream"}`)

	run, err := svc.GetRunStatus(context.Background(), "run-1")
	if err != nil || run.GetId() != "run-1" || run.GetStatus() != domainpb.RunStatus_RUN_STATUS_RUNNING {
		t.Fatalf("GetRunStatus = %+v, %v", run, err)
	}
	if run, err := svc.GetRunStatus(context.Background(), "run-missing"); err != nil || run != nil {
		t.Fatalf("GetRunStatus missing = %+v, %v; want nil, nil", run, err)
	}
	if _, err := svc.GetRunStatus(context.Background(), "run-err"); err == nil || !strings.Contains(err.Error(), "upstream") {
		t.Fatalf("GetRunStatus failure = %v", err)
	}
}

func TestGetRunByTagLooksUpTagPath(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	f.reply(http.MethodGet, "/api/v1/runs/tag/system-monitor-inv-3", http.StatusOK, &apipb.GetRunByTagResponse{Run: &domainpb.Run{Id: "run-3"}})
	f.fail(http.MethodGet, "/api/v1/runs/tag/broken", http.StatusInternalServerError, `{"error":"index corrupt"}`)

	run, err := svc.GetRunByTag(context.Background(), "system-monitor-inv-3")
	if err != nil || run.GetId() != "run-3" {
		t.Fatalf("GetRunByTag = %+v, %v", run, err)
	}
	if run, err := svc.GetRunByTag(context.Background(), "unknown"); err != nil || run != nil {
		t.Fatalf("GetRunByTag unknown = %+v, %v; want nil, nil", run, err)
	}
	if _, err := svc.GetRunByTag(context.Background(), "broken"); err == nil || !strings.Contains(err.Error(), "index corrupt") {
		t.Fatalf("GetRunByTag failure = %v", err)
	}
}

func TestListActiveRunsFiltersRunningSystemMonitorRuns(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	f.reply(http.MethodGet, "/api/v1/runs", http.StatusOK, &apipb.ListRunsResponse{Runs: []*domainpb.Run{{Id: "a"}, {Id: "b"}}})

	runs, err := svc.ListActiveRuns(context.Background())
	if err != nil || len(runs) != 2 || runs[0].GetId() != "a" || runs[1].GetId() != "b" {
		t.Fatalf("ListActiveRuns = %v, %v", runs, err)
	}
	calls := f.callsTo(http.MethodGet, "/api/v1/runs")
	wantQuery := "status=" + strconv.Itoa(int(domainpb.RunStatus_RUN_STATUS_RUNNING)) + "&tag_prefix=system-monitor-"
	if len(calls) != 1 || calls[0].Query != wantQuery {
		t.Fatalf("list query = %+v, want %q", calls, wantQuery)
	}

	f.fail(http.MethodGet, "/api/v1/runs", http.StatusInternalServerError, "not json")
	if _, err := svc.ListActiveRuns(context.Background()); err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("ListActiveRuns failure = %v", err)
	}
}

func TestStopRunPostsStopAndSurfacesFailure(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	f.reply(http.MethodPost, "/api/v1/runs/run-1/stop", http.StatusOK, &apipb.StopRunResponse{})

	if err := svc.StopRun(context.Background(), "run-1"); err != nil {
		t.Fatalf("StopRun: %v", err)
	}
	if n := len(f.callsTo(http.MethodPost, "/api/v1/runs/run-1/stop")); n != 1 {
		t.Fatalf("stop calls = %d, want 1", n)
	}
	if err := svc.StopRun(context.Background(), "run-unknown"); err == nil || !strings.Contains(err.Error(), "no route") {
		t.Fatalf("StopRun unknown = %v, want agent-manager error", err)
	}
}

func TestStopAllRunsStopsOnlySystemMonitorRunsAndReturnsCount(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	f.reply(http.MethodPost, "/api/v1/runs/stop-all", http.StatusOK, &apipb.StopAllRunsResponse{Result: &domainpb.StopAllResult{StoppedCount: 3}})

	n, err := svc.StopAllRuns(context.Background())
	if err != nil || n != 3 {
		t.Fatalf("StopAllRuns = %d, %v; want 3", n, err)
	}
	var sent apipb.StopAllRunsRequest
	decodeBody(t, f.callsTo(http.MethodPost, "/api/v1/runs/stop-all")[0], &sent)
	if sent.GetTagPrefix() != "system-monitor-" {
		t.Fatalf("stop-all tag prefix = %q, want system-monitor-", sent.GetTagPrefix())
	}

	f.reply(http.MethodPost, "/api/v1/runs/stop-all", http.StatusOK, &apipb.StopAllRunsResponse{})
	if n, err := svc.StopAllRuns(context.Background()); err != nil || n != 0 {
		t.Fatalf("StopAllRuns without result = %d, %v; want 0", n, err)
	}

	f.fail(http.MethodPost, "/api/v1/runs/stop-all", http.StatusInternalServerError, `{"error":"stop failed"}`)
	if _, err := svc.StopAllRuns(context.Background()); err == nil || !strings.Contains(err.Error(), "stop failed") {
		t.Fatalf("StopAllRuns failure = %v", err)
	}

	f.fail(http.MethodPost, "/api/v1/runs/stop-all", http.StatusOK, `{not json`)
	if _, err := svc.StopAllRuns(context.Background()); err == nil || !strings.Contains(err.Error(), "unmarshal response") {
		t.Fatalf("StopAllRuns malformed = %v", err)
	}
}

func TestGetRunEventsReturnsAllEventsForRun(t *testing.T) {
	f := newFakeAgentManager(t)
	svc := newTestService(t, f, "key")
	f.reply(http.MethodGet, "/api/v1/runs/run-1/events", http.StatusOK, &apipb.GetRunEventsResponse{Events: []*domainpb.RunEvent{{Sequence: 1}, {Sequence: 2}}})

	events, err := svc.GetRunEvents(context.Background(), "run-1")
	if err != nil || len(events) != 2 || events[0].GetSequence() != 1 || events[1].GetSequence() != 2 {
		t.Fatalf("GetRunEvents = %v, %v", events, err)
	}
	if q := f.callsTo(http.MethodGet, "/api/v1/runs/run-1/events")[0].Query; q != "" {
		t.Fatalf("events query = %q, want all events (no after_sequence)", q)
	}
	if _, err := svc.GetRunEvents(context.Background(), "run-unknown"); err == nil || !strings.Contains(err.Error(), "no route") {
		t.Fatalf("GetRunEvents unknown = %v", err)
	}
}
