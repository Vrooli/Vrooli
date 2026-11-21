package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"browser-automation-studio/cli/internal/api"
	"browser-automation-studio/cli/internal/appctx"
)

// ExportExecution exports the execution results to a folder.
func ExportExecution(ctx *appctx.Context, executionID, outputDir string) error {
	payload := map[string]any{
		"format":     "folder",
		"output_dir": outputDir,
	}
	bodyPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	status, body, err := api.Do(ctx, "POST",
		"/executions/"+executionID+"/export",
		nil, bodyPayload, nil)
	if err != nil {
		return err
	}

	if status == 404 {
		return fmt.Errorf("execution not found")
	}
	if status != 200 {
		return fmt.Errorf("export failed (status %d): %s", status, extractMessage(body))
	}

	return nil
}

// extractMessage attempts to extract a message from a JSON response body.
func extractMessage(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return strings.TrimSpace(string(body))
	}
	if value, ok := payload["message"].(string); ok {
		return strings.TrimSpace(value)
	}
	if value, ok := payload["error"].(string); ok {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(string(body))
}
