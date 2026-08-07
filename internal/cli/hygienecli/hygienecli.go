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
	DriftOnly    bool
	NoDrift      bool
	NoFreshness  bool
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
				{Name: "--plans", Description: "Ask Plan Manager to reconcile plan mirrors and source markdown during --fix-safe"},
				{Name: "--fail-on", ValueName: "severity", Description: "Exit non-zero on warning or error"},
				{Name: "--summary", Description: "Show compact status, findings, and plan counts"},
				{Name: "--details", Description: "Show full findings, checks, and plan candidate lists"},
				{Name: "--next", Description: "Show only recommended next commands and actions"},
				{Name: "--plans-only", Description: "Run only plan lifecycle hygiene checks"},
				{Name: "--contract-only", Description: "Run only repository contract hygiene checks"},
				{Name: "--drift-only", Description: "Run only SDA-backed dependency freshness hygiene"},
				{Name: "--no-drift", Description: "Skip SDA-backed dependency freshness hygiene"},
				{Name: "--no-freshness", Description: "Skip the advisory test-freshness check (changed scenarios vs test-genie runs)"},
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
	plansOnly := parsed.HasFlag("--plans-only")
	contractOnly := parsed.HasFlag("--contract-only")
	driftOnly := parsed.HasFlag("--drift-only")
	noDrift := parsed.HasFlag("--no-drift")
	onlyCount := 0
	for _, set := range []bool{plansOnly, contractOnly, driftOnly} {
		if set {
			onlyCount++
		}
	}
	if onlyCount > 1 {
		return Request{}, fmt.Errorf("--plans-only, --contract-only, and --drift-only are mutually exclusive")
	}
	if driftOnly && noDrift {
		return Request{}, fmt.Errorf("--drift-only and --no-drift are mutually exclusive")
	}
	return Request{
		FixSafe:      parsed.HasFlag("--fix-safe"),
		Plans:        parsed.HasFlag("--plans"),
		FailOn:       failOn,
		OutputMode:   mode,
		PlansOnly:    plansOnly,
		ContractOnly: contractOnly,
		DriftOnly:    driftOnly,
		NoDrift:      noDrift,
		NoFreshness:  parsed.HasFlag("--no-freshness"),
	}, nil
}

