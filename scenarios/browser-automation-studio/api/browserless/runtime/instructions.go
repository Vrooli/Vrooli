package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
)

// Instruction represents a normalized execution step that can be shipped to Browserless.
type Instruction struct {
	Index       int              `json:"index"`
	NodeID      string           `json:"nodeId"`
	Type        string           `json:"type"`
	Params      InstructionParam `json:"params"`
	PreloadHTML string           `json:"preloadHtml,omitempty"`
}

// InstructionParam captures the parameter payload for a Browserless instruction.
type InstructionParam struct {
	URL                   string   `json:"url,omitempty"`
	Scenario              string   `json:"scenario,omitempty"`        // Scenario name for navigate nodes
	ScenarioPath          string   `json:"scenarioPath,omitempty"`    // Scenario path for navigate nodes
	DestinationType       string   `json:"destinationType,omitempty"` // url or scenario
	WaitUntil             string   `json:"waitUntil,omitempty"`
	TimeoutMs             int      `json:"timeoutMs,omitempty"`
	WaitForMs             int      `json:"waitForMs,omitempty"`
	WaitType              string   `json:"waitType,omitempty"`
	DurationMs            int      `json:"durationMs,omitempty"`
	Selector              string   `json:"selector,omitempty"`
	WaitForSelector       string   `json:"waitForSelector,omitempty"`
	Name                  string   `json:"name,omitempty"`
	FullPage              *bool    `json:"fullPage,omitempty"`
	ViewportWidth         int      `json:"viewportWidth,omitempty"`
	ViewportHeight        int      `json:"viewportHeight,omitempty"`
	Button                string   `json:"button,omitempty"`
	ClickCount            int      `json:"clickCount,omitempty"`
	Text                  string   `json:"text,omitempty"`
	DelayMs               int      `json:"delayMs,omitempty"`
	Clear                 *bool    `json:"clear,omitempty"`
	Submit                *bool    `json:"submit,omitempty"`
	ExtractType           string   `json:"extractType,omitempty"`
	Attribute             string   `json:"attribute,omitempty"`
	AllMatches            *bool    `json:"allMatches,omitempty"`
	FocusSelector         string   `json:"focusSelector,omitempty"`
	HighlightSelectors    []string `json:"highlightSelectors,omitempty"`
	HighlightColor        string   `json:"highlightColor,omitempty"`
	HighlightPadding      int      `json:"highlightPadding,omitempty"`
	HighlightBorderRadius int      `json:"highlightBorderRadius,omitempty"`
	MaskSelectors         []string `json:"maskSelectors,omitempty"`
	MaskOpacity           float64  `json:"maskOpacity,omitempty"`
	Background            string   `json:"background,omitempty"`
	ZoomFactor            float64  `json:"zoomFactor,omitempty"`
	CaptureDomSnapshot    bool     `json:"captureDomSnapshot,omitempty"`
	AssertMode            string   `json:"assertMode,omitempty"`
	ExpectedValue         any      `json:"expectedValue,omitempty"`
	FailureMessage        string   `json:"failureMessage,omitempty"`
	Expression            string   `json:"expression,omitempty"`
	CaseSensitive         *bool    `json:"caseSensitive,omitempty"`
	Negate                *bool    `json:"negate,omitempty"`
	ContinueOnFailure     *bool    `json:"continueOnFailure,omitempty"`
	RetryAttempts         int      `json:"retryAttempts,omitempty"`
	RetryDelayMs          int      `json:"retryDelayMs,omitempty"`
	RetryBackoffFactor    float64  `json:"retryBackoffFactor,omitempty"`
	ProbeX                int      `json:"probeX,omitempty"`
	ProbeY                int      `json:"probeY,omitempty"`
	ProbeRadius           int      `json:"probeRadius,omitempty"`
	ProbeSamples          int      `json:"probeSamples,omitempty"`
	ShortcutKeys          []string `json:"shortcutKeys,omitempty"`
	ShortcutDelayMs       int      `json:"shortcutDelayMs,omitempty"`
}

