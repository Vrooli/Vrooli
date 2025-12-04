package dependencies

import (
	"context"
	"fmt"
	"io"
	"testing"

	"test-genie/internal/dependencies/commands"
	"test-genie/internal/dependencies/packages"
	"test-genie/internal/dependencies/resources"
	"test-genie/internal/dependencies/runtime"
	"test-genie/internal/structure/types"
)

// mockCommandChecker implements commands.Checker for testing.
type mockCommandChecker struct {
	checkFunc    func(name string) (string, error)
	checkAllFunc func(reqs []commands.CommandRequirement) commands.Result
}

func (m *mockCommandChecker) Check(name string) (string, error) {
	if m.checkFunc != nil {
		return m.checkFunc(name)
	}
	return "/usr/bin/" + name, nil
}

func (m *mockCommandChecker) CheckAll(reqs []commands.CommandRequirement) commands.Result {
	if m.checkAllFunc != nil {
		return m.checkAllFunc(reqs)
	}
	var obs []types.Observation
	for _, req := range reqs {
		obs = append(obs, types.NewSuccessObservation(fmt.Sprintf("command available: %s", req.Name)))
	}
	return commands.Result{Success: true, Observations: obs}
}

// mockRuntimeDetector implements runtime.Detector for testing.
type mockRuntimeDetector struct {
	runtimes []runtime.Runtime
}

func (m *mockRuntimeDetector) Detect() []runtime.Runtime {
	return m.runtimes
}

// mockPackageDetector implements packages.Detector for testing.
type mockPackageDetector struct {
	managers      []packages.Manager
	nodeWorkspace bool
}

func (m *mockPackageDetector) Detect() []packages.Manager {
	return m.managers
}

func (m *mockPackageDetector) HasNodeWorkspace() bool {
	return m.nodeWorkspace
}

// mockResourceLoader implements resources.ExpectationsLoader for testing.
type mockResourceLoader struct {
	resources []string
	err       error
}

func (m *mockResourceLoader) Load() ([]string, error) {
	return m.resources, m.err
}

// mockResourceChecker implements resources.HealthChecker for testing.
type mockResourceChecker struct {
	result resources.HealthResult
}

func (m *mockResourceChecker) Check(ctx context.Context) resources.HealthResult {
	return m.result
}

// Ensure mock types satisfy interfaces at compile time.
var (
	_ commands.Checker            = (*mockCommandChecker)(nil)
	_ runtime.Detector            = (*mockRuntimeDetector)(nil)
	_ packages.Detector           = (*mockPackageDetector)(nil)
	_ resources.ExpectationsLoader = (*mockResourceLoader)(nil)
	_ resources.HealthChecker     = (*mockResourceChecker)(nil)
)

func TestRunnerRunSuccess(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config,
		WithLogger(io.Discard),
		WithCommandChecker(&mockCommandChecker{}),
		WithRuntimeDetector(&mockRuntimeDetector{
			runtimes: []runtime.Runtime{
				{Name: "Go", Command: "go", Reason: "test"},
			},
		}),
		WithPackageDetector(&mockPackageDetector{
			managers: []packages.Manager{
				{Name: "pnpm", Reason: "test"},
			},
		}),
		WithResourceLoader(&mockResourceLoader{
			resources: []string{"postgres"},
		}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Fatalf("expected observations")
	}
}

func TestRunnerRunFailsOnMissingCommand(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config,
		WithLogger(io.Discard),
		WithCommandChecker(&mockCommandChecker{
			checkAllFunc: func(reqs []commands.CommandRequirement) commands.Result {
				return commands.Result{
					Success:     false,
					Error:       fmt.Errorf("missing: bash"),
					Remediation: "Install bash",
				}
			},
		}),
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatalf("expected failure for missing command")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Fatalf("expected missing_dependency classification, got %s", result.FailureClass)
	}
}

