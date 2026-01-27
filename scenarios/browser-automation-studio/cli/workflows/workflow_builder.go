package workflows

import (
	"fmt"
	"strings"
)

// BuildWorkflowFromSteps creates a workflow definition from parsed step specs.
func BuildWorkflowFromSteps(steps []*StepSpec) (map[string]any, error) {
	if len(steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	nodes := make([]map[string]any, len(steps))
	edges := make([]map[string]any, 0, len(steps)-1)

	for i, step := range steps {
		nodeID := fmt.Sprintf("step-%d", i+1)
		node, err := buildNode(nodeID, step, i)
		if err != nil {
			return nil, fmt.Errorf("step %d (%s): %w", i+1, step.Type, err)
		}
		nodes[i] = node

		// Create edge from previous node
		if i > 0 {
			edges = append(edges, map[string]any{
				"id":     fmt.Sprintf("edge-%d", i),
				"source": fmt.Sprintf("step-%d", i),
				"target": nodeID,
			})
		}
	}

	return map[string]any{
		"metadata": map[string]any{
			"name":        "inline-workflow",
			"description": "Generated from CLI --step flags",
		},
		"nodes": nodes,
		"edges": edges,
	}, nil
}

// buildNode creates a node definition from a step spec.
func buildNode(id string, step *StepSpec, idx int) (map[string]any, error) {
	var data map[string]any
	var err error

	switch step.Type {
	case "navigate":
		data, err = buildNavigateData(step)
	case "click":
		data, err = buildClickData(step)
	case "type":
		data, err = buildTypeData(step)
	case "assert":
		data, err = buildAssertData(step)
	case "wait":
		data, err = buildWaitData(step)
	case "screenshot":
		data, err = buildScreenshotData(step)
	case "evaluate":
		data, err = buildEvaluateData(step)
	case "hover":
		data, err = buildHoverData(step)
	case "focus":
		data, err = buildFocusData(step)
	case "blur":
		data, err = buildBlurData(step)
	case "select":
		data, err = buildSelectData(step)
	case "keyboard":
		data, err = buildKeyboardData(step)
	case "shortcut":
		data, err = buildShortcutData(step)
	case "extract":
		data, err = buildExtractData(step)
	default:
		return nil, fmt.Errorf("unsupported step type: %s", step.Type)
	}

	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":   id,
		"type": step.Type,
		"data": data,
	}, nil
}

// applyKVPairs applies key-value pairs to a data map, supporting nested keys.
func applyKVPairs(data map[string]any, pairs map[string]string, skip ...string) {
	skipSet := make(map[string]bool)
	for _, s := range skip {
		skipSet[s] = true
	}

	for key, value := range pairs {
		if skipSet[key] {
			continue
		}

		// Handle nested keys (e.g., resilience.maxAttempts)
		if strings.Contains(key, ".") {
			setNestedValue(data, key, parseValue(value))
		} else {
			data[key] = parseValue(value)
		}
	}
}

// buildNavigateData creates data for a navigate node.
func buildNavigateData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	// Check for scenario-based navigation
	if scenario := step.KVPairs["scenario"]; scenario != "" {
		data["destinationType"] = "scenario"
		data["scenario"] = scenario
		if path := step.KVPairs["path"]; path != "" {
			data["scenarioPath"] = path
		}
	} else if step.Positional != "" {
		data["destinationType"] = "url"
		data["url"] = step.Positional
	} else if url := step.KVPairs["url"]; url != "" {
		data["destinationType"] = "url"
		data["url"] = url
	} else {
		return nil, fmt.Errorf("navigate requires URL or scenario=")
	}

	// Apply additional key=value pairs
	applyKVPairs(data, step.KVPairs, "scenario", "path", "url")

	return data, nil
}

// buildClickData creates data for a click node.
func buildClickData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["selector"] = step.Positional
	} else if selector := step.KVPairs["selector"]; selector != "" {
		data["selector"] = selector
	} else {
		return nil, fmt.Errorf("click requires selector")
	}

	applyKVPairs(data, step.KVPairs, "selector")
	return data, nil
}

// buildTypeData creates data for a type node.
func buildTypeData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["selector"] = step.Positional
	} else if selector := step.KVPairs["selector"]; selector != "" {
		data["selector"] = selector
	} else {
		return nil, fmt.Errorf("type requires selector")
	}

	// text= is required for type
	if text := step.KVPairs["text"]; text != "" {
		data["text"] = text
	} else if value := step.KVPairs["value"]; value != "" {
		data["text"] = value
	}

	applyKVPairs(data, step.KVPairs, "selector", "text", "value")
	return data, nil
}

// buildAssertData creates data for an assert node.
func buildAssertData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["selector"] = step.Positional
	} else if selector := step.KVPairs["selector"]; selector != "" {
		data["selector"] = selector
	} else {
		return nil, fmt.Errorf("assert requires selector")
	}

	if assertMode := step.KVPairs["assertMode"]; assertMode != "" {
		data["assertMode"] = assertMode
	} else {
		return nil, fmt.Errorf("assert requires assertMode=")
	}

	applyKVPairs(data, step.KVPairs, "selector", "assertMode")
	return data, nil
}

