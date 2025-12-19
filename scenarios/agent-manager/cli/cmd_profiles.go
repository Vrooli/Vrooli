package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Profile Command Dispatcher
// =============================================================================

func (a *App) cmdProfile(args []string) error {
	if len(args) == 0 {
		return a.profileHelp()
	}

	switch args[0] {
	case "list":
		return a.profileList(args[1:])
	case "get":
		return a.profileGet(args[1:])
	case "create":
		return a.profileCreate(args[1:])
	case "update":
		return a.profileUpdate(args[1:])
	case "delete":
		return a.profileDelete(args[1:])
	case "help", "-h", "--help":
		return a.profileHelp()
	default:
		return fmt.Errorf("unknown profile subcommand: %s\n\nRun 'agent-manager profile help' for usage", args[0])
	}
}

func (a *App) profileHelp() error {
	fmt.Println(`Usage: agent-manager profile <subcommand> [options]

Subcommands:
  list              List all agent profiles
  get <id>          Get profile details
  create            Create a new profile
  update <id>       Update an existing profile
  delete <id>       Delete a profile

Options:
  --json            Output raw JSON
  --quiet           Output only IDs (for piping)

Examples:
  agent-manager profile list
  agent-manager profile get abc123
  agent-manager profile create --name "My Agent" --runner-type claude-code
  agent-manager profile delete abc123`)
	return nil
}

// =============================================================================
// Profile List
// =============================================================================

