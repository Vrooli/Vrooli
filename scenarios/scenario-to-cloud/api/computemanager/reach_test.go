package computemanager

import (
	"context"
	"errors"
	"testing"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

type fixedInstanceGetter struct {
	ref InstanceReference
	err error
}

func (f fixedInstanceGetter) GetInstance(context.Context, string) (InstanceReference, error) {
	return f.ref, f.err
}

type reachRecorder struct{ calls int }

func (r *reachRecorder) Exec(context.Context, identity.TargetRef, reach.Command) (reach.Result, error) {
	r.calls++
	return reach.Result{ExitCode: 0}, nil
}
func (r *reachRecorder) Deliver(context.Context, identity.TargetRef, reach.Delivery) (reach.DeliveryReceipt, error) {
	r.calls++
	return reach.DeliveryReceipt{}, nil
}
func (r *reachRecorder) Negotiate(context.Context, identity.TargetRef) (reach.Capabilities, error) {
	r.calls++
	return reach.Capabilities{Transport: identity.TransportSSH, Online: true}, nil
}

func TestVerifiedLocalQEMUReachUsesManagerOwnershipBeforeSSH(t *testing.T) {
	manager := fixedInstanceGetter{ref: InstanceReference{
		ManagerInstanceID: "qemu-1", Provider: QEMUProvider, Address: "127.0.0.1", SSHUser: "vrooli", SSHPort: 2222,
	}}
	delegate := &reachRecorder{}
	adapter := &VerifiedLocalQEMUReach{Manager: manager, SSH: delegate}
	target := identity.TargetRef{Transport: identity.TransportSSH, ComputeManagerID: "qemu-1", Locator: identity.TargetLocator{Host: "127.0.0.1", Port: 2222, User: "vrooli", Workdir: "/srv/vrooli"}}
	if _, err := adapter.Exec(context.Background(), target, reach.Command{Verb: "cloud-target receipt get"}); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if delegate.calls != 1 {
		t.Fatalf("delegate calls = %d, want 1", delegate.calls)
	}
}

func TestVerifiedLocalQEMUReachRefusesSpoofedOrNonLoopbackLocator(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  InstanceReference
		loc  identity.TargetLocator
	}{
		{name: "wrong provider", ref: InstanceReference{ManagerInstanceID: "qemu-1", Provider: "hetzner", Address: "127.0.0.1", SSHUser: "vrooli", SSHPort: 2222}, loc: identity.TargetLocator{Host: "127.0.0.1", Port: 2222, User: "vrooli"}},
		{name: "non-loopback manager address", ref: InstanceReference{ManagerInstanceID: "qemu-1", Provider: QEMUProvider, Address: "192.0.2.10", SSHUser: "vrooli", SSHPort: 2222}, loc: identity.TargetLocator{Host: "192.0.2.10", Port: 2222, User: "vrooli"}},
		{name: "spoofed host", ref: InstanceReference{ManagerInstanceID: "qemu-1", Provider: QEMUProvider, Address: "127.0.0.1", SSHUser: "vrooli", SSHPort: 2222}, loc: identity.TargetLocator{Host: "192.0.2.10", Port: 2222, User: "vrooli"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			delegate := &reachRecorder{}
			adapter := &VerifiedLocalQEMUReach{Manager: fixedInstanceGetter{ref: tc.ref}, SSH: delegate}
			target := identity.TargetRef{Transport: identity.TransportSSH, ComputeManagerID: "qemu-1", Locator: tc.loc}
			_, err := adapter.Exec(context.Background(), target, reach.Command{Verb: "cloud-target receipt get"})
			if err == nil || !reach.IsKind(err, reach.KindInvalidArgument) {
				t.Fatalf("Exec error = %v, want typed refusal", err)
			}
			if delegate.calls != 0 {
				t.Fatalf("SSH delegate called before ownership validation: %d", delegate.calls)
			}
		})
	}
}

func TestVerifiedLocalQEMUReachRefusesWhenManagerLookupFails(t *testing.T) {
	delegate := &reachRecorder{}
	adapter := &VerifiedLocalQEMUReach{Manager: fixedInstanceGetter{err: errors.New("manager unavailable")}, SSH: delegate}
	target := identity.TargetRef{Transport: identity.TransportSSH, ComputeManagerID: "qemu-1", Locator: identity.TargetLocator{Host: "127.0.0.1", Port: 2222, User: "vrooli"}}
	_, err := adapter.Exec(context.Background(), target, reach.Command{Verb: "cloud-target receipt get"})
	if !reach.IsKind(err, reach.KindUnavailable) {
		t.Fatalf("Exec error = %v, want unavailable", err)
	}
	if delegate.calls != 0 {
		t.Fatalf("manager outage fell through to SSH: calls=%d", delegate.calls)
	}
}
