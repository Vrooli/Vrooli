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

func TestInstructionFromStepResilienceConfig(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  1,
		NodeID: "click-1",
		Type:   compiler.StepClick,
		Params: map[string]any{
			"selector": "#submit",
			"resilience": map[string]any{
				"maxAttempts":           4,
				"delayMs":               750,
				"backoffFactor":         2.0,
				"preconditionSelector":  ".ready",
				"preconditionTimeoutMs": 9000,
				"preconditionWaitMs":    150,
				"successSelector":       ".complete",
				"successTimeoutMs":      12000,
				"successWaitMs":         400,
			},
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("instructionFromStep returned error: %v", err)
	}

	if inst.Params.RetryAttempts != 3 {
		t.Fatalf("expected retryAttempts 3, got %d", inst.Params.RetryAttempts)
	}
	if inst.Params.RetryDelayMs != 750 {
		t.Fatalf("expected retryDelayMs 750, got %d", inst.Params.RetryDelayMs)
	}
	if inst.Params.RetryBackoffFactor != 2.0 {
		t.Fatalf("expected retryBackoffFactor 2.0, got %f", inst.Params.RetryBackoffFactor)
	}
	if inst.Params.PreconditionSelector != ".ready" {
		t.Fatalf("expected precondition selector .ready, got %s", inst.Params.PreconditionSelector)
	}
	if inst.Params.PreconditionTimeoutMs != 9000 {
		t.Fatalf("expected precondition timeout 9000, got %d", inst.Params.PreconditionTimeoutMs)
	}
	if inst.Params.PreconditionWaitMs != 150 {
		t.Fatalf("expected precondition wait 150, got %d", inst.Params.PreconditionWaitMs)
	}
	if inst.Params.SuccessSelector != ".complete" {
		t.Fatalf("expected success selector .complete, got %s", inst.Params.SuccessSelector)
	}
	if inst.Params.SuccessTimeoutMs != 12000 {
		t.Fatalf("expected success timeout 12000, got %d", inst.Params.SuccessTimeoutMs)
	}
	if inst.Params.SuccessWaitMs != 400 {
		t.Fatalf("expected success wait 400, got %d", inst.Params.SuccessWaitMs)
	}
}

func TestInstructionFromStepClickDefaultResilience(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  2,
		NodeID: "click-default",
		Type:   compiler.StepClick,
		Params: map[string]any{
			"selector":  "#submit",
			"timeoutMs": 4000,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating click instruction: %v", err)
	}

	if inst.Params.RetryAttempts != defaultResilienceMaxAttempts-1 {
		t.Fatalf("expected default retry attempts %d, got %d", defaultResilienceMaxAttempts-1, inst.Params.RetryAttempts)
	}
	if inst.Params.RetryDelayMs != defaultResilienceRetryDelayMs {
		t.Fatalf("expected default retry delay %d, got %d", defaultResilienceRetryDelayMs, inst.Params.RetryDelayMs)
	}
	if inst.Params.PreconditionSelector != "#submit" {
		t.Fatalf("expected precondition selector to default to primary selector, got %q", inst.Params.PreconditionSelector)
	}
	if inst.Params.PreconditionTimeoutMs != 4000 {
		t.Fatalf("expected precondition timeout to reuse node timeout, got %d", inst.Params.PreconditionTimeoutMs)
	}
}

