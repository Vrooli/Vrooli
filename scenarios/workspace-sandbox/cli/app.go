package main

import (
	"encoding/json"
	"fmt"
	"net/url"
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
			{Name: "create", NeedsAPI: true, Description: "Create a new sandbox (--scope=PATH [--project=DIR] [--owner=ID])", Run: a.cmdCreate},
			{Name: "list", NeedsAPI: true, Description: "List sandboxes ([--status=STATUS] [--owner=ID] [--json])", Run: a.cmdList},
			{Name: "inspect", NeedsAPI: true, Description: "Show sandbox details", Run: a.cmdInspect},
			{Name: "stop", NeedsAPI: true, Description: "Stop a sandbox (unmount but preserve)", Run: a.cmdStop},
			{Name: "delete", NeedsAPI: true, Description: "Delete a sandbox and all data", Run: a.cmdDelete},
			{Name: "workspace", NeedsAPI: true, Description: "Get workspace path for a sandbox", Run: a.cmdWorkspace},
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

	return []cliapp.CommandGroup{health, sandboxes, diff, gc, driver, config}
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
		fmt.Println("The following sandboxes WOULD be collected:\n")
	} else {
		fmt.Println("=== GC Results ===")
		fmt.Println("The following sandboxes were collected:\n")
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

	fmt.Println("=== Conflict Check ===\n")

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

	fmt.Println("=== Rebase Successful ===\n")
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
