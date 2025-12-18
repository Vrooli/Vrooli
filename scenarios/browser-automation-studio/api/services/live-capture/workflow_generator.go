// Package livecapture provides business logic for live capture mode functionality.
// This includes converting recorded actions to workflows, action merging, and smart wait insertion.
package livecapture

import (
	"fmt"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/actions"
)

// WorkflowGenerator converts recorded actions into workflow definitions.
type WorkflowGenerator struct{}

// NewWorkflowGenerator creates a new workflow generator.
func NewWorkflowGenerator() *WorkflowGenerator {
	return &WorkflowGenerator{}
}

// GenerateWorkflow converts recorded actions to a workflow flow definition.
// It applies action merging and inserts smart wait nodes to improve reliability.
func (g *WorkflowGenerator) GenerateWorkflow(actions []RecordedAction) map[string]interface{} {
	// First, merge consecutive actions for cleaner workflows
	mergedActions := MergeConsecutiveActions(actions)

	// Insert smart wait nodes between actions that need them
	nodes, edges := insertSmartWaits(mergedActions)

	return map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}
}

// MergeConsecutiveActions optimizes recorded actions by merging:
// - Consecutive type actions on the same selector (text is concatenated)
// - Consecutive scroll actions (uses final scroll position)
// - Removes focus events that precede type events on the same element
func MergeConsecutiveActions(actions []RecordedAction) []RecordedAction {
	if len(actions) <= 1 {
		return actions
	}

	merged := make([]RecordedAction, 0, len(actions))

	for i := 0; i < len(actions); i++ {
		action := actions[i]

		// Skip focus events that are immediately followed by type on the same element
		if action.ActionType == "focus" && i+1 < len(actions) {
			next := actions[i+1]
			if next.ActionType == "type" && selectorsMatch(action.Selector, next.Selector) {
				continue // Skip this focus event
			}
		}

		// Merge consecutive type actions on same selector
		if action.ActionType == "type" && action.Selector != nil {
			mergedText := ""
			if action.Payload != nil {
				if text, ok := action.Payload["text"].(string); ok {
					mergedText = text
				}
			}

			// Look ahead for more type actions on same element
			for i+1 < len(actions) {
				next := actions[i+1]
				if next.ActionType != "type" || !selectorsMatch(action.Selector, next.Selector) {
					break
				}
				// Merge the text
				if next.Payload != nil {
					if text, ok := next.Payload["text"].(string); ok {
						mergedText += text
					}
				}
				i++ // Skip this action, we've merged it
			}

			// Update the action with merged text
			if mergedText != "" {
				if action.Payload == nil {
					action.Payload = make(map[string]interface{})
				}
				action.Payload["text"] = mergedText
			}
		}

		// Merge consecutive scroll actions
		if action.ActionType == "scroll" {
			var finalScrollY float64
			if action.Payload != nil {
				if y, ok := action.Payload["scrollY"].(float64); ok {
					finalScrollY = y
				}
			}

			// Look ahead for more scroll actions
			for i+1 < len(actions) {
				next := actions[i+1]
				if next.ActionType != "scroll" {
					break
				}
				// Use the final scroll position
				if next.Payload != nil {
					if y, ok := next.Payload["scrollY"].(float64); ok {
						finalScrollY = y
					}
				}
				i++ // Skip this action, we've merged it
			}

			// Update the action with final scroll position
			if action.Payload == nil {
				action.Payload = make(map[string]interface{})
			}
			action.Payload["scrollY"] = finalScrollY
		}

		merged = append(merged, action)
	}

	return merged
}

// selectorsMatch checks if two SelectorSets refer to the same element
func selectorsMatch(a, b *SelectorSet) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Primary == b.Primary
}

// ApplyActionRange returns the requested action subset, clamping indices to the available actions.
func ApplyActionRange(actions []RecordedAction, start, end int) []RecordedAction {
	if len(actions) == 0 {
		return actions
	}

	if start < 0 {
		start = 0
	}
	if end >= len(actions) {
		end = len(actions) - 1
	}
	if start <= end && start < len(actions) {
		return actions[start : end+1]
	}
	return actions
}