// InstructionsFromPlan converts a compiled execution plan into Browserless instructions.
func InstructionsFromPlan(ctx context.Context, plan *compiler.ExecutionPlan) ([]Instruction, error) {
	if plan == nil {
		return nil, fmt.Errorf("execution plan is nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	instructions := make([]Instruction, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		instr, err := instructionFromStep(ctx, step)
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, instr)
	}

	return instructions, nil
}

type navigateConfig struct {
	URL             string `json:"url"`
	WaitUntil       string `json:"waitUntil"`
	TimeoutMs       int    `json:"timeoutMs"`
	WaitForMs       int    `json:"waitForMs"`
	Scenario        string `json:"scenario"`
	ScenarioName    string `json:"scenarioName"`
	ScenarioPath    string `json:"scenarioPath"`
	ScenarioPort    string `json:"scenarioPort"`
	DestinationType string `json:"destinationType"`
}

type waitConfig struct {
	Type      string `json:"type"`
	WaitType  string `json:"waitType"`
	Duration  int    `json:"duration"`
	Selector  string `json:"selector"`
	TimeoutMs int    `json:"timeoutMs"`
}

type screenshotConfig struct {
	Name                  string   `json:"name"`
	FullPage              *bool    `json:"fullPage"`
	ViewportWidth         int      `json:"viewportWidth"`
	ViewportHeight        int      `json:"viewportHeight"`
	WaitForMs             int      `json:"waitForMs"`
	FocusSelector         string   `json:"focusSelector"`
	HighlightSelectors    []string `json:"highlightSelectors"`
	HighlightColor        string   `json:"highlightColor"`
	HighlightPadding      int      `json:"highlightPadding"`
	HighlightBorderRadius int      `json:"highlightBorderRadius"`
	MaskSelectors         []string `json:"maskSelectors"`
	MaskOpacity           float64  `json:"maskOpacity"`
	Background            string   `json:"background"`
	ZoomFactor            float64  `json:"zoomFactor"`
	CaptureDomSnapshot    bool     `json:"captureDomSnapshot"`
}

type clickConfig struct {
	Selector        string `json:"selector"`
	Button          string `json:"button"`
	ClickCount      int    `json:"clickCount"`
	TimeoutMs       int    `json:"timeoutMs"`
	WaitForMs       int    `json:"waitForMs"`
	WaitForSelector string `json:"waitForSelector"`
}

type typeConfig struct {
	Selector  string `json:"selector"`
	Text      string `json:"text"`
	DelayMs   int    `json:"delayMs"`
	Clear     *bool  `json:"clear"`
	Submit    *bool  `json:"submit"`
	TimeoutMs int    `json:"timeoutMs"`
}

type shortcutConfig struct {
	Shortcuts     []string `json:"shortcuts"`
	Shortcut      string   `json:"shortcut"`
	Sequence      []string `json:"sequence"`
	Keys          []string `json:"keys"`
	DelayMs       int      `json:"delayMs"`
	TimeoutMs     int      `json:"timeoutMs"`
	FocusSelector string   `json:"focusSelector"`
}

type extractConfig struct {
	Selector    string `json:"selector"`
	ExtractType string `json:"extractType"`
	Attribute   string `json:"attribute"`
	AllMatches  *bool  `json:"allMatches"`
}

type assertConfig struct {
	AssertMode        string `json:"assertMode"`
	Mode              string `json:"mode"`
	Comparison        string `json:"comparison"`
	Selector          string `json:"selector"`
	ExpectedValue     any    `json:"expectedValue"`
	Expected          any    `json:"expected"`
	Text              string `json:"text"`
	TimeoutMs         int    `json:"timeoutMs"`
	Expression        string `json:"expression"`
	Attribute         string `json:"attribute"`
	FailureMessage    string `json:"failureMessage"`
	CaseSensitive     *bool  `json:"caseSensitive"`
	Negate            *bool  `json:"negate"`
	ContinueOnFailure *bool  `json:"continueOnFailure"`
}

