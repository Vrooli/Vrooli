package hygiene

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	plansapp "github.com/vrooli/vrooli/internal/app/plans"
)

type fakePlanReconciler struct {
	req    PlanReconcileRequest
	report PlanReconcileReport
	err    error
}

func (f *fakePlanReconciler) ReconcilePlans(_ context.Context, req PlanReconcileRequest) (PlanReconcileReport, error) {
	f.req = req
	return f.report, f.err
}

func TestPlansProviderUsesPlanManagerReconcileForFixes(t *testing.T) {
	root := t.TempDir()
	reconciler := &fakePlanReconciler{report: PlanReconcileReport{
		Items: []PlanReconcileItem{{
			Action:       "imported",
			PlanID:       "plan-1",
			Slug:         "legacy",
			Title:        "Legacy",
			SourcePath:   filepath.Join(root, "docs", "plans", "legacy.md"),
			MirrorPath:   filepath.Join(root, ".vrooli", "plans", "legacy.md"),
			MirrorStatus: "fresh",
		}},
	}}
	report, err := Service{Root: root, Home: root, PlanReconciler: reconciler}.Run(Request{
		FixSafe:      true,
		Plans:        true,
		FailOn:       SeverityError,
		IncludePlans: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !reconciler.req.RepairMirrors || !reconciler.req.SourceIntake || !reconciler.req.RetireSources || reconciler.req.DryRun {
		t.Fatalf("reconcile request = %#v, want execute repair+adopt", reconciler.req)
	}
	if reconciler.req.WorkspaceRoot != root {
		t.Fatalf("workspace root = %q, want %q", reconciler.req.WorkspaceRoot, root)
	}
	if len(report.FixesApplied) != 1 {
		t.Fatalf("FixesApplied = %#v, want one imported fix", report.FixesApplied)
	}
	if got := report.FixesApplied[0].Plan.ID; got != "plan-1" {
		t.Fatalf("fix plan id = %q, want plan-1", got)
	}
	if got := report.FixesApplied[0].Action; got != "imported" {
		t.Fatalf("fix action = %q, want imported", got)
	}
	if got := report.FixesApplied[0].Mirror.Status; got != "fresh" {
		t.Fatalf("fix mirror status = %q, want fresh", got)
	}
}

func TestPlansProviderRecordsSkippedDuplicateAsReconcileOutcome(t *testing.T) {
	root := t.TempDir()
	reconciler := &fakePlanReconciler{report: PlanReconcileReport{
		Items: []PlanReconcileItem{{
			Action:       "skipped_duplicate",
			PlanID:       "plan-1",
			Slug:         "legacy",
			Title:        "Legacy",
			SourcePath:   filepath.Join(root, "plans", "legacy.md"),
			MirrorPath:   filepath.Join(root, ".vrooli", "plans", "legacy.md"),
			MirrorStatus: "fresh",
		}},
	}}
	report, err := Service{Root: root, Home: root, PlanReconciler: reconciler}.Run(Request{
		FixSafe:      true,
		Plans:        true,
		FailOn:       SeverityError,
		IncludePlans: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.FixesApplied) != 0 {
		t.Fatalf("FixesApplied = %#v, want skipped duplicate to remain a no-op outcome", report.FixesApplied)
	}
	if len(report.PlanReconcileOutcomes) != 1 {
		t.Fatalf("PlanReconcileOutcomes = %#v, want skipped duplicate result", report.PlanReconcileOutcomes)
	}
	if got := report.PlanReconcileOutcomes[0].Action; got != "skipped_duplicate" {
		t.Fatalf("outcome action = %q, want skipped_duplicate", got)
	}
}

func TestPlansProviderReportsSourceRetirementPlannedAsActionable(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "docs", "plans", "legacy.md")
	reconciler := &fakePlanReconciler{report: PlanReconcileReport{
		Items: []PlanReconcileItem{{
			Action:                  "already_canonical",
			PlanID:                  "plan-1",
			Slug:                    "legacy",
			Title:                   "Legacy",
			SourcePath:              source,
			SourceUntouched:         true,
			SourceRetirementPlanned: true,
		}},
	}}
	report, err := Service{Root: root, Home: root, PlanReconciler: reconciler}.Run(Request{
		FailOn:       SeverityError,
		IncludePlans: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !reconciler.req.DryRun || !reconciler.req.RetireSources {
		t.Fatalf("reconcile request = %#v, want dry-run cleanup preview", reconciler.req)
	}
	if len(report.PlanCandidates) != 1 || report.PlanCandidates[0].Status != "source_retirement_planned" {
		t.Fatalf("PlanCandidates = %#v, want cleanup-planned candidate", report.PlanCandidates)
	}
	if len(report.Actions) == 0 || report.Actions[0].Command != "vrooli hygiene --fix-safe --plans" {
		t.Fatalf("Actions = %#v, want fix-safe plans command", report.Actions)
	}
}

func TestPlansProviderSplitsAutomaticAndInvalidLegacyGuidance(t *testing.T) {
	root := t.TempDir()
	cleanupSource := filepath.Join(root, "docs", "plans", "legacy.md")
	invalidSource := filepath.Join(root, "docs", "plans", "broken.md")
	reconciler := &fakePlanReconciler{report: PlanReconcileReport{
		Items: []PlanReconcileItem{
			{
				Action:                  "already_canonical",
				PlanID:                  "plan-1",
				Slug:                    "legacy",
				Title:                   "Legacy",
				SourcePath:              cleanupSource,
				SourceRetirementPlanned: true,
			},
			{
				Action:     "parse_failed",
				SourcePath: invalidSource,
				Error:      "missing phase heading",
			},
		},
	}}
	report, err := Service{Root: root, Home: root, PlanReconciler: reconciler}.Run(Request{
		FailOn:       SeverityError,
		IncludePlans: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !hasFinding(report.Findings, "plan_manager_reconcile", FixabilityAutomatic) {
		t.Fatalf("Findings = %#v, want automatic reconcile finding", report.Findings)
	}
	if !hasFinding(report.Findings, "invalid_plan_sources", FixabilityGuided) {
		t.Fatalf("Findings = %#v, want guided invalid source finding", report.Findings)
	}
	if !hasActionCommand(report.Actions, "vrooli hygiene --fix-safe --plans") {
		t.Fatalf("Actions = %#v, want automatic safe-fix action", report.Actions)
	}
	if !hasActionCommand(report.Actions, "vrooli hygiene --plans-only --details") {
		t.Fatalf("Actions = %#v, want invalid source inspection action", report.Actions)
	}
	if !hasCandidate(report.PlanCandidates, "source_retirement_planned", "docs/plans/legacy.md") {
		t.Fatalf("PlanCandidates = %#v, want cleanup-planned candidate", report.PlanCandidates)
	}
	if !hasCandidate(report.PlanCandidates, "parse_failed", "docs/plans/broken.md") {
		t.Fatalf("PlanCandidates = %#v, want parse-failed candidate", report.PlanCandidates)
	}
}

func TestPlansProviderParseOnlyReportDoesNotSuggestSafeFix(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "docs", "plans", "broken.md")
	reconciler := &fakePlanReconciler{report: PlanReconcileReport{
		Items: []PlanReconcileItem{{
			Action:     "parse_failed",
			SourcePath: source,
			Error:      "front matter is invalid",
		}},
	}}
	report, err := Service{Root: root, Home: root, PlanReconciler: reconciler}.Run(Request{
		FailOn:       SeverityError,
		IncludePlans: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if hasFinding(report.Findings, "plan_manager_reconcile", FixabilityAutomatic) {
		t.Fatalf("Findings = %#v, did not want automatic reconcile finding", report.Findings)
	}
	if hasActionCommand(report.Actions, "vrooli hygiene --fix-safe --plans") {
		t.Fatalf("Actions = %#v, did not want safe-fix action for parse-only report", report.Actions)
	}
	if !hasFinding(report.Findings, "invalid_plan_sources", FixabilityGuided) {
		t.Fatalf("Findings = %#v, want guided invalid source finding", report.Findings)
	}
	if len(report.PlanCandidates) != 1 || report.PlanCandidates[0].Reason != "front matter is invalid" {
		t.Fatalf("PlanCandidates = %#v, want parse error reason", report.PlanCandidates)
	}
}

func TestPlansProviderRecordsSourceRemovedAsFix(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "docs", "plans", "legacy.md")
	reconciler := &fakePlanReconciler{report: PlanReconcileReport{
		Items: []PlanReconcileItem{{
			Action:        "already_canonical",
			PlanID:        "plan-1",
			Slug:          "legacy",
			Title:         "Legacy",
			SourcePath:    source,
			SourceRemoved: true,
		}},
	}}
	report, err := Service{Root: root, Home: root, PlanReconciler: reconciler}.Run(Request{
		FixSafe:      true,
		Plans:        true,
		FailOn:       SeverityError,
		IncludePlans: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.FixesApplied) != 1 {
		t.Fatalf("FixesApplied = %#v, want source removal fix", report.FixesApplied)
	}
	if got := report.FixesApplied[0].Action; got != "source_removed" {
		t.Fatalf("fix action = %q, want source_removed", got)
	}
}

func TestPlansProviderFallsBackToStaticScanWhenPlanManagerUnavailable(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	writeFile(t, filepath.Join(root, "docs", "plans", "scratch-plan.md"), "# Scratch\n")
	reconciler := &fakePlanReconciler{err: plansapp.ErrPlanManagerUnavailable}

	report, err := Service{Root: root, Home: root, PlanReconciler: reconciler}.Run(Request{
		FailOn:       SeverityError,
		IncludePlans: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.PlanCandidates) != 1 {
		t.Fatalf("PlanCandidates = %#v, want one static fallback candidate", report.PlanCandidates)
	}
	if len(report.FixesApplied) != 0 {
		t.Fatalf("fallback must not create offline plan fixes: %#v", report.FixesApplied)
	}
	if len(report.Actions) == 0 || report.Actions[0].Command == "" {
		t.Fatalf("expected actionable rerun command, got %#v", report.Actions)
	}
}

func TestPlansProviderWarnsWhenPlanManagerUnavailableWithoutCandidates(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	reconciler := &fakePlanReconciler{err: plansapp.ErrPlanManagerUnavailable}

	report, err := Service{Root: root, Home: root, PlanReconciler: reconciler}.Run(Request{
		FailOn:       SeverityError,
		IncludePlans: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.PlanCandidates) != 0 {
		t.Fatalf("PlanCandidates = %#v, want none", report.PlanCandidates)
	}
	if len(report.Findings) != 1 || report.Findings[0].Severity != SeverityWarning {
		t.Fatalf("Findings = %#v, want warning for skipped verification", report.Findings)
	}
	if len(report.Actions) == 0 || report.Actions[0].Command != "vrooli scenario start plan-manager && vrooli hygiene --plans-only" {
		t.Fatalf("Actions = %#v, want plan-manager start action", report.Actions)
	}
}

func TestPlansProviderReportsCanonicalPlanManagerFailure(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	writeFile(t, filepath.Join(root, "docs", "plans", "scratch-plan.md"), "# Scratch\n")
	reconciler := &fakePlanReconciler{err: plansapp.ErrPlanManagerInvalid}

	report, err := Service{Root: root, Home: root, PlanReconciler: reconciler}.Run(Request{
		FailOn:       SeverityError,
		IncludePlans: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(report.PlanCandidates) != 0 {
		t.Fatalf("PlanCandidates = %#v, want no static fallback candidates for canonical failure", report.PlanCandidates)
	}
	if len(report.Findings) != 1 || report.Findings[0].Code != "plan_manager_reconcile_failed" {
		t.Fatalf("Findings = %#v, want canonical reconcile failure", report.Findings)
	}
	if !strings.Contains(report.Findings[0].Message, "plan-manager canonical reconcile failed") {
		t.Fatalf("finding message = %q, want canonical failure reason", report.Findings[0].Message)
	}
	if len(report.FixesApplied) != 0 {
		t.Fatalf("canonical failure must not create fixes: %#v", report.FixesApplied)
	}
}

func hasFinding(findings []Finding, code string, fixability Fixability) bool {
	for _, finding := range findings {
		if finding.Code == code && finding.Fixability == fixability {
			return true
		}
	}
	return false
}

func hasActionCommand(actions []Action, command string) bool {
	for _, action := range actions {
		if action.Command == command {
			return true
		}
	}
	return false
}

func hasCandidate(candidates []PlanCandidate, status, path string) bool {
	for _, candidate := range candidates {
		if candidate.Status == status && candidate.Path == path {
			return true
		}
	}
	return false
}
