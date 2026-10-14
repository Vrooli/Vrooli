package validation

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"workflow-health/internal/execution"
	internalvalidation "workflow-health/internal/validation"
	workflowrun "workflow-health/internal/validationrun"

	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/metrics"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	core "github.com/vrooli/api-core/validationrun"
	"github.com/vrooli/maturity-go/assessment"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestValidateScenarioReturnsSharedProviderResponse(t *testing.T) {
	handler := testHandler(t)
	target := filepath.Join(t.TempDir(), "empty-workflows")
	require.NoError(t, mkdir(target))

	resp, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "empty-workflows",
		Path:     target,
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "empty-workflows", resp.Msg.GetScenario())
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, resp.Msg.GetStatus())
	require.NotNil(t, resp.Msg.GetAssessment())
	require.Equal(t, "workflow-health", resp.Msg.GetAssessment().GetProvider())
	require.Equal(t, "workflow", resp.Msg.GetAssessment().GetPhase())
	require.NotNil(t, resp.Msg.GetNativeDetail())
}

func TestValidateScenarioExecutionFailureReturnsSharedFinding(t *testing.T) {
	target := filepath.Join(t.TempDir(), "failed-workflow")
	writeWorkflowJSON(t, filepath.Join(target, "bas", "cases", "smoke.json"), `{
  "metadata": {
    "name": "Smoke",
    "description": "Runs the smoke workflow.",
    "execution_mode": "observer",
    "labels": { "reset": "none" }
  },
  "nodes": []
}`)
	writeWorkflowJSON(t, filepath.Join(target, "requirements", "index.json"), `{
  "imports": ["module.json"]
}`)
	writeWorkflowJSON(t, filepath.Join(target, "requirements", "module.json"), `{
  "requirements": [
    {
      "id": "REQ-SMOKE",
      "validation": [{ "ref": "bas/cases/smoke.json" }]
    }
  ]
}`)
	handler := testHandlerWithBAS(t, &validationFakeBASClient{
		result: &execution.ExecuteResult{
			ExecutionID: "exec-failed",
			Status:      basbase.ExecutionStatus_EXECUTION_STATUS_FAILED,
			Error:       "selector timed out",
		},
	})

	report, err := handler.run(context.Background(), "failed-workflow", target, execution.Options{IncludeExecution: true})
	require.NoError(t, err)
	// The durable path measures its own run and passes the result through, so
	// the response carries real metrics rather than the empty struct that made
	// metrics_present read 1 over no data.
	collector := metrics.Start()
	terminal, err := handler.responseForReport(report, collector.Stop())
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, terminal.GetStatus())
	require.NotNil(t, terminal.GetMetrics().GetResources(),
		"a durable validation must report measured resources, not an empty metrics struct")
	require.NotNil(t, terminal.GetAssessment())
	finding := requireAssessmentFindingCode(t, terminal.GetAssessment().GetFindings(), internalvalidation.CodeExecutionFailed)
	require.Equal(t, internalvalidation.CodeExecutionFailed, finding.GetCode())
	require.Equal(t, "bas/cases/smoke.json", finding.GetLocation())
	require.Contains(t, finding.GetMessage(), "selector timed out")
}

