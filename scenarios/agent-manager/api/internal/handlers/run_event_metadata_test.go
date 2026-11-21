package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRunEventMetadataIsPayloadFreeAndPagesByStableIdentity(t *testing.T) {
	_, router, repos, events := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "metadata source", ScopePath: ".", Status: domain.TaskStatusQueued}
	require.NoError(t, repos.Tasks.Create(ctx, task))
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted}
	require.NoError(t, repos.Runs.Create(ctx, run))
	first := domain.NewMessageEvent(run.ID, "user", "PRIVATE_PROMPT_BODY_ALPHA")
	second := domain.NewMessageEvent(run.ID, "assistant", "PRIVATE_OUTPUT_BODY_BETA")
	require.NoError(t, events.Append(ctx, run.ID, first, second))

	page := httptest.NewRecorder()
	router.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID.String()+"/events/metadata?limit=1", nil))
	require.Equal(t, http.StatusOK, page.Code, page.Body.String())
	require.NotContains(t, page.Body.String(), "PRIVATE_PROMPT_BODY_ALPHA")
	require.NotContains(t, page.Body.String(), "PRIVATE_OUTPUT_BODY_BETA")
	var decoded struct {
		RunID  string `json:"run_id"`
		Events []struct {
			ID       string `json:"event_id"`
			Sequence int64  `json:"sequence"`
			Role     string `json:"role"`
		} `json:"events"`
		HasMore      bool `json:"has_more"`
		MetadataOnly bool `json:"metadata_only"`
	}
	require.NoError(t, json.Unmarshal(page.Body.Bytes(), &decoded))
	require.Equal(t, run.ID.String(), decoded.RunID)
	require.True(t, decoded.MetadataOnly)
	require.True(t, decoded.HasMore)
	require.Len(t, decoded.Events, 1)
	require.Equal(t, first.ID.String(), decoded.Events[0].ID)
	require.Equal(t, int64(0), decoded.Events[0].Sequence)
	require.Equal(t, "user", decoded.Events[0].Role)

	continued := httptest.NewRecorder()
	router.ServeHTTP(continued, httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+run.ID.String()+"/events/metadata?after_sequence=0&limit=1", nil))
	require.Equal(t, http.StatusOK, continued.Code, continued.Body.String())
	require.NotContains(t, continued.Body.String(), "PRIVATE_PROMPT_BODY_ALPHA")
	require.NotContains(t, continued.Body.String(), "PRIVATE_OUTPUT_BODY_BETA")
	var last struct {
		Events []struct {
			ID       string `json:"event_id"`
			Sequence int64  `json:"sequence"`
			Role     string `json:"role"`
		} `json:"events"`
		HasMore bool `json:"has_more"`
	}
	require.NoError(t, json.Unmarshal(continued.Body.Bytes(), &last))
	require.False(t, last.HasMore)
	require.Len(t, last.Events, 1)
	require.Equal(t, second.ID.String(), last.Events[0].ID)
	require.Equal(t, int64(1), last.Events[0].Sequence)
	require.Equal(t, "assistant", last.Events[0].Role)
	require.False(t, strings.Contains(continued.Body.String(), "content"))
}

func TestRunEventMetadataRejectsInvalidPageBounds(t *testing.T) {
	_, router, _, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	runID := uuid.New()
	for _, query := range []string{"limit=0", "limit=1001", "after_sequence=invalid"} {
		t.Run(query, func(t *testing.T) {
			r := httptest.NewRecorder()
			router.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+runID.String()+"/events/metadata?"+query, nil))
			require.Equal(t, http.StatusBadRequest, r.Code)
		})
	}
}
