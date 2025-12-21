package workflows

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"browser-automation-studio/cli/internal/appctx"
)

func runExecute(ctx *appctx.Context, args []string) error {
	workflow := ""

	paramsRaw := "{}"
	wait := false
	outputDir := ""
	projectRoot := ""
	adhoc := false
	requiresVideo := false
	fromFile := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--params":
			if i+1 >= len(args) {
				return fmt.Errorf("--params requires a value")
			}
			paramsRaw = args[i+1]
			i++
		case "--from-file":
			if i+1 >= len(args) {
				return fmt.Errorf("--from-file requires a value")
			}
			fromFile = args[i+1]
			i++
		case "--wait":
			wait = true
		case "--output-screenshots":
			if i+1 >= len(args) {
				return fmt.Errorf("--output-screenshots requires a value")
			}
			outputDir = args[i+1]
			i++
		case "--project-root":
			if i+1 >= len(args) {
				return fmt.Errorf("--project-root requires a value")
			}
			projectRoot = args[i+1]
			i++
		case "--adhoc":
			adhoc = true
		case "--record-video", "--requires-video":
			requiresVideo = true
		default:
			if strings.HasPrefix(args[i], "--") {
				return fmt.Errorf("unknown option: %s", args[i])
			}
			if workflow == "" {
				workflow = args[i]
			} else {
				return fmt.Errorf("unexpected argument: %s", args[i])
			}
		}
	}

	if strings.TrimSpace(fromFile) == "" && strings.TrimSpace(workflow) == "" {
		return fmt.Errorf("workflow ID/name or --from-file is required")
	}

	if fromFile != "" {
		fmt.Printf("Executing workflow file: %s\n", fromFile)
	} else {
		fmt.Printf("Executing workflow: %s\n", workflow)
	}

	if projectRoot == "" && fromFile != "" {
		absFile, err := filepath.Abs(fromFile)
		if err != nil {
			return fmt.Errorf("resolve --from-file path: %w", err)
		}
		projectRoot = findBasRoot(filepath.Dir(absFile))
		if projectRoot == "" {
			return fmt.Errorf("unable to infer --project-root from file path; set --project-root /abs/path/to/bas")
		}
	}

	if projectRoot == "" {
		if envRoot := strings.TrimSpace(os.Getenv("BAS_PROJECT_ROOT")); envRoot != "" {
			projectRoot = envRoot
		} else if ctx.ScenarioRoot != "" {
			candidate := filepath.Join(ctx.ScenarioRoot, "bas")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				projectRoot = candidate
			}
		}
	}

	if projectRoot != "" {
		adhoc = true
	}

	fmt.Printf("API URL: %s\n", ctx.ResolvedAPIV1Base())
	if projectRoot != "" {
		fmt.Printf("Project root: %s\n", projectRoot)
	} else {
		fmt.Println("Project root: (not set)")
		fmt.Println("WARN: Subflows using workflow_path may fail. Provide --project-root /abs/path/to/bas")
	}

	if adhoc {
		fmt.Println("Execution mode: adhoc")
		if wait {
			fmt.Println("Wait mode: client-side polling (adhoc returns immediately)")
		}
	} else {
		fmt.Println("Execution mode: direct")
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(paramsRaw), &params); err != nil {
		return fmt.Errorf("invalid JSON for --params")
	}

	if projectRoot != "" {
		params["project_root"] = projectRoot
		fmt.Println("Project root injected into parameters as project_root.")
	}

	var response []byte
	if adhoc {
		workflowID, err := resolveWorkflowID(ctx, workflow)
		if fromFile == "" {
			if err != nil {
				return err
			}
		}

		payload := map[string]any{
			"wait_for_completion": wait,
			"parameters":          params,
		}

		if fromFile != "" {
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("file not found: %s", fromFile)
			}
			var flowDef any
			if err := json.Unmarshal(data, &flowDef); err != nil {
				return fmt.Errorf("invalid JSON in %s", fromFile)
			}
			payload["flow_definition"] = flowDef
		} else {
			workflowDetail, err := getWorkflow(ctx, workflowID)
			if err != nil {
				return err
			}
			if len(workflowDetail.Workflow.FlowDefinition) == 0 || string(workflowDetail.Workflow.FlowDefinition) == "null" {
				return fmt.Errorf("missing flow_definition for workflow %s", workflowID)
			}
			payload["flow_definition"] = json.RawMessage(workflowDetail.Workflow.FlowDefinition)
		}

		executePath := ctx.APIPath("/workflows/execute-adhoc")
		if requiresVideo {
			executePath += "?requires_video=true"
		}
		response, err = ctx.Core.APIClient.Request("POST", executePath, nil, payload)
		if err != nil {
			return err
		}
	} else {
		executePath := ctx.APIPath("/workflows/" + workflow + "/execute")
		if requiresVideo {
			executePath += "?requires_video=true"
		}
		payload := map[string]any{
			"parameters":          params,
			"wait_for_completion": wait,
		}
		var err error
		response, err = ctx.Core.APIClient.Request("POST", executePath, nil, payload)
		if err != nil {
			return err
		}
	}

	var execResp executeResponse
	if err := json.Unmarshal(response, &execResp); err != nil || execResp.ExecutionID == "" {
		fmt.Println("error: failed to start execution")
		fmt.Println(string(response))
		if adhoc {
			fmt.Println("Note: adhoc executions can still start even if the API times out.")
			fmt.Println("Check running executions with: browser-automation-studio execution list")
		} else if projectRoot == "" {
			fmt.Println("Hint: set --project-root /abs/path/to/bas so workflow_path subflows can resolve.")
		}
		return fmt.Errorf("execution start failed")
	}

	executionID := execResp.ExecutionID
	fmt.Println("OK: Execution started!")
	fmt.Printf("Execution ID: %s\n", executionID)

	recordingsRoot := ""
	if ctx.ScenarioRoot != "" {
		recordingsRoot = filepath.Join(ctx.ScenarioRoot, "data", "recordings", executionID)
	}

	if !wait {
		fmt.Println("")
		fmt.Printf("Artifacts will be available after completion. Watch with: browser-automation-studio execution watch %s\n", executionID)
		if recordingsRoot != "" {
			fmt.Printf("Artifacts stored at: %s\n", recordingsRoot)
		}
	}

	if wait {
		fmt.Println("Waiting for completion...")
		maxAttempts := 60
		lastStatus := ""
		completed := false
		failed := false
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			statusResp, err := ctx.Core.APIClient.Get(ctx.APIPath("/executions/"+executionID), nil)
			if err != nil {
				fmt.Println(".")
				time.Sleep(5 * time.Second)
				continue
			}
			status := normalizeExecutionStatus(extractString(statusResp, "status"))
			lastStatus = status
			if status == "completed" {
				fmt.Println("OK: Execution completed successfully")
				completed = true
				break
			}
			if status == "failed" || status == "cancelled" {
				fmt.Println("ERROR: Execution failed")
				errorMessage := extractString(statusResp, "error")
				if errorMessage == "" {
					errorMessage = extractString(statusResp, "error_message")
				}
				if errorMessage != "" {
					fmt.Printf("Error: %s\n", errorMessage)
				}
				failed = true
				completed = true
				break
			}
			fmt.Print(".")
			time.Sleep(5 * time.Second)
		}
		if completed {
			printCollectedArtifacts(ctx, executionID, recordingsRoot, failed, requiresVideo)
		}
		if !completed {
			if lastStatus == "" {
				lastStatus = "unknown"
			}
			fmt.Println("")
			fmt.Printf("TIMEOUT: Execution did not finish after %d seconds (last status: %s). Use: browser-automation-studio execution watch %s\n", maxAttempts*5, lastStatus, executionID)
		}
	}

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err == nil {
			fmt.Printf("Screenshots saved to: %s\n", outputDir)
		}
	}

	return nil
}