// nodeTypeToV2ActionType maps V1 node type strings to V2 ACTION_TYPE_ enum values.
func nodeTypeToV2ActionType(nodeType string) string {
	switch nodeType {
	case "navigate":
		return "ACTION_TYPE_NAVIGATE"
	case "click":
		return "ACTION_TYPE_CLICK"
	case "type":
		return "ACTION_TYPE_INPUT"
	case "wait":
		return "ACTION_TYPE_WAIT"
	case "assert":
		return "ACTION_TYPE_ASSERT"
	case "scroll":
		return "ACTION_TYPE_SCROLL"
	case "select":
		return "ACTION_TYPE_SELECT"
	case "evaluate":
		return "ACTION_TYPE_EVALUATE"
	case "keyboard":
		return "ACTION_TYPE_KEYBOARD"
	case "hover":
		return "ACTION_TYPE_HOVER"
	case "screenshot":
		return "ACTION_TYPE_SCREENSHOT"
	case "focus":
		return "ACTION_TYPE_FOCUS"
	case "blur":
		return "ACTION_TYPE_BLUR"
	default:
		return "ACTION_TYPE_UNSPECIFIED"
	}
}

// nodeTypeToV2ParamKey maps V1 node type to the V2 action param key.
func nodeTypeToV2ParamKey(nodeType string) string {
	switch nodeType {
	case "type":
		return "input" // V2 uses "input" for type actions
	case "select":
		return "select_option"
	default:
		return nodeType
	}
}

// mapActionToNode converts a single recorded action to a workflow node in V2 format.
// Uses the action registry to look up type-specific configuration.
func mapActionToNode(action RecordedAction, nodeID string, index int) map[string]interface{} {
	// Calculate position (vertical layout)
	posX := 250.0
	posY := float64(100 + index*120)

	// Get action configuration from registry
	cfg := GetActionNodeConfig(actions.ActionType(action.ActionType))

	// Build node data using registry builder
	data, _ := cfg.BuildNode(action)

	// Generate label using registry label generator
	label := cfg.GenerateLabel(action)

	// Build V2 action definition
	v2ActionType := nodeTypeToV2ActionType(cfg.NodeType)
	v2ParamKey := nodeTypeToV2ParamKey(cfg.NodeType)

	// Build typed params for the action
	actionParams := buildV2ActionParams(cfg.NodeType, action, data)

	actionDef := map[string]interface{}{
		"type": v2ActionType,
		"metadata": map[string]interface{}{
			"label": label,
		},
	}
	// Add the typed params under the appropriate key
	if len(actionParams) > 0 {
		actionDef[v2ParamKey] = actionParams
	}

	node := map[string]interface{}{
		"id":     nodeID,
		"action": actionDef,
		"position": map[string]interface{}{
			"x": posX,
			"y": posY,
		},
	}

	return node
}

// buildV2ActionParams builds the typed params for a V2 action.
func buildV2ActionParams(nodeType string, action RecordedAction, data map[string]interface{}) map[string]interface{} {
	params := make(map[string]interface{})

	switch nodeType {
	case "navigate":
		if url := action.URL; url != "" {
			params["url"] = url
		}
		if waitFor, ok := data["waitForSelector"]; ok {
			params["wait_for_selector"] = waitFor
		}
	case "click":
		if action.Selector != nil {
			params["selector"] = action.Selector.Primary
		}
		if btn, ok := data["button"]; ok {
			params["button"] = btn
		}
		if count, ok := data["clickCount"]; ok {
			params["click_count"] = count
		}
	case "type":
		if action.Selector != nil {
			params["selector"] = action.Selector.Primary
		}
		if text, ok := data["text"].(string); ok {
			params["value"] = text
		}
	case "wait":
		if action.Selector != nil {
			params["selector"] = action.Selector.Primary
		}
		if timeout, ok := data["timeoutMs"]; ok {
			params["timeout_ms"] = timeout
		}
	case "scroll":
		if action.Selector != nil {
			params["selector"] = action.Selector.Primary
		}
		if y, ok := data["y"]; ok {
			params["y"] = y
		}
	case "select":
		if action.Selector != nil {
			params["selector"] = action.Selector.Primary
		}
		if val, ok := data["value"]; ok {
			params["value"] = val
		}
	case "hover":
		if action.Selector != nil {
			params["selector"] = action.Selector.Primary
		}
	case "keyboard":
		if key, ok := data["key"]; ok {
			params["key"] = key
		}
	case "screenshot":
		if name, ok := data["name"]; ok {
			params["name"] = name
		}
	case "assert":
		if action.Selector != nil {
			params["selector"] = action.Selector.Primary
		}
		// Copy all payload data as assert params
		for k, v := range data {
			if k != "selector" && k != "label" {
				params[k] = v
			}
		}
	default:
		// For unknown types, copy data as params
		if action.Selector != nil {
			params["selector"] = action.Selector.Primary
		}
	}

	return params
}