func TestRunnerRunFailsOnResourceLoadError(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config,
		WithLogger(io.Discard),
		WithCommandChecker(&mockCommandChecker{}),
		WithRuntimeDetector(&mockRuntimeDetector{}),
		WithPackageDetector(&mockPackageDetector{}),
		WithResourceLoader(&mockResourceLoader{
			err: fmt.Errorf("manifest not found"),
		}),
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatalf("expected failure for resource load error")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Fatalf("expected misconfiguration classification, got %s", result.FailureClass)
	}
}

func TestRunnerRunFailsOnUnhealthyResource(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config,
		WithLogger(io.Discard),
		WithCommandChecker(&mockCommandChecker{}),
		WithRuntimeDetector(&mockRuntimeDetector{}),
		WithPackageDetector(&mockPackageDetector{}),
		WithResourceLoader(&mockResourceLoader{
			resources: []string{"postgres"},
		}),
		WithResourceChecker(&mockResourceChecker{
			result: resources.HealthResult{
				Success:      false,
				Error:        fmt.Errorf("postgres unhealthy"),
				FailureClass: "missing_dependency",
				Remediation:  "Start postgres",
			},
		}),
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatalf("expected failure for unhealthy resource")
	}
}

func TestRunnerRunWithCancelledContext(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config, WithLogger(io.Discard))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := runner.Run(ctx)

	if result.Success {
		t.Fatalf("expected failure for cancelled context")
	}
	if result.FailureClass != FailureClassSystem {
		t.Fatalf("expected system classification, got %s", result.FailureClass)
	}
}

