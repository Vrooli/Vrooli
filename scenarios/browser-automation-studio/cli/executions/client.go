package executions

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"browser-automation-studio/cli/internal/appctx"
)

type listResponse struct {
	Executions []executionSummary `json:"executions"`
}

type executionSummary struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
}

type executionDetail struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	CurrentStep string `json:"current_step"`
	Error       string `json:"error"`
}

type exportSummary struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	Package struct {
		Summary struct {
			FrameCount      int `json:"frame_count"`
			TotalDurationMs int `json:"total_duration_ms"`
		} `json:"summary"`
		Theme struct {
			AccentColor string `json:"accent_color"`
		} `json:"theme"`
	} `json:"package"`
}

func listExecutions(ctx *appctx.Context, values url.Values) ([]executionSummary, []byte, error) {
	body, err := ctx.Core.APIClient.Get(ctx.APIPath("/executions"), values)
	if err != nil {
		return nil, nil, err
	}
	items, err := parseExecutionSummaries(body)
	if err == nil {
		return items, body, nil
	}
	var parsed listResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, body, err
	}
	return parsed.Executions, body, nil
}

func getExecution(ctx *appctx.Context, executionID string) (executionDetail, []byte, error) {
	var detail executionDetail
	body, err := ctx.Core.APIClient.Get(ctx.APIPath("/executions/"+executionID), nil)
	if err != nil {
		return detail, nil, err
	}
	if err := populateExecutionDetail(&detail, body); err != nil {
		_ = json.Unmarshal(body, &detail)
	}
	return detail, body, nil
}

func parseExecutionSummaries(body []byte) ([]executionSummary, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	rawExecutions, ok := payload["executions"].([]any)
	if !ok {
		return nil, fmt.Errorf("execution list not found")
	}
	summaries := make([]executionSummary, 0, len(rawExecutions))
	for _, raw := range rawExecutions {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		summaries = append(summaries, executionSummary{
			ID:        extractString(item, "id", "execution_id"),
			Status:    normalizeExecutionStatus(extractString(item, "status")),
			StartedAt: extractString(item, "started_at", "startedAt"),
		})
	}
	return summaries, nil
}

func populateExecutionDetail(detail *executionDetail, body []byte) error {
	if detail == nil {
		return fmt.Errorf("execution detail target missing")
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	detail.ID = extractString(payload, "id", "execution_id")
	detail.Status = normalizeExecutionStatus(extractString(payload, "status"))
	detail.Progress = extractInt(payload, "progress")
	detail.CurrentStep = extractString(payload, "current_step", "currentStep")
	detail.Error = extractString(payload, "error", "error_message", "errorMessage", "message")
	return nil
}

func extractString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}
	return ""
}

func extractInt(payload map[string]any, key string) int {
	if payload == nil {
		return 0
	}
	if value, ok := payload[key]; ok {
		switch typed := value.(type) {
		case float64:
			return int(typed)
		case int:
			return typed
		}
	}
	return 0
}

func normalizeExecutionStatus(raw string) string {
	switch strings.TrimSpace(strings.ToUpper(raw)) {
	case "EXECUTION_STATUS_COMPLETED", "COMPLETED":
		return "completed"
	case "EXECUTION_STATUS_FAILED", "FAILED":
		return "failed"
	case "EXECUTION_STATUS_CANCELLED", "CANCELLED", "CANCELED":
		return "cancelled"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}
