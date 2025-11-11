package runtime

import (
	"context"
	"testing"

	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
)

func TestInstructionFromStepScenario(t *testing.T) {
	restore := scenarioport.SetPortLookupFuncForTests(func(ctx context.Context, scenario string, port string) (int, error) {
		return 4242, nil
	})
	defer restore()

	step := compiler.ExecutionStep{
		Index:  0,
		NodeID: "navigate-1",
		Type:   compiler.StepNavigate,
		Params: map[string]any{
			"destinationType": "scenario",
			"scenario":        "app-monitor",
			"scenarioPath":    "/dashboard",
		},
	}

	instruction, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("instructionFromStep returned error: %v", err)
	}

	expectedURL := "http://localhost:4242/dashboard"
	if instruction.Params.URL != expectedURL {
		t.Fatalf("expected resolved URL %q, got %q", expectedURL, instruction.Params.URL)
	}
}

func TestInstructionFromStepScenarioMissingName(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  0,
		NodeID: "navigate-2",
		Type:   compiler.StepNavigate,
		Params: map[string]any{
			"destinationType": "scenario",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when scenario name is missing")
	}
}

func TestInstructionFromStepURLFallback(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  0,
		NodeID: "navigate-3",
		Type:   compiler.StepNavigate,
		Params: map[string]any{
			"url": " https://example.com ",
		},
	}

	instruction, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error converting navigate step: %v", err)
	}

	if instruction.Params.URL != "https://example.com" {
		t.Fatalf("expected URL to be trimmed, got %q", instruction.Params.URL)
	}
}

func TestInstructionFromStepEvaluateSuccess(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  3,
		NodeID: "script-1",
		Type:   compiler.StepEvaluate,
		Params: map[string]any{
			"expression":  " return document.title ",
			"timeoutMs":   1500,
			"storeResult": " pageTitle ",
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating evaluate instruction: %v", err)
	}

	if inst.Params.Expression != "return document.title" {
		t.Fatalf("expected expression to be trimmed, got %q", inst.Params.Expression)
	}
	if inst.Params.TimeoutMs != 1500 {
		t.Fatalf("expected timeout to propagate, got %d", inst.Params.TimeoutMs)
	}
	if inst.Params.StoreResult != "pageTitle" {
		t.Fatalf("expected storeResult to be trimmed, got %q", inst.Params.StoreResult)
	}
}

func TestInstructionFromStepEvaluateMissingExpression(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  4,
		NodeID: "script-2",
		Type:   compiler.StepEvaluate,
		Params: map[string]any{
			"expression": "   ",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when expression is empty")
	}
}

func TestInstructionFromStepKeyboardKeydown(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  5,
		NodeID: "keyboard-1",
		Type:   compiler.StepKeyboard,
		Params: map[string]any{
			"key":       "Enter",
			"eventType": "keydown",
			"delayMs":   200,
			"timeoutMs": 4500,
			"modifiers": map[string]any{
				"ctrl": true,
				"alt":  true,
			},
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating keyboard instruction: %v", err)
	}

	if inst.Params.KeyValue != "Enter" {
		t.Fatalf("expected key value to be Enter, got %q", inst.Params.KeyValue)
	}
	if inst.Params.KeyEventType != "keydown" {
		t.Fatalf("expected event type to be keydown, got %q", inst.Params.KeyEventType)
	}
	if inst.Params.DelayMs != 200 {
		t.Fatalf("expected delay to propagate, got %d", inst.Params.DelayMs)
	}
	if inst.Params.TimeoutMs != 4500 {
		t.Fatalf("expected timeout to propagate, got %d", inst.Params.TimeoutMs)
	}
	mods := inst.Params.KeyModifiers
	if len(mods) != 2 {
		t.Fatalf("expected two modifiers, got %v", mods)
	}
	modSet := map[string]bool{}
	for _, mod := range mods {
		modSet[mod] = true
	}
	for _, expected := range []string{"ctrl", "alt"} {
		if !modSet[expected] {
			t.Fatalf("expected modifier %s to be set", expected)
		}
	}
}

func TestInstructionFromStepKeyboardDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  6,
		NodeID: "keyboard-2",
		Type:   compiler.StepKeyboard,
		Params: map[string]any{
			"key": "a",
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating keyboard instruction: %v", err)
	}

	if inst.Params.KeyEventType != "keypress" {
		t.Fatalf("expected default event type to be keypress, got %q", inst.Params.KeyEventType)
	}
	if inst.Params.KeyModifiers != nil {
		t.Fatalf("expected no modifiers, got %v", inst.Params.KeyModifiers)
	}
}

func TestInstructionFromStepKeyboardMissingKey(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  7,
		NodeID: "keyboard-3",
		Type:   compiler.StepKeyboard,
		Params: map[string]any{
			"key": "  ",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when key is blank")
	}
}

