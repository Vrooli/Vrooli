// Package services contains business logic orchestration.
// This file handles async guidance for completion requests.
package services

import (
	"fmt"
	"strings"
)

// buildAsyncGuidanceMessage creates a system message about active async operations.
func (s *CompletionService) buildAsyncGuidanceMessage(ops []*AsyncOperation) string {
	var toolNames []string
	for _, op := range ops {
		toolNames = append(toolNames, op.ToolName)
	}

	return fmt.Sprintf(
		"IMPORTANT: The following tools are currently executing asynchronously: %s. "+
			"You will receive their results automatically when they complete. "+
			"DO NOT call any status-checking or polling tools - the results will be delivered to you without any action on your part. "+
			"Please wait patiently or continue with other tasks while these operations complete.",
		strings.Join(toolNames, ", "),
	)
}

// Note: isImageGenerationModel logic is in ContextManager.IsImageGenerationModel
