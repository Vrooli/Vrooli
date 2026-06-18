package fleet

import (
	"context"

	"vrooli-bridge/internal/fleet"
	"vrooli-bridge/internal/provision"
	"vrooli-bridge/internal/registry"

	"google.golang.org/protobuf/types/known/timestamppb"

	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/fleet"
)

// This file is the single translation point between the proto-free fleet domain
// (its seams + DTOs) and the concrete registry / presence / provision services
// and the proto wire types. The fleet domain never imports a sibling domain or
// proto; these adapters do.

// ---- proto <-> domain translations (api-steer §7) ----

func domainRolloutToProto(r fleet.Rollout) *fleetv1.Rollout {
	return &fleetv1.Rollout{
		Id:             r.ID,
		TargetRevision: r.TargetRevision,
		Status:         rolloutStatusToProto(r.Status),
		TotalNodes:     int32(r.TotalNodes),
		Dispatched:     int32(r.Dispatched),
		Skipped:        int32(r.Skipped),
		Failed:         int32(r.Failed),
		CreatedAt:      timestamppb.New(r.CreatedAt),
	}
}

func rolloutStatusToProto(s fleet.RolloutStatus) fleetv1.RolloutStatus {
	switch s {
	case fleet.StatusDispatched:
		return fleetv1.RolloutStatus_ROLLOUT_STATUS_DISPATCHED
	case fleet.StatusPartial:
		return fleetv1.RolloutStatus_ROLLOUT_STATUS_PARTIAL
	case fleet.StatusFailed:
		return fleetv1.RolloutStatus_ROLLOUT_STATUS_FAILED
	default:
		return fleetv1.RolloutStatus_ROLLOUT_STATUS_UNSPECIFIED
	}
}

func domainResultToProto(r fleet.NodeResult) *fleetv1.NodeRolloutResult {
	return &fleetv1.NodeRolloutResult{
		NodeId:      r.NodeID,
		Disposition: dispositionToProto(r.Disposition),
		OpId:        r.OpID,
		Detail:      r.Detail,
	}
}

func dispositionToProto(d fleet.NodeDisposition) fleetv1.NodeRolloutDisposition {
	switch d {
	case fleet.DispositionDispatched:
		return fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_DISPATCHED
	case fleet.DispositionSkippedOffline:
		return fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_SKIPPED_OFFLINE
	case fleet.DispositionSkippedNeedsUpdate:
		return fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_SKIPPED_NEEDS_UPDATE
	case fleet.DispositionSkippedRevoked:
		return fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_SKIPPED_REVOKED
	case fleet.DispositionFailed:
		return fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_FAILED
	default:
		return fleetv1.NodeRolloutDisposition_NODE_ROLLOUT_DISPOSITION_UNSPECIFIED
	}
}

func domainResultsToProto(results []fleet.NodeResult) []*fleetv1.NodeRolloutResult {
	out := make([]*fleetv1.NodeRolloutResult, 0, len(results))
	for _, r := range results {
		out = append(out, domainResultToProto(r))
	}
	return out
}

// ---- seam adapters (proto-free domain <-> concrete services) ----

// nodeListerAdapter projects registry nodes down to the fleet NodeRef set.
type nodeListerAdapter struct {
	svc registry.Service
}

var _ fleet.NodeLister = nodeListerAdapter{}

func (a nodeListerAdapter) ListNodes(ctx context.Context) ([]fleet.NodeRef, error) {
	nodes, err := a.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]fleet.NodeRef, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, fleet.NodeRef{ID: n.ID, Revoked: n.Revoked()})
	}
	return out, nil
}

// provisionerAdapter wraps the provision service's Sync so a roll dispatches a
// privileged provisioning op per eligible node WITHOUT fleet importing the
// provision domain in its own package. Each op is independently audited by the
// provision service (accountability preserved).
type provisionerAdapter struct {
	svc provision.Service
}

var _ fleet.Provisioner = provisionerAdapter{}

func (a provisionerAdapter) Provision(ctx context.Context, in fleet.ProvisionRequest) (string, error) {
	dec, err := a.svc.Sync(ctx, provision.SyncInput{
		Actor:          in.Actor,
		NodeID:         in.NodeID,
		TargetRevision: in.TargetRevision,
		TimeoutSeconds: in.TimeoutSeconds,
	})
	if err != nil {
		return "", err
	}
	return dec.OpID, nil
}