func instructionFromStep(ctx context.Context, step compiler.ExecutionStep) (Instruction, error) {
	base := Instruction{
		Index:  step.Index,
		NodeID: step.NodeID,
		Type:   string(step.Type),
		Params: InstructionParam{},
	}

	if attempts, ok := getIntParam(step.Params, "retryAttempts", "retry_attempts", "retry"); ok {
		if attempts < 0 {
			attempts = 0
		}
		base.Params.RetryAttempts = attempts
	}
	if delay, ok := getIntParam(step.Params, "retryDelayMs", "retry_delay_ms", "retryDelay"); ok && delay > 0 {
		base.Params.RetryDelayMs = delay
	}
	if factor, ok := getFloatParam(step.Params, "retryBackoffFactor", "retry_backoff_factor", "retryBackoff"); ok && factor > 0 {
		base.Params.RetryBackoffFactor = factor
	}

	switch step.Type {
	case compiler.StepNavigate:
		var cfg navigateConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("navigate node %s has invalid data: %w", step.NodeID, err)
		}
		destinationType := strings.ToLower(strings.TrimSpace(cfg.DestinationType))
		scenarioName := strings.TrimSpace(cfg.Scenario)
		if scenarioName == "" {
			scenarioName = strings.TrimSpace(cfg.ScenarioName)
		}
		scenarioSelected := destinationType == "scenario" || (destinationType == "" && scenarioName != "")

		if scenarioSelected {
			if scenarioName == "" {
				return Instruction{}, fmt.Errorf("navigate node %s missing scenario name", step.NodeID)
			}

			portCandidates := []string{}
			if strings.TrimSpace(cfg.ScenarioPort) != "" {
				portCandidates = append(portCandidates, cfg.ScenarioPort)
			}

			resolvedURL, _, err := scenarioport.ResolveURL(ctx, scenarioName, cfg.ScenarioPath, portCandidates...)
			if err != nil {
				return Instruction{}, fmt.Errorf("navigate node %s failed to resolve scenario %s: %w", step.NodeID, scenarioName, err)
			}
			base.Params.URL = resolvedURL
			base.Params.Scenario = scenarioName
			base.Params.ScenarioPath = strings.TrimSpace(cfg.ScenarioPath)
			base.Params.DestinationType = "scenario"
		} else {
			trimmedURL := strings.TrimSpace(cfg.URL)
			if trimmedURL == "" {
				return Instruction{}, fmt.Errorf("navigate node %s missing url", step.NodeID)
			}
			base.Params.URL = trimmedURL
			base.Params.DestinationType = "url"
		}
		if wait := strings.TrimSpace(cfg.WaitUntil); wait != "" {
			base.Params.WaitUntil = wait
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepWait:
		var cfg waitConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("wait node %s has invalid data: %w", step.NodeID, err)
		}
		waitType := strings.ToLower(strings.TrimSpace(firstNonEmpty(cfg.Type, cfg.WaitType)))
		if waitType == "" || waitType == "time" {
			base.Params.WaitType = "time"
			if cfg.Duration > 0 {
				base.Params.DurationMs = cfg.Duration
			} else {
				base.Params.DurationMs = 1000
			}
		} else if waitType == "element" {
			base.Params.WaitType = "element"
			base.Params.Selector = strings.TrimSpace(cfg.Selector)
			base.Params.TimeoutMs = cfg.TimeoutMs
			if base.Params.Selector == "" {
				return Instruction{}, fmt.Errorf("wait node %s requires selector for element wait", step.NodeID)
			}
		} else if waitType == "navigation" {
			base.Params.WaitType = "navigation"
			if cfg.TimeoutMs > 0 {
				base.Params.TimeoutMs = cfg.TimeoutMs
			}
		} else {
			return Instruction{}, fmt.Errorf("wait node %s has unsupported type %q", step.NodeID, cfg.Type)
		}
	case compiler.StepScreenshot:
		var cfg screenshotConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("screenshot node %s has invalid data: %w", step.NodeID, err)
		}
		base.Params.Name = cfg.Name
		base.Params.ViewportWidth = cfg.ViewportWidth
		base.Params.ViewportHeight = cfg.ViewportHeight
		base.Params.WaitForMs = cfg.WaitForMs
		base.Params.FullPage = cfg.FullPage
		if trimmed := strings.TrimSpace(cfg.FocusSelector); trimmed != "" {
			base.Params.FocusSelector = trimmed
		}
		if selectors := normalizeStringSlice(cfg.HighlightSelectors); len(selectors) > 0 {
			base.Params.HighlightSelectors = selectors
		}
		if color := strings.TrimSpace(cfg.HighlightColor); color != "" {
			base.Params.HighlightColor = color
		}
		if cfg.HighlightPadding > 0 {
			base.Params.HighlightPadding = cfg.HighlightPadding
		}
		if cfg.HighlightBorderRadius > 0 {
			base.Params.HighlightBorderRadius = cfg.HighlightBorderRadius
		}
		if selectors := normalizeStringSlice(cfg.MaskSelectors); len(selectors) > 0 {
			base.Params.MaskSelectors = selectors
		}
		if cfg.MaskOpacity > 0 {
			base.Params.MaskOpacity = cfg.MaskOpacity
		}
		if background := strings.TrimSpace(cfg.Background); background != "" {
			base.Params.Background = background
		}
		if cfg.ZoomFactor > 0 {
			base.Params.ZoomFactor = cfg.ZoomFactor
		}
		if cfg.CaptureDomSnapshot {
			base.Params.CaptureDomSnapshot = true
		} else if capture, ok := getBoolParam(step.Params, "captureDomSnapshot", "capture_dom_snapshot", "captureDom"); ok && capture {
			base.Params.CaptureDomSnapshot = true
		}
	case compiler.StepClick:
		var cfg clickConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("click node %s has invalid data: %w", step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("click node %s missing selector", step.NodeID)
		}
		base.Params.Selector = selector
		if button := strings.ToLower(strings.TrimSpace(cfg.Button)); button != "" && button != "left" {
			base.Params.Button = button
		}
		if cfg.ClickCount > 1 {
			base.Params.ClickCount = cfg.ClickCount
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
		if trimmed := strings.TrimSpace(cfg.WaitForSelector); trimmed != "" {
			base.Params.WaitForSelector = trimmed
		}
	case compiler.StepTypeInput:
		var cfg typeConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("type node %s has invalid data: %w", step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("type node %s missing selector", step.NodeID)
		}
		base.Params.Selector = selector
		base.Params.Text = cfg.Text
		if cfg.DelayMs > 0 {
			base.Params.DelayMs = cfg.DelayMs
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.Clear != nil {
			base.Params.Clear = cfg.Clear
		}
		if cfg.Submit != nil {
			base.Params.Submit = cfg.Submit
		}
	case compiler.StepShortcut:
		var cfg shortcutConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("shortcut node %s has invalid data: %w", step.NodeID, err)
		}

		combos := collectShortcutCombos(cfg.Shortcuts, cfg.Sequence, cfg.Keys, cfg.Shortcut)
		if len(combos) == 0 {
			return Instruction{}, fmt.Errorf("shortcut node %s missing shortcuts", step.NodeID)
		}

		normalized := make([]string, 0, len(combos))
		for _, combo := range combos {
			if normalizedCombo := normalizeShortcutCombo(combo); normalizedCombo != "" {
				normalized = append(normalized, normalizedCombo)
			}
		}

		if len(normalized) == 0 {
			return Instruction{}, fmt.Errorf("shortcut node %s produced no valid shortcuts", step.NodeID)
		}

		base.Params.ShortcutKeys = normalized
		if cfg.DelayMs > 0 {
			base.Params.ShortcutDelayMs = cfg.DelayMs
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if trimmed := strings.TrimSpace(cfg.FocusSelector); trimmed != "" {
			base.Params.FocusSelector = trimmed
		}
	case compiler.StepExtract:
		var cfg extractConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("extract node %s has invalid data: %w", step.NodeID, err)
		}
		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" {
			return Instruction{}, fmt.Errorf("extract node %s missing selector", step.NodeID)
		}
		base.Params.Selector = selector
		typeValue := strings.ToLower(strings.TrimSpace(cfg.ExtractType))
		if typeValue == "" {
			typeValue = "text"
		}
		base.Params.ExtractType = typeValue
		if strings.TrimSpace(cfg.Attribute) != "" {
			base.Params.Attribute = strings.TrimSpace(cfg.Attribute)
		}
		if cfg.AllMatches != nil {
			base.Params.AllMatches = cfg.AllMatches
		}
	case compiler.StepWorkflowCall:
		return Instruction{}, fmt.Errorf("workflowCall node %s is not yet supported", step.NodeID)
	case compiler.StepAssert:
		var cfg assertConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("assert node %s has invalid data: %w", step.NodeID, err)
		}

		mode := normalizeAssertMode(firstNonEmpty(cfg.AssertMode, cfg.Mode, cfg.Comparison))
		if mode == "" {
			mode = "exists"
		}

		switch mode {
		case "exists", "not_exists", "text_equals", "text_contains", "attribute_equals", "attribute_contains", "expression":
		default:
			return Instruction{}, fmt.Errorf("assert node %s has unsupported mode %q", step.NodeID, mode)
		}

		selector := strings.TrimSpace(cfg.Selector)
		if selector == "" && mode != "expression" {
			return Instruction{}, fmt.Errorf("assert node %s requires selector for mode %s", step.NodeID, mode)
		}

		base.Params.AssertMode = mode
		if selector != "" {
			base.Params.Selector = selector
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.CaseSensitive != nil {
			base.Params.CaseSensitive = cfg.CaseSensitive
		}
		if cfg.Negate != nil {
			base.Params.Negate = cfg.Negate
		}
		if cfg.ContinueOnFailure != nil {
			base.Params.ContinueOnFailure = cfg.ContinueOnFailure
		}
		if trimmed := strings.TrimSpace(cfg.FailureMessage); trimmed != "" {
			base.Params.FailureMessage = trimmed
		}
		if trimmed := strings.TrimSpace(cfg.Expression); trimmed != "" {
			base.Params.Expression = trimmed
		}
		if attr := strings.TrimSpace(cfg.Attribute); attr != "" {
			base.Params.Attribute = attr
		}

		expected := cfg.ExpectedValue
		if expected == nil && cfg.Expected != nil {
			expected = cfg.Expected
		}
		if expected == nil && strings.TrimSpace(cfg.Text) != "" {
			expected = strings.TrimSpace(cfg.Text)
		}

		if needsExpectedValue(mode) && expected == nil {
			return Instruction{}, fmt.Errorf("assert node %s requires expected value for mode %s", step.NodeID, mode)
		}

		if expected != nil {
			base.Params.ExpectedValue = expected
		}
	case compiler.StepCustom:
		return Instruction{}, fmt.Errorf("custom node %s is not yet supported", step.NodeID)
	default:
		return Instruction{}, fmt.Errorf("step type %q is not supported", step.Type)
	}

	return base, nil
}

