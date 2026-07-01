package hygiene

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/api-core/discovery"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

const planManagerScenario = "plan-manager"

type planManagerReconciler struct {
	client  *http.Client
	baseURL string
}

func NewDefaultPlanReconciler(ctx context.Context) (PlanReconciler, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, planManagerScenario)
	if err != nil {
		return nil, err
	}
	return planManagerReconciler{
		client:  &http.Client{Timeout: 10 * time.Second},
		baseURL: url,
	}, nil
}

func (r planManagerReconciler) ReconcilePlans(ctx context.Context, req PlanReconcileRequest) (PlanReconcileReport, error) {
	body, err := protojson.Marshal(&plansv1.ReconcilePlansRequest{
		DryRun:                 req.DryRun,
		RepairMirrors:          req.RepairMirrors,
		AdoptLegacy:            req.AdoptLegacy,
		IncludeArchived:        req.IncludeArchived,
		ConflictPolicy:         reconcileConflictPolicy(req.SkipExisting),
		SourceRuntimeHomePlans: true,
		SourceDocsPlans:        true,
		SourceRepoPlans:        true,
		Workspace:              planWorkspaceScope(req.WorkspaceRoot),
	})
	if err != nil {
		return PlanReconcileReport{}, err
	}
	url := r.baseURL + "/vrooli.plan_manager.v1.plans.PlansService/ReconcilePlans"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return PlanReconcileReport{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	resp, err := r.client.Do(httpReq)
	if err != nil {
		return PlanReconcileReport{}, fmt.Errorf("plan-manager reconcile at %s: %w", r.baseURL, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return PlanReconcileReport{}, fmt.Errorf("read plan-manager reconcile response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return PlanReconcileReport{}, fmt.Errorf("plan-manager reconcile at %s returned HTTP %d: %s", r.baseURL, resp.StatusCode, string(raw))
	}
	var msg plansv1.ReconcilePlansResponse
	if err := protojson.Unmarshal(raw, &msg); err != nil {
		return PlanReconcileReport{}, fmt.Errorf("decode plan-manager reconcile response: %w", err)
	}
	out := PlanReconcileReport{DryRun: msg.GetDryRun()}
	for _, item := range msg.GetItems() {
		if item == nil {
			continue
		}
		mirror := item.GetMirror()
		out.Items = append(out.Items, PlanReconcileItem{
			Action:          reconcileActionString(item.GetAction()),
			PlanID:          item.GetPlanId(),
			Slug:            item.GetSlug(),
			Title:           item.GetTitle(),
			SourcePath:      item.GetSourcePath(),
			MirrorPath:      mirror.GetPath(),
			MirrorStatus:    mirrorStatusString(mirror.GetStatus()),
			SourceUntouched: item.GetSourceUntouched(),
			Error:           item.GetError(),
		})
	}
	return out, nil
}

func planWorkspaceScope(root string) *plansv1.WorkspaceScope {
	if root == "" {
		return nil
	}
	return &plansv1.WorkspaceScope{Root: root}
}

func reconcileConflictPolicy(skipExisting bool) plansv1.ReconcileConflictPolicy {
	if skipExisting {
		return plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_SKIP_EXISTING
	}
	return plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_REPORT_ONLY
}

func reconcileActionString(action plansv1.ReconcileAction) string {
	switch action {
	case plansv1.ReconcileAction_RECONCILE_ACTION_ALREADY_CANONICAL:
		return "already_canonical"
	case plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_FRESH:
		return "mirror_fresh"
	case plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_REPAIR_NEEDED:
		return "mirror_repair_needed"
	case plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_REPAIRED:
		return "mirror_repaired"
	case plansv1.ReconcileAction_RECONCILE_ACTION_IMPORT_PLANNED:
		return "import_planned"
	case plansv1.ReconcileAction_RECONCILE_ACTION_IMPORTED:
		return "imported"
	case plansv1.ReconcileAction_RECONCILE_ACTION_SKIPPED_DUPLICATE:
		return "skipped_duplicate"
	case plansv1.ReconcileAction_RECONCILE_ACTION_PARSE_FAILED:
		return "parse_failed"
	case plansv1.ReconcileAction_RECONCILE_ACTION_CONFLICT:
		return "conflict"
	default:
		return ""
	}
}

func mirrorStatusString(status sharedv1.RenderedMirrorStatus) string {
	switch status {
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_FRESH:
		return "fresh"
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_MISSING:
		return "missing"
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_STALE:
		return "stale"
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_WRITE_FAILED:
		return "write_failed"
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_UNKNOWN:
		return "unknown"
	default:
		return ""
	}
}
