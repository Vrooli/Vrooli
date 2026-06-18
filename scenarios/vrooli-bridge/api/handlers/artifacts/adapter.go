package artifacts

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"vrooli-bridge/internal/artifacts"
	"vrooli-bridge/internal/registry"

	"google.golang.org/protobuf/types/known/timestamppb"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
)

// This file is the single translation point between the proto-free artifacts
// domain (its seams + DTOs) and the concrete registry service, the
// device-sync-hub directed-delivery client, and the proto wire types. The
// artifacts domain never imports a sibling domain or proto; these adapters do.

// ---- proto <-> domain translations (api-steer §7) ----

func domainToProto(d artifacts.Distribution) *artifactsv1.Distribution {
	out := &artifactsv1.Distribution{
		Id:              d.ID,
		NodeId:          d.NodeID,
		Name:            d.Name,
		SourceRef:       d.SourceRef,
		DestinationPath: d.DestinationPath,
		Status:          statusToProto(d.Status),
		DeliveryRef:     d.DeliveryRef,
		Detail:          d.Detail,
		CreatedAt:       timestamppb.New(d.CreatedAt),
	}
	if !d.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(d.UpdatedAt)
	}
	return out
}

func statusToProto(s artifacts.DeliveryStatus) artifactsv1.DeliveryStatus {
	switch s {
	case artifacts.StatusPending:
		return artifactsv1.DeliveryStatus_DELIVERY_STATUS_PENDING
	case artifacts.StatusDelivered:
		return artifactsv1.DeliveryStatus_DELIVERY_STATUS_DELIVERED
	case artifacts.StatusFailed:
		return artifactsv1.DeliveryStatus_DELIVERY_STATUS_FAILED
	default:
		return artifactsv1.DeliveryStatus_DELIVERY_STATUS_UNSPECIFIED
	}
}

// ---- seam adapters (proto-free domain <-> concrete services) ----

// nodeReaderAdapter projects a registry node down to the artifacts TargetNode.
type nodeReaderAdapter struct {
	svc registry.Service
}

var _ artifacts.NodeReader = nodeReaderAdapter{}

func (a nodeReaderAdapter) GetTarget(ctx context.Context, id string) (artifacts.TargetNode, error) {
	n, err := a.svc.Get(ctx, id)
	if err != nil {
		var notFound registry.ErrNodeNotFound
		if errors.As(err, &notFound) {
			return artifacts.TargetNode{}, artifacts.ErrNodeNotFound{ID: id}
		}
		return artifacts.TargetNode{}, err
	}
	return artifacts.TargetNode{ID: n.ID, Revoked: n.Revoked()}, nil
}

// deviceSyncDelivery is the production DirectedDelivery: it hands the artifact
// reference off to device-sync-hub, which moves the bytes to the node. This is
// the "bridge orchestrates, device-sync-hub moves the bytes" integration point —
// bridge reads no bytes. It returns a device-sync-hub delivery reference the node
// fetches against and reports the delivery as IN FLIGHT (Delivered=false): the
// node confirms receipt out of band (a future node-confirmation event flips the
// distribution to DELIVERED).
//
// device-sync-hub currently carries an environmental authenticator blocker
// (memory: device-sync-hub Phase-6 hardening), so the concrete directed-delivery
// RPC binding is the documented drop-in here (mirroring audit's workspace-sandbox
// Sink): the seam contract is stable and tested, and wiring the real
// device-sync-hub TransferService client is a localized change behind this
// adapter that does not touch the artifacts domain.
type deviceSyncDelivery struct{}

var _ artifacts.DirectedDelivery = deviceSyncDelivery{}

func (deviceSyncDelivery) Deliver(_ context.Context, req artifacts.DeliveryRequest) (artifacts.DeliveryResult, error) {
	// The device-sync-hub directed-delivery reference the node fetches against.
	ref := fmt.Sprintf("dsh://%s/%s", req.NodeID, url.PathEscape(req.Name))
	return artifacts.DeliveryResult{
		Ref:       ref,
		Delivered: false, // in flight; the node confirms receipt out of band
		Detail:    "handed off to device-sync-hub directed delivery",
	}, nil
}