func TestDurableValidationRunStartsPromptlyAndWaitsForTerminalResult(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(workflowrun.Schema)))
	handler := testHandlerWithBAS(t, &validationFakeBASClient{})
	handler.deps.Ledger = workflowrun.Repository{DB: database}
	target := filepath.Join(t.TempDir(), "empty-workflows")
	require.NoError(t, mkdir(target))

	started, err := handler.StartValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{Scenario: "empty-workflows", Path: target, IdempotencyKey: "request-1", ParentRunId: "parent-1"}))
	require.NoError(t, err)
	require.NotEmpty(t, started.Msg.GetRun().GetRunId())
	require.NotNil(t, started.Msg.GetRun().GetPreliminaryStaticResult())

	replay, err := handler.StartValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{Scenario: "empty-workflows", Path: target, IdempotencyKey: "request-1", ParentRunId: "parent-1"}))
	require.NoError(t, err)
	require.Equal(t, started.Msg.GetRun().GetRunId(), replay.Msg.GetRun().GetRunId())

	completed, err := handler.WaitValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{RunId: started.Msg.GetRun().GetRunId(), Timeout: durationpb.New(time.Second)}))
	require.NoError(t, err)
	require.True(t, completed.Msg.GetRun().GetState() == scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_SUCCEEDED || completed.Msg.GetRun().GetState() == scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_FAILED)
	require.NotNil(t, completed.Msg.GetRun().GetTerminalResult())
}

func TestPreviewFixReturnsWorkflowCandidates(t *testing.T) {
	handler := testHandler(t)
	target := filepath.Join(t.TempDir(), "fixable-workflows")
	writeWorkflowJSON(t, filepath.Join(target, "bas", "cases", "smoke.json"), `{
  "metadata": {
    "execution_mode": "weird",
    "labels": { "reset": "database" }
  },
  "nodes": []
}`)

	resp, err := handler.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
		Scenario: "fixable-workflows",
		Path:     target,
		RuleIds:  []string{internalvalidation.CodeExecutionModeInvalid},
	}))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "fixable-workflows", resp.Msg.GetScenario())
	require.False(t, resp.Msg.GetApplied())
	require.Len(t, resp.Msg.GetCandidates(), 1)
	require.Equal(t, internalvalidation.CodeExecutionModeInvalid, resp.Msg.GetCandidates()[0].GetRuleId())
	require.Contains(t, resp.Msg.GetCandidates()[0].GetAfter(), `"execution_mode": "observer"`)
}

func testHandler(t *testing.T) *connectHandler {
	t.Helper()
	return testHandlerWithBAS(t, nil)
}

func testHandlerWithBAS(t *testing.T, basClient execution.BASClient) *connectHandler {
	t.Helper()
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRootFromTest(t), "scenarios", "workflow-health"))
	require.NoError(t, err)
	return NewConnectHandler(Deps{
		Engine:       internalvalidation.NewEngine(),
		MaturitySpec: spec,
		BASClient:    basClient,
	})
}

func repoRootFromTest(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	require.NoError(t, err)
	for {
		if filepath.Base(dir) == "api" && filepath.Base(filepath.Dir(dir)) == "workflow-health" {
			return filepath.Clean(filepath.Join(dir, "..", "..", ".."))
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "repo root not found")
		dir = parent
	}
}

func mkdir(path string) error {
	return writeFile(filepath.Join(path, ".keep"), "")
}

func writeWorkflowJSON(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, writeFile(path, body+"\n"))
}

func writeFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

type validationFakeBASClient struct {
	result *execution.ExecuteResult
}

func (f *validationFakeBASClient) ValidateResolved(context.Context, map[string]any) (*execution.ValidationResult, error) {
	return &execution.ValidationResult{Valid: true}, nil
}

func (f *validationFakeBASClient) ExecuteAdhoc(context.Context, execution.ExecuteRequest) (*execution.ExecuteResult, error) {
	if f.result != nil {
		return f.result, nil
	}
	return &execution.ExecuteResult{ExecutionID: "exec", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED}, nil
}

func (f *validationFakeBASClient) Timeline(context.Context, string, map[string]string) (*bastimeline.ExecutionTimeline, error) {
	return nil, nil
}

func requireAssessmentFindingCode(t *testing.T, findings []*commonv1.AssessmentFinding, code string) *commonv1.AssessmentFinding {
	t.Helper()
	for _, finding := range findings {
		if finding.GetCode() == code {
			return finding
		}
	}
	require.Failf(t, "finding not found", "code %s in %+v", code, findings)
	return nil
}

