package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
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
	require.FileExists(t, run.Artifact.Latest)
	require.FileExists(t, run.Artifact.Timeline)
}

func TestRunScenarioRefusesObserverWithStaticSafetyErrorAndRunsSafeSibling(t *testing.T) {
	root := makeExecutionFixture(t, false)
	writeFixtureJSON(t, filepath.Join(root, "bas", "cases", "observer-unsafe.json"), map[string]any{
		"metadata": map[string]any{
			"name":           "Unsafe observer",
			"description":    "Contains a mutating browser action.",
			"execution_mode": "observer",
			"labels":         map[string]any{"reset": "none"},
		},
		"nodes": []map[string]any{{
			"id": "click-write", "action": map[string]any{"type": "ACTION_TYPE_CLICK"},
		}},
	})
	client := &fakeBASClient{result: &ExecuteResult{ExecutionID: "safe-sibling", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED}}

	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		Selector:         AssetSelector{CasesProvided: true, CasePaths: []string{"bas/cases/dashboard.json", "bas/cases/observer-unsafe.json"}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, report.Summary.Selected)
	require.Equal(t, 1, report.Summary.Refused)
	require.Equal(t, 1, report.Summary.Passed)
	require.Equal(t, 1, client.validateCalls, "only the safe sibling may reach BAS validation")
	require.Equal(t, 1, client.executeCalls, "only the safe sibling may execute")
	var unsafeRun *WorkflowRun
	for i := range report.Runs {
		if report.Runs[i].Asset.Path == "bas/cases/observer-unsafe.json" {
			unsafeRun = &report.Runs[i]
			break
		}
	}
	require.NotNil(t, unsafeRun, "selected observer asset must have its own run result")
	require.True(t, unsafeRun.Refused)
	require.Contains(t, unsafeRun.Error, "static safety error")
	requireFindingCode(t, report.Findings, validation.CodeObserverContentUnsafe)
}

func TestRunScenarioDoesNotInstallIsolationForObserverCase(t *testing.T) {
	root := makeExecutionFixture(t, false)
	client := &fakeBASClient{result: &ExecuteResult{ExecutionID: "exec-observer", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED}}
	isolation := &fakeIsolation{}

	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		Isolation:        isolation,
	})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Passed)
	require.Equal(t, 1, client.validateCalls)
	require.Equal(t, 1, client.executeCalls)
	require.False(t, isolation.acquired, "observer workflows must execute against the current target")
	require.False(t, isolation.closed)
	require.NotContains(t, client.lastRequest.Parameters.ExtraHeaders, "X-Vrooli-Test-Mode")
}

func TestRunScenarioRefusesRoutedObserverWhenIsolationInstallFailsBeforeBAS(t *testing.T) {
	root := makeExecutionFixture(t, false)
	writeRoutedObserverCase(t, root)
	client := &fakeBASClient{}
	isolation := &fakeIsolation{acquireErr: os.ErrPermission}

	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		Isolation:        isolation,
	})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Refused)
	require.Zero(t, client.validateCalls)
	require.Zero(t, client.executeCalls)
	require.True(t, isolation.acquired)
	require.Contains(t, report.Isolation.InstallError, "permission")
	require.Contains(t, report.Runs[0].Error, "proven routed test isolation")
}

func TestRunScenarioExecutesRoutedObserverWithLeaseIdentity(t *testing.T) {
	root := makeExecutionFixture(t, false)
	writeRoutedObserverCase(t, root)
	client := &fakeBASClient{result: &ExecuteResult{ExecutionID: "routed-observer", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED}}
	isolation := &fakeIsolation{evidence: IsolationEvidence{Installed: true, LeaseID: "fixture-lease"}}

	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		Isolation:        isolation,
	})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Passed)
	require.Equal(t, 1, client.validateCalls)
	require.Equal(t, 1, client.executeCalls)
	require.True(t, isolation.acquired)
	require.True(t, isolation.closed)
	require.True(t, report.Isolation.Installed)
	require.Equal(t, "fixture-lease", report.Isolation.LeaseID)
	require.Equal(t, "1", client.lastRequest.Parameters.ExtraHeaders["X-Vrooli-Test-Mode"])
	require.Equal(t, "fixture-lease", client.lastRequest.Parameters.ExtraHeaders["X-Vrooli-Test-Lease-ID"])
}