func TestInstructionFromStepClickResilienceOptOut(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  3,
		NodeID: "click-manual-resilience",
		Type:   compiler.StepClick,
		Params: map[string]any{
			"selector": "#submit",
			"resilience": map[string]any{
				"maxAttempts": 1,
			},
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating click instruction: %v", err)
	}
	if inst.Params.RetryAttempts != 0 {
		t.Fatalf("expected opt-out retry attempts to remain 0, got %d", inst.Params.RetryAttempts)
	}
	if inst.Params.PreconditionSelector != "#submit" {
		t.Fatalf("expected precondition selector fallback even when resilience configured, got %q", inst.Params.PreconditionSelector)
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

func TestInstructionFromStepSetCookieSuccess(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  7,
		NodeID: "cookie-set-1",
		Type:   compiler.StepSetCookie,
		Params: map[string]any{
			"name":       " sessionId ",
			"value":      "abc123",
			"url":        "https://example.com/app",
			"sameSite":   "LAX",
			"ttlSeconds": 90,
			"waitForMs":  150,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("setCookie should compile: %v", err)
	}
	if inst.Params.CookieName != "sessionId" {
		t.Fatalf("expected cookie name to trim whitespace, got %q", inst.Params.CookieName)
	}
	if inst.Params.CookieURL != "https://example.com/app" {
		t.Fatalf("expected url to persist, got %q", inst.Params.CookieURL)
	}
	if inst.Params.CookieSameSite != "lax" {
		t.Fatalf("expected sameSite normalization, got %q", inst.Params.CookieSameSite)
	}
	if inst.Params.CookieTTLSeconds != 90 {
		t.Fatalf("expected ttlSeconds to propagate, got %d", inst.Params.CookieTTLSeconds)
	}
	if inst.Params.WaitForMs != 150 {
		t.Fatalf("expected waitForMs to propagate, got %d", inst.Params.WaitForMs)
	}
}

func TestInstructionFromStepNetworkMockResponse(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  11,
		NodeID: "network-mock-1",
		Type:   compiler.StepNetworkMock,
		Params: map[string]any{
			"urlPattern": " https://api.example.com/* ",
			"method":     "post",
			"mockType":   "response",
			"statusCode": 202,
			"delayMs":    1200,
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
			"body": map[string]any{
				"ok": true,
			},
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating networkMock instruction: %v", err)
	}
	if inst.Params.NetworkURLPattern != "https://api.example.com/*" {
		t.Fatalf("expected trimmed pattern, got %q", inst.Params.NetworkURLPattern)
	}
	if inst.Params.NetworkMethod != "POST" {
		t.Fatalf("expected method to normalize to POST, got %q", inst.Params.NetworkMethod)
	}
	if inst.Params.NetworkMockType != "response" {
		t.Fatalf("expected mock type response, got %q", inst.Params.NetworkMockType)
	}
	if inst.Params.NetworkStatusCode != 202 {
		t.Fatalf("expected status code 202, got %d", inst.Params.NetworkStatusCode)
	}
	if inst.Params.NetworkDelayMs != 1200 {
		t.Fatalf("expected delay to persist, got %d", inst.Params.NetworkDelayMs)
	}
	headers := inst.Params.NetworkHeaders
	if len(headers) != 1 || headers["Content-Type"] != "application/json" {
		t.Fatalf("expected header map to normalize, got %v", headers)
	}
	body, ok := inst.Params.NetworkBody.(map[string]any)
	if !ok || body["ok"] != true {
		t.Fatalf("expected body to round-trip as map, got %#v", inst.Params.NetworkBody)
	}
}

func TestInstructionFromStepNetworkMockAbortDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  12,
		NodeID: "network-mock-2",
		Type:   compiler.StepNetworkMock,
		Params: map[string]any{
			"urlPattern":  "https://api.example.com/*",
			"mockType":    "abort",
			"abortReason": "blocked",
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating abort mock instruction: %v", err)
	}
	if inst.Params.NetworkMockType != "abort" {
		t.Fatalf("expected abort mock type, got %q", inst.Params.NetworkMockType)
	}
	if inst.Params.NetworkAbortReason != "BlockedByClient" {
		t.Fatalf("expected abort reason to normalize, got %q", inst.Params.NetworkAbortReason)
	}
	if inst.Params.NetworkMethod != "" {
		t.Fatalf("expected empty method when not provided, got %q", inst.Params.NetworkMethod)
	}
}

func TestInstructionFromStepNetworkMockDelayRequiresDuration(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  13,
		NodeID: "network-mock-3",
		Type:   compiler.StepNetworkMock,
		Params: map[string]any{
			"urlPattern": "https://example.com/*",
			"mockType":   "delay",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when delay mock omits delayMs")
	}
}

func TestInstructionFromStepNetworkMockMissingPattern(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  14,
		NodeID: "network-mock-4",
		Type:   compiler.StepNetworkMock,
		Params: map[string]any{
			"mockType": "response",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when urlPattern is missing")
	}
}

func TestInstructionFromStepSetCookieRequiresTarget(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  8,
		NodeID: "cookie-set-2",
		Type:   compiler.StepSetCookie,
		Params: map[string]any{
			"name":  "session",
			"value": "123",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when url and domain missing")
	}
}

func TestInstructionFromStepGetCookieDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  9,
		NodeID: "cookie-get-1",
		Type:   compiler.StepGetCookie,
		Params: map[string]any{
			"name": "auth",
		},
	}
	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("getCookie should compile: %v", err)
	}
	if inst.Params.CookieName != "auth" {
		t.Fatalf("expected cookie name to propagate")
	}
	if inst.Params.CookieResultFormat != "value" {
		t.Fatalf("expected default format to be value, got %q", inst.Params.CookieResultFormat)
	}
	if inst.Params.StoreResult != "auth" {
		t.Fatalf("expected default storeResult to match name, got %q", inst.Params.StoreResult)
	}
}

