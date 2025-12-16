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
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, repo, config}
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
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]struct {
		Connected bool   `json:"connected"`
		Status    string `json:"status"`
	} `json:"dependencies"`
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
