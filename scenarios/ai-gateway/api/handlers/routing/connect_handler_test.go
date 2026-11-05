package routing_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	handler "ai-gateway/handlers/routing"
	internalrouting "ai-gateway/internal/routing"
)

func TestPreviewRouteUsesGatewayValidationBeforeRouting(t *testing.T) { // [REQ:AIGW-ROUTE-PREVIEW]
	h := handler.NewConnectHandler(handler.Deps{})
	resp, err := h.PreviewRoute(context.Background(), connect.NewRequest(&routingv1.PreviewRouteRequest{
		Request: &sharedv1.GatewayRequest{
			Kind:         sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
			Role:         "chat.default",
			Profile:      sharedv1.Profile_PROFILE_REMOTE_ONLY,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET,
		},
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetValid())
	require.Len(t, resp.Msg.GetIssues(), 1)
	require.Equal(t, "privacy_profile_conflict", resp.Msg.GetIssues()[0].GetCode())
	require.Empty(t, resp.Msg.GetCandidates(), "Phase 2 preview must not synthesize provider candidates")
}

func TestPrivatePlanDoesNotAcceptRequestControlledUseClass(t *testing.T) {
	router := mux.NewRouter()
	handler.Module(handler.Deps{}).Mount(router)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/private/provider-routes/plan", strings.NewReader(`{"operation_id":"op-1","feature_id":"demo.feature","model":"model-a","budget_micros":100,"use_class":"own_account"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "unknown field")
}

func TestPrivatePlanUsesOnlyRegisteredFeatures(t *testing.T) {
	router := mux.NewRouter()
	handler.Module(handler.Deps{}).Mount(router)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/private/provider-routes/plan", strings.NewReader(`{"operation_id":"op-2","feature_id":"not.registered","model":"model-a","budget_micros":100}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Contains(t, response.Body.String(), "feature_not_found")
}

func TestSubmitMediaNeverManufacturesAnExecutionWithoutDurableStore(t *testing.T) { // [REQ:AIGW-MEDIA-EXECUTION]
	h := handler.NewConnectHandler(handler.Deps{})
	_, err := h.SubmitMedia(context.Background(), connect.NewRequest(&routingv1.SubmitMediaRequest{
		Request: &sharedv1.GatewayRequest{
			Kind:         sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION,
			Role:         "media.generate",
			Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		},
		Prompt:         "a red square",
		OutputCount:    1,
		IdempotencyKey: "media-test-1",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestMediaHandlerMapsIdempotencyConflictAndUnknownRetry(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(internalrouting.Schema)))
	h := handler.NewConnectHandler(handler.Deps{DB: db})
	request := &routingv1.SubmitMediaRequest{
		Request: &sharedv1.GatewayRequest{Kind: sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION, Role: "media.generate", Scenario: "fixture", Operation: "generate"},
		Prompt:  "a red square", OutputCount: 1, IdempotencyKey: "handler-binding", OutputReference: filepath.Join(t.TempDir(), "output.png"),
	}
	first, err := h.SubmitMedia(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	_, err = h.WaitMediaExecution(context.Background(), connect.NewRequest(&routingv1.WaitMediaExecutionRequest{ExecutionId: first.Msg.GetExecution().GetExecutionId()}))
	require.NoError(t, err)
	changed := &routingv1.SubmitMediaRequest{Request: request.Request, Prompt: "changed", OutputCount: request.OutputCount, IdempotencyKey: request.IdempotencyKey, OutputReference: request.OutputReference}
	_, err = h.SubmitMedia(context.Background(), connect.NewRequest(changed))
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
	_, err = db.Exec(`UPDATE media_executions SET error_code = 'outcome_unknown', error_message = 'provider outcome unknown; cost unknown' WHERE execution_id = ?`, first.Msg.GetExecution().GetExecutionId())
	require.NoError(t, err)
	_, err = h.RetryMediaExecution(context.Background(), connect.NewRequest(&routingv1.RetryMediaExecutionRequest{ExecutionId: first.Msg.GetExecution().GetExecutionId(), IdempotencyKey: "retry-key"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	_, err = db.Exec(`UPDATE media_executions SET request_json = '{' WHERE execution_id = ?`, first.Msg.GetExecution().GetExecutionId())
	require.NoError(t, err)
	_, err = h.SubmitMedia(context.Background(), connect.NewRequest(request))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

type recoveryCountingExecutor struct{ calls atomic.Int64 }

var _ internalrouting.MediaExecutor = (*recoveryCountingExecutor)(nil)

func (e *recoveryCountingExecutor) Execute(context.Context, *routingv1.SubmitMediaRequest) (*internalrouting.MediaExecutionResult, error) {
	e.calls.Add(1)
	return nil, errors.New("unexpected dispatch")
}

func (*recoveryCountingExecutor) Cancel(context.Context, string) error { return nil }

func TestMediaStartupReportsQuarantineWriteFailureWithoutDispatch(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(internalrouting.Schema)))
	_, err := db.Exec(`INSERT INTO media_executions(execution_id,idempotency_key,status,created_at,request_json) VALUES('interrupted','interrupted',2,'2026-11-01T00:00:00Z','{');
	 CREATE TRIGGER prevent_media_quarantine BEFORE UPDATE ON media_executions
	 WHEN NEW.error_code = 'outcome_unknown' BEGIN SELECT RAISE(FAIL, 'quarantine storage failure'); END;`)
	require.NoError(t, err)
	executor := &recoveryCountingExecutor{}
	var reports []error
	reporter := handler.MediaRecoveryReporter(func(err error) { reports = append(reports, err) })
	handler.Module(handler.Deps{DB: db, MediaExecutor: executor, ReportMediaRecoveryError: reporter})
	require.Len(t, reports, 1)
	require.ErrorContains(t, reports[0], "quarantine interrupted media executions")
	require.Zero(t, executor.calls.Load())
	var status int32
	require.NoError(t, db.QueryRow(`SELECT status FROM media_executions WHERE execution_id = 'interrupted'`).Scan(&status))
	require.Equal(t, int32(routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING), status)
}