// WaitTemplate describes a wait node to be inserted between actions.
type WaitTemplate struct {
	WaitType  string // "selector" or "timeout"
	Selector  string // For selector waits
	TimeoutMs int    // Timeout for selector waits, or duration for timeout waits
	Label     string // Human-readable label
}

// analyzeTransitionForWait examines two consecutive actions and determines
// if a wait node should be inserted between them.
// Returns nil if no wait is needed.
func analyzeTransitionForWait(current, next RecordedAction) *WaitTemplate {
	// Check if the next action needs its selector to exist (uses action registry)
	if NeedsSelectorWait(next.ActionType) && next.Selector != nil && next.Selector.Primary != "" {
		// If current action might trigger DOM changes, add a wait (uses action registry)
		triggersChanges := TriggersDOMChanges(current.ActionType)

		// Check for URL change (indicates navigation happened)
		urlChanged := current.URL != next.URL

		// Check for significant time gap (>500ms suggests async activity)
		var timeDiff int64
		if current.Timestamp != "" && next.Timestamp != "" {
			currentTime, err1 := time.Parse(time.RFC3339Nano, current.Timestamp)
			nextTime, err2 := time.Parse(time.RFC3339Nano, next.Timestamp)
			if err1 == nil && err2 == nil {
				timeDiff = nextTime.Sub(currentTime).Milliseconds()
			}
		}
		significantGap := timeDiff > 500

		// Insert wait if any condition is met
		if triggersChanges || urlChanged || significantGap {
			label := fmt.Sprintf("Wait for %s", describeElement(next))
			return &WaitTemplate{
				WaitType:  "selector",
				Selector:  next.Selector.Primary,
				TimeoutMs: 10000, // 10 second default timeout
				Label:     label,
			}
		}
	}

	// Check for large time gaps that suggest async operations even without selector needs
	if current.Timestamp != "" && next.Timestamp != "" {
		currentTime, err1 := time.Parse(time.RFC3339Nano, current.Timestamp)
		nextTime, err2 := time.Parse(time.RFC3339Nano, next.Timestamp)
		if err1 == nil && err2 == nil {
			timeDiff := nextTime.Sub(currentTime).Milliseconds()
			// If gap > 2 seconds, insert a proportional wait (capped at 5 seconds)
			if timeDiff > 2000 {
				waitDuration := timeDiff / 2 // Wait for half the observed gap
				if waitDuration > 5000 {
					waitDuration = 5000
				}
				return &WaitTemplate{
					WaitType:  "timeout",
					TimeoutMs: int(waitDuration),
					Label:     "Wait for page to stabilize",
				}
			}
		}
	}

	return nil
}

// describeElement creates a human-readable description of an element for labels.
func describeElement(action RecordedAction) string {
	if action.ElementMeta != nil {
		if action.ElementMeta.InnerText != "" {
			text := truncateString(action.ElementMeta.InnerText, 15)
			return fmt.Sprintf("\"%s\"", text)
		}
		if action.ElementMeta.AriaLabel != "" {
			return action.ElementMeta.AriaLabel
		}
		if action.ElementMeta.TagName != "" {
			return action.ElementMeta.TagName
		}
	}
	return "element"
}

