package branch

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type info struct {
	Name      string `json:"name"`
	Upstream  string `json:"upstream,omitempty"`
	OID       string `json:"oid,omitempty"`
	IsCurrent bool   `json:"is_current,omitempty"`
}

type listResponse struct {
	Current string `json:"current"`
	Locals  []info `json:"locals"`
	Remotes []info `json:"remotes"`
}

type warning struct {
	Message              string `json:"message"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
	RequiresTracking     bool   `json:"requires_tracking,omitempty"`
	RequiresFetch        bool   `json:"requires_fetch,omitempty"`
}

type createRequest struct {
	Name       string `json:"name"`
	From       string `json:"from,omitempty"`
	Checkout   bool   `json:"checkout,omitempty"`
	AllowDirty bool   `json:"allow_dirty,omitempty"`
}

type createResponse struct {
	Success          bool     `json:"success"`
	Branch           *info    `json:"branch,omitempty"`
	Warning          *warning `json:"warning,omitempty"`
	Error            string   `json:"error,omitempty"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
}

type switchRequest struct {
	Name        string `json:"name"`
	AllowDirty  bool   `json:"allow_dirty,omitempty"`
	TrackRemote bool   `json:"track_remote,omitempty"`
}

type switchResponse struct {
	Success bool     `json:"success"`
	Branch  *info    `json:"branch,omitempty"`
	Warning *warning `json:"warning,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type publishRequest struct {
	Remote string `json:"remote,omitempty"`
	Branch string `json:"branch,omitempty"`
	Fetch  bool   `json:"fetch,omitempty"`
}

type publishResponse struct {
	Success bool     `json:"success"`
	Remote  string   `json:"remote"`
	Branch  string   `json:"branch"`
	Warning *warning `json:"warning,omitempty"`
	Error   string   `json:"error,omitempty"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "branch",
		Description: "Manage repository branches",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", NeedsAPI: true, Description: "List branches", Run: func(args []string) error { return runList(core, args) }},
			{Name: "create", NeedsAPI: true, Description: "Create branch NAME [--from=BASE] [--no-checkout] [--allow-dirty]", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "switch", NeedsAPI: true, Description: "Switch branch NAME [--allow-dirty] [--track-remote]", Run: func(args []string) error { return runSwitch(core, args) }},
			{Name: "publish", NeedsAPI: true, Description: "Publish current branch ([--remote=NAME] [--branch=NAME] [--fetch])", Run: func(args []string) error { return runPublish(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, _ []string) error {
	body, err := core.Get("/repo/branches", nil)
	if err != nil {
		return err
	}
	var resp listResponse
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

type createFlags struct {
	name       string
	from       string
	checkout   bool
	allowDirty bool
}

func parseCreateFlags(args []string) createFlags {
	f := createFlags{checkout: true}
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

func printCreateResult(resp *createResponse, name string, allowDirty bool) {
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

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	f := parseCreateFlags(args)
	if f.name == "" {
		return fmt.Errorf("usage: branch create NAME [--from=BASE] [--no-checkout] [--allow-dirty]")
	}
	req := createRequest{Name: f.name, From: f.from, Checkout: f.checkout, AllowDirty: f.allowDirty}
	body, err := core.Request("POST", "/repo/branch/create", nil, req)
	if err != nil {
		return err
	}
	var resp createResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil {
		printCreateResult(&resp, f.name, f.allowDirty)
		return nil
	}
	cliutil.PrintJSON(body)
	return nil
}

type switchFlags struct {
	name        string
	allowDirty  bool
	trackRemote bool
}

func parseSwitchFlags(args []string) switchFlags {
	var f switchFlags
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

func printSwitchResult(resp *switchResponse, f switchFlags) {
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

func runSwitch(core *cliapp.ScenarioApp, args []string) error {
	f := parseSwitchFlags(args)
	if f.name == "" {
		return fmt.Errorf("usage: branch switch NAME [--allow-dirty] [--track-remote]")
	}
	req := switchRequest{Name: f.name, AllowDirty: f.allowDirty, TrackRemote: f.trackRemote}
	body, err := core.Request("POST", "/repo/branch/switch", nil, req)
	if err != nil {
		return err
	}
	var resp switchResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil {
		printSwitchResult(&resp, f)
		return nil
	}
	cliutil.PrintJSON(body)
	return nil
}

type publishFlags struct {
	remote string
	branch string
	fetch  bool
}

func parsePublishFlags(args []string) publishFlags {
	var f publishFlags
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

func printPublishResult(resp *publishResponse, fetch bool) {
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

func runPublish(core *cliapp.ScenarioApp, args []string) error {
	f := parsePublishFlags(args)
	req := publishRequest{Remote: f.remote, Branch: f.branch, Fetch: f.fetch}
	body, err := core.Request("POST", "/repo/branch/publish", nil, req)
	if err != nil {
		return err
	}
	var resp publishResponse
	if unmarshalErr := json.Unmarshal(body, &resp); unmarshalErr == nil {
		printPublishResult(&resp, f.fetch)
		return nil
	}
	cliutil.PrintJSON(body)
	return nil
}
