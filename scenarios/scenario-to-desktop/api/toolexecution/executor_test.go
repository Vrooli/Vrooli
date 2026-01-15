package toolexecution

import (
	"context"
	"testing"
	"time"
)

// Mock implementations for testing

type mockPipelineOrchestrator struct {
	pipelines map[string]*PipelineStatus
}

func newMockPipelineOrchestrator() *mockPipelineOrchestrator {
	return &mockPipelineOrchestrator{
		pipelines: make(map[string]*PipelineStatus),
	}
}

func (m *mockPipelineOrchestrator) RunPipeline(ctx context.Context, config *PipelineConfig) (*PipelineStatus, error) {
	status := &PipelineStatus{
		PipelineID:   "pipeline-test-123",
		ScenarioName: config.ScenarioName,
		Status:       "running",
		CurrentStage: "bundle",
		Stages:       []StageStatus{},
		CreatedAt:    time.Now(),
	}
	m.pipelines[status.PipelineID] = status
	return status, nil
}

func (m *mockPipelineOrchestrator) ResumePipeline(ctx context.Context, pipelineID string, config *PipelineConfig) (*PipelineStatus, error) {
	newStatus := &PipelineStatus{
		PipelineID:   "pipeline-resumed-456",
		ScenarioName: "test-scenario",
		Status:       "running",
		CurrentStage: "build",
		Stages:       []StageStatus{},
		CreatedAt:    time.Now(),
	}
	m.pipelines[newStatus.PipelineID] = newStatus
	return newStatus, nil
}

func (m *mockPipelineOrchestrator) GetStatus(pipelineID string) (*PipelineStatus, bool) {
	status, ok := m.pipelines[pipelineID]
	return status, ok
}

func (m *mockPipelineOrchestrator) CancelPipeline(pipelineID string) bool {
	status, ok := m.pipelines[pipelineID]
	if !ok {
		return false
	}
	if status.Status == "running" {
		status.Status = "cancelled"
		return true
	}
	return false
}

func (m *mockPipelineOrchestrator) ListPipelines() []*PipelineStatus {
	var result []*PipelineStatus
	for _, s := range m.pipelines {
		result = append(result, s)
	}
	return result
}

type mockBuildStore struct {
	builds map[string]BuildStatus
}

func newMockBuildStore() *mockBuildStore {
	return &mockBuildStore{
		builds: make(map[string]BuildStatus),
	}
}

func (m *mockBuildStore) Get(buildID string) (BuildStatus, bool) {
	status, ok := m.builds[buildID]
	return status, ok
}

func (m *mockBuildStore) Save(status BuildStatus) {
	m.builds[status.BuildID] = status
}

func (m *mockBuildStore) Snapshot() map[string]BuildStatus {
	return m.builds
}

// Test helpers

func newTestExecutor() *ServerExecutor {
	return NewServerExecutor(ServerExecutorConfig{
		PipelineOrchestrator: newMockPipelineOrchestrator(),
		BuildStore:           newMockBuildStore(),
		VrooliRoot:           "/tmp/test",
	})
}

// Tests

func TestExecutor_Execute_UnknownTool(t *testing.T) {
	executor := newTestExecutor()

	result, err := executor.Execute(context.Background(), "unknown_tool", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Errorf("expected failure for unknown tool")
	}
	if result.Code != CodeUnknownTool {
		t.Errorf("expected code %q, got %q", CodeUnknownTool, result.Code)
	}
}