func TestRunScenarioRefusesRoutedObserverWithoutCompleteLeaseProof(t *testing.T) {
	for name, evidence := range map[string]IsolationEvidence{
		"not installed":    {LeaseID: "fixture-lease"},
		"missing identity": {Installed: true},
	} {
		t.Run(name, func(t *testing.T) {
			root := makeExecutionFixture(t, false)
			writeRoutedObserverCase(t, root)
			client := &fakeBASClient{}
			isolation := &fakeIsolation{evidence: evidence}

			report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{
				IncludeExecution: true,
				Isolation:        isolation,
			})
			require.NoError(t, err)
			require.Equal(t, 1, report.Summary.Refused)
			require.Zero(t, client.validateCalls)
			require.Zero(t, client.executeCalls)
			require.True(t, isolation.closed, "an unproven acquired lease must still be cleaned up")
			require.False(t, report.Isolation.Installed)
			require.NotEmpty(t, report.Isolation.InstallError)
			require.Contains(t, report.Runs[0].Error, "proven routed test isolation")
		})
	}
}

func writeRoutedObserverCase(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "bas", "cases", "dashboard.json")
	var doc map[string]any
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &doc))
	metadata := doc["metadata"].(map[string]any)
	labels := metadata["labels"].(map[string]any)
	labels["routed_isolation"] = true
	writeFixtureJSON(t, path, doc)
}

func TestRunScenarioBindsConfiguredTargetLocalSessionOnlyToBrowserProfile(t *testing.T) {
	t.Setenv(TargetLocalSessionTokenFileEnv, filepath.Join(t.TempDir(), "session.token"))
	tokenPath := os.Getenv(TargetLocalSessionTokenFileEnv)
	require.NoError(t, os.WriteFile(tokenPath, []byte("observer-session"), 0o600))
	root := makeExecutionFixture(t, false)
	client := &fakeBASClient{result: &ExecuteResult{ExecutionID: "exec-auth", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED}}

	_, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{IncludeExecution: true, RunID: "run-auth"})
	require.NoError(t, err)
	require.Equal(t, "LocalSession observer-session", client.lastRequest.Parameters.ExtraHeaders["Authorization"])
}

func TestRunScenarioFailsClosedForConfiguredTargetLocalSessionFile(t *testing.T) {
	root := makeExecutionFixture(t, false)
	client := &fakeBASClient{}
	service := NewService(client)

	t.Run("missing", func(t *testing.T) {
		t.Setenv(TargetLocalSessionTokenFileEnv, filepath.Join(t.TempDir(), "missing.token"))
		_, err := service.RunScenario(context.Background(), "sample", root, Options{IncludeExecution: true})
		require.Error(t, err)
		require.Contains(t, err.Error(), "read target local session token file")
		require.Zero(t, client.executeCalls)
	})
	t.Run("empty", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.token")
		t.Setenv(TargetLocalSessionTokenFileEnv, path)
		require.NoError(t, os.WriteFile(path, []byte("\n"), 0o600))
		_, err := service.RunScenario(context.Background(), "sample", root, Options{IncludeExecution: true})
		require.Error(t, err)
		require.Contains(t, err.Error(), "is empty")
		require.Zero(t, client.executeCalls)
	})
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
	require.Equal(t, "lease-1", client.lastRequest.Parameters.ExtraHeaders["X-Vrooli-Test-Lease-ID"])
	require.Equal(t, 1, report.Summary.Passed)
	require.True(t, report.Isolation.Installed)
	require.Equal(t, int64(1), report.Isolation.TestRootWrites)
}

