package workflows

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/apihttp"
	coredb "github.com/vrooli/api-core/database"

	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/credits"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// ---------------------------------------------------------------------------
// fakes
// ---------------------------------------------------------------------------

type fakeCatalog struct {
	createResp      *basapi.CreateWorkflowResponse
	createErr       error
	listResp        *basapi.ListWorkflowsResponse
	listErr         error
	getResp         *basapi.GetWorkflowResponse
	getErr          error
	updateResp      *basapi.UpdateWorkflowResponse
	updateErr       error
	deleteResp      *basapi.DeleteWorkflowResponse
	deleteErr       error
	versionsResp    *basapi.WorkflowVersionList
	versionsErr     error
	versionResp     *basapi.WorkflowVersion
	versionErr      error
	restoreResp     *basapi.RestoreWorkflowVersionResponse
	restoreErr      error
	modifyResp      *basapi.UpdateWorkflowResponse
	modifyErr       error
	lastUpdate      *basapi.UpdateWorkflowRequest
	lastModifyID    uuid.UUID
	lastModifyText  string
	lastListReq     *basapi.ListWorkflowsRequest
	lastVersionsArg uuid.UUID
	lastVersionArg  struct {
		id  uuid.UUID
		ver int32
	}
	lastRestoreArg struct {
		id  uuid.UUID
		ver int32
		msg string
	}
}

func (f *fakeCatalog) CreateWorkflow(_ context.Context, _ *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error) {
	return f.createResp, f.createErr
}

func (f *fakeCatalog) ListWorkflows(_ context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error) {
	f.lastListReq = req
	return f.listResp, f.listErr
}

func (f *fakeCatalog) GetWorkflowAPI(_ context.Context, _ *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error) {
	return f.getResp, f.getErr
}

func (f *fakeCatalog) UpdateWorkflow(_ context.Context, req *basapi.UpdateWorkflowRequest) (*basapi.UpdateWorkflowResponse, error) {
	f.lastUpdate = req
	return f.updateResp, f.updateErr
}

func (f *fakeCatalog) DeleteWorkflow(_ context.Context, _ *basapi.DeleteWorkflowRequest) (*basapi.DeleteWorkflowResponse, error) {
	return f.deleteResp, f.deleteErr
}

func (f *fakeCatalog) ListWorkflowVersionsAPI(_ context.Context, id uuid.UUID) (*basapi.WorkflowVersionList, error) {
	f.lastVersionsArg = id
	return f.versionsResp, f.versionsErr
}

func (f *fakeCatalog) GetWorkflowVersionAPI(_ context.Context, id uuid.UUID, ver int32) (*basapi.WorkflowVersion, error) {
	f.lastVersionArg.id = id
	f.lastVersionArg.ver = ver
	return f.versionResp, f.versionErr
}

func (f *fakeCatalog) RestoreWorkflowVersionAPI(_ context.Context, id uuid.UUID, ver int32, msg string) (*basapi.RestoreWorkflowVersionResponse, error) {
	f.lastRestoreArg.id = id
	f.lastRestoreArg.ver = ver
	f.lastRestoreArg.msg = msg
	return f.restoreResp, f.restoreErr
}

func (f *fakeCatalog) ModifyWorkflowAPI(_ context.Context, id uuid.UUID, text string, _ *basworkflows.WorkflowDefinitionV2) (*basapi.UpdateWorkflowResponse, error) {
	f.lastModifyID = id
	f.lastModifyText = text
	return f.modifyResp, f.modifyErr
}

type fakeExecutor struct {
	execResp      *basapi.ExecuteWorkflowResponse
	execErr       error
	adhocResp     *basexecution.ExecuteAdhocResponse
	adhocErr      error
	lastExecReq   *basapi.ExecuteWorkflowRequest
	lastExecOpts  *workflowservice.ExecuteOptions
	lastAdhocReq  *basexecution.ExecuteAdhocRequest
	lastAdhocOpts *workflowservice.ExecuteOptions
	lastAdhocCtx  context.Context
}

func (f *fakeExecutor) ExecuteWorkflowAPIWithOptions(_ context.Context, req *basapi.ExecuteWorkflowRequest, opts *workflowservice.ExecuteOptions) (*basapi.ExecuteWorkflowResponse, error) {
	f.lastExecReq = req
	f.lastExecOpts = opts
	return f.execResp, f.execErr
}