func TestOrdinaryDurableStartRejectsExplicitSelectorBeforeReplay(t *testing.T) {
	h := testHandler(t)
	for _, paths := range [][]string{nil, {"bas/cases/smoke.json"}} {
		_, err := h.StartValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: paths}}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.Contains(t, err.Error(), "StartSelectedValidationRun")
	}
}

func selectedFixture(t *testing.T) (*connectHandler, string) {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(workflowrun.Schema)))
	h := testHandlerWithBAS(t, &validationFakeBASClient{})
	h.deps.Ledger = workflowrun.Repository{DB: database}
	target := filepath.Join(t.TempDir(), "selected")
	for _, name := range []string{"a", "b"} {
		writeWorkflowJSON(t, filepath.Join(target, "bas", "cases", name+".json"), `{"metadata":{"name":"`+name+`","execution_mode":"observer","labels":{"reset":"none"}},"nodes":[]}`)
	}
	return h, target
}

func TestSelectedDurableRouteIdentityTerminalReadbackAndCatalogDrift(t *testing.T) {
	h, target := selectedFixture(t)
	path, handler := validationconnect.NewSelectedValidationRunServiceHandler(h)
	router := http.NewServeMux()
	router.Handle(path, handler)
	ordinaryPath, ordinaryHandler := validationconnect.NewDurableValidationRunServiceHandler(h)
	router.Handle(ordinaryPath, ordinaryHandler)
	server := httptest.NewServer(router)
	defer server.Close()
	selected := validationconnect.NewSelectedValidationRunServiceClient(server.Client(), server.URL)
	durable := validationconnect.NewDurableValidationRunServiceClient(server.Client(), server.URL)
	request := &scenariovalidationv1.StartValidationRunRequest{Scenario: "selected", Path: target, IdempotencyKey: "identity", BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: []string{" ./bas/cases/b.json ", "bas/cases/a.json", "bas/cases/a.json"}}}
	started, err := selected.StartSelectedValidationRun(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, []string{"bas/cases/a.json", "bas/cases/b.json"}, started.Msg.Run.BrowserCaseSelector.CasePaths)
	request.BrowserCaseSelector.CasePaths = []string{"bas/cases/b.json", "bas/cases/a.json"}
	replay, err := selected.StartSelectedValidationRun(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, started.Msg.Run.RunId, replay.Msg.Run.RunId)
	terminal, err := durable.WaitValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{RunId: started.Msg.Run.RunId, Timeout: durationpb.New(5 * time.Second)}))
	require.NoError(t, err)
	require.Equal(t, started.Msg.Run.BrowserCaseSelector, terminal.Msg.Run.BrowserCaseSelector)
	reopened := testHandler(t)
	reopened.deps.Ledger = h.deps.Ledger
	readback, err := reopened.GetValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.GetValidationRunRequest{RunId: started.Msg.Run.RunId}))
	require.NoError(t, err)
	require.Equal(t, terminal.Msg.Run.BrowserCaseSelector, readback.Msg.Run.BrowserCaseSelector)
	require.NoError(t, os.RemoveAll(filepath.Join(target, "bas")))
	replay, err = selected.StartSelectedValidationRun(context.Background(), connect.NewRequest(request))
	require.NoError(t, err)
	require.Equal(t, started.Msg.Run.RunId, replay.Msg.Run.RunId)
	request.BrowserCaseSelector.CasePaths = []string{"bas/cases/a.json"}
	_, err = selected.StartSelectedValidationRun(context.Background(), connect.NewRequest(request))
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
	request.BrowserCaseSelector = nil
	_, err = durable.StartValidationRun(context.Background(), connect.NewRequest(request))
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
}