// buildWaitData creates data for a wait node.
func buildWaitData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	// Check what type of wait this is
	if durationMs := step.KVPairs["durationMs"]; durationMs != "" {
		data["waitType"] = "duration"
		data["durationMs"] = parseValue(durationMs)
	} else if selector := step.KVPairs["selector"]; selector != "" {
		data["waitType"] = "element"
		data["selector"] = selector
	} else if step.Positional != "" {
		// Positional could be a selector or duration
		data["waitType"] = "element"
		data["selector"] = step.Positional
	} else {
		return nil, fmt.Errorf("wait requires durationMs= or selector=")
	}

	applyKVPairs(data, step.KVPairs, "durationMs", "selector")
	return data, nil
}

// buildScreenshotData creates data for a screenshot node.
func buildScreenshotData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	// Optional selector for element screenshot
	if step.Positional != "" {
		data["selector"] = step.Positional
	}

	applyKVPairs(data, step.KVPairs)
	return data, nil
}

// buildEvaluateData creates data for an evaluate node.
func buildEvaluateData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["expression"] = step.Positional
	} else if expression := step.KVPairs["expression"]; expression != "" {
		data["expression"] = expression
	} else if script := step.KVPairs["script"]; script != "" {
		data["expression"] = script
	} else if code := step.KVPairs["code"]; code != "" {
		data["expression"] = code
	} else {
		return nil, fmt.Errorf("evaluate requires expression")
	}

	applyKVPairs(data, step.KVPairs, "expression", "script", "code")
	return data, nil
}

// buildHoverData creates data for a hover node.
func buildHoverData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["selector"] = step.Positional
	} else if selector := step.KVPairs["selector"]; selector != "" {
		data["selector"] = selector
	} else {
		return nil, fmt.Errorf("hover requires selector")
	}

	applyKVPairs(data, step.KVPairs, "selector")
	return data, nil
}

// buildFocusData creates data for a focus node.
func buildFocusData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["selector"] = step.Positional
	} else if selector := step.KVPairs["selector"]; selector != "" {
		data["selector"] = selector
	} else {
		return nil, fmt.Errorf("focus requires selector")
	}

	applyKVPairs(data, step.KVPairs, "selector")
	return data, nil
}

// buildBlurData creates data for a blur node.
func buildBlurData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["selector"] = step.Positional
	} else if selector := step.KVPairs["selector"]; selector != "" {
		data["selector"] = selector
	} else {
		return nil, fmt.Errorf("blur requires selector")
	}

	applyKVPairs(data, step.KVPairs, "selector")
	return data, nil
}

// buildSelectData creates data for a select node.
func buildSelectData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["selector"] = step.Positional
	} else if selector := step.KVPairs["selector"]; selector != "" {
		data["selector"] = selector
	} else {
		return nil, fmt.Errorf("select requires selector")
	}

	// Require at least one of optionText, optionValue, or optionIndex
	hasOption := false
	if v := step.KVPairs["optionText"]; v != "" {
		data["optionText"] = v
		hasOption = true
	}
	if v := step.KVPairs["optionValue"]; v != "" {
		data["optionValue"] = v
		hasOption = true
	}
	if v := step.KVPairs["optionIndex"]; v != "" {
		data["optionIndex"] = parseValue(v)
		hasOption = true
	}
	if !hasOption {
		return nil, fmt.Errorf("select requires optionText=, optionValue=, or optionIndex=")
	}

	applyKVPairs(data, step.KVPairs, "selector", "optionText", "optionValue", "optionIndex")
	return data, nil
}

// buildKeyboardData creates data for a keyboard node.
func buildKeyboardData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["key"] = step.Positional
	} else if key := step.KVPairs["key"]; key != "" {
		data["key"] = key
	}

	applyKVPairs(data, step.KVPairs, "key")
	return data, nil
}

// buildShortcutData creates data for a shortcut node.
func buildShortcutData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["keys"] = step.Positional
	} else if keys := step.KVPairs["keys"]; keys != "" {
		data["keys"] = keys
	} else {
		return nil, fmt.Errorf("shortcut requires keys")
	}

	applyKVPairs(data, step.KVPairs, "keys")
	return data, nil
}

// buildExtractData creates data for an extract node.
func buildExtractData(step *StepSpec) (map[string]any, error) {
	data := make(map[string]any)

	if step.Positional != "" {
		data["selector"] = step.Positional
	} else if selector := step.KVPairs["selector"]; selector != "" {
		data["selector"] = selector
	} else {
		return nil, fmt.Errorf("extract requires selector")
	}

	applyKVPairs(data, step.KVPairs, "selector")
	return data, nil
}
