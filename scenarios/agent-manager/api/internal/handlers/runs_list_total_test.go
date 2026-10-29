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
