package executions

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"browser-automation-studio/cli/internal/api"
	"browser-automation-studio/cli/internal/appctx"
)

func runRender(ctx *appctx.Context, args []string) error {
	executionID := ""
	outputDir := ""
	overwrite := false

	for i := 0; i < len(args); i++ {
		token := args[i]
		switch {
		case token == "--output" || token == "--out" || token == "--dir":
			if i+1 >= len(args) {
				return fmt.Errorf("--output flag requires a directory path")
			}
			outputDir = args[i+1]
			i++
		case strings.HasPrefix(token, "--output="):
			outputDir = strings.TrimPrefix(token, "--output=")
		case strings.HasPrefix(token, "--out="):
			outputDir = strings.TrimPrefix(token, "--out=")
		case strings.HasPrefix(token, "--dir="):
			outputDir = strings.TrimPrefix(token, "--dir=")
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

	if executionID == "" {
		return fmt.Errorf("execution ID is required")
	}

	if outputDir == "" {
		outputDir = fmt.Sprintf("bas-replay-%s", executionID)
	}

	if info, err := os.Stat(outputDir); err == nil && info.IsDir() && !overwrite {
		return fmt.Errorf("%s already exists (use --overwrite)", outputDir)
	}

	archiveName := filepath.Base(outputDir) + ".zip"
	payload := map[string]any{
		"format":    "html",
		"file_name": archiveName,
	}
	bodyPayload, _ := json.Marshal(payload)

	headers := map[string]string{"Accept": "application/zip", "Content-Type": "application/json"}
	status, body, err := api.Do(ctx, "POST", ctx.APIPath("/executions/"+executionID+"/export"), nil, bodyPayload, headers)
	if err != nil {
		return err
	}
	if status == 404 {
		return fmt.Errorf("execution not found")
	}
	if status != 200 {
		return fmt.Errorf("failed to generate export: %s", extractMessage(body))
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	if err := extractZip(body, outputDir); err != nil {
		return fmt.Errorf("failed to extract replay package: %w", err)
	}

	fmt.Printf("OK: Replay package ready: open %s/index.html\n", outputDir)
	return nil
}

func extractZip(data []byte, destination string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(destination, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}

		dst, err := os.Create(path)
		if err != nil {
			src.Close()
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			src.Close()
			return err
		}
		if err := dst.Close(); err != nil {
			src.Close()
			return err
		}
		src.Close()
	}

	return nil
}