func TestInstructionFromStepClearCookieAll(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  10,
		NodeID: "cookie-clear-1",
		Type:   compiler.StepClearCookie,
		Params: map[string]any{
			"clearAll": true,
		},
	}
	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("clearCookie should compile: %v", err)
	}
	if !inst.Params.CookieClearAll {
		t.Fatalf("expected clearAll flag to be set")
	}
}

func TestInstructionFromStepSetStorageNormalizesType(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  11,
		NodeID: "storage-set-1",
		Type:   compiler.StepSetStorage,
		Params: map[string]any{
			"storageType": "session",
			"key":         "token",
			"value":       "abc",
			"valueType":   "json",
			"timeoutMs":   8000,
		},
	}
	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("setStorage should compile: %v", err)
	}
	if inst.Params.StorageType != "sessionStorage" {
		t.Fatalf("expected storage type normalization, got %q", inst.Params.StorageType)
	}
	if inst.Params.StorageValueType != "json" {
		t.Fatalf("expected value type to remain json, got %q", inst.Params.StorageValueType)
	}
	if inst.Params.TimeoutMs != 8000 {
		t.Fatalf("expected timeout to propagate, got %d", inst.Params.TimeoutMs)
	}
}

func TestInstructionFromStepGetStorageDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  12,
		NodeID: "storage-get-1",
		Type:   compiler.StepGetStorage,
		Params: map[string]any{
			"key": "profile",
		},
	}
	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("getStorage should compile: %v", err)
	}
	if inst.Params.StorageType != "localStorage" {
		t.Fatalf("expected default storage to be localStorage, got %q", inst.Params.StorageType)
	}
	if inst.Params.StoreResult != "profile" {
		t.Fatalf("expected storeResult to default to key, got %q", inst.Params.StoreResult)
	}
	if inst.Params.StorageResultFormat != "text" {
		t.Fatalf("expected default result format text, got %q", inst.Params.StorageResultFormat)
	}
}

func TestInstructionFromStepClearStorageAll(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  13,
		NodeID: "storage-clear-1",
		Type:   compiler.StepClearStorage,
		Params: map[string]any{
			"clearAll": true,
		},
	}
	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("clearStorage should compile: %v", err)
	}
	if !inst.Params.StorageClearAll {
		t.Fatalf("expected storage clearAll flag to be set")
	}
}

