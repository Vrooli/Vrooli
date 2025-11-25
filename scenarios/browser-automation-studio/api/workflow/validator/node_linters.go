package validator

import (
	"fmt"
	"strings"
)

// Node-specific lint functions for workflow validation.
// Each function validates a specific node type and returns errors and warnings.

// lintWaitNode validates wait node configuration.
func lintWaitNode(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue) {
	waitType := strings.ToLower(strings.TrimSpace(getString(data["waitType"])))
	var errorsList []Issue
	var warningsList []Issue
	nodeID := getString(node["id"])
	nodeType := getString(node["type"])

	switch waitType {
	case "element":
		if strings.TrimSpace(getString(data["selector"])) == "" {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_WAIT_SELECTOR_REQUIRED",
				Message:  "Element wait must define a selector",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  fmt.Sprintf("/nodes/%d/data/selector", idx),
			})
		}
	case "duration", "delay", "time":
		duration := toFloat64(data["durationMs"])
		if duration <= 0 {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_WAIT_DURATION_REQUIRED",
				Message:  "Duration wait must define durationMs > 0",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  fmt.Sprintf("/nodes/%d/data/durationMs", idx),
			})
		} else if duration > 60000 {
			warningsList = append(warningsList, Issue{
				Severity: SeverityWarning,
				Code:     "WF_WAIT_DURATION_LONG",
				Message:  "Wait duration exceeds 60s; consider shorter waits with assertions",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  fmt.Sprintf("/nodes/%d/data/durationMs", idx),
			})
		}
	default:
		warningsList = append(warningsList, Issue{
			Severity: SeverityWarning,
			Code:     "WF_WAIT_TYPE_UNKNOWN",
			Message:  "Wait node does not specify a known waitType; defaulting to duration",
			NodeID:   nodeID,
			NodeType: nodeType,
			Pointer:  fmt.Sprintf("/nodes/%d/data/waitType", idx),
		})
	}
	return errorsList, warningsList
}

// lintSelectorValue validates selector syntax and checks against registered data-testid values.
func lintSelectorValue(selector, pointer, nodeID, nodeType string, registry map[string]struct{}) []Issue {
	if len(registry) == 0 {
		return nil
	}
	matches := dataTestIDPattern.FindAllStringSubmatch(selector, -1)
	if len(matches) == 0 {
		return nil
	}
	var issues []Issue
	for _, match := range matches {
		value := match[1]
		if value == "" && len(match) > 2 {
			value = match[2]
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := registry[value]; !ok {
			location := selectorSource
			if location == "" {
				location = "ui/src/consts/" + manifestFilename
			}
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Code:     "WF_SELECTOR_UNKNOWN_TESTID",
				Message:  "Selector references data-testid '" + value + "' which is not registered in " + location,
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  pointer,
				Hint:     "Define the selector in " + location + " or update the workflow",
			})
		}
	}
	return issues
}

// lintKeyboardNode validates keyboard input node configuration.
func lintKeyboardNode(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue) {
	var warnings []Issue
	nodeID := getString(node["id"])
	nodeType := getString(node["type"])
	pointer := fmt.Sprintf("/nodes/%d/data", idx)

	if len(toSlice(data["keys"])) > 0 {
		return nil, nil
	}
	if strings.TrimSpace(getString(data["sequence"])) != "" {
		return nil, nil
	}

	if key := strings.TrimSpace(getString(data["key"])); key != "" {
		warnings = append(warnings, Issue{
			Severity: SeverityWarning,
			Code:     "WF_KEYBOARD_KEY_FIELD",
			Message:  "Keyboard node uses deprecated 'key' field; prefer 'keys' array",
			NodeID:   nodeID,
			NodeType: nodeType,
			Pointer:  pointer + "/key",
		})
		return nil, warnings
	}

	return []Issue{{
		Severity: SeverityError,
		Code:     "WF_KEYBOARD_INPUT_REQUIRED",
		Message:  "Keyboard node must define keys[] or sequence",
		NodeID:   nodeID,
		NodeType: nodeType,
		Pointer:  pointer,
	}}, nil
}

// lintTypeNode validates type/input node configuration.
func lintTypeNode(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue) {
	nodeID := getString(node["id"])
	nodeType := getString(node["type"])
	pointer := fmt.Sprintf("/nodes/%d/data", idx)

	if _, ok := data["value"]; ok {
		return nil, nil
	}
	if _, ok := data["text"]; ok {
		return nil, nil
	}
	if strings.TrimSpace(getString(data["variable"])) != "" {
		return nil, nil
	}

	return []Issue{{
		Severity: SeverityError,
		Code:     "WF_TYPE_INPUT_REQUIRED",
		Message:  "Type node must define text, value, or variable",
		NodeID:   nodeID,
		NodeType: nodeType,
		Pointer:  pointer,
	}}, nil
}

