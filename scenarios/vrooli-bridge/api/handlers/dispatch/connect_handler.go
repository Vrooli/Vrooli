package dispatch

import (
	"context"
	"log"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/dispatch"

	"connectrpc.com/connect"

	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
)

// dryRunHeader is the canonical header cli-core sets on `--dry-run`
// (cli-core/cliutil.DryRunHeader). Mutating Connect methods honour it by
// validating then short-circuiting before the first side effect.
const dryRunHeader = "X-Dry-Run"

// Deps wires the seams the Connect dispatch handler needs.
type Deps struct {
	Service dispatch.Service
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

// DispatchJob validates {scenario, verb, args} against the allowlist + the
// node's scopes, then (unless a dry-run) creates a durable run, audits the
// dispatch, and pushes the typed job to the node. Owner-gated.
func (h *connectHandler) DispatchJob(ctx context.Context, req *connect.Request[dispatchv1.DispatchJobRequest]) (*connect.Response[dispatchv1.DispatchJobResponse], error) {
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

	decision, err := h.deps.Service.Dispatch(ctx, dispatch.DispatchInput{
		Actor:  actor,
		DryRun: dryRun,
		Job: dispatch.Job{
			NodeID:         req.Msg.NodeId,
			Scenario:       req.Msg.Scenario,
			Verb:           req.Msg.Verb,
			Args:           req.Msg.Args,
			TimeoutSeconds: req.Msg.TimeoutSeconds,
			DeviceID:       req.Msg.DeviceId,
			LeaseToken:     req.Msg.LeaseToken,
		},
	})
	if err != nil {
		connectErr := dispatch.ToConnectError(err)
		// Rejections (allowlist/scope/precondition) are expected operator errors;
		// only log genuine internal failures to avoid noise.
		if !dispatch.IsRejection(err) && connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("dispatch.DispatchJob(node=%q verb=%q): %v", req.Msg.NodeId, req.Msg.Verb, err)
		}
		return nil, connectErr
	}

	return connect.NewResponse(&dispatchv1.DispatchJobResponse{
		RunId:    decision.RunID,
		DryRun:   decision.DryRun,
		NodeId:   decision.Job.NodeID,
		Scenario: decision.Job.Scenario,
		Verb:     decision.Job.Verb,
		Args:     decision.Job.Args,
		Queued:   decision.Queued,
	}), nil
}
