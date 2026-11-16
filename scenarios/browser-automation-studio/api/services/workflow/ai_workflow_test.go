package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowGenerationFailsClosedWithoutInjectedAIClient(t *testing.T) {
	service := &WorkflowService{}
	_, err := service.generateWorkflowDefinitionFromPrompt(context.Background(), "build a workflow", nil)
	require.EqualError(t, err, "ai client not configured")
}
