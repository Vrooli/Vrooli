package hygiene

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	plansapp "github.com/vrooli/vrooli/internal/app/plans"
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
	SourceIntake    bool
	RetireSources   bool
	SkipExisting    bool
	IncludeArchived bool
	WorkspaceRoot   string
}

type PlanReconcileReport struct {
	DryRun bool
	Items  []PlanReconcileItem
}

type PlanReconcileItem struct {
	Action                  string
	PlanID                  string
	Slug                    string
	Title                   string
	SourcePath              string
	MirrorPath              string
	MirrorStatus            string
	SourceUntouched         bool
	SourceRetirementPlanned bool
	SourceRemoved           bool
	Error                   string
}

type plansProvider struct {
	root              string
	reconciler        PlanReconciler
	unavailableReason string
}

func (p plansProvider) ID() string { return plansProviderID }

func (p plansProvider) Run(ctx context.Context, req Request, report *Report) error {
	if p.reconciler == nil {
		reason := p.unavailableReason
		if reason == "" {
			reason = "plan-manager reconciler is not configured"
		}
		p.runStaticFallback(report, reason)
		return nil
	}
	runReq := PlanReconcileRequest{
		DryRun:          !(req.FixSafe && req.Plans),
		RepairMirrors:   true,
		SourceIntake:    true,
		RetireSources:   true,
		SkipExisting:    true,
		IncludeArchived: false,
		WorkspaceRoot:   p.root,
	}
	reconcile, err := p.reconciler.ReconcilePlans(ctx, runReq)
	if err != nil {
		if planManagerDegraded(err) {
			p.runStaticFallback(report, err.Error())
			return nil
		}
		p.reportCanonicalFailure(report, err)
		return nil
	}
	p.applyReconcile(report, req, reconcile)
	return nil
}

func planManagerDegraded(err error) bool {
	return errors.Is(err, plansapp.ErrPlanManagerUnavailable) || errors.Is(err, plansapp.ErrPlanManagerTimeout)
}

func (p plansProvider) reportCanonicalFailure(report *Report, err error) {
	message := fmt.Sprintf("plan-manager canonical reconcile failed: %v", err)
	action := Action{
		Code:       "inspect_plan_manager_reconcile",
		Message:    "Inspect the Plan Manager reconcile error and rerun hygiene after correcting the canonical failure.",
		Command:    "plan-manager plans reconcile --dry-run",
		Fixability: FixabilityGuided,
	}
	report.addFinding(Finding{
		Severity:    SeverityError,
		Code:        "plan_manager_reconcile_failed",
		Message:     message,
		Why:         "Plan Manager is reachable enough to return an authoritative failure; root hygiene must not replace that result with a static markdown scan.",
		Fixability:  FixabilityGuided,
		NextActions: []Action{action},
	})
	report.Actions = append(report.Actions, action)
	report.addCheck("plan_manager_reconcile", false, SeverityError, message)
}