// lintAssertNode validates assertion node configuration.
func lintAssertNode(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue) {
	var errorsList []Issue
	nodeID := getString(node["id"])
	nodeType := getString(node["type"])
	mode := strings.ToLower(strings.TrimSpace(getString(data["assertMode"])))
	pointer := fmt.Sprintf("/nodes/%d/data", idx)

	needExpected := map[string]bool{
		"text_equals":        true,
		"text_contains":      true,
		"attribute_equals":   true,
		"attribute_contains": true,
	}
	if needExpected[mode] && strings.TrimSpace(getString(data["expectedValue"])) == "" {
		errorsList = append(errorsList, Issue{
			Severity: SeverityError,
			Code:     "WF_ASSERT_EXPECTED_VALUE",
			Message:  "Assertion requires an expectedValue",
			NodeID:   nodeID,
			NodeType: nodeType,
			Pointer:  pointer + "/expectedValue",
		})
	}
	if strings.HasPrefix(mode, "attribute") && strings.TrimSpace(getString(data["attributeName"])) == "" {
		errorsList = append(errorsList, Issue{
			Severity: SeverityError,
			Code:     "WF_ASSERT_ATTRIBUTE_NAME",
			Message:  "Attribute assertions require attributeName",
			NodeID:   nodeID,
			NodeType: nodeType,
			Pointer:  pointer + "/attributeName",
		})
	}
	return errorsList, nil
}

// lintSubflowNode validates subflow node configuration.
func lintSubflowNode(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue) {
	return lintSubflowLikeNode(node, data, idx)
}

// lintSubflowLikeNode validates subflow-like nodes (subflow, call, etc.).
func lintSubflowLikeNode(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue) {
	var errorsList []Issue
	var warningsList []Issue
	nodeID := getString(node["id"])
	nodeType := getString(node["type"])
	pointer := fmt.Sprintf("/nodes/%d/data", idx)

	workflowID := strings.TrimSpace(getString(data["workflowId"]))
	inlineDef, hasInline := toMap(data["workflowDefinition"])

	if workflowID == "" && (!hasInline || len(inlineDef) == 0) {
		errorsList = append(errorsList, Issue{
			Severity: SeverityError,
			Code:     "WF_SUBFLOW_TARGET",
			Message:  "subflow node must define workflowId or workflowDefinition",
			NodeID:   nodeID,
			NodeType: nodeType,
			Pointer:  pointer,
		})
		return errorsList, warningsList
	}

	if hasInline && len(inlineDef) > 0 {
		nodes := toSlice(inlineDef["nodes"])
		if len(nodes) == 0 {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_SUBFLOW_INLINE_NODES",
				Message:  "subflow workflowDefinition must include at least one node",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  pointer + "/workflowDefinition/nodes",
			})
		}
		if _, ok := inlineDef["edges"]; !ok {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_SUBFLOW_INLINE_EDGES",
				Message:  "subflow workflowDefinition must include edges even if empty",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  pointer + "/workflowDefinition/edges",
			})
		}
	}

	return errorsList, warningsList
}

// lintNavigateNode validates navigation node configuration.
func lintNavigateNode(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue) {
	var errorsList []Issue
	var warningsList []Issue
	nodeID := getString(node["id"])
	nodeType := getString(node["type"])
	pointerBase := fmt.Sprintf("/nodes/%d/data", idx)
	destType := strings.ToLower(strings.TrimSpace(getString(data["destinationType"])))
	switch destType {
	case "url":
		if strings.TrimSpace(getString(data["url"])) == "" {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_NAVIGATE_URL_REQUIRED",
				Message:  "Navigate node must supply a url when destinationType=url",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  pointerBase + "/url",
			})
		}
	case "scenario":
		if strings.TrimSpace(getString(data["scenario"])) == "" {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_NAVIGATE_SCENARIO_REQUIRED",
				Message:  "Scenario navigation requires scenario name",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  pointerBase + "/scenario",
			})
		}
		if strings.TrimSpace(getString(data["scenarioPath"])) == "" {
			warningsList = append(warningsList, Issue{
				Severity: SeverityWarning,
				Code:     "WF_NAVIGATE_SCENARIO_PATH",
				Message:  "Scenario navigation missing scenarioPath; default '/' will be used",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  pointerBase + "/scenarioPath",
			})
		}
	default:
		warningsList = append(warningsList, Issue{
			Severity: SeverityWarning,
			Code:     "WF_NAVIGATE_DESTINATION_UNKNOWN",
			Message:  "Destination type not recognized; expected 'url' or 'scenario'",
			NodeID:   nodeID,
			NodeType: nodeType,
			Pointer:  pointerBase + "/destinationType",
		})
	}
	return errorsList, warningsList
}

// lintScreenshotNode validates screenshot node configuration.
func lintScreenshotNode(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue) {
	var errorsList []Issue
	nodeID := getString(node["id"])
	nodeType := getString(node["type"])
	hasSelector := strings.TrimSpace(getString(data["selector"])) != ""
	fullPage := getBool(data["fullPage"])
	if !hasSelector && !fullPage {
		errorsList = append(errorsList, Issue{
			Severity: SeverityError,
			Code:     "WF_SCREENSHOT_TARGET_REQUIRED",
			Message:  "Screenshot node must set fullPage=true or provide a selector",
			NodeID:   nodeID,
			NodeType: nodeType,
			Pointer:  fmt.Sprintf("/nodes/%d/data", idx),
		})
	}
	return errorsList, nil
}