type executionResultSummary struct {
	Artifacts []struct {
		ArtifactType string `json:"artifact_type"`
	} `json:"artifacts"`
}

func printCollectedArtifacts(ctx *appctx.Context, executionID, recordingsRoot string, failed bool, requiresVideo bool) {
	fmt.Println("")
	fmt.Println("Execution artifacts")

	hasTimeline := false
	hasScreenshots := false
	hasVideos := false

	if recordingsRoot != "" {
		timelinePath := filepath.Join(recordingsRoot, "timeline.proto.json")
		if info, err := os.Stat(timelinePath); err == nil && !info.IsDir() {
			hasTimeline = true
		}

		resultPath := filepath.Join(recordingsRoot, "result.json")
		if data, err := os.ReadFile(resultPath); err == nil {
			var result executionResultSummary
			if err := json.Unmarshal(data, &result); err == nil {
				for _, artifact := range result.Artifacts {
					switch strings.ToLower(strings.TrimSpace(artifact.ArtifactType)) {
					case "screenshot", "screenshot_inline":
						hasScreenshots = true
					case "video", "video_meta":
						hasVideos = true
					}
				}
			}
		}
	}

	if hasTimeline {
		fmt.Printf("Timeline: %s/executions/%s/timeline\n", ctx.ResolvedAPIV1Base(), executionID)
	}
	if hasScreenshots {
		fmt.Printf("Screenshots: %s/executions/%s/screenshots\n", ctx.ResolvedAPIV1Base(), executionID)
	}
	if hasVideos {
		fmt.Printf("Recorded videos: %s/executions/%s/recorded-videos\n", ctx.ResolvedAPIV1Base(), executionID)
	}
	if !hasTimeline && !hasScreenshots && !hasVideos {
		fmt.Println("No artifacts detected yet.")
	}

	if recordingsRoot != "" {
		fmt.Printf("Artifacts stored at: %s\n", recordingsRoot)
	}

	if !hasVideos && requiresVideo {
		fmt.Println("Video capture was requested but no recordings were produced.")
		fmt.Println("Check playwright-driver logs for video capture errors.")
	}

	if !hasVideos && !requiresVideo {
		optional := "To collect video recordings, rerun with: --requires-video"
		if failed {
			optional = "To collect video recordings on a retry, rerun with: --requires-video"
		}
		fmt.Println(optional)
	}
}

func resolveWorkflowID(ctx *appctx.Context, workflow string) (string, error) {
	if isUUID(workflow) {
		return workflow, nil
	}
	workflows, _, err := listWorkflows(ctx)
	if err != nil {
		return "", err
	}
	matches := []string{}
	for _, item := range workflows {
		entry := item
		if entry.ID == "" && entry.Workflow != nil {
			entry = *entry.Workflow
		}
		if entry.Name == workflow {
			matches = append(matches, entry.ID)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("workflow not found by name '%s'", workflow)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple workflows match name '%s'", workflow)
	}
	return matches[0], nil
}

func findBasRoot(startDir string) string {
	dir := strings.TrimSpace(startDir)
	for dir != "" && dir != "." && dir != string(filepath.Separator) {
		if filepath.Base(dir) == "bas" {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func isUUID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 36 {
		return false
	}
	return strings.Count(value, "-") == 4
}

func extractString(data []byte, key string) string {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
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
