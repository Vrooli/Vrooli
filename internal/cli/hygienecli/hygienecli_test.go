package hygienecli

import (
	"bytes"
	"strings"
	"testing"

	hygieneapp "github.com/vrooli/vrooli/internal/app/hygiene"
	"github.com/vrooli/vrooli/internal/cliout"
)

func TestParseRequestSupportsDetailAndScopeFlags(t *testing.T) {
	req, err := ParseRequest([]string{"--details", "--plans-only", "--fail-on", "warning"})
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.OutputMode != OutputModeDetails {
		t.Fatalf("OutputMode = %q, want %q", req.OutputMode, OutputModeDetails)
	}
	if !req.PlansOnly {
		t.Fatalf("PlansOnly = false, want true")
	}
	if req.FailOn != hygieneapp.SeverityWarning {
		t.Fatalf("FailOn = %q, want warning", req.FailOn)
	}
}

func TestParseRequestRejectsConflictingModes(t *testing.T) {
	if _, err := ParseRequest([]string{"--summary", "--details"}); err == nil {
		t.Fatalf("ParseRequest accepted conflicting output modes")
	}
	if _, err := ParseRequest([]string{"--plans-only", "--contract-only"}); err == nil {
		t.Fatalf("ParseRequest accepted conflicting scope modes")
	}
	if _, err := ParseRequest([]string{"--drift-only", "--plans-only"}); err == nil {
		t.Fatalf("ParseRequest accepted conflicting drift+plans-only")
	}
	if _, err := ParseRequest([]string{"--drift-only", "--no-drift"}); err == nil {
		t.Fatalf("ParseRequest accepted drift-only with no-drift")
	}
	if _, err := ParseRequest([]string{"--pnpm-only", "--contract-only"}); err == nil {
		t.Fatalf("ParseRequest accepted conflicting pnpm+contract-only")
	}
}

func TestParseRequestSupportsPnpmOnly(t *testing.T) {
	req, err := ParseRequest([]string{"--pnpm-only", "--fix-safe"})
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if !req.PnpmOnly {
		t.Fatalf("PnpmOnly = false, want true")
	}
	if !req.FixSafe {
		t.Fatalf("FixSafe = false, want true")
	}
}

func TestRenderShowsConfigFixes(t *testing.T) {
	report := hygieneapp.Report{
		Root:        "/repo",
		Success:     true,
		ConfigFixes: []string{"restored canonical pnpm-workspace.yaml (comment block + workspace settings)"},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeDefault); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), "Config fixes applied:") {
		t.Fatalf("render missing config fixes section:\n%s", out.String())
	}
}

func TestRenderLabelsPlanManagerFixActions(t *testing.T) {
	report := hygieneapp.Report{
		Root:    "/repo",
		Success: true,
		FixesApplied: []hygieneapp.PlanFix{
			{Source: "docs/plans/a.md", Action: "imported", Plan: hygieneapp.HygienePlan{Path: "/repo/.vrooli/plans/a.md"}},
			{Source: "/repo/.vrooli/plans/b.md", Action: "mirror_repaired", Plan: hygieneapp.HygienePlan{Path: "/repo/.vrooli/plans/b.md"}},
		},
		PlanReconcileOutcomes: []hygieneapp.PlanReconcileOutcome{
			{Source: "plans/c.md", Action: "skipped_duplicate", Plan: hygieneapp.HygienePlan{Path: "/repo/.vrooli/plans/c.md"}, Mirror: hygieneapp.HygieneMirror{Status: "fresh"}, SourceUntouched: true},
		},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeDefault); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"- imported docs/plans/a.md -> /repo/.vrooli/plans/a.md",
		"- mirror repaired /repo/.vrooli/plans/b.md -> /repo/.vrooli/plans/b.md",
		"Plan reconcile results:",
		"- skipped duplicate plans/c.md -> /repo/.vrooli/plans/c.md [fresh] (source untouched)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("render missing %q:\n%s", want, text)
		}
	}
}

func TestParseRequestSupportsDriftFlags(t *testing.T) {
	req, err := ParseRequest([]string{"--drift-only"})
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if !req.DriftOnly {
		t.Fatalf("DriftOnly = false, want true")
	}
	req, err = ParseRequest([]string{"--no-drift"})
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if !req.NoDrift {
		t.Fatalf("NoDrift = false, want true")
	}
}

func TestParseRequestSupportsNoFreshness(t *testing.T) {
	req, err := ParseRequest([]string{"--no-freshness"})
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if !req.NoFreshness {
		t.Fatalf("NoFreshness = false, want true")
	}
}