func TestRunScenarioBindsBrowserRoutesToTheLeasedInstance(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	root := makeExecutionFixture(t, true)
	file := filepath.Join(root, "bas", "cases", "dashboard.json")
	raw, err := os.ReadFile(file)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))
	doc["nodes"] = []any{
		map[string]any{"id": "canonical", "action": map[string]any{"type": "ACTION_TYPE_NAVIGATE", "navigate": map[string]any{"scenario": "sample", "scenario_path": "/plan"}}},
		map[string]any{"id": "self", "action": map[string]any{"type": "ACTION_TYPE_NAVIGATE", "navigate": map[string]any{"scenario": "@scenario/self", "scenario_path": "/plan"}}},
		map[string]any{"id": "other", "action": map[string]any{"type": "ACTION_TYPE_NAVIGATE", "navigate": map[string]any{"scenario": "other", "scenario_path": "/status"}}},
		map[string]any{"id": "url", "action": map[string]any{"type": "ACTION_TYPE_NAVIGATE", "navigate": map[string]any{"scenario": "sample", "url": "https://example.test"}}},
	}
	writeFixtureJSON(t, file, doc)
	before, err := os.ReadFile(file)
	require.NoError(t, err)
	client := &fakeBASClient{result: &ExecuteResult{ExecutionID: "instance-execution", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED}}
	isolation := &fakeIsolation{evidence: IsolationEvidence{Installed: true, LeaseID: "instance-lease"}}
	report, err := NewService(client).RunScenario(context.Background(), "sample@shadow", root, Options{IncludeExecution: true, Isolation: isolation})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Passed)
	require.Equal(t, "sample@shadow", isolation.scenario)
	nodes := client.lastRequest.Definition["nodes"].([]any)
	for i, want := range []string{"sample@shadow", "sample@shadow", "other", "sample"} {
		navigate := nodes[i].(map[string]any)["action"].(map[string]any)["navigate"].(map[string]any)
		require.Equal(t, want, navigate["scenario"])
	}
	require.Equal(t, client.validated, client.lastRequest.Definition, "validation and execution must use the same bound definition")
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, before, after, "original registered cases must remain unchanged")
	require.True(t, isolation.closed)
}

func TestRunScenarioRefusesAnInstanceFromAnotherCatalogBeforeEffects(t *testing.T) {
	root := makeExecutionFixture(t, true)
	client := &fakeBASClient{}
	isolation := &fakeIsolation{}
	_, err := NewService(client).RunScenario(context.Background(), "other@shadow", root, Options{IncludeExecution: true, Isolation: isolation})
	require.ErrorContains(t, err, "does not match catalog")
	require.False(t, isolation.acquired)
	require.Zero(t, client.validateCalls)
	require.Zero(t, client.executeCalls)
}

func TestRunScenarioRetainsPathOnlyStaticValidation(t *testing.T) {
	root := makeExecutionFixture(t, false)
	report, err := NewService(nil).RunScenario(context.Background(), "", root, Options{})
	require.NoError(t, err)
	require.Equal(t, "sample", report.Scenario)
	require.Equal(t, 1, report.Summary.Selected)
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
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
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
	require.FileExists(t, report.Runs[0].Artifact.Latest)
	artifact, err := os.ReadFile(report.Runs[0].Artifact.Latest)
	require.NoError(t, err)
	require.Contains(t, string(artifact), "button not found")
	finding := requireFindingCode(t, report.Findings, validation.CodeExecutionFailed)
	require.Equal(t, "bas/cases/dashboard.json", finding.FilePath)
	require.Contains(t, finding.Description, "button not found")
}

func TestRunScenarioRefusesElectronTargetWithoutProvenIsolation(t *testing.T) {
	root := makeExecutionFixture(t, false)
	client := &fakeBASClient{}
	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution:  true,
		ElectronTarget:    &ElectronTarget{TargetID: "target-1", CDPEndpoint: "http://127.0.0.1:9222", RendererID: "renderer-1"},
		ValidationContext: &ValidationContext{ContextID: "ctx-1", IsolationLeaseID: "lease-1"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Refused)
	require.Zero(t, client.executeCalls)
	require.Contains(t, report.Runs[0].Error, "proven routed test isolation")
}