func (f *fakeExecutor) ExecuteAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *workflowservice.ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
	f.lastAdhocCtx = ctx
	f.lastAdhocReq = req
	f.lastAdhocOpts = opts
	return f.adhocResp, f.adhocErr
}

type fakeValidator struct {
	result *basapi.WorkflowValidationResult
	called int
}

func (f *fakeValidator) ValidateDefinition(_ context.Context, _ *basworkflows.WorkflowDefinitionV2) *basapi.WorkflowValidationResult {
	f.called++
	return f.result
}

type fakeSeedRunner struct {
	applyToken string
	applyState map[string]any
	applyErr   error
	cleanups   []string
	cleanupErr error
}

func (f *fakeSeedRunner) ApplySeed(_ context.Context, _ string, _ bool) (string, map[string]any, error) {
	return f.applyToken, f.applyState, f.applyErr
}

func (f *fakeSeedRunner) CleanupSeed(_ context.Context, _ string, token string) error {
	f.cleanups = append(f.cleanups, token)
	return f.cleanupErr
}

type fakeSeedScheduler struct {
	scheduled [][3]string
	err       error
}

func (f *fakeSeedScheduler) Schedule(executionID, scenario, token string) error {
	f.scheduled = append(f.scheduled, [3]string{executionID, scenario, token})
	return f.err
}

type fakeCreditCharger struct {
	calls []credits.ChargeRequest
	err   error
}

func (f *fakeCreditCharger) Charge(_ context.Context, req credits.ChargeRequest) (*credits.ChargeResult, error) {
	f.calls = append(f.calls, req)
	if f.err != nil {
		return nil, f.err
	}
	return &credits.ChargeResult{}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newServerAndClient(t *testing.T, deps Deps) (apiconnect.WorkflowsServiceClient, func()) {
	t.Helper()
	if deps.Logger == nil {
		l := logrus.New()
		l.SetLevel(logrus.PanicLevel)
		deps.Logger = l
	}
	if deps.UserIdentity == nil {
		deps.UserIdentity = func(context.Context) string { return "test-user" }
	}
	mount := Module(deps)
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	client := apiconnect.NewWorkflowsServiceClient(srv.Client(), srv.URL)
	return client, srv.Close
}

// ---------------------------------------------------------------------------
// Module wiring
// ---------------------------------------------------------------------------

func TestModulePanicsWithoutDeps(t *testing.T) {
	t.Run("logger", func(t *testing.T) {
		require.Panics(t, func() { Module(Deps{}) })
	})
	t.Run("catalog", func(t *testing.T) {
		require.Panics(t, func() { Module(Deps{Logger: logrus.New()}) })
	})
	t.Run("executor", func(t *testing.T) {
		require.Panics(t, func() { Module(Deps{Logger: logrus.New(), Catalog: &fakeCatalog{}}) })
	})
	t.Run("validator", func(t *testing.T) {
		require.Panics(t, func() {
			Module(Deps{Logger: logrus.New(), Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}})
		})
	})
}

// ---------------------------------------------------------------------------
// ListWorkflows
// ---------------------------------------------------------------------------

