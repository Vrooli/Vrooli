package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"test-genie/internal/playbooks/execution"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

func TestWriteWorkflowArtifacts_Basic(t *testing.T) {
	tempDir := t.TempDir()
	writer := NewWriter(tempDir, "test-scenario", tempDir)

	timelineData := []byte(`{"frames":[],"logs":[]}`)
	parsed := &execution.ParsedTimeline{
		ExecutionID: "exec-123",
		Status:      "completed",
		Summary: execution.TimelineSummary{
			TotalSteps:   2,
			TotalAsserts: 1,
			AssertsPassed: 1,
		},
	}
	result := &WorkflowResult{
		WorkflowFile: "test/playbooks/smoke.json",
		Description:  "Smoke test",
		ExecutionID:  "exec-123",
		Success:      true,
		Status:       "passed",
		DurationMs:   1500,
		Timestamp:    time.Now().UTC(),
		Summary:      parsed.Summary,
	}

	artifacts, err := writer.WriteWorkflowArtifacts(
		"test/playbooks/smoke.json",
		timelineData,
		parsed,
		nil, // no screenshots
		result,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check artifacts dir was created
	if artifacts.Dir == "" {
		t.Error("expected Dir to be set")
	}

	// Check timeline was written
	if artifacts.Timeline == "" {
		t.Error("expected Timeline path to be set")
	}
	timelinePath := filepath.Join(tempDir, artifacts.Timeline)
	if _, err := os.Stat(timelinePath); os.IsNotExist(err) {
		t.Errorf("timeline file not created at %s", timelinePath)
	}

	// Check README was written
	if artifacts.Readme == "" {
		t.Error("expected Readme path to be set")
	}

	// Check latest.json was written
	if artifacts.Latest == "" {
		t.Error("expected Latest path to be set")
	}
}

func TestWriteWorkflowArtifacts_WithScreenshots(t *testing.T) {
	tempDir := t.TempDir()
	writer := NewWriter(tempDir, "test-scenario", tempDir)

	parsed := &execution.ParsedTimeline{
		ExecutionID: "exec-123",
		Status:      "completed",
	}
	result := &WorkflowResult{
		WorkflowFile: "test/playbooks/with-screenshots.json",
		Success:      true,
		Status:       "passed",
		Timestamp:    time.Now().UTC(),
	}

	// Simulate downloaded screenshots
	screenshots := []ScreenshotData{
		{StepIndex: 0, StepName: "navigate", Filename: "step-00-navigate.png", Data: []byte("fake-png-1")},
		{StepIndex: 1, StepName: "click", Filename: "step-01-click.png", Data: []byte("fake-png-2")},
	}

	artifacts, err := writer.WriteWorkflowArtifacts(
		"test/playbooks/with-screenshots.json",
		[]byte(`{}`),
		parsed,
		screenshots,
		result,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check screenshots were written
	if len(artifacts.Screenshots) != 2 {
		t.Errorf("expected 2 screenshots, got %d", len(artifacts.Screenshots))
	}

	// Verify screenshot files exist
	for _, ssPath := range artifacts.Screenshots {
		fullPath := filepath.Join(tempDir, ssPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("screenshot file not created at %s", fullPath)
		}
	}
}

func TestWriteWorkflowArtifacts_WithLogs(t *testing.T) {
	tempDir := t.TempDir()
	writer := NewWriter(tempDir, "test-scenario", tempDir)

	parsed := &execution.ParsedTimeline{
		ExecutionID: "exec-123",
		Status:      "completed",
		Logs: []execution.ParsedLog{
			{ID: "1", Level: "error", Message: "Test error"},
			{ID: "2", Level: "warn", Message: "Test warning"},
		},
	}
	result := &WorkflowResult{
		WorkflowFile: "test/playbooks/with-logs.json",
		Success:      true,
		Status:       "passed",
		Timestamp:    time.Now().UTC(),
	}

	artifacts, err := writer.WriteWorkflowArtifacts(
		"test/playbooks/with-logs.json",
		[]byte(`{}`),
		parsed,
		nil,
		result,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check console.json was written
	if artifacts.Console == "" {
		t.Error("expected Console path to be set")
	}

	// Read and verify console.json content
	consolePath := filepath.Join(tempDir, artifacts.Console)
	content, err := os.ReadFile(consolePath)
	if err != nil {
		t.Fatalf("failed to read console.json: %v", err)
	}

	var logs []execution.ParsedLog
	if err := json.Unmarshal(content, &logs); err != nil {
		t.Fatalf("failed to parse console.json: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs in console.json, got %d", len(logs))
	}
}

func TestWriteWorkflowArtifacts_WithAssertions(t *testing.T) {
	tempDir := t.TempDir()
	writer := NewWriter(tempDir, "test-scenario", tempDir)

	parsed := &execution.ParsedTimeline{
		ExecutionID: "exec-123",
		Status:      "failed",
		Assertions: []execution.ParsedAssertion{
			{StepIndex: 0, Passed: true, AssertionType: "visible"},
			{StepIndex: 1, Passed: false, AssertionType: "text", Expected: "Hello", Actual: "World"},
		},
	}
	result := &WorkflowResult{
		WorkflowFile: "test/playbooks/with-assertions.json",
		Success:      false,
		Status:       "failed",
		Error:        "Assertion failed",
		Timestamp:    time.Now().UTC(),
	}

	artifacts, err := writer.WriteWorkflowArtifacts(
		"test/playbooks/with-assertions.json",
		[]byte(`{}`),
		parsed,
		nil,
		result,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check assertions.json was written
	if artifacts.Assertions == "" {
		t.Error("expected Assertions path to be set")
	}

	// Verify content
	assertPath := filepath.Join(tempDir, artifacts.Assertions)
	content, err := os.ReadFile(assertPath)
	if err != nil {
		t.Fatalf("failed to read assertions.json: %v", err)
	}

	var assertions []execution.ParsedAssertion
	if err := json.Unmarshal(content, &assertions); err != nil {
		t.Fatalf("failed to parse assertions.json: %v", err)
	}
	if len(assertions) != 2 {
		t.Errorf("expected 2 assertions, got %d", len(assertions))
	}
}

func TestWriteWorkflowArtifacts_WithDOM(t *testing.T) {
	tempDir := t.TempDir()
	writer := NewWriter(tempDir, "test-scenario", tempDir)

	parsed := &execution.ParsedTimeline{
		ExecutionID: "exec-123",
		Status:      "completed",
		FinalDOM:    "<html><body>Test DOM</body></html>",
	}
	result := &WorkflowResult{
		WorkflowFile: "test/playbooks/with-dom.json",
		Success:      true,
		Status:       "passed",
		Timestamp:    time.Now().UTC(),
	}

	artifacts, err := writer.WriteWorkflowArtifacts(
		"test/playbooks/with-dom.json",
		[]byte(`{}`),
		parsed,
		nil,
		result,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check dom.html was written
	if artifacts.DOM == "" {
		t.Error("expected DOM path to be set")
	}

	// Verify content
	domPath := filepath.Join(tempDir, artifacts.DOM)
	content, err := os.ReadFile(domPath)
	if err != nil {
		t.Fatalf("failed to read dom.html: %v", err)
	}
	if string(content) != "<html><body>Test DOM</body></html>" {
		t.Errorf("unexpected DOM content: %s", string(content))
	}
}

func TestExtractScreenshotsFromTimeline(t *testing.T) {
	parsed := &execution.ParsedTimeline{
		Frames: []execution.ParsedFrame{
			{
				StepIndex: 0,
				StepType:  "navigate",
				Screenshot: &execution.FrameScreenshot{
					ArtifactID: "ss-1",
					URL:        "/api/v1/screenshots/ss-1",
				},
			},
			{
				StepIndex:  1,
				StepType:   "click",
				Screenshot: nil, // No screenshot for this step
			},
			{
				StepIndex: 2,
				StepType:  "assert",
				Screenshot: &execution.FrameScreenshot{
					ArtifactID: "ss-2",
					URL:        "/api/v1/screenshots/ss-2",
				},
			},
		},
	}

	refs := ExtractScreenshotsFromTimeline(parsed)
	if len(refs) != 2 {
		t.Fatalf("expected 2 screenshot refs, got %d", len(refs))
	}
	if refs[0].StepIndex != 0 {
		t.Errorf("expected first ref step_index 0, got %d", refs[0].StepIndex)
	}
	if refs[1].StepIndex != 2 {
		t.Errorf("expected second ref step_index 2, got %d", refs[1].StepIndex)
	}
}

func TestGenerateScreenshotFilename(t *testing.T) {
	tests := []struct {
		ref      ScreenshotRef
		expected string
	}{
		{ScreenshotRef{StepIndex: 0, StepType: "navigate"}, "step-00-navigate.png"},
		{ScreenshotRef{StepIndex: 1, StepType: "click"}, "step-01-click.png"},
		{ScreenshotRef{StepIndex: 10, StepType: "assert"}, "step-10-assert.png"},
		{ScreenshotRef{StepIndex: 5, StepType: ""}, "step-05-step.png"},
	}

	for _, tc := range tests {
		result := GenerateScreenshotFilename(tc.ref)
		if result != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, result)
		}
	}
}

func TestGenerateWorkflowReadme_Success(t *testing.T) {
	result := &WorkflowResult{
		WorkflowFile: "test/playbooks/smoke.json",
		Description:  "Smoke test workflow",
		Requirements: []string{"REQ-001", "REQ-002"},
		ExecutionID:  "exec-123",
		Success:      true,
		Status:       "passed",
		DurationMs:   1500,
		Timestamp:    time.Now().UTC(),
		Summary: execution.TimelineSummary{
			TotalSteps:    3,
			TotalAsserts:  2,
			AssertsPassed: 2,
		},
		Artifacts: WorkflowArtifacts{
			Timeline:    "coverage/automation/smoke/timeline.json",
			Screenshots: []string{"screenshots/step-00-navigate.png"},
		},
	}

	readme := GenerateWorkflowReadme(result)

	// Check for key content
	if !strings.Contains(readme, "✅ Passed") {
		t.Error("expected success status in README")
	}
	if !strings.Contains(readme, "smoke.json") {
		t.Error("expected workflow name in README")
	}
	if !strings.Contains(readme, "REQ-001") {
		t.Error("expected requirement ID in README")
	}
	if !strings.Contains(readme, "2/2 passed") {
		t.Error("expected assertion stats in README")
	}
	if !strings.Contains(readme, "timeline.json") {
		t.Error("expected timeline link in README")
	}
}

func TestGenerateWorkflowReadme_Failure(t *testing.T) {
	result := &WorkflowResult{
		WorkflowFile: "test/playbooks/failing.json",
		Success:      false,
		Status:       "failed",
		Error:        "Element not found: [data-testid='submit']",
		Timestamp:    time.Now().UTC(),
		ParsedSummary: &execution.ParsedTimeline{
			FailedFrame: &execution.ParsedFrame{
				StepIndex: 2,
				StepType:  "click",
				NodeID:    "click-submit",
				Error:     "Element not found",
			},
		},
	}

	readme := GenerateWorkflowReadme(result)

	// Check for failure content
	if !strings.Contains(readme, "❌ Failed") {
		t.Error("expected failure status in README")
	}
	if !strings.Contains(readme, "Error Details") {
		t.Error("expected error details section in README")
	}
	if !strings.Contains(readme, "Troubleshooting") {
		t.Error("expected troubleshooting section in README")
	}
	if !strings.Contains(readme, "click") {
		t.Error("expected click step type in troubleshooting hints")
	}
}

func TestWorkflowDir_Sanitization(t *testing.T) {
	tempDir := t.TempDir()
	writer := NewWriter(tempDir, "test-scenario", tempDir)

	tests := []string{
		"test/playbooks/simple.json",
		"test/playbooks/with spaces.json",
		"test/playbooks/special!chars.json",
		"deeply/nested/path/workflow.json",
	}

	for _, workflowFile := range tests {
		dir := writer.workflowDir(workflowFile)
		// Should not contain any invalid path characters
		if strings.ContainsAny(filepath.Base(dir), "!@#$%^&*(){}[]|\\:\"'<>?,") {
			t.Errorf("workflow dir contains invalid chars: %s", dir)
		}
		// Should be under AutomationDir
		if !strings.Contains(dir, sharedartifacts.AutomationDir) {
			t.Errorf("workflow dir not under automation dir: %s", dir)
		}
	}
}
