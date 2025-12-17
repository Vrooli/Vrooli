package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "workspace-sandbox"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Workspace Sandbox CLI - Safe copy-on-write workspaces for agents and tools",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}
	app := &App{core: core}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health and driver status", Run: a.cmdStatus},
		},
	}

	sandboxes := cliapp.CommandGroup{
		Title: "Sandbox Operations",
		Commands: []cliapp.Command{
			{Name: "create", NeedsAPI: true, Description: "Create a new sandbox", Run: a.cmdCreate, Flags: []cliapp.Flag{
				{Name: "scope", Alias: "s", Default: "", Description: "Scope path within project"},
				{Name: "project", Alias: "p", Default: "", Description: "Project root directory"},
				{Name: "owner", Alias: "o", Default: "", Description: "Owner identifier"},
			}},
			{Name: "list", NeedsAPI: true, Description: "List sandboxes", Run: a.cmdList, Flags: []cliapp.Flag{
				{Name: "status", Default: "", Description: "Filter by status (active,stopped,approved,rejected)"},
				{Name: "owner", Default: "", Description: "Filter by owner"},
				{Name: "json", Default: "false", Description: "Output as JSON"},
			}},
			{Name: "inspect", NeedsAPI: true, Description: "Show sandbox details", Run: a.cmdInspect},
			{Name: "stop", NeedsAPI: true, Description: "Stop a sandbox (unmount but preserve)", Run: a.cmdStop},
			{Name: "delete", NeedsAPI: true, Description: "Delete a sandbox and all data", Run: a.cmdDelete},
			{Name: "workspace", NeedsAPI: true, Description: "Get workspace path for a sandbox", Run: a.cmdWorkspace},
		},
	}

	diff := cliapp.CommandGroup{
		Title: "Diff & Approval",
		Commands: []cliapp.Command{
			{Name: "diff", NeedsAPI: true, Description: "Show changes in a sandbox", Run: a.cmdDiff, Flags: []cliapp.Flag{
				{Name: "raw", Default: "false", Description: "Show raw unified diff"},
			}},
			{Name: "approve", NeedsAPI: true, Description: "Apply sandbox changes to the repo", Run: a.cmdApprove, Flags: []cliapp.Flag{
				{Name: "message", Alias: "m", Default: "", Description: "Commit message"},
			}},
			{Name: "reject", NeedsAPI: true, Description: "Reject and discard sandbox changes", Run: a.cmdReject},
		},
	}

	driver := cliapp.CommandGroup{
		Title: "Driver",
		Commands: []cliapp.Command{
			{Name: "driver-info", NeedsAPI: true, Description: "Show driver information", Run: a.cmdDriverInfo},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, sandboxes, diff, driver, config}
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

// Sandbox types
type sandboxResponse struct {
	ID          string    `json:"id"`
	ScopePath   string    `json:"scopePath"`
	ProjectRoot string    `json:"projectRoot"`
	Owner       string    `json:"owner,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	MergedDir   string    `json:"mergedDir,omitempty"`
	SizeBytes   int64     `json:"sizeBytes"`
	FileCount   int       `json:"fileCount"`
	ErrorMsg    string    `json:"errorMessage,omitempty"`
}

type listResponse struct {
	Sandboxes  []sandboxResponse `json:"sandboxes"`
	TotalCount int               `json:"totalCount"`
}

type diffResponse struct {
	SandboxID     string `json:"sandboxId"`
	UnifiedDiff   string `json:"unifiedDiff"`
	TotalAdded    int    `json:"totalAdded"`
	TotalDeleted  int    `json:"totalDeleted"`
	TotalModified int    `json:"totalModified"`
	Files         []struct {
		FilePath   string `json:"filePath"`
		ChangeType string `json:"changeType"`
	} `json:"files"`
}

type approvalResponse struct {
	Success    bool   `json:"success"`
	Applied    int    `json:"applied"`
	CommitHash string `json:"commitHash,omitempty"`
	ErrorMsg   string `json:"error,omitempty"`
}

type healthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]string `json:"dependencies"`
	Error      string            `json:"error,omitempty"`
	Message    string            `json:"message,omitempty"`
	Operations map[string]any    `json:"operations,omitempty"`
}

func (a *App) cmdStatus(_ []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/health"), nil)
	if err != nil {
		return err
	}

	var parsed healthResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.Status != "" {
		fmt.Printf("Status: %s\n", parsed.Status)
		fmt.Printf("Ready: %v\n", parsed.Readiness)
		if parsed.Service != "" {
			fmt.Printf("Service: %s\n", parsed.Service)
		}
		if parsed.Version != "" {
			fmt.Printf("Version: %s\n", parsed.Version)
		}
		if len(parsed.Deps) > 0 {
			fmt.Println("Dependencies:")
			for key, value := range parsed.Deps {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdCreate(args []string) error {
	scope := a.core.CLI.String("scope")
	project := a.core.CLI.String("project")
	owner := a.core.CLI.String("owner")

	reqBody := map[string]interface{}{
		"scopePath": scope,
	}
	if project != "" {
		reqBody["projectRoot"] = project
	}
	if owner != "" {
		reqBody["owner"] = owner
	}

	body, err := a.core.APIClient.Post(a.apiPath("/sandboxes"), reqBody, nil)
	if err != nil {
		return err
	}

	var sb sandboxResponse
	if err := json.Unmarshal(body, &sb); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Created sandbox: %s\n", sb.ID)
	fmt.Printf("Status: %s\n", sb.Status)
	fmt.Printf("Scope: %s\n", sb.ScopePath)
	if sb.MergedDir != "" {
		fmt.Printf("Workspace: %s\n", sb.MergedDir)
	}

	return nil
}

func (a *App) cmdList(_ []string) error {
	status := a.core.CLI.String("status")
	owner := a.core.CLI.String("owner")
	asJSON := a.core.CLI.Bool("json")

	params := make(map[string]string)
	if status != "" {
		params["status"] = status
	}
	if owner != "" {
		params["owner"] = owner
	}

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes"), params)
	if err != nil {
		return err
	}

	if asJSON {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Sandboxes) == 0 {
		fmt.Println("No sandboxes found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tSCOPE\tOWNER\tCREATED\tFILES")
	for _, sb := range resp.Sandboxes {
		created := sb.CreatedAt.Format("2006-01-02 15:04")
		scope := sb.ScopePath
		if len(scope) > 40 {
			scope = "..." + scope[len(scope)-37:]
		}
		owner := sb.Owner
		if owner == "" {
			owner = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
			sb.ID[:8], sb.Status, scope, owner, created, sb.FileCount)
	}
	w.Flush()

	fmt.Printf("\nTotal: %d sandboxes\n", resp.TotalCount)
	return nil
}

func (a *App) cmdInspect(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: workspace-sandbox inspect <sandbox-id>")
	}

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes/"+args[0]), nil)
	if err != nil {
		return err
	}

	var sb sandboxResponse
	if err := json.Unmarshal(body, &sb); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("ID:          %s\n", sb.ID)
	fmt.Printf("Status:      %s\n", sb.Status)
	fmt.Printf("Scope:       %s\n", sb.ScopePath)
	fmt.Printf("Project:     %s\n", sb.ProjectRoot)
	fmt.Printf("Owner:       %s\n", sb.Owner)
	fmt.Printf("Created:     %s\n", sb.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Files:       %d\n", sb.FileCount)
	fmt.Printf("Size:        %d bytes\n", sb.SizeBytes)
	if sb.MergedDir != "" {
		fmt.Printf("Workspace:   %s\n", sb.MergedDir)
	}
	if sb.ErrorMsg != "" {
		fmt.Printf("Error:       %s\n", sb.ErrorMsg)
	}

	return nil
}

func (a *App) cmdStop(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: workspace-sandbox stop <sandbox-id>")
	}

	body, err := a.core.APIClient.Post(a.apiPath("/sandboxes/"+args[0]+"/stop"), nil, nil)
	if err != nil {
		return err
	}

	var sb sandboxResponse
	if err := json.Unmarshal(body, &sb); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Sandbox %s stopped\n", sb.ID)
	return nil
}

func (a *App) cmdDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: workspace-sandbox delete <sandbox-id>")
	}

	_, err := a.core.APIClient.Delete(a.apiPath("/sandboxes/"+args[0]), nil)
	if err != nil {
		return err
	}

	fmt.Printf("Sandbox %s deleted\n", args[0])
	return nil
}

func (a *App) cmdWorkspace(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: workspace-sandbox workspace <sandbox-id>")
	}

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes/"+args[0]+"/workspace"), nil)
	if err != nil {
		return err
	}

	var resp struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println(resp.Path)
	return nil
}

func (a *App) cmdDiff(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: workspace-sandbox diff <sandbox-id>")
	}

	raw := a.core.CLI.Bool("raw")

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes/"+args[0]+"/diff"), nil)
	if err != nil {
		return err
	}

	var diff diffResponse
	if err := json.Unmarshal(body, &diff); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if raw {
		fmt.Println(diff.UnifiedDiff)
		return nil
	}

	fmt.Printf("Sandbox: %s\n", diff.SandboxID)
	fmt.Printf("Changes: %d added, %d modified, %d deleted\n\n",
		diff.TotalAdded, diff.TotalModified, diff.TotalDeleted)

	if len(diff.Files) > 0 {
		fmt.Println("Files:")
		for _, f := range diff.Files {
			symbol := " "
			switch f.ChangeType {
			case "added":
				symbol = "+"
			case "deleted":
				symbol = "-"
			case "modified":
				symbol = "~"
			}
			fmt.Printf("  %s %s\n", symbol, f.FilePath)
		}
	}

	if diff.UnifiedDiff != "" {
		fmt.Println("\nDiff:")
		fmt.Println(diff.UnifiedDiff)
	}

	return nil
}

func (a *App) cmdApprove(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: workspace-sandbox approve <sandbox-id>")
	}

	message := a.core.CLI.String("message")

	reqBody := map[string]interface{}{
		"mode": "all",
	}
	if message != "" {
		reqBody["commitMessage"] = message
	}

	body, err := a.core.APIClient.Post(a.apiPath("/sandboxes/"+args[0]+"/approve"), reqBody, nil)
	if err != nil {
		return err
	}

	var resp approvalResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.Success {
		fmt.Printf("Applied %d changes\n", resp.Applied)
		if resp.CommitHash != "" {
			fmt.Printf("Commit: %s\n", resp.CommitHash)
		}
	} else {
		fmt.Printf("Approval failed: %s\n", resp.ErrorMsg)
		return fmt.Errorf("approval failed")
	}

	return nil
}

func (a *App) cmdReject(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: workspace-sandbox reject <sandbox-id>")
	}

	body, err := a.core.APIClient.Post(a.apiPath("/sandboxes/"+args[0]+"/reject"), nil, nil)
	if err != nil {
		return err
	}

	var sb sandboxResponse
	if err := json.Unmarshal(body, &sb); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Sandbox %s rejected\n", sb.ID)
	return nil
}

func (a *App) cmdDriverInfo(_ []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/driver/info"), nil)
	if err != nil {
		return err
	}

	cliutil.PrintJSON(body)
	return nil
}
