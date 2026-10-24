package safety

import (
	"context"
	"errors"
	"log"

	internalsafety "data-backup-manager/internal/safety"

	"connectrpc.com/connect"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety"
)

// Deps wires the seams the Connect safety handler needs. The service is
// constructed in main.go because it composes the destinations/plans/runs/targets
// services through adapters.
type Deps struct {
	Service internalsafety.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the safety Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) EnsureSafetyDestination(ctx context.Context, req *connect.Request[safetyv1.EnsureSafetyDestinationRequest]) (*connect.Response[safetyv1.EnsureSafetyDestinationResponse], error) {
	dest, created, err := h.deps.Service.EnsureSafetyDestination(ctx, req.Msg.CapBytes)
	if err != nil {
		return nil, h.translate("EnsureSafetyDestination", err)
	}
	return connect.NewResponse(&safetyv1.EnsureSafetyDestinationResponse{
		Destination: destinationToProto(dest),
		Created:     created,
	}), nil
}

func (h *connectHandler) BackupScenarioNow(ctx context.Context, req *connect.Request[safetyv1.BackupScenarioNowRequest]) (*connect.Response[safetyv1.BackupScenarioNowResponse], error) {
	res, err := h.deps.Service.BackupScenarioNow(ctx, req.Msg.Scenario, req.Msg.KeepLatest)
	if err != nil {
		return nil, h.translate("BackupScenarioNow", err)
	}
	return connect.NewResponse(&safetyv1.BackupScenarioNowResponse{
		RunId:         res.Run.ID,
		PlanId:        res.Run.PlanID,
		DestinationId: res.DestinationID,
		TargetCount:   int32(res.TargetCount),
		Status:        res.Run.Status,
	}), nil
}

func (h *connectHandler) RegisterScenarioTargets(ctx context.Context, req *connect.Request[safetyv1.RegisterScenarioTargetsRequest]) (*connect.Response[safetyv1.RegisterScenarioTargetsResponse], error) {
	res, err := h.deps.Service.RegisterScenarioTargets(ctx, req.Msg.Scenario)
	if err != nil {
		return nil, h.translate("RegisterScenarioTargets", err)
	}
	registered := make([]*safetyv1.RegisteredTarget, 0, len(res.Registered))
	for _, t := range res.Registered {
		registered = append(registered, &safetyv1.RegisteredTarget{
			Name:       t.Name,
			SourceKind: string(t.Kind),
			Locator:    t.Locator,
		})
	}
	skipped := make([]*safetyv1.SkippedTarget, 0, len(res.Skipped))
	for _, s := range res.Skipped {
		skipped = append(skipped, &safetyv1.SkippedTarget{SourceKind: s.Kind, Reason: s.Reason})
	}
	return connect.NewResponse(&safetyv1.RegisterScenarioTargetsResponse{
		Scenario:   res.Scenario,
		Registered: registered,
		Skipped:    skipped,
	}), nil
}

func (h *connectHandler) PopulateShadow(ctx context.Context, req *connect.Request[safetyv1.PopulateShadowRequest]) (*connect.Response[safetyv1.PopulateShadowResponse], error) {
	mappings := make([]internalsafety.ShadowMapping, 0, len(req.Msg.Mappings))
	for _, m := range req.Msg.Mappings {
		mappings = append(mappings, internalsafety.ShadowMapping{TargetName: m.GetTargetName(), Location: m.GetLocation()})
	}
	res, err := h.deps.Service.PopulateShadow(ctx, req.Msg.Scenario, req.Msg.RunId, mappings)
	if err != nil {
		return nil, h.translate("PopulateShadow", err)
	}
	restores := make([]*safetyv1.ShadowRestore, 0, len(res.Restores))
	for _, r := range res.Restores {
		restores = append(restores, &safetyv1.ShadowRestore{
			TargetName: r.TargetName,
			TargetId:   r.TargetID,
			SnapshotId: r.SnapshotID,
			RestoreId:  r.RestoreID,
			Location:   r.Location,
			Status:     r.Status,
		})
	}
	skipped := make([]*safetyv1.ShadowPopulateSkip, 0, len(res.Skipped))
	for _, s := range res.Skipped {
		skipped = append(skipped, &safetyv1.ShadowPopulateSkip{TargetName: s.TargetName, Reason: s.Reason})
	}
	return connect.NewResponse(&safetyv1.PopulateShadowResponse{
		Scenario: res.Scenario,
		RunId:    res.RunID,
		Restores: restores,
		Skipped:  skipped,
	}), nil
}

func destinationToProto(d internalsafety.DestinationRef) *safetyv1.SafetyDestination {
	return &safetyv1.SafetyDestination{
		Id:                 d.ID,
		Name:               d.Name,
		Location:           d.Location,
		RepositoryLocation: d.RepositoryLocation,
	}
}

// translate maps domain errors to Connect codes. The substrate's only typed
// error is ErrNoTargets (the caller must register the scenario's targets first);
// everything else is an upstream domain failure surfaced as Internal and logged.
func (h *connectHandler) translate(op string, err error) error {
	switch {
	case errors.Is(err, internalsafety.ErrNoTargets),
		errors.Is(err, internalsafety.ErrNoSafetyBackup),
		errors.Is(err, internalsafety.ErrRunNotTerminal):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		h.deps.Logger.Printf("safety.%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	}
}
