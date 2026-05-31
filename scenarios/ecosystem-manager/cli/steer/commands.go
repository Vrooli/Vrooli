// Package steer provides CLI commands for auto-steer profile management.
package steer

import (
	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/internal/format"
	"flag"
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// ProfileListResponse represents a list of auto-steer profiles.
type ProfileListResponse struct {
	Profiles []Profile `json:"profiles"`
	Count    int       `json:"count"`
}

// Profile represents an objective-function auto-steer profile.
type Profile struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	AllowedSkills []string `json:"allowed_skills,omitempty"`
	AuditPreset   string   `json:"audit_preset,omitempty"`
	Objective     struct {
		DimensionWeights map[string]float64 `json:"dimension_weights,omitempty"`
		Targets          struct {
			MaxOpenSeverity       string  `json:"max_open_severity,omitempty"`
			OperationalTargetsPct float64 `json:"operational_targets_pct,omitempty"`
		} `json:"targets"`
	} `json:"objective"`
	Budget struct {
		MaxIterations           int     `json:"max_iterations"`
		DiminishingReturnsFloor float64 `json:"diminishing_returns_floor"`
		ReauditCadence          int     `json:"reaudit_cadence"`
	} `json:"budget"`
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
  show, get <id>  Show profile details

Examples:
  ecosystem-manager steer profiles
  ecosystem-manager steer templates --json
  ecosystem-manager steer show balanced`
}

func cmdProfiles(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("profiles", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp ProfileListResponse
	if err := ctx.Get("/auto-steer/profiles", &resp); err != nil {
		return format.WrapAPIError("Failed to list profiles", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if len(resp.Profiles) == 0 {
		return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
			Summary:        []string{"No auto-steer profiles found"},
			Results:        nil,
			RetrievalHints: []string{"ecosystem-manager steer templates"},
		})
	}

	results := make([]string, 0, len(resp.Profiles))
	for _, p := range resp.Profiles {
		desc := ""
		if p.Description != "" {
			desc = fmt.Sprintf(" - %s", p.Description)
		}
		results = append(results, fmt.Sprintf("%s (%d skills, max %d iters)%s [%s]",
			p.Name, len(p.AllowedSkills), p.Budget.MaxIterations, desc, p.ID))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Auto-steer profiles available: %d", resp.Count)},
		Results:        results,
		RetrievalHints: []string{"ecosystem-manager steer show <profile-id>"},
	})
}

func cmdTemplates(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("templates", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp TemplateListResponse
	if err := ctx.Get("/auto-steer/templates", &resp); err != nil {
		return format.WrapAPIError("Failed to list templates", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if len(resp.Templates) == 0 {
		return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
			Summary:        []string{"No auto-steer templates found"},
			RetrievalHints: []string{"ecosystem-manager status"},
		})
	}

	results := make([]string, 0, len(resp.Templates))
	for _, t := range resp.Templates {
		desc := ""
		if t.Description != "" {
			desc = fmt.Sprintf(" - %s", t.Description)
		}
		results = append(results, fmt.Sprintf("%s%s [%s]", t.Name, desc, t.ID))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Auto-steer templates available: %d", resp.Count)},
		Results:        results,
		RetrievalHints: []string{"ecosystem-manager steer profiles"},
	})
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
		return format.WrapAPIError("Failed to get profile", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, profile)
	}

	summary := []string{
		fmt.Sprintf("Profile: %s", profile.Name),
		fmt.Sprintf("ID: %s", profile.ID),
	}
	if profile.Description != "" {
		summary = append(summary, fmt.Sprintf("Description: %s", profile.Description))
	}
	results := []string{}
	if len(profile.AllowedSkills) > 0 {
		results = append(results, fmt.Sprintf("Allowed skills: %v", profile.AllowedSkills))
	}
	if len(profile.Objective.DimensionWeights) > 0 {
		results = append(results, fmt.Sprintf("Dimension weights: %v", profile.Objective.DimensionWeights))
	}
	if sev := profile.Objective.Targets.MaxOpenSeverity; sev != "" {
		results = append(results, fmt.Sprintf("Target — max open severity: %s", sev))
	}
	if pct := profile.Objective.Targets.OperationalTargetsPct; pct > 0 {
		results = append(results, fmt.Sprintf("Target — operational targets ≥ %.0f%%", pct))
	}
	results = append(results,
		fmt.Sprintf("Budget — max iterations: %d, diminishing-returns floor: %.3f, re-audit cadence: %d",
			profile.Budget.MaxIterations, profile.Budget.DiminishingReturnsFloor, profile.Budget.ReauditCadence))
	if profile.AuditPreset != "" {
		results = append(results, fmt.Sprintf("Audit preset: %s", profile.AuditPreset))
	}
	if len(profile.Tags) > 0 {
		results = append(results, fmt.Sprintf("Tags: %v", profile.Tags))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        summary,
		Results:        results,
		RetrievalHints: []string{"ecosystem-manager steer profiles"},
	})
}
