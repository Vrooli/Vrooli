package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

func (s *Session) ExecuteSetStorage(ctx context.Context, params runtime.InstructionParam) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"storageType": params.StorageType,
		"key":         params.StorageKey,
		"valueType":   params.StorageValueType,
	}}

	timeout := params.TimeoutMs
	if timeout <= 0 {
		timeout = 15000
	}

	timeoutCtx, cancel := context.WithTimeout(s.ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	script, err := buildSetStorageScript(params)
	if err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	payload := map[string]any{}
	if err := s.evalWithFrame(timeoutCtx, script, &payload); err != nil {
		result.Error = fmt.Sprintf("setStorage failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.ExtractedData = payload
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

func (s *Session) ExecuteGetStorage(ctx context.Context, params runtime.InstructionParam) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"storageType": params.StorageType,
		"key":         params.StorageKey,
		"format":      params.StorageResultFormat,
	}}

	timeout := params.TimeoutMs
	if timeout <= 0 {
		timeout = 15000
	}

	timeoutCtx, cancel := context.WithTimeout(s.ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	script, err := buildGetStorageScript(params)
	if err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	payload := map[string]any{}
	if err := s.evalWithFrame(timeoutCtx, script, &payload); err != nil {
		result.Error = fmt.Sprintf("getStorage failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	var extracted any
	rawValue, _ := payload["value"].(string)
	if params.StorageResultFormat == "json" && rawValue != "" {
		var parsed any
		if err := json.Unmarshal([]byte(rawValue), &parsed); err != nil {
			result.Error = fmt.Sprintf("failed to parse JSON value: %v", err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
		extracted = parsed
	} else {
		extracted = rawValue
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.ExtractedData = extracted
	result.DebugContext["exists"] = payload["exists"]
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

func (s *Session) ExecuteClearStorage(ctx context.Context, params runtime.InstructionParam) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"storageType": params.StorageType,
		"key":         params.StorageKey,
		"clearAll":    params.StorageClearAll,
	}}

	timeout := params.TimeoutMs
	if timeout <= 0 {
		timeout = 15000
	}

	timeoutCtx, cancel := context.WithTimeout(s.ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	script := buildClearStorageScript(params)
	payload := map[string]any{}
	if err := s.evalWithFrame(timeoutCtx, script, &payload); err != nil {
		result.Error = fmt.Sprintf("clearStorage failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DebugContext["cleared"] = payload
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

func buildSetStorageScript(params runtime.InstructionParam) (string, error) {
	keyLiteral, err := json.Marshal(params.StorageKey)
	if err != nil {
		return "", fmt.Errorf("failed to encode storage key: %w", err)
	}

	storageAccessor := fmt.Sprintf("window[%q]", params.StorageType)
	switch params.StorageValueType {
	case "json":
		trimmed := strings.TrimSpace(params.StorageValue)
		if trimmed == "" {
			return "", fmt.Errorf("json value required for storage key %s", params.StorageKey)
		}
		if !json.Valid([]byte(trimmed)) {
			return "", fmt.Errorf("invalid json payload for storage key %s", params.StorageKey)
		}
		return fmt.Sprintf(`(() => {
            const storage = %s;
            if (!storage) { throw new Error(%s); }
            const serialized = JSON.stringify(%s);
            storage.setItem(%s, serialized);
            return { key: %s, value: storage.getItem(%s) };
        })()`,
			storageAccessor,
			jsStringLiteral(fmt.Sprintf("%s unavailable", params.StorageType)),
			trimmed,
			keyLiteral,
			keyLiteral,
			keyLiteral,
		), nil
	default:
		valueLiteral, err := json.Marshal(params.StorageValue)
		if err != nil {
			return "", fmt.Errorf("failed to encode storage value: %w", err)
		}
		return fmt.Sprintf(`(() => {
            const storage = %s;
            if (!storage) { throw new Error(%s); }
            storage.setItem(%s, %s);
            return { key: %s, value: storage.getItem(%s) };
        })()`,
			storageAccessor,
			jsStringLiteral(fmt.Sprintf("%s unavailable", params.StorageType)),
			keyLiteral,
			valueLiteral,
			keyLiteral,
			keyLiteral,
		), nil
	}
}

func buildGetStorageScript(params runtime.InstructionParam) (string, error) {
	keyLiteral, err := json.Marshal(params.StorageKey)
	if err != nil {
		return "", fmt.Errorf("failed to encode storage key: %w", err)
	}
	return fmt.Sprintf(`(() => {
        const storage = window[%q];
        if (!storage) { throw new Error(%s); }
        const value = storage.getItem(%s);
        return { key: %s, value, exists: value !== null };
    })()`,
		params.StorageType,
		jsStringLiteral(fmt.Sprintf("%s unavailable", params.StorageType)),
		keyLiteral,
		keyLiteral,
	), nil
}

func buildClearStorageScript(params runtime.InstructionParam) string {
	keyLiteral, _ := json.Marshal(params.StorageKey)
	if params.StorageClearAll {
		return fmt.Sprintf(`(() => {
            const storage = window[%q];
            if (!storage) { throw new Error(%s); }
            storage.clear();
            return { clearedAll: true };
        })()`,
			params.StorageType,
			jsStringLiteral(fmt.Sprintf("%s unavailable", params.StorageType)),
		)
	}
	return fmt.Sprintf(`(() => {
        const storage = window[%q];
        if (!storage) { throw new Error(%s); }
        storage.removeItem(%s);
        return { clearedAll: false, key: %s };
    })()`,
		params.StorageType,
		jsStringLiteral(fmt.Sprintf("%s unavailable", params.StorageType)),
		keyLiteral,
		keyLiteral,
	)
}

func jsStringLiteral(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
