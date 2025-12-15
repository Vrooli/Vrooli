//go:build legacyproto
// +build legacyproto

package execution_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"test-genie/internal/playbooks/execution"
	"test-genie/internal/playbooks/types"

	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
)

// These are contract tests that verify the BAS API behaves as expected.
// They require a running BAS instance and are skipped if BAS is unavailable.
//
// To run these tests:
//   1. Start browser-automation-studio: make -C scenarios/browser-automation-studio start
//   2. Set BAS_API_URL environment variable (default: http://localhost:8080/api/v1)
//   3. Run: go test -v ./internal/playbooks/execution/... -run Contract

const (
	defaultBASURL = "http://localhost:8080/api/v1"
	testTimeout   = 30 * time.Second
)

func getBASURL() string {
	if url := os.Getenv("BAS_API_URL"); url != "" {
		return url
	}
	return defaultBASURL
}

func skipIfBASUnavailable(t *testing.T) *execution.HTTPClient {
	t.Helper()

	client := execution.NewClient(getBASURL())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Health(ctx); err != nil {
		t.Skipf("BAS not available at %s: %v (set BAS_API_URL to override)", getBASURL(), err)
	}

	return client
}

// TestContractHealth verifies the health endpoint contract.
func TestContractHealth(t *testing.T) {
	client := skipIfBASUnavailable(t)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	err := client.Health(ctx)
	if err != nil {
		t.Errorf("Health() error = %v, want nil", err)
	}
}

// TestContractHealthResponse verifies the health endpoint response format.
func TestContractHealthResponse(t *testing.T) {
	_ = skipIfBASUnavailable(t)

	resp, err := http.Get(getBASURL() + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Health endpoint should return JSON (may be empty or have status field)
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && contentType != "application/json" {
		t.Logf("Note: /health Content-Type = %q (expected application/json or empty)", contentType)
	}
}

// TestContractExecuteWorkflowMinimal verifies workflow execution with minimal definition.
func TestContractExecuteWorkflowMinimal(t *testing.T) {
	client := skipIfBASUnavailable(t)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Minimal valid workflow: single navigate node
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "start",
				"type": "navigate",
				"data": map[string]any{
					"label":           "Navigate to test page",
					"destinationType": "url",
					"url":             "about:blank",
				},
			},
		},
		"edges": []any{},
	}

	executionID, err := client.ExecuteWorkflow(ctx, definition, "contract-test-minimal")
	if err != nil {
		t.Fatalf("ExecuteWorkflow() error = %v", err)
	}

	if executionID == "" {
		t.Error("ExecuteWorkflow() returned empty execution ID")
	}

	t.Logf("Created execution: %s", executionID)

	// Verify we can get status
	status, err := client.GetStatus(ctx, executionID)
	if err != nil {
		t.Errorf("GetStatus(%s) error = %v", executionID, err)
	}

	t.Logf("Execution status: %+v", status)

	// Status should have a non-empty status field
	if status == nil || status.GetStatus() == browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED {
		t.Error("GetStatus() returned empty status")
	}
}

// TestContractExecuteAndWait verifies full workflow execution cycle.
func TestContractExecuteAndWait(t *testing.T) {
	client := skipIfBASUnavailable(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Simple workflow that should complete quickly
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "wait-node",
				"type": "wait",
				"data": map[string]any{
					"label":      "Wait briefly",
					"waitType":   "duration",
					"durationMs": 100,
				},
			},
		},
		"edges": []any{},
	}

	executionID, err := client.ExecuteWorkflow(ctx, definition, "contract-test-wait")
	if err != nil {
		t.Fatalf("ExecuteWorkflow() error = %v", err)
	}

	t.Logf("Created execution: %s", executionID)

	// Wait for completion with progress tracking
	progressCount := 0
	err = client.WaitForCompletionWithProgress(ctx, executionID, func(status *types.ExecutionStatus, elapsed time.Duration) error {
		if status == nil {
			return nil
		}
		progressCount++
		t.Logf("Progress update %d: status=%s, elapsed=%s", progressCount, status.GetStatus(), elapsed)
		return nil
	})

	if err != nil {
		t.Errorf("WaitForCompletionWithProgress() error = %v", err)
	}

	// Get final status
	finalStatus, err := client.GetStatus(ctx, executionID)
	if err != nil {
		t.Errorf("Final GetStatus() error = %v", err)
	}

	t.Logf("Final status: %+v", finalStatus)
}

