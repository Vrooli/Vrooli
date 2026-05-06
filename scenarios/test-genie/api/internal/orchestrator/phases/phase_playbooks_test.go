package phases

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooks"
	"test-genie/internal/playbooks/isolation"
	"test-genie/internal/shared"
)

// playbooksTestHarness provides a consistent test setup for playbooks phase tests.
type playbooksTestHarness struct {
	env         workspace.Environment
	scenarioDir string
	testDir     string
	appRoot     string
}

type fakeIsolation struct {
	result *isolation.Result
	err    error
	called bool
}

type fakeTargetRuntime struct {
	restartCalls int
	restoreCalls int
	restartEnv   map[string]string
	restartErr   error
	restoreErr   error
}

func (f *fakeTargetRuntime) RestartWithEnv(ctx context.Context, env map[string]string, logWriter io.Writer) error {
	f.restartCalls++
	f.restartEnv = env
	return f.restartErr
}

func (f *fakeTargetRuntime) Restore(ctx context.Context, logWriter io.Writer) error {
	f.restoreCalls++
	return f.restoreErr
}

func (f *fakeIsolation) Prepare(context.Context) (*isolation.Result, error) {
	f.called = true
	return f.result, f.err
}

func overrideIsolationManager(fake isolationProvider) func() {
	prev := isolationManagerFactory
	isolationManagerFactory = func(cfg isolation.Config) isolationProvider {
		return fake
	}
	return func() { isolationManagerFactory = prev }
}

func overrideCommandExecNoop() func() {
	return OverrideCommandExecutor(func(context.Context, string, io.Writer, string, ...string) error {
		return nil
	})
}

