package computemanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	instancev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance/instance_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"scenario-to-cloud/identity"
)

type fixedInstanceService struct {
	instancev1connect.UnimplementedInstanceServiceHandler
	request *instancev1.RequestInstanceRequest
	getID   string
	destroy string
	listed  []*instancev1.Instance
	item    *instancev1.Instance
}

func (s *fixedInstanceService) RequestInstance(_ context.Context, request *connect.Request[instancev1.RequestInstanceRequest]) (*connect.Response[instancev1.RequestInstanceResponse], error) {
	s.request = request.Msg
	return connect.NewResponse(&instancev1.RequestInstanceResponse{Instance: s.item}), nil
}

func (s *fixedInstanceService) GetInstance(_ context.Context, request *connect.Request[instancev1.GetInstanceRequest]) (*connect.Response[instancev1.GetInstanceResponse], error) {
	s.getID = request.Msg.GetId()
	return connect.NewResponse(&instancev1.GetInstanceResponse{Instance: s.item}), nil
}

func (s *fixedInstanceService) ListInstances(context.Context, *connect.Request[instancev1.ListInstancesRequest]) (*connect.Response[instancev1.ListInstancesResponse], error) {
	return connect.NewResponse(&instancev1.ListInstancesResponse{Instances: s.listed}), nil
}

func (s *fixedInstanceService) DestroyInstance(_ context.Context, request *connect.Request[instancev1.DestroyInstanceRequest]) (*connect.Response[instancev1.DestroyInstanceResponse], error) {
	s.destroy = request.Msg.GetId()
	return connect.NewResponse(&instancev1.DestroyInstanceResponse{Instance: s.item}), nil
}

