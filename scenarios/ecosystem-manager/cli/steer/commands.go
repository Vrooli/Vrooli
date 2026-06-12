// Package steer provides CLI commands for auto-steer profile management.
package steer

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"ecosystem-manager/cli/internal/appctx"
	"ecosystem-manager/cli/internal/format"

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
	DeniedSkills  []string `json:"denied_skills,omitempty"`
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
	// Ladder mirrors the profile's maturity-ladder block; nil/absent ⇒ plain
	// greedy. The CLI must declare it or `steer show --json` silently drops it.
	Ladder *struct {
		Enabled           bool    `json:"enabled"`
		TopRung           string  `json:"top_rung,omitempty"`
		BoostFactor       float64 `json:"boost_factor,omitempty"`
		DimensionMaxCount int     `json:"dimension_max_count,omitempty"`
	} `json:"ladder,omitempty"`
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

	// CurrentRung is the maturity rung the controller worked this iteration (the
	// lowest unsatisfied rung); empty when the profile has no ladder. GamingCause
	// is the anti-gaming classifier verdict (empty = clean). Both are persisted on
	// the trace and surfaced by the API/UI; the CLI must mirror them or its --json
	// silently drops them.
	CurrentRung string `json:"current_rung"`
	GamingCause string `json:"gaming_cause"`

	// DTV transparency (P2): the chosen skill's fitness verdict, the cold-start
	// trust/cost prior DTV seeded, any Layer-1 exclusions, and degradation flags.
	DTVVerdict      string            `json:"dtv_verdict"`
	DTVPrior        float64           `json:"dtv_prior"`
	DTVExcluded     map[string]string `json:"dtv_excluded"`
	DTVGateOverride bool              `json:"dtv_gate_override"`
	DTVDegraded     bool              `json:"dtv_degraded"`
}

// CoverageReport is the auto-steer coverage doctor/preflight response.
type CoverageReport struct {
	ProfileID              string                 `json:"profile_id"`
	ProfileName            string                 `json:"profile_name"`
	Scenario               string                 `json:"scenario,omitempty"`
	EffectiveAllowSet      []string               `json:"effective_allow_set"`
	RelevantDimensions     []string               `json:"relevant_dimensions"`
	GatedUncovered         []CoverageDimensionGap `json:"gated_uncovered"`
	WeightedUnactionable   []CoverageDimensionGap `json:"weighted_unactionable"`
	ExcludedSkills         []string               `json:"excluded_skills"`
	KnownUncoveredInPlay   []CoverageKnownEntry   `json:"known_uncovered_in_play"`
	ReconciliationWarnings []string               `json:"reconciliation_warnings,omitempty"`
}

// CoverageDimensionGap describes a dimension that has no eligible skill.
type CoverageDimensionGap struct {
	Dimension   string `json:"dimension"`
	Reason      string `json:"reason,omitempty"`
	TrackingRef string `json:"tracking_ref,omitempty"`
}