func TestRunScenarioBindsElectronValidationToSelectedAsset(t *testing.T) {
	root := makeExecutionFixture(t, false)
	client := &fakeBASClient{}
	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		Isolation:        &fakeIsolation{evidence: IsolationEvidence{Installed: true, LeaseID: "lease-1"}},
		ElectronTarget: &ElectronTarget{
			TargetID: "target-1", CDPEndpoint: "http://127.0.0.1:9222", RendererID: "renderer-1",
			ScenarioName: "sample", ArtifactDigest: "sha256:app", ContextID: "ctx-1", CDPTransport: "loopback-authenticated",
		},
		ValidationContext: &ValidationContext{ContextID: "ctx-1", ScenarioName: "sample", ArtifactDigest: "sha256:app", TargetID: "target-1", ProfileID: "normal", IsolationLeaseID: "lease-1"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Passed)
	require.NotNil(t, client.lastRequest.Options.ElectronTarget)
	require.Equal(t, report.Runs[0].Asset.ID, client.lastRequest.Options.ValidationContext.WorkflowID)
	require.Equal(t, "lease-1", client.lastRequest.Options.ValidationContext.IsolationLeaseID)
}

func TestRunScenarioStopsWhenIsolationHeartbeatFails(t *testing.T) {
	root := makeExecutionFixture(t, true)
	client := &fakeBASClient{waitForContext: true, started: make(chan struct{})}
	isolation := &fakeIsolation{
		evidence:   IsolationEvidence{Installed: true, LeaseID: "lease-1"},
		healthDone: make(chan struct{}),
		healthErr:  fmt.Errorf("heartbeat unavailable"),
	}
	close(isolation.healthDone)

	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		Isolation:        isolation,
	})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Refused)
	require.True(t, report.Runs[0].Refused)
	require.Contains(t, report.Runs[0].Error, "heartbeat unavailable")
	require.Contains(t, report.Isolation.HeartbeatError, "heartbeat unavailable")
	require.Contains(t, report.Findings, validation.Finding{
		Code:        validation.CodeExecutionRefused,
		Severity:    validation.SeverityError,
		Title:       "Routed test isolation unavailable",
		Description: "routed test isolation heartbeat failed: heartbeat unavailable",
		Remediation: "Wire database.RoutedDB, test-mode middleware, and devrouting file roots on the target scenario.",
	})
}

func TestRunScenarioCancelsBASWhenIsolationHeartbeatFailsDuringExecution(t *testing.T) {
	root := makeExecutionFixture(t, true)
	isolation := &fakeIsolation{evidence: IsolationEvidence{Installed: true, LeaseID: "lease-1"}, healthDone: make(chan struct{})}
	client := &fakeBASClient{
		waitForContext: true,
		cancelled:      make(chan struct{}),
		onExecute: func() {
			isolation.failHealth(fmt.Errorf("heartbeat lost during execution"))
		},
	}

	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{
		IncludeExecution: true,
		Isolation:        isolation,
	})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Failed)
	require.Contains(t, report.Runs[0].Error, "heartbeat lost during execution")
	select {
	case <-client.cancelled:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("BAS execution was not canceled after the isolation heartbeat failed")
	}
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
	validated      map[string]any
	validateCalls  int
	executeCalls   int
	timelineCalls  int
	validate       *ValidationResult
	result         *ExecuteResult
	timeline       *bastimeline.ExecutionTimeline
	lastRequest    ExecuteRequest
	waitForContext bool
	started        chan struct{}
	cancelled      chan struct{}
	onExecute      func()
}

type fakeIsolation struct {
	scenario   string
	evidence   IsolationEvidence
	acquireErr error
	acquired   bool
	closed     bool
	healthDone chan struct{}
	healthErr  error
	healthMu   sync.Mutex
	healthOnce sync.Once
}

