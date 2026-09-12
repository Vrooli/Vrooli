package runs_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	"flow-verifier/handlers/runs"
	internalruns "flow-verifier/internal/runs"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs/runs_v1connect"
)

func newClient(t *testing.T, repo internalruns.Repository) runsconnect.RunsServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	svc := internalruns.NewService(repo)
	path, handler := runsconnect.NewRunsServiceHandler(runs.NewConnectHandler(runs.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return runsconnect.NewRunsServiceClient(server.Client(), server.URL)
}

func TestListRunsHappyPath(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	client := newClient(t, &fakeRepo{rows: []internalruns.Run{
		{ID: "a", FlowID: "fa", Status: internalruns.StatusPassed, Mode: internalruns.ModeRun, StartedAt: now, FinishedAt: now},
		{ID: "b", FlowID: "fb", Status: internalruns.StatusFailed, FailureReason: "missing_artifacts", Mode: internalruns.ModeCheck, MissingArtifacts: []string{"a/b.go"}},
	}})

	resp, err := client.ListRuns(context.Background(), connect.NewRequest(&runsv1.ListRunsRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Runs, 2)
	require.Equal(t, "a", resp.Msg.Runs[0].Id)
	require.Equal(t, runsv1.RunStatus_RUN_STATUS_PASSED, resp.Msg.Runs[0].Status)
	require.Equal(t, runsv1.RunStatus_RUN_STATUS_FAILED, resp.Msg.Runs[1].Status)
	require.Equal(t, runsv1.FailureReason_FAILURE_REASON_MISSING_ARTIFACTS, resp.Msg.Runs[1].FailureReason)
	require.Equal(t, []string{"a/b.go"}, resp.Msg.Runs[1].MissingArtifacts)
}

func TestGetRunNotFound(t *testing.T) {
	client := newClient(t, &fakeRepo{})
	_, err := client.GetRun(context.Background(), connect.NewRequest(&runsv1.GetRunRequest{Id: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

type fakeRepo struct{ rows []internalruns.Run }

func (f *fakeRepo) Insert(_ context.Context, r internalruns.Run) (internalruns.Run, error) {
	f.rows = append(f.rows, r)
	return r, nil
}

func (f *fakeRepo) Get(_ context.Context, id string) (internalruns.Run, error) {
	for _, r := range f.rows {
		if r.ID == id {
			return r, nil
		}
	}
	return internalruns.Run{}, internalruns.ErrNotFound{ID: id}
}

func (f *fakeRepo) List(_ context.Context, _ internalruns.ListQuery) ([]internalruns.Run, error) {
	return f.rows, nil
}