// createWaitNode generates a workflow wait node from a WaitTemplate in V2 format.
func createWaitNode(template *WaitTemplate, nodeID string, posY float64) map[string]interface{} {
	// Build V2 wait params
	waitParams := map[string]interface{}{
		"timeout_ms": template.TimeoutMs,
	}

	if template.WaitType == "selector" && template.Selector != "" {
		waitParams["selector"] = template.Selector
		waitParams["state"] = "visible" // Default to waiting for visible state
	} else if template.WaitType == "timeout" {
		waitParams["duration_ms"] = template.TimeoutMs
	}

	// Build V2 action definition
	actionDef := map[string]interface{}{
		"type": "ACTION_TYPE_WAIT",
		"wait": waitParams,
		"metadata": map[string]interface{}{
			"label": template.Label,
		},
	}

	return map[string]interface{}{
		"id":     nodeID,
		"action": actionDef,
		"position": map[string]interface{}{
			"x": 250.0,
			"y": posY,
		},
	}
}

// insertSmartWaits analyzes action transitions and inserts wait nodes where needed.
// This improves reliability of recorded workflows by ensuring elements exist before interaction.
func insertSmartWaits(actions []RecordedAction) ([]map[string]interface{}, []map[string]interface{}) {
	if len(actions) == 0 {
		return nil, nil
	}

	nodes := make([]map[string]interface{}, 0, len(actions)*2)
	edges := make([]map[string]interface{}, 0, len(actions)*2)

	var prevNodeID string
	nodeIndex := 0
	edgeIndex := 0
	posY := 100.0
	posYIncrement := 120.0

	for i, action := range actions {
		// Create the action node
		nodeID := fmt.Sprintf("node_%d", nodeIndex+1)
		node := mapActionToNode(action, nodeID, nodeIndex)
		// Override position to account for inserted wait nodes
		node["position"] = map[string]interface{}{
			"x": 250.0,
			"y": posY,
		}
		nodes = append(nodes, node)
		posY += posYIncrement
		nodeIndex++

		// Create edge from previous node
		if prevNodeID != "" {
			edges = append(edges, map[string]interface{}{
				"id":     fmt.Sprintf("edge_%d", edgeIndex+1),
				"source": prevNodeID,
				"target": nodeID,
			})
			edgeIndex++
		}
		prevNodeID = nodeID

		// Check if we need a wait before the next action
		if i < len(actions)-1 {
			nextAction := actions[i+1]
			waitTemplate := analyzeTransitionForWait(action, nextAction)

			if waitTemplate != nil {
				// Create wait node
				waitNodeID := fmt.Sprintf("wait_%d", nodeIndex+1)
				waitNode := createWaitNode(waitTemplate, waitNodeID, posY)
				nodes = append(nodes, waitNode)
				posY += posYIncrement
				nodeIndex++

				// Create edge from action to wait
				edges = append(edges, map[string]interface{}{
					"id":     fmt.Sprintf("edge_%d", edgeIndex+1),
					"source": prevNodeID,
					"target": waitNodeID,
				})
				edgeIndex++
				prevNodeID = waitNodeID
			}
		}
	}

	return nodes, edges
}

// generateClickLabel creates a readable label for a click action.
func generateClickLabel(action RecordedAction) string {
	if action.ElementMeta != nil {
		if action.ElementMeta.InnerText != "" {
			text := truncateString(action.ElementMeta.InnerText, 20)
			return fmt.Sprintf("Click: %s", text)
		}
		if action.ElementMeta.AriaLabel != "" {
			return fmt.Sprintf("Click: %s", action.ElementMeta.AriaLabel)
		}
		return fmt.Sprintf("Click %s", action.ElementMeta.TagName)
	}
	return "Click element"
}

// truncateString truncates a string to maxLen and adds "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractHostname extracts the hostname from a URL (or truncates if too long).
func extractHostname(urlStr string) string {
	if len(urlStr) > 50 {
		return urlStr[:50] + "..."
	}
	return urlStr
}
