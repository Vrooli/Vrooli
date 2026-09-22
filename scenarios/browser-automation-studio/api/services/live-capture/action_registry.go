package livecapture

import (
	"fmt"
	"maps"

	"github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/proto"
)

// prepareRecordedActions checks target representability before any merging can
// discard identity, and joins browser drag phases into a complete observation.
func prepareRecordedActions(recorded []driver.RecordedAction) ([]driver.RecordedAction, error) {
	result := make([]driver.RecordedAction, 0, len(recorded))
	if err := validateRecordedTargets(recorded); err != nil {
		return nil, err
	}
	dragSource := ""
	for index, action := range recorded {
		if action.ActionType == "drag-drop" {
			phase, _ := action.Payload["phase"].(string)
			switch phase {
			case "start":
				if dragSource != "" || action.Selector == nil || action.Selector.Primary == "" {
					return nil, fmt.Errorf("recorded action %d: incomplete or overlapping drag", index+1)
				}
				dragSource = action.Selector.Primary
				continue
			case "drop":
				source, _ := action.Payload["sourceSelector"].(string)
				target, _ := action.Payload["targetSelector"].(string)
				if source == "" || target == "" || (dragSource != "" && source != dragSource) {
					return nil, fmt.Errorf("recorded action %d: drag source/target is missing or inconsistent", index+1)
				}
				dragSource = ""
				action.ActionType = "dragDrop"
			default:
				return nil, fmt.Errorf("recorded action %d: unsupported drag phase %q", index+1, phase)
			}
		}
		result = append(result, action)
	}
	if dragSource != "" {
		return nil, fmt.Errorf("recording ends with an unfinished drag")
	}
	return result, nil
}

// validateRecordedTargets refuses contexts whose lifetime cannot yet be rebuilt.
func validateRecordedTargets(recorded []driver.RecordedAction) error {
	type pageIdentity struct{ page, driver string }
	targets := map[pageIdentity]struct{}{}
	for index, action := range recorded {
		if action.FrameID != "" {
			return fmt.Errorf("recorded action %d: frame replay requires a logical frame binding", index+1)
		}
		targets[pageIdentity{action.PageID, action.DriverPageID}] = struct{}{}
	}
	if len(targets) > 1 {
		return fmt.Errorf("recording has multiple or ambiguous pages; replay requires logical tab bindings")
	}
	return nil
}

// recordedActionDefinition only adapts recording-specific fields. The compiler
// owns action type/parameter conversion; there is no intermediate V1 node.
func recordedActionDefinition(action driver.RecordedAction) (*basactions.ActionDefinition, error) {
	params := maps.Clone(action.Payload)
	if params == nil {
		params = map[string]any{}
	}
	if action.Selector != nil {
		params["selector"] = action.Selector.Primary
	}
	var label string
	switch action.ActionType {
	case "click":
		label = generateClickLabel(action)
		if delay, ok := params["delay"]; ok {
			params["delayMs"] = delay
		}
	case "type":
		if _, ok := params["text"].(string); !ok {
			return nil, fmt.Errorf("input observation requires a full text value, including an explicit empty string")
		}
		label = generateTypeLabel(action)
		params["clearFirst"] = true // Captured input values are complete snapshots.
	case "navigate":
		label = generateNavigateLabel(action)
		params["url"] = action.URL
	case "scroll":
		label = "Scroll"
		if x, ok := params["scrollX"]; ok {
			params["x"] = x
			delete(params, "deltaX")
		}
		if y, ok := params["scrollY"]; ok {
			params["y"] = y
			delete(params, "deltaY")
		}
	case "select":
		label = "Select option"
	case "focus":
		label = "Focus element"
	case "blur":
		label = "Blur element"
	case "hover":
		label = "Hover element"
	case "keyboard":
		label = generateKeypressLabel(action)
	case "dragDrop":
		label = "Drag and drop"
		if _, ok := params["sourceSelector"]; !ok {
			params["sourceSelector"] = params["selector"]
		}
	case "wait":
		label = "Wait"
		if params["selector"] == nil {
			params["durationMs"] = params["timeoutMs"]
		}
	case "assert":
		label = "Assert"
	case "screenshot":
		label = "Screenshot"
	default:
		return nil, fmt.Errorf("unsupported recorded action type %q", action.ActionType)
	}
	result, err := compiler.BuildActionDefinition(action.ActionType, params)
	if err != nil {
		return nil, err
	}
	result.Metadata = &basactions.ActionMetadata{Label: proto.String(label)}
	return result, nil
}

// Label generator functions

func generateTypeLabel(action driver.RecordedAction) string {
	if action.Payload != nil {
		if text, ok := action.Payload["text"].(string); ok {
			return fmt.Sprintf("Type: %q", truncateString(text, 20))
		}
	}
	return "Type text"
}

func generateNavigateLabel(action driver.RecordedAction) string {
	return fmt.Sprintf("Navigate to %s", extractHostname(action.URL))
}

func generateKeypressLabel(action driver.RecordedAction) string {
	if action.Payload != nil {
		if key, ok := action.Payload["key"].(string); ok {
			return fmt.Sprintf("Press %s", key)
		}
	}
	return "Press key"
}
