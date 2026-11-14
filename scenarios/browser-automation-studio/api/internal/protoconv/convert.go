package protoconv

import (
	"fmt"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

// ExecutionToProto converts a database.ExecutionIndex (DB index-only row) into the generated proto message.
// Detailed execution data is sourced from filesystem artifacts, not the database.
func ExecutionToProto(execution *database.ExecutionIndex) (*basexecution.Execution, error) {
	if execution == nil {
		return nil, fmt.Errorf("execution is nil")
	}

	pb := &basexecution.Execution{
		ExecutionId: execution.ID.String(),
		WorkflowId:  execution.WorkflowID.String(),
		Status:      StringToExecutionStatus(execution.Status),
		StartedAt:   autocontracts.TimeToTimestamp(execution.StartedAt),
		CreatedAt:   autocontracts.TimeToTimestamp(execution.CreatedAt),
		UpdatedAt:   autocontracts.TimeToTimestamp(execution.UpdatedAt),
	}

	pb.CompletedAt = autocontracts.TimePtrToTimestamp(execution.CompletedAt)
	if execution.ErrorMessage != "" {
		errMsg := execution.ErrorMessage
		pb.Error = &errMsg
	}
	if execution.ResumedFromID != nil {
		resumedFrom := execution.ResumedFromID.String()
		pb.ResumedFrom = &resumedFrom
	}

	return pb, nil
}

// ExecutionExportPreviewToProto converts the workflow.ExecutionExportPreview to the proto message.
func ExecutionExportPreviewToProto(preview *workflow.ExecutionExportPreview) (*basexecution.ExecutionExportPreview, error) {
	if preview == nil {
		return nil, fmt.Errorf("preview is nil")
	}

	return &basexecution.ExecutionExportPreview{
		ExecutionId:         preview.ExecutionID.String(),
		SpecId:              preview.SpecID,
		Status:              StringToExportStatus(preview.Status),
		Message:             preview.Message,
		CapturedFrameCount:  int32(preview.CapturedFrameCount),
		AvailableAssetCount: int32(preview.AvailableAssetCount),
		TotalDurationMs:     int32(preview.TotalDurationMs),
		Package:             preview.Package,
	}, nil
}
