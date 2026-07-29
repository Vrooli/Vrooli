package execution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"workflow-health/internal/validation"

	"github.com/stretchr/testify/require"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

func TestRunScenarioStaticOnlyDoesNotCallBAS(t *testing.T) {
	root := makeExecutionFixture(t, false)
	client := &fakeBASClient{}
	service := NewService(client)

	report, err := service.RunScenario(context.Background(), "sample", root, Options{})
	require.NoError(t, err)

	require.Equal(t, 1, report.Summary.Selected)
	require.Equal(t, 1, report.Summary.Skipped)
	require.Zero(t, client.executeCalls)
	require.Empty(t, report.Runs)
}

func TestRunScenarioExecutesObserverCaseAndWritesArtifacts(t *testing.T) {
	root := makeExecutionFixture(t, false)
	client := &fakeBASClient{
		result: &ExecuteResult{ExecutionID: "exec-1", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED},
		timeline: &bastimeline.ExecutionTimeline{
			ExecutionId: "exec-1",
			Status:      basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
			Logs:        []*bastimeline.TimelineLog{{Level: basbase.LogLevel_LOG_LEVEL_INFO, Message: "done"}},
		},
	}
	service := NewService(client)
	service.Now = fixedClock(time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC))

	report, err := service.RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		RunID:            "run-1",
	})
	require.NoError(t, err)

	require.Equal(t, 1, report.Summary.Executed)
	require.Equal(t, 1, report.Summary.Passed)
	require.Equal(t, 1, client.validateCalls)
	require.Equal(t, 1, client.executeCalls)
	require.Equal(t, filepath.Join(root, "bas"), client.lastRequest.Parameters.ProjectRoot)
	require.Len(t, report.Runs, 1)
	run := report.Runs[0]
	require.True(t, run.Success)
	require.Equal(t, "exec-1", run.ExecutionID)
	require.NotEmpty(t, run.Artifact.Latest)
	require.FileExists(t, filepath.Join(root, run.Artifact.Latest))
	require.FileExists(t, filepath.Join(root, run.Artifact.Timeline))
}

func TestRunScenarioInstallsIsolationForEveryCaseAndClosesLease(t *testing.T) {
	root := makeExecutionFixture(t, true)
	client := &fakeBASClient{result: &ExecuteResult{ExecutionID: "exec-1", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED}}
	isolation := &fakeIsolation{evidence: IsolationEvidence{Installed: true, LeaseID: "lease-1", TestPoolRequests: 1, TestRootWrites: 1}}
	service := NewService(client)

	report, err := service.RunScenario(context.Background(), "sample", root, Options{IncludeExecution: true, RunID: "run-1", Isolation: isolation})
	require.NoError(t, err)
	require.True(t, isolation.acquired)
	require.True(t, isolation.closed)
	require.Equal(t, "1", client.lastRequest.Parameters.ExtraHeaders["X-Vrooli-Test-Mode"])
	require.Equal(t, 1, report.Summary.Passed)
	require.True(t, report.Isolation.Installed)
	require.Equal(t, int64(1), report.Isolation.TestRootWrites)
}

func TestRunScenarioRefusesMutatingCaseWhenIsolationInstallFails(t *testing.T) {
	root := makeExecutionFixture(t, true)
	client := &fakeBASClient{}
	service := NewService(client)

	report, err := service.RunScenario(context.Background(), "sample", root, Options{IncludeExecution: true, Isolation: &fakeIsolation{acquireErr: os.ErrPermission}})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Refused)
	require.Zero(t, client.executeCalls)
	require.Contains(t, report.Isolation.InstallError, "permission")
	require.NotEmpty(t, requireFindingCode(t, report.Findings, validation.CodeExecutionRefused))
}

