package protoconv

import (
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestWorkflowSummaryToProto(t *testing.T) {
	now := time.Now()
	wfID := uuid.New()
	projectID := uuid.New()
	workflow := &database.Workflow{
		ID:          wfID,
		ProjectID:   &projectID,
		Name:        "Workflow",
		FolderPath:  "/",
		Description: "desc",
		Tags:        []string{"a", "b"},
		Version:     3,
		IsTemplate:  false,
		CreatedBy:   "tester",
		CreatedAt:   now,
		UpdatedAt:   now,
		FlowDefinition: database.JSONMap{
			"nodes": []map[string]any{{"id": "n1", "type": "navigate"}},
			"edges": []map[string]any{},
		},
	}
	pb, err := WorkflowSummaryToProto(workflow)
	if err != nil {
		t.Fatalf("convert summary: %v", err)
	}
	if pb.Id != wfID.String() {
		t.Fatalf("id mismatch")
	}
	if pb.ProjectId != projectID.String() {
		t.Fatalf("project_id mismatch")
	}
	if pb.FlowDefinition == nil || len(pb.FlowDefinition.Nodes) != 1 {
		t.Fatalf("expected flow definition populated")
	}
}

func TestWorkflowVersionToProto(t *testing.T) {
	now := time.Now()
	workflowID := uuid.New()
	version := &database.WorkflowVersion{
		ID:                uuid.New(),
		WorkflowID:        workflowID,
		Version:           2,
		FlowDefinition:    database.JSONMap{"nodes": []map[string]any{}, "edges": []map[string]any{}},
		ChangeDescription: "desc",
		CreatedBy:         "me",
		CreatedAt:         now,
	}
	pb, err := WorkflowVersionToProto(version)
	if err != nil {
		t.Fatalf("convert version: %v", err)
	}
	if pb.WorkflowId != workflowID.String() || pb.Version != int32(version.Version) {
		t.Fatalf("mismatch in version fields")
	}
	if pb.FlowDefinition == nil {
		t.Fatalf("missing flow definition")
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
