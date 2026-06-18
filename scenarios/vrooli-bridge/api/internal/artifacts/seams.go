package artifacts

import "context"

// NodeReader is the registry read seam: distribution projects a node down to the
// TargetNode it needs (id + revocation). The handler adapter wraps the registry
// service. A missing node surfaces as ErrNodeNotFound.
type NodeReader interface {
	GetTarget(ctx context.Context, id string) (TargetNode, error)
}

// DirectedDelivery is the device-sync-hub directed-delivery seam: hand an
// artifact reference off to device-sync-hub, which moves the bytes to the target
// node. Bridge implements NO byte transport of its own — this seam IS the
// "bridge orchestrates, device-sync-hub moves the bytes" boundary. The handler
// adapter binds the concrete device-sync-hub client (a documented integration
// point, mirroring audit's workspace-sandbox drop-in); tests substitute a fake.
type DirectedDelivery interface {
	Deliver(ctx context.Context, req DeliveryRequest) (DeliveryResult, error)
}

// DeliveryRequest is the proto-free DTO handed to device-sync-hub.
type DeliveryRequest struct {
	NodeID          string
	Name            string
	SourceRef       string
	DestinationPath string
}

// DeliveryResult is device-sync-hub's response: the delivery reference the node
// fetches against, whether the artifact has already reached the node
// (Delivered), and an optional human-readable detail.
type DeliveryResult struct {
	Ref       string
	Delivered bool
	Detail    string
}
