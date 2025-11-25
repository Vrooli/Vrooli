package validator

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	selectorOnce      sync.Once
	selectorValues    map[string]struct{}
	selectorSource    string
	dataTestIDPattern = regexp.MustCompile(`(?i)data-testid\s*=\s*(?:"([^"]+)"|'([^']+)')`)
)

type nodeRule struct {
	requiredData []string
	requireOneOf [][]string
	custom       func(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue)
}

var nodeRules = map[string]nodeRule{
	"click": {
		requiredData: []string{"selector"},
	},
	"hover": {
		requiredData: []string{"selector"},
	},
	"dragDrop": {
		requireOneOf: [][]string{{"sourceSelector", "sourceCoordinates"}},
		requiredData: []string{"targetSelector"},
	},
	"focus": {
		requiredData: []string{"selector"},
	},
	"blur": {
		requiredData: []string{"selector"},
	},
	"type": {
		requiredData: []string{"selector"},
		custom:       lintTypeNode,
	},
	"keyboard": {
		custom: lintKeyboardNode,
	},
	"shortcut": {
		requiredData: []string{"keys"},
	},
	"select": {
		requiredData: []string{"selector"},
		requireOneOf: [][]string{
			{"optionText", "optionValue", "optionIndex"},
		},
	},
	"screenshot": {
		custom: lintScreenshotNode,
	},
	"wait": {
		custom: lintWaitNode,
	},
	"extract": {
		requiredData: []string{"selector"},
	},
	"assert": {
		requiredData: []string{"selector", "assertMode"},
		custom:       lintAssertNode,
	},
	"navigate": {
		requiredData: []string{"destinationType"},
		custom:       lintNavigateNode,
	},
	"loop": {
		requiredData: []string{"loopType"},
	},
	"evaluate": {
		requireOneOf: [][]string{{"expression", "script", "code"}},
	},
	"subflow": {
		custom: lintSubflowNode,
	},
}

