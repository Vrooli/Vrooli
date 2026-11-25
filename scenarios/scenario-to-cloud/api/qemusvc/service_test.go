package qemusvc

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	instancev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance/instance_v1connect"
	qemuv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/qemu"
	"scenario-to-cloud/computemanager"
)

type capacityRefusalService struct {
	instancev1connect.UnimplementedInstanceServiceHandler
	request *instancev1.RequestInstanceRequest
}

func (s *capacityRefusalService) RequestInstance(_ context.Context, request *connect.Request[instancev1.RequestInstanceRequest]) (*connect.Response[instancev1.RequestInstanceResponse], error) {
	s.request = request.Msg
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("fixture broker capacity denial"))
}

// This exercises the same qemu.acquire Connect entry consumed by the governed
// Program Runtime branch, over the real generated Compute Manager client. The
// fixture manager returns the capacity-denial response, so no runtime VM or
// lease is created by the test.
func TestAcquirePropagatesComputeManagerCapacityRefusalThroughGeneratedClient(t *testing.T) {
	manager := &capacityRefusalService{}
	path, handler := instancev1connect.NewInstanceServiceHandler(manager)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := computemanager.NewWithResolver(server.Client(), func(context.Context) (string, error) { return server.URL, nil })
	service := New(client, func(context.Context) (string, error) { return "ssh-ed25519 AAAAfixture", nil })
	_, err := service.Acquire(context.Background(), connect.NewRequest(&qemuv1.AcquireRequest{
		IdempotencyKey: "qemu-lane:RUN-01", Region: "local", Size: "headless-linux", Image: "/fixture/ubuntu.img",
		LifetimeSeconds: 3600, Memory: "2048", Cpus: 2, SshUser: "runner", Profile: "headless-linux",
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("Acquire error = %v, want propagated capacity refusal", err)
	}
	request := manager.request
	if request == nil || request.GetProvider() != "local-qemu" || request.GetIdempotencyKey() != "qemu-lane:RUN-01" || request.GetRegion() != "local" || request.GetSize() != "headless-linux" || request.GetProfile() != "headless-linux" || request.GetMemory() != "2048" || request.GetCpus() != 2 || request.GetSshAuthorizedKey() != "ssh-ed25519 AAAAfixture" {
		t.Fatalf("Compute Manager generated request = %#v", request)
	}
	if path == "" {
		t.Fatal("generated Compute Manager handler exposed no route")
	}
}

type managerFixture struct {
	request computemanager.QEMURequest
	receipt computemanager.InstanceReference
	calls   int
}

func (m *managerFixture) RequestQEMU(_ context.Context, request computemanager.QEMURequest) (computemanager.InstanceReference, error) {
	m.request = request
	m.calls++
	if m.receipt.ManagerInstanceID != "" {
		return m.receipt, nil
	}
	return computemanager.InstanceReference{ManagerInstanceID: "cm-qemu-1", Provider: "local-qemu", Region: "local", Size: request.Size, Image: request.Image, Address: "127.0.0.1", SSHPort: 2222, SSHUser: "runner"}, nil
}

func (m *managerFixture) GetInstance(_ context.Context, id string) (computemanager.InstanceReference, error) {
	return computemanager.InstanceReference{ManagerInstanceID: id, Provider: "local-qemu", Address: "127.0.0.1", SSHPort: 2222, SSHUser: "runner"}, nil
}

func (m *managerFixture) DestroyInstance(_ context.Context, id string) (computemanager.InstanceReference, error) {
	return computemanager.InstanceReference{ManagerInstanceID: id}, nil
}

func TestAcquireRequestsManagerOwnedTarget(t *testing.T) {
	manager := &managerFixture{}
	service := New(manager, func(context.Context) (string, error) { return "ssh-ed25519 AAAAfixture", nil })
	response, err := service.Acquire(context.Background(), connect.NewRequest(&qemuv1.AcquireRequest{
		IdempotencyKey: "qemu-lane:RUN-01", Region: "local", Size: "headless-linux", Image: "ubuntu-24.04",
		LifetimeSeconds: 3600, SshUser: "runner", Profile: "headless-linux",
	}))
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if manager.calls != 1 || manager.request.IdempotencyKey != "qemu-lane:RUN-01" || manager.request.SSHAuthorizedKey != "ssh-ed25519 AAAAfixture" {
		t.Fatalf("request=%+v calls=%d", manager.request, manager.calls)
	}
	if response.Msg.GetManagerInstanceId() != "cm-qemu-1" || response.Msg.GetTarget().GetLocator().GetPort() != 2222 {
		t.Fatalf("reference=%+v", response.Msg)
	}
}

func TestAcquireRefusesMismatchedComputeManagerReceipt(t *testing.T) {
	manager := &managerFixture{receipt: computemanager.InstanceReference{
		ManagerInstanceID: "cm-qemu-1", Provider: "hetzner", Region: "fsn1",
		Address: "203.0.113.20", SSHPort: 22, SSHUser: "runner",
	}}
	service := New(manager, func(context.Context) (string, error) { return "ssh-ed25519 AAAAfixture", nil })
	_, err := service.Acquire(context.Background(), connect.NewRequest(&qemuv1.AcquireRequest{
		IdempotencyKey: "qemu-lane:RUN-01", Region: "local", Size: "headless-linux", Image: "ubuntu-24.04",
		LifetimeSeconds: 3600, Profile: "headless-linux",
	}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("code=%s err=%v, want refusal of mismatched provider receipt", connect.CodeOf(err), err)
	}
}

func TestAcquireRefusesNonCanonicalOfferBeforeManagerCall(t *testing.T) {
	manager := &managerFixture{}
	service := New(manager, func(context.Context) (string, error) { return "ssh-ed25519 AAAAfixture", nil })
	_, err := service.Acquire(context.Background(), connect.NewRequest(&qemuv1.AcquireRequest{
		IdempotencyKey: "qemu-lane:RUN-01", Region: "fsn1", Size: "headless-linux", Image: "ubuntu-24.04",
		LifetimeSeconds: 3600, Profile: "headless-linux",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument || manager.calls != 0 {
		t.Fatalf("code=%s calls=%d err=%v", connect.CodeOf(err), manager.calls, err)
	}
}

func TestAcquireRequiresCredentialAuthorityPublicKeyWithoutAcceptingCallerKey(t *testing.T) {
	manager := &managerFixture{}
	service := New(manager, nil)
	_, err := service.Acquire(context.Background(), connect.NewRequest(&qemuv1.AcquireRequest{
		IdempotencyKey: "qemu-lane:RUN-01", Region: "local", Size: "headless-linux", Image: "ubuntu-24.04",
		LifetimeSeconds: 3600, Profile: "headless-linux",
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || manager.calls != 0 {
		t.Fatalf("code=%s calls=%d err=%v", connect.CodeOf(err), manager.calls, err)
	}
}
