package protoconv

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/browser-automation-studio/database"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FlowDefinitionToProto converts a loosely-typed JSON flow definition into the typed proto.
// This helper remains for legacy JSON payloads (ex: record-mode) that are still maps.
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

// WorkflowValidationResultToProto converts validator.Result to proto.
func WorkflowValidationResultToProto(result *workflowvalidator.Result) *basapi.WorkflowValidationResult {
	if result == nil {
		return nil
	}
	issueToProto := func(issue workflowvalidator.Issue) *basapi.WorkflowValidationIssue {
		return &basapi.WorkflowValidationIssue{
			Severity: StringToValidationSeverity(string(issue.Severity)),
			Code:     issue.Code,
			Message:  issue.Message,
			NodeId:   issue.NodeID,
			NodeType: StringToActionType(issue.NodeType),
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