func runLint(definition map[string]any) (Stats, []Issue, []Issue) {
	var errorsList []Issue
	var warningsList []Issue
	stats := Stats{}
	allowedSelectors := loadSelectorSet()

	nodes := toSlice(definition["nodes"])
	edges := toSlice(definition["edges"])
	stats.NodeCount = len(nodes)
	stats.EdgeCount = len(edges)

	if stats.NodeCount == 0 {
		errorsList = append(errorsList, Issue{
			Severity: SeverityError,
			Code:     "WF_NODE_EMPTY",
			Message:  "Workflow must contain at least one node",
			Pointer:  "/nodes",
		})
	}

	if stats.EdgeCount == 0 {
		warningsList = append(warningsList, Issue{
			Severity: SeverityWarning,
			Code:     "WF_EDGE_EMPTY",
			Message:  "Workflow does not contain any edges; execution order may be undefined",
			Pointer:  "/edges",
		})
	}

	nodeIndex := map[string]map[string]any{}
	selectorSet := map[string]struct{}{}

	for idx, rawNode := range nodes {
		nodeMap, ok := toMap(rawNode)
		if !ok {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_NODE_SHAPE",
				Message:  fmt.Sprintf("Node %d is not an object", idx),
				Pointer:  fmt.Sprintf("/nodes/%d", idx),
			})
			continue
		}

		nodeID := getString(nodeMap["id"])
		nodeType := strings.TrimSpace(getString(nodeMap["type"]))
		if nodeID == "" {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_NODE_ID_MISSING",
				Message:  fmt.Sprintf("Node %d is missing an id", idx),
				Pointer:  fmt.Sprintf("/nodes/%d/id", idx),
				NodeType: nodeType,
			})
		} else {
			if _, exists := nodeIndex[nodeID]; exists {
				errorsList = append(errorsList, Issue{
					Severity: SeverityError,
					Code:     "WF_NODE_ID_DUPLICATE",
					Message:  fmt.Sprintf("Node id '%s' is duplicated", nodeID),
					NodeID:   nodeID,
					NodeType: nodeType,
					Pointer:  fmt.Sprintf("/nodes/%d/id", idx),
				})
			}
			nodeIndex[nodeID] = nodeMap
		}

		if nodeType == "" {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_NODE_TYPE_MISSING",
				Message:  fmt.Sprintf("Node '%s' is missing a type", nodeID),
				NodeID:   nodeID,
				Pointer:  fmt.Sprintf("/nodes/%d/type", idx),
			})
			continue
		}

		dataMap := map[string]any{}
		if rawData, ok := nodeMap["data"]; ok {
			if coerced, ok := toMap(rawData); ok {
				dataMap = coerced
			}
		}

		selectorValue := strings.TrimSpace(getString(dataMap["selector"]))
		if selectorValue != "" {
			stats.SelectorCount++
			selectorSet[selectorValue] = struct{}{}
			warningsList = append(warningsList, lintSelectorValue(selectorValue, fmt.Sprintf("/nodes/%d/data/selector", idx), nodeID, nodeType, allowedSelectors)...)
		}
		if frameSelector := strings.TrimSpace(getString(dataMap["frameSelector"])); frameSelector != "" {
			stats.SelectorCount++
			selectorSet[frameSelector] = struct{}{}
			warningsList = append(warningsList, lintSelectorValue(frameSelector, fmt.Sprintf("/nodes/%d/data/frameSelector", idx), nodeID, nodeType, allowedSelectors)...)
		}

		if strings.ToLower(strings.TrimSpace(getString(dataMap["waitType"]))) == "element" {
			stats.ElementWaitCount++
		}

		if rule, ok := nodeRules[nodeType]; ok {
			errorsList = append(errorsList, applyNodeRule(rule, nodeID, nodeType, idx, dataMap)...)
			if rule.custom != nil {
				ruleErrors, ruleWarnings := rule.custom(nodeMap, dataMap, idx)
				errorsList = append(errorsList, ruleErrors...)
				warningsList = append(warningsList, ruleWarnings...)
			}
		}

		if label := strings.TrimSpace(getString(dataMap["label"])); label == "" {
			warningsList = append(warningsList, Issue{
				Severity: SeverityWarning,
				Code:     "WF_NODE_LABEL_MISSING",
				Message:  "Node is missing a label; consider adding one for readability",
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  fmt.Sprintf("/nodes/%d/data/label", idx),
			})
		}
	}

	stats.UniqueSelectorCount = len(selectorSet)

	for idx, rawEdge := range edges {
		edgeMap, ok := toMap(rawEdge)
		if !ok {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_SHAPE",
				Message:  fmt.Sprintf("Edge %d is not an object", idx),
				Pointer:  fmt.Sprintf("/edges/%d", idx),
			})
			continue
		}
		source := strings.TrimSpace(getString(edgeMap["source"]))
		target := strings.TrimSpace(getString(edgeMap["target"]))
		if source == "" || target == "" {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_ENDPOINT_MISSING",
				Message:  fmt.Sprintf("Edge %d must define a source and target", idx),
				Pointer:  fmt.Sprintf("/edges/%d", idx),
			})
			continue
		}
		if source == target {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_CYCLE_SELF",
				Message:  fmt.Sprintf("Edge %d connects node '%s' to itself", idx, source),
				Pointer:  fmt.Sprintf("/edges/%d", idx),
				NodeID:   source,
			})
		}
		if _, ok := nodeIndex[source]; !ok {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_SOURCE_UNKNOWN",
				Message:  fmt.Sprintf("Edge %d references unknown source node '%s'", idx, source),
				Pointer:  fmt.Sprintf("/edges/%d/source", idx),
			})
		}
		if _, ok := nodeIndex[target]; !ok {
			errorsList = append(errorsList, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_TARGET_UNKNOWN",
				Message:  fmt.Sprintf("Edge %d references unknown target node '%s'", idx, target),
				Pointer:  fmt.Sprintf("/edges/%d/target", idx),
			})
		}
	}

	if _, ok := toMap(definition["metadata"]); ok {
		stats.HasMetadata = true
	}

	if settings, ok := toMap(definition["settings"]); ok {
		if viewport, ok := toMap(settings["executionViewport"]); ok {
			stats.HasExecutionViewport = true
			width := toFloat64(viewport["width"])
			height := toFloat64(viewport["height"])
			if width < 200 || height < 200 {
				warningsList = append(warningsList, Issue{
					Severity: SeverityWarning,
					Code:     "WF_VIEWPORT_SMALL",
					Message:  "Execution viewport is unusually small; consider using at least 200x200",
					Pointer:  "/settings/executionViewport",
				})
			}
		}
	}

	if stats.NodeCount > 0 && stats.EdgeCount < stats.NodeCount-1 {
		warningsList = append(warningsList, Issue{
			Severity: SeverityWarning,
			Code:     "WF_EDGE_SPARSITY",
			Message:  "Workflow has fewer edges than nodes; verify that execution order is fully connected",
			Pointer:  "/edges",
		})
	}

	if stats.UniqueSelectorCount > 0 && stats.SelectorCount != stats.UniqueSelectorCount {
		warningsList = append(warningsList, Issue{
			Severity: SeverityWarning,
			Code:     "WF_SELECTOR_DUPLICATES",
			Message:  "Multiple nodes reuse the same selector; confirm this is intentional",
			Pointer:  "/nodes",
		})
	}

	return stats, errorsList, warningsList
}

