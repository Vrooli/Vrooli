package permissions

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions"
	permissionsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions/permissionsv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/decisionpermissions"
	permission "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/standinggrantpermissions"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RegistryProvider gives the transport the same composed owner registry used
// by domain handlers. Authentication and policy remain in the registry.
type RegistryProvider interface {
	PermissionRegistry() (*permission.Registry, error)
}

type connectHandler struct{ service RegistryProvider }

func NewConnectHandler(service RegistryProvider) permissionsv1connect.PermissionRegistryServiceHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListPermissions(ctx context.Context, req *connect.Request[permissionsv1.ListPermissionsRequest]) (*connect.Response[permissionsv1.ListPermissionsResponse], error) {
	ctx = decisionpermissions.WithIncomingOwnerProof(ctx, req.Header())
	ctx = standinggrantpermissions.WithIncomingOwnerProof(ctx, req.Header())
	if _, err := verifiedActor(ctx); err != nil {
		return nil, err
	}
	target := fromTarget(req.Msg.GetTarget())
	targetSpecified := target.MachineID != "" || target.NodeID != "" || target.EnrollmentGeneration != 0
	if targetSpecified && (target.MachineID == "" || target.NodeID == "" || target.EnrollmentGeneration == 0) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target filtering requires machine, node, and enrollment generation"))
	}
	registry, err := h.registry()
	if err != nil {
		return nil, err
	}
	var entries []permission.Descriptor
	if owner := req.Msg.GetOwner(); owner != "" {
		entries, err = registry.InventoryForOwner(ctx, owner)
	} else {
		entries, err = registry.Inventory(ctx)
	}
	if err != nil {
		return nil, mapError(err)
	}
	response := &permissionsv1.ListPermissionsResponse{}
	for _, entry := range entries {
		// An omitted target means discover this verified actor's full owner
		// inventory. Reads and writes still require the exact target binding.
		if targetSpecified && entry.TargetBound && entry.Target != target {
			continue
		}
		response.Permissions = append(response.Permissions, toDescriptor(entry))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ReadPermission(ctx context.Context, req *connect.Request[permissionsv1.ReadPermissionRequest]) (*connect.Response[permissionsv1.ReadPermissionResponse], error) {
	ctx = decisionpermissions.WithIncomingOwnerProof(ctx, req.Header())
	ctx = standinggrantpermissions.WithIncomingOwnerProof(ctx, req.Header())
	actor, err := verifiedActor(ctx)
	if err != nil {
		return nil, err
	}
	registry, err := h.registry()
	if err != nil {
		return nil, err
	}
	result, err := registry.Read(ctx, req.Msg.GetPermissionId(), permission.ReadRequest{ActorID: actor, Owner: req.Msg.GetOwner(), Target: fromTarget(req.Msg.GetTarget())})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&permissionsv1.ReadPermissionResponse{
		PermissionId: result.PermissionID, Owner: result.Owner, Target: toTarget(result.Target),
		Revision: result.Revision, ValueJson: append([]byte(nil), result.Value...), ValueDigest: result.ValueDigest,
		ObservedAt: timestamppb.New(result.ObservedAt),
	}), nil
}

func (h *connectHandler) WritePermission(ctx context.Context, req *connect.Request[permissionsv1.WritePermissionRequest]) (*connect.Response[permissionsv1.WritePermissionResponse], error) {
	actor, err := verifiedActor(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetIdempotencyKey() == "" || req.Msg.GetExpectedRevision() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("expected_revision and idempotency_key are required"))
	}
	registry, err := h.registry()
	if err != nil {
		return nil, err
	}
	receipt, err := registry.Write(ctx, req.Msg.GetPermissionId(), permission.WriteRequest{
		ActorID: actor, Owner: req.Msg.GetOwner(), Target: fromTarget(req.Msg.GetTarget()),
		ExpectedRevision: req.Msg.GetExpectedRevision(), IdempotencyKey: req.Msg.GetIdempotencyKey(),
		Value: append([]byte(nil), req.Msg.GetValueJson()...),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&permissionsv1.WritePermissionResponse{Receipt: toReceipt(receipt)}), nil
}

func (h *connectHandler) registry() (*permission.Registry, error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("permission registry is unavailable"))
	}
	registry, err := h.service.PermissionRegistry()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return registry, nil
}

func verifiedActor(ctx context.Context) (string, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || principal.Subject == "" {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified onboarding principal required"))
	}
	return principal.Subject, nil
}

func mapError(err error) error {
	if errors.Is(err, operatorstate.ErrMutationReceiptConflict) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	if errors.Is(err, operatorstate.ErrRevisionConflict) {
		return connect.NewError(connect.CodeAborted, err)
	}
	if code := connect.CodeOf(err); code != connect.CodeUnknown {
		return connect.NewError(code, err)
	}
	return connect.NewError(connect.CodePermissionDenied, err)
}

func fromTarget(value *permissionsv1.PermissionTarget) permission.TargetBinding {
	if value == nil {
		return permission.TargetBinding{}
	}
	return permission.TargetBinding{MachineID: value.GetMachineId(), NodeID: value.GetNodeId(), EnrollmentGeneration: value.GetEnrollmentGeneration()}
}

func toTarget(value permission.TargetBinding) *permissionsv1.PermissionTarget {
	return &permissionsv1.PermissionTarget{MachineId: value.MachineID, NodeId: value.NodeID, EnrollmentGeneration: value.EnrollmentGeneration}
}

func toDescriptor(value permission.Descriptor) *permissionsv1.PermissionDescriptor {
	return &permissionsv1.PermissionDescriptor{
		Version: value.Version, Id: value.ID, Owner: value.Owner, PrincipalKind: value.PrincipalKind,
		PrincipalId: value.PrincipalID, Scope: value.Scope, SubjectKind: value.SubjectKind,
		SubjectId: value.SubjectID, TargetBound: value.TargetBound, Target: toTarget(value.Target),
		Source: value.Source, SchemaVersion: value.SchemaVersion, CanRead: value.CanRead,
		CanWrite: value.CanWrite, Sensitivity: value.Sensitivity,
	}
}

func toReceipt(value permission.Receipt) *permissionsv1.PermissionReceipt {
	return &permissionsv1.PermissionReceipt{
		Id: value.ID, PermissionId: value.PermissionID, Owner: value.Owner, ActorId: value.ActorID,
		Target: toTarget(value.Target), ExpectedRevision: value.ExpectedRevision,
		ResultRevision: value.ResultRevision, IdempotencyKey: value.IdempotencyKey,
		RequestDigest: value.RequestDigest, DecisionRef: value.DecisionRef, Outcome: value.Outcome, OwnerState: value.OwnerState,
		At: timestamppb.New(value.At),
	}
}