func TestInstructionFromStepTabSwitchIndex(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  6,
		NodeID: "tab-switch-1",
		Type:   compiler.StepTabSwitch,
		Params: map[string]any{
			"switchBy":   "INDEX",
			"index":      2,
			"timeoutMs":  4500,
			"waitForNew": true,
			"closeOld":   true,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating tabSwitch instruction: %v", err)
	}
	if inst.Params.TabSwitchBy != "index" {
		t.Fatalf("expected switchBy=index, got %q", inst.Params.TabSwitchBy)
	}
	if inst.Params.TabIndex != 2 {
		t.Fatalf("expected tab index 2, got %d", inst.Params.TabIndex)
	}
	if !inst.Params.TabWaitForNew {
		t.Fatalf("expected waitForNew to be true")
	}
	if !inst.Params.TabCloseOld {
		t.Fatalf("expected closeOld to be true")
	}
	if inst.Params.TimeoutMs != 4500 {
		t.Fatalf("expected timeout to propagate, got %d", inst.Params.TimeoutMs)
	}
}

func TestInstructionFromStepTabSwitchTitleMissingPattern(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  7,
		NodeID: "tab-switch-2",
		Type:   compiler.StepTabSwitch,
		Params: map[string]any{
			"switchBy": "title",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when titleMatch missing")
	}
}

func TestInstructionFromStepFrameSwitchSelector(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  9,
		NodeID: "frame-switch-1",
		Type:   compiler.StepFrameSwitch,
		Params: map[string]any{
			"switchBy":  "selector",
			"selector":  " iframe#main ",
			"timeoutMs": 5000,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating frameSwitch instruction: %v", err)
	}
	if inst.Type != "frameSwitch" {
		t.Fatalf("expected instruction type frameSwitch, got %s", inst.Type)
	}
	if inst.Params.FrameSwitchBy != "selector" {
		t.Fatalf("expected switchBy=selector, got %q", inst.Params.FrameSwitchBy)
	}
	if inst.Params.FrameSelector != "iframe#main" {
		t.Fatalf("expected selector to be trimmed, got %q", inst.Params.FrameSelector)
	}
	if inst.Params.TimeoutMs != 5000 {
		t.Fatalf("expected timeout to propagate, got %d", inst.Params.TimeoutMs)
	}
}

func TestInstructionFromStepFrameSwitchIndexValidation(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  10,
		NodeID: "frame-switch-2",
		Type:   compiler.StepFrameSwitch,
		Params: map[string]any{
			"switchBy": "index",
			"index":    -1,
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when frame index is negative")
	}
}

func TestInstructionFromStepRotateDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  8,
		NodeID: "rotate-1",
		Type:   compiler.StepRotate,
		Params: map[string]any{},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating rotate instruction: %v", err)
	}
	if inst.Params.RotateOrientation != "portrait" {
		t.Fatalf("expected default orientation portrait, got %q", inst.Params.RotateOrientation)
	}
	if inst.Params.RotateAngle != 0 {
		t.Fatalf("expected default portrait angle 0, got %d", inst.Params.RotateAngle)
	}
	if inst.Params.WaitForMs != 0 {
		t.Fatalf("expected waitForMs to remain unset, got %d", inst.Params.WaitForMs)
	}
}

func TestInstructionFromStepRotateLandscapeAngle(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  9,
		NodeID: "rotate-2",
		Type:   compiler.StepRotate,
		Params: map[string]any{
			"orientation": "landscape",
			"angle":       270,
			"waitForMs":   650,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error creating rotate instruction: %v", err)
	}
	if inst.Params.RotateOrientation != "landscape" {
		t.Fatalf("expected orientation landscape, got %q", inst.Params.RotateOrientation)
	}
	if inst.Params.RotateAngle != 270 {
		t.Fatalf("expected angle 270, got %d", inst.Params.RotateAngle)
	}
	if inst.Params.WaitForMs != 650 {
		t.Fatalf("expected waitForMs to propagate, got %d", inst.Params.WaitForMs)
	}
}

func TestInstructionFromStepRotateInvalidOrientation(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  10,
		NodeID: "rotate-3",
		Type:   compiler.StepRotate,
		Params: map[string]any{
			"orientation": "diagonal",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error for unsupported orientation")
	}
}

func TestInstructionFromStepRotateAngleMismatch(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  11,
		NodeID: "rotate-4",
		Type:   compiler.StepRotate,
		Params: map[string]any{
			"orientation": "portrait",
			"angle":       90,
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when angle does not align with orientation")
	}
}

func TestInstructionFromStepGestureDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  12,
		NodeID: "gesture-1",
		Type:   compiler.StepGesture,
		Params: map[string]any{
			"gestureType": "swipe",
			"selector":    " #menu ",
			"direction":   "Left",
			"distance":    275,
			"durationMs":  640,
			"steps":       8,
			"timeoutMs":   4200,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("[REQ:BAS-NODE-GESTURE-MOBILE] expected gesture node to compile: %v", err)
	}
	if inst.Params.GestureType != "swipe" {
		t.Fatalf("gesture type should normalize to swipe, got %q", inst.Params.GestureType)
	}
	if inst.Params.GestureDirection != "left" {
		t.Fatalf("gesture direction should normalize to left, got %q", inst.Params.GestureDirection)
	}
	if inst.Params.GestureSelector != "#menu" {
		t.Fatalf("selector should be trimmed, got %q", inst.Params.GestureSelector)
	}
	if inst.Params.GestureDistance != 275 {
		t.Fatalf("distance should remain configured, got %d", inst.Params.GestureDistance)
	}
	if inst.Params.GestureDurationMs != 640 {
		t.Fatalf("duration should remain configured, got %d", inst.Params.GestureDurationMs)
	}
	if inst.Params.GestureSteps != 8 {
		t.Fatalf("steps should remain configured, got %d", inst.Params.GestureSteps)
	}
	if inst.Params.TimeoutMs != 4200 {
		t.Fatalf("timeout should propagate, got %d", inst.Params.TimeoutMs)
	}
	if inst.Params.GestureHasStart {
		t.Fatalf("gesture should not mark start coordinates when not provided")
	}
}

