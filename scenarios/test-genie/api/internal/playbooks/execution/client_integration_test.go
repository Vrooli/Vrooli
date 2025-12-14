//go:build integration

// Package execution provides BAS API client and proto conversion utilities.
// This file contains integration tests that require a running BAS instance.
package execution

import (
	"context"
	"os"
	"testing"
	"time"

	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestBASProtoContract validates that BAS returns proto-compliant responses.
// Run with: go test -tags=integration -run TestBASProtoContract
func TestBASProtoContract(t *testing.T) {
	basURL := os.Getenv("BAS_URL")
	if basURL == "" {
		basURL = "http://localhost:15000/api/v1" // Default BAS port
	}

	client := NewClient(basURL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test 1: Health check
	t.Run("Health", func(t *testing.T) {
		if err := client.Health(ctx); err != nil {
			t.Skipf("BAS not available at %s: %v", basURL, err)
		}
	})

	// Test 2: Execute a minimal workflow and validate proto response
	t.Run("ExecuteAdhoc", func(t *testing.T) {
		if err := client.Health(ctx); err != nil {
			t.Skip("BAS not available")
		}

		// Minimal valid workflow - just evaluates an expression
		definition := map[string]any{
			"nodes": []any{
				map[string]any{
					"id":   "eval-node",
					"type": "evaluate",
					"data": map[string]any{
						"label":       "Return test value",
						"expression":  `"test-genie-integration"`,
						"storeResult": "testValue",
					},
				},
			},
			"edges": []any{},
		}

		executionID, err := client.ExecuteWorkflow(ctx, definition, "proto-contract-test")
		if err != nil {
			t.Fatalf("ExecuteWorkflow failed: %v", err)
		}
		if executionID == "" {
			t.Fatal("ExecuteWorkflow returned empty execution ID")
		}
		t.Logf("Execution ID: %s", executionID)

		// Test 3: GetStatus returns valid proto
		t.Run("GetStatus", func(t *testing.T) {
			status, err := client.GetStatus(ctx, executionID)
			if err != nil {
				t.Fatalf("GetStatus failed: %v", err)
			}
			validateExecutionProto(t, status)
		})

		// Wait for completion
		if err := client.WaitForCompletion(ctx, executionID); err != nil {
			t.Logf("Workflow execution result: %v", err)
		}

		// Test 4: GetTimeline returns valid proto
		t.Run("GetTimeline", func(t *testing.T) {
			timeline, rawData, err := client.GetTimeline(ctx, executionID)
			if err != nil {
				t.Fatalf("GetTimeline failed: %v", err)
			}
			if timeline == nil {
				t.Fatal("GetTimeline returned nil timeline")
			}
			if len(rawData) == 0 {
				t.Fatal("GetTimeline returned empty raw data")
			}
			validateTimelineProto(t, timeline)

			// Verify raw data can be re-parsed as proto
			var reparsed bastimeline.ExecutionTimeline
			opts := protojson.UnmarshalOptions{DiscardUnknown: true}
			if err := opts.Unmarshal(rawData, &reparsed); err != nil {
				t.Fatalf("Failed to re-parse raw timeline as proto: %v", err)
			}
			if reparsed.GetExecutionId() != timeline.GetExecutionId() {
				t.Error("Re-parsed execution ID doesn't match")
			}
		})

		// Test 5: ParseFullTimeline extracts data correctly
		t.Run("ParseFullTimeline", func(t *testing.T) {
			_, rawData, err := client.GetTimeline(ctx, executionID)
			if err != nil {
				t.Fatalf("GetTimeline failed: %v", err)
			}

			parsed, err := ParseFullTimeline(rawData)
			if err != nil {
				t.Fatalf("ParseFullTimeline failed: %v", err)
			}
			if parsed == nil {
				t.Fatal("ParseFullTimeline returned nil")
			}
			if parsed.Proto == nil {
				t.Error("ParseFullTimeline didn't set Proto field")
			}
			if parsed.ExecutionID == "" {
				t.Error("ParseFullTimeline didn't extract execution ID")
			}
			if parsed.Status == "" {
				t.Error("ParseFullTimeline didn't extract status")
			}
			t.Logf("Parsed timeline: status=%s, progress=%d%%, frames=%d",
				parsed.Status, parsed.Progress, len(parsed.Frames))
		})
	})
}

// validateExecutionProto checks that an Execution proto has expected fields populated.
func validateExecutionProto(t *testing.T, exec *basexecution.Execution) {
	t.Helper()

	if exec.GetExecutionId() == "" {
		t.Error("Execution.ExecutionId is empty")
	}
	if exec.GetStatus() == basexecution.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED {
		t.Error("Execution.Status is unspecified")
	}

	// Progress should be in range
	if exec.GetProgress() < 0 || exec.GetProgress() > 100 {
		t.Errorf("Execution.Progress out of range: %d", exec.GetProgress())
	}

	t.Logf("Execution proto valid: id=%s status=%s progress=%d%%",
		exec.GetExecutionId(), exec.GetStatus(), exec.GetProgress())
}

// validateTimelineProto checks that an ExecutionTimeline proto has expected structure.
func validateTimelineProto(t *testing.T, timeline *bastimeline.ExecutionTimeline) {
	t.Helper()

	if timeline.GetExecutionId() == "" {
		t.Error("Timeline.ExecutionId is empty")
	}
	if timeline.GetStatus() == 0 {
		t.Error("Timeline.Status is unspecified")
	}

	// Should have at least one entry for any executed workflow
	entries := timeline.GetEntries()
	if len(entries) == 0 {
		t.Log("Warning: Timeline has no entries (workflow may have failed early)")
	}

	// Validate each entry
	for i, entry := range entries {
		if entry.GetNodeId() == "" {
			t.Errorf("Entry[%d].NodeId is empty", i)
		}

		// Check action type
		action := entry.GetAction()
		if action == nil {
			t.Errorf("Entry[%d].Action is nil", i)
		}

		// Check context
		ctx := entry.GetContext()

		// Check for assertion entries
		if action != nil && action.GetType().String() == "ACTION_TYPE_ASSERT" {
			if ctx != nil {
				if assertion := ctx.GetAssertion(); assertion != nil {
					t.Logf("Entry[%d] assertion: mode=%s selector=%s success=%v",
						i, assertion.GetMode(), assertion.GetSelector(), assertion.GetSuccess())
				}
			}
		}
	}

	t.Logf("Timeline proto valid: execution_id=%s status=%s entries=%d",
		timeline.GetExecutionId(), timeline.GetStatus(), len(entries))
}