func serveComputeManager(t *testing.T, service *fixedInstanceService) (*httptest.Server, *Client) {
	t.Helper()
	path, handler := instancev1connect.NewInstanceServiceHandler(service)
	if path == "" || handler == nil {
		t.Fatal("generated Connect handler returned no route")
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewWithResolver(server.Client(), func(context.Context) (string, error) { return server.URL, nil })
	return server, client
}

func bridgeInstance() *instancev1.Instance {
	return &instancev1.Instance{
		Id: "cm-instance-1", Provider: QEMUProvider, ProviderInstanceId: "qemu-guest-1",
		State: instancev1.InstanceState_INSTANCE_STATE_RUNNING, Region: "local", Size: "headless-linux",
		Image: "ubuntu-24.04", Address: "127.0.0.1", BridgeMachineId: "bridge-machine-1",
		SshUser: "runner", SshPort: 22441,
		ExpiresAt: timestamppb.New(time.Date(2026, 11, 23, 14, 0, 0, 0, time.UTC)),
	}
}

func TestRequestQEMUUsesGeneratedAPIAndConvertsBridgeTarget(t *testing.T) {
	service := &fixedInstanceService{item: bridgeInstance()}
	_, client := serveComputeManager(t, service)
	lifetime := int64(1800)
	ref, err := client.RequestQEMU(context.Background(), QEMURequest{
		IdempotencyKey: "scenario-to-cloud:qemu:run-01:target-a",
		Region:         "local", Size: "headless-linux", Image: "ubuntu-24.04", LifetimeSeconds: lifetime,
		Memory: "4G", CPUs: 2, SSHUser: "runner", SSHPort: 0, Profile: "headless-linux", SSHAuthorizedKey: "ssh-ed25519 AAAAfixture runner",
	})
	if err != nil {
		t.Fatalf("RequestQEMU: %v", err)
	}
	request := service.request
	if request == nil {
		t.Fatal("generated RequestInstance RPC was not called")
	}
	if request.GetIdempotencyKey() != "scenario-to-cloud:qemu:run-01:target-a" || request.GetProvider() != QEMUProvider || request.GetRequestedBy() != QEMURequester || request.GetLifetimeSeconds() != lifetime {
		t.Fatalf("request identity/provider/lifetime = %q/%q/%q/%d", request.GetIdempotencyKey(), request.GetProvider(), request.GetRequestedBy(), request.GetLifetimeSeconds())
	}
	if request.GetRegion() != "local" || request.GetSize() != "headless-linux" || request.GetImage() != "ubuntu-24.04" || request.GetMemory() != "4G" || request.GetCpus() != 2 || request.GetSshUser() != "runner" || request.GetSshPort() != 0 || request.GetProfile() != "headless-linux" || request.GetSshAuthorizedKey() != "ssh-ed25519 AAAAfixture runner" {
		t.Fatalf("request resource profile not preserved: %+v", request)
	}
	if ref.ManagerInstanceID != "cm-instance-1" || ref.ProviderInstanceID != "qemu-guest-1" || ref.BridgeMachineID != "bridge-machine-1" || !ref.ExpiresAt.Equal(time.Date(2026, 11, 23, 14, 0, 0, 0, time.UTC)) {
		t.Fatalf("typed Compute Manager reference = %+v", ref)
	}
	target, err := ref.TargetRef()
	if err != nil {
		t.Fatalf("TargetRef: %v", err)
	}
	want := identity.TargetRef{
		MachineID: "bridge-machine-1", ComputeManagerID: "cm-instance-1", Transport: identity.TransportBridge,
		Locator: identity.TargetLocator{Host: "127.0.0.1", Port: 22441, User: "runner"},
	}
	if !reflect.DeepEqual(target, want) {
		t.Fatalf("target = %+v, want %+v", target, want)
	}
}

func TestTargetRefUsesSSHLocatorWhenComputeManagerHasNoBridgeIdentity(t *testing.T) {
	ref := InstanceReference{ManagerInstanceID: "cm-qemu-2", Address: "127.0.0.1", SSHPort: 22550, SSHUser: "runner"}
	target, err := ref.TargetRef()
	if err != nil {
		t.Fatalf("TargetRef: %v", err)
	}
	want := identity.TargetRef{ComputeManagerID: "cm-qemu-2", Transport: identity.TransportSSH, Locator: identity.TargetLocator{Host: "127.0.0.1", Port: 22550, User: "runner"}}
	if !reflect.DeepEqual(target, want) {
		t.Fatalf("target = %+v, want %+v", target, want)
	}
	if target.MachineID != "" || ref.ManagerInstanceID != "cm-qemu-2" {
		t.Fatalf("manager ID must remain separate from Bridge MachineID: ref=%+v target=%+v", ref, target)
	}
}

func TestGeneratedGetListAndDestroyCalls(t *testing.T) {
	item := bridgeInstance()
	service := &fixedInstanceService{item: item, listed: []*instancev1.Instance{item}}
	_, client := serveComputeManager(t, service)
	ctx := context.Background()

	got, err := client.GetInstance(ctx, "cm-instance-1")
	if err != nil || got.ManagerInstanceID != "cm-instance-1" || service.getID != "cm-instance-1" {
		t.Fatalf("GetInstance = %+v, getID=%q, err=%v", got, service.getID, err)
	}
	list, err := client.ListInstances(ctx)
	if err != nil || len(list) != 1 || list[0].ManagerInstanceID != "cm-instance-1" {
		t.Fatalf("ListInstances = %+v, err=%v", list, err)
	}
	destroyed, err := client.DestroyInstance(ctx, "cm-instance-1")
	if err != nil || destroyed.ManagerInstanceID != "cm-instance-1" || service.destroy != "cm-instance-1" {
		t.Fatalf("DestroyInstance = %+v, destroyID=%q, err=%v", destroyed, service.destroy, err)
	}
}

func TestQEMURequestRejectsMissingIdempotencyOrLifetime(t *testing.T) {
	client := NewWithResolver(http.DefaultClient, func(context.Context) (string, error) {
		t.Fatal("invalid request must be refused before service discovery")
		return "", nil
	})
	base := QEMURequest{Region: "local", Size: "headless-linux", Profile: "headless-linux", Image: "ubuntu-24.04", LifetimeSeconds: 60, SSHAuthorizedKey: "ssh-ed25519 AAAAfixture"}
	if _, err := client.RequestQEMU(context.Background(), base); err == nil {
		t.Fatal("empty idempotency key was accepted")
	}
	base.IdempotencyKey = "stable-key"
	base.LifetimeSeconds = 0
	if _, err := client.RequestQEMU(context.Background(), base); err == nil {
		t.Fatal("non-positive lifetime was accepted")
	}
}

func TestClientRejectsInstancesWithoutIDsAndInvalidSSHTargets(t *testing.T) {
	if _, err := referenceFromProto(&instancev1.Instance{}); err == nil {
		t.Fatal("instance without manager ID was accepted")
	}
	ref := InstanceReference{ManagerInstanceID: "cm-qemu-invalid", Address: "127.0.0.1", SSHPort: 0}
	if _, err := ref.TargetRef(); err == nil {
		t.Fatal("SSH target without a returned port was accepted")
	}
}
