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

func runRenderVideo(ctx *appctx.Context, args []string) error {
	executionID := ""
	outputFile := ""
	format := "mp4"
	overwrite := false

	for i := 0; i < len(args); i++ {
		token := args[i]
		switch {
		case token == "--output" || token == "--out" || token == "--file":
			if i+1 >= len(args) {
				return fmt.Errorf("--output flag requires a file path")
			}
			outputFile = args[i+1]
			i++
		case strings.HasPrefix(token, "--output="):
			outputFile = strings.TrimPrefix(token, "--output=")
		case strings.HasPrefix(token, "--out="):
			outputFile = strings.TrimPrefix(token, "--out=")
		case strings.HasPrefix(token, "--file="):
			outputFile = strings.TrimPrefix(token, "--file=")
		case token == "--format" || token == "--video-format":
			if i+1 >= len(args) {
				return fmt.Errorf("--format flag requires mp4 or webm")
			}
			format = args[i+1]
			i++
		case strings.HasPrefix(token, "--format="):
			format = strings.TrimPrefix(token, "--format=")
		case strings.HasPrefix(token, "--video-format="):
			format = strings.TrimPrefix(token, "--video-format=")
		case token == "--overwrite" || token == "-f":
			overwrite = true
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

	format = strings.ToLower(strings.TrimSpace(format))
	if executionID == "" {
		return fmt.Errorf("execution ID is required")
	}
	if format != "mp4" && format != "webm" {
		return fmt.Errorf("unsupported format: %s - expected mp4 or webm", format)
	}

	checkPayload, _ := json.Marshal(map[string]any{"format": "json"})
	status, body, err := api.Do(ctx, "POST", ctx.APIPath("/executions/"+executionID+"/export"), nil, checkPayload, nil)
	if err != nil {
		return err
	}
	if status == 404 {
		return fmt.Errorf("execution not found")
	}
	if status != 200 {
		return fmt.Errorf("failed to generate export: %s", extractMessage(body))
	}

	if outputFile == "" {
		outputFile = fmt.Sprintf("bas-replay-%s.%s", executionID, format)
	}
	if info, err := os.Stat(outputFile); err == nil && !info.IsDir() && !overwrite {
		return fmt.Errorf("%s already exists (use --overwrite)", outputFile)
	}

	acceptHeader := "video/mp4"
	if format == "webm" {
		acceptHeader = "video/webm"
	}

	renderPayload, _ := json.Marshal(map[string]any{
		"format":    format,
		"file_name": filepath.Base(outputFile),
	})

	headers := map[string]string{
		"Accept":       acceptHeader,
		"Content-Type": "application/json",
	}

	status, body, err = api.Do(ctx, "POST", ctx.APIPath("/executions/"+executionID+"/export"), nil, renderPayload, headers)
	if err != nil {
		return err
	}
	if status == 404 {
		return fmt.Errorf("execution not found")
	}
	if status != 200 {
		return fmt.Errorf("failed to render replay: %s", extractMessage(body))
	}

	tmpFile := outputFile + ".download"
	if err := os.WriteFile(tmpFile, body, 0o644); err != nil {
		return fmt.Errorf("write replay video: %w", err)
	}
	if err := os.Rename(tmpFile, outputFile); err != nil {
		return fmt.Errorf("save replay video: %w", err)
	}

	fmt.Printf("OK: Replay video ready: %s\n", outputFile)
	return nil
}