func TestRunScenarioAbsolutizesRelativeTargetProjectRoot(t *testing.T) {
	root := makeExecutionFixture(t, false)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	relTarget, err := filepath.Rel(cwd, root)
	require.NoError(t, err)
	require.False(t, filepath.IsAbs(relTarget), "relTarget should be relative for this test to be meaningful")

	client := &fakeBASClient{
		result: &ExecuteResult{ExecutionID: "exec-1", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED},
	}
	service := NewService(client)

	_, err = service.RunScenario(context.Background(), "sample", relTarget, Options{
		IncludeExecution: true,
	})
	require.NoError(t, err)

	require.Equal(t, 1, client.executeCalls)
	got := client.lastRequest.Parameters.ProjectRoot
	require.True(t, filepath.IsAbs(got), "ProjectRoot must be absolute, got %q", got)
	require.Equal(t, filepath.Join(root, "bas"), got)
}

func TestRunScenarioRefusesMutatingCaseBeforeBASWithoutProviderOwnedIsolation(t *testing.T) {
	root := makeExecutionFixture(t, true)
	client := &fakeBASClient{}
	service := NewService(client)

	report, err := service.RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		ExtraHeaders:     map[string]string{"X-Vrooli-Test-Mode": "1"},
	})
	require.NoError(t, err)

	require.Equal(t, 1, report.Summary.Refused)
	require.Zero(t, client.validateCalls)
	require.Zero(t, client.executeCalls)
	require.Len(t, report.Runs, 1)
	require.True(t, report.Runs[0].Refused)
	require.Contains(t, report.Runs[0].Error, "routed test isolation")
	finding := requireFindingCode(t, report.Findings, validation.CodeExecutionRefused)
	require.Equal(t, "bas/cases/dashboard.json", finding.FilePath)
	require.Contains(t, finding.Description, "routed test isolation")
}

func TestRunScenarioRefusesMutatingCaseMissingSafetyMetadataBeforeBAS(t *testing.T) {
	root := makeExecutionFixture(t, true)
	writeFixtureJSON(t, filepath.Join(root, "bas", "cases", "dashboard.json"), map[string]any{
		"metadata": map[string]any{
			"name":           "Open dashboard",
			"description":    "Opens the dashboard.",
			"execution_mode": "mutating",
			"labels":         map[string]any{"reset": "full"},
		},
		"nodes": []map[string]any{},
	})
	client := &fakeBASClient{}
	service := NewService(client)

	report, err := service.RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		Isolation:        &fakeIsolation{},
	})
	require.NoError(t, err)

	require.Equal(t, 1, report.Summary.Refused)
	require.Zero(t, client.validateCalls)
	require.Zero(t, client.executeCalls)
	require.Contains(t, report.Runs[0].Error, "requires_confirmation")
}

func TestRunScenarioRefusesDestructiveCaseMissingSafetyMetadataBeforeBAS(t *testing.T) {
	root := makeExecutionFixture(t, false)
	writeFixtureJSON(t, filepath.Join(root, "bas", "cases", "dashboard.json"), map[string]any{
		"metadata": map[string]any{
			"name":           "Delete dashboard",
			"description":    "Deletes dashboard data.",
			"execution_mode": "destructive",
			"labels":         map[string]any{"reset": "none"},
		},
		"nodes": []map[string]any{},
	})
	client := &fakeBASClient{}
	service := NewService(client)

	report, err := service.RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		Isolation:        &fakeIsolation{},
	})
	require.NoError(t, err)

	require.Equal(t, 1, report.Summary.Refused)
	require.Zero(t, client.validateCalls)
	require.Zero(t, client.executeCalls)
	require.Contains(t, report.Runs[0].Error, "requires_confirmation")
}

