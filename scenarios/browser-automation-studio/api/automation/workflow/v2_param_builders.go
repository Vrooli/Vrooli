package workflow

import (
	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
)

// Parameter builder functions convert V1 node data maps to typed proto params.

func buildNavigateParams(data map[string]any) *basv1.NavigateParams {
	p := &basv1.NavigateParams{}
	if url, ok := data["url"].(string); ok {
		p.Url = url
	}
	if wfs, ok := data["waitForSelector"].(string); ok {
		p.WaitForSelector = &wfs
	}
	if tm, ok := toInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	if wu, ok := data["waitUntil"].(string); ok {
		p.WaitUntil = &wu
	}
	return p
}

func buildClickParams(data map[string]any) *basv1.ClickParams {
	p := &basv1.ClickParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if button, ok := data["button"].(string); ok {
		p.Button = &button
	}
	if cc, ok := toInt32(data["clickCount"]); ok {
		p.ClickCount = &cc
	}
	if dm, ok := toInt32(data["delayMs"]); ok {
		p.DelayMs = &dm
	}
	if mods, ok := data["modifiers"].([]any); ok {
		for _, m := range mods {
			if s, ok := m.(string); ok {
				p.Modifiers = append(p.Modifiers, s)
			}
		}
	}
	if force, ok := data["force"].(bool); ok {
		p.Force = &force
	}
	return p
}

func buildInputParams(data map[string]any) *basv1.InputParams {
	p := &basv1.InputParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if value, ok := data["value"].(string); ok {
		p.Value = value
	}
	if text, ok := data["text"].(string); ok && p.Value == "" {
		p.Value = text // Legacy field name
	}
	if sensitive, ok := data["isSensitive"].(bool); ok {
		p.IsSensitive = &sensitive
	}
	if submit, ok := data["submit"].(bool); ok {
		p.Submit = &submit
	}
	if clear, ok := data["clearFirst"].(bool); ok {
		p.ClearFirst = &clear
	}
	return p
}

func buildWaitParams(data map[string]any) *basv1.WaitParams {
	p := &basv1.WaitParams{}
	if dm, ok := toInt32(data["durationMs"]); ok {
		p.WaitFor = &basv1.WaitParams_DurationMs{DurationMs: dm}
	} else if dur, ok := toInt32(data["duration"]); ok {
		p.WaitFor = &basv1.WaitParams_DurationMs{DurationMs: dur}
	} else if selector, ok := data["selector"].(string); ok && selector != "" {
		p.WaitFor = &basv1.WaitParams_Selector{Selector: selector}
	}
	if state, ok := data["state"].(string); ok {
		p.State = &state
	}
	if tm, ok := toInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

func buildAssertParams(data map[string]any) *basv1.AssertParams {
	p := &basv1.AssertParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if mode, ok := data["mode"].(string); ok {
		p.Mode = mode
	} else if mode, ok := data["assertMode"].(string); ok {
		p.Mode = mode // Legacy field name
	}
	if exp := data["expected"]; exp != nil {
		p.Expected = anyToJsonValue(exp)
	}
	if negated, ok := data["negated"].(bool); ok {
		p.Negated = &negated
	}
	return p
}

func buildScrollParams(data map[string]any) *basv1.ScrollParams {
	p := &basv1.ScrollParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = &selector
	}
	if x, ok := toInt32(data["x"]); ok {
		p.X = &x
	}
	if y, ok := toInt32(data["y"]); ok {
		p.Y = &y
	}
	if dx, ok := toInt32(data["deltaX"]); ok {
		p.DeltaX = &dx
	}
	if dy, ok := toInt32(data["deltaY"]); ok {
		p.DeltaY = &dy
	}
	if behavior, ok := data["behavior"].(string); ok {
		p.Behavior = &behavior
	}
	return p
}

func buildSelectParams(data map[string]any) *basv1.SelectParams {
	p := &basv1.SelectParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if value, ok := data["value"].(string); ok {
		p.SelectBy = &basv1.SelectParams_Value{Value: value}
	} else if label, ok := data["label"].(string); ok {
		p.SelectBy = &basv1.SelectParams_Label{Label: label}
	} else if idx, ok := toInt32(data["index"]); ok {
		p.SelectBy = &basv1.SelectParams_Index{Index: idx}
	}
	return p
}

func buildEvaluateParams(data map[string]any) *basv1.EvaluateParams {
	p := &basv1.EvaluateParams{}
	if expr, ok := data["expression"].(string); ok {
		p.Expression = expr
	}
	if store, ok := data["storeResult"].(string); ok {
		p.StoreResult = &store
	}
	return p
}

func buildKeyboardParams(data map[string]any) *basv1.KeyboardParams {
	p := &basv1.KeyboardParams{}
	if key, ok := data["key"].(string); ok {
		p.Key = &key
	}
	if keys, ok := data["keys"].([]any); ok {
		for _, k := range keys {
			if s, ok := k.(string); ok {
				p.Keys = append(p.Keys, s)
			}
		}
	}
	if mods, ok := data["modifiers"].([]any); ok {
		for _, m := range mods {
			if s, ok := m.(string); ok {
				p.Modifiers = append(p.Modifiers, s)
			}
		}
	}
	if action, ok := data["action"].(string); ok {
		p.Action = &action
	}
	return p
}

func buildHoverParams(data map[string]any) *basv1.HoverParams {
	p := &basv1.HoverParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if tm, ok := toInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

func buildScreenshotParams(data map[string]any) *basv1.ScreenshotParams {
	p := &basv1.ScreenshotParams{}
	if fullPage, ok := data["fullPage"].(bool); ok {
		p.FullPage = &fullPage
	}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = &selector
	}
	if quality, ok := toInt32(data["quality"]); ok {
		p.Quality = &quality
	}
	return p
}

func buildFocusParams(data map[string]any) *basv1.FocusParams {
	p := &basv1.FocusParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if scroll, ok := data["scroll"].(bool); ok {
		p.Scroll = &scroll
	}
	if tm, ok := toInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

func buildBlurParams(data map[string]any) *basv1.BlurParams {
	p := &basv1.BlurParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = &selector
	}
	if tm, ok := toInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

func buildActionMetadata(data map[string]any) *basv1.ActionMetadata {
	meta := &basv1.ActionMetadata{}
	hasData := false

	if label, ok := data["label"].(string); ok {
		meta.Label = &label
		hasData = true
	}

	if confidence, ok := toFloat64(data["confidence"]); ok {
		meta.Confidence = &confidence
		hasData = true
	}

	// Extract selector candidates if present
	if candidates, ok := data["selectorCandidates"].([]any); ok && len(candidates) > 0 {
		for _, c := range candidates {
			if cm, ok := c.(map[string]any); ok {
				candidate := &basv1.SelectorCandidate{}
				if t, ok := cm["type"].(string); ok {
					candidate.Type = t
				}
				if v, ok := cm["value"].(string); ok {
					candidate.Value = v
				}
				if conf, ok := toFloat64(cm["confidence"]); ok {
					candidate.Confidence = conf
				}
				if spec, ok := toInt32(cm["specificity"]); ok {
					candidate.Specificity = spec
				}
				meta.SelectorCandidates = append(meta.SelectorCandidates, candidate)
			}
		}
		hasData = true
	}

	if !hasData {
		return nil
	}
	return meta
}
