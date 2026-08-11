package programs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// DelegationCharge extracts only an explicitly per-run monetary charge. The
// agent-manager response currently exposes workflow evidence but not a charge
// field, so absence is returned as measured=false and is never converted into
// a misleading zero-cost observation.
func DelegationCharge(result map[string]any) (costMicros int64, measured bool, note string) {
	if value, ok := findNumber(result, "total_charge_micro_usd", "totalChargeMicroUsd", "charge_micro_usd", "chargeMicroUsd"); ok {
		return value, true, "agent-manager returned an explicit per-run charge"
	}
	if value, ok := findNumber(result, "cost_micros", "costMicros"); ok {
		return value, true, "agent-manager returned an explicit per-run cost"
	}
	return 0, false, "agent-manager delegation result contained no per-run charge field"
}

func findNumber(value any, keys ...string) (int64, bool) {
	if object, ok := value.(map[string]any); ok {
		for _, key := range keys {
			if candidate, exists := object[key]; exists {
				switch number := candidate.(type) {
				case float64:
					return int64(number), true
				case int64:
					return number, true
				case int:
					return int64(number), true
				}
			}
		}
		for _, child := range object {
			if number, ok := findNumber(child, keys...); ok {
				return number, true
			}
		}
	}
	if list, ok := value.([]any); ok {
		for _, child := range list {
			if number, ok := findNumber(child, keys...); ok {
				return number, true
			}
		}
	}
	return 0, false
}

// HTTPDelegator is the only program-runtime client for delegated agent work.
// It uses agent-manager's typed Connect JSON routes, waits for terminal state,
// and returns the explicit result projection as bounded handle data.
type HTTPDelegator struct {
	baseURL string
	client  *http.Client
}

func NewHTTPDelegator(baseURL string) *HTTPDelegator {
	return &HTTPDelegator{baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), client: &http.Client{Timeout: 45 * time.Second}}
}

func NewDiscoveryDelegator(client *http.Client) *HTTPDelegator {
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	return &HTTPDelegator{client: client}
}

func (d *HTTPDelegator) Delegate(ctx context.Context, request DelegationRequest) (map[string]any, error) {
	if strings.TrimSpace(request.SessionID) == "" {
		return nil, fmt.Errorf("delegation session_id is required")
	}
	if strings.TrimSpace(request.Owner) == "" || strings.TrimSpace(request.WorkflowKey) == "" {
		return nil, fmt.Errorf("delegation owner and workflow_key are required")
	}
	base := d.baseURL
	if base == "" {
		var err error
		base, err = discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
		if err != nil {
			return nil, fmt.Errorf("resolve agent-manager: %w", err)
		}
		base = strings.TrimRight(base, "/")
	}

	payload := map[string]any{
		"owner":       request.Owner,
		"workflowKey": request.WorkflowKey,
		"input":       request.Input,
	}
	if payload["input"] == nil {
		payload["input"] = map[string]any{}
	}
	if strings.TrimSpace(request.DefinitionDigest) != "" {
		payload["definitionDigest"] = request.DefinitionDigest
	}
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("program-runtime-%s-%d", request.SessionID, time.Now().UnixNano())
	}
	payload["idempotencyKey"] = idempotencyKey
	start, err := d.post(ctx, base+"/api/v1/workflow-executions", payload)
	if err != nil {
		return nil, fmt.Errorf("start delegated run: %w", err)
	}
	execution, ok := start["execution"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("start delegated run: response did not contain execution")
	}
	executionID, _ := execution["id"].(string)
	if strings.TrimSpace(executionID) == "" {
		return nil, fmt.Errorf("start delegated run: response did not contain execution id")
	}

	wait, err := d.post(ctx, base+"/api/v1/workflow-executions/"+executionID+"/wait", map[string]any{
		"executionId":    executionID,
		"timeoutSeconds": 30,
	})
	if err != nil {
		return nil, fmt.Errorf("wait delegated run %s: %w", executionID, err)
	}
	waitExecution, _ := wait["execution"].(map[string]any)
	if waitExecution == nil {
		waitExecution = execution
	}

	result, err := d.get(ctx, base+"/api/v1/workflow-executions/"+executionID+"/result?explicitly_authorized=true")
	if err != nil {
		return nil, fmt.Errorf("collect delegated run %s evidence: %w", executionID, err)
	}
	resultExecution, _ := result["execution"].(map[string]any)
	if resultExecution == nil {
		resultExecution = waitExecution
	}
	return map[string]any{
		"execution_id": executionID,
		"status":       valueOr(resultExecution, "status", valueOr(waitExecution, "status", "")),
		"evidence":     resultExecution["output"],
		"observations": resultExecution["observations"],
	}, nil
}

func (d *HTTPDelegator) post(ctx context.Context, url string, payload map[string]any) (map[string]any, error) {
	return d.do(ctx, http.MethodPost, url, payload)
}

func (d *HTTPDelegator) get(ctx context.Context, url string) (map[string]any, error) {
	return d.do(ctx, http.MethodGet, url, nil)
}

func (d *HTTPDelegator) do(ctx context.Context, method, url string, payload map[string]any) (map[string]any, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("remote status %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

func valueOr(values map[string]any, key string, fallback any) any {
	if value, ok := values[key]; ok && value != nil {
		return value
	}
	return fallback
}