func normalizeStringSlice(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

var shortcutKeyAliases = map[string]string{
	"ctrl":       "Control",
	"control":    "Control",
	"cmd":        "Meta",
	"command":    "Meta",
	"meta":       "Meta",
	"super":      "Meta",
	"windows":    "Meta",
	"win":        "Meta",
	"alt":        "Alt",
	"option":     "Alt",
	"shift":      "Shift",
	"enter":      "Enter",
	"return":     "Enter",
	"esc":        "Escape",
	"escape":     "Escape",
	"space":      "Space",
	"spacebar":   "Space",
	"tab":        "Tab",
	"backspace":  "Backspace",
	"delete":     "Delete",
	"del":        "Delete",
	"home":       "Home",
	"end":        "End",
	"pageup":     "PageUp",
	"pagedown":   "PageDown",
	"arrowup":    "ArrowUp",
	"arrowdown":  "ArrowDown",
	"arrowleft":  "ArrowLeft",
	"arrowright": "ArrowRight",
	"up":         "ArrowUp",
	"down":       "ArrowDown",
	"left":       "ArrowLeft",
	"right":      "ArrowRight",
	"plus":       "+",
}

func collectShortcutCombos(shortcuts, sequence, keys []string, single string) []string {
	combos := make([]string, 0, len(shortcuts)+len(sequence)+len(keys)+1)
	seen := make(map[string]struct{})
	add := func(values []string) {
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			combos = append(combos, trimmed)
		}
	}

	add(shortcuts)
	add(sequence)
	add(keys)

	if trimmed := strings.TrimSpace(single); trimmed != "" {
		if _, exists := seen[trimmed]; !exists {
			combos = append(combos, trimmed)
		}
	}

	return combos
}

