package hygiene

import (
	"context"
	"fmt"
	"path/filepath"
)

const plansProviderID = "plans"

// PlanReconciler is the Plan Manager authority seam for plan hygiene. The
// default implementation calls Plan Manager's ReconcilePlans RPC; tests can
// substitute a fake, and offline fallback remains read-only.
type PlanReconciler interface {
	ReconcilePlans(ctx context.Context, req PlanReconcileRequest) (PlanReconcileReport, error)
}

type PlanReconcileRequest struct {
	DryRun          bool
	RepairMirrors   bool
	AdoptLegacy     bool
	SkipExisting    bool
	IncludeArchived bool
	WorkspaceRoot   string
}

type PlanReconcileReport struct {
	DryRun bool
	Items  []PlanReconcileItem
}

type PlanReconcileItem struct {
	Action          string
	PlanID          string
	Slug            string
	Title           string
	SourcePath      string
	MirrorPath      string
	MirrorStatus    string
	SourceUntouched bool
	Error           string
}

type plansProvider struct {
	root       string
	reconciler PlanReconciler
}

func (p plansProvider) ID() string { return plansProviderID }

func (p plansProvider) Run(ctx context.Context, req Request, report *Report) error {
	if p.reconciler == nil {
		p.runStaticFallback(report, "plan-manager reconciler is not configured")
		return nil
	}
	runReq := PlanReconcileRequest{
		DryRun:          !(req.FixSafe && req.Plans),
		RepairMirrors:   true,
		AdoptLegacy:     true,
		SkipExisting:    true,
		IncludeArchived: false,
		WorkspaceRoot:   p.root,
	}
	reconcile, err := p.reconciler.ReconcilePlans(ctx, runReq)
	if err != nil {
		p.runStaticFallback(report, err.Error())
		return nil
	}
	p.applyReconcile(report, req, reconcile)
	return nil
}

func (p plansProvider) applyReconcile(report *Report, req Request, reconcile PlanReconcileReport) {
	var actionable []PlanReconcileItem
	var parseOrConflict []PlanReconcileItem
	var fixes []PlanFix
	for _, item := range reconcile.Items {
		switch item.Action {
		case "import_planned", "mirror_repair_needed":
			actionable = append(actionable, item)
			if item.SourcePath != "" {
				report.PlanCandidates = append(report.PlanCandidates, PlanCandidate{
					Path:   relPath(p.root, item.SourcePath),
					Status: item.Action,
					Reason: "plan-manager reconcile reports legacy adoption or mirror repair is needed",
				})
			}
		case "parse_failed", "conflict":
			parseOrConflict = append(parseOrConflict, item)
			if item.SourcePath != "" {
				report.PlanCandidates = append(report.PlanCandidates, PlanCandidate{
					Path:   relPath(p.root, item.SourcePath),
					Status: item.Action,
					Reason: item.Error,
				})
			}
		case "imported", "mirror_repaired":
			fixes = append(fixes, planFixFromReconcile(item))
		}
	}
	if len(fixes) > 0 {
		report.FixesApplied = append(report.FixesApplied, fixes...)
		report.addCheck("plan_manager_reconcile", true, SeverityInfo, fmt.Sprintf("reconciled %d plan item(s)", len(fixes)))
	}
	if len(actionable) == 0 && len(parseOrConflict) == 0 {
		report.addCheck("plan_manager_reconcile", true, SeverityInfo, "plan-manager reports no misplaced or stale plans")
		return
	}
	message := fmt.Sprintf("%d plan hygiene item(s) need reconcile", len(actionable)+len(parseOrConflict))
	severity := SeverityWarning
	fixability := FixabilityAutomatic
	if len(parseOrConflict) > 0 {
		fixability = FixabilityGuided
	}
	action := Action{
		Code:       "reconcile_plan_manager_plans",
		Message:    "Ask Plan Manager to repair rendered mirrors and adopt legacy plan files.",
		Command:    "vrooli hygiene --fix-safe --plans",
		Fixability: FixabilityAutomatic,
	}
	report.addFinding(Finding{
		Severity:    severity,
		Code:        "plan_manager_reconcile",
		Message:     message,
		Why:         "Plan Manager is the canonical plan authority; legacy markdown sources should be adopted into structured records and mirrors should be repaired from SQLite.",
		Locations:   reconcileLocations(append(actionable, parseOrConflict...)),
		Fixability:  fixability,
		NextActions: []Action{action},
	})
	report.Actions = append(report.Actions, action)
	report.addCheck("plan_manager_reconcile", true, severity, message)
}

func (p plansProvider) runStaticFallback(report *Report, reason string) {
	candidates, err := DetectPlanCandidates(p.root)
	if err != nil {
		report.addCheck("plan_candidates", false, SeverityError, err.Error())
		report.addFinding(Finding{
			Severity:   SeverityError,
			Code:       "plan_candidate_scan",
			Message:    err.Error(),
			Fixability: FixabilityManual,
			NextActions: []Action{{
				Code:    "inspect_plan_candidate_scan",
				Message: "Inspect the plan-candidate scan error and rerun hygiene.",
			}},
		})
		return
	}
	report.PlanCandidates = candidates
	message := "plan-manager unavailable; canonical plan hygiene verification was skipped"
	if len(candidates) > 0 {
		message = fmt.Sprintf("plan-manager unavailable; static scan found %d likely scratch plan candidates", len(candidates))
	}
	action := Action{
		Code:       "rerun_plan_manager_reconcile",
		Message:    "Start Plan Manager and rerun hygiene so it can verify/adopt plans without creating markdown-only truth.",
		Command:    "vrooli scenario start plan-manager && vrooli hygiene --plans-only",
		Fixability: FixabilityGuided,
	}
	report.addFinding(Finding{
		Severity:   SeverityWarning,
		Code:       "plan_candidates",
		Message:    message,
		Why:        "Static hygiene can spot likely misplaced plan files, but only Plan Manager can verify canonical mirrors, already-adopted sources, duplicates, and parse failures. " + reason,
		Fixability: FixabilityGuided,
		NextActions: []Action{
			action,
			{
				Code:       "preview_plan_manager_reconcile",
				Message:    "Preview Plan Manager's authoritative adoption and mirror-repair report.",
				Command:    "plan-manager plans reconcile --dry-run",
				Fixability: FixabilityGuided,
			},
		},
	})
	report.Actions = append(report.Actions, action)
	report.addCheck("plan_candidates", true, SeverityWarning, message)
}

func planFixFromReconcile(item PlanReconcileItem) PlanFix {
	return PlanFix{
		Source: item.SourcePath,
		Action: item.Action,
		Plan: HygienePlan{
			ID:     item.PlanID,
			Title:  item.Title,
			Slug:   item.Slug,
			Path:   item.MirrorPath,
			Source: item.SourcePath,
		},
		Mirror: HygieneMirror{Path: item.MirrorPath, Status: item.MirrorStatus},
	}
}

func reconcileLocations(items []PlanReconcileItem) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range items {
		loc := item.SourcePath
		if loc == "" {
			loc = item.MirrorPath
		}
		if loc == "" || seen[loc] {
			continue
		}
		seen[loc] = true
		out = append(out, loc)
	}
	return out
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == "" || rel[0] == '.' {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
