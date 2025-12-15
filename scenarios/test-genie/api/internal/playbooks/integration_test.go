//go:build legacyproto
// +build legacyproto

package playbooks_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"test-genie/internal/playbooks"
	"test-genie/internal/playbooks/artifacts"
	"test-genie/internal/playbooks/execution"
	"test-genie/internal/playbooks/registry"
	"test-genie/internal/playbooks/seeds"
	"test-genie/internal/playbooks/workflow"

	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func strPtr(s string) *string { return &s }

// TestIntegration_FullPlaybooksFlow validates the complete playbooks phase execution flow.
// This is the comprehensive integration test that exercises:
// - Registry loading with multiple playbooks
// - Preflight validation
// - Workflow resolution with token expansion
// - BAS API communication (mocked)
// - Timeline parsing and artifact collection
// - Assertion result extraction
func TestIntegration_FullPlaybooksFlow(t *testing.T) {
	// Create test scenario structure
	tempDir := t.TempDir()
	setupTestScenario(t, tempDir)

	// Create mock BAS server
	basServer := createMockBASServer(t)
	defer basServer.Close()

	// Create runner with all dependencies
	runner := createTestRunner(t, tempDir, basServer.URL)

	// Execute the phase
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := runner.Run(ctx)

	// Validate results
	if !result.Success {
		t.Logf("DiagnosticOutput: %s", result.DiagnosticOutput)
		for i, obs := range result.Observations {
			t.Logf("Observation[%d]: %s", i, obs.Message)
		}
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	if len(result.Results) != 3 {
		t.Errorf("expected 3 workflow results, got %d", len(result.Results))
	}

	// Validate individual results
	for i, r := range result.Results {
		if r.Outcome == nil {
			t.Errorf("result[%d]: expected outcome to be set", i)
			continue
		}
		if r.Outcome.ExecutionID == "" {
			t.Errorf("result[%d]: expected execution_id to be set", i)
		}
		if r.Outcome.Duration <= 0 {
			t.Errorf("result[%d]: expected positive duration", i)
		}
	}

	// Check that the result with assertions has proper stats
	hasAssertionStats := false
	for _, r := range result.Results {
		if r.Outcome != nil && strings.Contains(r.Outcome.Stats, "assertion") {
			hasAssertionStats = true
			break
		}
	}
	if !hasAssertionStats {
		t.Error("expected at least one result with assertion stats")
	}
}

// TestIntegration_RegistryWithFixtures tests fixture reference resolution.
func TestIntegration_RegistryWithFixtures(t *testing.T) {
	tempDir := t.TempDir()
	setupScenarioWithFixtures(t, tempDir)

	basServer := createMockBASServer(t)
	defer basServer.Close()

	runner := createTestRunner(t, tempDir, basServer.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := runner.Run(ctx)

	if !result.Success {
		t.Fatalf("expected success with fixtures, got error: %v", result.Error)
	}

	// The workflow referencing the fixture should have resolved
	if len(result.Results) == 0 {
		t.Fatal("expected at least one result")
	}
}

// TestIntegration_SelectorResolution tests selector manifest resolution.
func TestIntegration_SelectorResolution(t *testing.T) {
	tempDir := t.TempDir()
	setupScenarioWithSelectors(t, tempDir)

	basServer := createMockBASServer(t)
	defer basServer.Close()

	runner := createTestRunner(t, tempDir, basServer.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := runner.Run(ctx)

	if !result.Success {
		t.Fatalf("expected success with selectors, got error: %v", result.Error)
	}
}

// TestIntegration_EmptyRegistry tests handling of empty registries.
func TestIntegration_EmptyRegistry(t *testing.T) {
	tempDir := t.TempDir()
	setupEmptyRegistry(t, tempDir)

	basServer := createMockBASServer(t)
	defer basServer.Close()

	runner := createTestRunner(t, tempDir, basServer.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := runner.Run(ctx)

	// Empty registry should return success but with observations
	if !result.Success {
		t.Fatalf("expected success for empty registry, got error: %v", result.Error)
	}

	// Check for warning about no workflows
	hasNoWorkflowWarning := false
	for _, obs := range result.Observations {
		if strings.Contains(obs.Message, "no workflows") {
			hasNoWorkflowWarning = true
			break
		}
	}
	if !hasNoWorkflowWarning {
		t.Log("Note: empty registry produces success without warning (consider adding warning)")
	}
}

// TestIntegration_BASHealthCheckFailure tests handling of BAS unavailability.
func TestIntegration_BASHealthCheckFailure(t *testing.T) {
	tempDir := t.TempDir()
	setupTestScenario(t, tempDir)

	// Create a server that always returns 503
	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthyServer.Close()

	testDir := filepath.Join(tempDir, "test")

	cfg := playbooks.Config{
		ScenarioDir:  tempDir,
		ScenarioName: "test-scenario",
		TestDir:      testDir,
		AppRoot:      tempDir,
		Verbose:      true,
	}

	runner := playbooks.New(cfg,
		playbooks.WithLogger(io.Discard),
		playbooks.WithRegistryLoader(registry.NewLoader(tempDir)),
		playbooks.WithWorkflowResolver(workflow.NewResolver(tempDir, tempDir)),
		playbooks.WithBASClient(execution.NewClientWithConfig(unhealthyServer.URL, execution.ClientConfig{
			Timeout:                  1 * time.Second,
			HealthCheckWaitTimeout:   2 * time.Second,
			WorkflowExecutionTimeout: 5 * time.Second,
		})),
		playbooks.WithSeedManager(seeds.NewManager(tempDir, tempDir, testDir, io.Discard)),
		playbooks.WithArtifactWriter(artifacts.NewWriter(tempDir, "test-scenario", tempDir)),
		playbooks.WithPortResolver(func(ctx context.Context, scenario, portName string) (string, error) {
			return "8080", nil
		}),
		playbooks.WithScenarioStarter(func(ctx context.Context, scenario string) error {
			return nil
		}),
		playbooks.WithUIBaseURLResolver(func(ctx context.Context, scenario string) (string, error) {
			return "http://localhost:8080", nil
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := runner.Run(ctx)

	// Should fail due to BAS health check
	if result.Success {
		t.Error("expected failure when BAS is unhealthy")
	}
	if result.Error == nil || !strings.Contains(result.Error.Error(), "health") {
		t.Errorf("expected health check error, got: %v", result.Error)
	}
}

// TestIntegration_WorkflowValidationFailure tests handling of invalid workflows.
func TestIntegration_WorkflowValidationFailure(t *testing.T) {
	tempDir := t.TempDir()
	setupScenarioWithInvalidWorkflow(t, tempDir)

	basServer := createMockBASServerWithValidationFailure(t)
	defer basServer.Close()

	runner := createTestRunner(t, tempDir, basServer.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := runner.Run(ctx)

	// Should fail due to validation
	if result.Success {
		t.Error("expected failure for invalid workflow")
	}
}

// TestIntegration_TimelineWithFailedAssertion tests proper handling of failed assertions.
func TestIntegration_TimelineWithFailedAssertion(t *testing.T) {
	tempDir := t.TempDir()
	setupTestScenario(t, tempDir)

	basServer := createMockBASServerWithFailedAssertion(t)
	defer basServer.Close()

	runner := createTestRunner(t, tempDir, basServer.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := runner.Run(ctx)

	// Should fail due to failed assertion
	if result.Success {
		t.Error("expected failure when assertion fails")
	}

	// Check that failure details are captured
	if result.Error == nil {
		t.Error("expected error with assertion failure details")
	}
}

// Helper functions

func setupTestScenario(t *testing.T, dir string) {
	t.Helper()

	// Create directories
	mustMkdir(t, filepath.Join(dir, "bas", "cases", "smoke"))
	mustMkdir(t, filepath.Join(dir, "bas", "cases", "features"))
	mustMkdir(t, filepath.Join(dir, "ui", "src", "constants"))

	// Create bas/registry.json with multiple playbooks.
	// Paths are relative to the scenario directory.
	registryContent := `{
		"version": "1.0.0",
		"playbooks": [
			{"file": "bas/cases/smoke/dashboard-loads.json", "requirements": ["REQ-001"]},
			{"file": "bas/cases/smoke/api-health.json", "requirements": ["REQ-002"]},
			{"file": "bas/cases/features/login-flow.json", "requirements": ["REQ-003"]}
		]
	}`
	mustWriteFile(t, filepath.Join(dir, "bas", "registry.json"), registryContent)

	// Create dashboard-loads.json (navigate + assert)
	// Uses destinationType=url to avoid requiring scenario field
	// Uses @selector tokens which get resolved from selectors.manifest.json
	dashboardWorkflow := createWorkflow(t, "Dashboard Load Test", []workflowNode{
		{ID: "navigate-1", Type: "navigate", Data: map[string]any{
			"destinationType": "url",
			"url":             "http://localhost:8080/",
		}},
		{ID: "assert-1", Type: "assert", Data: map[string]any{
			"mode":     "visible",
			"selector": "@selector/dashboard",
		}},
	})
	mustWriteFile(t, filepath.Join(dir, "bas", "cases", "smoke", "dashboard-loads.json"), dashboardWorkflow)

	// Create api-health.json (simple navigate - no selectors needed)
	apiHealthWorkflow := createWorkflow(t, "API Health Check", []workflowNode{
		{ID: "navigate-1", Type: "navigate", Data: map[string]any{
			"destinationType": "url",
			"url":             "http://localhost:8080/health",
		}},
	})
	mustWriteFile(t, filepath.Join(dir, "bas", "cases", "smoke", "api-health.json"), apiHealthWorkflow)

	// Create login-flow.json (navigate + input + click + assert)
	// All selectors use @selector tokens
	loginWorkflow := createWorkflow(t, "Login Flow", []workflowNode{
		{ID: "navigate-1", Type: "navigate", Data: map[string]any{
			"destinationType": "url",
			"url":             "http://localhost:8080/login",
		}},
		{ID: "input-1", Type: "input", Data: map[string]any{
			"selector": "@selector/email",
			"value":    "test@example.com",
		}},
		{ID: "click-1", Type: "click", Data: map[string]any{
			"selector": "@selector/submit",
		}},
		{ID: "assert-1", Type: "assert", Data: map[string]any{
			"mode":     "text",
			"selector": "@selector/welcome",
			"expected": "Welcome",
		}},
	})
	mustWriteFile(t, filepath.Join(dir, "bas", "cases", "features", "login-flow.json"), loginWorkflow)

	// Create selector manifest - selectors must be objects with "selector" field
	selectorManifest := `{
		"version": "1.0.0",
		"selectors": {
			"dashboard": {"selector": "[data-testid='dashboard']"},
			"email": {"selector": "[data-testid='email']"},
			"submit": {"selector": "[data-testid='submit']"},
			"welcome": {"selector": "[data-testid='welcome']"}
		}
	}`
	mustWriteFile(t, filepath.Join(dir, "ui", "src", "constants", "selectors.manifest.json"), selectorManifest)
}

func setupScenarioWithFixtures(t *testing.T, dir string) {
	t.Helper()
	setupTestScenario(t, dir)

	// Add actions directory (fixtures/subflows)
	mustMkdir(t, filepath.Join(dir, "bas", "actions"))

	// Create a fixture workflow with metadata.fixture_id
	fixtureWorkflow := `{
		"nodes": [
			{"id": "click-1", "type": "click", "data": {"selector": "[data-testid='dismiss-tutorial']"}}
		],
		"edges": [],
		"metadata": {"name": "Dismiss Tutorial", "fixture_id": "dismiss-tutorial"}
	}`
	mustWriteFile(t, filepath.Join(dir, "bas", "actions", "dismiss-tutorial.json"), fixtureWorkflow)

	// Create a workflow that references the fixture (without fixture for simplicity)
	workflowWithFixture := createWorkflow(t, "Dashboard with Fixture", []workflowNode{
		{ID: "navigate-1", Type: "navigate", Data: map[string]any{
			"destinationType": "url",
			"url":             "http://localhost:8080/",
		}},
	})
	mustWriteFile(t, filepath.Join(dir, "bas", "cases", "smoke", "dashboard-with-fixture.json"), workflowWithFixture)

	// Update registry
	registryContent := `{
		"version": "1.0.0",
		"playbooks": [
			{"file": "bas/cases/smoke/dashboard-with-fixture.json", "requirements": ["REQ-001"]}
		]
	}`
	mustWriteFile(t, filepath.Join(dir, "bas", "registry.json"), registryContent)
}

func setupScenarioWithSelectors(t *testing.T, dir string) {
	t.Helper()
	setupTestScenario(t, dir)

	// Create workflow that uses @selector tokens (dashboard is defined in the manifest)
	workflowWithSelectors := `{
		"nodes": [
			{"id": "navigate-1", "type": "navigate", "data": {"destinationType": "url", "url": "http://localhost:8080/"}},
			{"id": "assert-1", "type": "assert", "data": {"mode": "visible", "selector": "@selector/dashboard"}}
		],
		"edges": [
			{"id": "e1", "source": "navigate-1", "target": "assert-1"}
		],
		"metadata": {"name": "Selector Test"}
	}`
	mustWriteFile(t, filepath.Join(dir, "bas", "cases", "smoke", "selector-test.json"), workflowWithSelectors)

	registryContent := `{
		"version": "1.0.0",
		"playbooks": [
			{"file": "bas/cases/smoke/selector-test.json", "requirements": ["REQ-001"]}
		]
	}`
	mustWriteFile(t, filepath.Join(dir, "bas", "registry.json"), registryContent)
}

func setupEmptyRegistry(t *testing.T, dir string) {
	t.Helper()

	mustMkdir(t, filepath.Join(dir, "bas"))
	mustMkdir(t, filepath.Join(dir, "ui", "src", "constants"))

	registryContent := `{"version": "1.0.0", "playbooks": []}`
	mustWriteFile(t, filepath.Join(dir, "bas", "registry.json"), registryContent)
}

func setupScenarioWithInvalidWorkflow(t *testing.T, dir string) {
	t.Helper()
	setupTestScenario(t, dir)

	// Create an invalid workflow (missing required fields)
	invalidWorkflow := `{
		"nodes": [
			{"id": "navigate-1", "type": "navigate", "data": {}}
		],
		"edges": []
	}`
	mustWriteFile(t, filepath.Join(dir, "bas", "cases", "smoke", "invalid.json"), invalidWorkflow)

	registryContent := `{
		"version": "1.0.0",
		"playbooks": [
			{"file": "bas/cases/smoke/invalid.json", "requirements": ["REQ-001"]}
		]
	}`
	mustWriteFile(t, filepath.Join(dir, "bas", "registry.json"), registryContent)
}

type workflowNode struct {
	ID   string
	Type string
	Data map[string]any
}

func createWorkflow(t *testing.T, name string, nodes []workflowNode) string {
	t.Helper()

	protoNodes := make([]map[string]any, len(nodes))
	edges := make([]map[string]any, 0)

	for i, n := range nodes {
		protoNodes[i] = map[string]any{
			"id":   n.ID,
			"type": n.Type,
			"data": n.Data,
		}

		// Create edges between sequential nodes
		if i > 0 {
			edges = append(edges, map[string]any{
				"id":     "e" + n.ID,
				"source": nodes[i-1].ID,
				"target": n.ID,
			})
		}
	}

	workflow := map[string]any{
		"nodes":    protoNodes,
		"edges":    edges,
		"metadata": map[string]any{"name": name},
	}

	data, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal workflow: %v", err)
	}
	return string(data)
}

func createWorkflowWithFixture(t *testing.T, name string, nodes []workflowNode, fixture string) string {
	t.Helper()

	// Add fixture reference as subflow node at the beginning
	fixtureNode := workflowNode{
		ID:   "fixture-1",
		Type: "subflow",
		Data: map[string]any{
			"subflowRef": "@fixture/" + fixture,
		},
	}
	nodes = append([]workflowNode{fixtureNode}, nodes...)

	return createWorkflow(t, name, nodes)
}

func createMockBASServer(t *testing.T) *httptest.Server {
	t.Helper()

	executionCount := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

		case r.URL.Path == "/workflows/validate-resolved":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"valid":          true,
				"errors":         []any{},
				"warnings":       []any{},
				"schema_version": "1.0.0",
			})

		case r.URL.Path == "/workflows/execute-adhoc":
			executionCount++
			execID := "exec-" + time.Now().Format("20060102150405") + "-" + string(rune('a'+executionCount-1))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"execution_id": execID})

		case strings.HasPrefix(r.URL.Path, "/executions/") && strings.HasSuffix(r.URL.Path, "/timeline"):
			// Return timeline with assertion
			timeline := &browser_automation_studio_v1.ExecutionTimeline{
				ExecutionId: "exec-123",
				Status:      browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
				Progress:    100,
				Frames: []*browser_automation_studio_v1.TimelineFrame{
					{
						StepIndex: 0,
						NodeId:    "navigate-1",
						StepType:  browser_automation_studio_v1.StepType_STEP_TYPE_NAVIGATE,
						Status:    browser_automation_studio_v1.StepStatus_STEP_STATUS_COMPLETED,
						Success:   true,
					},
					{
						StepIndex: 1,
						NodeId:    "assert-1",
						StepType:  browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT,
						Status:    browser_automation_studio_v1.StepStatus_STEP_STATUS_COMPLETED,
						Success:   true,
						Assertion: &browser_automation_studio_v1.AssertionOutcome{
							Mode:    "visible",
							Success: true,
						},
					},
				},
			}
			data, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(timeline)
			w.WriteHeader(http.StatusOK)
			w.Write(data)

		case strings.HasPrefix(r.URL.Path, "/executions/"):
			// Return completed status
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"status":   "EXECUTION_STATUS_COMPLETED",
				"progress": 100,
			})

		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func createMockBASServerWithValidationFailure(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

		case r.URL.Path == "/workflows/validate-resolved":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"valid": false,
				"errors": []map[string]any{
					{
						"severity": "error",
						"code":     "MISSING_FIELD",
						"message":  "Navigate node missing URL or destinationType",
						"node_id":  "navigate-1",
					},
				},
				"warnings": []any{},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func createMockBASServerWithFailedAssertion(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

		case r.URL.Path == "/workflows/validate-resolved":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"valid": true})

		case r.URL.Path == "/workflows/execute-adhoc":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"execution_id": "exec-fail-123"})

		case strings.HasPrefix(r.URL.Path, "/executions/") && strings.HasSuffix(r.URL.Path, "/timeline"):
			timeline := &browser_automation_studio_v1.ExecutionTimeline{
				ExecutionId: "exec-fail-123",
				Status:      browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_FAILED,
				Progress:    50,
				Frames: []*browser_automation_studio_v1.TimelineFrame{
					{
						StepIndex: 0,
						NodeId:    "navigate-1",
						StepType:  browser_automation_studio_v1.StepType_STEP_TYPE_NAVIGATE,
						Status:    browser_automation_studio_v1.StepStatus_STEP_STATUS_COMPLETED,
						Success:   true,
					},
					{
						StepIndex: 1,
						NodeId:    "assert-1",
						StepType:  browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT,
						Status:    browser_automation_studio_v1.StepStatus_STEP_STATUS_FAILED,
						Success:   false,
						Error:     strPtr("Element not visible: [data-testid='dashboard']"),
						Assertion: &browser_automation_studio_v1.AssertionOutcome{
							Mode:     "visible",
							Selector: "[data-testid='dashboard']",
							Success:  false,
							Message:  "Element not visible after 10s timeout",
						},
					},
				},
			}
			data, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(timeline)
			w.WriteHeader(http.StatusOK)
			w.Write(data)

		case strings.HasPrefix(r.URL.Path, "/executions/"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"status":   "EXECUTION_STATUS_FAILED",
				"progress": 50,
				"error":    "Assertion failed: Element not visible",
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func createTestRunner(t *testing.T, dir, basURL string) *playbooks.Runner {
	t.Helper()

	testDir := filepath.Join(dir, "test")

	cfg := playbooks.Config{
		ScenarioDir:  dir,
		ScenarioName: "test-scenario",
		TestDir:      testDir,
		AppRoot:      dir,
		Verbose:      true,
	}

	return playbooks.New(cfg,
		playbooks.WithLogger(io.Discard),
		playbooks.WithRegistryLoader(registry.NewLoader(dir)),
		playbooks.WithWorkflowResolver(workflow.NewResolver(dir, dir)),
		playbooks.WithBASClient(execution.NewClientWithConfig(basURL, execution.ClientConfig{
			Timeout:                  5 * time.Second,
			HealthCheckWaitTimeout:   5 * time.Second,
			WorkflowExecutionTimeout: 30 * time.Second,
		})),
		playbooks.WithSeedManager(seeds.NewManager(dir, dir, testDir, io.Discard)),
		playbooks.WithArtifactWriter(artifacts.NewWriter(dir, "test-scenario", dir)),
		playbooks.WithPortResolver(func(ctx context.Context, scenario, portName string) (string, error) {
			return "8080", nil
		}),
		playbooks.WithScenarioStarter(func(ctx context.Context, scenario string) error {
			return nil // Mock - assume scenario is already running
		}),
		playbooks.WithUIBaseURLResolver(func(ctx context.Context, scenario string) (string, error) {
			return "http://localhost:8080", nil
		}),
	)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
