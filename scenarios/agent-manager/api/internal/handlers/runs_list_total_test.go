package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// ListRuns.total is the size of the filtered population, not the page, so a
// limit=1 read answers "how many runs are running" for the dashboard.
func TestListRunsTotalCountsTheFilteredPopulationIndependentOfPaging(t *testing.T) {
	_, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "list total", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	for i, status := range []domain.RunStatus{domain.RunStatusRunning, domain.RunStatusRunning, domain.RunStatusRunning, domain.RunStatusNeedsReview, domain.RunStatusComplete} {
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: "list-total-" + string(rune('a'+i)), Status: status, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
	}

	for _, tc := range []struct {
		query       string
		wantRuns    int
		wantTotal   int32
		wantHasMore bool
	}{
		{"status=running&limit=1", 1, 3, true},
		{"status=running&limit=1&offset=2", 1, 3, false},
		{"status=running&limit=10", 3, 3, false},
		{"status=needs_review&limit=1", 1, 1, false},
		{"status=failed&limit=1", 0, 0, false},
		{"limit=2", 2, 5, true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/runs?"+tc.query, nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			var response apipb.ListRunsResponse
			decodeProtoJSON(t, rr.Body.Bytes(), &response)
			if len(response.GetRuns()) != tc.wantRuns || response.GetTotal() != tc.wantTotal || response.GetHasMore() != tc.wantHasMore {
				t.Fatalf("runs=%d total=%d hasMore=%t, want runs=%d total=%d hasMore=%t", len(response.GetRuns()), response.GetTotal(), response.GetHasMore(), tc.wantRuns, tc.wantTotal, tc.wantHasMore)
			}
		})
	}
}

// top_level and origin narrow the list server-side, and each listed run carries
// its sub-agent count from the same query.
func TestListRunsTopLevelOriginAndSubAgentCount(t *testing.T) {
	_, router, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "origins", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	create := func(tag, label string, mode domain.ExecutionMode, parent *uuid.UUID, kind ...domain.WorkloadKind) *domain.Run {
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: tag, Label: label, ExecutionMode: mode, Status: domain.RunStatusComplete, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone, ParentRunID: parent}
		if len(kind) > 0 {
			run.Workload.Kind = kind[0]
		}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	parent := create("parent", "Fix the parser", domain.ExecutionModeCodecPipe, nil)
	create("child-a", "slice a", domain.ExecutionModeCodecPipe, &parent.ID)
	create("child-b", "slice b", domain.ExecutionModeCodecPipe, &parent.ID)
	create("step", "draft", domain.ExecutionModeCodecPipe, nil, domain.WorkloadKindWorkflowNode)
	create("labelled", "Workflow 11111111-2222-3333-4444-555555555555 node draft", domain.ExecutionModeCodecPipe, nil)
	create("session", "codex session", domain.ExecutionModeImported, nil)

	list := func(query string) *apipb.ListRunsResponse {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/runs?"+query, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s", query, rr.Code, rr.Body.String())
		}
		var response apipb.ListRunsResponse
		decodeProtoJSON(t, rr.Body.Bytes(), &response)
		return &response
	}
	for _, tc := range []struct {
		query     string
		wantTotal int32
	}{
		{"limit=10", 6},
		{"top_level=true&limit=10", 4},
		{"top_level=true&origin=agent_manager", 2},
		{"origin=agent_manager", 4},
		{"origin=workflows", 1},
		{"origin=operator_sessions", 1},
		{"parent_run_id=" + parent.ID.String() + "&limit=1", 2},
	} {
		if got := list(tc.query); got.GetTotal() != tc.wantTotal {
			t.Errorf("%s: total=%d, want %d", tc.query, got.GetTotal(), tc.wantTotal)
		}
	}
	for _, run := range list("top_level=true&origin=agent_manager").GetRuns() {
		want := int32(0)
		if run.GetId() == parent.ID.String() {
			want = 2
		}
		if run.GetSubAgentCount() != want {
			t.Errorf("run %s: sub_agent_count=%d, want %d", run.GetId(), run.GetSubAgentCount(), want)
		}
	}
}
