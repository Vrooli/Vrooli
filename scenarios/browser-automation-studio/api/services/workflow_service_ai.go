package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// generateWorkflowFromPrompt generates a workflow definition via OpenRouter.
func (s *WorkflowService) generateWorkflowFromPrompt(ctx context.Context, prompt string) (database.JSONMap, error) {
	if s.aiClient == nil {
		return nil, errors.New("openrouter client not configured")
	}

	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return nil, errors.New("empty AI prompt")
	}

	instruction := fmt.Sprintf(`You are a strict JSON generator. Produce exactly one JSON object with the following structure:
{
  "nodes": [
    {
      "id": "node-1",
      "type": "navigate" | "click" | "type" | "shortcut" | "wait" | "screenshot" | "extract",
      "position": {"x": <number>, "y": <number>},
      "data": { ... } // include all parameters needed for the step (url, selector, text, waitMs, etc.)
    }
  ],
  "edges": [
    {"id": "edge-1", "source": "node-1", "target": "node-2"}
  ]
}

Rules:
1. Tailor every node and selector to the user's request. Never use placeholders such as "https://example.com" or generic selectors.
2. Provide only the fields needed to execute the step (e.g., url, selector, text, waitMs, timeoutMs, screenshot name). Keep the response concise.
3. Arrange nodes with sensible coordinates (e.g., x increments by ~180 horizontally, y by ~120 vertically for branches).
4. Include necessary wait/ensure steps before interactions to make the automation reliable. Use the "wait" type for waits/ensure conditions.
5. Valid node types are limited to: navigate, click, type, shortcut, wait, screenshot, extract. Do not invent new types or nested workflow nodes.
6. Wrap the JSON in markers exactly like this: <WORKFLOW_JSON>{...}</WORKFLOW_JSON>.
7. The response MUST start with '<WORKFLOW_JSON>{' and end with '}</WORKFLOW_JSON>'. Output minified JSON on a single line (no spaces or newlines) and keep it under 1200 characters in total.
8. If you cannot produce a valid workflow, respond with <WORKFLOW_JSON>{"error":"reason"}</WORKFLOW_JSON>.

User prompt:
%s`, trimmed)

	start := time.Now()
	response, err := s.aiClient.ExecutePrompt(ctx, instruction)
	if err != nil {
		return nil, fmt.Errorf("openrouter execution error: %w", err)
	}
	s.log.WithFields(logrus.Fields{
		"model":            s.aiClient.model,
		"duration_ms":      time.Since(start).Milliseconds(),
		"response_preview": truncateForLog(response, 400),
	}).Info("AI workflow generated via OpenRouter")

	cleaned := extractJSONObject(stripJSONCodeFence(response))

	var payload map[string]any
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"raw_response": truncateForLog(response, 2000),
			"cleaned":      truncateForLog(cleaned, 2000),
		}).Error("Failed to parse workflow JSON returned by OpenRouter")
		return nil, fmt.Errorf("failed to parse OpenRouter JSON: %w", err)
	}

	if err := detectAIWorkflowError(payload); err != nil {
		return nil, err
	}

	definition, err := normalizeFlowDefinition(payload)
	if err != nil {
		return nil, err
	}

	if len(toInterfaceSlice(definition["nodes"])) == 0 {
		return nil, &AIWorkflowError{Reason: "AI workflow generation did not return any steps. Specify real URLs, selectors, and actions, then try again."}
	}

	return definition, nil
}

func stripJSONCodeFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		if idx := strings.Index(trimmed, "\n"); idx != -1 {
			trimmed = trimmed[idx+1:]
		}
		if idx := strings.LastIndex(trimmed, "```"); idx != -1 {
			trimmed = trimmed[:idx]
		}
	}
	return strings.TrimSpace(trimmed)
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return strings.TrimSpace(raw)
	}
	return strings.TrimSpace(raw[start : end+1])
}

func normalizeFlowDefinition(payload map[string]any) (database.JSONMap, error) {
	candidate := payload
	if workflow, ok := payload["workflow"].(map[string]any); ok {
		candidate = workflow
	}

	if err := detectAIWorkflowError(candidate); err != nil {
		return nil, err
	}

	rawNodes, ok := candidate["nodes"]
	if !ok {
		if steps, ok := candidate["steps"].([]any); ok {
			candidate["nodes"] = steps
			rawNodes = steps
		}
	}

	nodes, ok := rawNodes.([]any)
	if !ok || nodes == nil {
		nodes = []any{}
	}

	for i, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok {
			return nil, errors.New("AI node payload is not an object")
		}
		if _, ok := node["id"].(string); !ok {
			node["id"] = fmt.Sprintf("node-%d", i+1)
		}
		if _, ok := node["position"].(map[string]any); !ok {
			node["position"] = map[string]any{
				"x": float64(100 + i*200),
				"y": float64(100 + i*120),
			}
		}
		if dataValue, hasData := node["data"]; hasData {
			if cleaned, changed := stripPreviewData(dataValue); changed {
				node["data"] = cleaned
			}
		}
		nodes[i] = node
	}
	candidate["nodes"] = nodes

	edgesRaw, hasEdges := candidate["edges"]
	edges, _ := edgesRaw.([]any)
	if !hasEdges || edges == nil {
		edges = []any{}
	}

	if len(edges) == 0 && len(nodes) > 1 {
		edges = make([]any, 0, len(nodes)-1)
		for i := 0; i < len(nodes)-1; i++ {
			source := nodes[i].(map[string]any)["id"].(string)
			target := nodes[i+1].(map[string]any)["id"].(string)
			edges = append(edges, map[string]any{
				"id":     fmt.Sprintf("edge-%d", i+1),
				"source": source,
				"target": target,
			})
		}
	}
	candidate["edges"] = edges

	return sanitizeWorkflowDefinition(database.JSONMap(candidate)), nil
}

