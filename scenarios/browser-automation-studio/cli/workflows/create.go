package workflows

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

func runCreate(ctx *appctx.Context, args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("workflow name is required")
	}
	name := args[0]
	args = args[1:]

	folder := "/"
	fromFile := ""
	aiPrompt := ""
	projectID := ""
	projectName := "Demo Browser Automations"
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--folder":
			if i+1 >= len(args) {
				return fmt.Errorf("--folder requires a value")
			}
			folder = args[i+1]
			i++
		case "--from-file":
			if i+1 >= len(args) {
				return fmt.Errorf("--from-file requires a value")
			}
			fromFile = args[i+1]
			i++
		case "--ai-prompt":
			if i+1 >= len(args) {
				return fmt.Errorf("--ai-prompt requires a value")
			}
			aiPrompt = args[i+1]
			i++
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
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}

	if projectID == "" {
		projectID = resolveProjectID(ctx, projectName)
	}

	if projectID == "" && projectName == "Demo Browser Automations" {
		projectID = createDemoProject(ctx)
	}

	if projectID == "" {
		return fmt.Errorf("unable to resolve project_id")
	}

	payload := map[string]any{
		"project_id":  projectID,
		"name":        name,
		"folder_path": folder,
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
	} else if aiPrompt != "" {
		payload["ai_prompt"] = aiPrompt
	}

	body, err := ctx.Core.APIClient.Request("POST", ctx.APIPath("/workflows/create"), nil, payload)
	if err != nil {
		return err
	}

	if jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Creating workflow: %s\n", name)
	var parsed createResponse
	if json.Unmarshal(body, &parsed) == nil {
		workflowID := parsed.ID
		if workflowID == "" {
			workflowID = parsed.Workflow.ID
		}
		fmt.Println("OK: Workflow created successfully!")
		if workflowID != "" {
			fmt.Printf("Workflow ID: %s\n", workflowID)
		}
		fmt.Printf("Project: %s\n", projectName)
		fmt.Printf("Folder: %s\n", folder)
		return nil
	}

	fmt.Println("OK: Workflow created successfully!")
	fmt.Printf("Project: %s\n", projectName)
	fmt.Printf("Folder: %s\n", folder)
	fmt.Println(string(body))
	return nil
}

func resolveProjectID(ctx *appctx.Context, name string) string {
	body, err := ctx.Core.APIClient.Get(ctx.APIPath("/projects"), nil)
	if err != nil {
		return ""
	}
	var parsed projectListResponse
	if json.Unmarshal(body, &parsed) != nil {
		return ""
	}
	for _, project := range parsed.Projects {
		if project.Project.Name == name {
			return project.Project.ID
		}
	}
	return ""
}

func createDemoProject(ctx *appctx.Context) string {
	demoFolder := ""
	if ctx.ScenarioRoot != "" {
		demoFolder = filepath.Join(ctx.ScenarioRoot, "data", "projects", "demo")
	}
	payload := map[string]any{
		"name":        "Demo Browser Automations",
		"folder_path": demoFolder,
		"preset":      "recommended",
	}
	body, err := ctx.Core.APIClient.Request("POST", ctx.APIPath("/projects"), nil, payload)
	if err != nil {
		return ""
	}
	var parsed createResponse
	if json.Unmarshal(body, &parsed) == nil {
		if parsed.ID != "" {
			return parsed.ID
		}
		if parsed.Workflow.ID != "" {
			return parsed.Workflow.ID
		}
	}

	var raw map[string]any
	if json.Unmarshal(body, &raw) == nil {
		if id, ok := raw["id"].(string); ok {
			return id
		}
		if project, ok := raw["project"].(map[string]any); ok {
			if id, ok := project["id"].(string); ok {
				return id
			}
		}
	}
	return ""
}
