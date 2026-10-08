package gate

import (
	"context"
	"log"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/gate"

	"connectrpc.com/connect"

	gatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate"
)

// dryRunHeader is the canonical header cli-core sets on `--dry-run`. RunGate
// honours it by selecting + classifying a node per target OS then short-
// circuiting before creating a gate or dispatching any validation run.
const dryRunHeader = "X-Dry-Run"

// Deps wires the seams the Connect gate handler needs. Operator verbs are
// owner-gated via auth.RequireOwner.
type Deps struct {
	Service gate.Service
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

// RunGate fans a scenario's validation out across the target OSes. Owner-gated.
func (h *connectHandler) RunGate(ctx context.Context, req *connect.Request[gatev1.RunGateRequest]) (*connect.Response[gatev1.RunGateResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	actor := ownerActor(owner)

	dryRun := req.Header().Get(dryRunHeader) == "true"

	dec, err := h.deps.Service.Run(ctx, gate.RunInput{
		Actor:          actor,
		Scenario:       req.Msg.Scenario,
		TargetRevision: req.Msg.TargetRevision,
		TargetOSes:     req.Msg.TargetOses,
		Verb:           req.Msg.Verb,
		Args:           req.Msg.Args,
		TimeoutSeconds: req.Msg.TimeoutSeconds,
		DryRun:         dryRun,
	})
	if err != nil {
		connectErr := gate.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("gate.RunGate(scenario=%q rev=%q): %v", req.Msg.Scenario, req.Msg.TargetRevision, err)
		}
		return nil, connectErr
	}

	return connect.NewResponse(&gatev1.RunGateResponse{
		GateId:  dec.GateID,
		DryRun:  dec.DryRun,
		Verdict: verdictToProto(dec.Verdict),
		Results: domainResultsToProto(dec.Results),
	}), nil
}

func (h *connectHandler) GetGate(ctx context.Context, req *connect.Request[gatev1.GetGateRequest]) (*connect.Response[gatev1.GetGateResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	g, results, err := h.deps.Service.GetGate(ctx, req.Msg.Id)
	if err != nil {
		connectErr := gate.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("gate.GetGate(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&gatev1.GetGateResponse{
		Gate:    domainGateToProto(g),
		Results: domainResultsToProto(results),
	}), nil
}

func (h *connectHandler) WaitGate(ctx context.Context, req *connect.Request[gatev1.WaitGateRequest]) (*connect.Response[gatev1.WaitGateResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	timeout := time.Duration(req.Msg.TimeoutSeconds) * time.Second
	g, results, timedOut, err := h.deps.Service.WaitGate(ctx, req.Msg.Id, timeout)
	if err != nil {
		connectErr := gate.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("gate.WaitGate(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&gatev1.WaitGateResponse{
		Gate:     domainGateToProto(g),
		Results:  domainResultsToProto(results),
		TimedOut: timedOut,
	}), nil
}

func (h *connectHandler) ListGates(ctx context.Context, req *connect.Request[gatev1.ListGatesRequest]) (*connect.Response[gatev1.ListGatesResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	list, err := h.deps.Service.ListGates(ctx, gate.ListFilter{Limit: int(req.Msg.Limit)})
	if err != nil {
		h.deps.Logger.Printf("gate.ListGates: %v", err)
		return nil, gate.ToConnectError(err)
	}
	resp := &gatev1.ListGatesResponse{Gates: make([]*gatev1.Gate, 0, len(list))}
	for _, g := range list {
		resp.Gates = append(resp.Gates, domainGateToProto(g))
	}
	return connect.NewResponse(resp), nil
}

// ownerActor derives a stable actor label from the authenticated owner for the
// per-OS validation dispatch audit trail.
func ownerActor(owner auth.Identity) string {
	if owner.OwnerID != "" {
		return owner.OwnerID
	}
	if owner.Email != "" {
		return owner.Email
	}
	return "owner"
}
