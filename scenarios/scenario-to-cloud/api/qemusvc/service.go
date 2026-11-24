package qemusvc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/identity"
	qemuv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/qemu"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/qemu/qemuv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"scenario-to-cloud/computemanager"
)

type Manager interface {
	RequestQEMU(context.Context, computemanager.QEMURequest) (computemanager.InstanceReference, error)
	GetInstance(context.Context, string) (computemanager.InstanceReference, error)
	DestroyInstance(context.Context, string) (computemanager.InstanceReference, error)
}

type PublicKeyResolver func(context.Context) (string, error)

// Service is the generated S2C Connect consumer boundary over Compute Manager.
// It owns no QEMU process, disk, or provider lifecycle state.
type Service struct {
	manager   Manager
	publicKey PublicKeyResolver
}

func New(manager Manager, publicKey PublicKeyResolver) *Service {
	return &Service{manager: manager, publicKey: publicKey}
}

func (s *Service) Handler(opts ...connect.HandlerOption) (string, http.Handler) {
	return qemuv1connect.NewQEMUServiceHandler(s, opts...)
}

func (s *Service) Acquire(ctx context.Context, request *connect.Request[qemuv1.AcquireRequest]) (*connect.Response[qemuv1.InstanceReference], error) {
	if s == nil || s.manager == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("compute-manager client is unavailable"))
	}
	item := request.Msg
	profile := strings.TrimSpace(item.GetProfile())
	if (profile != "headless-linux" && profile != "desktop-linux") || item.GetSize() != profile || item.GetRegion() != "local" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("local QEMU requires region=local and size=profile=headless-linux or desktop-linux"))
	}
	if strings.TrimSpace(item.GetIdempotencyKey()) == "" || strings.TrimSpace(item.GetImage()) == "" || item.GetLifetimeSeconds() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("idempotency_key, image and positive lifetime are required"))
	}
	if s.publicKey == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("operator SSH public key is unavailable through credential authority"))
	}
	publicKey, err := s.publicKey(ctx)
	if err != nil || strings.TrimSpace(publicKey) == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("operator SSH public key is unavailable through credential authority"))
	}
	ref, err := s.manager.RequestQEMU(ctx, computemanager.QEMURequest{
		IdempotencyKey: item.GetIdempotencyKey(), Region: item.GetRegion(), Size: item.GetSize(),
		Image: item.GetImage(), LifetimeSeconds: item.GetLifetimeSeconds(), Memory: item.GetMemory(),
		CPUs: item.GetCpus(), SSHUser: item.GetSshUser(), SSHPort: item.GetSshPort(),
		Profile: profile, SSHAuthorizedKey: strings.TrimSpace(publicKey),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	result, err := reference(ref)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(result), nil
}

func (s *Service) Get(ctx context.Context, request *connect.Request[qemuv1.GetRequest]) (*connect.Response[qemuv1.InstanceReference], error) {
	if s == nil || s.manager == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("compute-manager client is unavailable"))
	}
	id := strings.TrimSpace(request.Msg.GetInstanceId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("instance_id is required"))
	}
	ref, err := s.manager.GetInstance(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	result, err := reference(ref)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(result), nil
}

func (s *Service) Release(ctx context.Context, request *connect.Request[qemuv1.ReleaseRequest]) (*connect.Response[qemuv1.ReleaseResponse], error) {
	if s == nil || s.manager == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("compute-manager client is unavailable"))
	}
	id := strings.TrimSpace(request.Msg.GetInstanceId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("instance_id is required"))
	}
	ref, err := s.manager.DestroyInstance(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&qemuv1.ReleaseResponse{InstanceId: ref.ManagerInstanceID, Status: "destroyed"}), nil
}

func reference(ref computemanager.InstanceReference) (*qemuv1.InstanceReference, error) {
	target, err := ref.TargetRef()
	if err != nil {
		return nil, err
	}
	result := &qemuv1.InstanceReference{
		ManagerInstanceId: ref.ManagerInstanceID, Provider: ref.Provider, Region: ref.Region,
		Size: ref.Size, Image: ref.Image,
		Target: &identityv1.TargetRef{MachineId: target.MachineID, ComputeManagerInstanceId: target.ComputeManagerID, NodeId: target.NodeID,
			EnrollmentGeneration: target.EnrollmentGeneration, Transport: target.Transport,
			Locator: &identityv1.TargetLocator{Host: target.Locator.Host, Port: int32(target.Locator.Port), User: target.Locator.User, Workdir: target.Locator.Workdir}},
	}
	if !ref.ExpiresAt.IsZero() {
		result.ExpiresAt = timestamppb.New(ref.ExpiresAt)
	}
	return result, nil
}
