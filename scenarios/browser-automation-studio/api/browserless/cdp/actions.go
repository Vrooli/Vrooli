package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

func checkSelectorVisibility(ctx context.Context, selector string) (bool, bool, error) {
	var res struct {
		Found   bool `json:"found"`
		Visible bool `json:"visible"`
	}
	script := fmt.Sprintf(`(() => {
		const element = document.querySelector(%q);
		if (!element) {
			return { found: false, visible: false };
		}
		const rect = element.getBoundingClientRect();
		const vertVisible = rect.bottom > 0 && rect.top < (window.innerHeight || document.documentElement.clientHeight);
		const horizVisible = rect.right > 0 && rect.left < (window.innerWidth || document.documentElement.clientWidth);
		return { found: true, visible: vertVisible && horizVisible };
	})()`, selector)

	if err := chromedp.Run(ctx, chromedp.Evaluate(script, &res)); err != nil {
		return false, false, err
	}
	return res.Found, res.Visible, nil
}

func scrollBehaviorOrAuto(behavior string) string {
	switch strings.ToLower(strings.TrimSpace(behavior)) {
	case "smooth":
		return "smooth"
	default:
		return "auto"
	}
}

func scrollDeltaFromDirection(direction string, amount int) (int, int) {
	switch strings.ToLower(direction) {
	case "up":
		return 0, -amount
	case "down":
		return 0, amount
	case "left":
		return -amount, 0
	case "right":
		return amount, 0
	default:
		return 0, amount
	}
}

func resolveViewportForOrientation(width, height int, orientation string) (int, int) {
	if width <= 0 {
		width = 1920
	}
	if height <= 0 {
		height = 1080
	}
	switch strings.ToLower(orientation) {
	case rotateOrientationPortrait:
		if width > height {
			return height, width
		}
	case rotateOrientationLandscape:
		if height > width {
			return height, width
		}
	}
	return width, height
}

func deriveScreenOrientationType(orientation string, angle int) string {
	switch strings.ToLower(orientation) {
	case rotateOrientationLandscape:
		if angle == 270 {
			return "landscapeSecondary"
		}
		return "landscapePrimary"
	default:
		if angle == 180 {
			return "portraitSecondary"
		}
		return "portraitPrimary"
	}
}

type keyDefinition struct {
	Key     string
	Code    string
	Text    string
	KeyCode int
}

var specialKeyDefinitions = map[string]keyDefinition{
	"enter":      {Key: "Enter", Code: "Enter", KeyCode: 13},
	"return":     {Key: "Enter", Code: "Enter", KeyCode: 13},
	"tab":        {Key: "Tab", Code: "Tab", KeyCode: 9},
	"escape":     {Key: "Escape", Code: "Escape", KeyCode: 27},
	"esc":        {Key: "Escape", Code: "Escape", KeyCode: 27},
	"backspace":  {Key: "Backspace", Code: "Backspace", KeyCode: 8},
	"delete":     {Key: "Delete", Code: "Delete", KeyCode: 46},
	"space":      {Key: " ", Code: "Space", KeyCode: 32, Text: " "},
	"spacebar":   {Key: " ", Code: "Space", KeyCode: 32, Text: " "},
	"arrowup":    {Key: "ArrowUp", Code: "ArrowUp", KeyCode: 38},
	"arrowdown":  {Key: "ArrowDown", Code: "ArrowDown", KeyCode: 40},
	"arrowleft":  {Key: "ArrowLeft", Code: "ArrowLeft", KeyCode: 37},
	"arrowright": {Key: "ArrowRight", Code: "ArrowRight", KeyCode: 39},
	"home":       {Key: "Home", Code: "Home", KeyCode: 36},
	"end":        {Key: "End", Code: "End", KeyCode: 35},
	"pageup":     {Key: "PageUp", Code: "PageUp", KeyCode: 33},
	"pagedown":   {Key: "PageDown", Code: "PageDown", KeyCode: 34},
}

func normalizeKeyboardEventType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "keydown":
		return "keydown"
	case "keyup":
		return "keyup"
	default:
		return "keypress"
	}
}

func buildKeyEventParams(event input.KeyType, def keyDefinition, modifiers []string) *input.DispatchKeyEventParams {
	params := input.DispatchKeyEvent(event).
		WithKey(def.Key).
		WithCode(def.Code)
	if def.KeyCode > 0 {
		params = params.
			WithWindowsVirtualKeyCode(int64(def.KeyCode)).
			WithNativeVirtualKeyCode(int64(def.KeyCode))
	}
	if def.Text != "" {
		params = params.
			WithText(def.Text).
			WithUnmodifiedText(def.Text)
	}
	return applyModifierFlags(params, modifiers)
}

