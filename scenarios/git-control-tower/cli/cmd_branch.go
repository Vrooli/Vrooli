package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

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

type branchCreateFlags struct {
	name       string
	from       string
	checkout   bool
	allowDirty bool
}

func parseBranchCreateFlags(args []string) branchCreateFlags {
	f := branchCreateFlags{checkout: true}
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--from="):
			f.from = strings.TrimPrefix(arg, "--from=")
		case arg == "--no-checkout":
			f.checkout = false
		case arg == "--checkout":
			f.checkout = true
		case arg == "--allow-dirty":
			f.allowDirty = true
		case !strings.HasPrefix(arg, "-") && f.name == "":
			f.name = arg
		}
	}
	return f
}

func printBranchCreateResult(resp *branchCreateResponse, name string, allowDirty bool) {
	if resp.Success {
		fmt.Printf("Created branch: %s\n", name)
		return
	}
	if resp.Warning != nil {
		fmt.Printf("Warning: %s\n", resp.Warning.Message)
		if resp.Warning.RequiresConfirmation && !allowDirty {
			fmt.Println("Retry with --allow-dirty to force checkout")
		}
		return
	}
	if len(resp.ValidationErrors) > 0 {
		fmt.Println("Validation errors:")
		for _, e := range resp.ValidationErrors {
			fmt.Printf("  ! %s\n", e)
		}
		return
	}
	if resp.Error != "" {
		fmt.Printf("Error: %s\n", resp.Error)
	}
}

func (a *App) cmdBranchCreate(args []string) error {
	f := parseBranchCreateFlags(args)

	if f.name == "" {
		return fmt.Errorf("usage: branch-create NAME [--from=BASE] [--no-checkout] [--allow-dirty]")
	}

	req := branchCreateRequest{
		Name:       f.name,
		From:       f.from,
		Checkout:   f.checkout,
		AllowDirty: f.allowDirty,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/branch/create"), nil, req)
	if err != nil {
		return err
	}

	var resp branchCreateResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil {
		printBranchCreateResult(&resp, f.name, f.allowDirty)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

type branchSwitchFlags struct {
	name        string
	allowDirty  bool
	trackRemote bool
}

func parseBranchSwitchFlags(args []string) branchSwitchFlags {
	var f branchSwitchFlags
	for _, arg := range args {
		switch {
		case arg == "--allow-dirty":
			f.allowDirty = true
		case arg == "--track-remote":
			f.trackRemote = true
		case !strings.HasPrefix(arg, "-") && f.name == "":
			f.name = arg
		}
	}
	return f
}

func printBranchSwitchResult(resp *branchSwitchResponse, f branchSwitchFlags) {
	if resp.Success {
		fmt.Printf("Switched to: %s\n", f.name)
		return
	}
	if resp.Warning != nil {
		fmt.Printf("Warning: %s\n", resp.Warning.Message)
		if resp.Warning.RequiresTracking && !f.trackRemote {
			fmt.Println("Retry with --track-remote to track and switch")
		}
		if resp.Warning.RequiresConfirmation && !f.allowDirty {
			fmt.Println("Retry with --allow-dirty to force switch")
		}
		return
	}
	if resp.Error != "" {
		fmt.Printf("Error: %s\n", resp.Error)
	}
}

func (a *App) cmdBranchSwitch(args []string) error {
	f := parseBranchSwitchFlags(args)

	if f.name == "" {
		return fmt.Errorf("usage: branch-switch NAME [--allow-dirty] [--track-remote]")
	}

	req := branchSwitchRequest{
		Name:        f.name,
		AllowDirty:  f.allowDirty,
		TrackRemote: f.trackRemote,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/branch/switch"), nil, req)
	if err != nil {
		return err
	}

	var resp branchSwitchResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil {
		printBranchSwitchResult(&resp, f)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

type branchPublishFlags struct {
	remote string
	branch string
	fetch  bool
}

func parseBranchPublishFlags(args []string) branchPublishFlags {
	var f branchPublishFlags
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--remote="):
			f.remote = strings.TrimPrefix(arg, "--remote=")
		case strings.HasPrefix(arg, "--branch="):
			f.branch = strings.TrimPrefix(arg, "--branch=")
		case arg == "--fetch":
			f.fetch = true
		}
	}
	return f
}

func printBranchPublishResult(resp *branchPublishResponse, fetch bool) {
	if resp.Success {
		fmt.Printf("Published: %s to %s\n", resp.Branch, resp.Remote)
		return
	}
	if resp.Warning != nil {
		fmt.Printf("Warning: %s\n", resp.Warning.Message)
		if resp.Warning.RequiresFetch && !fetch {
			fmt.Println("Retry with --fetch to refresh remote status")
		}
		return
	}
	if resp.Error != "" {
		fmt.Printf("Error: %s\n", resp.Error)
	}
}

func (a *App) cmdBranchPublish(args []string) error {
	f := parseBranchPublishFlags(args)

	req := branchPublishRequest{
		Remote: f.remote,
		Branch: f.branch,
		Fetch:  f.fetch,
	}

	body, err := a.core.APIClient.Request("POST", a.apiPath("/repo/branch/publish"), nil, req)
	if err != nil {
		return err
	}

	var resp branchPublishResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil {
		printBranchPublishResult(&resp, f.fetch)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}
