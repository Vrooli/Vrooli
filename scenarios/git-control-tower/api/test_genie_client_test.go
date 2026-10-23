package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"

	"github.com/vrooli/api-core/discovery"
	httpx "github.com/vrooli/api-core/servertest"
)

func TestTestGenieClient_ExecuteSuite(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/executions": func(w http.ResponseWriter, r *http.Request) {
			httpx.AssertMethod(t, r, http.MethodPost)
			req := httpx.DecodeJSON[TestExecutionRequest](t, r)
			if req.ScenarioName != "git-control-tower" {
				t.Errorf("unexpected scenario name: %s", req.ScenarioName)
			}
			httpx.WriteJSON(t, w, http.StatusOK, TestExecutionResult{
				ExecutionID:  "exec-001",
				ScenarioName: req.ScenarioName,
				Success:      true,
				StartedAt:    "2026-03-10T12:00:00Z",
				CompletedAt:  "2026-03-10T12:05:00Z",
				Phases: []TestPhaseResult{
					{Name: "build", Status: "passed", DurationSeconds: 30},
					{Name: "unit", Status: "passed", DurationSeconds: 60},
				},
				PhaseSummary: TestPhaseSummary{Total: 2, Passed: 2, DurationSeconds: 90},
			})
		},
	})

	client := &TestGenieClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "test-genie",
		},
	}

	result, err := client.ExecuteSuite(context.Background(), TestExecutionRequest{
		ScenarioName: "git-control-tower",
		Preset:       "comprehensive",
	})
	if err != nil {
		t.Fatalf("ExecuteSuite returned error: %v", err)
	}
	if result.ExecutionID != "exec-001" {
		t.Errorf("expected execution ID exec-001, got %s", result.ExecutionID)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if len(result.Phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(result.Phases))
	}
}

func TestTestGenieClient_ExecuteSuite_ServerError(t *testing.T) {
	t.Parallel()

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/api/v1/executions": func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteJSON(t, w, http.StatusInternalServerError, map[string]string{"error": "internal failure"})
		},
	})

	client := &TestGenieClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "test-genie",
		},
	}

	_, err := client.ExecuteSuite(context.Background(), TestExecutionRequest{
		ScenarioName: "git-control-tower",
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

type reviewRunHistoryFixture struct {
	runs_v1connect.UnimplementedRunsServiceHandler
	lists int
}

func (f *reviewRunHistoryFixture) ListRuns(_ context.Context, r *connect.Request[runspb.ListRunsRequest]) (*connect.Response[runspb.ListRunsResponse], error) {
	if r.Msg.Target != "scenario-to-desktop" || r.Msg.Limit != 1 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	f.lists++
	return connect.NewResponse(&runspb.ListRunsResponse{Runs: []*runspb.RunInfo{reviewHistoryRun()}}), nil
}

func (f *reviewRunHistoryFixture) GetRun(_ context.Context, r *connect.Request[runspb.GetRunRequest]) (*connect.Response[runspb.GetRunResponse], error) {
	if r.Msg.Target != "scenario-to-desktop" || r.Msg.RunId != "20261022-175755-c2fccb14" {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&runspb.GetRunResponse{Run: reviewHistoryRun()}), nil
}

func reviewHistoryRun() *runspb.RunInfo {
	return &runspb.RunInfo{RunId: "20261022-175755-c2fccb14", Target: "scenario-to-desktop", Status: "failed", CompletedAt: "2026-10-22T18:04:00Z", Phases: []*runspb.PhaseInfo{{Name: "unit", Status: "passed"}, {Name: "dependencies", Status: "error"}, {Name: "integration", Status: "failed"}}}
}

func TestReviewTestsUseCanonicalNativeRunHistory(t *testing.T) {
	fixture := &reviewRunHistoryFixture{}
	_, handler := runs_v1connect.NewRunsServiceHandler(fixture)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := &TestGenieClient{BaseClient: BaseClient{httpClient: server.Client(), resolver: discovery.NewStaticResolver(server.URL), serviceName: "test-genie"}}
	srv := &Server{testGenieClient: client, capabilities: NewCapabilityRegistry([]CapabilityDef{{ID: "test-genie", DependencySlug: "test-genie"}}, map[string]StatusChecker{"test-genie": &FakeStatusChecker{Status: StatusAvailable}}, time.Minute)}
	for _, id := range []string{"", "20261022-175755-c2fccb14"} {
		dim, available := srv.fetchTestsDimension(context.Background(), "scenario-to-desktop", id, 1)
		if !available || dim == nil || dim.Total != 3 || dim.PassedCount != 1 || dim.FailedCount != 2 || dim.Passed || dim.LastRun == "" || len(dim.Failures) != 1 || dim.Failures[0].Phase != "dependencies" {
			t.Fatalf("review omitted or misrepresented native run %q: %+v", id, dim)
		}
	}
	before := fixture.lists
	if _, err := client.ReadRun(context.Background(), "scenario-to-desktop", "missing-exact-run"); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing exact run: %v", err)
	}
	if fixture.lists != before {
		t.Fatal("missing exact run silently fell back to latest")
	}
}
