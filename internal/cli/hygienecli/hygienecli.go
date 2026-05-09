package hygienecli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	hygieneapp "github.com/vrooli/vrooli/internal/app/hygiene"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
)

type Request struct {
	FixSafe      bool
	Plans        bool
	FailOn       hygieneapp.Severity
	OutputMode   OutputMode
	ContractOnly bool
	PlansOnly    bool
}

type OutputMode string

const (
	OutputModeDefault OutputMode = ""
	OutputModeSummary OutputMode = "summary"
	OutputModeDetails OutputMode = "details"
	OutputModeNext    OutputMode = "next"
)

func CommandSpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{
		Name:    "hygiene",
		Summary: "Run repository hygiene checks",
		Group:   "Maintenance Commands",
		Args: commandtree.ArgSchema{
			Options: []commandtree.OptionArg{
				{Name: "--fix-safe", Description: "Apply safe, non-destructive fixes"},
				{Name: "--plans", Description: "Include safe plan candidate imports when fixing"},
				{Name: "--fail-on", ValueName: "severity", Description: "Exit non-zero on warning or error"},
				{Name: "--summary", Description: "Show compact status, findings, and plan counts"},
				{Name: "--details", Description: "Show full findings, checks, and plan candidate lists"},
				{Name: "--next", Description: "Show only recommended next commands and actions"},
				{Name: "--plans-only", Description: "Run only plan lifecycle hygiene checks"},
				{Name: "--contract-only", Description: "Run only repository contract hygiene checks"},
				commandtree.JSONOption(),
			},
		},
		Handler: "hygiene",
	}
}

func ParseRequest(args []string) (Request, error) {
	spec := CommandSpec()
	parsed, err := commandtree.ParseArgs("hygiene", commandtree.SpecHelpText("", "vrooli hygiene", spec), spec.Args, args)
	if err != nil {
		return Request{}, err
	}
	failOn := hygieneapp.Severity(parsed.FlagValue("--fail-on"))
	switch failOn {
	case "", hygieneapp.SeverityError, hygieneapp.SeverityWarning:
	default:
		return Request{}, fmt.Errorf("--fail-on must be warning or error")
	}
	mode := OutputModeDefault
	modeFlags := 0
	for _, item := range []struct {
		flag string
		mode OutputMode
	}{
		{flag: "--summary", mode: OutputModeSummary},
		{flag: "--details", mode: OutputModeDetails},
		{flag: "--next", mode: OutputModeNext},
	} {
		if parsed.HasFlag(item.flag) {
			mode = item.mode
			modeFlags++
		}
	}
	if modeFlags > 1 {
		return Request{}, fmt.Errorf("--summary, --details, and --next are mutually exclusive")
	}
	if parsed.HasFlag("--plans-only") && parsed.HasFlag("--contract-only") {
		return Request{}, fmt.Errorf("--plans-only and --contract-only are mutually exclusive")
	}
	return Request{
		FixSafe:      parsed.HasFlag("--fix-safe"),
		Plans:        parsed.HasFlag("--plans"),
		FailOn:       failOn,
		OutputMode:   mode,
		PlansOnly:    parsed.HasFlag("--plans-only"),
		ContractOnly: parsed.HasFlag("--contract-only"),
	}, nil
}

func Render(w io.Writer, format cliout.Format, report hygieneapp.Report, mode OutputMode) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, report)
	}
	if mode == OutputModeNext {
		renderNextSteps(w, report, false)
		return nil
	}
	status := "passed"
	if !report.Success {
		status = "failed"
	}
	_, _ = fmt.Fprintf(w, "Status: hygiene %s\n", status)
	_, _ = fmt.Fprintf(w, "Root: %s\n", report.Root)
	_, _ = fmt.Fprintf(w, "Blocking issues: %d\n", report.BlockingFailures)
	_, _ = fmt.Fprintf(w, "Warnings: %d\n", report.Warnings)
	renderFindings(w, report)
	if len(report.PlanCandidates) > 0 {
		renderPlanSummary(w, report.PlanCandidates, mode)
	}
	if mode == OutputModeDetails && len(report.Checks) > 0 {
		_, _ = fmt.Fprintln(w, "\nChecks:")
		for _, check := range report.Checks {
			state := "ok"
			if !check.Passed {
				state = "failed"
			}
			_, _ = fmt.Fprintf(w, "- %s: %s (%s)\n", check.Name, state, check.Message)
		}
	}
	if len(report.FixesApplied) > 0 {
		_, _ = fmt.Fprintln(w, "\nFixes applied:")
		for _, fix := range report.FixesApplied {
			_, _ = fmt.Fprintf(w, "- imported %s -> %s\n", fix.Source, fix.Plan.Path)
		}
	}
	renderNextSteps(w, report, true)
	return nil
}

