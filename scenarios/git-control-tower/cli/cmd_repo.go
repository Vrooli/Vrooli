package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// [REQ:GCT-OT-P0-003] File diff endpoint

type diffResponse struct {
	RepoDir string `json:"repo_dir"`
	Path    string `json:"path"`
	Staged  bool   `json:"staged"`
	HasDiff bool   `json:"has_diff"`
	Stats   struct {
		Additions   int     `json:"additions"`
		Deletions   int     `json:"deletions"`
		Files       int     `json:"files"`
		NetLines    int     `json:"net_lines"`
		HunkCount   int     `json:"hunk_count"`
		LargestHunk int     `json:"largest_hunk"`
		Density     float64 `json:"density"`
	} `json:"stats"`
	Raw string `json:"raw"`
}

type diffFlags struct {
	path   string
	staged bool
}

func parseDiffFlags(args []string) diffFlags {
	var f diffFlags
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--path="):
			f.path = strings.TrimPrefix(arg, "--path=")
		case arg == "--staged":
			f.staged = true
		}
	}
	return f
}

func formatDiffOutput(parsed *diffResponse) {
	fmt.Printf("Diff for: %s\n", parsed.Path)
	if parsed.Staged {
		fmt.Println("(staged changes)")
	}
	statLine := fmt.Sprintf("Stats: +%d -%d (net %+d)",
		parsed.Stats.Additions, parsed.Stats.Deletions, parsed.Stats.NetLines)
	if parsed.Stats.HunkCount > 0 {
		statLine += fmt.Sprintf(" | %d hunks, largest: %d lines",
			parsed.Stats.HunkCount, parsed.Stats.LargestHunk)
	}
	fmt.Println(statLine)
	if parsed.Raw != "" {
		fmt.Println("---")
		fmt.Println(parsed.Raw)
	}
}

func (a *App) cmdDiff(args []string) error {
	f := parseDiffFlags(args)

	query := url.Values{}
	if f.path != "" {
		query.Set("path", f.path)
	}
	if f.staged {
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
		formatDiffOutput(&parsed)
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

type stageFlags struct {
	scope string
	paths []string
}

func parseStageFlags(args []string) stageFlags {
	var f stageFlags
	for _, arg := range args {
		if strings.HasPrefix(arg, "--scope=") {
			f.scope = strings.TrimPrefix(arg, "--scope=")
		} else if !strings.HasPrefix(arg, "-") {
			f.paths = append(f.paths, arg)
		}
	}
	return f
}

func printStageResult(parsed *stageResponse) {
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
}

func printUnstageResult(parsed *stageResponse) {
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
}

func (a *App) cmdStage(args []string) error {
	f := parseStageFlags(args)

	if len(f.paths) == 0 && f.scope == "" {
		return fmt.Errorf("usage: stage FILE... or --scope=scenario:name")
	}

	req := stageRequest{Paths: f.paths, Scope: f.scope}
	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/stage"), nil, req)
	if err != nil {
		return err
	}

	var parsed stageResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil {
		printStageResult(&parsed)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

func (a *App) cmdUnstage(args []string) error {
	f := parseStageFlags(args)

	if len(f.paths) == 0 && f.scope == "" {
		return fmt.Errorf("usage: unstage FILE... or --scope=scenario:name")
	}

	req := stageRequest{Paths: f.paths, Scope: f.scope}
	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/unstage"), nil, req)
	if err != nil {
		return err
	}

	var parsed stageResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil {
		printUnstageResult(&parsed)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// [REQ:GCT-OT-P0-005] Commit composition API

type commitRequest struct {
	Message              string `json:"message"`
	ValidateConventional bool   `json:"validate_conventional,omitempty"`
	Amend                bool   `json:"amend,omitempty"`
}

type commitResponse struct {
	Success          bool     `json:"success"`
	Hash             string   `json:"hash,omitempty"`
	Message          string   `json:"message,omitempty"`
	Amended          bool     `json:"amended,omitempty"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
	Error            string   `json:"error,omitempty"`
}

type commitFlags struct {
	message      string
	conventional bool
	amend        bool
}

func parseCommitFlags(args []string) commitFlags {
	var f commitFlags
	for i, arg := range args {
		switch {
		case arg == "-m" && i+1 < len(args):
			f.message = args[i+1]
		case strings.HasPrefix(arg, "-m="):
			f.message = strings.TrimPrefix(arg, "-m=")
		case strings.HasPrefix(arg, "--message="):
			f.message = strings.TrimPrefix(arg, "--message=")
		case arg == "--conventional":
			f.conventional = true
		case arg == "--amend":
			f.amend = true
		}
	}
	return f
}

func printCommitResult(parsed *commitResponse) {
	if parsed.Success {
		action := "Committed"
		if parsed.Amended {
			action = "Amended"
		}
		fmt.Printf("%s: %s\n", action, parsed.Hash)
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
}

func (a *App) cmdCommit(args []string) error {
	f := parseCommitFlags(args)

	if f.message == "" && !f.amend {
		return fmt.Errorf("usage: commit [-m MESSAGE] [--conventional] [--amend]")
	}

	req := commitRequest{
		Message:              f.message,
		ValidateConventional: f.conventional,
		Amend:                f.amend,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/commit"), nil, req)
	if err != nil {
		return err
	}

	var parsed commitResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil {
		printCommitResult(&parsed)
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

type syncStatusFlags struct {
	fetch  bool
	remote string
}

func parseSyncStatusFlags(args []string) syncStatusFlags {
	var f syncStatusFlags
	for _, arg := range args {
		switch {
		case arg == "--fetch":
			f.fetch = true
		case strings.HasPrefix(arg, "--remote="):
			f.remote = strings.TrimPrefix(arg, "--remote=")
		}
	}
	return f
}

func (a *App) cmdSyncStatus(args []string) error {
	f := parseSyncStatusFlags(args)

	query := url.Values{}
	if f.fetch {
		query.Set("fetch", "true")
	}
	if f.remote != "" {
		query.Set("remote", f.remote)
	}

	body, err := a.core.APIClient.Get(a.apiPath("/repo/sync-status"), query)
	if err != nil {
		return err
	}

	var resp syncStatusResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil && resp.Branch != "" {
		formatSyncBranchInfo(&resp)
		formatSyncActions(&resp)
		formatSyncWarnings(&resp)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}
