package fleet

import (
	"context"
	"log"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/fleet"

	"connectrpc.com/connect"

	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/fleet"
)

// dryRunHeader is the canonical header cli-core sets on `--dry-run`. RollFleet
// honours it by classifying every node then short-circuiting before creating a
// rollout or dispatching any provisioning op.
const dryRunHeader = "X-Dry-Run"

// Deps wires the seams the Connect fleet handler needs. Operator verbs are
// owner-gated via auth.RequireOwner.
type Deps struct {
	Service fleet.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the handler, defaulting the logger.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// RollFleet pins the fleet (or named subset) to a target revision. Owner-gated.
func (h *connectHandler) RollFleet(ctx context.Context, req *connect.Request[fleetv1.RollFleetRequest]) (*connect.Response[fleetv1.RollFleetResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	actor := owner.OwnerID
	if actor == "" {
		actor = owner.Email
	}
	if actor == "" {
		actor = "owner"
	}

	dryRun := req.Header().Get(dryRunHeader) == "true"

	dec, err := h.deps.Service.Roll(ctx, fleet.RollInput{
		Actor:          actor,
		TargetRevision: req.Msg.TargetRevision,
		NodeIDs:        req.Msg.NodeIds,
		TimeoutSeconds: req.Msg.TimeoutSeconds,
		DryRun:         dryRun,
	})
	if err != nil {
		connectErr := fleet.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("fleet.RollFleet(rev=%q): %v", req.Msg.TargetRevision, err)
		}
		return nil, connectErr
	}

	return connect.NewResponse(&fleetv1.RollFleetResponse{
		RolloutId: dec.RolloutID,
		DryRun:    dec.DryRun,
		Status:    rolloutStatusToProto(dec.Status),
		Results:   domainResultsToProto(dec.Results),
	}), nil
}

func (h *connectHandler) GetRollout(ctx context.Context, req *connect.Request[fleetv1.GetRolloutRequest]) (*connect.Response[fleetv1.GetRolloutResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	rollout, results, err := h.deps.Service.GetRollout(ctx, req.Msg.Id)
	if err != nil {
		connectErr := fleet.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("fleet.GetRollout(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&fleetv1.GetRolloutResponse{
		Rollout: domainRolloutToProto(rollout),
		Results: domainResultsToProto(results),
	}), nil
}

func (h *connectHandler) ListRollouts(ctx context.Context, req *connect.Request[fleetv1.ListRolloutsRequest]) (*connect.Response[fleetv1.ListRolloutsResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	list, err := h.deps.Service.ListRollouts(ctx, fleet.ListFilter{Limit: int(req.Msg.Limit)})
	if err != nil {
		h.deps.Logger.Printf("fleet.ListRollouts: %v", err)
		return nil, fleet.ToConnectError(err)
	}
	resp := &fleetv1.ListRolloutsResponse{Rollouts: make([]*fleetv1.Rollout, 0, len(list))}
	for _, r := range list {
		resp.Rollouts = append(resp.Rollouts, domainRolloutToProto(r))
	}
	return connect.NewResponse(resp), nil
}
