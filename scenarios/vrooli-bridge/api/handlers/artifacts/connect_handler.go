package artifacts

import (
	"context"
	"log"

	"vrooli-bridge/internal/artifacts"
	"vrooli-bridge/internal/auth"

	"connectrpc.com/connect"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
)

// dryRunHeader is the canonical header cli-core sets on `--dry-run`.
// DistributeArtifact honours it by validating then short-circuiting before
// recording or delivering anything.
const dryRunHeader = "X-Dry-Run"

// Deps wires the seams the Connect artifacts handler needs. All verbs are
// owner-gated via auth.RequireOwner.
type Deps struct {
	Service artifacts.Service
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

// DistributeArtifact ships a non-git artifact to a node via device-sync-hub
// directed delivery. Owner-gated. Honours X-Dry-Run.
func (h *connectHandler) DistributeArtifact(ctx context.Context, req *connect.Request[artifactsv1.DistributeArtifactRequest]) (*connect.Response[artifactsv1.DistributeArtifactResponse], error) {
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

	dec, err := h.deps.Service.Distribute(ctx, artifacts.DistributeInput{
		Actor:           actor,
		NodeID:          req.Msg.NodeId,
		Name:            req.Msg.Name,
		SourceRef:       req.Msg.SourceRef,
		DestinationPath: req.Msg.DestinationPath,
		DryRun:          dryRun,
	})
	if err != nil {
		connectErr := artifacts.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("artifacts.DistributeArtifact(node=%q name=%q): %v", req.Msg.NodeId, req.Msg.Name, err)
		}
		return nil, connectErr
	}

	return connect.NewResponse(&artifactsv1.DistributeArtifactResponse{
		DistributionId: dec.DistributionID,
		DryRun:         dec.DryRun,
		Status:         statusToProto(dec.Status),
		DeliveryRef:    dec.DeliveryRef,
	}), nil
}

func (h *connectHandler) GetDistribution(ctx context.Context, req *connect.Request[artifactsv1.GetDistributionRequest]) (*connect.Response[artifactsv1.GetDistributionResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	d, err := h.deps.Service.GetDistribution(ctx, req.Msg.Id)
	if err != nil {
		connectErr := artifacts.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("artifacts.GetDistribution(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&artifactsv1.GetDistributionResponse{Distribution: domainToProto(d)}), nil
}

func (h *connectHandler) ListDistributions(ctx context.Context, req *connect.Request[artifactsv1.ListDistributionsRequest]) (*connect.Response[artifactsv1.ListDistributionsResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	list, err := h.deps.Service.ListDistributions(ctx, artifacts.ListFilter{
		NodeID: req.Msg.NodeId,
		Limit:  int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("artifacts.ListDistributions: %v", err)
		return nil, artifacts.ToConnectError(err)
	}
	resp := &artifactsv1.ListDistributionsResponse{Distributions: make([]*artifactsv1.Distribution, 0, len(list))}
	for _, d := range list {
		resp.Distributions = append(resp.Distributions, domainToProto(d))
	}
	return connect.NewResponse(resp), nil
}
