package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	"workflow-health/internal/execution"
	"workflow-health/internal/testutil/db"
	internalvalidation "workflow-health/internal/validation"
	workflowrun "workflow-health/internal/validationrun"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/maturity-go/assessment"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
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
	terminal, err := handler.responseForReport(report)
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, terminal.GetStatus())
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

func (f *validationFakeBASClient) Timeline(context.Context, string) (*bastimeline.ExecutionTimeline, error) {
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