func (f *fakeIsolation) Acquire(_ context.Context, scenario, _ string) (IsolationLease, error) {
	f.scenario = scenario
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

func (f *fakeIsolation) Done() <-chan struct{} { return f.healthDone }

func (f *fakeIsolation) Err() error {
	f.healthMu.Lock()
	defer f.healthMu.Unlock()
	return f.healthErr
}

func (f *fakeIsolation) failHealth(err error) {
	f.healthMu.Lock()
	f.healthErr = err
	f.healthMu.Unlock()
	f.healthOnce.Do(func() { close(f.healthDone) })
}

func (f *fakeBASClient) ValidateResolved(_ context.Context, definition map[string]any) (*ValidationResult, error) {
	f.validated = definition
	f.validateCalls++
	if f.validate != nil {
		return f.validate, nil
	}
	return &ValidationResult{Valid: true}, nil
}

func (f *fakeBASClient) ExecuteAdhoc(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	f.executeCalls++
	f.lastRequest = req
	if f.onExecute != nil {
		f.onExecute()
	}
	if f.waitForContext {
		if f.started != nil {
			close(f.started)
		}
		select {
		case <-ctx.Done():
			if f.cancelled != nil {
				close(f.cancelled)
			}
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return &ExecuteResult{ExecutionID: "exec", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED}, nil
		}
	}
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

func TestBrowserSelectionRefusesWholeRequestBeforeEffects(t *testing.T) {
	for _, paths := range [][]string{nil, {}, {""}, {"../bas/cases/dashboard.json"}, {"/bas/cases/dashboard.json"}, {"bas/cases/unknown.json"}, {"bas/cases/dashboard.json", "bas/cases/unknown.json"}} {
		t.Run(fmt.Sprintf("%q", paths), func(t *testing.T) {
			storage := t.TempDir()
			t.Setenv("VROOLI_STORAGE_ROOT", storage)
			root := makeExecutionFixture(t, true)
			client := &fakeBASClient{}
			isolation := &fakeIsolation{}
			_, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{IncludeExecution: true, Isolation: isolation, Selector: AssetSelector{CasesProvided: true, CasePaths: paths}})
			require.Error(t, err)
			require.Zero(t, client.validateCalls)
			require.Zero(t, client.executeCalls)
			require.False(t, isolation.acquired)
			require.False(t, isolation.closed)
			entries, err := os.ReadDir(storage)
			require.NoError(t, err)
			require.Empty(t, entries, "refusal must not publish artifacts")
		})
	}
}

func TestBrowserSelectionExecutesExactCalendarCasesWithOneOwnedLease(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	root := makeExecutionFixture(t, true)
	raw, err := os.ReadFile(filepath.Join(root, "bas", "cases", "dashboard.json"))
	require.NoError(t, err)
	names := []string{"duplicate-submit-once", "stale-revision-conflict", "uncertain-save-recovery"}
	paths := make([]string, 0, 3)
	for _, name := range names {
		p := "bas/cases/calendar/" + name + ".json"
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, p), raw, 0o644))
		paths = append(paths, p)
	}
	client := &fakeBASClient{}
	isolation := &fakeIsolation{evidence: IsolationEvidence{Installed: true, LeaseID: "provider-lease", TestPoolRequests: 3}}
	selected := append([]string{" ./" + paths[2] + " "}, paths...)
	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{IncludeExecution: true, Isolation: isolation, Selector: AssetSelector{CasesProvided: true, CasePaths: selected}})
	require.NoError(t, err)
	require.Equal(t, 3, report.Summary.Selected)
	require.Equal(t, 3, client.executeCalls)
	require.True(t, isolation.acquired)
	require.True(t, isolation.closed)
	got := make([]string, 0, 3)
	for _, run := range report.Runs {
		got = append(got, run.Asset.Path)
	}
	require.Equal(t, paths, got)
	require.Equal(t, int64(3), report.Isolation.TestPoolRequests)
}

func TestBrowserSelectionStillRefusesMutatingCasesWithoutLease(t *testing.T) {
	root := makeExecutionFixture(t, true)
	client := &fakeBASClient{}
	report, err := NewService(client).RunScenario(context.Background(), "sample", root, Options{IncludeExecution: true, Selector: AssetSelector{CasesProvided: true, CasePaths: []string{"bas/cases/dashboard.json"}}})
	require.NoError(t, err)
	require.Equal(t, 1, report.Summary.Refused)
	require.Zero(t, client.executeCalls)
}

func TestBrowserSelectionPresentEmptySliceDoesNotDefault(t *testing.T) {
	root := makeExecutionFixture(t, false)
	_, err := NewService(&fakeBASClient{}).RunScenario(context.Background(), "sample", root, Options{Selector: AssetSelector{CasePaths: []string{}}})
	require.Error(t, err)
}
