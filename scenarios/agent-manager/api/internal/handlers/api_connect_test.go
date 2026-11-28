package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"
	"agent-manager/internal/repository"
	"agent-manager/internal/runreport"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type wrappedNotFoundRunRepository struct {
	repository.RunRepository
	runID string
}

func (r wrappedNotFoundRunRepository) Get(context.Context, uuid.UUID) (*domain.Run, error) {
	return nil, fmt.Errorf("storage lookup: %w", domain.NewNotFoundErrorWithID("Run", r.runID))
}

func TestAgentManagerConnectListRunsUsesBoundedDefault(t *testing.T) {
	handler, _ := setupTestHandler(t)
	router := mux.NewRouter()
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(NewAgentManagerConnectHandler(handler))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: connectHandler})

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)

	response, err := client.ListRuns(context.Background(), connect.NewRequest(&apipb.ListRunsRequest{}))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Msg)
}

func TestAgentManagerConnectMutationAndCompatibilityHandlersRejectMissingRequest(t *testing.T) {
	handler := &AgentManagerConnectHandler{}
	ctx := context.Background()
	checks := []func() error{
		func() error { _, err := handler.CreateTask(ctx, nil); return err },
		func() error { _, err := handler.CreateRun(ctx, nil); return err },
		func() error { _, err := handler.EnsureProfile(ctx, nil); return err },
		func() error { _, err := handler.GetTask(ctx, nil); return err },
		func() error { _, err := handler.GetRunByTag(ctx, nil); return err },
		func() error { _, err := handler.GetRunDiff(ctx, nil); return err },
		func() error { _, err := handler.ApproveRun(ctx, nil); return err },
		func() error { _, err := handler.RejectRun(ctx, nil); return err },
		func() error { _, err := handler.StopRun(ctx, nil); return err },
		func() error { _, err := handler.StopRunByTag(ctx, nil); return err },
		func() error { _, err := handler.SyncRunFromSandbox(ctx, nil); return err },
		func() error { _, err := handler.StopAllRuns(ctx, nil); return err },
		func() error { _, err := handler.ContinueRun(ctx, nil); return err },
		func() error { _, err := handler.GetRunnerStatus(ctx, nil); return err },
	}
	for index, check := range checks {
		if err := check(); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("handler %d error = %v; want invalid_argument", index, err)
		}
	}
}