func renderFindings(w io.Writer, report hygieneapp.Report) {
	blockers, warnings := splitFindings(report.Findings)
	if len(blockers) > 0 {
		_, _ = fmt.Fprintln(w, "\nBlocking issues:")
		for i, finding := range blockers {
			renderFinding(w, i+1, finding)
		}
	}
	if len(warnings) > 0 {
		_, _ = fmt.Fprintln(w, "\nWarnings:")
		for i, finding := range warnings {
			renderFinding(w, i+1, finding)
		}
	}
}

func renderFinding(w io.Writer, index int, finding hygieneapp.Finding) {
	_, _ = fmt.Fprintf(w, "%d. %s", index, finding.Code)
	if finding.Fixability != "" {
		_, _ = fmt.Fprintf(w, " [%s]", finding.Fixability)
	}
	_, _ = fmt.Fprintln(w)
	if finding.Path != "" {
		_, _ = fmt.Fprintf(w, "   Path: %s\n", finding.Path)
	}
	if len(finding.Locations) > 0 {
		_, _ = fmt.Fprintf(w, "   Locations: %s\n", strings.Join(finding.Locations, ", "))
	}
	_, _ = fmt.Fprintf(w, "   %s\n", finding.Message)
	if finding.Why != "" {
		_, _ = fmt.Fprintf(w, "   Why: %s\n", finding.Why)
	}
	if len(finding.NextActions) > 0 {
		action := finding.NextActions[0]
		if action.Command != "" {
			_, _ = fmt.Fprintf(w, "   Next: %s\n", action.Command)
		} else {
			_, _ = fmt.Fprintf(w, "   Next: %s\n", action.Message)
		}
	}
}

func splitFindings(findings []hygieneapp.Finding) ([]hygieneapp.Finding, []hygieneapp.Finding) {
	blockers := make([]hygieneapp.Finding, 0, len(findings))
	warnings := make([]hygieneapp.Finding, 0, len(findings))
	for _, finding := range findings {
		switch finding.Severity {
		case hygieneapp.SeverityError:
			blockers = append(blockers, finding)
		case hygieneapp.SeverityWarning:
			warnings = append(warnings, finding)
		}
	}
	return blockers, warnings
}

func renderPlanSummary(w io.Writer, candidates []hygieneapp.PlanCandidate, mode OutputMode) {
	counts := map[string]int{}
	var modified []string
	var untracked []string
	for _, candidate := range candidates {
		status := candidate.Status
		if status == "" {
			status = "unknown"
		}
		counts[status]++
		switch status {
		case "modified":
			modified = append(modified, candidate.Path)
		case "untracked":
			untracked = append(untracked, candidate.Path)
		}
	}
	_, _ = fmt.Fprintf(w, "\nPlan candidates: %d\n", len(candidates))
	for _, status := range sortedKeys(counts) {
		_, _ = fmt.Fprintf(w, "- %d %s\n", counts[status], status)
	}
	if len(modified) > 0 {
		_, _ = fmt.Fprintln(w, "\nModified plan candidates:")
		for _, path := range modified {
			_, _ = fmt.Fprintf(w, "- %s\n", path)
		}
	}
	if len(untracked) > 0 {
		_, _ = fmt.Fprintln(w, "\nUntracked plan candidates:")
		limit := len(untracked)
		if mode != OutputModeDetails && limit > 10 {
			limit = 10
		}
		for _, path := range untracked[:limit] {
			_, _ = fmt.Fprintf(w, "- %s\n", path)
		}
		if limit < len(untracked) {
			_, _ = fmt.Fprintf(w, "- ... %d more; run `vrooli hygiene --details` to list all\n", len(untracked)-limit)
		}
	}
}

func renderNextSteps(w io.Writer, report hygieneapp.Report, leadingBlank bool) {
	actions := collectActions(report)
	if len(actions) == 0 {
		return
	}
	if leadingBlank {
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w, "Next steps:")
	for _, action := range actions {
		if action.Command != "" {
			_, _ = fmt.Fprintf(w, "- %s\n", action.Command)
			if action.Message != "" {
				_, _ = fmt.Fprintf(w, "  %s\n", action.Message)
			}
		} else {
			_, _ = fmt.Fprintf(w, "- %s\n", action.Message)
		}
	}
}

func collectActions(report hygieneapp.Report) []hygieneapp.Action {
	var actions []hygieneapp.Action
	seen := map[string]bool{}
	add := func(action hygieneapp.Action) {
		key := action.Code + "\x00" + action.Command + "\x00" + action.Message
		if seen[key] {
			return
		}
		seen[key] = true
		actions = append(actions, action)
	}
	for _, finding := range report.Findings {
		for _, action := range finding.NextActions {
			add(action)
		}
	}
	for _, action := range report.Actions {
		add(action)
	}
	if len(report.Findings) > 0 {
		add(hygieneapp.Action{
			Code:    "rerun_hygiene",
			Message: "Rerun hygiene after fixes.",
			Command: "vrooli hygiene",
		})
	}
	return actions
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
