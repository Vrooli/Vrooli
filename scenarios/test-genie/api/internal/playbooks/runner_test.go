package playbooks

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/playbooks/artifacts"
	"test-genie/internal/playbooks/execution"
	"test-genie/internal/playbooks/registry"
	"test-genie/internal/playbooks/seeds"
	"test-genie/internal/playbooks/workflow"
)

// Mock implementations for testing

type mockRegistryLoader struct {
	registry Registry
	err      error
}

func (m *mockRegistryLoader) Load() (Registry, error) {
	return m.registry, m.err
}

type mockWorkflowResolver struct {
	definition map[string]any
	err        error
}

func (m *mockWorkflowResolver) Resolve(ctx context.Context, path string) (map[string]any, error) {
	return m.definition, m.err
}

type mockBASClient struct {
	healthErr       error
	executeID       string
	executeErr      error
	status          ExecutionStatus
	statusErr       error
	waitErr         error
	timeline        []byte
	timelineErr     error
	healthCallCount int
	waitCallCount   int
}

func (m *mockBASClient) Health(ctx context.Context) error {
	m.healthCallCount++
	return m.healthErr
}

func (m *mockBASClient) WaitForHealth(ctx context.Context) error {
	return m.healthErr
}

func (m *mockBASClient) ExecuteWorkflow(ctx context.Context, definition map[string]any, name string) (string, error) {
	return m.executeID, m.executeErr
}

func (m *mockBASClient) GetStatus(ctx context.Context, executionID string) (ExecutionStatus, error) {
	return m.status, m.statusErr
}

func (m *mockBASClient) WaitForCompletion(ctx context.Context, executionID string) error {
	m.waitCallCount++
	return m.waitErr
}

func (m *mockBASClient) GetTimeline(ctx context.Context, executionID string) ([]byte, error) {
	return m.timeline, m.timelineErr
}

type mockSeedManager struct {
	hasSeeds bool
	cleanup  func()
	err      error
}

func (m *mockSeedManager) Apply(ctx context.Context) (func(), error) {
	return m.cleanup, m.err
}

func (m *mockSeedManager) HasSeeds() bool {
	return m.hasSeeds
}

type mockArtifactWriter struct {
	timelinePath string
	timelineErr  error
	phaseErr     error
	writeCount   int
}

func (m *mockArtifactWriter) WriteTimeline(workflowFile string, timelineData []byte) (string, error) {
	return m.timelinePath, m.timelineErr
}

func (m *mockArtifactWriter) WritePhaseResults(results []Result) error {
	m.writeCount++
	return m.phaseErr
}

// Tests

func TestRunnerRunSkippedViaEnv(t *testing.T) {
	os.Setenv("TEST_GENIE_SKIP_PLAYBOOKS", "1")
	defer os.Unsetenv("TEST_GENIE_SKIP_PLAYBOOKS")

	runner := New(Config{})
	result := runner.Run(context.Background())

	if !result.Success {
		t.Error("expected success when skipped via env")
	}
	if len(result.Observations) != 1 {
		t.Errorf("expected 1 observation, got %d", len(result.Observations))
	}
}

func TestRunnerRunNoUI(t *testing.T) {
	tempDir := t.TempDir()
	// Don't create ui/ directory

	runner := New(Config{ScenarioDir: tempDir})
	result := runner.Run(context.Background())

	if !result.Success {
		t.Error("expected success when no UI")
	}
}

func TestRunnerRunRegistryError(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{err: errors.New("registry error")}),
	)

	result := runner.Run(context.Background())
	if result.Success {
		t.Error("expected failure for registry error")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %v", result.FailureClass)
	}
}

func TestRunnerRunEmptyRegistry(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{registry: Registry{Playbooks: []Entry{}}}),
	)

	result := runner.Run(context.Background())
	if !result.Success {
		t.Error("expected success for empty registry")
	}
}

func TestRunnerRunBASUnavailable(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "test.json"}}},
		}),
		WithBASClient(&mockBASClient{healthErr: errors.New("connection refused")}),
	)

	result := runner.Run(context.Background())
	if result.Success {
		t.Error("expected failure when BAS unavailable")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Errorf("expected missing dependency, got %v", result.FailureClass)
	}
}

func TestRunnerRunSeedError(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "test.json"}}},
		}),
		WithBASClient(&mockBASClient{}),
		WithSeedManager(&mockSeedManager{err: errors.New("seed error")}),
	)

	result := runner.Run(context.Background())
	if result.Success {
		t.Error("expected failure for seed error")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %v", result.FailureClass)
	}
}