func TestRenderIncludesDriftSummary(t *testing.T) {
	report := hygieneapp.Report{
		Root: "/repo",
		SharedDrift: &hygieneapp.DependencyFreshnessCompatReport{
			Clean:           false,
			Root:            "/repo",
			OnlyTouchedUsed: true,
			Scenarios: []hygieneapp.DependencyFreshnessScenario{
				{Path: "scenarios/foo", Status: hygieneapp.DependencyFreshnessStatusStaleModules, DiffPaths: []string{"scenarios/foo/api/go.sum"}},
				{Path: "scenarios/bar", Status: hygieneapp.DependencyFreshnessStatusClean},
			},
		},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeDefault); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"Shared-package drift: 2 scenarios checked (only-touched)",
		"Stale scenarios:",
		"scenarios/foo [stale-modules]",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in render:\n%s", want, text)
		}
	}
}

func TestRenderDefaultShowsFindingsPlanSummaryAndNextSteps(t *testing.T) {
	report := hygieneapp.Report{
		Success:          false,
		Root:             "/repo",
		BlockingFailures: 1,
		Warnings:         1,
		Findings: []hygieneapp.Finding{
			{
				Severity:   hygieneapp.SeverityError,
				Code:       "repo_contract_personal_absolute_paths",
				Locations:  []string{"internal/setup/example_test.go:12"},
				Message:    "personal absolute paths found",
				Why:        "Committed personal home paths make tests machine-specific.",
				Fixability: hygieneapp.FixabilityManual,
				NextActions: []hygieneapp.Action{{
					Code:    "remove_personal_paths",
					Message: "Replace personal absolute paths.",
				}},
			},
			{
				Severity:   hygieneapp.SeverityWarning,
				Code:       "plan_candidates",
				Message:    "12 likely scratch plan candidates found",
				Fixability: hygieneapp.FixabilityAutomatic,
				NextActions: []hygieneapp.Action{{
					Code:    "import_plan_candidates",
					Message: "Import untracked scratch plan candidates into user-scoped plan storage.",
					Command: "vrooli hygiene --fix-safe --plans",
				}},
			},
		},
		PlanCandidates: []hygieneapp.PlanCandidate{
			{Path: "docs/plans/a.md", Status: "untracked"},
			{Path: "docs/plans/b.md", Status: "modified"},
		},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeDefault); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"Blocking issues:",
		"repo_contract_personal_absolute_paths [manual]",
		"Locations: internal/setup/example_test.go:12",
		"Why: Committed personal home paths make tests machine-specific.",
		"Warnings:",
		"plan_candidates [automatic]",
		"Plan candidates: 2",
		"- 1 modified",
		"- 1 untracked",
		"Modified plan candidates:",
		"Next steps:",
		"vrooli hygiene --fix-safe --plans",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output missing %q:\n%s", want, text)
		}
	}
}

func TestRenderShowsInvalidLegacyPlanReasons(t *testing.T) {
	report := hygieneapp.Report{
		Success:  false,
		Root:     "/repo",
		Warnings: 1,
		Findings: []hygieneapp.Finding{{
			Severity:   hygieneapp.SeverityWarning,
			Code:       "invalid_legacy_plan_sources",
			Message:    "1 invalid legacy plan source(s) need guided remediation",
			Fixability: hygieneapp.FixabilityGuided,
			NextActions: []hygieneapp.Action{{
				Code:       "inspect_invalid_legacy_plans",
				Message:    "List the invalid legacy plan files and Plan Manager parser or conflict reason.",
				Command:    "vrooli hygiene --plans-only --details",
				Fixability: hygieneapp.FixabilityGuided,
			}},
		}},
		PlanCandidates: []hygieneapp.PlanCandidate{
			{Path: "docs/plans/broken.md", Status: "parse_failed", Reason: "missing phase heading"},
		},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeSummary); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"invalid_legacy_plan_sources [guided]",
		"Plan candidates: 1",
		"- 1 parse_failed",
		"Invalid legacy plan candidates:",
		"docs/plans/broken.md: missing phase heading",
		"vrooli hygiene --plans-only --details",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output missing %q:\n%s", want, text)
		}
	}
}

