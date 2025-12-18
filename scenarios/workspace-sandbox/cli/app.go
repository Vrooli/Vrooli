package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"golang.org/x/term"
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
			{Name: "create", NeedsAPI: true, Description: "Create a new sandbox (--scope=PATH [--project=DIR] [--owner=ID])", Run: a.cmdCreate},
			{Name: "list", NeedsAPI: true, Description: "List sandboxes ([--status=STATUS] [--owner=ID] [--json])", Run: a.cmdList},
			{Name: "inspect", NeedsAPI: true, Description: "Show sandbox details", Run: a.cmdInspect},
			{Name: "stop", NeedsAPI: true, Description: "Stop a sandbox (unmount but preserve)", Run: a.cmdStop},
			{Name: "delete", NeedsAPI: true, Description: "Delete a sandbox and all data", Run: a.cmdDelete},
			{Name: "workspace", NeedsAPI: true, Description: "Get workspace path for a sandbox", Run: a.cmdWorkspace},
		},
	}

	execution := cliapp.CommandGroup{
		Title: "Process Execution",
		Commands: []cliapp.Command{
			{Name: "exec", NeedsAPI: true, Description: "Execute a command in a sandbox (--memory=MB --timeout=SEC --vrooli-aware)", Run: a.cmdExec},
			{Name: "run", NeedsAPI: true, Description: "Start a background process in a sandbox (--memory=MB --name=NAME)", Run: a.cmdRun},
			{Name: "processes", NeedsAPI: true, Description: "List processes in a sandbox ([--running] [--json])", Run: a.cmdProcesses},
			{Name: "kill", NeedsAPI: true, Description: "Kill a process in a sandbox (--pid=PID or --all)", Run: a.cmdKill},
			{Name: "logs", NeedsAPI: true, Description: "View process logs (--pid=PID [--follow] [--tail=N])", Run: a.cmdLogs},
			{Name: "shell", NeedsAPI: true, Description: "Open interactive shell in a sandbox (--vrooli-aware --memory=MB)", Run: a.cmdShell},
			{Name: "attach", NeedsAPI: true, Description: "Run interactive command with PTY (--vrooli-aware --memory=MB)", Run: a.cmdAttach},
		},
	}

	diff := cliapp.CommandGroup{
		Title: "Diff & Approval",
		Commands: []cliapp.Command{
			{Name: "diff", NeedsAPI: true, Description: "Show changes in a sandbox ([--raw])", Run: a.cmdDiff},
			{Name: "approve", NeedsAPI: true, Description: "Apply sandbox changes to the repo ([-m MESSAGE] [--force])", Run: a.cmdApprove},
			{Name: "reject", NeedsAPI: true, Description: "Reject and discard sandbox changes", Run: a.cmdReject},
			{Name: "conflicts", NeedsAPI: true, Description: "Check for conflicts with canonical repo [OT-P2-003]", Run: a.cmdConflicts},
			{Name: "rebase", NeedsAPI: true, Description: "Update sandbox baseline to current repo state [OT-P2-003]", Run: a.cmdRebase},
		},
	}

	driver := cliapp.CommandGroup{
		Title: "Driver",
		Commands: []cliapp.Command{
			{Name: "driver-info", NeedsAPI: true, Description: "Show driver information", Run: a.cmdDriverInfo},
		},
	}

	gc := cliapp.CommandGroup{
		Title: "Garbage Collection",
		Commands: []cliapp.Command{
			{Name: "gc", NeedsAPI: true, Description: "Run garbage collection ([--dry-run] [--max-age=DURATION] [--idle=DURATION] [--limit=N])", Run: a.cmdGC},
			{Name: "gc-preview", NeedsAPI: true, Description: "Preview what GC would collect (alias for gc --dry-run)", Run: a.cmdGCPreview},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, sandboxes, execution, diff, gc, driver, config}
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

// resolveSandboxID resolves a short sandbox ID prefix to a full UUID.
// Accepts either:
//   - Full UUID (36 chars with dashes): returned as-is
//   - Short prefix (e.g., first 8 chars): resolved by matching against existing sandboxes
//
// Returns an error if:
//   - The prefix matches multiple sandboxes (ambiguous)
//   - The prefix matches no sandboxes (not found)
func (a *App) resolveSandboxID(shortID string) (string, error) {
	shortID = strings.TrimSpace(shortID)
	if shortID == "" {
		return "", fmt.Errorf("sandbox ID required")
	}

	// Full UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 chars)
	if len(shortID) == 36 && strings.Count(shortID, "-") == 4 {
		return shortID, nil
	}

	// Short ID - need to resolve by fetching sandbox list
	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes"), nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch sandboxes for ID resolution: %w", err)
	}

	var resp listResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse sandbox list: %w", err)
	}

	// Find all sandboxes matching the prefix
	var matches []sandboxResponse
	shortIDLower := strings.ToLower(shortID)
	for _, sb := range resp.Sandboxes {
		if strings.HasPrefix(strings.ToLower(sb.ID), shortIDLower) {
			matches = append(matches, sb)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no sandbox found matching prefix %q", shortID)
	case 1:
		return matches[0].ID, nil
	default:
		// Ambiguous - show the user the matching IDs
		var ids []string
		for _, m := range matches {
			ids = append(ids, m.ID[:12]+"...")
		}
		return "", fmt.Errorf("ambiguous prefix %q matches %d sandboxes: %s",
			shortID, len(matches), strings.Join(ids, ", "))
	}
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
	var scope, project, owner string

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--scope="):
			scope = strings.TrimPrefix(arg, "--scope=")
		case strings.HasPrefix(arg, "-s="):
			scope = strings.TrimPrefix(arg, "-s=")
		case strings.HasPrefix(arg, "--project="):
			project = strings.TrimPrefix(arg, "--project=")
		case strings.HasPrefix(arg, "-p="):
			project = strings.TrimPrefix(arg, "-p=")
		case strings.HasPrefix(arg, "--owner="):
			owner = strings.TrimPrefix(arg, "--owner=")
		case strings.HasPrefix(arg, "-o="):
			owner = strings.TrimPrefix(arg, "-o=")
		}
	}

	if scope == "" {
		return fmt.Errorf("usage: workspace-sandbox create --scope=PATH [--project=DIR] [--owner=ID]")
	}

	reqBody := map[string]interface{}{
		"scopePath": scope,
	}
	if project != "" {
		reqBody["projectRoot"] = project
	}
	if owner != "" {
		reqBody["owner"] = owner
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/sandboxes"), nil, reqBody)
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

func (a *App) cmdList(args []string) error {
	var status, owner string
	var asJSON bool

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--status="):
			status = strings.TrimPrefix(arg, "--status=")
		case strings.HasPrefix(arg, "--owner="):
			owner = strings.TrimPrefix(arg, "--owner=")
		case arg == "--json":
			asJSON = true
		}
	}

	query := url.Values{}
	if status != "" {
		query.Set("status", status)
	}
	if owner != "" {
		query.Set("owner", owner)
	}

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes"), query)
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

	sandboxID, err := a.resolveSandboxID(args[0])
	if err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes/"+sandboxID), nil)
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

	sandboxID, err := a.resolveSandboxID(args[0])
	if err != nil {
		return err
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/sandboxes/"+sandboxID+"/stop"), nil, nil)
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

	sandboxID, err := a.resolveSandboxID(args[0])
	if err != nil {
		return err
	}

	_, err = a.core.APIClient.Request("DELETE", a.apiPath("/sandboxes/"+sandboxID), nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Sandbox %s deleted\n", sandboxID)
	return nil
}

func (a *App) cmdWorkspace(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: workspace-sandbox workspace <sandbox-id>")
	}

	sandboxID, err := a.resolveSandboxID(args[0])
	if err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes/"+sandboxID+"/workspace"), nil)
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
	var sandboxID string
	var raw bool

	for _, arg := range args {
		switch {
		case arg == "--raw":
			raw = true
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	if sandboxID == "" {
		return fmt.Errorf("usage: workspace-sandbox diff <sandbox-id> [--raw]")
	}

	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes/"+resolvedID+"/diff"), nil)
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
	var sandboxID, message string
	var force bool

	for i, arg := range args {
		switch {
		case arg == "-m" && i+1 < len(args):
			message = args[i+1]
		case strings.HasPrefix(arg, "-m="):
			message = strings.TrimPrefix(arg, "-m=")
		case strings.HasPrefix(arg, "--message="):
			message = strings.TrimPrefix(arg, "--message=")
		case arg == "--force" || arg == "-f":
			force = true
		case !strings.HasPrefix(arg, "-") && (i == 0 || !strings.HasPrefix(args[i-1], "-m")):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	if sandboxID == "" {
		return fmt.Errorf("usage: workspace-sandbox approve <sandbox-id> [-m MESSAGE] [--force]")
	}

	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	reqBody := map[string]interface{}{
		"mode": "all",
	}
	if message != "" {
		reqBody["commitMessage"] = message
	}
	if force {
		reqBody["force"] = true
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/sandboxes/"+resolvedID+"/approve"), nil, reqBody)
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

	sandboxID, err := a.resolveSandboxID(args[0])
	if err != nil {
		return err
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/sandboxes/"+sandboxID+"/reject"), nil, nil)
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

// --- GC Types and Commands [OT-P1-003] ---

type gcCollectedSandbox struct {
	ID        string    `json:"id"`
	ScopePath string    `json:"scopePath"`
	Status    string    `json:"status"`
	SizeBytes int64     `json:"sizeBytes"`
	CreatedAt time.Time `json:"createdAt"`
	Reason    string    `json:"reason"`
}

type gcResult struct {
	Collected           []gcCollectedSandbox `json:"collected"`
	TotalCollected      int                  `json:"totalCollected"`
	TotalBytesReclaimed int64                `json:"totalBytesReclaimed"`
	Errors              []struct {
		SandboxID string `json:"sandboxId"`
		Error     string `json:"error"`
	} `json:"errors,omitempty"`
	DryRun      bool      `json:"dryRun"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
}

func (a *App) cmdGC(args []string) error {
	return a.runGC(args, false)
}

func (a *App) cmdGCPreview(args []string) error {
	return a.runGC(args, true)
}

func (a *App) runGC(args []string, forceDryRun bool) error {
	var dryRun bool
	var maxAge, idleTimeout string
	var limit int
	var asJSON bool

	for i, arg := range args {
		switch {
		case arg == "--dry-run":
			dryRun = true
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "--max-age="):
			maxAge = strings.TrimPrefix(arg, "--max-age=")
		case strings.HasPrefix(arg, "--idle="):
			idleTimeout = strings.TrimPrefix(arg, "--idle=")
		case strings.HasPrefix(arg, "--limit="):
			limitStr := strings.TrimPrefix(arg, "--limit=")
			fmt.Sscanf(limitStr, "%d", &limit)
		case arg == "--limit" && i+1 < len(args):
			fmt.Sscanf(args[i+1], "%d", &limit)
		}
	}

	// Force dry run for preview command
	if forceDryRun {
		dryRun = true
	}

	// Build request
	reqBody := map[string]interface{}{
		"dryRun": dryRun,
	}

	policy := map[string]interface{}{}
	if maxAge != "" {
		policy["maxAge"] = maxAge
	}
	if idleTimeout != "" {
		policy["idleTimeout"] = idleTimeout
	}
	if len(policy) > 0 {
		reqBody["policy"] = policy
	}
	if limit > 0 {
		reqBody["limit"] = limit
	}

	// Choose endpoint
	endpoint := "/gc"
	if dryRun {
		endpoint = "/gc/preview"
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath(endpoint), nil, reqBody)
	if err != nil {
		return err
	}

	if asJSON {
		cliutil.PrintJSON(body)
		return nil
	}

	var result gcResult
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Print results
	if result.DryRun {
		fmt.Println("=== GC Preview (Dry Run) ===")
		fmt.Println("The following sandboxes WOULD be collected:")
		fmt.Println()
	} else {
		fmt.Println("=== GC Results ===")
		fmt.Println("The following sandboxes were collected:")
		fmt.Println()
	}

	if len(result.Collected) == 0 {
		fmt.Println("No sandboxes eligible for collection")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tSIZE\tCREATED\tREASON")
	for _, sb := range result.Collected {
		created := sb.CreatedAt.Format("2006-01-02 15:04")
		size := formatBytes(sb.SizeBytes)
		reason := sb.Reason
		if len(reason) > 40 {
			reason = reason[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			sb.ID[:8], sb.Status, size, created, reason)
	}
	w.Flush()

	fmt.Printf("\nTotal: %d sandboxes", result.TotalCollected)
	if result.TotalBytesReclaimed > 0 {
		fmt.Printf(", %s reclaimed", formatBytes(result.TotalBytesReclaimed))
	}
	fmt.Println()

	if len(result.Errors) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("  - %s: %s\n", e.SandboxID[:8], e.Error)
		}
	}

	duration := result.CompletedAt.Sub(result.StartedAt)
	fmt.Printf("\nCompleted in %v\n", duration.Round(time.Millisecond))

	return nil
}

// formatBytes formats bytes as human-readable string
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// --- Conflict Detection and Rebase Commands [OT-P2-003] ---

type conflictCheckResponse struct {
	HasConflict         bool      `json:"hasConflict"`
	BaseCommitHash      string    `json:"baseCommitHash,omitempty"`
	CurrentHash         string    `json:"currentHash,omitempty"`
	RepoChangedFiles    []string  `json:"repoChangedFiles,omitempty"`
	SandboxChangedFiles []string  `json:"sandboxChangedFiles,omitempty"`
	ConflictingFiles    []string  `json:"conflictingFiles,omitempty"`
	CheckedAt           time.Time `json:"checkedAt"`
}

type rebaseResponse struct {
	Success          bool      `json:"success"`
	PreviousBaseHash string    `json:"previousBaseHash,omitempty"`
	NewBaseHash      string    `json:"newBaseHash,omitempty"`
	ConflictingFiles []string  `json:"conflictingFiles,omitempty"`
	RepoChangedFiles []string  `json:"repoChangedFiles,omitempty"`
	Strategy         string    `json:"strategy"`
	ErrorMsg         string    `json:"error,omitempty"`
	RebasedAt        time.Time `json:"rebasedAt"`
}

func (a *App) cmdConflicts(args []string) error {
	var sandboxID string
	var asJSON bool

	for _, arg := range args {
		switch {
		case arg == "--json":
			asJSON = true
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	if sandboxID == "" {
		return fmt.Errorf("usage: workspace-sandbox conflicts <sandbox-id> [--json]")
	}

	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes/"+resolvedID+"/conflicts"), nil)
	if err != nil {
		return err
	}

	if asJSON {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp conflictCheckResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("=== Conflict Check ===")
	fmt.Println()

	if !resp.HasConflict {
		fmt.Println("✓ No conflicts detected")
		fmt.Printf("  Base commit:    %s\n", truncateHash(resp.BaseCommitHash))
		fmt.Printf("  Current commit: %s\n", truncateHash(resp.CurrentHash))
		return nil
	}

	fmt.Println("⚠ Conflicts detected!")
	fmt.Printf("  Base commit:    %s\n", truncateHash(resp.BaseCommitHash))
	fmt.Printf("  Current commit: %s\n", truncateHash(resp.CurrentHash))

	if len(resp.ConflictingFiles) > 0 {
		fmt.Printf("\nConflicting files (%d):\n", len(resp.ConflictingFiles))
		for _, f := range resp.ConflictingFiles {
			fmt.Printf("  ✗ %s\n", f)
		}
	}

	if len(resp.RepoChangedFiles) > 0 {
		fmt.Printf("\nRepo changed files (%d):\n", len(resp.RepoChangedFiles))
		for _, f := range resp.RepoChangedFiles {
			fmt.Printf("    %s\n", f)
		}
	}

	if len(resp.SandboxChangedFiles) > 0 {
		fmt.Printf("\nSandbox changed files (%d):\n", len(resp.SandboxChangedFiles))
		for _, f := range resp.SandboxChangedFiles {
			fmt.Printf("    %s\n", f)
		}
	}

	fmt.Println("\nOptions:")
	fmt.Println("  1. Rebase: workspace-sandbox rebase <sandbox-id>")
	fmt.Println("  2. Force:  workspace-sandbox approve <sandbox-id> --force")

	return nil
}

func (a *App) cmdRebase(args []string) error {
	var sandboxID string
	var asJSON bool

	for _, arg := range args {
		switch {
		case arg == "--json":
			asJSON = true
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	if sandboxID == "" {
		return fmt.Errorf("usage: workspace-sandbox rebase <sandbox-id> [--json]")
	}

	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	reqBody := map[string]interface{}{
		"strategy": "regenerate",
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/sandboxes/"+resolvedID+"/rebase"), nil, reqBody)
	if err != nil {
		return err
	}

	if asJSON {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp rebaseResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !resp.Success {
		fmt.Printf("Rebase failed: %s\n", resp.ErrorMsg)
		return fmt.Errorf("rebase failed")
	}

	fmt.Println("=== Rebase Successful ===")
	fmt.Println()
	fmt.Printf("Previous base: %s\n", truncateHash(resp.PreviousBaseHash))
	fmt.Printf("New base:      %s\n", truncateHash(resp.NewBaseHash))

	if len(resp.RepoChangedFiles) > 0 {
		fmt.Printf("\nRepo changed since sandbox creation (%d files):\n", len(resp.RepoChangedFiles))
		for _, f := range resp.RepoChangedFiles {
			fmt.Printf("    %s\n", f)
		}
	}

	if len(resp.ConflictingFiles) > 0 {
		fmt.Printf("\n⚠ Potential conflicts (%d files):\n", len(resp.ConflictingFiles))
		for _, f := range resp.ConflictingFiles {
			fmt.Printf("  ✗ %s\n", f)
		}
		fmt.Println("\nThese files were modified in both the sandbox and the repo.")
		fmt.Println("Review carefully before approving.")
	}

	return nil
}

// truncateHash returns a shortened version of a git hash for display.
func truncateHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

// --- Process Execution Commands ---

// execResponse mirrors the API response for exec
type execResponse struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	PID      int    `json:"pid,omitempty"`
	TimedOut bool   `json:"timedOut,omitempty"`
}

// cmdExec executes a command in a sandbox with optional resource limits.
// Usage: workspace-sandbox exec <sandbox-id> [OPTIONS] -- <command> [args...]
func (a *App) cmdExec(args []string) error {
	var sandboxID string
	var memoryMB, cpuTime, timeout, maxProcs, maxFiles int
	var vrooliAware, network, asJSON bool
	var workDir string
	envVars := make(map[string]string)

	// Find the -- separator
	cmdIdx := -1
	for i, arg := range args {
		if arg == "--" {
			cmdIdx = i
			break
		}
	}

	// Parse flags before --
	flagArgs := args
	if cmdIdx >= 0 {
		flagArgs = args[:cmdIdx]
	}

	for i := 0; i < len(flagArgs); i++ {
		arg := flagArgs[i]
		switch {
		case strings.HasPrefix(arg, "--memory="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--memory="), "%d", &memoryMB)
		case strings.HasPrefix(arg, "--cpu-time="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--cpu-time="), "%d", &cpuTime)
		case strings.HasPrefix(arg, "--timeout="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--timeout="), "%d", &timeout)
		case strings.HasPrefix(arg, "--max-procs="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--max-procs="), "%d", &maxProcs)
		case strings.HasPrefix(arg, "--max-files="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--max-files="), "%d", &maxFiles)
		case arg == "--vrooli-aware":
			vrooliAware = true
		case arg == "--network":
			network = true
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "--workdir="):
			workDir = strings.TrimPrefix(arg, "--workdir=")
		case strings.HasPrefix(arg, "--env="):
			envStr := strings.TrimPrefix(arg, "--env=")
			if idx := strings.Index(envStr, "="); idx > 0 {
				envVars[envStr[:idx]] = envStr[idx+1:]
			}
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	// Get command and args after --
	var command string
	var cmdArgs []string
	if cmdIdx >= 0 && cmdIdx+1 < len(args) {
		command = args[cmdIdx+1]
		if cmdIdx+2 < len(args) {
			cmdArgs = args[cmdIdx+2:]
		}
	}

	if sandboxID == "" || command == "" {
		return fmt.Errorf("usage: workspace-sandbox exec <sandbox-id> [OPTIONS] -- <command> [args...]\n\n" +
			"Options:\n" +
			"  --memory=<MB>        Memory limit in MB (0 = unlimited)\n" +
			"  --cpu-time=<SEC>     CPU time limit in seconds (0 = unlimited)\n" +
			"  --timeout=<SEC>      Wall-clock timeout in seconds (0 = unlimited)\n" +
			"  --max-procs=<N>      Max child processes (0 = unlimited)\n" +
			"  --max-files=<N>      Max open file descriptors (0 = unlimited)\n" +
			"  --network            Allow network access (default: blocked)\n" +
			"  --vrooli-aware       Use Vrooli-aware isolation (can access CLIs, localhost)\n" +
			"  --workdir=<PATH>     Working directory inside sandbox (default: /workspace)\n" +
			"  --env=KEY=VALUE      Set environment variable (repeatable)\n" +
			"  --json               Output result as JSON")
	}

	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	// Build request body
	reqBody := map[string]interface{}{
		"command": command,
	}
	if len(cmdArgs) > 0 {
		reqBody["args"] = cmdArgs
	}
	if memoryMB > 0 {
		reqBody["memoryLimitMB"] = memoryMB
	}
	if cpuTime > 0 {
		reqBody["cpuTimeSec"] = cpuTime
	}
	if timeout > 0 {
		reqBody["timeoutSec"] = timeout
	}
	if maxProcs > 0 {
		reqBody["maxProcesses"] = maxProcs
	}
	if maxFiles > 0 {
		reqBody["maxOpenFiles"] = maxFiles
	}
	if network {
		reqBody["allowNetwork"] = true
	}
	if vrooliAware {
		reqBody["isolationLevel"] = "vrooli-aware"
	}
	if workDir != "" {
		reqBody["workingDir"] = workDir
	}
	if len(envVars) > 0 {
		reqBody["env"] = envVars
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/sandboxes/"+resolvedID+"/exec"), nil, reqBody)
	if err != nil {
		return err
	}

	if asJSON {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp execResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Print stdout
	if resp.Stdout != "" {
		fmt.Print(resp.Stdout)
	}

	// Print stderr to stderr
	if resp.Stderr != "" {
		fmt.Fprint(os.Stderr, resp.Stderr)
	}

	// Show timeout warning
	if resp.TimedOut {
		fmt.Fprintf(os.Stderr, "\n[Process timed out with exit code %d]\n", resp.ExitCode)
	}

	// Return exit code as error if non-zero
	if resp.ExitCode != 0 {
		return fmt.Errorf("exit code %d", resp.ExitCode)
	}

	return nil
}

// runResponse mirrors the API response for start-process
type runResponse struct {
	PID       int       `json:"pid"`
	SandboxID string    `json:"sandboxId"`
	Command   string    `json:"command"`
	Name      string    `json:"name,omitempty"`
	StartedAt time.Time `json:"startedAt,omitempty"`
}

// cmdRun starts a background process in a sandbox.
// Usage: workspace-sandbox run <sandbox-id> [OPTIONS] -- <command> [args...]
func (a *App) cmdRun(args []string) error {
	var sandboxID string
	var memoryMB, cpuTime, maxProcs, maxFiles int
	var vrooliAware, network, asJSON bool
	var workDir, name string
	envVars := make(map[string]string)

	// Find the -- separator
	cmdIdx := -1
	for i, arg := range args {
		if arg == "--" {
			cmdIdx = i
			break
		}
	}

	// Parse flags before --
	flagArgs := args
	if cmdIdx >= 0 {
		flagArgs = args[:cmdIdx]
	}

	for i := 0; i < len(flagArgs); i++ {
		arg := flagArgs[i]
		switch {
		case strings.HasPrefix(arg, "--memory="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--memory="), "%d", &memoryMB)
		case strings.HasPrefix(arg, "--cpu-time="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--cpu-time="), "%d", &cpuTime)
		case strings.HasPrefix(arg, "--max-procs="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--max-procs="), "%d", &maxProcs)
		case strings.HasPrefix(arg, "--max-files="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--max-files="), "%d", &maxFiles)
		case arg == "--vrooli-aware":
			vrooliAware = true
		case arg == "--network":
			network = true
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "--workdir="):
			workDir = strings.TrimPrefix(arg, "--workdir=")
		case strings.HasPrefix(arg, "--name="):
			name = strings.TrimPrefix(arg, "--name=")
		case strings.HasPrefix(arg, "--env="):
			envStr := strings.TrimPrefix(arg, "--env=")
			if idx := strings.Index(envStr, "="); idx > 0 {
				envVars[envStr[:idx]] = envStr[idx+1:]
			}
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	// Get command and args after --
	var command string
	var cmdArgs []string
	if cmdIdx >= 0 && cmdIdx+1 < len(args) {
		command = args[cmdIdx+1]
		if cmdIdx+2 < len(args) {
			cmdArgs = args[cmdIdx+2:]
		}
	}

	if sandboxID == "" || command == "" {
		return fmt.Errorf("usage: workspace-sandbox run <sandbox-id> [OPTIONS] -- <command> [args...]\n\n" +
			"Options:\n" +
			"  --name=<NAME>        Friendly name for the process\n" +
			"  --memory=<MB>        Memory limit in MB (0 = unlimited)\n" +
			"  --cpu-time=<SEC>     CPU time limit in seconds (0 = unlimited)\n" +
			"  --max-procs=<N>      Max child processes (0 = unlimited)\n" +
			"  --max-files=<N>      Max open file descriptors (0 = unlimited)\n" +
			"  --network            Allow network access (default: blocked)\n" +
			"  --vrooli-aware       Use Vrooli-aware isolation (can access CLIs, localhost)\n" +
			"  --workdir=<PATH>     Working directory inside sandbox (default: /workspace)\n" +
			"  --env=KEY=VALUE      Set environment variable (repeatable)\n" +
			"  --json               Output result as JSON")
	}

	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	// Build request body
	reqBody := map[string]interface{}{
		"command": command,
	}
	if len(cmdArgs) > 0 {
		reqBody["args"] = cmdArgs
	}
	if name != "" {
		reqBody["name"] = name
	}
	if memoryMB > 0 {
		reqBody["memoryLimitMB"] = memoryMB
	}
	if cpuTime > 0 {
		reqBody["cpuTimeSec"] = cpuTime
	}
	if maxProcs > 0 {
		reqBody["maxProcesses"] = maxProcs
	}
	if maxFiles > 0 {
		reqBody["maxOpenFiles"] = maxFiles
	}
	if network {
		reqBody["allowNetwork"] = true
	}
	if vrooliAware {
		reqBody["isolationLevel"] = "vrooli-aware"
	}
	if workDir != "" {
		reqBody["workingDir"] = workDir
	}
	if len(envVars) > 0 {
		reqBody["env"] = envVars
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/sandboxes/"+resolvedID+"/start-process"), nil, reqBody)
	if err != nil {
		return err
	}

	if asJSON {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp runResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Started process: PID %d\n", resp.PID)
	if resp.Name != "" {
		fmt.Printf("Name: %s\n", resp.Name)
	}
	fmt.Printf("Command: %s\n", resp.Command)
	fmt.Printf("Sandbox: %s\n", resp.SandboxID)

	return nil
}

// processInfo represents a tracked process
type processInfo struct {
	PID       int       `json:"pid"`
	Command   string    `json:"command"`
	Running   bool      `json:"running"`
	StartedAt time.Time `json:"startedAt"`
	SessionID string    `json:"sessionId,omitempty"`
}

// processListResponse represents the API response for process listing
type processListResponse struct {
	Processes []processInfo `json:"processes"`
	Total     int           `json:"total"`
	Running   int           `json:"running"`
}

// cmdProcesses lists processes in a sandbox.
func (a *App) cmdProcesses(args []string) error {
	var sandboxID string
	var runningOnly, asJSON bool

	for _, arg := range args {
		switch {
		case arg == "--running":
			runningOnly = true
		case arg == "--json":
			asJSON = true
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	if sandboxID == "" {
		return fmt.Errorf("usage: workspace-sandbox processes <sandbox-id> [--running] [--json]")
	}

	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	query := url.Values{}
	if runningOnly {
		query.Set("running", "true")
	}

	body, err := a.core.APIClient.Get(a.apiPath("/sandboxes/"+resolvedID+"/processes"), query)
	if err != nil {
		return err
	}

	if asJSON {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp processListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Processes) == 0 {
		fmt.Println("No processes found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PID\tSTATUS\tCOMMAND\tSTARTED")
	for _, p := range resp.Processes {
		status := "stopped"
		if p.Running {
			status = "running"
		}
		started := p.StartedAt.Format("15:04:05")
		command := p.Command
		if len(command) > 40 {
			command = command[:37] + "..."
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", p.PID, status, command, started)
	}
	w.Flush()

	fmt.Printf("\nTotal: %d processes (%d running)\n", resp.Total, resp.Running)
	return nil
}

// cmdKill kills a process in a sandbox.
func (a *App) cmdKill(args []string) error {
	var sandboxID string
	var pid int
	var killAll bool

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--pid="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--pid="), "%d", &pid)
		case arg == "--all":
			killAll = true
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	if sandboxID == "" {
		return fmt.Errorf("usage: workspace-sandbox kill <sandbox-id> (--pid=<PID> | --all)")
	}

	if pid == 0 && !killAll {
		return fmt.Errorf("must specify --pid=<PID> or --all")
	}

	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	if killAll {
		// Kill all processes
		body, err := a.core.APIClient.Request("DELETE", a.apiPath("/sandboxes/"+resolvedID+"/processes"), nil, nil)
		if err != nil {
			return err
		}

		var resp struct {
			Killed int      `json:"killed"`
			Errors []string `json:"errors,omitempty"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Killed %d processes\n", resp.Killed)
		if len(resp.Errors) > 0 {
			fmt.Println("Errors:")
			for _, e := range resp.Errors {
				fmt.Printf("  - %s\n", e)
			}
		}
	} else {
		// Kill specific process
		_, err := a.core.APIClient.Request("DELETE", a.apiPath(fmt.Sprintf("/sandboxes/%s/processes/%d", resolvedID, pid)), nil, nil)
		if err != nil {
			return err
		}

		fmt.Printf("Killed process %d\n", pid)
	}

	return nil
}

// logResponse mirrors the API response for logs
type logResponse struct {
	PID       int    `json:"pid"`
	SandboxID string `json:"sandboxId"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	IsActive  bool   `json:"isActive"`
	Content   string `json:"content"`
}

// logsListResponse mirrors the API response for listing logs
type logsListResponse struct {
	Logs  []logResponse `json:"logs"`
	Total int           `json:"total"`
}

// cmdLogs displays logs for a process in a sandbox.
// Usage: workspace-sandbox logs <sandbox-id> --pid=<PID> [--follow] [--tail=N]
func (a *App) cmdLogs(args []string) error {
	var sandboxID string
	var pid, tail int
	var follow, asJSON bool

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--pid="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--pid="), "%d", &pid)
		case strings.HasPrefix(arg, "--tail="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--tail="), "%d", &tail)
		case arg == "--follow" || arg == "-f":
			follow = true
		case arg == "--json":
			asJSON = true
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	if sandboxID == "" {
		return fmt.Errorf("usage: workspace-sandbox logs <sandbox-id> --pid=<PID> [--follow] [--tail=N] [--json]")
	}

	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	// If no PID specified, list all logs
	if pid == 0 {
		body, err := a.core.APIClient.Get(a.apiPath("/sandboxes/"+resolvedID+"/logs"), nil)
		if err != nil {
			return err
		}

		if asJSON {
			cliutil.PrintJSON(body)
			return nil
		}

		var resp logsListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(resp.Logs) == 0 {
			fmt.Println("No logs found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PID\tSIZE\tACTIVE\tPATH")
		for _, log := range resp.Logs {
			status := "no"
			if log.IsActive {
				status = "yes"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", log.PID, formatBytes(log.SizeBytes), status, log.Path)
		}
		w.Flush()
		fmt.Printf("\nTotal: %d logs\n", resp.Total)
		return nil
	}

	// Follow mode - use SSE streaming
	if follow {
		return a.streamLogs(resolvedID, pid)
	}

	// Build query for specific log
	query := url.Values{}
	if tail > 0 {
		query.Set("tail", fmt.Sprintf("%d", tail))
	}

	body, err := a.core.APIClient.Get(a.apiPath(fmt.Sprintf("/sandboxes/%s/processes/%d/logs", resolvedID, pid)), query)
	if err != nil {
		return err
	}

	if asJSON {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp logResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Print log content directly
	fmt.Print(resp.Content)

	return nil
}

// streamLogs streams logs via SSE
func (a *App) streamLogs(sandboxID string, pid int) error {
	// For SSE streaming, we need to make a raw HTTP request
	// The CLI core doesn't support SSE natively, so we'll use polling
	// as a fallback for v1

	fmt.Printf("Streaming logs for PID %d (press Ctrl+C to stop)...\n\n", pid)

	var lastOffset int64 = 0
	for {
		query := url.Values{}
		query.Set("offset", fmt.Sprintf("%d", lastOffset))

		body, err := a.core.APIClient.Get(a.apiPath(fmt.Sprintf("/sandboxes/%s/processes/%d/logs", sandboxID, pid)), query)
		if err != nil {
			return err
		}

		var resp logResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Print new content
		if len(resp.Content) > 0 {
			fmt.Print(resp.Content)
			lastOffset = resp.SizeBytes
		}

		// Check if process is still active
		if !resp.IsActive {
			fmt.Println("\n[Process ended]")
			return nil
		}

		// Poll every 500ms
		time.Sleep(500 * time.Millisecond)
	}
}

// --- Interactive Session Commands (Phase 6) ---

// cmdShell opens an interactive shell session in a sandbox.
// Usage: workspace-sandbox shell <sandbox-id> [OPTIONS]
func (a *App) cmdShell(args []string) error {
	var sandboxID string
	var vrooliAware bool
	var memoryMB int
	var network bool

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--memory="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--memory="), "%d", &memoryMB)
		case arg == "--vrooli-aware":
			vrooliAware = true
		case arg == "--network":
			network = true
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	if sandboxID == "" {
		return fmt.Errorf("usage: workspace-sandbox shell <sandbox-id> [--vrooli-aware] [--network] [--memory=MB]")
	}

	// Resolve sandbox ID
	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	// Default shell
	shell := "/bin/sh"
	if envShell := os.Getenv("SHELL"); envShell != "" {
		// Use user's preferred shell if available
		shell = envShell
	}

	// Run interactive session
	return a.runInteractiveSession(resolvedID, shell, nil, vrooliAware, network, memoryMB)
}

// cmdAttach attaches to a running command with PTY support.
// Usage: workspace-sandbox attach <sandbox-id> [OPTIONS] -- <command> [args...]
func (a *App) cmdAttach(args []string) error {
	var sandboxID string
	var vrooliAware bool
	var memoryMB int
	var network bool

	// Find the -- separator
	cmdIdx := -1
	for i, arg := range args {
		if arg == "--" {
			cmdIdx = i
			break
		}
	}

	// Parse flags before --
	flagArgs := args
	if cmdIdx >= 0 {
		flagArgs = args[:cmdIdx]
	}

	for _, arg := range flagArgs {
		switch {
		case strings.HasPrefix(arg, "--memory="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--memory="), "%d", &memoryMB)
		case arg == "--vrooli-aware":
			vrooliAware = true
		case arg == "--network":
			network = true
		case !strings.HasPrefix(arg, "-"):
			if sandboxID == "" {
				sandboxID = arg
			}
		}
	}

	// Get command and args after --
	var command string
	var cmdArgs []string
	if cmdIdx >= 0 && cmdIdx+1 < len(args) {
		command = args[cmdIdx+1]
		if cmdIdx+2 < len(args) {
			cmdArgs = args[cmdIdx+2:]
		}
	}

	if sandboxID == "" || command == "" {
		return fmt.Errorf("usage: workspace-sandbox attach <sandbox-id> [--vrooli-aware] [--network] [--memory=MB] -- <command> [args...]")
	}

	// Resolve sandbox ID
	resolvedID, err := a.resolveSandboxID(sandboxID)
	if err != nil {
		return err
	}

	// Run interactive session
	return a.runInteractiveSession(resolvedID, command, cmdArgs, vrooliAware, network, memoryMB)
}

// interactiveMessage is the WebSocket message format.
type interactiveMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
	Code int    `json:"code,omitempty"`
}

// interactiveStartRequest is sent to initiate an interactive session.
type interactiveStartRequest struct {
	Command        string `json:"command"`
	Args           []string `json:"args,omitempty"`
	IsolationLevel string `json:"isolationLevel,omitempty"`
	AllowNetwork   bool   `json:"allowNetwork,omitempty"`
	MemoryLimitMB  int    `json:"memoryLimitMB,omitempty"`
	Cols           int    `json:"cols,omitempty"`
	Rows           int    `json:"rows,omitempty"`
}

// runInteractiveSession connects to the WebSocket endpoint and runs an interactive session.
func (a *App) runInteractiveSession(sandboxID, command string, cmdArgs []string, vrooliAware, network bool, memoryMB int) error {
	// Build WebSocket URL from HTTP base URL
	baseURL := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/api/v1/sandboxes/%s/exec-interactive", wsURL, sandboxID)

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to interactive session: %w", err)
	}
	defer conn.Close()

	// Get terminal size
	cols, rows := getTerminalSize()

	// Send start request
	startReq := interactiveStartRequest{
		Command: command,
		Args:    cmdArgs,
		Cols:    cols,
		Rows:    rows,
	}
	if vrooliAware {
		startReq.IsolationLevel = "vrooli-aware"
	}
	if network {
		startReq.AllowNetwork = true
	}
	if memoryMB > 0 {
		startReq.MemoryLimitMB = memoryMB
	}

	if err := conn.WriteJSON(startReq); err != nil {
		return fmt.Errorf("failed to send start request: %w", err)
	}

	// Set up terminal raw mode
	oldState, err := makeRaw(os.Stdin.Fd())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not set terminal to raw mode: %v\n", err)
	} else {
		defer restoreTerminal(os.Stdin.Fd(), oldState)
	}

	// Channel for coordinating shutdown
	done := make(chan struct{})
	var exitCode int

	// Read from WebSocket and write to stdout
	go func() {
		defer close(done)
		for {
			var msg interactiveMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}

			switch msg.Type {
			case "stdout", "stderr":
				os.Stdout.Write([]byte(msg.Data))
			case "exit":
				exitCode = msg.Code
				return
			case "error":
				fmt.Fprintf(os.Stderr, "\nError: %s\n", msg.Data)
				return
			}
		}
	}()

	// Read from stdin and send to WebSocket
	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-done:
				return
			default:
			}

			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				msg := interactiveMessage{
					Type: "stdin",
					Data: string(buf[:n]),
				}
				if err := conn.WriteJSON(msg); err != nil {
					return
				}
			}
		}
	}()

	// Handle terminal resize (SIGWINCH) - simplified version
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				// Periodically check for resize
				newCols, newRows := getTerminalSize()
				if newCols != cols || newRows != rows {
					cols, rows = newCols, newRows
					conn.WriteJSON(interactiveMessage{
						Type: "resize",
						Cols: cols,
						Rows: rows,
					})
				}
			}
		}
	}()

	// Wait for session to end
	<-done

	if exitCode != 0 {
		return fmt.Errorf("process exited with code %d", exitCode)
	}
	return nil
}

// getTerminalSize returns the current terminal dimensions.
func getTerminalSize() (cols, rows int) {
	// Default size
	cols, rows = 80, 24

	// Try to get actual size using stty
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	output, err := cmd.Output()
	if err == nil {
		fmt.Sscanf(string(output), "%d %d", &rows, &cols)
	}
	return
}

// makeRaw puts the terminal into raw mode and returns the previous state.
func makeRaw(fd uintptr) (*term.State, error) {
	return term.MakeRaw(int(fd))
}

// restoreTerminal restores the terminal to its previous state.
func restoreTerminal(fd uintptr, state *term.State) {
	if state != nil {
		term.Restore(int(fd), state)
	}
}
