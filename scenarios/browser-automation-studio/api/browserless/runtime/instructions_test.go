package runtime

import (
	"context"
	"os"
	"path/filepath"
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

func TestInstructionFromStepUploadFileSuccess(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "avatar.png")
	if err := os.WriteFile(filePath, []byte("png"), 0o600); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	step := compiler.ExecutionStep{
		Index:  5,
		NodeID: "upload-1",
		Type:   compiler.StepUploadFile,
		Params: map[string]any{
			"selector":  "  #file-input  ",
			"filePath":  filePath,
			"timeoutMs": 1234,
			"waitForMs": 500,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating uploadFile instruction: %v", err)
	}
	if inst.Params.Selector != "#file-input" {
		t.Fatalf("expected selector to be trimmed, got %q", inst.Params.Selector)
	}
	if len(inst.Params.FilePaths) != 1 || inst.Params.FilePaths[0] != filePath {
		t.Fatalf("expected filePaths to contain %q, got %v", filePath, inst.Params.FilePaths)
	}
	if inst.Params.FilePath != filePath {
		t.Fatalf("expected filePath to be set, got %q", inst.Params.FilePath)
	}
	if inst.Params.TimeoutMs != 1234 {
		t.Fatalf("expected timeout to propagate, got %d", inst.Params.TimeoutMs)
	}
	if inst.Params.WaitForMs != 500 {
		t.Fatalf("expected waitFor to propagate, got %d", inst.Params.WaitForMs)
	}
}

func TestInstructionFromStepUploadFileMultiplePaths(t *testing.T) {
	tempDir := t.TempDir()
	first := filepath.Join(tempDir, "first.txt")
	second := filepath.Join(tempDir, "second.txt")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
			t.Fatalf("failed to seed temp file: %v", err)
		}
	}
	step := compiler.ExecutionStep{
		Index:  6,
		NodeID: "upload-2",
		Type:   compiler.StepUploadFile,
		Params: map[string]any{
			"selector":  "#files",
			"filePaths": []any{first, second},
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating uploadFile instruction: %v", err)
	}
	if len(inst.Params.FilePaths) != 2 {
		t.Fatalf("expected two file paths, got %d", len(inst.Params.FilePaths))
	}
	if inst.Params.FilePaths[0] != first || inst.Params.FilePaths[1] != second {
		t.Fatalf("expected file paths to preserve order, got %v", inst.Params.FilePaths)
	}
}

func TestInstructionFromStepUploadFileValidation(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  7,
		NodeID: "upload-3",
		Type:   compiler.StepUploadFile,
		Params: map[string]any{
			"selector": "#files",
			"filePath": "relative/path.txt",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when file path is relative")
	}

	tempDir := t.TempDir()
	missing := filepath.Join(tempDir, "missing.txt")
	step.Params["filePath"] = missing
	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when file is missing")
	}

	step.Params["selector"] = ""
	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when selector missing")
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

func TestInstructionFromStepSetVariableStatic(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  10,
		NodeID: "var-1",
		Type:   compiler.StepSetVariable,
		Params: map[string]any{
			"name":       "greeting",
			"sourceType": "static",
			"value":      " Hello ",
			"valueType":  "text",
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error building setVariable instruction: %v", err)
	}
	if inst.Params.VariableName != "greeting" {
		t.Fatalf("expected variable name to be greeting, got %q", inst.Params.VariableName)
	}
	if inst.Params.VariableSource != "static" {
		t.Fatalf("expected static source, got %q", inst.Params.VariableSource)
	}
	if inst.Params.VariableValue != " Hello " {
		t.Fatalf("expected raw value to remain intact, got %+v", inst.Params.VariableValue)
	}
}

func TestInstructionFromStepSetVariableMissingName(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  11,
		NodeID: "var-2",
		Type:   compiler.StepSetVariable,
		Params: map[string]any{
			"sourceType": "static",
			"value":      "hi",
		},
	}
	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when variable name missing")
	}
}