func TestRunnerRunWorkflowSuccess(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir, ScenarioName: "test"},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{
				{File: "test.json", Description: "Test workflow", Requirements: []string{"REQ-1"}},
			}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{
			definition: map[string]any{"nodes": []any{}, "edges": []any{}},
		}),
		WithBASClient(&mockBASClient{
			executeID: "exec-123",
			timeline:  []byte(`{"frames": [{"step_type": "navigate"}]}`),
		}),
		WithSeedManager(&mockSeedManager{}),
		WithArtifactWriter(&mockArtifactWriter{}),
	)

	result := runner.Run(context.Background())
	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
	// Expect 2 observations: workflow completion + summary
	if len(result.Observations) != 2 {
		t.Errorf("expected 2 observations (workflow + summary), got %d", len(result.Observations))
	}
	// Verify summary is populated
	if result.Summary.WorkflowsExecuted != 1 {
		t.Errorf("expected 1 workflow executed, got %d", result.Summary.WorkflowsExecuted)
	}
	if result.Summary.WorkflowsPassed != 1 {
		t.Errorf("expected 1 workflow passed, got %d", result.Summary.WorkflowsPassed)
	}
}

func TestRunnerRunWorkflowFailure(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir, ScenarioName: "test"},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "test.json"}}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{
			definition: map[string]any{"nodes": []any{}},
		}),
		WithBASClient(&mockBASClient{
			executeID: "exec-123",
			waitErr:   errors.New("element not found"),
			timeline:  []byte(`{"frames": []}`),
		}),
		WithSeedManager(&mockSeedManager{}),
		WithArtifactWriter(&mockArtifactWriter{timelinePath: "artifacts/timeline.json"}),
	)

	result := runner.Run(context.Background())
	if result.Success {
		t.Error("expected failure for workflow error")
	}
	if result.FailureClass != FailureClassExecution {
		t.Errorf("expected execution failure, got %v", result.FailureClass)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].ArtifactPath == "" {
		t.Error("expected artifact path to be set on failure")
	}
}

func TestRunnerRunMultipleWorkflows(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir, ScenarioName: "test"},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{
				{File: "workflow1.json", Description: "First"},
				{File: "workflow2.json", Description: "Second"},
				{File: "workflow3.json", Description: "Third"},
			}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{
			definition: map[string]any{"nodes": []any{}},
		}),
		WithBASClient(&mockBASClient{executeID: "exec-123"}),
		WithSeedManager(&mockSeedManager{}),
		WithArtifactWriter(&mockArtifactWriter{}),
	)

	result := runner.Run(context.Background())
	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}
	if len(result.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Results))
	}
}

func TestRunnerRunContextCanceled(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	runner := New(Config{ScenarioDir: tempDir})
	result := runner.Run(ctx)

	if result.Success {
		t.Error("expected failure for canceled context")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure, got %v", result.FailureClass)
	}
}

func TestRunnerRunWithUIBaseURL(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir, ScenarioName: "test"},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "test.json"}}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{
			definition: map[string]any{
				"nodes": []any{
					map[string]any{
						"data": map[string]any{
							"url": "${BASE_URL}/login",
						},
					},
				},
			},
		}),
		WithBASClient(&mockBASClient{executeID: "exec-123"}),
		WithSeedManager(&mockSeedManager{}),
		WithArtifactWriter(&mockArtifactWriter{}),
		WithUIBaseURLResolver(func(ctx context.Context, scenario string) (string, error) {
			return "http://localhost:3000", nil
		}),
	)

	result := runner.Run(context.Background())
	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}
}

func TestRunnerSeedCleanupCalled(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	cleanupCalled := false
	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "test.json"}}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{
			definition: map[string]any{"nodes": []any{}},
		}),
		WithBASClient(&mockBASClient{executeID: "exec-123"}),
		WithSeedManager(&mockSeedManager{
			cleanup: func() { cleanupCalled = true },
		}),
		WithArtifactWriter(&mockArtifactWriter{}),
	)

	runner.Run(context.Background())

	if !cleanupCalled {
		t.Error("expected seed cleanup to be called")
	}
}