func TestInstructionFromStepHoverClampsParams(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  8,
		NodeID: "hover-1",
		Type:   compiler.StepHover,
		Params: map[string]any{
			"selector":   "  #menu ",
			"timeoutMs":  1500,
			"waitForMs":  200,
			"steps":      maxHoverSteps + 10,
			"durationMs": maxHoverDurationMs + 5000,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating hover instruction: %v", err)
	}

	if inst.Params.Selector != "#menu" {
		t.Fatalf("expected selector to be trimmed, got %q", inst.Params.Selector)
	}
	if inst.Params.TimeoutMs != 1500 {
		t.Fatalf("expected timeout to propagate, got %d", inst.Params.TimeoutMs)
	}
	if inst.Params.WaitForMs != 200 {
		t.Fatalf("expected waitFor to propagate, got %d", inst.Params.WaitForMs)
	}
	if inst.Params.MovementSteps != maxHoverSteps {
		t.Fatalf("expected steps to clamp to %d, got %d", maxHoverSteps, inst.Params.MovementSteps)
	}
	if inst.Params.DurationMs != maxHoverDurationMs {
		t.Fatalf("expected duration to clamp to %d, got %d", maxHoverDurationMs, inst.Params.DurationMs)
	}
}

func TestInstructionFromStepHoverDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  9,
		NodeID: "hover-2",
		Type:   compiler.StepHover,
		Params: map[string]any{
			"selector": "#menu",
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating hover instruction: %v", err)
	}

	if inst.Params.MovementSteps != defaultHoverSteps {
		t.Fatalf("expected default steps %d, got %d", defaultHoverSteps, inst.Params.MovementSteps)
	}
	if inst.Params.DurationMs != defaultHoverDurationMs {
		t.Fatalf("expected default duration %d, got %d", defaultHoverDurationMs, inst.Params.DurationMs)
	}
}

func TestInstructionFromStepHoverMissingSelector(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  10,
		NodeID: "hover-3",
		Type:   compiler.StepHover,
		Params: map[string]any{},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when selector is missing")
	}
}

func TestInstructionFromStepScrollPageDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  11,
		NodeID: "scroll-1",
		Type:   compiler.StepScroll,
		Params: map[string]any{},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating scroll instruction: %v", err)
	}

	if inst.Params.ScrollType != "page" {
		t.Fatalf("expected scroll type page, got %q", inst.Params.ScrollType)
	}
	if inst.Params.ScrollDirection != "down" {
		t.Fatalf("expected default direction down, got %q", inst.Params.ScrollDirection)
	}
	if inst.Params.ScrollAmount != defaultScrollAmount {
		t.Fatalf("expected default amount %d, got %d", defaultScrollAmount, inst.Params.ScrollAmount)
	}
	if inst.Params.ScrollBehavior != "auto" {
		t.Fatalf("expected default behavior auto, got %q", inst.Params.ScrollBehavior)
	}
}

func TestInstructionFromStepScrollElementRequiresSelector(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  12,
		NodeID: "scroll-2",
		Type:   compiler.StepScroll,
		Params: map[string]any{
			"scrollType": "element",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when selector missing for element scroll")
	}
}

func TestInstructionFromStepScrollUntilVisibleFallsBackToSelector(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  13,
		NodeID: "scroll-3",
		Type:   compiler.StepScroll,
		Params: map[string]any{
			"scrollType": "untilVisible",
			"selector":   "#lazy-item",
			"maxScrolls": 500,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating untilVisible instruction: %v", err)
	}

	if inst.Params.ScrollTargetSelector != "#lazy-item" {
		t.Fatalf("expected target selector fallback, got %q", inst.Params.ScrollTargetSelector)
	}
	if inst.Params.ScrollDirection != "down" {
		t.Fatalf("expected default direction down, got %q", inst.Params.ScrollDirection)
	}
	if inst.Params.ScrollMaxAttempts != maxScrollAttempts {
		t.Fatalf("expected attempts to clamp to %d, got %d", maxScrollAttempts, inst.Params.ScrollMaxAttempts)
	}
}

func TestInstructionFromStepScrollPositionClampsCoordinates(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  14,
		NodeID: "scroll-4",
		Type:   compiler.StepScroll,
		Params: map[string]any{
			"scrollType": "position",
			"x":          maxScrollCoordinate + 1000,
			"y":          minScrollCoordinate - 1000,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating position scroll instruction: %v", err)
	}

	if inst.Params.ScrollX != maxScrollCoordinate {
		t.Fatalf("expected x to clamp to %d, got %d", maxScrollCoordinate, inst.Params.ScrollX)
	}
	if inst.Params.ScrollY != minScrollCoordinate {
		t.Fatalf("expected y to clamp to %d, got %d", minScrollCoordinate, inst.Params.ScrollY)
	}
}
