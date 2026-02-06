// Package deploytarget provides CLI commands for managing deploy targets.
package deploytarget

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"scenario-to-desktop/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

// Commands provides deploy target CLI commands.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new deploy target Commands instance.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

func (c *Commands) apiPath(path string) string {
	return cmdutil.APIPath(path)
}

// List shows all saved deploy targets.
func (c *Commands) List(args []string) error {
	fs := flag.NewFlagSet("deploy-target-list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	body, err := c.api.Get(c.apiPath("/deploy-targets"), nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Targets map[string]struct {
			Label         string `json:"label"`
			ScenarioName  string `json:"scenario_name"`
			RemoteProfile string `json:"remote_profile"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(resp.Targets) == 0 {
		fmt.Println("No deploy targets configured")
		fmt.Println("Add one: deploy-target add <name> --scenario <s> --profile <p>")
		return nil
	}

	fmt.Println("Deploy targets:")
	for name, t := range resp.Targets {
		label := t.Label
		if label == "" {
			label = name
		}
		fmt.Printf("  %-20s scenario=%s profile=%s\n", label, t.ScenarioName, t.RemoteProfile)
	}
	return nil
}

// Add creates or updates a deploy target.
func (c *Commands) Add(args []string) error {
	fs := flag.NewFlagSet("deploy-target-add", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "LPBS scenario name (required)")
	profile := fs.String("profile", "", "Remote profile tag (required)")
	label := fs.String("label", "", "Human-readable label")
	jsonOutput := cliutil.JSONFlag(fs)

	// Reorder args so flags come before positional arguments
	reordered := reorderArgs(args)
	if err := fs.Parse(reordered); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: deploy-target add <name> --scenario <s> --profile <p> [--label <l>]")
	}
	name := fs.Args()[0]

	if *scenario == "" || *profile == "" {
		return fmt.Errorf("--scenario and --profile are required")
	}

	req := map[string]interface{}{
		"scenario_name":  *scenario,
		"remote_profile": *profile,
	}
	if *label != "" {
		req["label"] = *label
	}

	body, err := c.api.Request("PUT", c.apiPath("/deploy-targets/"+name), nil, req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Deploy target %q saved\n", name)
	return nil
}

// Remove deletes a deploy target.
func (c *Commands) Remove(args []string) error {
	fs := flag.NewFlagSet("deploy-target-remove", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: deploy-target remove <name>")
	}
	name := fs.Args()[0]

	body, err := c.api.Request("DELETE", c.apiPath("/deploy-targets/"+name), nil, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Deploy target %q removed\n", name)
	return nil
}

// Test validates a deploy target's remote profile session.
func (c *Commands) Test(args []string) error {
	fs := flag.NewFlagSet("deploy-target-test", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: deploy-target test <name>")
	}
	name := fs.Args()[0]

	body, err := c.api.Request("POST", c.apiPath("/deploy-targets/"+name+"/test"), nil, nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Deploy target %q: remote profile session is active\n", name)
	return nil
}

// reorderArgs moves flag arguments before positional arguments.
func reorderArgs(args []string) []string {
	var flags []string
	var positional []string

	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if next arg is a value for this flag
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
		i++
	}

	return append(flags, positional...)
}
