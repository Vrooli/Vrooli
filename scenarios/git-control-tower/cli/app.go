package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "git-control-tower"
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
		Description:       "Git Control Tower CLI",
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
			{Name: "status", NeedsAPI: true, Description: "Check API health", Run: a.cmdStatus},
		},
	}

	repo := cliapp.CommandGroup{
		Title: "Repository",
		Commands: []cliapp.Command{
			{Name: "repo-status", NeedsAPI: true, Description: "Show repository status (branch + changed files)", Run: a.cmdRepoStatus},
			{Name: "diff", NeedsAPI: true, Description: "Show git diff (--path=FILE --staged)", Run: a.cmdDiff},
			{Name: "stage", NeedsAPI: true, Description: "Stage files (FILE... or --scope=scenario:name)", Run: a.cmdStage},
			{Name: "unstage", NeedsAPI: true, Description: "Unstage files (FILE... or --scope=scenario:name)", Run: a.cmdUnstage},
			{Name: "commit", NeedsAPI: true, Description: "Create a commit (-m MESSAGE [--conventional])", Run: a.cmdCommit},
			{Name: "sync-status", NeedsAPI: true, Description: "Check push/pull status ([--fetch] [--remote=NAME])", Run: a.cmdSyncStatus},
			{Name: "branch-list", NeedsAPI: true, Description: "List branches", Run: a.cmdBranchList},
			{Name: "branch-create", NeedsAPI: true, Description: "Create branch NAME [--from=BASE] [--no-checkout] [--allow-dirty]", Run: a.cmdBranchCreate},
			{Name: "branch-switch", NeedsAPI: true, Description: "Switch branch NAME [--allow-dirty] [--track-remote]", Run: a.cmdBranchSwitch},
			{Name: "branch-publish", NeedsAPI: true, Description: "Publish current branch ([--remote=NAME] [--branch=NAME] [--fetch])", Run: a.cmdBranchPublish},
		},
	}

	audit := cliapp.CommandGroup{
		Title: "Audit",
		Commands: []cliapp.Command{
			{Name: "audit", NeedsAPI: true, Description: "Query audit logs ([--operation=TYPE] [--limit=N])", Run: a.cmdAudit},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, repo, audit, config}
}

func (a *App) apiPath(v1Path string) string {
	return apiPathFromBaseURL(a.core.HTTPClient.BaseURL(), v1Path)
}

