package hygiene

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
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
	if !reconciler.req.RepairMirrors || !reconciler.req.AdoptLegacy || reconciler.req.DryRun {
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
}

func TestPlansProviderFallsBackToStaticScanWhenPlanManagerUnavailable(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	writeFile(t, filepath.Join(root, "docs", "plans", "scratch-plan.md"), "# Scratch\n")
	reconciler := &fakePlanReconciler{err: errors.New("plan-manager unavailable")}

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
	reconciler := &fakePlanReconciler{err: errors.New("plan-manager unavailable")}

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
