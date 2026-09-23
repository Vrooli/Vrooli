package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/proto"
)

func (s *WorkflowService) ListExecutions(ctx context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	if s == nil {
		return nil, 0, fmt.Errorf("workflow service not configured")
	}
	return s.repo.ListExecutions(ctx, query)
}

// ResumeExecution resumes a failed or stopped execution from its last checkpoint.
// It creates a new execution record linked to the original and starts execution
// from the step after the last successfully completed step.
func (s *WorkflowService) ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	if s == nil {
		return nil, errors.New("workflow service not configured")
	}

	// Validate execution is resumable (includes workflow version check)
	if err := s.ValidateResumable(ctx, executionID); err != nil {
		return nil, fmt.Errorf("execution %s cannot be resumed: %w", executionID, err)
	}

	// Load original execution
	originalExec, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load execution: %w", err)
	}

	// Extract checkpoint state from disk (timeline.proto.json)
	checkpoint, err := s.ExtractCheckpointState(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract checkpoint: %w", err)
	}

	// Merge parameters (new params override checkpoint params)
	mergedParams := make(map[string]any)
	for k, v := range checkpoint.Params {
		mergedParams[k] = v
	}
	if parameters == nil {
		parameters = map[string]any{}
	}
	for k, v := range parameters {
		mergedParams[k] = v
	}

	// Extract resume_url if provided in parameters
	resumeURL := ""
	if urlVal, ok := parameters["resume_url"]; ok {
		if urlStr, isStr := urlVal.(string); isStr {
			resumeURL = strings.TrimSpace(urlStr)
			delete(mergedParams, "resume_url") // Don't pass as a workflow parameter
		}
	}

	// Preserve admission settings while replacing the recovered namespaces.
	admission := &basexecution.ExecutionParameters{}
	if checkpoint.AdmissionParameters != nil {
		admission = proto.Clone(checkpoint.AdmissionParameters).(*basexecution.ExecutionParameters)
	}
	admission.InitialStore = convertParamsToProto(checkpoint.Variables)
	admission.InitialParams = convertParamsToProto(mergedParams)
	admission.Env = convertParamsToProto(checkpoint.Env)
	if resumeURL != "" {
		admission.StartUrl = &resumeURL
	}
	response, err := s.ExecuteWorkflowAPIWithOptions(ctx, &basapi.ExecuteWorkflowRequest{
		WorkflowId:      originalExec.WorkflowID.String(),
		WorkflowVersion: proto.Int32(int32(checkpoint.WorkflowVersion)),
		Parameters:      admission,
	}, &ExecuteOptions{ResumeAfterStep: &checkpoint.LastStepIndex, ResumedFromID: &executionID})
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(response.ExecutionId)
	if err != nil {
		return nil, fmt.Errorf("invalid resumed execution identity: %w", err)
	}
	return s.repo.GetExecution(ctx, id)
}
