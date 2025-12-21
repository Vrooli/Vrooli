package executions

import (
	"encoding/json"
	"net/url"

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
	_ = json.Unmarshal(body, &detail)
	return detail, body, nil
}
