package livecapture

import (
	"fmt"
	"maps"

	"github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/proto"
)

// prepareRecordedActions checks target representability before any merging can
// discard identity, and joins browser drag phases into a complete observation.
func prepareRecordedActions(recorded []driver.RecordedAction) ([]driver.RecordedAction, error) {
	result := make([]driver.RecordedAction, 0, len(recorded))
	if err := validateRecordedFramePaths(recorded); err != nil {
		return nil, err
	}
	dragSource := ""
	for index, action := range recorded {
		if action.ActionType == "input" {
			// API timeline entries use the typed INPUT enum while the capture
			// pipeline's snapshot merger and compiler adapter call it "type".
			action.ActionType = "type"
		}
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

func validateRecordedFramePaths(recorded []driver.RecordedAction) error {
	for index, action := range recorded {
		if action.FrameID != "" && len(action.FramePath) == 0 {
			return fmt.Errorf("recorded action %d: frame replay requires a captured frame selector path", index+1)
		}
		if len(action.FramePath) > 32 {
			return fmt.Errorf("recorded action %d: frame selector path exceeds 32 levels", index+1)
		}
		for _, selector := range action.FramePath {
			if selector == "" {
				return fmt.Errorf("recorded action %d: frame selector path contains an empty selector", index+1)
			}
		}
	}
	return nil
}

func resolveRecordedPageTargets(recorded []driver.RecordedAction, pages []*domain.Page) ([]string, map[string]*domain.Page, error) {
	pageByDriver := make(map[string]*domain.Page, len(pages))
	pageByID := make(map[string]*domain.Page, len(pages))
	var initialPage *domain.Page
	for _, page := range pages {
		if page == nil {
			continue
		}
		pageByID[page.ID.String()] = page
		if page.IsInitial {
			if initialPage != nil && initialPage.ID != page.ID {
				return nil, nil, fmt.Errorf("recording has duplicate initial page bindings")
			}
			initialPage = page
		}
		if page.DriverPageID != "" {
			if _, exists := pageByDriver[page.DriverPageID]; exists {
				return nil, nil, fmt.Errorf("recording has duplicate bindings for driver page %q", page.DriverPageID)
			}
			pageByDriver[page.DriverPageID] = page
		}
	}

	targets := make([]string, len(recorded))
	unique := make(map[string]struct{}, len(recorded))
	for index, action := range recorded {
		byDriver := pageByDriver[action.DriverPageID]
		byID := pageByID[action.PageID]
		if byDriver != nil && byID != nil && byDriver.ID != byID.ID {
			return nil, nil, fmt.Errorf("recorded action %d: page and driver page identities conflict", index+1)
		}
		// The recorder synthesizes its first navigation before the browser event
		// route can attach page identity. Bind only that known pre-capture action
		// to the session's registered initial page; leave other missing identities
		// fail-closed in multi-page recordings.
		if index == 0 && action.ActionType == "navigate" && action.PageID == "" && action.DriverPageID == "" && initialPage != nil {
			byID = initialPage
		}
		if len(pageByID) > 1 && byDriver == nil && byID == nil {
			return nil, nil, fmt.Errorf("recorded action %d: multi-page recording is missing its logical page identity", index+1)
		}
		target := ""
		if byDriver != nil {
			target = byDriver.ID.String()
		} else if byID != nil {
			target = byID.ID.String()
		} else if action.PageID != "" || action.DriverPageID != "" {
			target = "unbound:" + action.PageID + ":" + action.DriverPageID
		} else {
			target = "initial"
		}
		targets[index] = target
		unique[target] = struct{}{}
	}
	if len(unique) > 1 {
		for target := range unique {
			if pageByID[target] == nil {
				return nil, nil, fmt.Errorf("recording has multiple or ambiguous pages; target %q has no logical tab binding", target)
			}
		}
	}
	return targets, pageByID, nil
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