// CoverageKnownEntry is a known-uncovered policy entry in play for this report.
type CoverageKnownEntry struct {
	Dimension   string `json:"dimension"`
	Reason      string `json:"reason"`
	TrackingRef string `json:"tracking_ref"`
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
			{
				Name:        "coverage",
				NeedsAPI:    true,
				Description: "Run auto-steer profile coverage preflight",
				Run: func(args []string) error {
					return cmdCoverage(ctx, args)
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
  ecosystem-manager steer trace <task-id>
  ecosystem-manager coverage --profile production-ready`
}

func cmdCoverage(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("coverage", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	profileID := fs.String("profile", "", "Profile ID to inspect")
	scenario := fs.String("scenario", "", "Optional scenario name for launch context")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *profileID == "" && fs.NArg() > 0 {
		*profileID = fs.Arg(0)
	}
	if *profileID == "" {
		return fmt.Errorf("usage: ecosystem-manager coverage --profile <profile-id>")
	}

	query := url.Values{"profile": []string{*profileID}}
	if *scenario != "" {
		query.Set("scenario", *scenario)
	}
	var report CoverageReport
	if err := ctx.GetWithQuery("/auto-steer/coverage", query, &report); err != nil {
		return format.WrapAPIError("Failed to build coverage report", err)
	}
	if *jsonOut {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return renderCoverageReport(report)
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
		skills := "derived skills"
		if len(p.AllowedSkills) > 0 {
			skills = fmt.Sprintf("%d-skill mask", len(p.AllowedSkills))
		}
		results = append(results, fmt.Sprintf("%s (%s, max %d iters)%s [%s]",
			p.Name, skills, p.Budget.MaxIterations, desc, p.ID))
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
		results = append(results, fmt.Sprintf("Allowed skill mask: %v", profile.AllowedSkills))
	} else {
		results = append(results, "Allowed skill mask: <derived from dimensions>")
	}
	if len(profile.DeniedSkills) > 0 {
		results = append(results, fmt.Sprintf("Denied skills: %v", profile.DeniedSkills))
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
	if l := profile.Ladder; l != nil && l.Enabled {
		top := l.TopRung
		if top == "" {
			top = "R4"
		}
		results = append(results, fmt.Sprintf("Maturity ladder: enabled, top rung %s, boost ×%.0f, dimension cap %d",
			top, l.BoostFactor, l.DimensionMaxCount))
	}
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
		if e.CurrentRung != "" {
			line += fmt.Sprintf(" · rung %s", e.CurrentRung)
		}
		if e.GamingCause != "" {
			line += fmt.Sprintf(" · GAMING: %s", e.GamingCause)
		}
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
		if dtv := dtvTraceClause(e); dtv != "" {
			line += dtv
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

// dtvTraceClause renders the DTV provenance of a trace entry (empty when DTV was
// inactive — verdict unset and no exclusions).
func dtvTraceClause(e TraceEntry) string {
	if e.DTVVerdict == "" && len(e.DTVExcluded) == 0 && !e.DTVDegraded {
		return ""
	}
	clause := ""
	if e.DTVVerdict != "" {
		clause += fmt.Sprintf(" | DTV %s (prior %.2f)", e.DTVVerdict, e.DTVPrior)
	}
	if len(e.DTVExcluded) > 0 {
		excluded := make([]string, 0, len(e.DTVExcluded))
		for skill, reason := range e.DTVExcluded {
			excluded = append(excluded, fmt.Sprintf("%s(%s)", skill, reason))
		}
		sort.Strings(excluded)
		clause += fmt.Sprintf(", gated %s", strings.Join(excluded, ","))
	}
	if e.DTVGateOverride {
		clause += " [all-red override]"
	}
	if e.DTVDegraded {
		clause += " [DTV degraded → P1]"
	}
	return clause
}

func renderCoverageReport(report CoverageReport) error {
	status := []string{
		fmt.Sprintf("Profile: %s [%s]", report.ProfileName, report.ProfileID),
		fmt.Sprintf("Effective allow-set: %d skill(s)", len(report.EffectiveAllowSet)),
		fmt.Sprintf("Relevant dimensions: %d", len(report.RelevantDimensions)),
	}
	if report.Scenario != "" {
		status = append(status, fmt.Sprintf("Scenario: %s", report.Scenario))
	}
	if len(report.GatedUncovered) == 0 && len(report.WeightedUnactionable) == 0 && len(report.ReconciliationWarnings) == 0 {
		status = append(status, "Coverage preflight: no untracked gaps")
	}

	triage := []cliapp.TriageGroup{
		{Heading: "Effective Skills", Items: compactList(report.EffectiveAllowSet, "none")},
		{Heading: "Relevant Dimensions", Items: compactList(report.RelevantDimensions, "none")},
	}
	if len(report.GatedUncovered) > 0 {
		triage = append(triage, cliapp.TriageGroup{
			Heading: "Gated Dimensions Without Eligible Skills",
			Items:   gapLines(report.GatedUncovered),
		})
	}
	if len(report.WeightedUnactionable) > 0 {
		triage = append(triage, cliapp.TriageGroup{
			Heading: "Weighted Dimensions Without Eligible Skills",
			Items:   gapLines(report.WeightedUnactionable),
		})
	}
	if len(report.ExcludedSkills) > 0 {
		triage = append(triage, cliapp.TriageGroup{
			Heading: "Catalog Skills Excluded By Vocabulary",
			Items:   compactList(report.ExcludedSkills, "none"),
		})
	}
	if len(report.KnownUncoveredInPlay) > 0 {
		triage = append(triage, cliapp.TriageGroup{
			Heading: "Known Uncovered Entries",
			Items:   knownEntryLines(report.KnownUncoveredInPlay),
		})
	}
	if len(report.ReconciliationWarnings) > 0 {
		triage = append(triage, cliapp.TriageGroup{
			Heading: "Profile Reconciliation",
			Items:   report.ReconciliationWarnings,
		})
	}

	return cliapp.RenderOperationalReport(os.Stdout, cliapp.OperationalReport{
		Status: status,
		Triage: triage,
		NextSteps: []string{
			fmt.Sprintf("ecosystem-manager coverage --profile %s --json", report.ProfileID),
			"ecosystem-manager steer show " + report.ProfileID,
		},
	})
}

func compactList(values []string, empty string) []string {
	if len(values) == 0 {
		return []string{empty}
	}
	return []string{strings.Join(values, ", ")}
}

func gapLines(gaps []CoverageDimensionGap) []string {
	lines := make([]string, 0, len(gaps))
	for _, gap := range gaps {
		line := gap.Dimension
		if gap.TrackingRef != "" {
			line += fmt.Sprintf(" — known uncovered (%s)", gap.TrackingRef)
		}
		if gap.Reason != "" {
			line += ": " + gap.Reason
		}
		lines = append(lines, line)
	}
	return lines
}

func knownEntryLines(entries []CoverageKnownEntry) []string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s — %s: %s", entry.Dimension, entry.TrackingRef, entry.Reason))
	}
	return lines
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
