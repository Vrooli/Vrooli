package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

func (s *WorkflowService) validateAdhocSubflows(ctx context.Context, flowDef *basworkflows.WorkflowDefinitionV2, workflowID uuid.UUID, projectRoot string) error {
	if flowDef == nil {
		return nil
	}

	for _, node := range flowDef.GetNodes() {
		action := node.GetAction()
		if action == nil {
			continue
		}
		subflow := action.GetSubflow()
		if subflow == nil {
			continue
		}
		workflowPath := strings.TrimSpace(subflow.GetWorkflowPath())
		if workflowPath == "" {
			continue
		}

		if strings.TrimSpace(projectRoot) == "" {
			return fmt.Errorf("subflow %s requires project_root to resolve workflow_path %q", node.GetId(), workflowPath)
		}

		if _, err := s.GetWorkflowByProjectPath(ctx, workflowID, workflowPath, projectRoot); err != nil {
			if s.log != nil {
				s.log.WithFields(logrus.Fields{
					"workflow_id":   workflowID.String(),
					"node_id":       node.GetId(),
					"workflow_path": workflowPath,
					"error":         err.Error(),
				}).Warn("Adhoc subflow resolution failed")
			}
			return fmt.Errorf("subflow %s cannot resolve workflow_path %q: %w", node.GetId(), workflowPath, err)
		}
	}

	return nil
}