func applyModifierFlags(params *input.DispatchKeyEventParams, modifiers []string) *input.DispatchKeyEventParams {
	for _, mod := range modifiers {
		switch strings.ToLower(mod) {
		case "ctrl", "control":
			params.Modifiers |= input.ModifierCtrl
		case "shift":
			params.Modifiers |= input.ModifierShift
		case "alt", "option":
			params.Modifiers |= input.ModifierAlt
		case "meta", "cmd", "command", "super":
			params.Modifiers |= input.ModifierMeta
		}
	}
	return params
}

func resolveKeyDefinition(raw string) (keyDefinition, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return keyDefinition{}, fmt.Errorf("keyboard key is required")
	}
	lower := strings.ToLower(trimmed)
	if def, ok := specialKeyDefinitions[lower]; ok {
		return def, nil
	}
	runes := []rune(trimmed)
	if len(runes) == 1 {
		r := runes[0]
		code := deriveCodeFromRune(r)
		keyCode := int(unicode.ToUpper(r))
		if keyCode == 0 {
			keyCode = int(r)
		}
		text := string(r)
		return keyDefinition{
			Key:     string(r),
			Code:    code,
			Text:    text,
			KeyCode: keyCode,
		}, nil
	}
	if strings.HasPrefix(lower, "f") {
		if number, err := strconv.Atoi(strings.TrimPrefix(lower, "f")); err == nil && number >= 1 && number <= 24 {
			return keyDefinition{
				Key:     strings.ToUpper(fmt.Sprintf("F%d", number)),
				Code:    strings.ToUpper(fmt.Sprintf("F%d", number)),
				KeyCode: 111 + number,
			}, nil
		}
	}
	return keyDefinition{Key: trimmed, Code: trimmed}, nil
}

func buildExtractionScript(selector, extractType, attribute string, allMatches bool) (string, error) {
	selectorJSON, err := json.Marshal(selector)
	if err != nil {
		return "", err
	}
	modeJSON, err := json.Marshal(strings.ToLower(strings.TrimSpace(extractType)))
	if err != nil {
		return "", err
	}
	attributeJSON, err := json.Marshal(attribute)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`(() => {
	const selector = %s;
	const mode = %s || 'text';
	const attribute = %s || '';
	const allMatches = %t;
	const nodes = Array.from(document.querySelectorAll(selector));
	if (!nodes.length) {
		return { error: 'No elements matched selector ' + selector };
	}
	const extractValue = (element) => {
		switch (mode) {
			case 'html':
				return element.innerHTML;
			case 'value':
				return element.value ?? element.getAttribute('value');
			case 'attribute':
				return attribute ? element.getAttribute(attribute) : null;
			default:
				return element.textContent;
		}
	};
	if (allMatches) {
		return { multiple: true, values: nodes.map(extractValue) };
	}
	const first = nodes[0];
	const rect = first.getBoundingClientRect();
	return {
		value: extractValue(first),
		boundingBox: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
	};
})()`, string(selectorJSON), string(modeJSON), string(attributeJSON), allMatches)
	return script, nil
}

func parseBoundingBox(payload map[string]any) *runtime.BoundingBox {
	if payload == nil {
		return nil
	}
	box := &runtime.BoundingBox{}
	if x, ok := payload["x"].(float64); ok {
		box.X = x
	}
	if y, ok := payload["y"].(float64); ok {
		box.Y = y
	}
	if width, ok := payload["width"].(float64); ok {
		box.Width = width
	}
	if height, ok := payload["height"].(float64); ok {
		box.Height = height
	}
	return box
}

func boolFromPointer(value *bool) bool {
	return value != nil && *value
}

func evaluateVariableCondition(variableName, operator string, expected any, variables map[string]any) (bool, any, error) {
	trimmed := strings.TrimSpace(variableName)
	if trimmed == "" {
		return false, nil, fmt.Errorf("conditional variable name is required")
	}
	if variables == nil {
		return false, nil, fmt.Errorf("execution context is empty")
	}
	actual, ok := variables[trimmed]
	if !ok {
		return false, nil, fmt.Errorf("variable %s is not defined", trimmed)
	}
	match, err := compareConditionValues(actual, expected, operator)
	return match, actual, err
}