func newPlaybooksTestHarness(t *testing.T) *playbooksTestHarness {
	t.Helper()
	appRoot := t.TempDir()
	scenarioName := "test-scenario"
	scenarioDir := filepath.Join(appRoot, "scenarios", scenarioName)

	// Create required directory structure
	requiredDirs := []string{
		"ui",
		"bas/cases",
		".vrooli",
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(filepath.Join(scenarioDir, dir), 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	testDir := filepath.Join(scenarioDir, "test")

	return &playbooksTestHarness{
		env: workspace.Environment{
			ScenarioName:  scenarioName,
			ScenarioDir:   scenarioDir,
			TestDir:       testDir,
			AppRoot:       appRoot,
			TargetRuntime: &fakeTargetRuntime{},
		},
		scenarioDir: scenarioDir,
		testDir:     testDir,
		appRoot:     appRoot,
	}
}

func (h *playbooksTestHarness) writeRegistry(t *testing.T, content string) {
	t.Helper()
	registryDir := filepath.Join(h.scenarioDir, "bas")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatalf("failed to create bas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryDir, "registry.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write registry.json: %v", err)
	}
}

func (h *playbooksTestHarness) writeWorkflow(t *testing.T, relativePath, content string) {
	t.Helper()
	fullPath := filepath.Join(h.scenarioDir, relativePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create workflow dir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}
}

func (h *playbooksTestHarness) writeTestingJSON(t *testing.T, content string) {
	t.Helper()
	configPath := filepath.Join(h.scenarioDir, ".vrooli", "testing.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("failed to create .vrooli dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write testing.json: %v", err)
	}
}

func newStubBASServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/workflows/validate-resolved":
			_, _ = w.Write([]byte(`{"valid":true,"errors":[]}`))
		case "/api/v1/workflows/execute-adhoc":
			_, _ = w.Write([]byte(`{"execution_id":"exec-123"}`))
		case "/api/v1/executions/exec-123":
			_, _ = w.Write([]byte(`{"execution_id":"exec-123","status":"EXECUTION_STATUS_COMPLETED","progress":100}`))
		case "/api/v1/executions/exec-123/timeline":
			_, _ = w.Write([]byte(`{"execution_id":"exec-123","status":"EXECUTION_STATUS_COMPLETED","progress":100,"frames":[],"logs":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))

	port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	port = strings.TrimPrefix(port, "http://localhost:")
	return server, port
}

func (h *playbooksTestHarness) removeUI(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(h.scenarioDir, "ui")); err != nil {
		t.Fatalf("failed to remove ui dir: %v", err)
	}
}

// Tests for runPlaybooksPhase

func TestRunPlaybooksPhaseNoUIDirectory(t *testing.T) {
	restoreIso := overrideIsolationManager(&fakeIsolation{
		result: &isolation.Result{RunID: "test-run", Env: map[string]string{}, Cleanup: func(context.Context) error { return nil }},
	})
	defer restoreIso()
	restoreCmd := overrideCommandExecNoop()
	defer restoreCmd()

	// Playbooks can target any scenario, not just ones with a local ui/ directory.
	// Removing the ui/ directory should NOT skip the phase - it should proceed
	// to load the registry and execute playbooks.
	h := newPlaybooksTestHarness(t)
	h.removeUI(t)
	h.writeRegistry(t, `{"playbooks": []}`) // Empty registry = no playbooks to run

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success with empty registry (no UI dir is fine), got error: %v", report.Err)
	}
	// Should have an observation about no playbooks registered
	hasNoPlaybooksObs := false
	for _, obs := range report.Observations {
		if strings.Contains(obs.Text, "no workflows") || strings.Contains(obs.Text, "playbooks") {
			hasNoPlaybooksObs = true
			break
		}
	}
	if !hasNoPlaybooksObs {
		t.Logf("observations: %v", report.Observations)
	}
}

func TestRunPlaybooksPhaseSkipViaEnv(t *testing.T) {
	fakeIso := &fakeIsolation{
		result: &isolation.Result{RunID: "test-run", Env: map[string]string{}, Cleanup: func(context.Context) error { return nil }},
	}
	restoreIso := overrideIsolationManager(fakeIso)
	defer restoreIso()
	restoreCmd := overrideCommandExecNoop()
	defer restoreCmd()

	h := newPlaybooksTestHarness(t)

	os.Setenv("TEST_GENIE_SKIP_PLAYBOOKS", "1")
	defer os.Unsetenv("TEST_GENIE_SKIP_PLAYBOOKS")

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success when skipped via env, got error: %v", report.Err)
	}
	if fakeIso.called {
		t.Fatalf("expected isolation to be skipped when TEST_GENIE_SKIP_PLAYBOOKS is set")
	}
}

func TestRunPlaybooksPhaseObserverModeSkipsIsolationAndRestart(t *testing.T) {
	fakeIso := &fakeIsolation{
		result: &isolation.Result{RunID: "test-run", Env: map[string]string{}, Cleanup: func(context.Context) error { return nil }},
	}
	restoreIso := overrideIsolationManager(fakeIso)
	defer restoreIso()

	h := newPlaybooksTestHarness(t)
	h.writeRegistry(t, `{
		"metadata": {"execution_mode": "observer"},
		"playbooks": [
			{"file":"bas/cases/01-basic/test.json","description":"test","order":"01.01","requirements":[],"fixtures":[],"reset":"none"}
		]
	}`)
	h.writeWorkflow(t, "bas/cases/01-basic/test.json", `{
  "metadata": {"description": "basic", "version": 1},
  "nodes": [{"id":"n1","type":"navigate","data":{"destinationType":"url","url":"http://example.com"}}],
  "edges": []
}`)

	basServer, basPort := newStubBASServer(t)
	defer basServer.Close()

	restoreExec := OverrideCommandExecutor(func(_ context.Context, _ string, _ io.Writer, name string, args ...string) error {
		return nil
	})
	defer restoreExec()

	restoreCapture := OverrideCommandCapture(func(_ context.Context, _ string, _ io.Writer, name string, args ...string) (string, error) {
		joined := strings.Join(append([]string{name}, args...), " ")
		if strings.Contains(joined, "browser-automation-studio") && strings.Contains(joined, "port") {
			return basPort, nil
		}
		return "", nil
	})
	defer restoreCapture()

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success for observer mode, got error: %v", report.Err)
	}
	if fakeIso.called {
		t.Fatalf("expected observer mode to skip isolation")
	}
	runtime := h.env.TargetRuntime.(*fakeTargetRuntime)
	if runtime.restartCalls != 0 || runtime.restoreCalls != 0 {
		t.Fatalf("expected observer mode to skip scenario restarts, got restart=%d restore=%d", runtime.restartCalls, runtime.restoreCalls)
	}
}

func TestRunPlaybooksPhaseSQLiteScenarioUsesIsolationOutsideObserverMode(t *testing.T) {
	fakeIso := &fakeIsolation{
		result: &isolation.Result{
			RunID:   "sqlite-run",
			Env:     map[string]string{"PLAYBOOKS_SQLITE_PATH": filepath.Join(t.TempDir(), "isolated.db")},
			Cleanup: func(context.Context) error { return nil },
		},
	}

	var capturedCfg isolation.Config
	prevFactory := isolationManagerFactory
	isolationManagerFactory = func(cfg isolation.Config) isolationProvider {
		capturedCfg = cfg
		return fakeIso
	}
	defer func() { isolationManagerFactory = prevFactory }()

	h := newPlaybooksTestHarness(t)
	if err := os.MkdirAll(filepath.Join(h.scenarioDir, "initialization", "storage", "sqlite"), 0o755); err != nil {
		t.Fatalf("failed to create sqlite schema dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(h.scenarioDir, "initialization", "storage", "sqlite", "schema.sql"), []byte("CREATE TABLE demo(id INTEGER);"), 0o644); err != nil {
		t.Fatalf("failed to write sqlite schema: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(h.scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(h.scenarioDir, "api", "go.mod"), []byte("module fixture\n\ngo 1.24\n\nrequire modernc.org/sqlite v1.40.1\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	h.writeRegistry(t, `{
		"playbooks": [
			{"file":"bas/cases/01-basic/test.json","description":"test","order":"01.01","requirements":[],"fixtures":[],"reset":"none"}
		]
	}`)
	h.writeWorkflow(t, "bas/cases/01-basic/test.json", `{
  "metadata": {"description": "basic", "version": 1},
  "nodes": [{"id":"n1","type":"navigate","data":{"destinationType":"url","url":"http://example.com"}}],
  "edges": []
}`)

	basServer, basPort := newStubBASServer(t)
	defer basServer.Close()

	restoreExec := OverrideCommandExecutor(func(_ context.Context, _ string, _ io.Writer, name string, args ...string) error {
		return nil
	})
	defer restoreExec()

	restoreCapture := OverrideCommandCapture(func(_ context.Context, _ string, _ io.Writer, name string, args ...string) (string, error) {
		joined := strings.Join(append([]string{name}, args...), " ")
		if strings.Contains(joined, "browser-automation-studio") && strings.Contains(joined, "port") {
			return basPort, nil
		}
		return "", nil
	})
	defer restoreCapture()

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success for sqlite isolation path, got error: %v", report.Err)
	}
	if !fakeIso.called {
		t.Fatal("expected non-observer playbooks to prepare isolation")
	}
	if !capturedCfg.RequireSQLite {
		t.Fatalf("expected sqlite isolation to be requested, got %#v", capturedCfg)
	}
	if capturedCfg.RequirePostgres || capturedCfg.RequireRedis {
		t.Fatalf("did not expect postgres/redis isolation for sqlite fixture, got %#v", capturedCfg)
	}
	runtime := h.env.TargetRuntime.(*fakeTargetRuntime)
	if runtime.restartCalls != 1 || runtime.restoreCalls != 1 {
		t.Fatalf("expected scenario restart and restore around isolated run, got restart=%d restore=%d", runtime.restartCalls, runtime.restoreCalls)
	}
	if runtime.restartEnv["PLAYBOOKS_SQLITE_PATH"] == "" {
		t.Fatalf("expected isolation env to be passed to runtime restart, got %#v", runtime.restartEnv)
	}
}

func TestRunPlaybooksPhaseEmptyRegistry(t *testing.T) {
	restoreIso := overrideIsolationManager(&fakeIsolation{
		result: &isolation.Result{RunID: "test-run", Env: map[string]string{}, Cleanup: func(context.Context) error { return nil }},
	})
	defer restoreIso()
	restoreCmd := overrideCommandExecNoop()
	defer restoreCmd()

	h := newPlaybooksTestHarness(t)
	h.writeRegistry(t, `{"playbooks": []}`)

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success for empty registry, got error: %v", report.Err)
	}
}

func TestRunPlaybooksPhaseRegistryNotFound(t *testing.T) {
	restoreIso := overrideIsolationManager(&fakeIsolation{
		result: &isolation.Result{RunID: "test-run", Env: map[string]string{}, Cleanup: func(context.Context) error { return nil }},
	})
	defer restoreIso()
	restoreCmd := overrideCommandExecNoop()
	defer restoreCmd()

	h := newPlaybooksTestHarness(t)
	// Don't create registry.json

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err == nil {
		t.Fatal("expected error when registry not found")
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got: %s", report.FailureClassification)
	}
}

func TestRunPlaybooksPhaseSelfTargetRequiresObserverMode(t *testing.T) {
	fakeIso := &fakeIsolation{
		result: &isolation.Result{RunID: "test-run", Env: map[string]string{}, Cleanup: func(context.Context) error { return nil }},
	}
	restoreIso := overrideIsolationManager(fakeIso)
	defer restoreIso()

	h := newPlaybooksTestHarness(t)
	h.env.ScenarioName = "test-genie"
	h.writeRegistry(t, `{
		"playbooks": [
			{"file":"bas/cases/01-basic/test.json","description":"test","order":"01.01","requirements":[],"fixtures":[],"reset":"none"}
		]
	}`)

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err == nil {
		t.Fatal("expected self-targeted non-observer playbooks to fail")
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Fatalf("expected misconfiguration, got %s", report.FailureClassification)
	}
	if !strings.Contains(report.Err.Error(), "terminate the active test-genie API process") {
		t.Fatalf("expected self-targeting error message, got %v", report.Err)
	}
	if fakeIso.called {
		t.Fatalf("expected self-target guard to fail before isolation")
	}
}

func TestRunPlaybooksPhaseInvalidRegistryJSON(t *testing.T) {
	restoreIso := overrideIsolationManager(&fakeIsolation{
		result: &isolation.Result{RunID: "test-run", Env: map[string]string{}, Cleanup: func(context.Context) error { return nil }},
	})
	defer restoreIso()
	restoreCmd := overrideCommandExecNoop()
	defer restoreCmd()

	h := newPlaybooksTestHarness(t)
	h.writeRegistry(t, `{"invalid json`)

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err == nil {
		t.Fatal("expected error for invalid registry JSON")
	}
	if report.FailureClassification != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got: %s", report.FailureClassification)
	}
}

func TestRunPlaybooksPhaseContextCancelled(t *testing.T) {
	restoreIso := overrideIsolationManager(&fakeIsolation{
		result: &isolation.Result{RunID: "test-run", Env: map[string]string{}, Cleanup: func(context.Context) error { return nil }},
	})
	defer restoreIso()
	restoreCmd := overrideCommandExecNoop()
	defer restoreCmd()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	h := newPlaybooksTestHarness(t)

	report := runPlaybooksPhase(ctx, h.env, io.Discard)

	if report.Err == nil {
		t.Fatal("expected error when context cancelled")
	}
	if report.FailureClassification != FailureClassSystem {
		t.Errorf("expected system failure, got: %s", report.FailureClassification)
	}
}

// Ensure the isolation env is cleared before BAS (or other scenarios) start so they
// don't inherit the temporary Playbooks resources.
func TestRunPlaybooksPhaseIsolationEnvRestoredBeforeBAS(t *testing.T) {
	markerKey := "PLAYBOOKS_MARKER"

	// Fake isolation with marker env that should not leak to BAS commands.
	restoreIso := overrideIsolationManager(&fakeIsolation{
		result: &isolation.Result{
			RunID: "run-123",
			Env: map[string]string{
				markerKey: "1",
			},
			Cleanup: func(context.Context) error { return nil },
		},
	})
	defer restoreIso()

	h := newPlaybooksTestHarness(t)

	// Minimal registry + workflow so runner executes BAS path.
	h.writeRegistry(t, `{"playbooks":[{"file":"bas/cases/01-basic/test.json","description":"test","order":"01.01","requirements":[],"fixtures":[],"reset":"none"}]}`)
	h.writeWorkflow(t, "bas/cases/01-basic/test.json", `{
  "metadata": {"description": "basic", "version": 1},
  "nodes": [{"id":"n1","type":"navigate","data":{"destinationType":"url","url":"http://example.com"}}],
  "edges": []
}`)

	// Stub BAS server to satisfy health/validate/execute/status/timeline calls.
	basServer, basPort := newStubBASServer(t)
	defer basServer.Close()

	var basEnvLeakDetected bool

	// Command executor: fail if BAS commands see the marker.
	restoreExec := OverrideCommandExecutor(func(_ context.Context, _ string, _ io.Writer, name string, args ...string) error {
		joined := strings.Join(append([]string{name}, args...), " ")
		if strings.Contains(joined, "browser-automation-studio") {
			if os.Getenv(markerKey) == "1" {
				basEnvLeakDetected = true
				t.Fatalf("marker env leaked to BAS command: %s", joined)
			}
		}
		return nil
	})
	defer restoreExec()

	// Command capture: return BAS port and ensure marker absent for BAS port lookups.
	restoreCapture := OverrideCommandCapture(func(_ context.Context, _ string, _ io.Writer, name string, args ...string) (string, error) {
		joined := strings.Join(append([]string{name}, args...), " ")
		if strings.Contains(joined, "browser-automation-studio") && strings.Contains(joined, "port") {
			if os.Getenv(markerKey) == "1" {
				basEnvLeakDetected = true
				t.Fatalf("marker env leaked to BAS port command: %s", joined)
			}
			return basPort, nil
		}
		return "", nil
	})
	defer restoreCapture()

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	if report.Err != nil {
		t.Fatalf("expected success, got error: %v", report.Err)
	}
	if basEnvLeakDetected {
		t.Fatal("marker env should not be visible to BAS commands")
	}
}

func TestConvertPlaybooksObservations(t *testing.T) {
	tests := []struct {
		name     string
		input    playbooks.Observation
		wantText string
	}{
		{
			name:     "success",
			input:    playbooks.NewSuccessObservation("test passed"),
			wantText: "test passed",
		},
		{
			name:     "warning",
			input:    playbooks.NewWarningObservation("test warning"),
			wantText: "test warning",
		},
		{
			name:     "error",
			input:    playbooks.NewErrorObservation("test error"),
			wantText: "test error",
		},
		{
			name:     "info",
			input:    playbooks.NewInfoObservation("test info"),
			wantText: "test info",
		},
		{
			name:     "skip",
			input:    playbooks.NewSkipObservation("test skip"),
			wantText: "test skip",
		},
		// Note: section observations store message in Section field, not Text
		// Tested separately in TestConvertPlaybooksObservationSection
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := ConvertObservationsGeneric([]playbooks.Observation{tc.input}, ExtractStandardObservation[playbooks.Observation])
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			result := results[0]
			if result.Text != tc.wantText {
				t.Errorf("expected text %q, got %q", tc.wantText, result.Text)
			}
		})
	}
}

func TestConvertPlaybooksObservationSection(t *testing.T) {
	input := playbooks.NewSectionObservation("🏗️", "Building phase")
	results := ConvertObservationsGeneric([]playbooks.Observation{input}, ExtractStandardObservation[playbooks.Observation])
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	result := results[0]

	if result.Section != "Building phase" {
		t.Errorf("expected section %q, got %q", "Building phase", result.Section)
	}
	if result.Icon != "🏗️" {
		t.Errorf("expected icon %q, got %q", "🏗️", result.Icon)
	}
}

func TestConvertPlaybooksFailureClass(t *testing.T) {
	tests := []struct {
		input shared.FailureClass
		want  shared.FailureClass
	}{
		{shared.FailureClassMisconfiguration, shared.FailureClassMisconfiguration},
		{shared.FailureClassMissingDependency, shared.FailureClassMissingDependency},
		{shared.FailureClassSystem, shared.FailureClassSystem},
		{shared.FailureClassExecution, shared.FailureClassSystem}, // execution maps to system
	}

	for _, tc := range tests {
		t.Run(string(tc.input), func(t *testing.T) {
			result := shared.StandardizeFailureClass(tc.input)
			if result != tc.want {
				t.Errorf("expected %q, got %q", tc.want, result)
			}
		})
	}
}

func TestResolveScenarioPort(t *testing.T) {
	// This test validates the parsing logic of ResolveScenarioPort
	// The actual vrooli CLI call is mocked in integration tests

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Will cause command to fail

	_, err := ResolveScenarioPort(ctx, io.Discard, "test-scenario", "API_PORT")
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}

func TestResolveScenarioBaseURL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ResolveScenarioBaseURL(ctx, io.Discard, "test-scenario")
	if err == nil {
		t.Error("expected error when port resolution fails")
	}
}

