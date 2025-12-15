package protoconv

import (
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/database"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

func TestFlowDefinitionToProto(t *testing.T) {
	def := database.JSONMap{
		"nodes": []map[string]any{
			{"id": "n1", "type": "navigate"},
		},
		"edges":    []map[string]any{},
		"metadata": map[string]any{"name": "Test"},
	}
	pb, err := FlowDefinitionToProto(def)
	if err != nil {
		t.Fatalf("convert definition: %v", err)
	}
	if len(pb.Nodes) != 1 || pb.Nodes[0].Id != "n1" {
		t.Fatalf("unexpected nodes: %+v", pb.Nodes)
	}
	if pb.Metadata == nil || pb.Metadata.Name == nil || *pb.Metadata.Name != "Test" {
		t.Fatalf("expected metadata with name")
	}
}

func TestWorkflowValidationResultToProto(t *testing.T) {
	now := time.Now()
	result := &workflowvalidator.Result{
		Valid: true,
		Errors: []workflowvalidator.Issue{
			{Severity: workflowvalidator.SeverityError, Code: "E1", Message: "msg"},
		},
		Warnings: []workflowvalidator.Issue{
			{Severity: workflowvalidator.SeverityWarning, Code: "W1", Message: "warn"},
		},
		Stats: workflowvalidator.Stats{
			NodeCount:            1,
			EdgeCount:            2,
			SelectorCount:        3,
			UniqueSelectorCount:  4,
			ElementWaitCount:     5,
			HasMetadata:          true,
			HasExecutionViewport: true,
		},
		SchemaVersion: "v1",
		CheckedAt:     now,
		DurationMs:    42,
	}

	pb := WorkflowValidationResultToProto(result)
	if pb == nil || len(pb.Errors) != 1 || len(pb.Warnings) != 1 {
		t.Fatalf("unexpected validation proto")
	}
	if pb.SchemaVersion != "v1" || pb.DurationMs != 42 {
		t.Fatalf("unexpected meta fields")
	}
}