func detectAIWorkflowError(payload map[string]any) error {
	if reason := extractAIErrorMessage(payload); reason != "" {
		return &AIWorkflowError{Reason: reason}
	}
	return nil
}

func extractAIErrorMessage(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if msg, ok := typed["error"].(string); ok {
			trimmed := strings.TrimSpace(msg)
			if trimmed != "" {
				return trimmed
			}
		}
		if msg, ok := typed["message"].(string); ok {
			trimmed := strings.TrimSpace(msg)
			if trimmed != "" {
				return trimmed
			}
		}
		if workflow, ok := typed["workflow"].(map[string]any); ok {
			if nested := extractAIErrorMessage(workflow); nested != "" {
				return nested
			}
		}
		for _, nested := range typed {
			if nestedMsg := extractAIErrorMessage(nested); nestedMsg != "" {
				return nestedMsg
			}
		}
	}
	return ""
}

func defaultWorkflowDefinition() database.JSONMap {
	return database.JSONMap{
		"nodes": []any{
			map[string]any{
				"id":   "node-1",
				"type": "navigate",
				"position": map[string]any{
					"x": float64(100),
					"y": float64(100),
				},
				"data": map[string]any{
					"url": "https://example.com",
				},
			},
			map[string]any{
				"id":   "node-2",
				"type": "screenshot",
				"position": map[string]any{
					"x": float64(350),
					"y": float64(220),
				},
				"data": map[string]any{
					"name": "homepage",
				},
			},
		},
		"edges": []any{
			map[string]any{
				"id":     "edge-1",
				"source": "node-1",
				"target": "node-2",
			},
		},
	}
}

// ModifyWorkflow applies an AI-driven modification to an existing workflow.
func (s *WorkflowService) ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, modificationPrompt string, overrideFlow map[string]any) (*database.Workflow, error) {
	if s.aiClient == nil {
		return nil, errors.New("openrouter client not configured")
	}
	if strings.TrimSpace(modificationPrompt) == "" {
		return nil, errors.New("modification prompt is required")
	}

	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	baseFlow := map[string]any{}
	if overrideFlow != nil {
		baseFlow = overrideFlow
	} else if workflow.FlowDefinition != nil {
		bytes, err := json.Marshal(workflow.FlowDefinition)
		if err == nil {
			_ = json.Unmarshal(bytes, &baseFlow)
		}
	}

	baseFlowJSON, err := json.MarshalIndent(baseFlow, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal base workflow: %w", err)
	}

	instruction := fmt.Sprintf(`You are a strict JSON generator. Update the existing workflow so it satisfies the user's request.

Rules:
1. Respond with a single JSON object that uses the same schema as the original workflow ("nodes" array + "edges" array).
2. Preserve existing node IDs when the step remains applicable. Modify node types/data/positions only where necessary, and keep data concise (only the parameters required to execute the step).
3. Keep the graph valid: edges must describe a reachable execution path.
4. Fill in realistic selectors, URLs, filenames, waits, etc.—no placeholders. Only use the allowed node types: navigate, click, type, shortcut, wait, screenshot, extract.
5. Wrap the JSON in markers exactly like this: <WORKFLOW_JSON>{...}</WORKFLOW_JSON>.
6. The response MUST start with '<WORKFLOW_JSON>{' and end with '}</WORKFLOW_JSON>'. Output minified JSON on a single line (no spaces or newlines) and keep it shorter than 1200 characters.
7. If the request cannot be satisfied, respond with <WORKFLOW_JSON>{"error":"reason"}</WORKFLOW_JSON>.

Original workflow JSON:
%s

User requested modifications:
%s`, string(baseFlowJSON), strings.TrimSpace(modificationPrompt))

	start := time.Now()
	response, err := s.aiClient.ExecutePrompt(ctx, instruction)
	if err != nil {
		return nil, fmt.Errorf("openrouter execution error: %w", err)
	}
	s.log.WithFields(logrus.Fields{
		"model":       s.aiClient.model,
		"duration_ms": time.Since(start).Milliseconds(),
		"workflow_id": workflowID,
	}).Info("AI workflow modification generated via OpenRouter")

	cleaned := extractJSONObject(stripJSONCodeFence(response))

	var payload map[string]any
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"raw_response": truncateForLog(response, 2000),
			"cleaned":      truncateForLog(cleaned, 2000),
			"workflow_id":  workflowID,
		}).Error("Failed to parse modified workflow JSON returned by OpenRouter")
		return nil, fmt.Errorf("failed to parse OpenRouter JSON: %w", err)
	}

	definition, err := normalizeFlowDefinition(payload)
	if err != nil {
		return nil, err
	}

	if len(toInterfaceSlice(definition["nodes"])) == 0 {
		return nil, &AIWorkflowError{Reason: "AI workflow generation did not return any steps. Specify real URLs, selectors, and actions, then try again."}
	}

	workflow.FlowDefinition = sanitizeWorkflowDefinition(definition)
	workflow.Version++
	workflow.UpdatedAt = time.Now()

	if err := s.repo.UpdateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}