func normalizeShortcutCombo(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	segments := strings.Split(trimmed, "+")
	result := make([]string, 0, len(segments))

	for _, segment := range segments {
		cleaned := strings.TrimSpace(segment)
		if cleaned == "" {
			continue
		}

		lower := strings.ToLower(cleaned)
		lower = strings.ReplaceAll(lower, "-", "")
		if alias, ok := shortcutKeyAliases[lower]; ok {
			result = append(result, alias)
			continue
		}

		if normalizedArrow := normalizeArrowKey(lower); normalizedArrow != "" {
			result = append(result, normalizedArrow)
			continue
		}

		if isFunctionKey(lower) {
			result = append(result, strings.ToUpper(lower))
			continue
		}

		if len(cleaned) == 1 {
			runeVal := []rune(cleaned)[0]
			if unicode.IsLetter(runeVal) {
				result = append(result, strings.ToUpper(cleaned))
				continue
			}
			if unicode.IsDigit(runeVal) || cleaned == "+" || cleaned == "/" || cleaned == "-" {
				result = append(result, cleaned)
				continue
			}
		}

		result = append(result, strings.ToUpper(lower[:1])+lower[1:])
	}

	if len(result) == 0 {
		return ""
	}

	return strings.Join(result, "+")
}

func normalizeArrowKey(value string) string {
	switch value {
	case "arrowup", "up":
		return "ArrowUp"
	case "arrowdown", "down":
		return "ArrowDown"
	case "arrowleft", "left":
		return "ArrowLeft"
	case "arrowright", "right":
		return "ArrowRight"
	default:
		return ""
	}
}

