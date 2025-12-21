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
	if len(args) == 0 || strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("workflow ID or name is required")
	}

	workflow := args[0]
	args = args[1:]

	paramsRaw := "{}"
	wait := false
	outputDir := ""
	projectRoot := ""
	adhoc := false
	requiresVideo := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--params":
			if i+1 >= len(args) {
				return fmt.Errorf("--params requires a value")
			}
			paramsRaw = args[i+1]
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
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	fmt.Printf("Executing workflow: %s\n", workflow)

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

	fmt.Printf("API URL: %s\n", ctx.APIV1Base())
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
		if err != nil {
			return err
		}
		workflowDetail, err := getWorkflow(ctx, workflowID)
		if err != nil {
			return err
		}
		if len(workflowDetail.Workflow.FlowDefinition) == 0 || string(workflowDetail.Workflow.FlowDefinition) == "null" {
			return fmt.Errorf("missing flow_definition for workflow %s", workflowID)
		}

		payload := map[string]any{
			"flow_definition":     json.RawMessage(workflowDetail.Workflow.FlowDefinition),
			"wait_for_completion": wait,
			"parameters":          params,
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
	fmt.Println("")
	fmt.Println("Execution artifacts")
	fmt.Printf("Timeline: %s/executions/%s/timeline\n", ctx.APIV1Base(), executionID)
	fmt.Printf("Screenshots: %s/executions/%s/screenshots\n", ctx.APIV1Base(), executionID)
	if ctx.ScenarioRoot != "" {
		fmt.Printf("Recordings root: %s\n", filepath.Join(ctx.ScenarioRoot, "data", "recordings", executionID))
	}
	fmt.Printf("Folder export: browser-automation-studio execution export %s --format folder --output-dir ./bas-export\n", executionID)

	if wait {
		fmt.Println("Waiting for completion...")
		maxAttempts := 60
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			statusResp, err := ctx.Core.APIClient.Get(ctx.APIPath("/executions/"+executionID), nil)
			if err != nil {
				fmt.Println(".")
				time.Sleep(5 * time.Second)
				continue
			}
			status := extractString(statusResp, "status")
			if status == "completed" {
				fmt.Println("OK: Execution completed successfully")
				break
			}
			if status == "failed" {
				fmt.Println("ERROR: Execution failed")
				errorMessage := extractString(statusResp, "error")
				if errorMessage != "" {
					fmt.Printf("Error: %s\n", errorMessage)
				}
				break
			}
			fmt.Print(".")
			time.Sleep(5 * time.Second)
		}
	}

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err == nil {
			fmt.Printf("Screenshots saved to: %s\n", outputDir)
		}
	}

	return nil
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
