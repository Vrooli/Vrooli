package recordings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"browser-automation-studio/cli/internal/appctx"
)

type importResponse struct {
	ExecutionID  string `json:"execution_id"`
	WorkflowName string `json:"workflow_name"`
	FrameCount   int    `json:"frame_count"`
	AssetCount   int    `json:"asset_count"`
}

func runImport(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("recording archive path is required")
	}

	filePath := ""
	projectID := ""
	projectName := ""
	workflowID := ""
	workflowName := ""
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project-id":
			if i+1 >= len(args) {
				return fmt.Errorf("--project-id requires a value")
			}
			projectID = args[i+1]
			i++
		case "--project-name":
			if i+1 >= len(args) {
				return fmt.Errorf("--project-name requires a value")
			}
			projectName = args[i+1]
			i++
		case "--workflow-id":
			if i+1 >= len(args) {
				return fmt.Errorf("--workflow-id requires a value")
			}
			workflowID = args[i+1]
			i++
		case "--workflow-name":
			if i+1 >= len(args) {
				return fmt.Errorf("--workflow-name requires a value")
			}
			workflowName = args[i+1]
			i++
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return fmt.Errorf("unknown option: %s", args[i])
			}
			if filePath == "" {
				filePath = args[i]
			} else {
				return fmt.Errorf("unexpected argument: %s", args[i])
			}
		}
	}

	if filePath == "" {
		return fmt.Errorf("recording archive path is required")
	}
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("recording archive not found at %s", filePath)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("prepare upload: %w", err)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	if _, err := fileWriter.Write(readAll(file)); err != nil {
		file.Close()
		return fmt.Errorf("read archive: %w", err)
	}
	file.Close()

	if projectID != "" {
		_ = writer.WriteField("project_id", projectID)
	}
	if projectName != "" {
		_ = writer.WriteField("project_name", projectName)
	}
	if workflowID != "" {
		_ = writer.WriteField("workflow_id", workflowID)
	}
	if workflowName != "" {
		_ = writer.WriteField("workflow_name", workflowName)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize upload: %w", err)
	}

	endpoint := strings.TrimRight(ctx.ResolvedAPIRoot(), "/") + ctx.APIPath("/recordings/import")

	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	if token := strings.TrimSpace(ctx.Token()); token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	timeout := ctx.Core.HTTPClient.Timeout()
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach API for recording import")
	}
	defer resp.Body.Close()

	respBody := readAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("import failed: %s", extractMessage(respBody))
	}

	if jsonOutput {
		fmt.Println(string(respBody))
		return nil
	}

	var parsed importResponse
	if json.Unmarshal(respBody, &parsed) == nil {
		fmt.Println("OK: Recording imported")
		if parsed.WorkflowName != "" {
			fmt.Printf("Workflow: %s\n", parsed.WorkflowName)
		}
		if parsed.ExecutionID != "" {
			fmt.Printf("Execution ID: %s\n", parsed.ExecutionID)
		}
		if parsed.FrameCount > 0 || parsed.AssetCount > 0 {
			fmt.Printf("Frames: %d, Assets: %d\n", parsed.FrameCount, parsed.AssetCount)
		}
		return nil
	}

	fmt.Println("OK: Recording imported")
	fmt.Println(string(respBody))
	return nil
}

func readAll(reader io.Reader) []byte {
	data, _ := io.ReadAll(reader)
	return data
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
