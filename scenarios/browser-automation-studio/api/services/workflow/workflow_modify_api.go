package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

// ModifyWorkflowAPI applies an AI prompt to a workflow definition and persists the updated proto workflow file.
func (s *WorkflowService) ModifyWorkflowAPI(ctx context.Context, workflowID uuid.UUID, prompt string, current *basworkflows.WorkflowDefinitionV2) (*basapi.UpdateWorkflowResponse, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, &AIWorkflowError{Reason: "modification_prompt is required"}
	}

	getResp, err := s.GetWorkflowAPI(ctx, &basapi.GetWorkflowRequest{WorkflowId: workflowID.String()})
	if err != nil {
		return nil, err
	}
	if getResp == nil || getResp.Workflow == nil {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}

	base := getResp.Workflow
	if current == nil {
		current = base.FlowDefinition
	}

	updatedDef, err := s.generateWorkflowDefinitionFromPrompt(ctx, prompt, current)
	if err != nil {
		return nil, err
	}

	expected := base.Version
	return s.UpdateWorkflow(ctx, &basapi.UpdateWorkflowRequest{
		WorkflowId:        proto.String(workflowID.String()),
		Name:             base.Name,
		Description:      base.Description,
		FolderPath:       base.FolderPath,
		Tags:             append([]string(nil), base.Tags...),
		FlowDefinition:   updatedDef,
		ChangeDescription: fmt.Sprintf("AI modification: %s", truncatePrompt(prompt)),
		Source:           basbase.ChangeSource_CHANGE_SOURCE_AI_GENERATED,
		ExpectedVersion:  expected,
	})
}

func truncatePrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if len(prompt) <= 96 {
		return prompt
	}
	return prompt[:96] + "…"
}