func applyNodeRule(rule nodeRule, nodeID, nodeType string, idx int, data map[string]any) []Issue {
	var issues []Issue
	for _, field := range rule.requiredData {
		if strings.TrimSpace(getString(data[field])) == "" {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Code:     "WF_NODE_FIELD_REQUIRED",
				Message:  fmt.Sprintf("Node '%s' (%s) requires field '%s'", nodeID, nodeType, field),
				NodeID:   nodeID,
				NodeType: nodeType,
				Field:    field,
				Pointer:  fmt.Sprintf("/nodes/%d/data/%s", idx, field),
			})
		}
	}
	for _, group := range rule.requireOneOf {
		if len(group) == 0 {
			continue
		}
		if !anyFieldPresent(data, group) {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Code:     "WF_NODE_FIELD_ONE_OF",
				Message:  fmt.Sprintf("Node '%s' (%s) must define one of: %s", nodeID, nodeType, strings.Join(group, ", ")),
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  fmt.Sprintf("/nodes/%d/data", idx),
			})
		}
	}
	return issues
}

func loadSelectorSet() map[string]struct{} {
	selectorOnce.Do(func() {
		manifest, source, err := loadSelectorManifest("")
		if err != nil {
			selectorValues = nil
			return
		}
		selectorSource = source
		selectorValues = make(map[string]struct{}, len(manifest))
		for testID := range manifest {
			selectorValues[testID] = struct{}{}
		}
	})
	return selectorValues
}

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

func lintSubflowNode(node map[string]any, data map[string]any, idx int) ([]Issue, []Issue) {
	return lintSubflowLikeNode(node, data, idx)
}

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

func anyFieldPresent(data map[string]any, fields []string) bool {
	for _, field := range fields {
		if strings.TrimSpace(getString(data[field])) != "" {
			return true
		}
	}
	return false
}

func toSlice(value any) []any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []any:
		return typed
	case []map[string]any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = typed[i]
		}
		return result
	default:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return nil
		}
		var arr []any
		if err := json.Unmarshal(bytes, &arr); err != nil {
			return nil
		}
		return arr
	}
}

func toMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case nil:
		return nil, false
	default:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return nil, false
		}
		var result map[string]any
		if err := json.Unmarshal(bytes, &result); err != nil {
			return nil, false
		}
		return result, true
	}
}

func getString(value any) string {
	return getStringOr(value, "")
}

func getStringOr(value any, fallback string) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case float64:
		if math.Mod(typed, 1) == 0 {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%f", typed)
	case int, int64, uint64:
		return fmt.Sprintf("%v", typed)
	default:
		return fallback
	}
}

func getBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		trimmed := strings.ToLower(strings.TrimSpace(typed))
		return trimmed == "true" || trimmed == "1" || trimmed == "yes"
	case float64:
		return typed != 0
	case int:
		return typed != 0
	case int64:
		return typed != 0
	default:
		return false
	}
}

func toFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		v, _ := typed.Float64()
		return v
	default:
		return 0
	}
}

// SortIssues returns a stable ordering (useful for CLI JSON output).
func SortIssues(issues []Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Severity != issues[j].Severity {
			return issues[i].Severity < issues[j].Severity
		}
		if issues[i].Code != issues[j].Code {
			return issues[i].Code < issues[j].Code
		}
		if issues[i].NodeID != issues[j].NodeID {
			return issues[i].NodeID < issues[j].NodeID
		}
		return issues[i].Message < issues[j].Message
	})
}
