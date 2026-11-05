package profiles

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"vrooli-orchestrator/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// Register exposes the `profiles` subcommand group covering profile CRUD,
// activation/deactivation, and scenario-specific active-profile status
// (via /api/v1/status, distinct from cli-core's built-in `status` command
// which probes root /health).
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "profiles",
		Description: "Manage orchestrator startup profiles",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Aliases: []string{"ls"}, Description: "List all profiles", Run: func(args []string) error { return runList(core, args) }},
			{Name: "get", Aliases: []string{"show"}, Description: "Show profile details", Run: func(args []string) error { return runGet(core, args) }},
			{Name: "create", Description: "Create a new profile", Run: func(args []string) error { return runCreate(core, args) }},
			{Name: "update", Description: "Update an existing profile", Run: func(args []string) error { return runUpdate(core, args) }},
			{Name: "delete", Aliases: []string{"rm"}, Description: "Delete a profile", Run: func(args []string) error { return runDelete(core, args) }},
			{Name: "activate", Description: "Activate a profile (start its resources and scenarios)", Run: func(args []string) error { return runActivate(core, args) }},
			{Name: "deactivate", Description: "Deactivate the currently active profile", Run: func(args []string) error { return runDeactivate(core, args) }},
			{Name: "active", Aliases: []string{"current"}, Description: "Show the active profile and orchestrator status", Run: func(args []string) error { return runActive(core, args) }},
		},
	}
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles list")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/profiles", nil)
	if err != nil {
		return err
	}
	var resp support.ProfileListResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Profiles: %d", resp.Count)},
		ResultsHeading: "Profiles",
		Results:        profileRows(resp.Profiles),
		RetrievalHints: []string{
			fmt.Sprintf("%s profiles get <name>", support.CLIName),
			fmt.Sprintf("%s profiles activate <name>", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runGet(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles get")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: profiles get <profile-name>")
	}
	name := fs.Arg(0)

	body, err := core.Get("/profiles/"+name, nil)
	if err != nil {
		return err
	}
	var profile support.Profile
	if err := support.Decode(body, &profile); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Profile: %s (%s)", profile.Name, profile.Status)},
		ResultsHeading: "Details",
		Results:        profileDetailRows(profile),
		RetrievalHints: []string{
			fmt.Sprintf("%s profiles activate %s", support.CLIName, profile.Name),
			fmt.Sprintf("%s profiles update %s --display-name ...", support.CLIName, profile.Name),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles create")
	displayName := fs.String("display-name", "", "Human-readable profile name")
	description := fs.String("description", "", "Profile description")
	resources := fs.String("resources", "", "Comma-separated resource list")
	scenarios := fs.String("scenarios", "", "Comma-separated scenario list")
	autoBrowser := fs.String("auto-browser", "", "Comma-separated URL list")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full create payload (merged with flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	payload := map[string]interface{}{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("parse %s: %w", *bodyFile, err)
		}
	}

	if fs.NArg() >= 1 {
		payload["name"] = fs.Arg(0)
	}
	if _, ok := payload["name"].(string); !ok {
		return fmt.Errorf("usage: profiles create <name> [--display-name ...] [--resources r1,r2] [--scenarios s1,s2] [--auto-browser url1,url2] [--body-file PATH]")
	}

	if flagSet(fs, "display-name") {
		payload["display_name"] = *displayName
	} else if _, ok := payload["display_name"]; !ok {
		payload["display_name"] = payload["name"]
	}
	if flagSet(fs, "description") {
		payload["description"] = *description
	}
	if flagSet(fs, "resources") {
		payload["resources"] = support.ParseCSVList(*resources)
	} else if _, ok := payload["resources"]; !ok {
		payload["resources"] = []string{}
	}
	if flagSet(fs, "scenarios") {
		payload["scenarios"] = support.ParseCSVList(*scenarios)
	} else if _, ok := payload["scenarios"]; !ok {
		payload["scenarios"] = []string{}
	}
	if flagSet(fs, "auto-browser") {
		payload["auto_browser"] = support.ParseCSVList(*autoBrowser)
	} else if _, ok := payload["auto_browser"]; !ok {
		payload["auto_browser"] = []string{}
	}

	body, err := core.Request("POST", "/profiles", nil, payload)
	if err != nil {
		return err
	}
	var profile support.Profile
	if err := support.Decode(body, &profile); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Profile created: %s", profile.Name),
			fmt.Sprintf("Resources: %d configured", len(profile.Resources)),
			fmt.Sprintf("Scenarios: %d configured", len(profile.Scenarios)),
		},
		Changes: []string{fmt.Sprintf("Created profile %s (%s)", profile.Name, support.ShortID(profile.ID))},
		NextCommand: []string{
			fmt.Sprintf("%s profiles get %s", support.CLIName, profile.Name),
			fmt.Sprintf("%s profiles activate %s", support.CLIName, profile.Name),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runUpdate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles update")
	displayName := fs.String("display-name", "", "Human-readable profile name")
	description := fs.String("description", "", "Profile description")
	resources := fs.String("resources", "", "Comma-separated resource list")
	scenarios := fs.String("scenarios", "", "Comma-separated scenario list")
	autoBrowser := fs.String("auto-browser", "", "Comma-separated URL list")
	bodyFile := fs.String("body-file", "", "Path to a JSON file with the full update payload (merged with flags)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: profiles update <profile-name> [--display-name ...] [--resources r1,r2] [--scenarios s1,s2] [--auto-browser u1,u2] [--body-file PATH]")
	}
	name := fs.Arg(0)

	updates := map[string]interface{}{}
	if strings.TrimSpace(*bodyFile) != "" {
		raw, err := support.ReadJSONFile(*bodyFile, true)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &updates); err != nil {
			return fmt.Errorf("parse %s: %w", *bodyFile, err)
		}
	}

	if flagSet(fs, "display-name") {
		updates["display_name"] = *displayName
	}
	if flagSet(fs, "description") {
		updates["description"] = *description
	}
	if flagSet(fs, "resources") {
		updates["resources"] = support.ParseCSVList(*resources)
	}
	if flagSet(fs, "scenarios") {
		updates["scenarios"] = support.ParseCSVList(*scenarios)
	}
	if flagSet(fs, "auto-browser") {
		updates["auto_browser"] = support.ParseCSVList(*autoBrowser)
	}

	if len(updates) == 0 {
		return fmt.Errorf("no updates specified: use --display-name, --description, --resources, --scenarios, --auto-browser, or --body-file")
	}

	body, err := core.Request("PUT", "/profiles/"+name, nil, updates)
	if err != nil {
		return err
	}
	var profile support.Profile
	if err := support.Decode(body, &profile); err != nil {
		return err
	}

	keys := make([]string, 0, len(updates))
	for k := range updates {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	changes := make([]string, 0, len(keys))
	for _, k := range keys {
		changes = append(changes, fmt.Sprintf("Updated %s", k))
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Profile updated: %s", profile.Name)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s profiles get %s", support.CLIName, profile.Name),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDelete(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles delete")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: profiles delete <profile-name>")
	}
	name := fs.Arg(0)

	body, err := core.Request("DELETE", "/profiles/"+name, nil, nil)
	if err != nil {
		return err
	}
	message := support.EnvelopeMessage(body)
	if message == "" {
		message = fmt.Sprintf("Profile '%s' deleted", name)
	}

	report := cliapp.MutationReport{
		Result:  []string{message},
		Changes: []string{fmt.Sprintf("Deleted profile %s", name)},
		NextCommand: []string{
			fmt.Sprintf("%s profiles list", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runActivate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles activate")
	force := fs.Bool("force", false, "Force activation even if another profile is active")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: profiles activate <profile-name> [--force]")
	}
	name := fs.Arg(0)

	payload := map[string]interface{}{"force": *force}
	body, err := core.Request("POST", "/profiles/"+name+"/activate", nil, payload)
	if err != nil {
		return err
	}
	var result support.ActivationResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:  activationResultLines(result),
		Changes: activationChangeLines(result),
		NextCommand: []string{
			fmt.Sprintf("%s profiles active", support.CLIName),
			fmt.Sprintf("%s profiles deactivate", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDeactivate(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles deactivate")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Request("POST", "/profiles/current/deactivate", nil, map[string]interface{}{})
	if err != nil {
		return err
	}
	var result support.DeactivationResult
	if err := support.Decode(body, &result); err != nil {
		return err
	}

	summary := result.Message
	if summary == "" {
		if result.Success {
			summary = "Profile deactivated"
		} else {
			summary = "Deactivation completed with errors"
		}
	}

	changes := []string{}
	for name, status := range result.ResourcesStatus {
		changes = append(changes, fmt.Sprintf("Resource %s: %s", name, renderSubStatus(status)))
	}
	for name, status := range result.ScenariosStatus {
		changes = append(changes, fmt.Sprintf("Scenario %s: %s", name, renderSubStatus(status)))
	}
	sort.Strings(changes)
	if len(changes) == 0 {
		changes = []string{"No active resources or scenarios"}
	}

	report := cliapp.MutationReport{
		Result:  []string{summary},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("%s profiles list", support.CLIName),
			fmt.Sprintf("%s profiles active", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runActive(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("profiles active")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/status", nil)
	if err != nil {
		return err
	}
	var status support.StatusResponse
	if err := support.Decode(body, &status); err != nil {
		return err
	}

	summary := []string{
		fmt.Sprintf("Service: %s v%s", defaultStr(status.Service, "vrooli-orchestrator"), defaultStr(status.Version, "unknown")),
		fmt.Sprintf("Status: %s", defaultStr(status.Status, "unknown")),
	}
	if status.ActiveProfile != nil {
		summary = append(summary, fmt.Sprintf("Active profile: %s", status.ActiveProfile.Name))
	} else {
		summary = append(summary, "Active profile: (none)")
	}

	results := []string{
		fmt.Sprintf("Resource count: %d", status.ResourceCount),
		fmt.Sprintf("Scenario count: %d", status.ScenarioCount),
	}
	if status.ActiveProfile != nil {
		results = append(results, profileDetailRows(*status.ActiveProfile)...)
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Orchestrator status",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s profiles list", support.CLIName),
			fmt.Sprintf("%s profiles deactivate", support.CLIName),
		},
	}
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// flagSet reports whether the named flag was explicitly set on fs.
func flagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func profileRows(profiles []support.Profile) []string {
	if len(profiles) == 0 {
		return []string{"No profiles found"}
	}
	rows := make([]string, 0, len(profiles))
	for _, p := range profiles {
		display := p.DisplayName
		if display == "" {
			display = p.Name
		}
		status := p.Status
		if status == "" {
			status = "unknown"
		}
		rows = append(rows, fmt.Sprintf("%s | %s | status=%s | resources=%d | scenarios=%d",
			p.Name, display, status, len(p.Resources), len(p.Scenarios)))
	}
	return rows
}

func profileDetailRows(p support.Profile) []string {
	rows := []string{
		fmt.Sprintf("Name: %s", p.Name),
		fmt.Sprintf("Display name: %s", defaultStr(p.DisplayName, p.Name)),
	}
	if p.Description != "" {
		rows = append(rows, fmt.Sprintf("Description: %s", p.Description))
	}
	if p.ID != "" {
		rows = append(rows, fmt.Sprintf("ID: %s", p.ID))
	}
	rows = append(rows, fmt.Sprintf("Status: %s", defaultStr(p.Status, "unknown")))
	rows = append(rows, fmt.Sprintf("Resources (%d): %s", len(p.Resources), joinOrDash(p.Resources)))
	rows = append(rows, fmt.Sprintf("Scenarios (%d): %s", len(p.Scenarios), joinOrDash(p.Scenarios)))
	if len(p.AutoBrowser) > 0 {
		rows = append(rows, fmt.Sprintf("Auto-browser: %s", strings.Join(p.AutoBrowser, ", ")))
	}
	if p.IdleShutdown != nil {
		rows = append(rows, fmt.Sprintf("Idle shutdown: %d minutes", *p.IdleShutdown))
	}
	if p.CreatedAt != nil {
		rows = append(rows, fmt.Sprintf("Created: %s", support.FormatTimeValue(*p.CreatedAt)))
	}
	if p.UpdatedAt != nil {
		rows = append(rows, fmt.Sprintf("Updated: %s", support.FormatTimeValue(*p.UpdatedAt)))
	}
	return rows
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func defaultStr(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func activationResultLines(r support.ActivationResult) []string {
	lines := []string{}
	if r.Message != "" {
		lines = append(lines, r.Message)
	} else if r.ProfileName != "" {
		if r.Success {
			lines = append(lines, fmt.Sprintf("Profile '%s' activated", r.ProfileName))
		} else {
			lines = append(lines, fmt.Sprintf("Profile '%s' activation completed with errors", r.ProfileName))
		}
	}
	if r.Error != "" {
		lines = append(lines, fmt.Sprintf("Error: %s", r.Error))
	}
	lines = append(lines,
		fmt.Sprintf("Resources attempted: %d", len(r.ResourcesStatus)),
		fmt.Sprintf("Scenarios attempted: %d", len(r.ScenariosStatus)),
	)
	if len(r.BrowserActions) > 0 {
		lines = append(lines, fmt.Sprintf("Browser actions: %d", len(r.BrowserActions)))
	}
	return lines
}

func activationChangeLines(r support.ActivationResult) []string {
	changes := []string{}
	for name, status := range r.ResourcesStatus {
		changes = append(changes, fmt.Sprintf("Resource %s: %s", name, renderSubStatus(status)))
	}
	for name, status := range r.ScenariosStatus {
		changes = append(changes, fmt.Sprintf("Scenario %s: %s", name, renderSubStatus(status)))
	}
	for _, action := range r.BrowserActions {
		changes = append(changes, fmt.Sprintf("Browser: %s", action))
	}
	sort.Strings(changes)
	if len(changes) == 0 {
		if r.Success {
			changes = []string{"No resources or scenarios to start"}
		} else {
			changes = []string{"(no detail returned)"}
		}
	}
	return changes
}

func renderSubStatus(status interface{}) string {
	m, ok := status.(map[string]interface{})
	if !ok {
		return support.RenderValue(status)
	}
	success := "unknown"
	if v, ok := m["success"].(bool); ok {
		if v {
			success = "ok"
		} else {
			success = "failed"
		}
	}
	if errMsg, ok := m["error"].(string); ok && errMsg != "" {
		return fmt.Sprintf("%s (%s)", success, errMsg)
	}
	return success
}