func Render(w io.Writer, format cliout.Format, report hygieneapp.Report, mode OutputMode) error {
	if format == cliout.FormatJSON {
		return writeHygieneReportJSON(w, report)
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
	if report.SharedDrift != nil {
		renderDriftSummary(w, report.SharedDrift, mode)
	}
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
			_, _ = fmt.Fprintf(w, "- %s %s -> %s\n", planFixActionLabel(fix.Action), fix.Source, fix.Plan.Path)
		}
	}
	if len(report.PlanReconcileOutcomes) > 0 {
		_, _ = fmt.Fprintln(w, "\nPlan reconcile results:")
		for _, outcome := range report.PlanReconcileOutcomes {
			target := outcome.Plan.Path
			if target == "" {
				target = outcome.Mirror.Path
			}
			if target == "" {
				target = outcome.Source
			}
			line := fmt.Sprintf("- %s %s", planFixActionLabel(outcome.Action), outcome.Source)
			if target != "" && target != outcome.Source {
				line += " -> " + target
			}
			if outcome.Mirror.Status != "" {
				line += " [" + outcome.Mirror.Status + "]"
			}
			if outcome.SourceUntouched {
				line += " (source untouched)"
			}
			if outcome.Error != "" {
				line += ": " + outcome.Error
			}
			_, _ = fmt.Fprintln(w, line)
		}
	}
	if len(report.ConfigFixes) > 0 {
		_, _ = fmt.Fprintln(w, "\nConfig fixes applied:")
		for _, fix := range report.ConfigFixes {
			_, _ = fmt.Fprintf(w, "- %s\n", fix)
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
		for i, action := range finding.NextActions {
			label := "Next"
			if i > 0 {
				label = "Also"
			}
			if action.Command != "" {
				_, _ = fmt.Fprintf(w, "   %s: %s\n", label, action.Command)
			} else {
				_, _ = fmt.Fprintf(w, "   %s: %s\n", label, action.Message)
			}
			if action.Fixability != "" || (action.Command != "" && action.Message != "") {
				var details []string
				if action.Fixability != "" {
					details = append(details, string(action.Fixability))
				}
				if action.Command != "" && action.Message != "" {
					details = append(details, action.Message)
				}
				_, _ = fmt.Fprintf(w, "   %s\n", strings.Join(details, " - "))
			}
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

func renderDriftSummary(w io.Writer, drift *hygieneapp.DependencyFreshnessCompatReport, mode OutputMode) {
	var stale []hygieneapp.DependencyFreshnessScenario
	var errored []hygieneapp.DependencyFreshnessScenario
	for _, sc := range drift.Scenarios {
		switch sc.Status {
		case hygieneapp.DependencyFreshnessStatusStaleModules, hygieneapp.DependencyFreshnessStatusStaleBuild:
			stale = append(stale, sc)
		case hygieneapp.DependencyFreshnessStatusError:
			errored = append(errored, sc)
		}
	}
	_, _ = fmt.Fprintf(w, "\nShared-package drift: %d scenarios checked", len(drift.Scenarios))
	if drift.OnlyTouchedUsed {
		_, _ = fmt.Fprintf(w, " (only-touched)")
	}
	_, _ = fmt.Fprintln(w)
	if len(stale) == 0 && len(errored) == 0 {
		_, _ = fmt.Fprintln(w, "- clean")
		return
	}
	if len(stale) > 0 {
		limit := len(stale)
		if mode != OutputModeDetails && limit > 10 {
			limit = 10
		}
		_, _ = fmt.Fprintln(w, "Stale scenarios:")
		for _, sc := range stale[:limit] {
			_, _ = fmt.Fprintf(w, "- %s [%s]\n", sc.Path, sc.Status)
		}
		if limit < len(stale) {
			_, _ = fmt.Fprintf(w, "- ... %d more; run `vrooli hygiene --details` to list all\n", len(stale)-limit)
		}
	}
	if len(errored) > 0 {
		_, _ = fmt.Fprintln(w, "Drift check errors:")
		limit := len(errored)
		if mode != OutputModeDetails && limit > 10 {
			limit = 10
		}
		for _, sc := range errored[:limit] {
			_, _ = fmt.Fprintf(w, "- %s — %s\n", sc.Path, dependencyFreshnessErrorSummary(sc.Error, mode))
		}
		if limit < len(errored) {
			_, _ = fmt.Fprintf(w, "- ... %d more; run `vrooli hygiene --details` to list all\n", len(errored)-limit)
		}
	}
}

func dependencyFreshnessErrorSummary(message string, mode OutputMode) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "freshness check failed"
	}
	if mode == OutputModeDetails {
		return message
	}
	if isMissingLocalReplaceError(message) {
		return "missing local replace for an in-repo Go module; preview repair with `scenario-dependency-analyzer deps reconcile --all --json`"
	}
	if idx := strings.IndexByte(message, '\n'); idx >= 0 {
		return strings.TrimSpace(message[:idx])
	}
	return message
}

func isMissingLocalReplaceError(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "github.com/vrooli/") &&
		strings.Contains(message, "go.mod at revision v0.0.0") &&
		strings.Contains(message, "repository not found")
}

func planFixActionLabel(action string) string {
	switch action {
	case "imported":
		return "imported"
	case "mirror_repaired":
		return "mirror repaired"
	case "skipped_duplicate":
		return "skipped duplicate"
	case "":
		return "reconciled"
	default:
		return strings.ReplaceAll(action, "_", " ")
	}
}

func renderPlanSummary(w io.Writer, candidates []hygieneapp.PlanCandidate, mode OutputMode) {
	counts := map[string]int{}
	var modified []string
	var untracked []string
	var invalid []hygieneapp.PlanCandidate
	var cleanupPlanned []hygieneapp.PlanCandidate
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
		case "parse_failed", "conflict":
			invalid = append(invalid, candidate)
		case "source_retirement_planned":
			cleanupPlanned = append(cleanupPlanned, candidate)
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
	if len(invalid) > 0 {
		_, _ = fmt.Fprintln(w, "\nInvalid plan sources:")
		renderPlanCandidatesWithReasons(w, invalid, mode)
	}
	if len(cleanupPlanned) > 0 && mode == OutputModeDetails {
		_, _ = fmt.Fprintln(w, "\nPlan sources ready for retirement:")
		renderPlanCandidatesWithReasons(w, cleanupPlanned, mode)
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

func renderPlanCandidatesWithReasons(w io.Writer, candidates []hygieneapp.PlanCandidate, mode OutputMode) {
	limit := len(candidates)
	if mode != OutputModeDetails && limit > 10 {
		limit = 10
	}
	for _, candidate := range candidates[:limit] {
		if candidate.Reason != "" {
			_, _ = fmt.Fprintf(w, "- %s: %s\n", candidate.Path, candidate.Reason)
		} else {
			_, _ = fmt.Fprintf(w, "- %s\n", candidate.Path)
		}
	}
	if limit < len(candidates) {
		_, _ = fmt.Fprintf(w, "- ... %d more; run `vrooli hygiene --details` to list all\n", len(candidates)-limit)
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