func TestInstructionFromStepGestureCoordinateFlags(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  13,
		NodeID: "gesture-2",
		Type:   compiler.StepGesture,
		Params: map[string]any{
			"gestureType": "pinch",
			"startX":      0,
			"startY":      0,
			"endY":        480,
			"scale":       1.6,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("expected gesture with coordinates to compile: %v", err)
	}
	if !inst.Params.GestureHasStart {
		t.Fatalf("expected gesture to flag explicit start coordinates")
	}
	if !inst.Params.GestureHasStartX || !inst.Params.GestureHasStartY {
		t.Fatalf("expected gesture to record which start axes were provided")
	}
	if inst.Params.GestureStartX != 0 || inst.Params.GestureStartY != 0 {
		t.Fatalf("expected start coordinates to remain zero, got %d,%d", inst.Params.GestureStartX, inst.Params.GestureStartY)
	}
	if !inst.Params.GestureHasEnd {
		t.Fatalf("expected gesture to flag explicit end coordinates")
	}
	if inst.Params.GestureHasEndX {
		t.Fatalf("did not expect endX to be marked when only Y provided")
	}
	if !inst.Params.GestureHasEndY {
		t.Fatalf("expected endY flag to be set")
	}
	if inst.Params.GestureEndY != 480 {
		t.Fatalf("expected endY to propagate, got %d", inst.Params.GestureEndY)
	}
	if inst.Params.GestureType != "pinch" {
		t.Fatalf("expected pinch type, got %q", inst.Params.GestureType)
	}
	if inst.Params.GestureScale <= 1 {
		t.Fatalf("expected pinch scale to remain >1 for zoom out, got %f", inst.Params.GestureScale)
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

func TestInstructionFromStepConditionalExpression(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  8,
		NodeID: "conditional-1",
		Type:   compiler.StepConditional,
		Params: map[string]any{
			"conditionType":  "expression",
			"expression":     " document.querySelector('#status') !== null ",
			"timeoutMs":      1500,
			"pollIntervalMs": 125,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("[REQ:BAS-NODE-CONDITIONAL] expected expression mode to compile, got error: %v", err)
	}
	if inst.Params.ConditionType != "expression" {
		t.Fatalf("expected condition type to normalize, got %q", inst.Params.ConditionType)
	}
	if inst.Params.ConditionExpression != "document.querySelector('#status') !== null" {
		t.Fatalf("expected expression to be trimmed, got %q", inst.Params.ConditionExpression)
	}
	if inst.Params.TimeoutMs != 1500 {
		t.Fatalf("expected timeout 1500, got %d", inst.Params.TimeoutMs)
	}
	if inst.Params.ConditionPollIntervalMs != 125 {
		t.Fatalf("expected poll interval 125, got %d", inst.Params.ConditionPollIntervalMs)
	}
}

func TestInstructionFromStepConditionalVariableMissingName(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  9,
		NodeID: "conditional-2",
		Type:   compiler.StepConditional,
		Params: map[string]any{
			"conditionType": "variable",
			"operator":      "equals",
			"value":         "ready",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("[REQ:BAS-NODE-CONDITIONAL] expected error when variable name missing")
	}
}

func TestInstructionFromStepLoopForEach(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  11,
		NodeID: "loop-foreach",
		Type:   compiler.StepLoop,
		Params: map[string]any{
			"loopType":      "forEach",
			"arraySource":   "rows",
			"maxIterations": 250,
			"itemVariable":  "row",
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("expected forEach loop to compile, got error: %v", err)
	}
	if inst.Params.LoopType != "foreach" {
		t.Fatalf("loop type should normalize to foreach, got %q", inst.Params.LoopType)
	}
	if inst.Params.LoopArraySource != "rows" {
		t.Fatalf("expected array source to propagate, got %q", inst.Params.LoopArraySource)
	}
	if inst.Params.LoopItemVariable != "row" {
		t.Fatalf("expected custom item variable, got %q", inst.Params.LoopItemVariable)
	}
	if inst.Params.LoopIndexVariable != defaultLoopIndexVariable {
		t.Fatalf("expected default index variable, got %q", inst.Params.LoopIndexVariable)
	}
	if inst.Params.LoopMaxIterations != 250 {
		t.Fatalf("expected max iterations to use provided value, got %d", inst.Params.LoopMaxIterations)
	}
}

func TestInstructionFromStepLoopTimeoutDefaults(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  14,
		NodeID: "loop-timeouts-default",
		Type:   compiler.StepLoop,
		Params: map[string]any{
			"loopType":    "forEach",
			"arraySource": "items",
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("[REQ:BAS-NODE-LOOP] expected defaults to apply when timeouts omitted: %v", err)
	}
	if inst.Params.LoopIterationTimeoutMs != defaultLoopIterationTimeoutMs {
		t.Fatalf("expected iteration timeout default %d, got %d", defaultLoopIterationTimeoutMs, inst.Params.LoopIterationTimeoutMs)
	}
	if inst.Params.LoopTotalTimeoutMs != defaultLoopTotalTimeoutMs {
		t.Fatalf("expected total timeout default %d, got %d", defaultLoopTotalTimeoutMs, inst.Params.LoopTotalTimeoutMs)
	}
}

func TestInstructionFromStepLoopTimeoutClamp(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  15,
		NodeID: "loop-timeouts-clamp",
		Type:   compiler.StepLoop,
		Params: map[string]any{
			"loopType":           "repeat",
			"count":              3,
			"iterationTimeoutMs": 10,
			"totalTimeoutMs":     900000000,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("[REQ:BAS-NODE-LOOP] expected repeat loop to compile for timeout clamp: %v", err)
	}
	if inst.Params.LoopIterationTimeoutMs != minLoopIterationTimeoutMs {
		t.Fatalf("expected iteration timeout to clamp to %d, got %d", minLoopIterationTimeoutMs, inst.Params.LoopIterationTimeoutMs)
	}
	if inst.Params.LoopTotalTimeoutMs != maxLoopTotalTimeoutMs {
		t.Fatalf("expected total timeout to clamp to %d, got %d", maxLoopTotalTimeoutMs, inst.Params.LoopTotalTimeoutMs)
	}
}

func TestInstructionFromStepLoopRepeatRequiresCount(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  12,
		NodeID: "loop-repeat",
		Type:   compiler.StepLoop,
		Params: map[string]any{
			"loopType": "repeat",
		},
	}
	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected repeat loop to require positive count")
	}
}