func TestSelectedDurableRefusesInvalidAndConflictingBindingsBeforeEffects(t *testing.T) {
	h, target := selectedFixture(t)
	for name, request := range map[string]*scenariovalidationv1.StartValidationRunRequest{
		"missing": {}, "empty": {BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{}},
		"mixed":      {BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: []string{"bas/cases/a.json", "bas/cases/unknown.json"}}},
		"unsafe":     {BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: []string{"bas/cases/../a.json"}}},
		"desktop":    {BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: []string{"bas/cases/a.json"}}, DesktopBinding: &scenariovalidationv1.DesktopValidationBinding{WorkflowPath: "bas/cases/b.json"}},
		"capability": {BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: []string{"bas/cases/a.json"}}, CapabilitySubset: []string{"unknown"}},
	} {
		t.Run(name, func(t *testing.T) {
			request.Scenario = "selected"
			request.Path = target
			request.IdempotencyKey = name
			_, err := h.StartSelectedValidationRun(context.Background(), connect.NewRequest(request))
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			_, err = h.deps.Ledger.FindByIdempotency(context.Background(), name)
			require.True(t, core.IsCode(err, core.ErrorNotFound))
		})
	}
}

func TestSelectedDurableConcurrentStartsReplayOneRun(t *testing.T) {
	h, target := selectedFixture(t)
	second := testHandlerWithBAS(t, &validationFakeBASClient{})
	second.deps.Ledger = h.deps.Ledger
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		worker := h
		if i%2 == 1 {
			worker = second
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := worker.StartSelectedValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{Scenario: "selected", Path: target, IdempotencyKey: "concurrent", BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: []string{"bas/cases/a.json"}}}))
			if err != nil {
				errs <- err
				return
			}
			ids <- resp.Msg.Run.RunId
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		require.Equal(t, first, id)
	}
	require.NotEmpty(t, first)
	h.mu.Lock()
	_, active := h.cancels[first]
	h.mu.Unlock()
	owner := second
	if active {
		owner = h
	}
	_, err := owner.WaitValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{RunId: first, Timeout: durationpb.New(5 * time.Second)}))
	require.NoError(t, err)
}

func TestValidationModuleMountsSelectedServiceAndRecoveryPreservesSelection(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(workflowrun.Schema)))
	routed := apidb.NewFromPrimary(database)
	repository := workflowrun.Repository{DB: routed}
	record := workflowrun.Record{Run: core.Run{ID: "interrupted", Target: core.Target{Scenario: "sample"}, IdempotencyKey: "restart", State: core.StateRunning, CreatedAt: time.Now().UTC(), Version: 1}, BrowserCasePaths: []string{"bas/cases/a.json"}}
	require.NoError(t, repository.Create(context.Background(), record))
	require.NoError(t, RecoverInterrupted(context.Background(), routed))
	require.NoError(t, RecoverInterrupted(context.Background(), routed))
	router := mux.NewRouter()
	Module(nil, repoRootFromTest(t), routed).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	selected := validationconnect.NewSelectedValidationRunServiceClient(server.Client(), server.URL)
	resp, err := selected.StartSelectedValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{Scenario: "sample", IdempotencyKey: "restart", BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: []string{"bas/cases/a.json"}}}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_RECOVERY_FAILED, resp.Msg.Run.State)
	require.Equal(t, []string{"bas/cases/a.json"}, resp.Msg.Run.BrowserCaseSelector.CasePaths)
	durable := validationconnect.NewDurableValidationRunServiceClient(server.Client(), server.URL)
	waited, err := durable.WaitValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{RunId: record.Run.ID}))
	require.NoError(t, err)
	require.Equal(t, resp.Msg.Run.BrowserCaseSelector, waited.Msg.Run.BrowserCaseSelector)
}

