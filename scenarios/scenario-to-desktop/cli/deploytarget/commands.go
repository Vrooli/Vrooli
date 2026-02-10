// Package deploytarget provides CLI commands for managing deploy targets.
package deploytarget

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strings"

	"scenario-to-desktop/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

// Commands provides deploy target CLI commands.
type Commands struct {
	api *cliutil.APIClient
}

type doctorReport struct {
	Ready         bool   `json:"ready"`
	Name          string `json:"name"`
	ScenarioName  string `json:"scenario_name"`
	RemoteProfile string `json:"remote_profile"`
	Checks        []struct {
		Name     string `json:"name"`
		Required bool   `json:"required"`
		Passed   bool   `json:"passed"`
		Blocked  bool   `json:"blocked"`
		Detail   string `json:"detail"`
	} `json:"checks"`
	NextSteps []string `json:"next_steps"`
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
	names := make([]string, 0, len(resp.Targets))
	for name := range resp.Targets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t := resp.Targets[name]
		label := t.Label
		if label == "" {
			label = name
		}
		fmt.Printf("  %-16s label=%q scenario=%s profile=%s\n", name, label, t.ScenarioName, t.RemoteProfile)
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
	requireServiceAuth := fs.Bool("require-service-auth", false, "Also verify LPBS service auth is enabled and LPBS_SERVICE_SECRET is set")
	jsonOutput := cliutil.JSONFlag(fs)
	reordered := reorderArgs(args)
	if err := fs.Parse(reordered); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: deploy-target test <name>")
	}
	name := fs.Args()[0]

	req := map[string]bool{
		"require_service_auth": *requireServiceAuth,
	}
	body, err := c.api.Request("POST", c.apiPath("/deploy-targets/"+name+"/test"), nil, req)
	if err != nil {
		if *requireServiceAuth && isServiceAuthReadinessError(err) {
			return fmt.Errorf(
				"%v\n\nNext steps:\n%s",
				err,
				buildServiceAuthNextSteps(err, name),
			)
		}
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if *requireServiceAuth {
		fmt.Printf("Deploy target %q: remote profile session is active and service auth is ready\n", name)
	} else {
		fmt.Printf("Deploy target %q: remote profile session is active\n", name)
	}
	return nil
}

// Doctor runs an end-to-end deploy-target readiness diagnosis with triage output.
func (c *Commands) Doctor(args []string) error {
	fs := flag.NewFlagSet("deploy-target-doctor", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	reordered := reorderArgs(args)
	if err := fs.Parse(reordered); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: deploy-target doctor <name>")
	}
	name := fs.Args()[0]

	body, err := c.api.Request("POST", c.apiPath("/deploy-targets/"+name+"/doctor"), nil, map[string]bool{})
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var report doctorReport
	if err := json.Unmarshal(body, &report); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	status := "NOT READY"
	if report.Ready {
		status = "READY"
	}
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Target: %s (scenario=%s profile=%s)\n", report.Name, report.ScenarioName, report.RemoteProfile)
	fmt.Println("Triage:")
	for _, check := range report.Checks {
		state := "PASS"
		if check.Blocked {
			state = "BLOCKED"
		} else if !check.Passed {
			state = "FAIL"
		}
		fmt.Printf("  [%s] %s: %s\n", state, check.Name, check.Detail)
	}
	if len(report.NextSteps) > 0 {
		fmt.Println("Next steps:")
		for i, step := range report.NextSteps {
			fmt.Printf("  %d) %s\n", i+1, step)
		}
	}
	if !report.Ready {
		return fmt.Errorf("deploy target doctor checks failed")
	}
	return nil
}

func isServiceAuthReadinessError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "lpbs_service_secret is not set") ||
		strings.Contains(msg, "service auth") ||
		strings.Contains(msg, "service-auth")
}

func buildServiceAuthNextSteps(err error, name string) string {
	msg := ""
	if err != nil {
		msg = strings.ToLower(strings.TrimSpace(err.Error()))
	}

	if strings.Contains(msg, "scenario-to-desktop runtime") {
		return fmt.Sprintf(
			"  1) Verify LPBS source secret exists: scenario-to-cloud secrets get LPBS_SERVICE_SECRET --scenario landing-page-business-suite --targets scenario\n  2) Set scenario-to-desktop secret to the same value: scenario-to-cloud secrets set LPBS_SERVICE_SECRET --scenario scenario-to-desktop --value <same_secret_value> --targets scenario\n  3) Retry deploy-target auth gate: scenario-to-desktop deploy-target test %s --require-service-auth",
			name,
		)
	}

	return fmt.Sprintf(
		"  1) Set shared secret (portable): scenario-to-cloud secrets set LPBS_SERVICE_SECRET --scenario landing-page-business-suite --generate hex:64 --targets scenario,deployment --domain <domain> --restart\n  2) Verify LPBS runtime auth gate: landing-page-business-suite service-auth-status --require-enabled\n  3) Retry deploy-target auth gate: scenario-to-desktop deploy-target test %s --require-service-auth",
		name,
	)
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