func TestInstructionFromStepLoopWhileMissingCondition(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  13,
		NodeID: "loop-while",
		Type:   compiler.StepLoop,
		Params: map[string]any{
			"loopType":      "while",
			"conditionType": "variable",
		},
	}
	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected while loop to require variable name")
	}

	step.Params["conditionVariable"] = "continueProcessing"
	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("expected while loop with condition variable to compile, got %v", err)
	}
	if inst.Params.LoopConditionType != "variable" {
		t.Fatalf("expected condition type to normalize to variable, got %q", inst.Params.LoopConditionType)
	}
	if inst.Params.LoopConditionVariable != "continueProcessing" {
		t.Fatalf("expected variable name to propagate, got %q", inst.Params.LoopConditionVariable)
	}
	if inst.Params.LoopConditionOperator == "" {
		t.Fatalf("expected default operator to be assigned")
	}
}

func TestInstructionFromStepWorkflowCallRequiresID(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  20,
		NodeID: "workflow-call-missing-id",
		Type:   compiler.StepWorkflowCall,
		Params: map[string]any{},
	}
	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected workflow call without workflowId or definition to error")
	}
}

func TestInstructionFromStepWorkflowCallParsesConfig(t *testing.T) {
	workflowID := "123e4567-e89b-12d3-a456-426614174000"
	step := compiler.ExecutionStep{
		Index:  21,
		NodeID: "workflow-call",
		Type:   compiler.StepWorkflowCall,
		Params: map[string]any{
			"workflowId":        workflowID,
			"workflowName":      "Shared Login",
			"waitForCompletion": true,
			"parameters": map[string]any{
				"username": "{{user}}",
				"retries":  3,
			},
			"outputMapping": map[string]any{
				"token": "authToken",
			},
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("expected workflow call instruction to compile: %v", err)
	}
	if inst.Params.WorkflowCallID != workflowID {
		t.Fatalf("expected workflow ID to propagate, got %q", inst.Params.WorkflowCallID)
	}
	if inst.Params.WorkflowCallName != "Shared Login" {
		t.Fatalf("expected workflow name to propagate, got %q", inst.Params.WorkflowCallName)
	}
	if inst.Params.WorkflowCallWait == nil || !*inst.Params.WorkflowCallWait {
		t.Fatalf("expected workflow call to default to wait for completion")
	}
	if inst.Params.WorkflowCallParams == nil || inst.Params.WorkflowCallParams["username"] != "{{user}}" {
		t.Fatalf("expected parameters to persist, got %+v", inst.Params.WorkflowCallParams)
	}
	if inst.Params.WorkflowCallOutputs == nil || inst.Params.WorkflowCallOutputs["token"] != "authToken" {
		t.Fatalf("expected output mapping to persist, got %+v", inst.Params.WorkflowCallOutputs)
	}
}

func TestInstructionFromStepWorkflowCallInlineDefinition(t *testing.T) {
	inlineDefinition := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":       "inline-nav",
				"type":     "navigate",
				"position": map[string]any{"x": 0, "y": 0},
				"data": map[string]any{
					"destinationType": "url",
					"url":             "https://example.com",
				},
			},
		},
		"edges": []any{},
	}
	step := compiler.ExecutionStep{
		Index:  22,
		NodeID: "workflow-call-inline",
		Type:   compiler.StepWorkflowCall,
		Params: map[string]any{
			"workflowName":       "Inline Flow",
			"workflowDefinition": inlineDefinition,
		},
	}

	inst, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("expected inline workflow call to compile: %v", err)
	}
	if inst.Params.WorkflowCallDefinition == nil {
		t.Fatalf("expected inline definition to propagate")
	}
	if inst.Params.WorkflowCallID != "" {
		t.Fatalf("expected workflow ID to be empty when using inline definition")
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