func TestRunnerWorkflowResolverError(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "missing.json"}}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{err: errors.New("file not found")}),
		WithBASClient(&mockBASClient{}),
		WithSeedManager(&mockSeedManager{}),
		WithArtifactWriter(&mockArtifactWriter{}),
	)

	result := runner.Run(context.Background())
	if result.Success {
		t.Error("expected failure for resolver error")
	}
	if !strings.Contains(result.Error.Error(), "failed to resolve") {
		t.Errorf("expected resolve error, got: %v", result.Error)
	}
}

func TestRunnerPhaseResultWriteError(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "test.json"}}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{
			definition: map[string]any{"nodes": []any{}},
		}),
		WithBASClient(&mockBASClient{executeID: "exec-123"}),
		WithSeedManager(&mockSeedManager{}),
		WithArtifactWriter(&mockArtifactWriter{phaseErr: errors.New("write error")}),
	)

	result := runner.Run(context.Background())
	if result.Success {
		t.Error("expected failure for phase result write error")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure, got %v", result.FailureClass)
	}
}

func TestRunnerExecutionOutcomeFields(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "test.json"}}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{
			definition: map[string]any{"nodes": []any{}},
		}),
		WithBASClient(&mockBASClient{
			executeID: "exec-456",
			timeline:  []byte(`{"frames": [{"step_type": "assert", "status": "completed"}]}`),
		}),
		WithSeedManager(&mockSeedManager{}),
		WithArtifactWriter(&mockArtifactWriter{}),
	)

	result := runner.Run(context.Background())
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}

	outcome := result.Results[0].Outcome
	if outcome == nil {
		t.Fatal("expected outcome to be set")
	}
	if outcome.ExecutionID != "exec-456" {
		t.Errorf("expected exec-456, got %s", outcome.ExecutionID)
	}
	if outcome.Duration < 0 {
		t.Error("expected non-negative duration")
	}
	// Stats should contain assertion info
	if !strings.Contains(outcome.Stats, "assertion") {
		t.Errorf("expected stats with assertion info, got %s", outcome.Stats)
	}
}

func TestRunnerEnsureBASWithPortResolver(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}

	portResolverCalled := false
	startCalled := false

	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "test.json"}}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{
			definition: map[string]any{"nodes": []any{}},
		}),
		WithSeedManager(&mockSeedManager{}),
		WithArtifactWriter(&mockArtifactWriter{}),
		WithPortResolver(func(ctx context.Context, scenario, portName string) (string, error) {
			portResolverCalled = true
			if scenario != BASScenarioName {
				t.Errorf("expected %s, got %s", BASScenarioName, scenario)
			}
			return "", errors.New("not found")
		}),
		WithScenarioStarter(func(ctx context.Context, scenario string) error {
			startCalled = true
			return errors.New("start failed")
		}),
	)

	result := runner.Run(context.Background())
	// Should fail because BAS couldn't be started
	if result.Success {
		t.Error("expected failure when BAS unavailable")
	}
	if !portResolverCalled {
		t.Error("expected port resolver to be called")
	}
	if !startCalled {
		t.Error("expected scenario starter to be called")
	}
}

// Benchmark tests

func BenchmarkRunnerRunNoWorkflows(b *testing.B) {
	tempDir := b.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755)

	runner := New(Config{ScenarioDir: tempDir},
		WithRegistryLoader(&mockRegistryLoader{registry: Registry{}}),
		WithBASClient(&mockBASClient{}),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}

func BenchmarkRunnerRunSingleWorkflow(b *testing.B) {
	tempDir := b.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "ui"), 0o755)

	runner := New(Config{ScenarioDir: tempDir, ScenarioName: "bench"},
		WithRegistryLoader(&mockRegistryLoader{
			registry: Registry{Playbooks: []Entry{{File: "test.json"}}},
		}),
		WithWorkflowResolver(&mockWorkflowResolver{
			definition: map[string]any{"nodes": []any{}},
		}),
		WithBASClient(&mockBASClient{executeID: "exec-123"}),
		WithSeedManager(&mockSeedManager{}),
		WithArtifactWriter(&mockArtifactWriter{}),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(context.Background())
	}
}

// Compile-time interface checks ensure mocks satisfy the interfaces they're mocking.
// This prevents test drift when interfaces change.
var (
	_ registry.Loader   = (*mockRegistryLoader)(nil)
	_ workflow.Resolver = (*mockWorkflowResolver)(nil)
	_ execution.Client  = (*mockBASClient)(nil)
	_ seeds.Manager     = (*mockSeedManager)(nil)
	_ artifacts.Writer  = (*mockArtifactWriter)(nil)
)
