package hygiene

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/demand"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/api-core/discovery"
	plansapp "github.com/vrooli/vrooli/internal/app/plans"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

const (
	planmanagerReconcilerConflict         = "conflict"
	planmanagerReconcilerImported         = "imported"
	planmanagerReconcilerMirrorRepaired   = "mirror_repaired"
	planmanagerReconcilerParseFailed      = "parse_failed"
	planmanagerReconcilerSkippedDuplicate = "skipped_duplicate"
)

const planManagerScenario = "plan-manager"

var planManagerReconcileDemandSequence uint64

type planManagerReconciler struct {
	client           *http.Client
	baseURL          string
	demand           demand.LeaseClient
	demandConsumerID string
	service          plansconnect.PlansServiceClient
	resolveURL       func(context.Context) (string, error)
}

func NewDefaultPlanReconciler(ctx context.Context) (PlanReconciler, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, planManagerScenario)
	if err != nil {
		return nil, fmt.Errorf("%w: discover %s: %v", plansapp.ErrPlanManagerUnavailable, planManagerScenario, err)
	}
	url = strings.TrimRight(url, "/")
	client := &http.Client{Timeout: tuning.ControlPlaneClientTimeout()}
	return planManagerReconciler{
		client:           client,
		baseURL:          url,
		demand:           demand.Client{},
		demandConsumerID: "control-plane:hygiene-plan-reconciler",
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, planManagerScenario)
		},
	}, nil
}

func (r planManagerReconciler) ReconcilePlans(ctx context.Context, req PlanReconcileRequest) (PlanReconcileReport, error) {
	baseURL := r.baseURL
	if r.resolveURL != nil {
		resolved, err := r.resolveURL(ctx)
		if err != nil {
			return PlanReconcileReport{}, fmt.Errorf("%w: resolve %s: %w", plansapp.ErrPlanManagerUnavailable, planManagerScenario, err)
		}
		baseURL = strings.TrimRight(resolved, "/")
	}
	release, err := r.acquireDemand(ctx, "reconcile-plans")
	if err != nil {
		return PlanReconcileReport{}, err
	}
	if release != nil {
		defer release()
	}
	request := connect.NewRequest(&plansv1.ReconcilePlansRequest{
		DryRun:                 req.DryRun,
		RepairMirrors:          req.RepairMirrors,
		SourceIntake:           req.SourceIntake,
		RetireSources:          req.RetireSources,
		IncludeArchived:        req.IncludeArchived,
		ConflictPolicy:         reconcileConflictPolicy(req.SkipExisting),
		SourceRuntimeHomePlans: true,
		SourceDocsPlans:        true,
		SourceRepoPlans:        true,
		Workspace:              planWorkspaceScope(req.WorkspaceRoot),
	})
	service := r.service
	if r.resolveURL != nil {
		client := r.client
		if client == nil {
			client = http.DefaultClient
		}
		service = plansconnect.NewPlansServiceClient(client, baseURL)
	}
	if service == nil {
		client := r.client
		if client == nil {
			client = http.DefaultClient
		}
		service = plansconnect.NewPlansServiceClient(client, baseURL)
	}
	resp, err := service.ReconcilePlans(ctx, request)
	if err != nil {
		return PlanReconcileReport{}, plansapp.ClassifyPlanManagerConnectError(err)
	}
	out := PlanReconcileReport{DryRun: resp.Msg.GetDryRun()}
	for _, item := range resp.Msg.GetItems() {
		// A nil protobuf list entry round-trips as an empty message. Skip the
		// unspecified sentinel while preserving any future non-zero enum value.
		if item == nil || item.GetAction() == plansv1.ReconcileAction_RECONCILE_ACTION_UNSPECIFIED {
			continue
		}
		mirror := item.GetMirror()
		out.Items = append(out.Items, PlanReconcileItem{
			Action:                  reconcileActionString(item.GetAction()),
			PlanID:                  item.GetPlanId(),
			Slug:                    item.GetSlug(),
			Title:                   item.GetTitle(),
			SourcePath:              item.GetSourcePath(),
			MirrorPath:              mirror.GetPath(),
			MirrorStatus:            mirrorStatusString(mirror.GetStatus()),
			SourceUntouched:         item.GetSourceUntouched(),
			SourceRetirementPlanned: item.GetSourceRetirementPlanned(),
			SourceRemoved:           item.GetSourceRemoved(),
			Error:                   item.GetError(),
		})
	}
	return out, nil
}

func (r planManagerReconciler) acquireDemand(ctx context.Context, requestID string) (func(), error) {
	if r.demand == nil {
		return nil, nil
	}
	consumerID := strings.TrimSpace(r.demandConsumerID)
	if consumerID == "" {
		consumerID = "control-plane:hygiene-plan-reconciler"
	}
	requestID = fmt.Sprintf("%s:%d", requestID, atomic.AddUint64(&planManagerReconcileDemandSequence, 1))
	_, release, err := demand.AcquireScoped(ctx, r.demand, demand.AcquireRequest{
		LeaseID:    demand.StableLeaseID(consumerID, planManagerScenario, requestID),
		Scenario:   planManagerScenario,
		ConsumerID: consumerID,
		Kind:       demand.KindDependency,
		RequestID:  requestID,
		Metadata:   "owner=hygiene-plan-reconciler",
	}, "plan-manager reconciliation complete")
	if err != nil {
		return nil, fmt.Errorf("acquire plan-manager demand lease: %w", err)
	}
	return release, nil
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
		return planmanagerReconcilerMirrorRepaired
	case plansv1.ReconcileAction_RECONCILE_ACTION_IMPORT_PLANNED:
		return "import_planned"
	case plansv1.ReconcileAction_RECONCILE_ACTION_IMPORTED:
		return planmanagerReconcilerImported
	case plansv1.ReconcileAction_RECONCILE_ACTION_SKIPPED_DUPLICATE:
		return planmanagerReconcilerSkippedDuplicate
	case plansv1.ReconcileAction_RECONCILE_ACTION_PARSE_FAILED:
		return planmanagerReconcilerParseFailed
	case plansv1.ReconcileAction_RECONCILE_ACTION_CONFLICT:
		return planmanagerReconcilerConflict
	default:
		return fmt.Sprintf("unknown(%d)", action)
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
		return fmt.Sprintf("unknown(%d)", status)
	}
}