func TestNormalizeCommandInvocationAddsNoStaleCheckForVrooli(t *testing.T) {
	name, args := normalizeCommandInvocation("vrooli", []string{"scenario", "start", "test-scenario", "--clean-stale"})
	if name != "vrooli" {
		t.Fatalf("name = %q, want %q", name, "vrooli")
	}
	if len(args) == 0 || args[0] != "--no-stale-check" {
		t.Fatalf("expected --no-stale-check prefix, got %v", args)
	}
}

func TestNormalizeCommandInvocationPreservesExistingNoStaleCheck(t *testing.T) {
	input := []string{"--no-stale-check", "scenario", "status", "test-scenario"}
	_, args := normalizeCommandInvocation("vrooli", input)
	if !reflect.DeepEqual(args, input) {
		t.Fatalf("args = %v, want %v", args, input)
	}
}

// Test helper to verify observations contain expected text
func observationsContain(observations []Observation, substr string) bool {
	for _, obs := range observations {
		if strings.Contains(obs.Text, substr) || strings.Contains(obs.Section, substr) {
			return true
		}
	}
	return false
}

func TestRunPlaybooksPhaseDeprecatedPlaybooksFallback(t *testing.T) {
	h := newPlaybooksTestHarness(t)
	// Use deprecated_playbooks field which should fall back to playbooks
	h.writeRegistry(t, `{"deprecated_playbooks": []}`)

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)

	// Should succeed with empty playbooks (from deprecated fallback)
	if report.Err != nil {
		t.Fatalf("expected success for deprecated playbooks fallback, got error: %v", report.Err)
	}
}

