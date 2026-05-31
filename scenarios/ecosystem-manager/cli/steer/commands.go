// Package steer provides CLI commands for auto-steer profile management.
package steer

import (
	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/internal/format"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

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

// EffectivenessResponse is the per-(skill, dimension) effectiveness ledger.
type EffectivenessResponse struct {
	Effectiveness []EffectivenessRow `json:"effectiveness"`
	Count         int                `json:"count"`
	Prior         float64            `json:"prior"`
	ShrinkageK    float64            `json:"shrinkage_k"`
}

// EffectivenessRow is one ledger row with the derived efficacy estimate.
type EffectivenessRow struct {
	SkillID                 string  `json:"skill_id"`
	Dimension               string  `json:"dimension"`
	ClosedCount             int64   `json:"closed_count"`
	IntroducedCount         int64   `json:"introduced_count"`
	NetClosed               int64   `json:"net_closed"`
	TotalRuns               int64   `json:"total_runs"`
	TotalTokens             int64   `json:"total_tokens"`
	AvgTokensPerRun         int64   `json:"avg_tokens_per_run"`
	ObservedEfficacyPerKtok float64 `json:"observed_efficacy_per_ktok"`
	ExpectedEfficacyPerKtok float64 `json:"expected_efficacy_per_ktok"`
}

// TraceResponse is the controller's per-iteration decision trace for a task.
type TraceResponse struct {
	TaskID string       `json:"task_id"`
	Trace  []TraceEntry `json:"trace"`
	Count  int          `json:"count"`
}

// TraceEntry is one iteration of the decision trace.
type TraceEntry struct {
	Iteration             int            `json:"iteration"`
	ChosenSkill           string         `json:"chosen_skill"`
	HeaviestDimension     string         `json:"heaviest_dimension"`
	Rationale             string         `json:"rationale"`
	ScoreBefore           float64        `json:"score_before"`
	ScoreAfter            float64        `json:"score_after"`
	RealizedDelta         float64        `json:"realized_delta"`
	TokensUsed            int64          `json:"tokens_used"`
	ClosedByDimension     map[string]int `json:"closed_by_dimension"`
	IntroducedByDimension map[string]int `json:"introduced_by_dimension"`
	Regressed             bool           `json:"regressed"`
	VetoApplied           bool           `json:"veto_applied"`
	HaltReason            string         `json:"halt_reason"`
}

// Commands returns the steer command group.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Auto Steer",
		Commands: []cliapp.Command{
			{
				Name:        "steer",
				NeedsAPI:    true,
				Description: "Manage auto-steer profiles (profiles|templates|show|effectiveness|trace)",
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
	case "effectiveness", "eff":
		return cmdEffectiveness(ctx, subArgs)
	case "trace":
		return cmdTrace(ctx, subArgs)
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
  profiles, ls           List auto-steer profiles
  templates              List auto-steer templates
  show, get <id>         Show profile details
  effectiveness, eff     Show the skill×dimension effectiveness ledger
  trace <taskId>         Show a run's per-iteration decision trace

Examples:
  ecosystem-manager steer profiles
  ecosystem-manager steer templates --json
  ecosystem-manager steer show balanced
  ecosystem-manager steer effectiveness --dimension standards
  ecosystem-manager steer trace <task-id>`
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

func cmdEffectiveness(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("effectiveness", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	skill := fs.String("skill", "", "Filter by skill ID")
	dim := fs.String("dimension", "", "Filter by dimension")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	path := "/auto-steer/effectiveness"
	if q := buildQuery(*skill, *dim); q != "" {
		path += "?" + q
	}

	var resp EffectivenessResponse
	if err := ctx.Get(path, &resp); err != nil {
		return format.WrapAPIError("Failed to load effectiveness ledger", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if len(resp.Effectiveness) == 0 {
		return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
			Summary:        []string{"No effectiveness data yet — the ledger fills as steered runs complete iterations"},
			RetrievalHints: []string{"ecosystem-manager steer profiles"},
		})
	}

	results := make([]string, 0, len(resp.Effectiveness))
	for _, e := range resp.Effectiveness {
		results = append(results, fmt.Sprintf(
			"%s × %s: net %+d (closed %d / introduced %d) over %d run(s), ~%d tok/run — efficacy %.2f/khtok (observed %.2f)",
			e.SkillID, e.Dimension, e.NetClosed, e.ClosedCount, e.IntroducedCount,
			e.TotalRuns, e.AvgTokensPerRun, e.ExpectedEfficacyPerKtok, e.ObservedEfficacyPerKtok))
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Skill effectiveness rows: %d (prior %.2f, shrinkage k=%.0f)", resp.Count, resp.Prior, resp.ShrinkageK),
		},
		Results:        results,
		RetrievalHints: []string{"ecosystem-manager steer effectiveness --dimension <dim>"},
	})
}

func cmdTrace(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("trace", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: steer trace <taskId>")
	}
	taskID := fs.Arg(0)

	var resp TraceResponse
	if err := ctx.Get(fmt.Sprintf("/auto-steer/execution/%s/trace", taskID), &resp); err != nil {
		return format.WrapAPIError("Failed to load decision trace", err)
	}

	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, resp)
	}

	if len(resp.Trace) == 0 {
		return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("No decision trace for task %s", taskID)},
			RetrievalHints: []string{"ecosystem-manager steer profiles"},
		})
	}

	results := make([]string, 0, len(resp.Trace))
	for _, e := range resp.Trace {
		line := fmt.Sprintf("iter %d: %s [%s] score %.1f→%.1f (Δ%+.1f), %d tok",
			e.Iteration, orNone(e.ChosenSkill), e.HeaviestDimension,
			e.ScoreBefore, e.ScoreAfter, e.RealizedDelta, e.TokensUsed)
		if n := sumCounts(e.ClosedByDimension); n > 0 {
			line += fmt.Sprintf(" — closed %d", n)
		}
		if n := sumCounts(e.IntroducedByDimension); n > 0 {
			line += fmt.Sprintf(", introduced %d", n)
		}
		if e.Regressed {
			line += " ⚠ regressed"
		}
		if e.VetoApplied {
			line += " (veto)"
		}
		if e.HaltReason != "" {
			line += fmt.Sprintf(" — HALT: %s", e.HaltReason)
		}
		results = append(results, line)
		if e.Rationale != "" {
			results = append(results, "    "+e.Rationale)
		}
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Decision trace for task %s: %d iteration(s)", taskID, resp.Count)},
		Results:        results,
		RetrievalHints: []string{"ecosystem-manager steer effectiveness"},
	})
}

func buildQuery(skill, dim string) string {
	parts := make([]string, 0, 2)
	if skill != "" {
		parts = append(parts, "skill="+url.QueryEscape(skill))
	}
	if dim != "" {
		parts = append(parts, "dimension="+url.QueryEscape(dim))
	}
	return strings.Join(parts, "&")
}

func sumCounts(m map[string]int) int {
	total := 0
	for _, n := range m {
		total += n
	}
	return total
}

func orNone(s string) string {
	if s == "" {
		return "(no skill)"
	}
	return s
}
