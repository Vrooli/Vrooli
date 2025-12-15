package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

func (s *WorkflowService) generateWorkflowDefinitionFromPrompt(ctx context.Context, prompt string, current *basworkflows.WorkflowDefinitionV2) (*basworkflows.WorkflowDefinitionV2, error) {
	if s == nil || s.aiClient == nil {
		return nil, errors.New("ai client not configured")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, &AIWorkflowError{Reason: "AI prompt is required"}
	}

	systemPrompt := buildWorkflowAIPrompt(prompt, current)
	raw, err := s.aiClient.ExecutePrompt(ctx, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("execute ai prompt: %w", err)
	}

	jsonBlob := extractBetweenMarkers(raw, workflowJSONStartMarker, workflowJSONEndMarker)
	if strings.TrimSpace(jsonBlob) == "" {
		// Allow AI to return raw JSON without markers.
		jsonBlob = strings.TrimSpace(raw)
	}

	// Preferred: protojson WorkflowDefinitionV2.
	var pb basworkflows.WorkflowDefinitionV2
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(jsonBlob), &pb); err == nil {
		return &pb, nil
	}

	// Fallback: interpret as legacy nodes/edges and convert to V2.
	var payload map[string]any
	if err := json.Unmarshal([]byte(jsonBlob), &payload); err != nil {
		return nil, &AIWorkflowError{Reason: "AI did not return valid JSON workflow definition"}
	}

	nodes := ToInterfaceSlice(payload["nodes"])
	edges := ToInterfaceSlice(payload["edges"])
	if flow, ok := payload["flow_definition"].(map[string]any); ok && (len(nodes) == 0 && len(edges) == 0) {
		nodes = ToInterfaceSlice(flow["nodes"])
		edges = ToInterfaceSlice(flow["edges"])
	}

	v2, err := V1NodesEdgesToV2Definition(nodes, edges, payload)
	if err != nil {
		return nil, &AIWorkflowError{Reason: fmt.Sprintf("AI workflow conversion failed: %s", err.Error())}
	}
	return v2, nil
}

func buildWorkflowAIPrompt(userPrompt string, current *basworkflows.WorkflowDefinitionV2) string {
	var currentJSON string
	if current != nil {
		opts := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}
		if raw, err := opts.Marshal(current); err == nil {
			currentJSON = string(raw)
		}
	}

	lines := []string{
		"You are generating Browser Automation Studio workflow definitions.",
		"Return ONLY a single JSON object between the markers.",
		"The JSON MUST be protojson for browser_automation_studio.v1.WorkflowDefinitionV2 (use_proto_names: true).",
		"",
		workflowJSONStartMarker,
		`{"nodes":[],"edges":[]}`,
		workflowJSONEndMarker,
		"",
		"User request:",
		userPrompt,
	}
	if strings.TrimSpace(currentJSON) != "" {
		lines = append(lines,
			"",
			"Current workflow definition (protojson):",
			currentJSON,
			"",
			"Modify the current definition to satisfy the user request.",
		)
	}
	return strings.Join(lines, "\n")
}

func extractBetweenMarkers(text, start, end string) string {
	startIdx := strings.Index(text, start)
	if startIdx < 0 {
		return ""
	}
	startIdx += len(start)
	endIdx := strings.Index(text[startIdx:], end)
	if endIdx < 0 {
		return ""
	}
	return strings.TrimSpace(text[startIdx : startIdx+endIdx])
}