func (a *App) profileList(args []string) error {
	fs := flag.NewFlagSet("profile list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	quiet := fs.Bool("quiet", false, "Output only IDs")
	limit := fs.Int("limit", 0, "Maximum number of profiles to return")
	offset := fs.Int("offset", 0, "Number of profiles to skip")

	if err := fs.Parse(args); err != nil {
		return err
	}

	body, profiles, err := a.services.Profiles.List(*limit, *offset)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if profiles == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if *quiet {
		for _, p := range profiles {
			fmt.Println(p.ID)
		}
		return nil
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles found")
		return nil
	}

	fmt.Printf("%-36s  %-20s  %-12s  %-8s  %-7s\n", "ID", "NAME", "RUNNER", "SANDBOX", "APPROVE")
	fmt.Printf("%-36s  %-20s  %-12s  %-8s  %-7s\n", strings.Repeat("-", 36), strings.Repeat("-", 20), strings.Repeat("-", 12), strings.Repeat("-", 8), strings.Repeat("-", 7))
	for _, p := range profiles {
		name := p.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		sandbox := "no"
		if p.RequiresSandbox {
			sandbox = "yes"
		}
		approval := "no"
		if p.RequiresApproval {
			approval = "yes"
		}
		fmt.Printf("%-36s  %-20s  %-12s  %-8s  %-7s\n", p.ID, name, p.RunnerType, sandbox, approval)
	}

	return nil
}

// =============================================================================
// Profile Get
// =============================================================================

func (a *App) profileGet(args []string) error {
	fs := flag.NewFlagSet("profile get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return fmt.Errorf("usage: agent-manager profile get <id>")
	}

	id := remaining[0]
	body, profile, err := a.services.Profiles.Get(id)
	if err != nil {
		return err
	}

	if *jsonOutput || profile == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("ID:             %s\n", profile.ID)
	fmt.Printf("Name:           %s\n", profile.Name)
	if profile.Description != "" {
		fmt.Printf("Description:    %s\n", profile.Description)
	}
	fmt.Printf("Runner Type:    %s\n", profile.RunnerType)
	if profile.Model != "" {
		fmt.Printf("Model:          %s\n", profile.Model)
	}
	if profile.MaxTurns > 0 {
		fmt.Printf("Max Turns:      %d\n", profile.MaxTurns)
	}
	if profile.Timeout != "" {
		fmt.Printf("Timeout:        %s\n", profile.Timeout)
	}
	fmt.Printf("Requires Sandbox:  %v\n", profile.RequiresSandbox)
	fmt.Printf("Requires Approval: %v\n", profile.RequiresApproval)
	if len(profile.AllowedTools) > 0 {
		fmt.Printf("Allowed Tools:  %s\n", strings.Join(profile.AllowedTools, ", "))
	}
	if len(profile.DeniedTools) > 0 {
		fmt.Printf("Denied Tools:   %s\n", strings.Join(profile.DeniedTools, ", "))
	}
	if profile.CreatedAt != "" {
		fmt.Printf("Created:        %s\n", profile.CreatedAt)
	}
	if profile.UpdatedAt != "" {
		fmt.Printf("Updated:        %s\n", profile.UpdatedAt)
	}

	return nil
}

// =============================================================================
// Profile Create
// =============================================================================

func (a *App) profileCreate(args []string) error {
	fs := flag.NewFlagSet("profile create", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	name := fs.String("name", "", "Profile name (required)")
	description := fs.String("description", "", "Profile description")
	runnerType := fs.String("runner-type", "claude-code", "Runner type (claude-code, codex, opencode)")
	model := fs.String("model", "", "Model to use")
	maxTurns := fs.Int("max-turns", 0, "Maximum turns")
	timeout := fs.String("timeout", "", "Execution timeout (e.g., 30m)")
	requiresSandbox := fs.Bool("sandbox", true, "Require sandbox execution")
	requiresApproval := fs.Bool("approval", true, "Require approval before applying changes")
	skipPermissions := fs.Bool("skip-permissions", false, "Skip permission prompts")
	allowedTools := fs.String("allowed-tools", "", "Comma-separated list of allowed tools")
	deniedTools := fs.String("denied-tools", "", "Comma-separated list of denied tools")
	createdBy := fs.String("created-by", "", "Creator identifier")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	req := CreateProfileRequest{
		Name:                 *name,
		Description:          *description,
		RunnerType:           *runnerType,
		Model:                *model,
		MaxTurns:             *maxTurns,
		Timeout:              *timeout,
		RequiresSandbox:      *requiresSandbox,
		RequiresApproval:     *requiresApproval,
		SkipPermissionPrompt: *skipPermissions,
		CreatedBy:            *createdBy,
	}

	if *allowedTools != "" {
		req.AllowedTools = strings.Split(*allowedTools, ",")
	}
	if *deniedTools != "" {
		req.DeniedTools = strings.Split(*deniedTools, ",")
	}

	body, profile, err := a.services.Profiles.Create(req)
	if err != nil {
		return err
	}

	if *jsonOutput || profile == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created profile: %s (%s)\n", profile.Name, profile.ID)
	return nil
}

// =============================================================================
// Profile Update
// =============================================================================

func (a *App) profileUpdate(args []string) error {
	fs := flag.NewFlagSet("profile update", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	name := fs.String("name", "", "Profile name")
	description := fs.String("description", "", "Profile description")
	runnerType := fs.String("runner-type", "", "Runner type")
	model := fs.String("model", "", "Model to use")
	maxTurns := fs.Int("max-turns", 0, "Maximum turns")
	timeout := fs.String("timeout", "", "Execution timeout")
	requiresSandbox := fs.Bool("sandbox", true, "Require sandbox execution")
	requiresApproval := fs.Bool("approval", true, "Require approval")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager profile update <id> [options]")
	}

	// First get the existing profile
	_, existing, err := a.services.Profiles.Get(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("profile not found: %s", id)
	}

	// Build update request, preserving existing values
	req := CreateProfileRequest{
		Name:             existing.Name,
		Description:      existing.Description,
		RunnerType:       existing.RunnerType,
		Model:            existing.Model,
		MaxTurns:         existing.MaxTurns,
		Timeout:          existing.Timeout,
		RequiresSandbox:  existing.RequiresSandbox,
		RequiresApproval: existing.RequiresApproval,
		AllowedTools:     existing.AllowedTools,
		DeniedTools:      existing.DeniedTools,
	}

	// Apply updates
	if *name != "" {
		req.Name = *name
	}
	if *description != "" {
		req.Description = *description
	}
	if *runnerType != "" {
		req.RunnerType = *runnerType
	}
	if *model != "" {
		req.Model = *model
	}
	if *maxTurns > 0 {
		req.MaxTurns = *maxTurns
	}
	if *timeout != "" {
		req.Timeout = *timeout
	}
	req.RequiresSandbox = *requiresSandbox
	req.RequiresApproval = *requiresApproval

	body, profile, err := a.services.Profiles.Update(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput || profile == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Updated profile: %s (%s)\n", profile.Name, profile.ID)
	return nil
}

// =============================================================================
// Profile Delete
// =============================================================================

func (a *App) profileDelete(args []string) error {
	fs := flag.NewFlagSet("profile delete", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager profile delete <id>")
	}

	if !*force {
		fmt.Printf("Delete profile %s? [y/N]: ", id)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := a.services.Profiles.Delete(id); err != nil {
		return err
	}

	fmt.Printf("Deleted profile: %s\n", id)
	return nil
}
