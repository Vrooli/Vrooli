package status

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

type statusOutput struct {
	APIServer struct {
		Running bool   `json:"running"`
		URL     string `json:"url"`
	} `json:"api_server"`
	Browserless struct {
		Status string `json:"status"`
	} `json:"browserless"`
	Workflows struct {
		Count int `json:"count"`
	} `json:"workflows"`
}

type workflowListResponse struct {
	Workflows []struct {
		ID string `json:"id"`
	} `json:"workflows"`
}

func Commands(ctx *appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{
				Name:        "status",
				NeedsAPI:    false,
				Description: "Show operational status and resource health",
				Run: func(args []string) error {
					return runStatus(ctx, args)
				},
			},
		},
	}
}

func runStatus(ctx *appctx.Context, args []string) error {
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	apiBase := strings.TrimRight(ctx.APIRoot(), "/")
	apiRunning := false
	if apiBase != "" {
		healthURL := apiBase + "/health"
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(healthURL)
		if err == nil {
			apiRunning = resp.StatusCode < 400
			_ = resp.Body.Close()
		}
	}

	browserlessStatus := "not_installed"
	if _, err := exec.LookPath("resource-browserless"); err == nil {
		cmd := exec.Command("resource-browserless", "status")
		output, err := cmd.Output()
		if err != nil {
			browserlessStatus = "stopped"
		} else if strings.Contains(strings.ToLower(string(output)), "running") {
			browserlessStatus = "running"
		} else {
			browserlessStatus = "stopped"
		}
	}

	workflowCount := 0
	if apiRunning {
		body, err := ctx.Core.APIClient.Get(ctx.APIPath("/workflows"), nil)
		if err == nil {
			var parsed workflowListResponse
			if json.Unmarshal(body, &parsed) == nil {
				workflowCount = len(parsed.Workflows)
			}
		}
	}

	if jsonOutput {
		var output statusOutput
		output.APIServer.Running = apiRunning
		output.APIServer.URL = ctx.APIV1Base()
		output.Browserless.Status = browserlessStatus
		output.Workflows.Count = workflowCount
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("Browser Automation Studio Status")
	fmt.Println("================================")
	if apiRunning {
		fmt.Println("status: running")
	} else {
		fmt.Println("status: down")
	}

	if apiRunning {
		fmt.Println("API Server: OK")
	} else {
		fmt.Println("API Server: ERROR (not responding)")
	}

	switch browserlessStatus {
	case "running":
		fmt.Println("Browserless: OK")
	case "stopped":
		fmt.Println("Browserless: WARN (not running)")
	default:
		fmt.Println("Browserless: ERROR (not installed)")
	}

	fmt.Printf("Workflows: %d\n", workflowCount)
	fmt.Println("")
	fmt.Println("Ready for automation!")
	return nil
}