func TestExecutor_RunPipeline(t *testing.T) {
	executor := newTestExecutor()

	t.Run("success with valid args", func(t *testing.T) {
		args := map[string]interface{}{
			"scenario_name":   "test-scenario",
			"platforms":       []interface{}{"linux", "win"},
			"deployment_mode": "bundled",
		}

		result, err := executor.Execute(context.Background(), "run_pipeline", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if !result.IsAsync {
			t.Errorf("expected async result")
		}
		if result.RunID == "" {
			t.Errorf("expected RunID to be set")
		}

		data, ok := result.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected result to be map")
		}
		if data["pipeline_id"] != "pipeline-test-123" {
			t.Errorf("expected pipeline_id 'pipeline-test-123', got %v", data["pipeline_id"])
		}
		if data["scenario_name"] != "test-scenario" {
			t.Errorf("expected scenario_name 'test-scenario', got %v", data["scenario_name"])
		}
	})

	t.Run("error without scenario_name", func(t *testing.T) {
		args := map[string]interface{}{
			"platforms": []interface{}{"linux"},
		}

		result, err := executor.Execute(context.Background(), "run_pipeline", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Success {
			t.Errorf("expected failure without scenario_name")
		}
		if result.Code != CodeInvalidArgs {
			t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
		}
	})

	t.Run("parses all config options", func(t *testing.T) {
		executor := newTestExecutor()
		args := map[string]interface{}{
			"scenario_name":        "test-scenario",
			"platforms":            []interface{}{"linux"},
			"deployment_mode":      "proxy",
			"template_type":        "advanced",
			"stop_after_stage":     "generate",
			"skip_preflight":       true,
			"skip_smoke_test":      true,
			"distribute":           true,
			"distribution_targets": []interface{}{"s3"},
			"sign":                 true,
			"clean":                true,
			"version":              "1.0.0",
			"proxy_url":            "https://example.com",
		}

		result, err := executor.Execute(context.Background(), "run_pipeline", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
	})
}

func TestExecutor_CheckPipelineStatus(t *testing.T) {
	executor := newTestExecutor()

	// First create a pipeline
	runArgs := map[string]interface{}{
		"scenario_name": "test-scenario",
	}
	_, err := executor.Execute(context.Background(), "run_pipeline", runArgs)
	if err != nil {
		t.Fatalf("unexpected error creating pipeline: %v", err)
	}

	t.Run("success with valid pipeline_id", func(t *testing.T) {
		args := map[string]interface{}{
			"pipeline_id": "pipeline-test-123",
		}

		result, err := executor.Execute(context.Background(), "check_pipeline_status", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if result.IsAsync {
			t.Errorf("check_pipeline_status should not be async")
		}

		data, ok := result.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected result to be map")
		}
		if data["pipeline_id"] != "pipeline-test-123" {
			t.Errorf("expected pipeline_id 'pipeline-test-123'")
		}
		if data["status"] != "running" {
			t.Errorf("expected status 'running', got %v", data["status"])
		}
	})

	t.Run("error without pipeline_id", func(t *testing.T) {
		result, err := executor.Execute(context.Background(), "check_pipeline_status", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Success {
			t.Errorf("expected failure without pipeline_id")
		}
		if result.Code != CodeInvalidArgs {
			t.Errorf("expected code %q, got %q", CodeInvalidArgs, result.Code)
		}
	})

	t.Run("error with nonexistent pipeline", func(t *testing.T) {
		args := map[string]interface{}{
			"pipeline_id": "nonexistent-pipeline",
		}

		result, err := executor.Execute(context.Background(), "check_pipeline_status", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Success {
			t.Errorf("expected failure for nonexistent pipeline")
		}
		if result.Code != CodeNotFound {
			t.Errorf("expected code %q, got %q", CodeNotFound, result.Code)
		}
	})
}

func TestExecutor_CancelPipeline(t *testing.T) {
	executor := newTestExecutor()

	// First create a pipeline
	runArgs := map[string]interface{}{
		"scenario_name": "test-scenario",
	}
	_, err := executor.Execute(context.Background(), "run_pipeline", runArgs)
	if err != nil {
		t.Fatalf("unexpected error creating pipeline: %v", err)
	}

	t.Run("success cancelling running pipeline", func(t *testing.T) {
		args := map[string]interface{}{
			"pipeline_id": "pipeline-test-123",
		}

		result, err := executor.Execute(context.Background(), "cancel_pipeline", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}

		data, ok := result.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected result to be map")
		}
		if data["status"] != "cancelled" {
			t.Errorf("expected status 'cancelled', got %v", data["status"])
		}
	})

	t.Run("error without pipeline_id", func(t *testing.T) {
		result, err := executor.Execute(context.Background(), "cancel_pipeline", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Success {
			t.Errorf("expected failure without pipeline_id")
		}
	})

	t.Run("error with nonexistent pipeline", func(t *testing.T) {
		args := map[string]interface{}{
			"pipeline_id": "nonexistent-pipeline",
		}

		result, err := executor.Execute(context.Background(), "cancel_pipeline", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Success {
			t.Errorf("expected failure for nonexistent pipeline")
		}
		if result.Code != CodeNotFound {
			t.Errorf("expected code %q, got %q", CodeNotFound, result.Code)
		}
	})
}

func TestExecutor_ResumePipeline(t *testing.T) {
	executor := newTestExecutor()

	// First create a pipeline
	runArgs := map[string]interface{}{
		"scenario_name": "test-scenario",
	}
	_, err := executor.Execute(context.Background(), "run_pipeline", runArgs)
	if err != nil {
		t.Fatalf("unexpected error creating pipeline: %v", err)
	}

	t.Run("success resuming pipeline", func(t *testing.T) {
		args := map[string]interface{}{
			"pipeline_id": "pipeline-test-123",
		}

		result, err := executor.Execute(context.Background(), "resume_pipeline", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
		if !result.IsAsync {
			t.Errorf("resume_pipeline should be async")
		}

		data, ok := result.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected result to be map")
		}
		if data["parent_pipeline_id"] != "pipeline-test-123" {
			t.Errorf("expected parent_pipeline_id 'pipeline-test-123'")
		}
	})

	t.Run("with stop_after_stage option", func(t *testing.T) {
		args := map[string]interface{}{
			"pipeline_id":      "pipeline-test-123",
			"stop_after_stage": "build",
		}

		result, err := executor.Execute(context.Background(), "resume_pipeline", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}
	})

	t.Run("error without pipeline_id", func(t *testing.T) {
		result, err := executor.Execute(context.Background(), "resume_pipeline", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Success {
			t.Errorf("expected failure without pipeline_id")
		}
	})

	t.Run("error with nonexistent pipeline", func(t *testing.T) {
		args := map[string]interface{}{
			"pipeline_id": "nonexistent-pipeline",
		}

		result, err := executor.Execute(context.Background(), "resume_pipeline", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Success {
			t.Errorf("expected failure for nonexistent pipeline")
		}
		if result.Code != CodeNotFound {
			t.Errorf("expected code %q, got %q", CodeNotFound, result.Code)
		}
	})
}

func TestExecutor_ListPipelines(t *testing.T) {
	executor := newTestExecutor()

	// Create some pipelines
	for _, scenario := range []string{"scenario-a", "scenario-b", "scenario-c"} {
		args := map[string]interface{}{
			"scenario_name": scenario,
		}
		executor.Execute(context.Background(), "run_pipeline", args)
	}

	t.Run("lists all pipelines", func(t *testing.T) {
		result, err := executor.Execute(context.Background(), "list_pipelines", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}

		data, ok := result.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected result to be map")
		}
		if _, ok := data["pipelines"]; !ok {
			t.Errorf("expected 'pipelines' in result")
		}
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		args := map[string]interface{}{
			"limit": 1,
		}

		result, err := executor.Execute(context.Background(), "list_pipelines", args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Errorf("expected success, got error: %s", result.Error)
		}

		data, ok := result.Result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected result to be map")
		}
		count := data["count"].(int)
		if count > 1 {
			t.Errorf("expected at most 1 pipeline, got %d", count)
		}
	})
}

func TestExecutor_NilOrchestrator(t *testing.T) {
	// Create executor without pipeline orchestrator
	executor := NewServerExecutor(ServerExecutorConfig{
		VrooliRoot: "/tmp/test",
	})

	tools := []string{
		"run_pipeline",
		"check_pipeline_status",
		"cancel_pipeline",
		"resume_pipeline",
		"list_pipelines",
	}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			args := map[string]interface{}{
				"scenario_name": "test",
				"pipeline_id":   "test-id",
			}

			result, err := executor.Execute(context.Background(), tool, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Success {
				t.Errorf("expected failure when orchestrator is nil")
			}
			if result.Code != CodeInternalError {
				t.Errorf("expected code %q, got %q", CodeInternalError, result.Code)
			}
		})
	}
}

// Argument helper tests

func TestGetStringArg(t *testing.T) {
	args := map[string]interface{}{
		"present": "value",
		"number":  42,
	}

	t.Run("returns value when present", func(t *testing.T) {
		if got := getStringArg(args, "present", "default"); got != "value" {
			t.Errorf("expected 'value', got %q", got)
		}
	})

	t.Run("returns default when missing", func(t *testing.T) {
		if got := getStringArg(args, "missing", "default"); got != "default" {
			t.Errorf("expected 'default', got %q", got)
		}
	})

	t.Run("returns default for non-string", func(t *testing.T) {
		if got := getStringArg(args, "number", "default"); got != "default" {
			t.Errorf("expected 'default', got %q", got)
		}
	})
}

func TestGetStringArrayArg(t *testing.T) {
	args := map[string]interface{}{
		"interface_array": []interface{}{"a", "b", "c"},
		"string_array":    []string{"x", "y"},
		"mixed":           []interface{}{"a", 1, "b"},
		"number":          42,
	}

	t.Run("returns interface array as strings", func(t *testing.T) {
		result := getStringArrayArg(args, "interface_array")
		if len(result) != 3 || result[0] != "a" {
			t.Errorf("unexpected result: %v", result)
		}
	})

	t.Run("returns string array", func(t *testing.T) {
		result := getStringArrayArg(args, "string_array")
		if len(result) != 2 || result[0] != "x" {
			t.Errorf("unexpected result: %v", result)
		}
	})

	t.Run("filters non-strings from mixed array", func(t *testing.T) {
		result := getStringArrayArg(args, "mixed")
		if len(result) != 2 {
			t.Errorf("expected 2 strings, got %d", len(result))
		}
	})

	t.Run("returns nil for missing", func(t *testing.T) {
		result := getStringArrayArg(args, "missing")
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestGetBoolArg(t *testing.T) {
	args := map[string]interface{}{
		"true_val":  true,
		"false_val": false,
		"string":    "true",
	}

	t.Run("returns true value", func(t *testing.T) {
		if got := getBoolArg(args, "true_val", false); !got {
			t.Errorf("expected true")
		}
	})

	t.Run("returns false value", func(t *testing.T) {
		if got := getBoolArg(args, "false_val", true); got {
			t.Errorf("expected false")
		}
	})

	t.Run("returns default for missing", func(t *testing.T) {
		if got := getBoolArg(args, "missing", true); !got {
			t.Errorf("expected default true")
		}
	})

	t.Run("returns default for non-bool", func(t *testing.T) {
		if got := getBoolArg(args, "string", false); got {
			t.Errorf("expected default false")
		}
	})
}

func TestGetIntArg(t *testing.T) {
	args := map[string]interface{}{
		"float":  float64(42),
		"int":    10,
		"string": "not a number",
	}

	t.Run("returns float as int", func(t *testing.T) {
		if got := getIntArg(args, "float", 0); got != 42 {
			t.Errorf("expected 42, got %d", got)
		}
	})

	t.Run("returns int", func(t *testing.T) {
		if got := getIntArg(args, "int", 0); got != 10 {
			t.Errorf("expected 10, got %d", got)
		}
	})

	t.Run("returns default for missing", func(t *testing.T) {
		if got := getIntArg(args, "missing", 99); got != 99 {
			t.Errorf("expected 99, got %d", got)
		}
	})

	t.Run("returns default for non-number", func(t *testing.T) {
		if got := getIntArg(args, "string", 50); got != 50 {
			t.Errorf("expected 50, got %d", got)
		}
	})
}

// ExecutionResult helper tests

func TestSuccessResult(t *testing.T) {
	result := SuccessResult(map[string]string{"key": "value"})

	if !result.Success {
		t.Errorf("expected Success to be true")
	}
	if result.IsAsync {
		t.Errorf("expected IsAsync to be false")
	}
	if result.Error != "" {
		t.Errorf("expected no error")
	}
}

func TestAsyncResult(t *testing.T) {
	result := AsyncResult(map[string]string{"key": "value"}, "run-123")

	if !result.Success {
		t.Errorf("expected Success to be true")
	}
	if !result.IsAsync {
		t.Errorf("expected IsAsync to be true")
	}
	if result.RunID != "run-123" {
		t.Errorf("expected RunID 'run-123', got %q", result.RunID)
	}
	if result.Status != StatusPending {
		t.Errorf("expected Status 'pending', got %q", result.Status)
	}
}

func TestErrorResult(t *testing.T) {
	result := ErrorResult("something went wrong", CodeInternalError)

	if result.Success {
		t.Errorf("expected Success to be false")
	}
	if result.Error != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got %q", result.Error)
	}
	if result.Code != CodeInternalError {
		t.Errorf("expected code %q, got %q", CodeInternalError, result.Code)
	}
}
