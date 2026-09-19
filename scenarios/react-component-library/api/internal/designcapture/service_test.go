package designcapture

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/encoding/protojson"
)

type dispatchFunc func(context.Context, Operation) (string, error)

func (f dispatchFunc) Start(ctx context.Context, op Operation) (string, error) { return f(ctx, op) }
func TestCaptureDispatchPersistsBeforeProducerAndDoesNotRepeat(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	calls := 0
	svc := Service{Repository: repo, Dispatcher: dispatchFunc(func(dispatchCtx context.Context, op Operation) (string, error) {
		calls++
		stored, err := repo.Get(dispatchCtx, op.ID)
		require.NoError(t, err)
		require.Equal(t, Dispatching, stored.State)
		cancel()
		require.NoError(t, dispatchCtx.Err())
		return "producer-run", nil
	})}
	op, err := svc.Start(ctx, "one", captureRequest())
	require.NoError(t, err)
	require.Equal(t, Running, op.State)
	require.Equal(t, "producer-run", op.ProducerID)
	retry, err := svc.Start(context.Background(), "one", captureRequest())
	require.NoError(t, err)
	require.Equal(t, op, retry)
	require.Equal(t, 1, calls)
}
func TestCaptureDispatchUncertaintyCannotRedispatch(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	calls := 0
	svc := Service{Repository: repo, Dispatcher: dispatchFunc(func(context.Context, Operation) (string, error) {
		calls++
		return "", errors.New("connection lost after send")
	})}
	op, err := svc.Start(ctx, "lost", captureRequest())
	require.NoError(t, err)
	require.Equal(t, DispatchUnknown, op.State)
	_, err = svc.Start(ctx, "lost", captureRequest())
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	svc.Dispatcher = dispatchFunc(func(context.Context, Operation) (string, error) {
		return "", NotDispatchedError{Err: errors.New("invalid target URL")}
	})
	rejected, err := svc.Start(ctx, "invalid", captureRequest())
	require.NoError(t, err)
	require.Equal(t, Failed, rejected.State)
}
func TestBASDispatcherUsesTypedExactTargetWorkflow(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "/browser_automation_studio.v1.WorkflowsService/ExecuteAdhocWorkflow", r.URL.Path)
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var request executionv1.ExecuteAdhocRequest
		require.NoError(t, protojson.Unmarshal(raw, &request))
		require.False(t, request.WaitForCompletion)
		require.Contains(t, string(raw), "/design-captures/capture_"+strings.Repeat("a", 64)+"/target.html")
		require.Contains(t, string(raw), "stale_render_target")
		require.Contains(t, string(raw), "document.fonts.ready")
		require.Equal(t, int32(390), request.FlowDefinition.Settings.GetViewportWidth())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"executionId":"bas-run","status":"EXECUTION_STATUS_RUNNING"}`))
	}))
	defer server.Close()
	dispatcher := BASDispatcher{BASBaseURL: server.URL, RCLTargetBaseURL: "http://rcl.local"}
	op := Operation{ID: "capture_" + strings.Repeat("a", 64), State: Dispatching, Request: captureRequest()}
	producer, err := dispatcher.Start(context.Background(), op)
	require.NoError(t, err)
	require.Equal(t, "bas-run", producer)
	require.Equal(t, 1, calls)
	op.Request.HTML += "tampered"
	_, err = dispatcher.Start(context.Background(), op)
	var rejected NotDispatchedError
	require.ErrorAs(t, err, &rejected)
	require.Equal(t, 1, calls)
}

type cancelFunc func(context.Context, Operation) error

func (f cancelFunc) Cancel(ctx context.Context, op Operation) error { return f(ctx, op) }

func TestCaptureCancellationPersistsIntentAndRequiresTerminalEvidence(t *testing.T) {
	ctx, detach := context.WithCancel(context.Background())
	defer detach()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	svc := Service{Repository: repo, Dispatcher: dispatchFunc(func(context.Context, Operation) (string, error) { return "producer-run", nil })}
	op, err := svc.Start(ctx, "cancel-running", captureRequest())
	require.NoError(t, err)
	calls := 0
	canceller := cancelFunc(func(callCtx context.Context, current Operation) error {
		calls++
		stored, err := repo.Get(callCtx, current.ID)
		require.NoError(t, err)
		require.Equal(t, CancelRequested, stored.State)
		require.Equal(t, "producer-run", current.ProducerID)
		detach()
		require.NoError(t, callCtx.Err())
		if calls == 1 {
			return errors.New("acknowledgement lost")
		}
		return nil
	})
	pending, err := svc.Cancel(ctx, op.ID, canceller)
	require.ErrorContains(t, err, "intent retained")
	require.Equal(t, CancelRequested, pending.State)
	pending, err = svc.Cancel(context.Background(), op.ID, canceller)
	require.NoError(t, err)
	require.Equal(t, CancelRequested, pending.State, "stop acknowledgement is not evidence of termination")
	require.Equal(t, 2, calls)
	terminal, err := repo.Transition(context.Background(), pending.ID, pending.Version, Cancelled, pending.ProducerID, nil, "Producer confirmed cancellation")
	require.NoError(t, err)
	again, err := svc.Cancel(context.Background(), op.ID, canceller)
	require.NoError(t, err)
	require.Equal(t, terminal, again)
	require.Equal(t, 2, calls)
}

func TestCaptureCancellationBeforeDispatchAndUnknownDispatch(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	svc := Service{Repository: repo}
	canceller := cancelFunc(func(context.Context, Operation) error {
		t.Fatal("must not contact an unidentified producer")
		return nil
	})
	op, err := repo.Create(ctx, "prepared", captureRequest())
	require.NoError(t, err)
	op, err = svc.Cancel(ctx, op.ID, canceller)
	require.NoError(t, err)
	require.Equal(t, Cancelled, op.State)
	svc.Dispatcher = dispatchFunc(func(context.Context, Operation) (string, error) { return "", errors.New("lost") })
	unknown, err := svc.Start(ctx, "unknown", captureRequest())
	require.NoError(t, err)
	_, err = svc.Cancel(ctx, unknown.ID, canceller)
	require.ErrorContains(t, err, "recover dispatch identity")
	stored, err := repo.Get(ctx, unknown.ID)
	require.NoError(t, err)
	require.Equal(t, unknown, stored)
}

func TestCaptureCancellationPreservesConcurrentProducerOutcome(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	svc := Service{Repository: repo, Dispatcher: dispatchFunc(func(context.Context, Operation) (string, error) { return "producer-run", nil })}
	op, err := svc.Start(ctx, "race", captureRequest())
	require.NoError(t, err)
	result, err := svc.Cancel(ctx, op.ID, cancelFunc(func(ctx context.Context, pending Operation) error {
		_, err := repo.Transition(ctx, pending.ID, pending.Version, Failed, pending.ProducerID, nil, "Producer failed while stop was in flight")
		require.NoError(t, err)
		return errors.New("stop response lost")
	}))
	require.NoError(t, err)
	require.Equal(t, Failed, result.State)
}

func TestBASCancellationUsesProducerStopAcknowledgement(t *testing.T) {
	for _, response := range []string{`{"status":"stopped"}`, `{}`, `{"status":"running"}`} {
		t.Run(response, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/browser_automation_studio.v1.ExecutionsService/StopExecution", r.URL.Path)
				raw, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var request apiv1.StopExecutionRequest
				require.NoError(t, protojson.Unmarshal(raw, &request))
				require.Equal(t, "producer-run", request.ExecutionId)
				_, _ = w.Write([]byte(response))
			}))
			defer server.Close()
			err := (BASDispatcher{BASBaseURL: server.URL}).Cancel(context.Background(), Operation{State: CancelRequested, ProducerID: "producer-run"})
			if response == `{"status":"stopped"}` {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestRetryCaptureCreatesLinkedAttemptOnlyAfterTerminalOutcome(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	calls := 0
	svc := Service{Repository: repo, Dispatcher: dispatchFunc(func(context.Context, Operation) (string, error) { calls++; return "producer", nil })}
	original, err := svc.Start(ctx, "original", captureRequest())
	require.NoError(t, err)
	_, err = svc.Retry(ctx, original.ID, "early")
	require.ErrorContains(t, err, "terminal")
	require.Equal(t, 1, calls)
	terminal, err := repo.Transition(ctx, original.ID, original.Version, Cancelled, original.ProducerID, nil, "cancelled")
	require.NoError(t, err)
	_, err = svc.Retry(ctx, terminal.ID, "original")
	require.ErrorIs(t, err, ErrConflict)
	retry, err := svc.Retry(ctx, terminal.ID, "retry")
	require.NoError(t, err)
	require.NotEqual(t, terminal.ID, retry.ID)
	require.Equal(t, terminal.ID, retry.Request.PreviousID)
	require.Equal(t, terminal.Request.HTML, retry.Request.HTML)
	require.Equal(t, terminal.Request.Target, retry.Request.Target)
	repeated, err := svc.Retry(ctx, terminal.ID, "retry")
	require.NoError(t, err)
	require.Equal(t, retry, repeated)
	require.Equal(t, 2, calls)
	stored, err := repo.Get(ctx, terminal.ID)
	require.NoError(t, err)
	require.Equal(t, terminal, stored)
}