// TestContractGetTimeline verifies timeline retrieval.
func TestContractGetTimeline(t *testing.T) {
	client := skipIfBASUnavailable(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Execute a simple workflow first
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "wait-node",
				"type": "wait",
				"data": map[string]any{
					"label":      "Wait",
					"waitType":   "duration",
					"durationMs": 100,
				},
			},
		},
		"edges": []any{},
	}

	executionID, err := client.ExecuteWorkflow(ctx, definition, "contract-test-timeline")
	if err != nil {
		t.Fatalf("ExecuteWorkflow() error = %v", err)
	}

	// Wait for completion
	if err := client.WaitForCompletion(ctx, executionID); err != nil {
		t.Fatalf("WaitForCompletion() error = %v", err)
	}

	// Get timeline
	timeline, timelineData, err := client.GetTimeline(ctx, executionID)
	if err != nil {
		t.Fatalf("GetTimeline() error = %v", err)
	}

	if len(timelineData) == 0 {
		t.Error("GetTimeline() returned empty data")
		return
	}

	if timeline == nil {
		t.Log("warning: parsed proto timeline is nil")
	}

	t.Logf("Timeline data length: %d bytes", len(timelineData))

	// Verify timeline is valid JSON
	var timelineJSON map[string]any
	if err := json.Unmarshal(timelineData, &timelineJSON); err != nil {
		t.Errorf("Timeline is not valid JSON: %v", err)
		t.Logf("Raw timeline (first 500 bytes): %s", truncate(string(timelineData), 500))
		return
	}

	// Try to parse with our ParseTimeline function
	summary, parseErr := execution.ParseTimeline(timelineData)
	if parseErr != nil {
		t.Logf("ParseTimeline returned error (may be expected if format differs): %v", parseErr)
	} else {
		t.Logf("Timeline summary: %s", summary.String())
	}
}

// TestContractStatusFields verifies that status response has expected fields.
func TestContractStatusFields(t *testing.T) {
	client := skipIfBASUnavailable(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Execute workflow
	definition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "wait",
				"type": "wait",
				"data": map[string]any{
					"label":      "Wait for status test",
					"waitType":   "duration",
					"durationMs": 500,
				},
			},
		},
		"edges": []any{},
	}

	executionID, err := client.ExecuteWorkflow(ctx, definition, "contract-test-status-fields")
	if err != nil {
		t.Fatalf("ExecuteWorkflow() error = %v", err)
	}

	// Get status while running
	time.Sleep(100 * time.Millisecond) // Give it time to start
	status, err := client.GetStatus(ctx, executionID)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("GetStatus() returned nil status")
	}

	t.Logf("Raw status: %+v", status)

	// Document which fields are actually populated by BAS
	t.Logf("Field availability:")
	t.Logf("  Status: %q", status.GetStatus())
	t.Logf("  Progress: %v", status.GetProgress())
	t.Logf("  CurrentStep: %q", status.GetCurrentStep())
	t.Logf("  TriggerType: %q", status.GetTriggerType())
	t.Logf("  WorkflowId: %q", status.GetWorkflowId())
	t.Logf("  WorkflowVersion: %d", status.GetWorkflowVersion())
	if status.Error != nil {
		t.Logf("  Error: %q", status.GetError())
	}

	// Wait for completion
	_ = client.WaitForCompletion(ctx, executionID)
}

// TestContractInvalidWorkflow verifies error handling for invalid workflows.
func TestContractInvalidWorkflow(t *testing.T) {
	client := skipIfBASUnavailable(t)
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Workflow with no nodes
	definition := map[string]any{
		"nodes": []any{},
		"edges": []any{},
	}

	_, err := client.ExecuteWorkflow(ctx, definition, "contract-test-invalid")

	// BAS might accept this (empty workflow completes immediately) or reject it
	// Either way, document the behavior
	if err != nil {
		t.Logf("Empty workflow rejected: %v (this is acceptable)", err)
	} else {
		t.Log("Empty workflow accepted (BAS allows empty workflows)")
	}
}

// TestContractExecuteWorkflowPayload verifies the exact request payload format.
func TestContractExecuteWorkflowPayload(t *testing.T) {
	_ = skipIfBASUnavailable(t)

	// This test documents the expected request format
	// If BAS changes its API, this test helps identify the mismatch

	expectedPayload := map[string]any{
		"flow_definition": map[string]any{
			"nodes": []any{},
			"edges": []any{},
		},
		"parameters":          map[string]any{},
		"wait_for_completion": false,
		"metadata": map[string]any{
			"name": "test-workflow",
		},
	}

	data, err := json.MarshalIndent(expectedPayload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Expected POST /workflows/execute-adhoc payload:\n%s", data)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
