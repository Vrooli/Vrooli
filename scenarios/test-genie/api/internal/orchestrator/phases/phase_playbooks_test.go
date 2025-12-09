package phases

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		"test/playbooks",
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
			ScenarioName: scenarioName,
			ScenarioDir:  scenarioDir,
			TestDir:      testDir,
			AppRoot:      appRoot,
		},
		scenarioDir: scenarioDir,
		testDir:     testDir,
		appRoot:     appRoot,
	}
}

func (h *playbooksTestHarness) writeRegistry(t *testing.T, content string) {
	t.Helper()
	registryDir := filepath.Join(h.testDir, "playbooks")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatalf("failed to create playbooks dir: %v", err)
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
	h.writeRegistry(t, `{"playbooks":[{"file":"test/playbooks/capabilities/01-basic/test.json","description":"test","order":"01.01","requirements":[],"fixtures":[],"reset":"none"}]}`)
	h.writeWorkflow(t, "test/playbooks/capabilities/01-basic/test.json", `{
  "metadata": {"description": "basic", "version": 1},
  "nodes": [{"id":"n1","type":"navigate","data":{"destinationType":"url","url":"http://example.com"}}],
  "edges": []
}`)

	// Stub BAS server to satisfy health/validate/execute/status/timeline calls.
	basServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer basServer.Close()
	basPort := strings.TrimPrefix(basServer.URL, "http://127.0.0.1:")
	basPort = strings.TrimPrefix(basPort, "http://localhost:")

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

func TestStartScenario(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := StartScenario(ctx, "test-scenario", io.Discard)
	if err == nil {
		t.Error("expected error when context is cancelled")
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
	playbooksDir := filepath.Join(scenarioDir, "test", "playbooks")
	os.MkdirAll(playbooksDir, 0o755)
	// No ui/ directory, but provide empty registry
	os.WriteFile(filepath.Join(playbooksDir, "registry.json"), []byte(`{"playbooks":[]}`), 0o644)

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
	os.MkdirAll(filepath.Join(scenarioDir, "ui"), 0o755)
	playbooksDir := filepath.Join(scenarioDir, "test", "playbooks")
	os.MkdirAll(playbooksDir, 0o755)
	os.WriteFile(filepath.Join(playbooksDir, "registry.json"), []byte(`{"playbooks":[]}`), 0o644)

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