func TestListWorkflows(t *testing.T) {
	cat := &fakeCatalog{listResp: &basapi.ListWorkflowsResponse{Total: 7}}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	resp, err := client.ListWorkflows(context.Background(), connect.NewRequest(&basapi.ListWorkflowsRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(7), resp.Msg.GetTotal())
}

func TestListWorkflows_RepoError(t *testing.T) {
	cat := &fakeCatalog{listErr: errors.New("db down")}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	_, err := client.ListWorkflows(context.Background(), connect.NewRequest(&basapi.ListWorkflowsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

// ---------------------------------------------------------------------------
// GetWorkflow
// ---------------------------------------------------------------------------

func TestGetWorkflow_InvalidID(t *testing.T) {
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	_, err := client.GetWorkflow(context.Background(), connect.NewRequest(&basapi.GetWorkflowRequest{WorkflowId: "nope"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGetWorkflow_NotFound(t *testing.T) {
	cat := &fakeCatalog{getErr: database.ErrNotFound}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	_, err := client.GetWorkflow(context.Background(), connect.NewRequest(&basapi.GetWorkflowRequest{WorkflowId: uuid.NewString()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGetWorkflow_OK(t *testing.T) {
	cat := &fakeCatalog{getResp: &basapi.GetWorkflowResponse{Workflow: &basapi.WorkflowSummary{Id: "abc"}}}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	resp, err := client.GetWorkflow(context.Background(), connect.NewRequest(&basapi.GetWorkflowRequest{WorkflowId: uuid.NewString()}))
	require.NoError(t, err)
	require.Equal(t, "abc", resp.Msg.GetWorkflow().GetId())
}

// ---------------------------------------------------------------------------
// CreateWorkflow
// ---------------------------------------------------------------------------

func TestCreateWorkflow_ValidationErrors(t *testing.T) {
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	t.Run("missing project_id", func(t *testing.T) {
		_, err := client.CreateWorkflow(context.Background(), connect.NewRequest(&basapi.CreateWorkflowRequest{Name: "x"}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("missing name", func(t *testing.T) {
		_, err := client.CreateWorkflow(context.Background(), connect.NewRequest(&basapi.CreateWorkflowRequest{ProjectId: uuid.NewString()}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}

func TestCreateWorkflow_Conflict(t *testing.T) {
	cat := &fakeCatalog{createErr: workflowservice.ErrWorkflowNameConflict}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	_, err := client.CreateWorkflow(context.Background(), connect.NewRequest(&basapi.CreateWorkflowRequest{
		ProjectId: uuid.NewString(),
		Name:      "Demo",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
}

func TestCreateWorkflow_OK(t *testing.T) {
	cat := &fakeCatalog{createResp: &basapi.CreateWorkflowResponse{Workflow: &basapi.WorkflowSummary{Id: "1"}}}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	resp, err := client.CreateWorkflow(context.Background(), connect.NewRequest(&basapi.CreateWorkflowRequest{
		ProjectId: uuid.NewString(),
		Name:      "Demo",
	}))
	require.NoError(t, err)
	require.Equal(t, "1", resp.Msg.GetWorkflow().GetId())
}

// ---------------------------------------------------------------------------
// UpdateWorkflow
// ---------------------------------------------------------------------------

func TestUpdateWorkflow_InvalidID(t *testing.T) {
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()
	_, err := client.UpdateWorkflow(context.Background(), connect.NewRequest(&basapi.UpdateWorkflowRequest{WorkflowId: stringPtr("nope")}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestUpdateWorkflow_VersionConflict(t *testing.T) {
	cat := &fakeCatalog{updateErr: workflowservice.ErrWorkflowVersionConflict}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()
	_, err := client.UpdateWorkflow(context.Background(), connect.NewRequest(&basapi.UpdateWorkflowRequest{WorkflowId: stringPtr(uuid.NewString())}))
	require.Error(t, err)
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
}

// ---------------------------------------------------------------------------
// DeleteWorkflow
// ---------------------------------------------------------------------------

func TestDeleteWorkflow_NotFound(t *testing.T) {
	cat := &fakeCatalog{deleteErr: database.ErrNotFound}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()
	_, err := client.DeleteWorkflow(context.Background(), connect.NewRequest(&basapi.DeleteWorkflowRequest{WorkflowId: uuid.NewString()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// ---------------------------------------------------------------------------
// ExecuteWorkflow
// ---------------------------------------------------------------------------

func TestExecuteWorkflow_InvalidID(t *testing.T) {
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()
	_, err := client.ExecuteWorkflow(context.Background(), connect.NewRequest(&basapi.ExecuteWorkflowRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestExecuteWorkflow_OK_NoOptions(t *testing.T) {
	exec := &fakeExecutor{execResp: &basapi.ExecuteWorkflowResponse{ExecutionId: "exec-1"}}
	charger := &fakeCreditCharger{}
	client, stop := newServerAndClient(t, Deps{
		Catalog:       &fakeCatalog{},
		Executor:      exec,
		Validator:     &fakeValidator{},
		CreditService: charger,
	})
	defer stop()

	resp, err := client.ExecuteWorkflow(context.Background(), connect.NewRequest(&basapi.ExecuteWorkflowRequest{
		WorkflowId: uuid.NewString(),
	}))
	require.NoError(t, err)
	require.Equal(t, "exec-1", resp.Msg.GetExecutionId())
	require.Nil(t, exec.lastExecOpts) // no options provided ⇒ nil
	require.Len(t, charger.calls, 1)
	require.Equal(t, credits.OpExecutionRun, charger.calls[0].Operation)
}

func TestExecuteWorkflow_OptionsTranslated(t *testing.T) {
	exec := &fakeExecutor{execResp: &basapi.ExecuteWorkflowResponse{ExecutionId: "exec-2"}}
	client, stop := newServerAndClient(t, Deps{
		Catalog:   &fakeCatalog{},
		Executor:  exec,
		Validator: &fakeValidator{},
	})
	defer stop()

	q := int32(80)
	_, err := client.ExecuteWorkflow(context.Background(), connect.NewRequest(&basapi.ExecuteWorkflowRequest{
		WorkflowId: uuid.NewString(),
		Options: &basexecution.ExecuteWorkflowOptions{
			RequiresVideo:         true,
			FrameStreaming:        true,
			FrameStreamingQuality: &q,
		},
	}))
	require.NoError(t, err)
	require.NotNil(t, exec.lastExecOpts)
	require.True(t, exec.lastExecOpts.RequiresVideo)
	require.True(t, exec.lastExecOpts.EnableFrameStreaming)
	require.Equal(t, 80, exec.lastExecOpts.FrameStreamingQuality)
}

func TestExecuteWorkflow_SeedSelfReference(t *testing.T) {
	exec := &fakeExecutor{execResp: &basapi.ExecuteWorkflowResponse{ExecutionId: "exec-x"}}
	seed := &fakeSeedRunner{applyToken: "tok", applyState: map[string]any{}}
	client, stop := newServerAndClient(t, Deps{
		Catalog:    &fakeCatalog{},
		Executor:   exec,
		Validator:  &fakeValidator{},
		SeedRunner: seed,
	})
	defer stop()

	_, err := client.ExecuteWorkflow(context.Background(), connect.NewRequest(&basapi.ExecuteWorkflowRequest{
		WorkflowId: uuid.NewString(),
		Options: &basexecution.ExecuteWorkflowOptions{
			SeedMode:     "needs-applying",
			SeedScenario: "browser-automation-studio",
		},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestExecuteWorkflow_SeedSync(t *testing.T) {
	exec := &fakeExecutor{execResp: &basapi.ExecuteWorkflowResponse{ExecutionId: "exec-3"}}
	seed := &fakeSeedRunner{applyToken: "tok-123", applyState: map[string]any{"k": "v"}}
	client, stop := newServerAndClient(t, Deps{
		Catalog:    &fakeCatalog{},
		Executor:   exec,
		Validator:  &fakeValidator{},
		SeedRunner: seed,
	})
	defer stop()

	_, err := client.ExecuteWorkflow(context.Background(), connect.NewRequest(&basapi.ExecuteWorkflowRequest{
		WorkflowId:        uuid.NewString(),
		WaitForCompletion: true,
		Options: &basexecution.ExecuteWorkflowOptions{
			SeedMode:     "needs-applying",
			SeedScenario: "test-genie",
		},
	}))
	require.NoError(t, err)
	require.Equal(t, []string{"tok-123"}, seed.cleanups)
}

func TestExecuteWorkflow_SeedAsync_NoScheduler(t *testing.T) {
	exec := &fakeExecutor{execResp: &basapi.ExecuteWorkflowResponse{ExecutionId: "exec-4"}}
	seed := &fakeSeedRunner{applyToken: "tok-456", applyState: map[string]any{}}
	client, stop := newServerAndClient(t, Deps{
		Catalog:    &fakeCatalog{},
		Executor:   exec,
		Validator:  &fakeValidator{},
		SeedRunner: seed,
	})
	defer stop()

	_, err := client.ExecuteWorkflow(context.Background(), connect.NewRequest(&basapi.ExecuteWorkflowRequest{
		WorkflowId: uuid.NewString(),
		Options: &basexecution.ExecuteWorkflowOptions{
			SeedMode:     "needs-applying",
			SeedScenario: "test-genie",
		},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestExecuteWorkflow_SeedAsync_Scheduled(t *testing.T) {
	exec := &fakeExecutor{execResp: &basapi.ExecuteWorkflowResponse{ExecutionId: "exec-5"}}
	seed := &fakeSeedRunner{applyToken: "tok-789", applyState: map[string]any{}}
	sched := &fakeSeedScheduler{}
	client, stop := newServerAndClient(t, Deps{
		Catalog:       &fakeCatalog{},
		Executor:      exec,
		Validator:     &fakeValidator{},
		SeedRunner:    seed,
		SeedScheduler: sched,
	})
	defer stop()

	_, err := client.ExecuteWorkflow(context.Background(), connect.NewRequest(&basapi.ExecuteWorkflowRequest{
		WorkflowId: uuid.NewString(),
		Options: &basexecution.ExecuteWorkflowOptions{
			SeedMode:     "needs-applying",
			SeedScenario: "test-genie",
		},
	}))
	require.NoError(t, err)
	require.Len(t, sched.scheduled, 1)
	require.Equal(t, "exec-5", sched.scheduled[0][0])
	require.Equal(t, "test-genie", sched.scheduled[0][1])
	require.Equal(t, "tok-789", sched.scheduled[0][2])
}

// ---------------------------------------------------------------------------
// ExecuteAdhocWorkflow
// ---------------------------------------------------------------------------

func TestExecuteAdhocWorkflow_MissingFlow(t *testing.T) {
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	_, err := client.ExecuteAdhocWorkflow(context.Background(), connect.NewRequest(&basexecution.ExecuteAdhocRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestExecuteAdhocWorkflow_OK(t *testing.T) {
	exec := &fakeExecutor{adhocResp: &basexecution.ExecuteAdhocResponse{ExecutionId: "adhoc-1"}}
	client, stop := newServerAndClient(t, Deps{
		Catalog:   &fakeCatalog{},
		Executor:  exec,
		Validator: &fakeValidator{},
	})
	defer stop()

	resp, err := client.ExecuteAdhocWorkflow(context.Background(), connect.NewRequest(&basexecution.ExecuteAdhocRequest{
		FlowDefinition: &basworkflows.WorkflowDefinitionV2{},
	}))
	require.NoError(t, err)
	require.Equal(t, "adhoc-1", resp.Msg.GetExecutionId())
}

func TestExecuteAdhocWorkflow_PreservesTestModeContextFromConnectHeader(t *testing.T) {
	t.Setenv(apihttp.TestModeForceEnableEnv, "1")
	exec := &fakeExecutor{adhocResp: &basexecution.ExecuteAdhocResponse{ExecutionId: "adhoc-test-mode"}}
	deps := Deps{Catalog: &fakeCatalog{}, Executor: exec, Validator: &fakeValidator{}, Logger: logrus.New()}
	mount := Module(deps)
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(apihttp.TestModeMiddleware(mux))
	defer srv.Close()

	client := apiconnect.NewWorkflowsServiceClient(srv.Client(), srv.URL)
	request := connect.NewRequest(&basexecution.ExecuteAdhocRequest{FlowDefinition: &basworkflows.WorkflowDefinitionV2{}})
	request.Header().Set(apihttp.TestModeHeader, apihttp.TestModeValue)
	_, err := client.ExecuteAdhocWorkflow(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, exec.lastAdhocCtx)
	require.True(t, coredb.IsTestMode(exec.lastAdhocCtx))
}

// ---------------------------------------------------------------------------
// Validate{,Resolved}Workflow
// ---------------------------------------------------------------------------

func TestValidateWorkflow_MissingDef(t *testing.T) {
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()
	_, err := client.ValidateWorkflow(context.Background(), connect.NewRequest(&basapi.ValidateWorkflowRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestValidateWorkflow_OK(t *testing.T) {
	v := &fakeValidator{result: &basapi.WorkflowValidationResult{Valid: true}}
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: v})
	defer stop()

	resp, err := client.ValidateWorkflow(context.Background(), connect.NewRequest(&basapi.ValidateWorkflowRequest{
		Workflow: &basworkflows.WorkflowDefinitionV2{},
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetResult().GetValid())
	require.Equal(t, 1, v.called)
}

func TestValidateResolvedWorkflow_OK(t *testing.T) {
	v := &fakeValidator{result: &basapi.WorkflowValidationResult{Valid: true}}
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: v})
	defer stop()

	resp, err := client.ValidateResolvedWorkflow(context.Background(), connect.NewRequest(&basapi.ValidateWorkflowRequest{
		Workflow: &basworkflows.WorkflowDefinitionV2{},
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetResult().GetValid())
	require.Equal(t, 1, v.called)
}

// ---------------------------------------------------------------------------
// ModifyWorkflow
// ---------------------------------------------------------------------------

func TestModifyWorkflow_Validation(t *testing.T) {
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	t.Run("invalid id", func(t *testing.T) {
		_, err := client.ModifyWorkflow(context.Background(), connect.NewRequest(&basapi.ModifyWorkflowRequest{ModificationPrompt: "x", CurrentFlow: &basworkflows.WorkflowDefinitionV2{}}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("missing prompt", func(t *testing.T) {
		_, err := client.ModifyWorkflow(context.Background(), connect.NewRequest(&basapi.ModifyWorkflowRequest{WorkflowId: uuid.NewString(), CurrentFlow: &basworkflows.WorkflowDefinitionV2{}}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("missing flow", func(t *testing.T) {
		_, err := client.ModifyWorkflow(context.Background(), connect.NewRequest(&basapi.ModifyWorkflowRequest{WorkflowId: uuid.NewString(), ModificationPrompt: "do thing"}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}

func TestModifyWorkflow_AIError(t *testing.T) {
	cat := &fakeCatalog{modifyErr: &workflowservice.AIWorkflowError{Reason: "model returned junk"}}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	_, err := client.ModifyWorkflow(context.Background(), connect.NewRequest(&basapi.ModifyWorkflowRequest{
		WorkflowId:         uuid.NewString(),
		ModificationPrompt: "do thing",
		CurrentFlow:        &basworkflows.WorkflowDefinitionV2{},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestModifyWorkflow_OK(t *testing.T) {
	cat := &fakeCatalog{modifyResp: &basapi.UpdateWorkflowResponse{Workflow: &basapi.WorkflowSummary{Id: "mod"}}}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	id := uuid.NewString()
	resp, err := client.ModifyWorkflow(context.Background(), connect.NewRequest(&basapi.ModifyWorkflowRequest{
		WorkflowId:         id,
		ModificationPrompt: "do thing",
		CurrentFlow:        &basworkflows.WorkflowDefinitionV2{},
	}))
	require.NoError(t, err)
	require.Equal(t, "mod", resp.Msg.GetWorkflow().GetId())
	require.Equal(t, id, cat.lastModifyID.String())
	require.Equal(t, "do thing", cat.lastModifyText)
}

// ---------------------------------------------------------------------------
// Versions
// ---------------------------------------------------------------------------

func TestListWorkflowVersions(t *testing.T) {
	cat := &fakeCatalog{versionsResp: &basapi.WorkflowVersionList{Versions: []*basapi.WorkflowVersion{{Version: 1}}}}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	resp, err := client.ListWorkflowVersions(context.Background(), connect.NewRequest(&basapi.ListWorkflowVersionsRequest{WorkflowId: uuid.NewString()}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetVersions(), 1)
}

func TestGetWorkflowVersion_VersionRequired(t *testing.T) {
	client, stop := newServerAndClient(t, Deps{Catalog: &fakeCatalog{}, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	_, err := client.GetWorkflowVersion(context.Background(), connect.NewRequest(&basapi.GetWorkflowVersionRequest{WorkflowId: uuid.NewString()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestRestoreWorkflowVersion_OK(t *testing.T) {
	cat := &fakeCatalog{restoreResp: &basapi.RestoreWorkflowVersionResponse{Workflow: &basapi.WorkflowSummary{Id: "r"}}}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	id := uuid.NewString()
	resp, err := client.RestoreWorkflowVersion(context.Background(), connect.NewRequest(&basapi.RestoreWorkflowVersionRequest{
		WorkflowId:        id,
		Version:           3,
		ChangeDescription: "rollback",
	}))
	require.NoError(t, err)
	require.Equal(t, "r", resp.Msg.GetWorkflow().GetId())
	require.Equal(t, id, cat.lastRestoreArg.id.String())
	require.Equal(t, int32(3), cat.lastRestoreArg.ver)
	require.Equal(t, "rollback", cat.lastRestoreArg.msg)
}

func TestRestoreWorkflowVersion_NotFound(t *testing.T) {
	cat := &fakeCatalog{restoreErr: workflowservice.ErrWorkflowVersionNotFound}
	client, stop := newServerAndClient(t, Deps{Catalog: cat, Executor: &fakeExecutor{}, Validator: &fakeValidator{}})
	defer stop()

	_, err := client.RestoreWorkflowVersion(context.Background(), connect.NewRequest(&basapi.RestoreWorkflowVersionRequest{
		WorkflowId: uuid.NewString(),
		Version:    1,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// ---------------------------------------------------------------------------
// utilities
// ---------------------------------------------------------------------------

func stringPtr(s string) *string { return &s }
