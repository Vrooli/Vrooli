package protoconv

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FlowDefinitionToProto converts the stored JSON flow definition into the typed proto.
func FlowDefinitionToProto(definition database.JSONMap) (*basworkflows.WorkflowDefinitionV2, error) {
	pb := &basworkflows.WorkflowDefinitionV2{}
	if definition == nil {
		return pb, nil
	}
	raw, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("marshal flow_definition: %w", err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, pb); err != nil {
		return nil, fmt.Errorf("unmarshal flow_definition: %w", err)
	}
	return pb, nil
}

// WorkflowSummaryToProto converts the DB workflow to the proto summary.
func WorkflowSummaryToProto(workflow *database.Workflow) (*basapi.WorkflowSummary, error) {
	if workflow == nil {
		return nil, nil
	}
	definition, err := FlowDefinitionToProto(workflow.FlowDefinition)
	if err != nil {
		return nil, err
	}
	pb := &basapi.WorkflowSummary{
		Id:                    workflow.ID.String(),
		Name:                  workflow.Name,
		FolderPath:            workflow.FolderPath,
		Description:           workflow.Description,
		Tags:                  workflow.Tags,
		Version:               int32(workflow.Version),
		IsTemplate:            workflow.IsTemplate,
		CreatedBy:             workflow.CreatedBy,
		LastChangeSource:      typeconv.StringToChangeSource(workflow.LastChangeSource),
		LastChangeDescription: workflow.LastChangeDescription,
		CreatedAt:             timestamppb.New(workflow.CreatedAt),
		UpdatedAt:             timestamppb.New(workflow.UpdatedAt),
		FlowDefinition:        definition,
	}
	if workflow.ProjectID != nil {
		pb.ProjectId = workflow.ProjectID.String()
	}
	return pb, nil
}

// WorkflowVersionToProto converts a workflow version record.
func WorkflowVersionToProto(version *database.WorkflowVersion) (*basapi.WorkflowVersion, error) {
	if version == nil {
		return nil, nil
	}
	definition, err := FlowDefinitionToProto(version.FlowDefinition)
	if err != nil {
		return nil, err
	}
	return &basapi.WorkflowVersion{
		WorkflowId:        version.WorkflowID.String(),
		Version:           int32(version.Version),
		FlowDefinition:    definition,
		ChangeDescription: version.ChangeDescription,
		CreatedBy:         version.CreatedBy,
		CreatedAt:         timestamppb.New(version.CreatedAt),
	}, nil
}

// WorkflowVersionSummaryToProto converts a workflow service summary to the proto.
// This is used by HTTP handlers that operate on summarized records instead of the raw DB row.
func WorkflowVersionSummaryToProto(summary *workflow.WorkflowVersionSummary) (*basapi.WorkflowVersion, error) {
	if summary == nil {
		return nil, nil
	}

	definition, err := FlowDefinitionToProto(summary.FlowDefinition)
	if err != nil {
		return nil, err
	}

	return &basapi.WorkflowVersion{
		WorkflowId:     summary.WorkflowID.String(),
		Version:        int32(summary.Version),
		FlowDefinition: definition,
		// CreatedBy may be empty; keep it as-is rather than guessing.
		CreatedBy: summary.CreatedBy,
		CreatedAt: timestamppb.New(summary.CreatedAt),
		// ChangeDescription is available on summaries and should be carried over.
		ChangeDescription: summary.ChangeDescription,
	}, nil
}

// CreateWorkflowResponseProto wraps the created workflow.
func CreateWorkflowResponseProto(workflow *database.Workflow) (*basapi.CreateWorkflowResponse, error) {
	summary, err := WorkflowSummaryToProto(workflow)
	if err != nil {
		return nil, err
	}
	return &basapi.CreateWorkflowResponse{
		Workflow:       summary,
		FlowDefinition: summary.GetFlowDefinition(),
	}, nil
}

// UpdateWorkflowResponseProto wraps the updated workflow.
func UpdateWorkflowResponseProto(workflow *database.Workflow) (*basapi.UpdateWorkflowResponse, error) {
	summary, err := WorkflowSummaryToProto(workflow)
	if err != nil {
		return nil, err
	}
	return &basapi.UpdateWorkflowResponse{
		Workflow:       summary,
		FlowDefinition: summary.GetFlowDefinition(),
	}, nil
}

// ExecuteWorkflowResponseProto returns the execution response wrapper.
func ExecuteWorkflowResponseProto(execution *database.Execution) (*basapi.ExecuteWorkflowResponse, error) {
	if execution == nil {
		return nil, fmt.Errorf("execution is nil")
	}

	pb := &basapi.ExecuteWorkflowResponse{
		ExecutionId: execution.ID.String(),
		Status:      typeconv.StringToExecutionStatus(execution.Status),
	}

	if execution.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*execution.CompletedAt)
	}
	if execution.Error.Valid {
		errMsg := execution.Error.String
		pb.Error = &errMsg
	}

	return pb, nil
}

// RestoreWorkflowVersionResponseProto wraps the restored workflow and version.
func RestoreWorkflowVersionResponseProto(workflow *database.Workflow, version *workflow.WorkflowVersionSummary) (*basapi.RestoreWorkflowVersionResponse, error) {
	if workflow == nil || version == nil {
		return nil, fmt.Errorf("workflow or version is nil")
	}

	wfSummary, err := WorkflowSummaryToProto(workflow)
	if err != nil {
		return nil, err
	}
	versionProto, err := WorkflowVersionSummaryToProto(version)
	if err != nil {
		return nil, err
	}

	return &basapi.RestoreWorkflowVersionResponse{
		Workflow:        wfSummary,
		RestoredVersion: versionProto,
	}, nil
}

// WorkflowValidationResultToProto converts validator.Result to proto.
func WorkflowValidationResultToProto(result *workflowvalidator.Result) *basapi.WorkflowValidationResult {
	if result == nil {
		return nil
	}
	issueToProto := func(issue workflowvalidator.Issue) *basapi.WorkflowValidationIssue {
		return &basapi.WorkflowValidationIssue{
			Severity: typeconv.StringToValidationSeverity(string(issue.Severity)),
			Code:     issue.Code,
			Message:  issue.Message,
			NodeId:   issue.NodeID,
			NodeType: typeconv.StringToActionType(issue.NodeType),
			Field:    issue.Field,
			Pointer:  issue.Pointer,
			Hint:     issue.Hint,
		}
	}

	errors := make([]*basapi.WorkflowValidationIssue, 0, len(result.Errors))
	for _, e := range result.Errors {
		errors = append(errors, issueToProto(e))
	}
	warnings := make([]*basapi.WorkflowValidationIssue, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		warnings = append(warnings, issueToProto(w))
	}

	return &basapi.WorkflowValidationResult{
		Valid:    result.Valid,
		Errors:   errors,
		Warnings: warnings,
		Stats: &basapi.WorkflowValidationStats{
			NodeCount:            int32(result.Stats.NodeCount),
			EdgeCount:            int32(result.Stats.EdgeCount),
			SelectorCount:        int32(result.Stats.SelectorCount),
			UniqueSelectorCount:  int32(result.Stats.UniqueSelectorCount),
			ElementWaitCount:     int32(result.Stats.ElementWaitCount),
			HasMetadata:          result.Stats.HasMetadata,
			HasExecutionViewport: result.Stats.HasExecutionViewport,
		},
		SchemaVersion: result.SchemaVersion,
		CheckedAt:     timestamppb.New(result.CheckedAt),
		DurationMs:    result.DurationMs,
	}
}

