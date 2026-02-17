// Package steer provides CLI commands for auto-steer profile management.
package steer

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"ecosystem-manager/cli/internal/appctx"
)

// ProfileListResponse represents a list of auto-steer profiles.
type ProfileListResponse struct {
	Profiles []Profile `json:"profiles"`
	Count    int       `json:"count"`
}

// Profile represents an auto-steer profile.
type Profile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	TaskTypes   []string `json:"task_types,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Enabled     bool     `json:"enabled"`
	Phases      []any    `json:"phases,omitempty"`
}

// TemplateListResponse represents a list of auto-steer templates.
type TemplateListResponse struct {
	Templates []Template `json:"templates"`
	Count     int        `json:"count"`
}

// Template represents an auto-steer template.
type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Commands returns the steer command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Auto Steer",
		Commands: []cliapp.Command{
			{
				Name:        "steer",
				NeedsAPI:    true,
				Description: "Manage auto-steer profiles (profiles|templates|show)",
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
		},
	}
}

func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "help":
		return printUsage()
	case "profiles", "ls":
		return cmdProfiles(ctx, subArgs)
	case "templates":
		return cmdTemplates(ctx, subArgs)
	case "show", "get":
		return cmdShow(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: ecosystem-manager steer <subcommand> [args]

Subcommands:
  profiles, ls    List auto-steer profiles
  templates       List auto-steer templates
  show, get <id>  Show profile details`
}

func cmdProfiles(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("profiles", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp ProfileListResponse
	if err := ctx.Get("/auto-steer/profiles", &resp); err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Profiles) == 0 {
		fmt.Println("No auto-steer profiles found")
		return nil
	}

	fmt.Printf("Auto-Steer Profiles (%d):\n", resp.Count)
	for _, p := range resp.Profiles {
		enabled := "enabled"
		if !p.Enabled {
			enabled = "disabled"
		}
		desc := ""
		if p.Description != "" {
			desc = fmt.Sprintf(" - %s", p.Description)
		}
		fmt.Printf("  %s (%s)%s [%s]\n", p.Name, enabled, desc, p.ID)
	}
	return nil
}

func cmdTemplates(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("templates", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp TemplateListResponse
	if err := ctx.Get("/auto-steer/templates", &resp); err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	if len(resp.Templates) == 0 {
		fmt.Println("No auto-steer templates found")
		return nil
	}

	fmt.Printf("Auto-Steer Templates (%d):\n", resp.Count)
	for _, t := range resp.Templates {
		desc := ""
		if t.Description != "" {
			desc = fmt.Sprintf(" - %s", t.Description)
		}
		fmt.Printf("  %s%s [%s]\n", t.Name, desc, t.ID)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: steer show <id>")
	}
	profileID := fs.Arg(0)

	var profile Profile
	if err := ctx.Get(fmt.Sprintf("/auto-steer/profiles/%s", profileID), &profile); err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(profile)
	}

	fmt.Printf("Name: %s\n", profile.Name)
	fmt.Printf("ID: %s\n", profile.ID)
	if profile.Description != "" {
		fmt.Printf("Description: %s\n", profile.Description)
	}
	enabled := "yes"
	if !profile.Enabled {
		enabled = "no"
	}
	fmt.Printf("Enabled: %s\n", enabled)
	if len(profile.TaskTypes) > 0 {
		fmt.Printf("Task Types: %v\n", profile.TaskTypes)
	}
	if len(profile.Tags) > 0 {
		fmt.Printf("Tags: %v\n", profile.Tags)
	}
	if len(profile.Phases) > 0 {
		fmt.Printf("Phases: %d configured\n", len(profile.Phases))
	}
	return nil
}