func normalizeConditionOperatorName(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "equals", "eq", "==":
		return "equals"
	case "not_equals", "!=", "neq":
		return "not_equals"
	case "contains", "includes":
		return "contains"
	case "starts_with", "prefix":
		return "starts_with"
	case "ends_with", "suffix":
		return "ends_with"
	case "gt", ">", "greater_than":
		return "gt"
	case "lt", "<", "less_than":
		return "lt"
	case "gte", ">=", "greater_than_or_equal":
		return "gte"
	case "lte", "<=", "less_than_or_equal":
		return "lte"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func compareConditionValues(actual, expected any, operator string) (bool, error) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	switch operator {
	case "equals", "", "eq":
		return actualStr == expectedStr, nil
	case "not_equals", "neq":
		return actualStr != expectedStr, nil
	case "contains":
		return strings.Contains(strings.ToLower(actualStr), strings.ToLower(expectedStr)), nil
	case "starts_with":
		return strings.HasPrefix(strings.ToLower(actualStr), strings.ToLower(expectedStr)), nil
	case "ends_with":
		return strings.HasSuffix(strings.ToLower(actualStr), strings.ToLower(expectedStr)), nil
	case "gt", "gte", "lt", "lte":
		actualNum, ok := toFloat64(actual)
		if !ok {
			return false, fmt.Errorf("actual value is not numeric")
		}
		expectedNum, ok := toFloat64(expected)
		if !ok {
			return false, fmt.Errorf("expected value is not numeric")
		}
		switch operator {
		case "gt":
			return actualNum > expectedNum, nil
		case "gte":
			return actualNum >= expectedNum, nil
		case "lt":
			return actualNum < expectedNum, nil
		case "lte":
			return actualNum <= expectedNum, nil
		}
	}
	return false, fmt.Errorf("unsupported operator %s", operator)
}

func toFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return parsed, true
		}
	case json.Number:
		if parsed, err := typed.Float64(); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func deriveCodeFromRune(r rune) string {
	switch {
	case unicode.IsLetter(r):
		return fmt.Sprintf("Key%s", strings.ToUpper(string(r)))
	case unicode.IsDigit(r):
		return fmt.Sprintf("Digit%c", r)
	}
	switch r {
	case ' ':
		return "Space"
	case '`':
		return "Backquote"
	case '-':
		return "Minus"
	case '=':
		return "Equal"
	case '[':
		return "BracketLeft"
	case ']':
		return "BracketRight"
	case '\\':
		return "Backslash"
	case ';':
		return "Semicolon"
	case '\'':
		return "Quote"
	case ',':
		return "Comma"
	case '.':
		return "Period"
	case '/':
		return "Slash"
	default:
		return fmt.Sprintf("Key%s", strings.ToUpper(string(r)))
	}
}

func centerFromBoxModel(box *dom.BoxModel) (float64, float64, error) {
	if box == nil || len(box.Content) < 8 {
		return 0, 0, fmt.Errorf("element box model unavailable")
	}
	x := (box.Content[0] + box.Content[4]) / 2
	y := (box.Content[1] + box.Content[5]) / 2
	return x, y, nil
}

func pickTabTarget(tabs []tabRecord, opts tabSwitchOptions) (*tabRecord, error) {
	if len(tabs) == 0 {
		return nil, fmt.Errorf("no tabs available")
	}
	mode := strings.ToLower(strings.TrimSpace(opts.SwitchBy))
	switch mode {
	case "index":
		if opts.Index < 0 || opts.Index >= len(tabs) {
			return nil, fmt.Errorf("tab index %d out of range", opts.Index)
		}
		return &tabs[opts.Index], nil
	case "title":
		pattern := strings.TrimSpace(opts.TitleMatch)
		if pattern == "" {
			return nil, fmt.Errorf("title pattern is required for switchBy=title")
		}
		for i := range tabs {
			if matchesPattern(tabs[i].Title, pattern) {
				return &tabs[i], nil
			}
		}
		return nil, fmt.Errorf("no tab matched title pattern %s", pattern)
	case "url":
		pattern := strings.TrimSpace(opts.URLMatch)
		if pattern == "" {
			return nil, fmt.Errorf("url pattern is required for switchBy=url")
		}
		for i := range tabs {
			if matchesPattern(tabs[i].URL, pattern) {
				return &tabs[i], nil
			}
		}
		return nil, fmt.Errorf("no tab matched url pattern %s", pattern)
	case "oldest":
		return &tabs[0], nil
	case "newest", "":
		return &tabs[len(tabs)-1], nil
	default:
		return &tabs[len(tabs)-1], nil
	}
}

func matchesPattern(value, pattern string) bool {
	if value == "" || pattern == "" {
		return false
	}
	if re, err := regexp.Compile(pattern); err == nil {
		return re.MatchString(value)
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
}

func waitWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
