package agentmanager_test

import (
	"context"
	"testing"

	"web-search/internal/research/agentmanager"

	"github.com/stretchr/testify/require"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// fakeClient is a canned agent-manager Client.
type fakeClient struct {
	createdTask *domainpb.Task
	createdRun  *domainpb.Run
	gotTask     *domainpb.Task
	gotRunReq   *apipb.CreateRunRequest
	runByID     map[string]*domainpb.Run
}

func (f *fakeClient) CreateTask(_ context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	f.gotTask = task
	return f.createdTask, nil
}

func (f *fakeClient) CreateRun(_ context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	f.gotRunReq = req
	return f.createdRun, nil
}

func (f *fakeClient) GetRun(_ context.Context, runID string) (*domainpb.Run, error) {
	return f.runByID[runID], nil
}

func TestSpawnCreatesTaskThenRun(t *testing.T) {
	client := &fakeClient{
		createdTask: &domainpb.Task{Id: "task-1"},
		createdRun:  &domainpb.Run{Id: "run-1", Status: domainpb.RunStatus_RUN_STATUS_PENDING},
	}
	svc := agentmanager.NewService(client)

	res, err := svc.Spawn(context.Background(), agentmanager.SpawnRequest{Query: "q", Prompt: "do research"})
	require.NoError(t, err)
	require.Equal(t, "task-1", res.TaskID)
	require.Equal(t, "run-1", res.RunID)
	require.Equal(t, "pending", res.Status)

	// The run was created against the task with a fresh conversation id + prompt.
	require.Equal(t, "task-1", client.gotRunReq.TaskId)
	require.NotNil(t, client.gotRunReq.ConversationId)
	require.NotEmpty(t, *client.gotRunReq.ConversationId)
	require.NotNil(t, client.gotRunReq.Prompt)
	require.Equal(t, "do research", *client.gotRunReq.Prompt)
}

func TestGetRunStateNormalizesStatus(t *testing.T) {
	client := &fakeClient{runByID: map[string]*domainpb.Run{
		"run-1": {Id: "run-1", Status: domainpb.RunStatus_RUN_STATUS_COMPLETE, Summary: &domainpb.RunSummary{Description: "done"}},
	}}
	svc := agentmanager.NewService(client)

	state, err := svc.GetRunState(context.Background(), "run-1")
	require.NoError(t, err)
	require.Equal(t, "complete", state.Status)
	require.Equal(t, "done", state.Summary)
}