func TestRenderNextOnlySuppressesStatus(t *testing.T) {
	report := hygieneapp.Report{
		Root: "/repo",
		Findings: []hygieneapp.Finding{{
			Severity: hygieneapp.SeverityWarning,
			Code:     "plan_candidates",
			Message:  "plans found",
			NextActions: []hygieneapp.Action{{
				Code:    "import_plan_candidates",
				Command: "vrooli hygiene --fix-safe --plans",
			}},
		}},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeNext); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	if strings.Contains(text, "Status:") {
		t.Fatalf("--next output included status:\n%s", text)
	}
	if !strings.Contains(text, "vrooli hygiene --fix-safe --plans") {
		t.Fatalf("--next output missing next command:\n%s", text)
	}
}

func TestRenderNextListsAutomaticAndInvalidLegacyActions(t *testing.T) {
	report := hygieneapp.Report{
		Root: "/repo",
		Findings: []hygieneapp.Finding{
			{
				Severity: hygieneapp.SeverityWarning,
				Code:     "plan_manager_reconcile",
				Message:  "1 plan hygiene item(s) can be reconciled automatically",
				NextActions: []hygieneapp.Action{{
					Code:    "reconcile_plan_manager_plans",
					Message: "Ask Plan Manager to repair rendered mirrors, adopt parseable legacy plan files, and remove adopted/proven legacy sources.",
					Command: "vrooli hygiene --fix-safe --plans",
				}},
			},
			{
				Severity: hygieneapp.SeverityWarning,
				Code:     "invalid_legacy_plan_sources",
				Message:  "1 invalid legacy plan source(s) need guided remediation",
				NextActions: []hygieneapp.Action{
					{
						Code:    "inspect_invalid_legacy_plans",
						Message: "List the invalid legacy plan files and Plan Manager parser or conflict reason.",
						Command: "vrooli hygiene --plans-only --details",
					},
					{
						Code:    "preview_plan_manager_reconcile",
						Message: "Preview Plan Manager's authoritative reconcile report before and after repairing legacy plan markdown.",
						Command: "plan-manager plans reconcile --dry-run --workspace \"/repo\"",
					},
				},
			},
		},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeNext); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"vrooli hygiene --fix-safe --plans",
		"vrooli hygiene --plans-only --details",
		"plan-manager plans reconcile --dry-run --workspace \"/repo\"",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("--next output missing %q:\n%s", want, text)
		}
	}
}

func TestRenderFindingShowsEveryNextAction(t *testing.T) {
	report := hygieneapp.Report{
		Root:             "/repo",
		Success:          false,
		BlockingFailures: 1,
		Findings: []hygieneapp.Finding{{
			Severity:   hygieneapp.SeverityError,
			Code:       "dependency_freshness",
			Message:    "SDA reports stale and errored dependency surfaces",
			Fixability: hygieneapp.FixabilityGuided,
			NextActions: []hygieneapp.Action{
				{
					Code:       "apply_go_tidy",
					Message:    "Run SDA-owned package freshness repair for impacted Go surfaces.",
					Command:    "scenario-dependency-analyzer freshness --touched --apply",
					Fixability: hygieneapp.FixabilityAutomatic,
				},
				{
					Code:       "preview_missing_local_replaces",
					Message:    "Preview SDA-owned local replace reconciliation for errored Go surfaces.",
					Command:    "scenario-dependency-analyzer deps reconcile --all --json",
					Fixability: hygieneapp.FixabilityGuided,
				},
			},
		}},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeDefault); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"Next: scenario-dependency-analyzer freshness --touched --apply",
		"automatic - Run SDA-owned package freshness repair for impacted Go surfaces.",
		"Also: scenario-dependency-analyzer deps reconcile --all --json",
		"guided - Preview SDA-owned local replace reconciliation for errored Go surfaces.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output missing %q:\n%s", want, text)
		}
	}
}

func TestRenderDriftSummarySummarizesMissingReplaceErrors(t *testing.T) {
	report := hygieneapp.Report{
		Root: "/repo",
		SharedDrift: &hygieneapp.DependencyFreshnessCompatReport{
			Clean:           false,
			Root:            "/repo",
			OnlyTouchedUsed: true,
			Scenarios: []hygieneapp.DependencyFreshnessScenario{
				{
					Path:   "scenarios/demo",
					Status: hygieneapp.DependencyFreshnessStatusError,
					Error:  "exit status 1: reading github.com/vrooli/cli-core/go.mod at revision v0.0.0: ERROR: Repository not found.\nfull details",
				},
			},
		},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeDefault); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "missing local replace for an in-repo Go module") {
		t.Fatalf("rendered output missing concise missing-replace guidance:\n%s", text)
	}
	if strings.Contains(text, "full details") {
		t.Fatalf("default render leaked full multiline error:\n%s", text)
	}
}
