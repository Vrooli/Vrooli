// Package livecapture provides business logic for live capture mode functionality.
// This includes converting recorded actions to workflows, action merging, and smart wait insertion.
package livecapture

import (
	"fmt"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/actions"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

// WorkflowGenerator converts recorded actions into workflow definitions.
type WorkflowGenerator struct{}

// NewWorkflowGenerator creates a new workflow generator.
func NewWorkflowGenerator() *WorkflowGenerator {
	return &WorkflowGenerator{}
}

// GenerateWorkflow derives typed candidates; unsupported observations never become clicks.
func (g *WorkflowGenerator) GenerateWorkflow(recorded []driver.RecordedAction) (*basworkflows.WorkflowDefinitionV2, error) {
	actions, err := prepareRecordedActions(recorded)
	if err != nil {
		return nil, err
	}
	actions = MergeConsecutiveActions(actions)
	flow := &basworkflows.WorkflowDefinitionV2{}
	appendNode := func(action *basactions.ActionDefinition) {
		index := len(flow.Nodes)
		node := &basworkflows.WorkflowNodeV2{Id: fmt.Sprintf("node_%d", index+1), Action: action, Position: &basbase.NodePosition{X: 250, Y: float64(100 + index*120)}}
		if index > 0 {
			flow.Edges = append(flow.Edges, &basworkflows.WorkflowEdgeV2{Id: fmt.Sprintf("edge_%d", index), Source: flow.Nodes[index-1].Id, Target: node.Id})
		}
		flow.Nodes = append(flow.Nodes, node)
	}
	for index, recordedAction := range actions {
		action, err := recordedActionDefinition(recordedAction)
		if err != nil {
			return nil, fmt.Errorf("recorded action %d (%s): %w", index+1, recordedAction.ActionType, err)
		}
		appendNode(action)
		if index+1 < len(actions) {
			if wait := analyzeTransitionForWait(recordedAction, actions[index+1]); wait != nil {
				appendNode(waitAction(wait))
			}
		}
	}
	return flow, nil
}

// MergeConsecutiveActions coalesces full input/scroll snapshots only within one
// target. It never writes through the recorded payload maps.
func MergeConsecutiveActions(recorded []driver.RecordedAction) []driver.RecordedAction {
	if recorded == nil {
		return nil
	}
	merged := make([]driver.RecordedAction, 0, len(recorded))
	for index, action := range recorded {
		if action.ActionType == "focus" && index+1 < len(recorded) && recorded[index+1].ActionType == "type" && sameTarget(action, recorded[index+1]) {
			continue
		}
		if len(merged) > 0 {
			previous := merged[len(merged)-1]
			if previous.ActionType == action.ActionType && sameTarget(previous, action) && coalescibleSnapshots(previous, action) {
				merged[len(merged)-1] = action
				continue
			}
		}
		merged = append(merged, action)
	}
	return merged
}

func sameTarget(a, b driver.RecordedAction) bool {
	if a.PageID != b.PageID || a.DriverPageID != b.DriverPageID || a.FrameID != b.FrameID || a.URL != b.URL {
		return false
	}
	if a.Selector == nil || b.Selector == nil {
		return a.Selector == nil && b.Selector == nil
	}
	return a.Selector.Primary == b.Selector.Primary
}

func coalescibleSnapshots(previous, next driver.RecordedAction) bool {
	switch next.ActionType {
	case "type":
		_, old := previous.Payload["text"].(string)
		_, current := next.Payload["text"].(string)
		return old && current && previous.Payload["submit"] != true
	case "scroll":
		_, oldX := previous.Payload["scrollX"]
		_, oldY := previous.Payload["scrollY"]
		_, newX := next.Payload["scrollX"]
		_, newY := next.Payload["scrollY"]
		return (oldX || oldY) && oldX == newX && oldY == newY
	default:
		return false
	}
}

// ApplyActionRange returns the requested action subset, clamping indices to the available actions.
func ApplyActionRange(actions []driver.RecordedAction, start, end int) []driver.RecordedAction {
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
func analyzeTransitionForWait(current, next driver.RecordedAction) *WaitTemplate {
	// Check if the next action needs its selector to exist (uses action registry)
	if actions.NeedsSelectorWait(actions.ActionType(next.ActionType)) && next.Selector != nil && next.Selector.Primary != "" {
		// If current action might trigger DOM changes, add a wait (uses action registry)
		triggersChanges := actions.TriggersDOMChanges(actions.ActionType(current.ActionType))

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
func describeElement(action driver.RecordedAction) string {
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

func waitAction(template *WaitTemplate) *basactions.ActionDefinition {
	params := &basactions.WaitParams{TimeoutMs: proto.Int32(int32(template.TimeoutMs))}
	if template.WaitType == "selector" {
		params.WaitFor = &basactions.WaitParams_Selector{Selector: template.Selector}
		params.State = basactions.WaitState_WAIT_STATE_VISIBLE.Enum()
	} else {
		params.WaitFor = &basactions.WaitParams_DurationMs{DurationMs: int32(template.TimeoutMs)}
	}
	return &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_WAIT, Params: &basactions.ActionDefinition_Wait{Wait: params}, Metadata: &basactions.ActionMetadata{Label: proto.String(template.Label)}}
}

// generateClickLabel creates a readable label for a click action.
func generateClickLabel(action driver.RecordedAction) string {
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
