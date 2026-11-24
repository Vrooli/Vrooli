package capture

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	_ "modernc.org/sqlite"
)

type idempotentExecutor struct {
	*fakeExecutor
	state       map[uuid.UUID]*database.ExecutionIndex
	waitErrOnce error
}

func (e *idempotentExecutor) ExecuteAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *workflow.ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
	resp, err := e.fakeExecutor.ExecuteAdhocWorkflowAPIWithOptions(ctx, req, opts)
	if err != nil {
		return resp, err
	}
	if opts == nil || opts.ExecutionID == nil {
		return resp, errors.New("reserved execution ID was not supplied")
	}
	id := *opts.ExecutionID
	now := time.Now()
	e.state[id] = &database.ExecutionIndex{ID: id, Status: database.ExecutionStatusCompleted, CompletedAt: &now}
	resp.ExecutionId = id.String()
	return resp, nil
}

func (e *idempotentExecutor) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if row := e.state[id]; row != nil {
		return row, nil
	}
	return nil, database.ErrNotFound
}

func (e *idempotentExecutor) WaitForExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if e.waitErrOnce != nil {
		err := e.waitErrOnce
		e.waitErrOnce = nil
		return nil, err
	}
	return e.GetExecution(ctx, id)
}

func idempotencyTestServer(t *testing.T) (captureconnect.CaptureServiceClient, *idempotentExecutor) {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE capture_requests (request_token TEXT PRIMARY KEY, request_hash TEXT NOT NULL, request_json TEXT NOT NULL, execution_id TEXT NOT NULL UNIQUE, state TEXT NOT NULL, response_json TEXT NOT NULL DEFAULT '', error TEXT NOT NULL DEFAULT '', created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	executor := &idempotentExecutor{fakeExecutor: &fakeExecutor{}, state: make(map[uuid.UUID]*database.ExecutionIndex)}
	deps := Deps{Executor: executor, RequestStore: NewSQLCaptureRequestStore(db), Logger: logrus.New(), CapturesRoot: t.TempDir(), Now: func() time.Time { return time.Unix(0, 0) }}
	mount := Module(deps)
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := testutil.StartHTTPServer(t, mux)
	t.Cleanup(srv.Close)
	return captureconnect.NewCaptureServiceClient(srv.Client(), srv.URL), executor
}

func TestCaptureRequestTokenReplaysAndLooksUpSameCapture(t *testing.T) {
	client, executor := idempotencyTestServer(t)
	request := &capturev1.CaptureRequest{Url: "https://example.com", RequestToken: "rcl-state:page-default"}
	first, err := client.Capture(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	second, err := client.Capture(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, first.Msg.GetExecutionId(), second.Msg.GetExecutionId())
	require.Equal(t, 1, executor.Calls, "a completed token replay must not execute another browser run")

	lookup, err := client.GetCaptureByRequestToken(context.Background(), connect.NewRequest(&capturev1.GetCaptureByRequestTokenRequest{RequestToken: "rcl-state:page-default"}))
	require.NoError(t, err)
	require.Equal(t, capturev1.GetCaptureByRequestTokenResponse_STATE_COMPLETED, lookup.Msg.GetState())
	require.Equal(t, first.Msg.GetExecutionId(), lookup.Msg.GetCapture().GetExecutionId())
}

func TestCaptureRequestTokenChangedInputConflictsBeforeDispatch(t *testing.T) {
	client, executor := idempotencyTestServer(t)
	_, err := client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{Url: "https://example.com", RequestToken: "same"}))
	require.NoError(t, err)
	_, err = client.Capture(context.Background(), connect.NewRequest(&capturev1.CaptureRequest{Url: "https://different.example", RequestToken: "same"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
	require.Equal(t, 1, executor.Calls)
}

func TestCaptureRequestTokenRecoversAfterWaitResponseIsLost(t *testing.T) {
	client, executor := idempotencyTestServer(t)
	executor.waitErrOnce = context.DeadlineExceeded
	request := &capturev1.CaptureRequest{Url: "https://example.com", RequestToken: "recover-this"}
	_, err := client.Capture(context.Background(), connect.NewRequest(request))
	require.Error(t, err)
	pending, err := client.GetCaptureByRequestToken(context.Background(), connect.NewRequest(&capturev1.GetCaptureByRequestTokenRequest{RequestToken: "recover-this"}))
	require.NoError(t, err)
	require.Equal(t, capturev1.GetCaptureByRequestTokenResponse_STATE_IN_PROGRESS, pending.Msg.GetState())
	require.NotEmpty(t, pending.Msg.GetExecutionId(), "lookup must return the stable execution ID needed for recovery")

	recovered, err := client.Capture(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	require.NotEmpty(t, recovered.Msg.GetExecutionId())
	require.Equal(t, 1, executor.Calls, "retry must join the preallocated execution instead of starting another")

	lookup, err := client.GetCaptureByRequestToken(context.Background(), connect.NewRequest(&capturev1.GetCaptureByRequestTokenRequest{RequestToken: "recover-this"}))
	require.NoError(t, err)
	require.Equal(t, capturev1.GetCaptureByRequestTokenResponse_STATE_COMPLETED, lookup.Msg.GetState())
}