func (p plansProvider) applyReconcile(report *Report, _ Request, reconcile PlanReconcileReport) {
	var automatic []PlanReconcileItem
	var invalidSources []PlanReconcileItem
	var fixes []PlanFix
	for _, item := range reconcile.Items {
		report.PlanReconcileOutcomes = append(report.PlanReconcileOutcomes, planOutcomeFromReconcile(item))
		if item.SourceRetirementPlanned {
			automatic = append(automatic, item)
			if item.SourcePath != "" {
				report.PlanCandidates = append(report.PlanCandidates, PlanCandidate{
					Path:   relPath(p.root, item.SourcePath),
					Status: "source_retirement_planned",
					Reason: "plan-manager reconcile reports the source is proven canonical/imported/duplicate and can be retired",
				})
			}
		} else {
			switch item.Action {
			case "import_planned", "mirror_repair_needed":
				automatic = append(automatic, item)
				if item.SourcePath != "" {
					report.PlanCandidates = append(report.PlanCandidates, PlanCandidate{
						Path:   relPath(p.root, item.SourcePath),
						Status: item.Action,
						Reason: "plan-manager reconcile reports source intake or mirror repair is needed",
					})
				}
			case "parse_failed", "conflict":
				invalidSources = append(invalidSources, item)
				if item.SourcePath != "" {
					report.PlanCandidates = append(report.PlanCandidates, PlanCandidate{
						Path:   relPath(p.root, item.SourcePath),
						Status: item.Action,
						Reason: invalidSourceReason(item),
					})
				}
			case "imported", "mirror_repaired", "already_canonical", "skipped_duplicate":
				if item.Error != "" {
					invalidSources = append(invalidSources, item)
				}
			}
		}
		switch item.Action {
		case "imported", "mirror_repaired":
			fixes = append(fixes, planFixFromReconcile(item))
		}
		if item.SourceRemoved {
			fixes = append(fixes, planFixFromReconcile(item))
		}
	}
	if len(fixes) > 0 {
		report.FixesApplied = append(report.FixesApplied, fixes...)
		report.addCheck("plan_manager_reconcile", true, SeverityInfo, fmt.Sprintf("reconciled %d plan item(s)", len(fixes)))
	}
	if len(automatic) == 0 && len(invalidSources) == 0 {
		report.addCheck("plan_manager_reconcile", true, SeverityInfo, "plan-manager reports no misplaced or stale plans")
		return
	}
	if len(automatic) > 0 {
		message := fmt.Sprintf("%d plan hygiene item(s) can be reconciled automatically", len(automatic))
		action := Action{
			Code:       "reconcile_plan_manager_plans",
			Message:    "Ask Plan Manager to repair rendered mirrors, canonicalize parseable plan source files, and retire proven sources.",
			Command:    "vrooli hygiene --fix-safe --plans",
			Fixability: FixabilityAutomatic,
		}
		report.addFinding(Finding{
			Severity:    SeverityWarning,
			Code:        "plan_manager_reconcile",
			Message:     message,
			Why:         "Plan Manager is the canonical plan authority; parseable markdown sources should be canonicalized into structured records, stale mirrors should be repaired from SQLite, and proven sources can be retired.",
			Locations:   reconcileLocations(automatic),
			Fixability:  FixabilityAutomatic,
			NextActions: []Action{action},
		})
		report.Actions = append(report.Actions, action)
		report.addCheck("plan_manager_reconcile", true, SeverityWarning, message)
	}
	if len(invalidSources) > 0 {
		message := fmt.Sprintf("%d invalid plan source(s) need guided remediation", len(invalidSources))
		actions := []Action{
			{
				Code:       "inspect_invalid_plan_sources",
				Message:    "List invalid plan source files and the Plan Manager parser or conflict reason.",
				Command:    "vrooli hygiene --plans-only --details",
				Fixability: FixabilityGuided,
			},
			{
				Code:       "preview_plan_manager_reconcile",
				Message:    "Preview Plan Manager's authoritative reconcile report before and after repairing plan source markdown.",
				Command:    reconcileDryRunCommand(p.root),
				Fixability: FixabilityGuided,
			},
			{
				Code:       "repair_invalid_plan_sources",
				Message:    "Repair each parse_failed file into importable plan markdown or move non-plan notes out of plan source locations, then rerun hygiene; safe-fix will only retire sources after Plan Manager can import or prove them canonical.",
				Fixability: FixabilityGuided,
			},
		}
		report.addFinding(Finding{
			Severity:    SeverityWarning,
			Code:        "invalid_plan_sources",
			Message:     message,
			Why:         "Plan Manager leaves parse failures and conflicts untouched so automatic cleanup cannot delete unimportable or ambiguous plan source files.",
			Locations:   reconcileLocations(invalidSources),
			Fixability:  FixabilityGuided,
			NextActions: actions,
		})
		report.Actions = append(report.Actions, actions...)
		report.addCheck("plan_manager_invalid_sources", true, SeverityWarning, message)
	}
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
		Message:    "Start Plan Manager and rerun hygiene so it can verify and canonicalize plan sources without creating markdown-only truth.",
		Command:    "vrooli scenario start plan-manager && vrooli hygiene --plans-only",
		Fixability: FixabilityGuided,
	}
	report.addFinding(Finding{
		Severity:   SeverityWarning,
		Code:       "plan_candidates",
		Message:    message,
		Why:        "Static hygiene can spot likely misplaced plan files, but only Plan Manager can verify canonical mirrors, already-imported sources, duplicates, and parse failures. " + reason,
		Fixability: FixabilityGuided,
		NextActions: []Action{
			action,
			{
				Code:       "preview_plan_manager_reconcile",
				Message:    "Preview Plan Manager's authoritative source-intake and mirror-repair report.",
				Command:    "plan-manager plans reconcile --dry-run",
				Fixability: FixabilityGuided,
			},
		},
	})
	report.Actions = append(report.Actions, action)
	report.addCheck("plan_candidates", true, SeverityWarning, message)
}

func planFixFromReconcile(item PlanReconcileItem) PlanFix {
	action := item.Action
	if item.SourceRemoved {
		action = "source_removed"
	}
	return PlanFix{
		Source: item.SourcePath,
		Action: action,
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

func planOutcomeFromReconcile(item PlanReconcileItem) PlanReconcileOutcome {
	return PlanReconcileOutcome{
		Source: item.SourcePath,
		Action: item.Action,
		Plan: HygienePlan{
			ID:     item.PlanID,
			Title:  item.Title,
			Slug:   item.Slug,
			Path:   item.MirrorPath,
			Source: item.SourcePath,
		},
		Mirror:                  HygieneMirror{Path: item.MirrorPath, Status: item.MirrorStatus},
		SourceUntouched:         item.SourceUntouched,
		SourceRetirementPlanned: item.SourceRetirementPlanned,
		SourceRemoved:           item.SourceRemoved,
		Error:                   item.Error,
	}
}

func invalidSourceReason(item PlanReconcileItem) string {
	if item.Error != "" {
		return item.Error
	}
	switch item.Action {
	case "parse_failed":
		return "plan-manager could not parse this plan source markdown"
	case "conflict":
		return "plan-manager found a conflict that needs manual review"
	default:
		return "plan-manager reported an invalid plan source"
	}
}

func reconcileDryRunCommand(root string) string {
	if root == "" {
		return "plan-manager plans reconcile --dry-run"
	}
	return fmt.Sprintf("plan-manager plans reconcile --dry-run --workspace %q", root)
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
