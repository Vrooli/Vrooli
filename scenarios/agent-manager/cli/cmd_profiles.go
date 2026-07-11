package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
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
	case "ensure":
		return a.profileEnsure(args[1:])
	case "reconcile-scenario":
		return a.profileReconcileScenario(args[1:])
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
  ensure            Resolve profile by key, creating with defaults if needed
  reconcile-scenario Reconcile profile files declared by a scenario

Options:
  --json            Output raw JSON
  --quiet           Output only IDs (for piping)

Examples:
  agent-manager profile list
  agent-manager profile get abc123
  agent-manager profile create --name "My Agent" --role-ref code.default
  agent-manager profile delete abc123
  agent-manager profile ensure --key "my-agent" --name "My Agent" --role-ref code.default
  agent-manager profile reconcile-scenario --scenario swarm-manager`)
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

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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

	fmt.Printf("%-36s  %-20s  %-16s  %-10s  %-12s\n", "ID", "NAME", "ROLE", "SANDBOX", "MANUAL_REVIEW")
	fmt.Printf("%-36s  %-20s  %-16s  %-10s  %-12s\n", strings.Repeat("-", 36), strings.Repeat("-", 20), strings.Repeat("-", 16), strings.Repeat("-", 10), strings.Repeat("-", 12))
	for _, p := range profiles {
		name := p.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		sandbox := profileSandboxMode(p)
		manualReview := "no"
		if p.SandboxConfig != nil && p.SandboxConfig.ManualReview {
			manualReview = "yes"
		}
		fmt.Printf("%-36s  %-20s  %-16s  %-10s  %-12s\n", p.Id, name, p.RoleRef, sandbox, manualReview)
	}

	return nil
}

// =============================================================================
// Profile Get
// =============================================================================

func (a *App) profileGet(args []string) error {
	fs := flag.NewFlagSet("profile get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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
	fmt.Printf("Role:           %s\n", profile.RoleRef)
	if profile.MaxTurns > 0 {
		fmt.Printf("Max Turns:      %d\n", profile.MaxTurns)
	}
	if timeout := formatDuration(profile.Timeout); timeout != "" {
		fmt.Printf("Timeout:        %s\n", timeout)
	}
	fmt.Printf("Sandbox Mode:      %s\n", profileSandboxMode(profile))
	if profile.SandboxConfig != nil && profile.SandboxConfig.ManualReview {
		fmt.Printf("Manual Review:     yes (apply gated by operator approval)\n")
	}
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
	roleRef := fs.String("role-ref", "", "Portable role from the active role-policy catalog (required)")
	maxTurns := fs.Int("max-turns", 0, "Maximum turns")
	timeout := fs.String("timeout", "", "Execution timeout (e.g., 30m)")
	sandboxMode := fs.String("sandbox-mode", "", "Sandbox mode (off/tracking/protected); empty preserves the SandboxConfig default")
	sandboxConfig := fs.String("sandbox-config", "", "Sandbox config JSON (proto JSON)")
	sandboxConfigFile := fs.String("sandbox-config-file", "", "Path to sandbox config JSON")
	sandboxRetentionMode := fs.String("sandbox-retention-mode", "", "Sandbox retention mode (keep_active, stop_on_terminal, delete_on_terminal)")
	sandboxRetentionTTL := fs.String("sandbox-retention-ttl", "", "Sandbox retention TTL (e.g., 2h, 30m)")
	skipPermissions := fs.Bool("skip-permissions", false, "Skip permission prompts")
	allowedTools := fs.String("allowed-tools", "", "Comma-separated list of allowed tools")
	deniedTools := fs.String("denied-tools", "", "Comma-separated list of denied tools")
	createdBy := fs.String("created-by", "", "Creator identifier")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}
	if strings.TrimSpace(*roleRef) == "" {
		return fmt.Errorf("--role-ref is required")
	}

	req := &domainpb.AgentProfile{
		Name:                 *name,
		ProfileKey:           strings.TrimSpace(*profileKey),
		Description:          *description,
		RoleRef:              strings.TrimSpace(*roleRef),
		MaxTurns:             int32(*maxTurns),
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
		cfg, err = applySandboxModeOverride(cfg, *sandboxMode)
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
	roleRef := fs.String("role-ref", "", "Portable role from the active role-policy catalog")
	maxTurns := fs.Int("max-turns", 0, "Maximum turns")
	timeout := fs.String("timeout", "", "Execution timeout")
	sandboxMode := fs.String("sandbox-mode", "", "Sandbox mode (off/tracking/protected); empty preserves the existing SandboxConfig")
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

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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
	if *roleRef != "" {
		req.RoleRef = strings.TrimSpace(*roleRef)
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
		cfg, err = applySandboxModeOverride(cfg, *sandboxMode)
		if err != nil {
			return err
		}
		if cfg != nil {
			req.SandboxConfig = cfg
		} else if mode := strings.ToLower(strings.TrimSpace(*sandboxMode)); mode != "" {
			// --sandbox-mode set but no SandboxConfig present yet: build one.
			cfg, err = applySandboxModeOverride(&domainpb.SandboxConfig{}, *sandboxMode)
			if err != nil {
				return err
			}
			req.SandboxConfig = cfg
		}
	}

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

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager profile delete <id>")
	}

	if !*force {
		fmt.Printf("Delete profile %s? [y/N]: ", id)
		var confirm string
		_, _ = fmt.Scanln(&confirm)
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

// =============================================================================
// Profile Ensure
// =============================================================================

func (a *App) profileEnsure(args []string) error {
	fs := flag.NewFlagSet("profile ensure", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	profileKey := fs.String("key", "", "Profile key (required)")
	updateExisting := fs.Bool("update", false, "Update existing profile with provided defaults")
	name := fs.String("name", "", "Default profile name")
	description := fs.String("description", "", "Default profile description")
	roleRef := fs.String("role-ref", "", "Default portable role from the active role-policy catalog (required)")
	maxTurns := fs.Int("max-turns", 0, "Default maximum turns")
	timeout := fs.String("timeout", "", "Default execution timeout (e.g., 30m)")
	sandboxMode := fs.String("sandbox-mode", "", "Default sandbox mode (off/tracking/protected); empty preserves the SandboxConfig default")
	sandboxConfig := fs.String("sandbox-config", "", "Default sandbox config JSON (proto JSON)")
	sandboxConfigFile := fs.String("sandbox-config-file", "", "Path to default sandbox config JSON")
	sandboxRetentionMode := fs.String("sandbox-retention-mode", "", "Default sandbox retention mode")
	sandboxRetentionTTL := fs.String("sandbox-retention-ttl", "", "Default sandbox retention TTL")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if *profileKey == "" {
		return fmt.Errorf("--key is required")
	}
	if strings.TrimSpace(*roleRef) == "" {
		return fmt.Errorf("--role-ref is required")
	}

	defaults := &domainpb.AgentProfile{
		ProfileKey:  *profileKey,
		Name:        *name,
		Description: *description,
		RoleRef:     strings.TrimSpace(*roleRef),
		MaxTurns:    int32(*maxTurns),
	}
	if defaults.Name == "" {
		defaults.Name = *profileKey
	}
	if *timeout != "" {
		parsed, err := time.ParseDuration(*timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		defaults.Timeout = durationpb.New(parsed)
	}
	if cfg, err := parseSandboxConfig(*sandboxConfig, *sandboxConfigFile); err != nil {
		return err
	} else {
		cfg, err = applySandboxRetention(cfg, *sandboxRetentionMode, *sandboxRetentionTTL)
		if err != nil {
			return err
		}
		cfg, err = applySandboxModeOverride(cfg, *sandboxMode)
		if err != nil {
			return err
		}
		if cfg != nil {
			defaults.SandboxConfig = cfg
		}
	}

	req := &apipb.EnsureProfileRequest{
		ProfileKey:     *profileKey,
		Defaults:       defaults,
		UpdateExisting: *updateExisting,
	}

	body, resp, err := a.services.Profiles.Ensure(req)
	if err != nil {
		return err
	}

	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	action := "Resolved"
	if resp.Created {
		action = "Created"
	} else if resp.Updated {
		action = "Updated"
	}
	fmt.Printf("%s profile: %s (%s)\n", action, resp.Profile.Name, resp.Profile.Id)
	return nil
}

func (a *App) profileReconcileScenario(args []string) error {
	fs := flag.NewFlagSet("profile reconcile-scenario", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	scenario := fs.String("scenario", "", "Scenario slug whose manifest declares profile sources")
	dryRun := fs.Bool("dry-run", false, "Validate and report actions without writing")

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" {
		return fmt.Errorf("usage: agent-manager profile reconcile-scenario --scenario <slug>")
	}

	body, resp, err := a.services.Profiles.ReconcileScenario(&apipb.ReconcileScenarioProfilesRequest{
		Scenario: strings.TrimSpace(*scenario),
		DryRun:   *dryRun,
	})
	if err != nil {
		return err
	}
	if *jsonOutput || resp == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Scenario:   %s\n", resp.Scenario)
	fmt.Printf("Created:    %d\n", resp.Created)
	fmt.Printf("Updated:    %d\n", resp.Updated)
	fmt.Printf("Unchanged:  %d\n", resp.Unchanged)
	fmt.Printf("Skipped:    %d\n", resp.Skipped)
	fmt.Printf("Conflicted: %d\n", resp.Conflicted)
	fmt.Printf("Failed:     %d\n", resp.Failed)
	for _, item := range resp.Results {
		fmt.Printf("- %s %s (%s)\n", item.ProfileKey, item.Status.String(), item.SourcePath)
		if item.Message != "" {
			fmt.Printf("  %s\n", item.Message)
		}
	}
	return nil
}
