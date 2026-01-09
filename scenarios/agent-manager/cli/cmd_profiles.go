package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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
			fmt.Println(p.Id)
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
		runner := formatEnumValue(p.RunnerType, "RUNNER_TYPE_", "-")
		fmt.Printf("%-36s  %-20s  %-12s  %-8s  %-7s\n", p.Id, name, runner, sandbox, approval)
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

	fmt.Printf("ID:             %s\n", profile.Id)
	fmt.Printf("Name:           %s\n", profile.Name)
	if profile.Description != "" {
		fmt.Printf("Description:    %s\n", profile.Description)
	}
	if profile.ProfileKey != "" {
		fmt.Printf("Profile Key:    %s\n", profile.ProfileKey)
	}
	fmt.Printf("Runner Type:    %s\n", formatEnumValue(profile.RunnerType, "RUNNER_TYPE_", "-"))
	if profile.Model != "" {
		fmt.Printf("Model:          %s\n", profile.Model)
	}
	if profile.MaxTurns > 0 {
		fmt.Printf("Max Turns:      %d\n", profile.MaxTurns)
	}
	if timeout := formatDuration(profile.Timeout); timeout != "" {
		fmt.Printf("Timeout:        %s\n", timeout)
	}
	fmt.Printf("Requires Sandbox:  %v\n", profile.RequiresSandbox)
	fmt.Printf("Requires Approval: %v\n", profile.RequiresApproval)
	if profile.SandboxConfig != nil {
		fmt.Printf("Sandbox Config:   %s\n", marshalProtoJSON(profile.SandboxConfig))
	}
	if len(profile.AllowedTools) > 0 {
		fmt.Printf("Allowed Tools:  %s\n", strings.Join(profile.AllowedTools, ", "))
	}
	if len(profile.DeniedTools) > 0 {
		fmt.Printf("Denied Tools:   %s\n", strings.Join(profile.DeniedTools, ", "))
	}
	if created := formatTimestamp(profile.CreatedAt); created != "" {
		fmt.Printf("Created:        %s\n", created)
	}
	if updated := formatTimestamp(profile.UpdatedAt); updated != "" {
		fmt.Printf("Updated:        %s\n", updated)
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
	profileKey := fs.String("profile-key", "", "Stable profile key (defaults to name)")
	description := fs.String("description", "", "Profile description")
	runnerType := fs.String("runner-type", "claude-code", "Runner type (claude-code, codex, opencode)")
	model := fs.String("model", "", "Model to use")
	maxTurns := fs.Int("max-turns", 0, "Maximum turns")
	timeout := fs.String("timeout", "", "Execution timeout (e.g., 30m)")
	requiresSandbox := fs.Bool("sandbox", true, "Require sandbox execution")
	requiresApproval := fs.Bool("approval", true, "Require approval before applying changes")
	sandboxConfig := fs.String("sandbox-config", "", "Sandbox config JSON (proto JSON)")
	sandboxConfigFile := fs.String("sandbox-config-file", "", "Path to sandbox config JSON")
	sandboxRetentionMode := fs.String("sandbox-retention-mode", "", "Sandbox retention mode (keep_active, stop_on_terminal, delete_on_terminal)")
	sandboxRetentionTTL := fs.String("sandbox-retention-ttl", "", "Sandbox retention TTL (e.g., 2h, 30m)")
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

	req := &domainpb.AgentProfile{
		Name:                 *name,
		ProfileKey:           strings.TrimSpace(*profileKey),
		Description:          *description,
		RunnerType:           parseRunnerType(*runnerType),
		Model:                *model,
		MaxTurns:             int32(*maxTurns),
		RequiresSandbox:      *requiresSandbox,
		RequiresApproval:     *requiresApproval,
		SkipPermissionPrompt: *skipPermissions,
		CreatedBy:            *createdBy,
	}
	if req.ProfileKey == "" {
		req.ProfileKey = strings.TrimSpace(*name)
	}
	if *timeout != "" {
		parsed, err := time.ParseDuration(*timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		req.Timeout = durationpb.New(parsed)
	}
	if cfg, err := parseSandboxConfig(*sandboxConfig, *sandboxConfigFile); err != nil {
		return err
	} else {
		cfg, err = applySandboxRetention(cfg, *sandboxRetentionMode, *sandboxRetentionTTL)
		if err != nil {
			return err
		}
		if cfg != nil {
			req.SandboxConfig = cfg
		}
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

	fmt.Printf("Created profile: %s (%s)\n", profile.Name, profile.Id)
	return nil
}

// =============================================================================
// Profile Update
// =============================================================================

func (a *App) profileUpdate(args []string) error {
	fs := flag.NewFlagSet("profile update", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	name := fs.String("name", "", "Profile name")
	profileKey := fs.String("profile-key", "", "Stable profile key")
	description := fs.String("description", "", "Profile description")
	runnerType := fs.String("runner-type", "", "Runner type")
	model := fs.String("model", "", "Model to use")
	maxTurns := fs.Int("max-turns", 0, "Maximum turns")
	timeout := fs.String("timeout", "", "Execution timeout")
	requiresSandbox := fs.Bool("sandbox", true, "Require sandbox execution")
	requiresApproval := fs.Bool("approval", true, "Require approval")
	sandboxConfig := fs.String("sandbox-config", "", "Sandbox config JSON (proto JSON)")
	sandboxConfigFile := fs.String("sandbox-config-file", "", "Path to sandbox config JSON")
	sandboxRetentionMode := fs.String("sandbox-retention-mode", "", "Sandbox retention mode (keep_active, stop_on_terminal, delete_on_terminal)")
	sandboxRetentionTTL := fs.String("sandbox-retention-ttl", "", "Sandbox retention TTL (e.g., 2h, 30m)")

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
	req := proto.Clone(existing).(*domainpb.AgentProfile)

	// Apply updates
	if *name != "" {
		req.Name = *name
	}
	if *profileKey != "" {
		req.ProfileKey = strings.TrimSpace(*profileKey)
	}
	if *description != "" {
		req.Description = *description
	}
	if *runnerType != "" {
		req.RunnerType = parseRunnerType(*runnerType)
	}
	if *model != "" {
		req.Model = *model
	}
	if *maxTurns > 0 {
		req.MaxTurns = int32(*maxTurns)
	}
	if *timeout != "" {
		parsed, err := time.ParseDuration(*timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		req.Timeout = durationpb.New(parsed)
	}
	if cfg, err := parseSandboxConfig(*sandboxConfig, *sandboxConfigFile); err != nil {
		return err
	} else {
		cfg, err = applySandboxRetention(cfg, *sandboxRetentionMode, *sandboxRetentionTTL)
		if err != nil {
			return err
		}
		if cfg != nil {
			req.SandboxConfig = cfg
		}
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

	fmt.Printf("Updated profile: %s (%s)\n", profile.Name, profile.Id)
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