func TestAgentManagerCompatibilityProceduresReuseRESTHandlers(t *testing.T) {
	handler, restRouter := setupTestHandler(t)
	router := mux.NewRouter()
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(NewAgentManagerConnectHandler(handler))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: connectHandler})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)
	ctx := context.Background()

	ensure, err := client.EnsureProfile(ctx, connect.NewRequest(&apipb.EnsureProfileRequest{
		ProfileKey: "connect-test",
		Defaults:   &domainpb.AgentProfile{Name: "Connect Test", ProfileKey: "connect-test", RoleRef: "code.default", MaxTurns: 10},
	}))
	require.NoError(t, err)
	require.NotNil(t, ensure.Msg.GetProfile())

	runners, err := client.GetRunnerStatus(ctx, connect.NewRequest(&apipb.GetRunnerStatusRequest{}))
	require.NoError(t, err)
	require.NotNil(t, runners.Msg)

	missingRunID := uuid.NewString()
	_, err = client.GetRunDiff(ctx, connect.NewRequest(&apipb.GetRunDiffRequest{RunId: missingRunID}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = client.ApproveRun(ctx, connect.NewRequest(&apipb.ApproveRunRequest{RunId: missingRunID}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = client.RejectRun(ctx, connect.NewRequest(&apipb.RejectRunRequest{RunId: missingRunID, Reason: "test"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = client.ContinueRun(ctx, connect.NewRequest(&domainpb.ContinueRunRequest{RunId: missingRunID, Message: "continue"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	restBody := encodeProtoJSON(t, &domainpb.ContinueRunRequest{RunId: missingRunID, Message: "continue"})
	restRequest := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+missingRunID+"/continue", bytes.NewReader(restBody))
	restResponse := httptest.NewRecorder()
	restRouter.ServeHTTP(restResponse, restRequest)
	require.Equal(t, http.StatusNotFound, restResponse.Code)

	_, err = client.GetTask(ctx, connect.NewRequest(&apipb.GetTaskRequest{TaskId: uuid.NewString()}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = client.GetRunByTag(ctx, connect.NewRequest(&apipb.GetRunByTagRequest{Tag: "missing-connect-tag"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	_, err = client.StopRunByTag(ctx, connect.NewRequest(&apipb.StopRunByTagRequest{Tag: "missing-connect-tag"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = client.StopRun(ctx, connect.NewRequest(&apipb.StopRunRequest{RunId: uuid.NewString()}))
	require.Error(t, err)
	_, err = client.StopAllRuns(ctx, connect.NewRequest(&apipb.StopAllRunsRequest{}))
	require.NoError(t, err)
}

func TestAgentManagerConnectSyncRunFromSandboxReusesOwnerAuthorizationAndState(t *testing.T) {
	handler, _, repos, _ := setupTestHandlerWithRunnerAndRepos(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode))
	task := &domain.Task{ID: uuid.New(), Title: "sandbox connect", ScopePath: ".", Status: domain.TaskStatusQueued}
	require.NoError(t, repos.Tasks.Create(t.Context(), task))
	sandboxID := uuid.New()
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, SandboxID: &sandboxID, Status: domain.RunStatusNeedsReview, Phase: domain.RunPhaseAwaitingReview, ApprovalState: domain.ApprovalStatePending}
	require.NoError(t, repos.Runs.Create(t.Context(), run))

	router := mux.NewRouter()
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(NewAgentManagerConnectHandler(handler))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: connectHandler})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)

	sandboxIDString := sandboxID.String()
	response, err := client.SyncRunFromSandbox(context.Background(), connect.NewRequest(&apipb.SyncRunFromSandboxRequest{
		RunId: run.ID.String(), SandboxId: &sandboxIDString, Status: "approved", Actor: "reviewer", Applied: 2, Remaining: 0, CommitHash: "abc123",
	}))
	require.NoError(t, err)
	require.Equal(t, "synced", response.Msg.GetStatus())
	require.Equal(t, run.ID.String(), response.Msg.GetRunId())
	require.Equal(t, string(domain.RunStatusComplete), response.Msg.GetRunStatus())
	require.Equal(t, string(domain.ApprovalStateApproved), response.Msg.GetApprovalState())

	stored, err := repos.Runs.Get(t.Context(), run.ID)
	require.NoError(t, err)
	require.Equal(t, domain.ApprovalStateApproved, stored.ApprovalState)
	require.Equal(t, "reviewer", stored.ApprovedBy)
}

func TestWrappedRunNotFoundPreservesBounded404AcrossRESTAndConnect(t *testing.T) {
	missingRunID := uuid.New()
	_, router, _, _ := setupTestHandlerWithRunnerAndReposOverride(t, runner.NewMockRunner(domain.RunnerTypeClaudeCode), func(repos *database.Repositories) {
		repos.Runs = wrappedNotFoundRunRepository{RunRepository: repos.Runs, runID: missingRunID.String()}
	})

	requestBody := encodeProtoJSON(t, &domainpb.ContinueRunRequest{RunId: missingRunID.String(), Message: "continue"})
	restRequest := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+missingRunID.String()+"/continue", bytes.NewReader(requestBody))
	restRequest.Header.Set("Authorization", "Bearer "+testOwnerCredential)
	restResponse := httptest.NewRecorder()
	router.ServeHTTP(restResponse, restRequest)
	require.Equal(t, http.StatusNotFound, restResponse.Code)

	var restError commonpb.ErrorResponse
	require.NoError(t, protoconv.UnmarshalJSON(restResponse.Body.Bytes(), &restError))
	require.Equal(t, string(domain.ErrCodeNotFoundRun), restError.GetCode())
	require.Contains(t, restError.GetMessage(), "Run not found")
	require.LessOrEqual(t, len(restError.GetMessage()), 160)
	require.NotContains(t, restError.GetMessage(), "storage lookup")

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)
	_, err := client.ContinueRun(context.Background(), connect.NewRequest(&domainpb.ContinueRunRequest{RunId: missingRunID.String(), Message: "continue"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	require.Contains(t, err.Error(), "Run not found")
	require.LessOrEqual(t, len(err.Error()), 200)
	require.NotContains(t, err.Error(), "storage lookup")
}

type runReportServiceStub struct {
	report *runreport.RunReport
	err    error
}

func (s runReportServiceStub) BuildRunReport(context.Context, uuid.UUID) (*runreport.RunReport, error) {
	return s.report, s.err
}

func TestAgentManagerConnectGetRunReportProjectsBoundedReportAndNotFound(t *testing.T) {
	runID := uuid.New()
	h := &AgentManagerConnectHandler{h: &Handler{svc: orchestration.HandlerServices{
		RunReportService: runReportServiceStub{report: &runreport.RunReport{RunID: runID, Status: "complete", Tokens: 7}},
	}}}

	response, err := h.GetRunReport(context.Background(), connect.NewRequest(&apipb.GetRunReportRequest{RunId: runID.String()}))
	require.NoError(t, err)
	require.Equal(t, runID.String(), response.Msg.GetRunId())
	require.Equal(t, "complete", response.Msg.GetStatus())
	require.EqualValues(t, 7, response.Msg.GetTokens())

	missing := uuid.New()
	h.h.svc.RunReportService = runReportServiceStub{err: domain.NewNotFoundError("Run", missing)}
	_, err = h.GetRunReport(context.Background(), connect.NewRequest(&apipb.GetRunReportRequest{RunId: missing.String()}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
