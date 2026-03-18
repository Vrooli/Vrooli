// Package services contains business logic orchestration.
// This file handles tool configuration resolution for completion requests.
package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"agent-inbox/domain"
	"agent-inbox/integrations"
)

// resolveToolConfiguration sets up tool and tool_choice fields on the request.
func (s *CompletionService) resolveToolConfiguration(ctx context.Context, req *CompletionRequest, chatID string, settings *ChatSettings, forcedTool string, isImageGen bool) {
	// Check for forced tool FIRST, before tools_enabled check.
	var forcedToolDef map[string]interface{}
	if forcedTool != "" && !isImageGen {
		var err error
		forcedToolDef, err = s.getForcedToolDefinition(ctx, forcedTool)
		if err != nil {
			log.Printf("[WARN] Forced tool '%s' not found or invalid: %v", forcedTool, err)
		}
	}

	shouldIncludeTools := (settings.ToolsEnabled || forcedToolDef != nil) && !isImageGen

	if shouldIncludeTools {
		if forcedToolDef != nil {
			s.applyForcedTool(req, forcedToolDef, settings.ToolsEnabled)
		} else if settings.ToolsEnabled {
			tools, err := s.toolRegistry.GetToolsForOpenAI(ctx, chatID)
			if err != nil {
				log.Printf("warning: failed to get tools from registry: %v", err)
			} else {
				req.Tools = tools
			}
		}
	}

	// Validation: log if forced tool was specified but couldn't be set
	if forcedTool != "" && req.ToolChoice == nil {
		log.Printf("[ERROR] Forced tool '%s' was specified but not set. Check format: 'scenario:tool_name'", forcedTool)
	}
}

// applyForcedTool configures the request to use a single forced tool.
func (s *CompletionService) applyForcedTool(req *CompletionRequest, forcedToolDef map[string]interface{}, toolsEnabled bool) {
	toolName := ""
	if fn, ok := forcedToolDef["function"].(map[string]interface{}); ok {
		if name, ok := fn["name"].(string); ok {
			toolName = name
		}
	}

	req.Tools = []map[string]interface{}{forcedToolDef}
	req.ToolChoice = integrations.ToolChoiceFunction{
		Type:     "function",
		Function: integrations.ToolChoiceFunctionName{Name: toolName},
	}
	log.Printf("[DEBUG] Forced tool mode: sending only tool '%s' (tools_enabled=%v)", toolName, toolsEnabled)
}

// getForcedToolDefinition retrieves a tool by name, bypassing enabled filters.
// The forcedTool format is "scenario:tool_name".
func (s *CompletionService) getForcedToolDefinition(ctx context.Context, forcedTool string) (map[string]interface{}, error) {
	parts := strings.SplitN(forcedTool, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid forced tool format: expected 'scenario:tool_name', got '%s'", forcedTool)
	}

	toolName := parts[1]

	toolDef, _, err := s.toolRegistry.GetToolByName(ctx, toolName)
	if err != nil {
		return nil, fmt.Errorf("tool '%s' not found: %w", toolName, err)
	}

	openAITool := domain.ToOpenAIFunction(toolDef)
	return openAITool, nil
}

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
