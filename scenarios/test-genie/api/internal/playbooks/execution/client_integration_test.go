//go:build integration

// Package execution provides BAS API client and proto conversion utilities.
// This file contains integration tests that require a running BAS instance.
package execution

import (
	"context"
	"os"
	"testing"
	"time"

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
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
			var reparsed basv1.ExecutionTimeline
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
func validateExecutionProto(t *testing.T, exec *basv1.Execution) {
	t.Helper()

	if exec.GetId() == "" {
		t.Error("Execution.Id is empty")
	}
	if exec.GetStatus() == "" {
		t.Error("Execution.Status is empty")
	}

	// Status should be one of the known values
	validStatuses := map[string]bool{
		"pending": true, "running": true, "completed": true,
		"failed": true, "cancelled": true,
	}
	if !validStatuses[exec.GetStatus()] {
		t.Errorf("Execution.Status has unexpected value: %s", exec.GetStatus())
	}

	// Progress should be in range
	if exec.GetProgress() < 0 || exec.GetProgress() > 100 {
		t.Errorf("Execution.Progress out of range: %d", exec.GetProgress())
	}

	t.Logf("Execution proto valid: id=%s status=%s progress=%d%%",
		exec.GetId(), exec.GetStatus(), exec.GetProgress())
}

// validateTimelineProto checks that an ExecutionTimeline proto has expected structure.
func validateTimelineProto(t *testing.T, timeline *basv1.ExecutionTimeline) {
	t.Helper()

	if timeline.GetExecutionId() == "" {
		t.Error("Timeline.ExecutionId is empty")
	}
	if timeline.GetStatus() == "" {
		t.Error("Timeline.Status is empty")
	}

	// Should have at least one frame for any executed workflow
	frames := timeline.GetFrames()
	if len(frames) == 0 {
		t.Log("Warning: Timeline has no frames (workflow may have failed early)")
	}

	// Validate each frame
	for i, frame := range frames {
		if frame.GetNodeId() == "" {
			t.Errorf("Frame[%d].NodeId is empty", i)
		}
		if frame.GetStepType() == "" {
			t.Errorf("Frame[%d].StepType is empty", i)
		}
		if frame.GetStatus() == "" {
			t.Errorf("Frame[%d].Status is empty", i)
		}

		// Check for assertion frames
		if frame.GetStepType() == "assert" {
			assertion := frame.GetAssertion()
			if assertion != nil {
				t.Logf("Frame[%d] assertion: mode=%s selector=%s success=%v",
					i, assertion.GetMode(), assertion.GetSelector(), assertion.GetSuccess())
			}
		}
	}

	t.Logf("Timeline proto valid: execution_id=%s status=%s frames=%d",
		timeline.GetExecutionId(), timeline.GetStatus(), len(frames))
}

// TestProtoConversion validates the workflow-to-proto conversion.
func TestProtoConversion(t *testing.T) {
	t.Run("ValidWorkflow", func(t *testing.T) {
		definition := map[string]any{
			"metadata": map[string]any{
				"description": "Test workflow",
				"version":     1,
			},
			"settings": map[string]any{
				"executionViewport": map[string]any{
					"width":  1440,
					"height": 900,
				},
			},
			"nodes": []any{
				map[string]any{
					"id":   "nav-node",
					"type": "navigate",
					"data": map[string]any{
						"url":       "https://example.com",
						"waitUntil": "networkidle",
					},
				},
				map[string]any{
					"id":   "assert-node",
					"type": "assert",
					"data": map[string]any{
						"selector":   "body",
						"assertMode": "exists",
					},
				},
			},
			"edges": []any{
				map[string]any{
					"id":     "e1",
					"source": "nav-node",
					"target": "assert-node",
					"type":   "smoothstep",
				},
			},
		}

		proto, err := WorkflowToProto(definition)
		if err != nil {
			t.Fatalf("WorkflowToProto failed: %v", err)
		}

		// Validate structure
		if len(proto.GetNodes()) != 2 {
			t.Errorf("Expected 2 nodes, got %d", len(proto.GetNodes()))
		}
		if len(proto.GetEdges()) != 1 {
			t.Errorf("Expected 1 edge, got %d", len(proto.GetEdges()))
		}

		// Validate node details
		navNode := proto.GetNodes()[0]
		if navNode.GetId() != "nav-node" {
			t.Errorf("Node ID mismatch: expected nav-node, got %s", navNode.GetId())
		}
		if navNode.GetType() != "navigate" {
			t.Errorf("Node type mismatch: expected navigate, got %s", navNode.GetType())
		}
		if navNode.GetData() == nil {
			t.Error("Node data is nil")
		}

		// Validate edge details
		edge := proto.GetEdges()[0]
		if edge.GetSource() != "nav-node" {
			t.Errorf("Edge source mismatch: expected nav-node, got %s", edge.GetSource())
		}
		if edge.GetTarget() != "assert-node" {
			t.Errorf("Edge target mismatch: expected assert-node, got %s", edge.GetTarget())
		}
	})

	t.Run("BuildAdhocRequest", func(t *testing.T) {
		definition := map[string]any{
			"nodes": []any{
				map[string]any{"id": "n1", "type": "evaluate", "data": map[string]any{}},
			},
			"edges": []any{},
		}

		req, err := BuildAdhocRequest(definition, "test-workflow")
		if err != nil {
			t.Fatalf("BuildAdhocRequest failed: %v", err)
		}

		if req.GetFlowDefinition() == nil {
			t.Error("FlowDefinition is nil")
		}
		if req.GetWaitForCompletion() {
			t.Error("WaitForCompletion should be false")
		}
		if req.GetMetadata() == nil {
			t.Error("Metadata is nil")
		}
		if req.GetMetadata().GetName() != "test-workflow" {
			t.Errorf("Metadata.Name mismatch: got %s", req.GetMetadata().GetName())
		}
	})

	t.Run("NilDefinition", func(t *testing.T) {
		_, err := WorkflowToProto(nil)
		if err == nil {
			t.Error("Expected error for nil definition")
		}
	})

	t.Run("EmptyDefinition", func(t *testing.T) {
		// Empty workflow is valid (no nodes, no edges)
		proto, err := WorkflowToProto(map[string]any{})
		if err != nil {
			t.Fatalf("WorkflowToProto failed for empty definition: %v", err)
		}
		if proto == nil {
			t.Error("Expected non-nil proto for empty definition")
		}
	})
}
