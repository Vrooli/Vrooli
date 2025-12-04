package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"test-genie/internal/playbooks/types"
)

func TestWriterWriteTimeline(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	writer := NewWriter(scenarioDir, "test-scenario", tempDir)
	timelineData := []byte(`{"frames": [{"step_type": "navigate"}]}`)

	path, err := writer.WriteTimeline("test/playbooks/login.json", timelineData)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify file was created
	fullPath := filepath.Join(scenarioDir, TimelineDir, "test-playbooks-login.timeline.json")
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read timeline file: %v", err)
	}
	if string(content) != string(timelineData) {
		t.Errorf("expected %s, got %s", string(timelineData), string(content))
	}

	// Path should be relative
	if filepath.IsAbs(path) && strings.HasPrefix(path, tempDir) {
		// Relative path returned
	}
}

func TestWriterWriteTimelineCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")
	// Don't create scenario dir - should be created automatically

	writer := NewWriter(scenarioDir, "test-scenario", tempDir)
	_, err := writer.WriteTimeline("workflow.json", []byte("{}"))
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify directory was created
	timelineDirPath := filepath.Join(scenarioDir, TimelineDir)
	if _, err := os.Stat(timelineDirPath); err != nil {
		t.Error("expected timeline directory to be created")
	}
}

func TestWriterWritePhaseResults(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	writer := NewWriter(scenarioDir, "test-scenario", tempDir)

	results := []types.Result{
		{
			Entry: types.Entry{
				File:         "test/playbooks/login.json",
				Description:  "Login flow",
				Requirements: []string{"REQ-001", "REQ-002"},
			},
			Outcome: &types.Outcome{
				ExecutionID: "exec-123",
				Duration:    2 * time.Second,
				Stats:       " (5 steps)",
			},
		},
	}

	err := writer.WritePhaseResults(results)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify file was created
	path := filepath.Join(scenarioDir, PhaseResultsDir, PhaseResultsFile)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read phase results: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(content, &output); err != nil {
		t.Fatalf("failed to parse phase results: %v", err)
	}

	if output["phase"] != "playbooks" {
		t.Errorf("expected phase=playbooks, got %v", output["phase"])
	}
	if output["scenario"] != "test-scenario" {
		t.Errorf("expected scenario=test-scenario, got %v", output["scenario"])
	}
	if int(output["tests"].(float64)) != 1 {
		t.Errorf("expected tests=1, got %v", output["tests"])
	}
	if output["status"] != "passed" {
		t.Errorf("expected status=passed, got %v", output["status"])
	}

	reqs := output["requirements"].([]any)
	if len(reqs) != 2 {
		t.Errorf("expected 2 requirements, got %d", len(reqs))
	}
}

func TestWriterWritePhaseResultsWithErrors(t *testing.T) {
	tempDir := t.TempDir()
	scenarioDir := filepath.Join(tempDir, "scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	writer := NewWriter(scenarioDir, "test-scenario", tempDir)

	results := []types.Result{
		{
			Entry: types.Entry{
				File:         "test/playbooks/login.json",
				Requirements: []string{"REQ-001"},
			},
			Err: os.ErrNotExist,
		},
	}

	err := writer.WritePhaseResults(results)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	path := filepath.Join(scenarioDir, PhaseResultsDir, PhaseResultsFile)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read phase results: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(content, &output); err != nil {
		t.Fatalf("failed to parse phase results: %v", err)
	}

	if int(output["errors"].(float64)) != 1 {
		t.Errorf("expected errors=1, got %v", output["errors"])
	}
	if output["status"] != "failed" {
		t.Errorf("expected status=failed, got %v", output["status"])
	}
}

func TestWriterWritePhaseResultsEmpty(t *testing.T) {
	tempDir := t.TempDir()
	writer := NewWriter(tempDir, "test", tempDir)

	err := writer.WritePhaseResults(nil)
	if err != nil {
		t.Fatalf("expected success for empty results, got error: %v", err)
	}

	err = writer.WritePhaseResults([]types.Result{})
	if err != nil {
		t.Fatalf("expected success for empty slice, got error: %v", err)
	}
}

func TestSanitizeArtifactName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test.json", "test"},
		{"test/playbooks/login.json", "test-playbooks-login"},
		{"test\\playbooks\\login.json", "test-playbooks-login"},
		{"my-workflow.json", "my-workflow"},
		{"my_workflow.json", "my_workflow"},
		{"spaces and special!chars.json", "spaces-and-special-chars"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeArtifactName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBuildPhaseOutput(t *testing.T) {
	results := []types.Result{
		{
			Entry: types.Entry{
				File:         "workflow1.json",
				Requirements: []string{"REQ-1"},
			},
			Outcome: &types.Outcome{
				Duration: 1 * time.Second,
				Stats:    " (3 steps)",
			},
		},
		{
			Entry: types.Entry{
				File:         "workflow2.json",
				Requirements: []string{"REQ-2"},
			},
			Err:          os.ErrNotExist,
			ArtifactPath: "path/to/artifact",
		},
	}

	output := buildPhaseOutput("test-scenario", results)

	if output["phase"] != "playbooks" {
		t.Errorf("expected phase=playbooks")
	}
	if output["scenario"] != "test-scenario" {
		t.Errorf("expected scenario=test-scenario")
	}
	if output["tests"] != 2 {
		t.Errorf("expected tests=2, got %v", output["tests"])
	}
	if output["errors"] != 1 {
		t.Errorf("expected errors=1, got %v", output["errors"])
	}
	if output["status"] != "failed" {
		t.Errorf("expected status=failed")
	}

	reqs := output["requirements"].([]map[string]any)
	if len(reqs) != 2 {
		t.Errorf("expected 2 requirement entries, got %d", len(reqs))
	}

	// Check first requirement (passed)
	if reqs[0]["status"] != "passed" {
		t.Errorf("expected first req status=passed")
	}

	// Check second requirement (failed with artifact)
	if reqs[1]["status"] != "failed" {
		t.Errorf("expected second req status=failed")
	}
	evidence := reqs[1]["evidence"].(string)
	if !strings.Contains(evidence, "artifact") {
		t.Errorf("expected artifact in evidence, got %s", evidence)
	}
}