func isFunctionKey(value string) bool {
	if len(value) < 2 || value[0] != 'f' {
		return false
	}
	for _, r := range value[1:] {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func getIntParam(params map[string]any, keys ...string) (int, bool) {
	if params == nil {
		return 0, false
	}
	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int:
			return v, true
		case int32:
			return int(v), true
		case int64:
			return int(v), true
		case float64:
			return int(v), true
		case float32:
			return int(v), true
		case json.Number:
			if i, err := v.Int64(); err == nil {
				return int(i), true
			}
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			if i, err := strconv.Atoi(trimmed); err == nil {
				return i, true
			}
		}
	}
	return 0, false
}

func getFloatParam(params map[string]any, keys ...string) (float64, bool) {
	if params == nil {
		return 0, false
	}
	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int32:
			return float64(v), true
		case int64:
			return float64(v), true
		case json.Number:
			if f, err := v.Float64(); err == nil {
				return f, true
			}
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
				return f, true
			}
		}
	}
	return 0, false
}

func getBoolParam(params map[string]any, keys ...string) (bool, bool) {
	if params == nil {
		return false, false
	}
	for _, key := range keys {
		value, ok := params[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v, true
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			if parsed, err := strconv.ParseBool(trimmed); err == nil {
				return parsed, true
			}
		case int:
			return v != 0, true
		case int32:
			return v != 0, true
		case int64:
			return v != 0, true
		case float32:
			return v != 0, true
		case float64:
			return v != 0, true
		case json.Number:
			if i, err := v.Int64(); err == nil {
				return i != 0, true
			}
		}
	}
	return false, false
}

func decodeParams(src map[string]any, target any) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeAssertMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "exists", "selector_exists", "success":
		return "exists"
	case "not_exists", "missing", "absent", "failure":
		return "not_exists"
	case "text", "text_equals", "equals", "equal", "text_equal":
		return "text_equals"
	case "text_contains", "contains", "text_contains_any":
		return "text_contains"
	case "attribute", "attribute_equals":
		return "attribute_equals"
	case "attribute_contains":
		return "attribute_contains"
	case "expression", "script":
		return "expression"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func needsExpectedValue(mode string) bool {
	switch mode {
	case "text_equals", "text_contains", "attribute_equals", "attribute_contains":
		return true
	default:
		return false
	}
}