func TestRunScenarioFailedExecutionAddsFinding(t *testing.T) {
	root := makeExecutionFixture(t, false)
	client := &fakeBASClient{
		result: &ExecuteResult{
			ExecutionID: "exec-failed",
			Status:      basbase.ExecutionStatus_EXECUTION_STATUS_FAILED,
			Error:       "button not found",
		},
	}
	service := NewService(client)

	report, err := service.RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
	})
	require.NoError(t, err)

	require.Equal(t, 1, report.Summary.Failed)
	require.Len(t, report.Runs, 1)
	require.NotEmpty(t, report.Runs[0].Artifact.Latest)
	require.FileExists(t, filepath.Join(root, report.Runs[0].Artifact.Latest))
	artifact, err := os.ReadFile(filepath.Join(root, report.Runs[0].Artifact.Latest))
	require.NoError(t, err)
	require.Contains(t, string(artifact), "button not found")
	finding := requireFindingCode(t, report.Findings, validation.CodeExecutionFailed)
	require.Equal(t, "bas/cases/dashboard.json", finding.FilePath)
	require.Contains(t, finding.Description, "button not found")
}

func makeExecutionFixture(t *testing.T, mutating bool) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "sample")
	metadata := map[string]any{
		"name":           "Open dashboard",
		"description":    "Opens the dashboard.",
		"execution_mode": "observer",
		"labels":         map[string]any{"reset": "none"},
	}
	if mutating {
		metadata["execution_mode"] = "mutating"
		metadata["labels"] = map[string]any{
			"reset":                 "full",
			"requires_confirmation": "true",
			"routed_isolation":      "true",
		}
	}
	writeFixtureJSON(t, filepath.Join(root, "bas", "cases", "dashboard.json"), map[string]any{
		"metadata": metadata,
		"nodes":    []map[string]any{},
	})
	writeFixtureJSON(t, filepath.Join(root, "requirements", "index.json"), map[string]any{
		"imports": []string{"module.json"},
	})
	writeFixtureJSON(t, filepath.Join(root, "requirements", "module.json"), map[string]any{
		"requirements": []map[string]any{
			{
				"id":         "REQ-DASHBOARD",
				"validation": []map[string]any{{"ref": "bas/cases/dashboard.json"}},
			},
		},
	})
	return root
}

func writeFixtureJSON(t *testing.T, path string, value any) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	data, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

type fakeBASClient struct {
	validateCalls int
	executeCalls  int
	timelineCalls int
	validate      *ValidationResult
	result        *ExecuteResult
	timeline      *bastimeline.ExecutionTimeline
	lastRequest   ExecuteRequest
}

type fakeIsolation struct {
	evidence   IsolationEvidence
	acquireErr error
	acquired   bool
	closed     bool
}

func (f *fakeIsolation) Acquire(context.Context, string, string) (IsolationLease, error) {
	f.acquired = true
	if f.acquireErr != nil {
		return nil, f.acquireErr
	}
	return f, nil
}

func (f *fakeIsolation) Evidence() IsolationEvidence { return f.evidence }

func (f *fakeIsolation) Close(context.Context) IsolationEvidence {
	f.closed = true
	return f.evidence
}

func (f *fakeBASClient) ValidateResolved(context.Context, map[string]any) (*ValidationResult, error) {
	f.validateCalls++
	if f.validate != nil {
		return f.validate, nil
	}
	return &ValidationResult{Valid: true}, nil
}

func (f *fakeBASClient) ExecuteAdhoc(_ context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	f.executeCalls++
	f.lastRequest = req
	if f.result != nil {
		return f.result, nil
	}
	return &ExecuteResult{ExecutionID: "exec", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED}, nil
}

func (f *fakeBASClient) Timeline(context.Context, string, map[string]string) (*bastimeline.ExecutionTimeline, error) {
	f.timelineCalls++
	return f.timeline, nil
}

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func requireFindingCode(t *testing.T, findings []validation.Finding, code string) validation.Finding {
	t.Helper()
	for _, finding := range findings {
		if finding.Code == code {
			return finding
		}
	}
	require.Failf(t, "finding not found", "code %s in %+v", code, findings)
	return validation.Finding{}
}