func TestRunPlaybooksPhaseDisabledInConfig(t *testing.T) {
	restoreIso := overrideIsolationManager(&fakeIsolation{
		result: &isolation.Result{RunID: "test-run", Env: map[string]string{}, Cleanup: func(context.Context) error { return nil }},
	})
	defer restoreIso()
	restoreCmd := overrideCommandExecNoop()
	defer restoreCmd()

	h := newPlaybooksTestHarness(t)
	h.writeRegistry(t, `{"playbooks": []}`)
	h.writeTestingJSON(t, `{"playbooks":{"enabled":false}}`)

	report := runPlaybooksPhase(context.Background(), h.env, io.Discard)
	if report.Err != nil {
		t.Fatalf("expected success when playbooks disabled via config, got error: %v", report.Err)
	}
	if len(report.Observations) == 0 || !observationsContain(report.Observations, "disabled") {
		t.Fatalf("expected skip observation when disabled via config, got %+v", report.Observations)
	}
}

// Benchmark tests

func BenchmarkRunPlaybooksPhaseEmptyRegistryNoUI(b *testing.B) {
	tempDir := b.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenarios", "bench-scenario")
	basDir := filepath.Join(scenarioDir, "bas")
	if err := os.MkdirAll(basDir, 0o755); err != nil {
		b.Fatalf("mkdir bas dir: %v", err)
	}
	// No ui/ directory, but provide empty registry
	if err := os.WriteFile(filepath.Join(basDir, "registry.json"), []byte(`{"playbooks":[]}`), 0o644); err != nil {
		b.Fatalf("write registry: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "bench-scenario",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
		AppRoot:      tempDir,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runPlaybooksPhase(context.Background(), env, io.Discard)
	}
}

func BenchmarkRunPlaybooksPhaseEmptyRegistry(b *testing.B) {
	tempDir := b.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenarios", "bench-scenario")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "ui"), 0o755); err != nil {
		b.Fatalf("mkdir ui dir: %v", err)
	}
	basDir := filepath.Join(scenarioDir, "bas")
	if err := os.MkdirAll(basDir, 0o755); err != nil {
		b.Fatalf("mkdir bas dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(basDir, "registry.json"), []byte(`{"playbooks":[]}`), 0o644); err != nil {
		b.Fatalf("write registry: %v", err)
	}

	env := workspace.Environment{
		ScenarioName: "bench-scenario",
		ScenarioDir:  scenarioDir,
		TestDir:      filepath.Join(scenarioDir, "test"),
		AppRoot:      tempDir,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runPlaybooksPhase(context.Background(), env, io.Discard)
	}
}