func TestRunnerRunNoRuntimes(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config,
		WithLogger(io.Discard),
		WithCommandChecker(&mockCommandChecker{}),
		WithRuntimeDetector(&mockRuntimeDetector{}), // No runtimes
		WithPackageDetector(&mockPackageDetector{}),
		WithResourceLoader(&mockResourceLoader{}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

func TestRunnerRunTracksValidationSummary(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config,
		WithLogger(io.Discard),
		WithCommandChecker(&mockCommandChecker{}),
		WithRuntimeDetector(&mockRuntimeDetector{
			runtimes: []runtime.Runtime{
				{Name: "Go", Command: "go"},
			},
		}),
		WithPackageDetector(&mockPackageDetector{
			managers: []packages.Manager{
				{Name: "pnpm"},
			},
		}),
		WithResourceLoader(&mockResourceLoader{
			resources: []string{"postgres", "redis"},
		}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success")
	}
	if result.Summary.RuntimesDetected != 1 {
		t.Fatalf("expected 1 runtime, got %d", result.Summary.RuntimesDetected)
	}
	if result.Summary.ManagersDetected != 1 {
		t.Fatalf("expected 1 manager, got %d", result.Summary.ManagersDetected)
	}
	if result.Summary.ResourcesChecked != 2 {
		t.Fatalf("expected 2 resources, got %d", result.Summary.ResourcesChecked)
	}
}

func TestValidationSummary_TotalChecks(t *testing.T) {
	tests := []struct {
		name     string
		summary  ValidationSummary
		expected int
	}{
		{
			name: "all categories",
			summary: ValidationSummary{
				CommandsChecked:  5,
				RuntimesDetected: 2,
				ManagersDetected: 1,
				ResourcesChecked: 3,
			},
			expected: 11,
		},
		{
			name: "commands only",
			summary: ValidationSummary{
				CommandsChecked: 3,
			},
			expected: 3,
		},
		{
			name:     "empty",
			summary:  ValidationSummary{},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.summary.TotalChecks(); got != tc.expected {
				t.Errorf("TotalChecks() = %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestValidationSummary_String(t *testing.T) {
	tests := []struct {
		name        string
		summary     ValidationSummary
		wantParts   []string
		notWantParts []string
	}{
		{
			name: "all categories",
			summary: ValidationSummary{
				CommandsChecked:  5,
				RuntimesDetected: 2,
				ManagersDetected: 1,
				ResourcesChecked: 3,
			},
			wantParts: []string{"5 commands", "2 runtimes", "1 managers", "3 resources"},
		},
		{
			name: "commands and runtimes only",
			summary: ValidationSummary{
				CommandsChecked:  3,
				RuntimesDetected: 1,
			},
			wantParts:    []string{"3 commands", "1 runtimes"},
			notWantParts: []string{"managers", "resources"},
		},
		{
			name:      "empty",
			summary:   ValidationSummary{},
			wantParts: []string{"no checks performed"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			str := tc.summary.String()
			for _, part := range tc.wantParts {
				if !containsStr(str, part) {
					t.Errorf("String() = %q, want to contain %q", str, part)
				}
			}
			for _, part := range tc.notWantParts {
				if containsStr(str, part) {
					t.Errorf("String() = %q, should not contain %q", str, part)
				}
			}
		})
	}
}

func TestRunnerRunFailsOnMissingRuntime(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config,
		WithLogger(io.Discard),
		WithCommandChecker(&mockCommandChecker{
			checkAllFunc: func(reqs []commands.CommandRequirement) commands.Result {
				// Baseline commands pass, but runtime check fails
				for _, req := range reqs {
					if req.Name == "go" {
						return commands.Result{
							Success:     false,
							Error:       fmt.Errorf("go not found"),
							Remediation: "Install Go 1.21+",
						}
					}
				}
				return commands.Result{Success: true}
			},
		}),
		WithRuntimeDetector(&mockRuntimeDetector{
			runtimes: []runtime.Runtime{
				{Name: "Go", Command: "go", Reason: "api/go.mod present"},
			},
		}),
		WithPackageDetector(&mockPackageDetector{}),
		WithResourceLoader(&mockResourceLoader{}),
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatalf("expected failure when runtime is missing")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Fatalf("expected missing_dependency classification, got %s", result.FailureClass)
	}
}

func TestRunnerRunFailsOnMissingPackageManager(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config,
		WithLogger(io.Discard),
		WithCommandChecker(&mockCommandChecker{
			checkAllFunc: func(reqs []commands.CommandRequirement) commands.Result {
				// Check if this is a package manager check (contains pnpm)
				for _, req := range reqs {
					if req.Name == "pnpm" {
						return commands.Result{
							Success:     false,
							Error:       fmt.Errorf("pnpm not found"),
							Remediation: "Install pnpm: npm install -g pnpm",
						}
					}
				}
				return commands.Result{Success: true}
			},
		}),
		WithRuntimeDetector(&mockRuntimeDetector{}),
		WithPackageDetector(&mockPackageDetector{
			managers: []packages.Manager{
				{Name: "pnpm", Reason: "pnpm-lock.yaml present"},
			},
		}),
		WithResourceLoader(&mockResourceLoader{}),
	)

	result := runner.Run(context.Background())

	if result.Success {
		t.Fatalf("expected failure when package manager is missing")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Fatalf("expected missing_dependency classification, got %s", result.FailureClass)
	}
}

func TestRunnerRunWithNodeWorkspaceButNoManager(t *testing.T) {
	config := Config{
		ScenarioDir:  "/scenarios/demo",
		ScenarioName: "demo",
	}

	runner := New(config,
		WithLogger(io.Discard),
		WithCommandChecker(&mockCommandChecker{}),
		WithRuntimeDetector(&mockRuntimeDetector{}),
		WithPackageDetector(&mockPackageDetector{
			managers:      nil, // No managers detected
			nodeWorkspace: true, // But has node workspace
		}),
		WithResourceLoader(&mockResourceLoader{}),
	)

	result := runner.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	// Should have an observation about defaulting to pnpm
	foundDefaultObs := false
	for _, obs := range result.Observations {
		if containsStr(obs.Message, "pnpm") && containsStr(obs.Message, "default") {
			foundDefaultObs = true
			break
		}
	}
	if !foundDefaultObs {
		t.Error("expected observation about defaulting to pnpm for node workspace")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
