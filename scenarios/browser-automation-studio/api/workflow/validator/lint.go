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

	// Patterns for detecting unresolved tokens that should have been substituted
	// NOTE: @selector/ is NOT included here because selectors are resolved at compile time
	// by BAS's compiler (compiler.go:resolveSelectors), not at workflow definition time.
	// This allows test-genie to pass @selector/ tokens through to BAS for native resolution.
	unresolvedTokenPatterns = []*unresolvedPattern{
		{regexp.MustCompile(`@fixture/[A-Za-z0-9_.-]+`), "fixture reference", "WF_UNRESOLVED_FIXTURE"},
		{regexp.MustCompile(`@seed/[A-Za-z0-9_.-]+`), "seed reference", "WF_UNRESOLVED_SEED"},
		{regexp.MustCompile(`\$\{[A-Za-z0-9_]+\}`), "placeholder", "WF_UNRESOLVED_PLACEHOLDER"},
		{regexp.MustCompile(`\{\{[A-Za-z0-9_]+\}\}`), "template variable", "WF_UNRESOLVED_TEMPLATE"},
	}
)

type unresolvedPattern struct {
	pattern *regexp.Regexp
	name    string
	code    string
}

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

// Node-specific lint functions moved to node_linters.go

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

// runResolvedLint checks for unresolved tokens that should have been substituted.
// This is used by the validate-resolved endpoint to catch resolution failures.
func runResolvedLint(definition map[string]any) []Issue {
	var issues []Issue
	nodes := toSlice(definition["nodes"])

	for idx, rawNode := range nodes {
		nodeMap, ok := toMap(rawNode)
		if !ok {
			continue
		}

		nodeID := getString(nodeMap["id"])
		nodeType := getString(nodeMap["type"])

		dataMap := map[string]any{}
		if rawData, ok := nodeMap["data"]; ok {
			if coerced, ok := toMap(rawData); ok {
				dataMap = coerced
			}
		}

		// Check all string fields in data for unresolved tokens
		for field, value := range dataMap {
			strValue := getString(value)
			if strValue == "" {
				continue
			}

			for _, up := range unresolvedTokenPatterns {
				matches := up.pattern.FindAllString(strValue, -1)
				for _, match := range matches {
					issues = append(issues, Issue{
						Severity: SeverityError,
						Code:     up.code,
						Message:  fmt.Sprintf("Unresolved %s '%s' in field '%s'", up.name, match, field),
						NodeID:   nodeID,
						NodeType: nodeType,
						Field:    field,
						Pointer:  fmt.Sprintf("/nodes/%d/data/%s", idx, field),
						Hint:     fmt.Sprintf("Ensure the %s is resolved before execution", up.name),
					})
				}
			}
		}

		// Special check for navigate nodes with destinationType=scenario
		// After resolution, the URL should be set and destinationType should be 'url'
		if nodeType == "navigate" {
			destType := strings.ToLower(strings.TrimSpace(getString(dataMap["destinationType"])))
			if destType == "scenario" {
				scenario := getString(dataMap["scenario"])
				issues = append(issues, Issue{
					Severity: SeverityError,
					Code:     "WF_UNRESOLVED_SCENARIO_URL",
					Message:  fmt.Sprintf("Navigate node still has destinationType=scenario (scenario: %s); URL should be resolved", scenario),
					NodeID:   nodeID,
					NodeType: nodeType,
					Field:    "destinationType",
					Pointer:  fmt.Sprintf("/nodes/%d/data/destinationType", idx),
					Hint:     "Scenario URL resolution should convert destinationType to 'url' with the resolved URL",
				})
			}
		}
	}

	return issues
}