func TestSelectedDurableExecutesOnlySavedCases(t *testing.T) {
	h, target := selectedFixture(t)
	recorder := &selectedRecordingBAS{}
	h.deps.BASClient = recorder
	resp, err := h.StartSelectedValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{Scenario: "selected", Path: target, IdempotencyKey: "context-only", BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: []string{"bas/cases/a.json"}}}))
	require.NoError(t, err)
	finished, err := h.WaitValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{RunId: resp.Msg.Run.RunId, Timeout: durationpb.New(5 * time.Second)}))
	require.NoError(t, err)
	require.Equal(t, []string{"bas/cases/a.json"}, finished.Msg.Run.BrowserCaseSelector.CasePaths)
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	require.Equal(t, []string{"a"}, recorder.names)
}

func TestLaunchRefusesSavedSelectionWithWorkflowBindingBeforeExecution(t *testing.T) {
	h, target := selectedFixture(t)
	record := workflowrun.Record{Run: core.Run{ID: "mixed-saved", Target: core.Target{Scenario: "selected", Path: target}, IdempotencyKey: "mixed-saved-key", State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}, BrowserCasePaths: []string{"bas/cases/a.json"}, ExecutionBinding: &scenariovalidationv1.DesktopValidationBinding{WorkflowPath: "bas/cases/b.json"}}
	require.NoError(t, h.deps.Ledger.Create(context.Background(), record))
	h.launch(record.Run.ID)
	finished, err := h.WaitValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.WaitValidationRunRequest{RunId: record.Run.ID, Timeout: durationpb.New(5 * time.Second)}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationRunState_VALIDATION_RUN_STATE_FAILED, finished.Msg.Run.State)
	require.Contains(t, finished.Msg.Run.Error.Message, "conflicts with workflow-path binding")
	require.Nil(t, finished.Msg.Run.TerminalResult)
	require.Empty(t, finished.Msg.Run.ArtifactReferences)
}

type selectedRecordingBAS struct {
	validationFakeBASClient
	mu    sync.Mutex
	names []string
}

func (f *selectedRecordingBAS) ExecuteAdhoc(ctx context.Context, req execution.ExecuteRequest) (*execution.ExecuteResult, error) {
	f.mu.Lock()
	f.names = append(f.names, req.Name)
	f.mu.Unlock()
	return f.validationFakeBASClient.ExecuteAdhoc(ctx, req)
}

func TestCompleteSelectedTargetContextBindingPreservesExecutionSelector(t *testing.T) {
	binding := &scenariovalidationv1.DesktopValidationBinding{TargetId: "target", CdpEndpoint: "http://127.0.0.1:9222", RendererId: "renderer", RendererUrl: "app://planner", ScenarioName: "personal-planner", ArtifactDigest: "sha256:artifact", ContextId: "context", CdpTransport: "http", ProfileId: "profile"}
	record := workflowrun.Record{Run: core.Run{ID: "provider-run"}, BrowserCasePaths: []string{"bas/cases/calendar/duplicate-submit-once.json"}, ExecutionBinding: binding}
	opts, err := executionOptions(record)
	require.NoError(t, err)
	require.True(t, opts.Selector.CasesProvided)
	require.Equal(t, []string{"bas/cases/calendar/duplicate-submit-once.json"}, opts.Selector.CasePaths)
	require.Empty(t, opts.Selector.FlowPaths)
	require.False(t, opts.AllowFlowExecution)
	require.Equal(t, "target", opts.ElectronTarget.TargetID)
	require.Equal(t, "renderer", opts.ElectronTarget.RendererID)
	require.Equal(t, "context", opts.ValidationContext.ContextID)
	require.Equal(t, "provider-run", opts.ValidationContext.IsolationLeaseID)
	h, _ := selectedFixture(t)
	_, err = h.StartSelectedValidationRun(context.Background(), connect.NewRequest(&scenariovalidationv1.StartValidationRunRequest{BrowserCaseSelector: &scenariovalidationv1.BrowserCaseSelector{CasePaths: record.BrowserCasePaths}, DesktopBinding: &scenariovalidationv1.DesktopValidationBinding{TargetId: "target", ContextId: "context"}}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, err.Error(), "complete target")
}
