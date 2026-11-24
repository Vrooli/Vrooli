package computemanager

import (
	"context"
	"net"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// VerifiedLocalQEMUReach routes only an explicitly Compute Manager-bound SSH
// target, and re-resolves ownership before each operation. It never falls
// through to a generic SSH or Bridge transport when the manager is unavailable.
type VerifiedLocalQEMUReach struct {
	Manager interface {
		GetInstance(context.Context, string) (InstanceReference, error)
	}
	SSH reach.Reach
}

var _ reach.Reach = (*VerifiedLocalQEMUReach)(nil)

func (r *VerifiedLocalQEMUReach) verify(ctx context.Context, target identity.TargetRef) error {
	if r == nil || r.Manager == nil || r.SSH == nil {
		return &reach.Error{Kind: reach.KindUnavailable, Transport: identity.TransportSSH, Target: target.Key(), Detail: "Compute Manager QEMU reach is not configured"}
	}
	if target.Transport != identity.TransportSSH || target.ComputeManagerID == "" {
		return &reach.Error{Kind: reach.KindInvalidArgument, Transport: target.Transport, Target: target.Key(), Detail: "managed QEMU reach requires a Compute Manager SSH identity"}
	}
	ref, err := r.Manager.GetInstance(ctx, target.ComputeManagerID)
	if err != nil {
		return &reach.Error{Kind: reach.KindUnavailable, Transport: identity.TransportSSH, Target: target.Key(), Detail: "Compute Manager could not verify local-QEMU ownership", Err: err}
	}
	if ref.ManagerInstanceID != target.ComputeManagerID || ref.Provider != QEMUProvider {
		return &reach.Error{Kind: reach.KindInvalidArgument, Transport: identity.TransportSSH, Target: target.Key(), Detail: "target does not name the expected local-QEMU Compute Manager instance"}
	}
	owned, err := ref.TargetRef()
	if err != nil {
		return &reach.Error{Kind: reach.KindUnavailable, Transport: identity.TransportSSH, Target: target.Key(), Detail: "Compute Manager instance has no usable SSH locator", Err: err}
	}
	ip := net.ParseIP(owned.Locator.Host)
	if ip == nil || !ip.IsLoopback() {
		return &reach.Error{Kind: reach.KindInvalidArgument, Transport: identity.TransportSSH, Target: target.Key(), Detail: "local-QEMU instance locator is not loopback-only"}
	}
	if target.Locator.Host != owned.Locator.Host || target.Locator.Port != owned.Locator.Port || target.Locator.User != owned.Locator.User {
		return &reach.Error{Kind: reach.KindInvalidArgument, Transport: identity.TransportSSH, Target: target.Key(), Detail: "deployment SSH locator does not match its manager-owned instance"}
	}
	return nil
}

func (r *VerifiedLocalQEMUReach) Exec(ctx context.Context, target identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	if err := r.verify(ctx, target); err != nil {
		return reach.Result{}, err
	}
	return r.SSH.Exec(ctx, target, cmd)
}

func (r *VerifiedLocalQEMUReach) Deliver(ctx context.Context, target identity.TargetRef, delivery reach.Delivery) (reach.DeliveryReceipt, error) {
	if err := r.verify(ctx, target); err != nil {
		return reach.DeliveryReceipt{}, err
	}
	return r.SSH.Deliver(ctx, target, delivery)
}

func (r *VerifiedLocalQEMUReach) Negotiate(ctx context.Context, target identity.TargetRef) (reach.Capabilities, error) {
	if err := r.verify(ctx, target); err != nil {
		return reach.Capabilities{}, err
	}
	return r.SSH.Negotiate(ctx, target)
}