func apiPathFromBaseURL(baseURL string, v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Readiness bool   `json:"readiness"`
	Timestamp string `json:"timestamp"`
	Deps      map[string]struct {
		Connected bool   `json:"connected"`
		Status    string `json:"status"`
	} `json:"dependencies"`
	Error      string         `json:"error,omitempty"`
	Message    string         `json:"message,omitempty"`
	Operations map[string]any `json:"operations,omitempty"`
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
				state := "disconnected"
				if value.Connected {
					state = "connected"
				}
				if value.Status != "" {
					fmt.Printf("  %s: %s (%s)\n", key, state, value.Status)
					continue
				}
				fmt.Printf("  %s: %s\n", key, state)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// [REQ:GCT-OT-P0-002] Repository status API

type repoStatusResponse struct {
	RepoDir string `json:"repo_dir"`
	Branch  struct {
		Head     string `json:"head"`
		Upstream string `json:"upstream"`
		Ahead    int    `json:"ahead"`
		Behind   int    `json:"behind"`
	} `json:"branch"`
	Summary struct {
		Staged    int `json:"staged"`
		Unstaged  int `json:"unstaged"`
		Untracked int `json:"untracked"`
		Conflicts int `json:"conflicts"`
	} `json:"summary"`
}

func (a *App) cmdRepoStatus(_ []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/repo/status"), nil)
	if err != nil {
		return err
	}

	var parsed repoStatusResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.RepoDir != "" {
		fmt.Printf("Repo: %s\n", parsed.RepoDir)
		if parsed.Branch.Head != "" {
			fmt.Printf("Branch: %s\n", parsed.Branch.Head)
		}
		if parsed.Branch.Upstream != "" {
			fmt.Printf("Upstream: %s (ahead %d, behind %d)\n", parsed.Branch.Upstream, parsed.Branch.Ahead, parsed.Branch.Behind)
		}
		fmt.Printf("Changes: staged=%d unstaged=%d untracked=%d conflicts=%d\n",
			parsed.Summary.Staged, parsed.Summary.Unstaged, parsed.Summary.Untracked, parsed.Summary.Conflicts)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// [REQ:GCT-OT-P0-003] File diff endpoint

type diffResponse struct {
	RepoDir string `json:"repo_dir"`
	Path    string `json:"path"`
	Staged  bool   `json:"staged"`
	HasDiff bool   `json:"has_diff"`
	Stats   struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		Files     int `json:"files"`
	} `json:"stats"`
	Raw string `json:"raw"`
}

func (a *App) cmdDiff(args []string) error {
	var path string
	var staged bool

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--path="):
			path = strings.TrimPrefix(arg, "--path=")
		case arg == "--staged":
			staged = true
		}
	}

	query := url.Values{}
	if path != "" {
		query.Set("path", path)
	}
	if staged {
		query.Set("staged", "true")
	}

	body, err := a.core.APIClient.Get(a.apiPath("/repo/diff"), query)
	if err != nil {
		return err
	}

	var parsed diffResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.RepoDir != "" {
		if !parsed.HasDiff {
			fmt.Println("No changes")
			return nil
		}
		fmt.Printf("Diff for: %s\n", parsed.Path)
		if parsed.Staged {
			fmt.Println("(staged changes)")
		}
		fmt.Printf("Stats: +%d -%d (%d files)\n",
			parsed.Stats.Additions, parsed.Stats.Deletions, parsed.Stats.Files)
		if parsed.Raw != "" {
			fmt.Println("---")
			fmt.Println(parsed.Raw)
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// [REQ:GCT-OT-P0-004] Stage/unstage operations

type stageRequest struct {
	Paths []string `json:"paths"`
	Scope string   `json:"scope,omitempty"`
}

type stageResponse struct {
	Success  bool     `json:"success"`
	Staged   []string `json:"staged"`
	Unstaged []string `json:"unstaged"`
	Failed   []string `json:"failed"`
	Errors   []string `json:"errors"`
}

func (a *App) cmdStage(args []string) error {
	var scope string
	var paths []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "--scope=") {
			scope = strings.TrimPrefix(arg, "--scope=")
		} else if !strings.HasPrefix(arg, "-") {
			paths = append(paths, arg)
		}
	}

	if len(paths) == 0 && scope == "" {
		return fmt.Errorf("usage: stage FILE... or --scope=scenario:name")
	}

	req := stageRequest{
		Paths: paths,
		Scope: scope,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/stage"), nil, req)
	if err != nil {
		return err
	}

	var parsed stageResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil {
		if parsed.Success {
			fmt.Printf("Staged %d file(s)\n", len(parsed.Staged))
			for _, f := range parsed.Staged {
				fmt.Printf("  + %s\n", f)
			}
		} else {
			fmt.Println("Staging failed:")
			for _, e := range parsed.Errors {
				fmt.Printf("  ! %s\n", e)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdUnstage(args []string) error {
	var scope string
	var paths []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "--scope=") {
			scope = strings.TrimPrefix(arg, "--scope=")
		} else if !strings.HasPrefix(arg, "-") {
			paths = append(paths, arg)
		}
	}

	if len(paths) == 0 && scope == "" {
		return fmt.Errorf("usage: unstage FILE... or --scope=scenario:name")
	}

	req := stageRequest{
		Paths: paths,
		Scope: scope,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/unstage"), nil, req)
	if err != nil {
		return err
	}

	var parsed stageResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil {
		if parsed.Success {
			fmt.Printf("Unstaged %d file(s)\n", len(parsed.Unstaged))
			for _, f := range parsed.Unstaged {
				fmt.Printf("  - %s\n", f)
			}
		} else {
			fmt.Println("Unstaging failed:")
			for _, e := range parsed.Errors {
				fmt.Printf("  ! %s\n", e)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// [REQ:GCT-OT-P0-005] Commit composition API

type commitRequest struct {
	Message              string `json:"message"`
	ValidateConventional bool   `json:"validate_conventional,omitempty"`
}

type commitResponse struct {
	Success          bool     `json:"success"`
	Hash             string   `json:"hash,omitempty"`
	Message          string   `json:"message,omitempty"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
	Error            string   `json:"error,omitempty"`
}

func (a *App) cmdCommit(args []string) error {
	var message string
	var conventional bool

	for i, arg := range args {
		switch {
		case arg == "-m" && i+1 < len(args):
			message = args[i+1]
		case strings.HasPrefix(arg, "-m="):
			message = strings.TrimPrefix(arg, "-m=")
		case strings.HasPrefix(arg, "--message="):
			message = strings.TrimPrefix(arg, "--message=")
		case arg == "--conventional":
			conventional = true
		}
	}

	if message == "" {
		return fmt.Errorf("usage: commit -m MESSAGE [--conventional]")
	}

	req := commitRequest{
		Message:              message,
		ValidateConventional: conventional,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/commit"), nil, req)
	if err != nil {
		return err
	}

	var parsed commitResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil {
		if parsed.Success {
			fmt.Printf("Committed: %s\n", parsed.Hash)
			fmt.Printf("Message: %s\n", parsed.Message)
		} else {
			fmt.Println("Commit failed:")
			if parsed.Error != "" {
				fmt.Printf("  Error: %s\n", parsed.Error)
			}
			for _, e := range parsed.ValidationErrors {
				fmt.Printf("  ! %s\n", e)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// [REQ:GCT-OT-P0-006] Push/pull status

type syncStatusResponse struct {
	Branch                string   `json:"branch"`
	Upstream              string   `json:"upstream,omitempty"`
	RemoteURL             string   `json:"remote_url,omitempty"`
	Ahead                 int      `json:"ahead"`
	Behind                int      `json:"behind"`
	HasUpstream           bool     `json:"has_upstream"`
	CanPush               bool     `json:"can_push"`
	CanPull               bool     `json:"can_pull"`
	NeedsPull             bool     `json:"needs_pull"`
	NeedsPush             bool     `json:"needs_push"`
	HasUncommittedChanges bool     `json:"has_uncommitted_changes"`
	SafetyWarnings        []string `json:"safety_warnings,omitempty"`
	Recommendations       []string `json:"recommendations,omitempty"`
	Fetched               bool     `json:"fetched"`
	FetchError            string   `json:"fetch_error,omitempty"`
}

func (a *App) cmdSyncStatus(args []string) error {
	var fetch bool
	var remote string

	for _, arg := range args {
		switch {
		case arg == "--fetch":
			fetch = true
		case strings.HasPrefix(arg, "--remote="):
			remote = strings.TrimPrefix(arg, "--remote=")
		}
	}

	query := url.Values{}
	if fetch {
		query.Set("fetch", "true")
	}
	if remote != "" {
		query.Set("remote", remote)
	}

	body, err := a.core.APIClient.Get(a.apiPath("/repo/sync-status"), query)
	if err != nil {
		return err
	}

	var resp syncStatusResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil && resp.Branch != "" {
		// Branch info
		fmt.Printf("Branch: %s\n", resp.Branch)
		if resp.Upstream != "" {
			fmt.Printf("Upstream: %s\n", resp.Upstream)
		}
		if resp.RemoteURL != "" {
			fmt.Printf("Remote: %s\n", resp.RemoteURL)
		}

		// Sync status
		if resp.HasUpstream {
			fmt.Printf("Ahead: %d  Behind: %d\n", resp.Ahead, resp.Behind)
		} else {
			fmt.Println("No upstream configured")
		}

		// Action indicators
		var actions []string
		if resp.CanPush {
			actions = append(actions, "can push")
		}
		if resp.CanPull {
			actions = append(actions, "can pull")
		}
		if resp.HasUncommittedChanges {
			actions = append(actions, "has uncommitted changes")
		}
		if len(actions) > 0 {
			fmt.Printf("Status: %s\n", strings.Join(actions, ", "))
		}

		// Warnings
		if len(resp.SafetyWarnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, w := range resp.SafetyWarnings {
				fmt.Printf("  ! %s\n", w)
			}
		}

		// Recommendations
		if len(resp.Recommendations) > 0 {
			fmt.Println("\nRecommendations:")
			for _, r := range resp.Recommendations {
				fmt.Printf("  -> %s\n", r)
			}
		}

		// Fetch info
		if resp.Fetched {
			fmt.Println("\n(fetched fresh data from remote)")
		}
		if resp.FetchError != "" {
			fmt.Printf("\n! Fetch error: %s\n", resp.FetchError)
		}

		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// [REQ:GCT-OT-P1-001] Branch operations

type branchInfo struct {
	Name      string `json:"name"`
	Upstream  string `json:"upstream,omitempty"`
	OID       string `json:"oid,omitempty"`
	IsCurrent bool   `json:"is_current,omitempty"`
}

type branchListResponse struct {
	Current string       `json:"current"`
	Locals  []branchInfo `json:"locals"`
	Remotes []branchInfo `json:"remotes"`
}

type branchWarning struct {
	Message              string `json:"message"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
	RequiresTracking     bool   `json:"requires_tracking,omitempty"`
	RequiresFetch        bool   `json:"requires_fetch,omitempty"`
}

type branchCreateRequest struct {
	Name       string `json:"name"`
	From       string `json:"from,omitempty"`
	Checkout   bool   `json:"checkout,omitempty"`
	AllowDirty bool   `json:"allow_dirty,omitempty"`
}

type branchCreateResponse struct {
	Success          bool           `json:"success"`
	Branch           *branchInfo    `json:"branch,omitempty"`
	Warning          *branchWarning `json:"warning,omitempty"`
	Error            string         `json:"error,omitempty"`
	ValidationErrors []string       `json:"validation_errors,omitempty"`
}

type branchSwitchRequest struct {
	Name        string `json:"name"`
	AllowDirty  bool   `json:"allow_dirty,omitempty"`
	TrackRemote bool   `json:"track_remote,omitempty"`
}

type branchSwitchResponse struct {
	Success bool           `json:"success"`
	Branch  *branchInfo    `json:"branch,omitempty"`
	Warning *branchWarning `json:"warning,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type branchPublishRequest struct {
	Remote string `json:"remote,omitempty"`
	Branch string `json:"branch,omitempty"`
	Fetch  bool   `json:"fetch,omitempty"`
}

type branchPublishResponse struct {
	Success bool           `json:"success"`
	Remote  string         `json:"remote"`
	Branch  string         `json:"branch"`
	Warning *branchWarning `json:"warning,omitempty"`
	Error   string         `json:"error,omitempty"`
}

func (a *App) cmdBranchList(_ []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/repo/branches"), nil)
	if err != nil {
		return err
	}

	var resp branchListResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil && resp.Current != "" {
		fmt.Printf("Current: %s\n", resp.Current)
		fmt.Println("Local branches:")
		for _, branch := range resp.Locals {
			prefix := "  "
			if branch.IsCurrent {
				prefix = "* "
			}
			if branch.Upstream != "" {
				fmt.Printf("%s%s -> %s\n", prefix, branch.Name, branch.Upstream)
			} else {
				fmt.Printf("%s%s\n", prefix, branch.Name)
			}
		}
		if len(resp.Remotes) > 0 {
			fmt.Println("Remote branches:")
			for _, branch := range resp.Remotes {
				fmt.Printf("  %s\n", branch.Name)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdBranchCreate(args []string) error {
	var name string
	var from string
	checkout := true
	allowDirty := false

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--from="):
			from = strings.TrimPrefix(arg, "--from=")
		case arg == "--no-checkout":
			checkout = false
		case arg == "--checkout":
			checkout = true
		case arg == "--allow-dirty":
			allowDirty = true
		case !strings.HasPrefix(arg, "-") && name == "":
			name = arg
		}
	}

	if name == "" {
		return fmt.Errorf("usage: branch-create NAME [--from=BASE] [--no-checkout] [--allow-dirty]")
	}

	req := branchCreateRequest{
		Name:       name,
		From:       from,
		Checkout:   checkout,
		AllowDirty: allowDirty,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/branch/create"), nil, req)
	if err != nil {
		return err
	}

	var resp branchCreateResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil {
		if resp.Success {
			fmt.Printf("Created branch: %s\n", name)
			return nil
		}
		if resp.Warning != nil {
			fmt.Printf("Warning: %s\n", resp.Warning.Message)
			if resp.Warning.RequiresConfirmation && !allowDirty {
				fmt.Println("Retry with --allow-dirty to force checkout")
			}
			return nil
		}
		if len(resp.ValidationErrors) > 0 {
			fmt.Println("Validation errors:")
			for _, e := range resp.ValidationErrors {
				fmt.Printf("  ! %s\n", e)
			}
			return nil
		}
		if resp.Error != "" {
			fmt.Printf("Error: %s\n", resp.Error)
			return nil
		}
	}

	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdBranchSwitch(args []string) error {
	var name string
	var allowDirty bool
	var trackRemote bool

	for _, arg := range args {
		switch {
		case arg == "--allow-dirty":
			allowDirty = true
		case arg == "--track-remote":
			trackRemote = true
		case !strings.HasPrefix(arg, "-") && name == "":
			name = arg
		}
	}

	if name == "" {
		return fmt.Errorf("usage: branch-switch NAME [--allow-dirty] [--track-remote]")
	}

	req := branchSwitchRequest{
		Name:        name,
		AllowDirty:  allowDirty,
		TrackRemote: trackRemote,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/branch/switch"), nil, req)
	if err != nil {
		return err
	}

	var resp branchSwitchResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil {
		if resp.Success {
			fmt.Printf("Switched to: %s\n", name)
			return nil
		}
		if resp.Warning != nil {
			fmt.Printf("Warning: %s\n", resp.Warning.Message)
			if resp.Warning.RequiresTracking && !trackRemote {
				fmt.Println("Retry with --track-remote to track and switch")
			}
			if resp.Warning.RequiresConfirmation && !allowDirty {
				fmt.Println("Retry with --allow-dirty to force switch")
			}
			return nil
		}
		if resp.Error != "" {
			fmt.Printf("Error: %s\n", resp.Error)
			return nil
		}
	}

	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdBranchPublish(args []string) error {
	var remote string
	var branch string
	var fetch bool

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--remote="):
			remote = strings.TrimPrefix(arg, "--remote=")
		case strings.HasPrefix(arg, "--branch="):
			branch = strings.TrimPrefix(arg, "--branch=")
		case arg == "--fetch":
			fetch = true
		}
	}

	req := branchPublishRequest{
		Remote: remote,
		Branch: branch,
		Fetch:  fetch,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/branch/publish"), nil, req)
	if err != nil {
		return err
	}

	var resp branchPublishResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil {
		if resp.Success {
			fmt.Printf("Published: %s to %s\n", resp.Branch, resp.Remote)
			return nil
		}
		if resp.Warning != nil {
			fmt.Printf("Warning: %s\n", resp.Warning.Message)
			if resp.Warning.RequiresFetch && !fetch {
				fmt.Println("Retry with --fetch to refresh remote status")
			}
			return nil
		}
		if resp.Error != "" {
			fmt.Printf("Error: %s\n", resp.Error)
			return nil
		}
	}

	cliutil.PrintJSON(body)
	return nil
}

// [REQ:GCT-OT-P0-007] Audit log query

type auditEntry struct {
	ID            int64    `json:"id"`
	Operation     string   `json:"operation"`
	RepoDir       string   `json:"repo_dir"`
	Branch        string   `json:"branch,omitempty"`
	Paths         []string `json:"paths,omitempty"`
	CommitHash    string   `json:"commit_hash,omitempty"`
	CommitMessage string   `json:"commit_message,omitempty"`
	Success       bool     `json:"success"`
	Error         string   `json:"error,omitempty"`
	Timestamp     string   `json:"timestamp"`
}

type auditResponse struct {
	Entries []auditEntry `json:"entries"`
	Total   int          `json:"total"`
}

func (a *App) cmdAudit(args []string) error {
	var operation string
	var limit string

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--operation="):
			operation = strings.TrimPrefix(arg, "--operation=")
		case strings.HasPrefix(arg, "--limit="):
			limit = strings.TrimPrefix(arg, "--limit=")
		}
	}

	query := url.Values{}
	if operation != "" {
		query.Set("operation", operation)
	}
	if limit != "" {
		query.Set("limit", limit)
	}

	body, err := a.core.APIClient.Get(a.apiPath("/audit"), query)
	if err != nil {
		return err
	}

	var resp auditResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil && resp.Entries != nil {
		if len(resp.Entries) == 0 {
			fmt.Println("No audit entries found")
			return nil
		}

		fmt.Printf("Audit Log (%d of %d entries)\n", len(resp.Entries), resp.Total)
		fmt.Println(strings.Repeat("-", 60))

		for _, e := range resp.Entries {
			status := "OK"
			if !e.Success {
				status = "FAIL"
			}
			fmt.Printf("[%s] %s %s\n", e.Operation, status, e.Timestamp)

			if e.CommitHash != "" {
				fmt.Printf("  Commit: %s\n", e.CommitHash)
			}
			if e.CommitMessage != "" {
				msg := e.CommitMessage
				if len(msg) > 50 {
					msg = msg[:47] + "..."
				}
				fmt.Printf("  Message: %s\n", msg)
			}
			if len(e.Paths) > 0 {
				fmt.Printf("  Paths: %s\n", strings.Join(e.Paths, ", "))
			}
			if e.Error != "" {
				fmt.Printf("  Error: %s\n", e.Error)
			}
		}

		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}
