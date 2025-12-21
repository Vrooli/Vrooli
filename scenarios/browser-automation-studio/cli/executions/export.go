package executions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"browser-automation-studio/cli/internal/api"
	"browser-automation-studio/cli/internal/appctx"
)

func runExport(ctx *appctx.Context, args []string) error {
	executionID := ""
	outputPath := ""
	outputDir := ""
	exportFormat := "json"

	for i := 0; i < len(args); i++ {
		token := args[i]
		switch {
		case token == "--format":
			if i+1 >= len(args) {
				return fmt.Errorf("--format flag requires a value")
			}
			exportFormat = args[i+1]
			i++
		case strings.HasPrefix(token, "--format="):
			exportFormat = strings.TrimPrefix(token, "--format=")
		case token == "--output" || token == "--out":
			if i+1 >= len(args) {
				return fmt.Errorf("--output flag requires a file path")
			}
			outputPath = args[i+1]
			i++
		case strings.HasPrefix(token, "--output="):
			outputPath = strings.TrimPrefix(token, "--output=")
		case strings.HasPrefix(token, "--out="):
			outputPath = strings.TrimPrefix(token, "--out=")
		case token == "--output-dir" || token == "--out-dir":
			if i+1 >= len(args) {
				return fmt.Errorf("--output-dir flag requires a directory path")
			}
			outputDir = args[i+1]
			i++
		case strings.HasPrefix(token, "--output-dir="):
			outputDir = strings.TrimPrefix(token, "--output-dir=")
		case strings.HasPrefix(token, "--out-dir="):
			outputDir = strings.TrimPrefix(token, "--out-dir=")
		case strings.HasPrefix(token, "--"):
			return fmt.Errorf("unknown option: %s", token)
		default:
			if executionID == "" {
				executionID = token
			} else {
				return fmt.Errorf("unexpected argument: %s", token)
			}
		}
	}

	if executionID == "" {
		return fmt.Errorf("execution ID is required")
	}

	exportFormat = strings.ToLower(strings.TrimSpace(exportFormat))
	if exportFormat != "json" && exportFormat != "folder" {
		return fmt.Errorf("--format must be 'json' or 'folder'")
	}

	if exportFormat == "folder" && outputDir == "" {
		return fmt.Errorf("--output-dir is required for format=folder")
	}

	var bodyPayload []byte
	if exportFormat == "folder" {
		payload := map[string]any{
			"format":     exportFormat,
			"output_dir": outputDir,
		}
		bodyPayload, _ = json.Marshal(payload)
	}

	status, body, err := api.Do(ctx, "POST", ctx.APIPath("/executions/"+executionID+"/export"), nil, bodyPayload, nil)
	if err != nil {
		return err
	}

	if status == 404 {
		return fmt.Errorf("execution not found")
	}
	if status != 200 {
		return fmt.Errorf("received response status %d: %s", status, extractMessage(body))
	}

	if exportFormat == "folder" {
		fmt.Printf("OK: Export ready: %s\n", fallback(extractMessage(body), "Execution export generated"))
		fmt.Printf("Output directory: %s\n", outputDir)
		return nil
	}

	var summary exportSummary
	_ = json.Unmarshal(body, &summary)
	message := strings.TrimSpace(summary.Message)
	if message == "" {
		message = "Replay export generated"
	}
	fmt.Printf("OK: Export ready: %s\n", message)
	if summary.Status != "" {
		fmt.Printf("Status: %s\n", summary.Status)
	}
	if summary.Package.Summary.FrameCount > 0 || summary.Package.Summary.TotalDurationMs > 0 {
		frameLabel := summary.Package.Summary.FrameCount
		durationLabel := summary.Package.Summary.TotalDurationMs
		fmt.Printf("Frames: %d, Total Duration: %dms\n", frameLabel, durationLabel)
	}
	if summary.Package.Theme.AccentColor != "" {
		fmt.Printf("Theme Accent: %s\n", summary.Package.Theme.AccentColor)
	}

	if outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err == nil {
			if err := os.WriteFile(outputPath, body, 0o644); err == nil {
				fmt.Printf("Saved export package to %s\n", outputPath)
				return nil
			}
		}
		fmt.Printf("Failed to write export package to %s\n", outputPath)
		return nil
	}

	fmt.Println("Tip: use --output <file> to save the export package.")
	return nil
}

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

func fallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