func TestInstructionFromStepUseVariableDefaults(t *testing.T) {
	required := true
	step := compiler.ExecutionStep{
		Index:  12,
		NodeID: "use-1",
		Type:   compiler.StepUseVariable,
		Params: map[string]any{
			"name":      "username",
			"transform": "Hello, {{value}}!",
			"required":  required,
		},
	}
	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.Params.StoreResult != "username" {
		t.Fatalf("expected default storeResult to match name, got %q", inst.Params.StoreResult)
	}
	if inst.Params.VariableTransform != "Hello, {{value}}!" {
		t.Fatalf("unexpected transform %q", inst.Params.VariableTransform)
	}
	if inst.Params.VariableRequired == nil || !*inst.Params.VariableRequired {
		t.Fatalf("expected required flag to be set")
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

func TestInstructionFromStepDragDropSuccess(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  12,
		NodeID: "drag-1",
		Type:   compiler.StepDragDrop,
		Params: map[string]any{
			"sourceSelector": "  .card:nth-child(2)  ",
			"targetSelector": "#drop-zone",
			"holdMs":         275,
			"steps":          80,
			"durationMs":     25000,
			"offsetX":        6400,
			"offsetY":        -6400,
			"timeoutMs":      4200,
			"waitForMs":      375,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating dragDrop instruction: %v", err)
	}
	if inst.Params.DragSourceSelector != ".card:nth-child(2)" {
		t.Fatalf("expected trimmed source selector, got %q", inst.Params.DragSourceSelector)
	}
	if inst.Params.DragTargetSelector != "#drop-zone" {
		t.Fatalf("expected target selector, got %q", inst.Params.DragTargetSelector)
	}
	if inst.Params.DragHoldMs != 275 {
		t.Fatalf("expected holdMs to remain 275, got %d", inst.Params.DragHoldMs)
	}
	if inst.Params.DragSteps != maxDragSteps {
		t.Fatalf("expected steps to clamp to %d, got %d", maxDragSteps, inst.Params.DragSteps)
	}
	if inst.Params.DragDurationMs != maxDragDurationMs {
		t.Fatalf("expected duration to clamp to %d, got %d", maxDragDurationMs, inst.Params.DragDurationMs)
	}
	if inst.Params.DragOffsetX != maxDragOffset {
		t.Fatalf("expected offsetX to clamp to %d, got %d", maxDragOffset, inst.Params.DragOffsetX)
	}
	if inst.Params.DragOffsetY != minDragOffset {
		t.Fatalf("expected offsetY to clamp to %d, got %d", minDragOffset, inst.Params.DragOffsetY)
	}
	if inst.Params.TimeoutMs != 4200 {
		t.Fatalf("expected timeoutMs to be set, got %d", inst.Params.TimeoutMs)
	}
	if inst.Params.WaitForMs != 375 {
		t.Fatalf("expected waitForMs to be set, got %d", inst.Params.WaitForMs)
	}
}

func TestInstructionFromStepDragDropDefaultsAndValidation(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  13,
		NodeID: "drag-2",
		Type:   compiler.StepDragDrop,
		Params: map[string]any{
			"sourceSelector": "#source",
			"targetSelector": "#target",
			"holdMs":         -10,
			"steps":          0,
			"durationMs":     0,
			"offsetX":        -99999,
			"offsetY":        99999,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating dragDrop instruction: %v", err)
	}
	if inst.Params.DragHoldMs != defaultDragHoldMs {
		t.Fatalf("expected default hold %d, got %d", defaultDragHoldMs, inst.Params.DragHoldMs)
	}
	if inst.Params.DragSteps != defaultDragSteps {
		t.Fatalf("expected default steps %d, got %d", defaultDragSteps, inst.Params.DragSteps)
	}
	if inst.Params.DragDurationMs != defaultDragDurationMs {
		t.Fatalf("expected default duration %d, got %d", defaultDragDurationMs, inst.Params.DragDurationMs)
	}
	if inst.Params.DragOffsetX != minDragOffset {
		t.Fatalf("expected offsetX to clamp to %d, got %d", minDragOffset, inst.Params.DragOffsetX)
	}
	if inst.Params.DragOffsetY != maxDragOffset {
		t.Fatalf("expected offsetY to clamp to %d, got %d", maxDragOffset, inst.Params.DragOffsetY)
	}

	missingSource := compiler.ExecutionStep{
		Index:  14,
		NodeID: "drag-3",
		Type:   compiler.StepDragDrop,
		Params: map[string]any{
			"targetSelector": "#t",
		},
	}
	if _, err := instructionFromStep(context.Background(), missingSource); err == nil {
		t.Fatalf("expected error when source selector missing")
	}
	missingTarget := compiler.ExecutionStep{
		Index:  15,
		NodeID: "drag-4",
		Type:   compiler.StepDragDrop,
		Params: map[string]any{
			"sourceSelector": "#s",
		},
	}
	if _, err := instructionFromStep(context.Background(), missingTarget); err == nil {
		t.Fatalf("expected error when target selector missing")
	}
}

func TestInstructionFromStepFocusRequiresSelector(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  11,
		NodeID: "focus-1",
		Type:   compiler.StepFocus,
		Params: map[string]any{
			"selector": "  ",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when focus selector is missing")
	}
}

func TestInstructionFromStepFocusAppliesTiming(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  12,
		NodeID: "focus-2",
		Type:   compiler.StepFocus,
		Params: map[string]any{
			"selector":  "#email",
			"timeoutMs": 4200,
			"waitForMs": 260,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error building focus instruction: %v", err)
	}

	if inst.Params.Selector != "#email" {
		t.Fatalf("expected selector to be set, got %q", inst.Params.Selector)
	}
	if inst.Params.TimeoutMs != 4200 {
		t.Fatalf("expected timeout to propagate, got %d", inst.Params.TimeoutMs)
	}
	if inst.Params.WaitForMs != 260 {
		t.Fatalf("expected waitForMs to propagate, got %d", inst.Params.WaitForMs)
	}
}

func TestInstructionFromStepBlurDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  13,
		NodeID: "blur-1",
		Type:   compiler.StepBlur,
		Params: map[string]any{
			"selector": "input[name=email]",
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error building blur instruction: %v", err)
	}

	if inst.Params.Selector != "input[name=email]" {
		t.Fatalf("expected selector to be applied, got %q", inst.Params.Selector)
	}
	if inst.Params.TimeoutMs != 0 {
		t.Fatalf("expected timeout to remain unset, got %d", inst.Params.TimeoutMs)
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

func TestInstructionFromStepSelectValue(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  15,
		NodeID: "select-1",
		Type:   compiler.StepSelect,
		Params: map[string]any{
			"selector":  " select.payment ",
			"selectBy":  "value",
			"value":     " visa ",
			"timeoutMs": 4500,
			"waitForMs": 200,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating select instruction: %v", err)
	}

	if inst.Params.Selector != "select.payment" {
		t.Fatalf("expected selector to be trimmed, got %q", inst.Params.Selector)
	}
	if inst.Params.SelectionMode != "value" {
		t.Fatalf("expected selection mode value, got %q", inst.Params.SelectionMode)
	}
	if inst.Params.OptionValue != "visa" {
		t.Fatalf("expected option value visa, got %q", inst.Params.OptionValue)
	}
	if inst.Params.TimeoutMs != 4500 {
		t.Fatalf("expected timeout 4500, got %d", inst.Params.TimeoutMs)
	}
	if inst.Params.WaitForMs != 200 {
		t.Fatalf("expected waitFor 200, got %d", inst.Params.WaitForMs)
	}
}

func TestInstructionFromStepSelectMulti(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  16,
		NodeID: "select-2",
		Type:   compiler.StepSelect,
		Params: map[string]any{
			"selector": ".tags",
			"multiple": true,
			"values":   []string{"  primary  ", "Secondary"},
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating multi-select instruction: %v", err)
	}

	if !inst.Params.MultiSelect {
		t.Fatalf("expected multiSelect flag to be true")
	}
	if inst.Params.SelectionMode != "value" {
		t.Fatalf("expected value mode for multi-select default, got %q", inst.Params.SelectionMode)
	}
	values := inst.Params.OptionValues
	if len(values) != 2 || values[0] != "primary" || values[1] != "Secondary" {
		t.Fatalf("unexpected option values %#v", values)
	}
}

func TestInstructionFromStepSelectIndexValidation(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  17,
		NodeID: "select-3",
		Type:   compiler.StepSelect,
		Params: map[string]any{
			"selector": "select.plan",
			"selectBy": "index",
			"index":    -1,
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when index is negative")
	}
}

func TestInstructionFromStepSelectMultiRequiresValues(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  18,
		NodeID: "select-4",
		Type:   compiler.StepSelect,
		Params: map[string]any{
			"selector": "select.roles",
			"multiple": true,
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when multi-select lacks values")
	}
}
