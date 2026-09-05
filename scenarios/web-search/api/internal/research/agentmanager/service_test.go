package agentmanager_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"web-search/internal/research/agentmanager"
)

type fakeClient struct {
	execution *domainpb.WorkflowExecution
	request   *apipb.StartWorkflowExecutionRequest
	waits     int
	timedOut  bool
}

func (f *fakeClient) Start(_ context.Context, r *apipb.StartWorkflowExecutionRequest) (*domainpb.WorkflowExecution, error) {
	f.request = r
	return f.execution, nil
}

func (f *fakeClient) Get(context.Context, string) (*domainpb.WorkflowExecution, error) {
	return f.execution, nil
}

func (f *fakeClient) Result(context.Context, string) (*domainpb.WorkflowExecution, error) {
	return f.execution, nil
}

func (f *fakeClient) Wait(context.Context, string, int) (*domainpb.WorkflowExecution, bool, error) {
	f.waits++
	return f.execution, f.timedOut, nil
}

func execution() *domainpb.WorkflowExecution {
	return &domainpb.WorkflowExecution{Id: "execution-1", Owner: "web-search", WorkflowKey: "web-search/research", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING}
}

// [REQ:REQ-P0-010] Start carries stable identity and typed input to a declared workflow.
func TestSpawnDeclaredWorkflow(t *testing.T) {
	f := &fakeClient{execution: execution()}
	s := agentmanager.NewService(f)
	r, e := s.Spawn(context.Background(), agentmanager.SpawnRequest{Query: "q", GatherCap: 20, MaxLoops: 4, ConfidenceGate: 0.75, IdempotencyKey: "request-1"})
	require.NoError(t, e)
	require.Equal(t, "execution-1", r.RunID)
	require.Equal(t, "web-search/research", f.request.WorkflowKey)
	require.Equal(t, "request-1", f.request.IdempotencyKey)
	require.Equal(t, "q", f.request.Input.GetStructValue().Fields["query"].GetStringValue())
}

// [REQ:REQ-P0-010] A wait timeout preserves the active execution and calls the owner once.
func TestWaitTimeoutPreservesExecution(t *testing.T) {
	f := &fakeClient{execution: execution(), timedOut: true}
	s := agentmanager.NewService(f).(agentmanager.Waiter)
	r, e := s.Wait(context.Background(), "execution-1", 30)
	require.NoError(t, e)
	require.True(t, r.TimedOut)
	require.Equal(t, "running", r.Status)
	require.Equal(t, 1, f.waits)
	_, e = s.Wait(context.Background(), "execution-1", 91)
	require.Error(t, e)
	require.Equal(t, 1, f.waits)
}

func TestForeignExecutionRefusedBeforeWait(t *testing.T) {
	f := &fakeClient{execution: execution()}
	f.execution.Owner = "other"
	_, e := agentmanager.NewService(f).(agentmanager.Waiter).Wait(context.Background(), "execution-1", 30)
	require.Error(t, e)
	require.Zero(t, f.waits)
}

func TestCompletedExecutionRequiresStructuredResult(t *testing.T) {
	f := &fakeClient{execution: execution()}
	f.execution.Status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED
	f.execution.EndedAt = timestamppb.New(time.Now())
	s := agentmanager.NewService(f)
	r, e := s.GetRunState(context.Background(), "execution-1")
	require.NoError(t, e)
	require.Equal(t, "failed", r.Status)
	require.Equal(t, "research_result_missing", r.ErrorMsg)
	f.execution.Output, _ = structpb.NewValue(map[string]any{"result": map[string]any{"status": "abstained", "summary": "insufficient sources", "claims": []any{}, "gaps": []any{"no sources"}, "contradictions": []any{}, "finding_ids": []any{}}})
	r, e = s.GetRunState(context.Background(), "execution-1")
	require.NoError(t, e)
	require.Equal(t, "complete", r.Status)
	require.Equal(t, "abstained", r.Result["status"])
}

// [REQ:REQ-P0-010] Success-shaped output cannot omit evidence or hide conflicts.
func TestAnsweredResultRejectsUnsupportedClaims(t *testing.T) {
	for _, status := range []string{"answered", "invented"} {
		t.Run(status, func(t *testing.T) {
			f := &fakeClient{execution: execution()}
			f.execution.Status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED
			f.execution.EndedAt = timestamppb.Now()
			f.execution.Output, _ = structpb.NewValue(map[string]any{"result": map[string]any{"status": status, "summary": "claim", "claims": []any{}, "gaps": []any{}, "contradictions": []any{}, "finding_ids": []any{}}})
			result, err := agentmanager.NewService(f).GetRunState(context.Background(), "execution-1")
			require.NoError(t, err)
			require.Equal(t, "failed", result.Status)
			require.Contains(t, result.ErrorMsg, "research_result_invalid")
		})
	}
}

// [REQ:REQ-P0-010] Cited results need plausible source identities and dates.
func TestResearchResultCitationValidation(t *testing.T) {
	for _, tc := range []struct {
		name, url, date string
		valid           bool
	}{
		{"supported", "https://primary.example/reference", time.Now().Add(-time.Minute).Format(time.RFC3339), true},
		{"unsafe scheme", "file:///etc/passwd", time.Now().Format(time.RFC3339), false},
		{"unknown date", "https://primary.example/reference", "unknown", false},
		{"future date", "https://primary.example/reference", time.Now().Add(time.Hour).Format(time.RFC3339), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeClient{execution: execution()}
			f.execution.Status = domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED
			f.execution.EndedAt = timestamppb.Now()
			f.execution.Output, _ = structpb.NewValue(map[string]any{"result": map[string]any{"status": "answered", "summary": "Supported claim", "claims": []any{map[string]any{"text": "Supported claim", "citations": []any{map[string]any{"url": tc.url, "title": "Reference", "retrieved_at": tc.date}}}}, "gaps": []any{}, "contradictions": []any{}, "finding_ids": []any{}}})
			r, err := agentmanager.NewService(f).GetRunState(context.Background(), "execution-1")
			require.NoError(t, err)
			if tc.valid {
				require.Equal(t, "complete", r.Status)
				require.NotNil(t, r.Result)
			} else {
				require.Equal(t, "failed", r.Status)
				require.Nil(t, r.Result)
				require.Contains(t, r.ErrorMsg, "research_result_invalid")
			}
		})
	}
}
