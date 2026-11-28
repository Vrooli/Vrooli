package permissions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/credentialpermissions"
	permission "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

type credentialMutationTargets struct{ target permission.TargetBinding }

func (t credentialMutationTargets) CurrentTargets(context.Context, string) ([]credentialpermissions.Target, error) {
	return []credentialpermissions.Target{{Binding: t.target}}, nil
}
func (t credentialMutationTargets) CurrentTarget(_ context.Context, _ string, target permission.TargetBinding) (credentialpermissions.Target, error) {
	if target != t.target {
		return credentialpermissions.Target{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stale target"))
	}
	return credentialpermissions.Target{Binding: t.target}, nil
}

type credentialMutationOwner struct {
	target             permission.TargetBinding
	grant              credentialpermissions.GrantMetadata
	receipt            credentialpermissions.MutationReceipt
	writes             int
	requestIDs         []string
	failAfterCommit    bool
	committedRequestID string
}

func (o *credentialMutationOwner) ListGrantMetadata(_ context.Context, _ string, target permission.TargetBinding) ([]credentialpermissions.GrantMetadata, error) {
	if target != o.target {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stale target"))
	}
	return []credentialpermissions.GrantMetadata{o.grant}, nil
}
func (o *credentialMutationOwner) GetGrantMetadata(_ context.Context, _ string, target permission.TargetBinding, id string) (credentialpermissions.GrantMetadata, error) {
	if target != o.target || id != o.grant.ID {
		return credentialpermissions.GrantMetadata{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("not found"))
	}
	return o.grant, nil
}
func (o *credentialMutationOwner) CreateGrantWithReceipt(_ context.Context, _ string, target permission.TargetBinding, requestID string, _ credentialpermissions.CreateMutation) (credentialpermissions.MutationReceipt, error) {
	o.writes++
	o.requestIDs = append(o.requestIDs, requestID)
	if target != o.target {
		return credentialpermissions.MutationReceipt{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stale target"))
	}
	if o.committedRequestID != "" && o.committedRequestID != requestID {
		return credentialpermissions.MutationReceipt{}, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("request ID conflicts with durable receipt"))
	}
	if o.failAfterCommit {
		o.committedRequestID = requestID
		o.failAfterCommit = false
		return credentialpermissions.MutationReceipt{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("owner response lost after durable commit"))
	}
	o.committedRequestID = requestID
	return o.receipt, nil
}
func (o *credentialMutationOwner) RevokeGrantWithReceipt(_ context.Context, _ string, target permission.TargetBinding, _ string, _ int64, requestID string) (credentialpermissions.MutationReceipt, error) {
	o.writes++
	o.requestIDs = append(o.requestIDs, requestID)
	if target != o.target {
		return credentialpermissions.MutationReceipt{}, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stale target"))
	}
	if o.committedRequestID != "" && o.committedRequestID != requestID {
		return credentialpermissions.MutationReceipt{}, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("request ID conflicts with durable receipt"))
	}
	if o.failAfterCommit {
		o.committedRequestID = requestID
		o.failAfterCommit = false
		return credentialpermissions.MutationReceipt{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("owner response lost after durable commit"))
	}
	o.committedRequestID = requestID
	return o.receipt, nil
}

type credentialRegistryService struct{ provider permission.Provider }

func (s credentialRegistryService) PermissionRegistry() (*permission.Registry, error) {
	return permission.NewRegistry(s.provider)
}

func TestCredentialGrantWriteRecoversAfterAmbiguousOwnerResponse(t *testing.T) {
	target := permission.TargetBinding{MachineID: "machine-recovery", NodeID: "node-recovery", EnrollmentGeneration: 12}
	grant := credentialpermissions.GrantMetadata{ID: "grant-recovery", NodeID: target.NodeID, LogicalID: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 1, AckedGeneration: 1, Version: "3", ReceiptAccepted: true}
	requestID := "revoke-recovery-1"
	request := permission.WriteRequest{ActorID: "operator-recovery", Owner: credentialpermissions.Owner, Target: target, ExpectedRevision: "3", IdempotencyKey: requestID, Value: []byte(`{"operation":"revoke"}`)}
	canonical := struct {
		Operation, ID, NodeID                  string
		EnrollmentGeneration, ExpectedRevision int64
	}{"revoke", grant.ID, target.NodeID, int64(target.EnrollmentGeneration), 3}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	ownerReceipt := credentialpermissions.MutationReceipt{ID: "credential-grant-mutation:" + requestID, Operation: "credential-grant.revoke", RequestID: requestID, ActorID: request.ActorID, NodeID: target.NodeID, EnrollmentGeneration: int64(target.EnrollmentGeneration), Subject: "credential-grant.revoke:" + grant.ID, ExpectedRevision: 3, ResultRevision: 4, PayloadDigest: "sha256:" + hex.EncodeToString(digest[:]), Outcome: "revoked", DeliveryState: "pending", At: time.Now().UTC(), Grant: credentialpermissions.GrantMetadata{ID: grant.ID, NodeID: target.NodeID, LogicalID: grant.LogicalID, Field: grant.Field, Class: grant.Class, Retention: grant.Retention, Generation: 1, Version: "4", RevokedAt: time.Now().UTC(), PurgeState: "pending"}}
	owner := &credentialMutationOwner{target: target, grant: grant, receipt: ownerReceipt, failAfterCommit: true}
	provider := credentialpermissions.NewProvider(credentialMutationTargets{target: target}, owner, func(context.Context) (string, error) { return "owner-token", nil }, credentialpermissions.WithOwnerSessionProvider(func(context.Context) (credentialpermissions.OwnerSession, error) {
		return credentialpermissions.OwnerSession{Token: "owner-token", OwnerID: request.ActorID}, nil
	}))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: request.ActorID, Verified: true})
	list := func() *permissionsv1.PermissionDescriptor {
		h := NewConnectHandler(credentialRegistryService{provider: provider})
		listed, err := h.ListPermissions(ctx, connect.NewRequest(&permissionsv1.ListPermissionsRequest{Owner: credentialpermissions.Owner}))
		if err != nil {
			t.Fatalf("list permissions: %v", err)
		}
		for _, d := range listed.Msg.GetPermissions() {
			if d.GetId() == "bridge/credential-grant-action/revoke/Z3JhbnQtcmVjb3Zlcnk" {
				return d
			}
		}
		t.Fatal("revoke action missing")
		return nil
	}
	write := func(d *permissionsv1.PermissionDescriptor) (*connect.Response[permissionsv1.WritePermissionResponse], error) {
		h := NewConnectHandler(credentialRegistryService{provider: provider}) // simulate onboarding API restart; owner receipt state persists
		return h.WritePermission(ctx, connect.NewRequest(&permissionsv1.WritePermissionRequest{PermissionId: d.GetId(), Owner: credentialpermissions.Owner, Target: &permissionsv1.PermissionTarget{MachineId: target.MachineID, NodeId: target.NodeID, EnrollmentGeneration: target.EnrollmentGeneration}, ExpectedRevision: request.ExpectedRevision, IdempotencyKey: requestID, ValueJson: request.Value}))
	}
	descriptor := list()
	if _, err := write(descriptor); connect.CodeOf(err) != connect.CodeUnavailable || owner.committedRequestID != requestID {
		t.Fatalf("first ambiguous write error=%v code=%s durable request=%q", err, connect.CodeOf(err), owner.committedRequestID)
	}
	replayed, err := write(descriptor)
	if err != nil {
		t.Fatalf("replay after ambiguous response: %v", err)
	}
	if got := replayed.Msg.GetReceipt(); got == nil || got.GetId() != ownerReceipt.ID || got.GetResultRevision() != "4" || got.GetIdempotencyKey() != requestID || got.GetActorId() != request.ActorID || got.GetTarget().GetEnrollmentGeneration() != target.EnrollmentGeneration || got.GetOwnerState() != "pending" || got.GetOutcome() != "revoked" {
		t.Fatalf("replay receipt=%+v", got)
	}
	if owner.writes != 2 || len(owner.requestIDs) != 2 || owner.requestIDs[0] != requestID || owner.requestIDs[1] != requestID {
		t.Fatalf("owner calls=%d request IDs=%v", owner.writes, owner.requestIDs)
	}
}

func TestCredentialGrantTypedConnectWriteUsesOwnerReceiptBinding(t *testing.T) {
	target := permission.TargetBinding{MachineID: "machine-1", NodeID: "node-1", EnrollmentGeneration: 7}
	grant := credentialpermissions.GrantMetadata{ID: "grant-1", NodeID: "node-1", LogicalID: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 2, AckedGeneration: 2, Version: "3", ReceiptAccepted: true, GrantedAt: time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)}
	request := permission.WriteRequest{ActorID: "operator-1", Owner: credentialpermissions.Owner, Target: target, ExpectedRevision: "3", IdempotencyKey: "revoke-request-1", Value: []byte(`{"operation":"revoke"}`)}
	canonical := struct {
		Operation, ID, NodeID                  string
		EnrollmentGeneration, ExpectedRevision int64
	}{"revoke", "grant-1", "node-1", 7, 3}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded)
	ownerReceipt := credentialpermissions.MutationReceipt{ID: "credential-grant-mutation:revoke-request-1", Operation: "credential-grant.revoke", RequestID: request.IdempotencyKey, ActorID: request.ActorID, NodeID: target.NodeID, EnrollmentGeneration: 7, Subject: "credential-grant.revoke:grant-1", ExpectedRevision: 3, ResultRevision: 4, PayloadDigest: "sha256:" + hex.EncodeToString(digest[:]), Outcome: "revoked", At: time.Now().UTC(), Grant: credentialpermissions.GrantMetadata{ID: "grant-1", NodeID: "node-1", LogicalID: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 2, AckedGeneration: 2, Version: "4", RevokedAt: time.Now().UTC(), PurgeState: "pending"}}
	owner := &credentialMutationOwner{target: target, grant: grant, receipt: ownerReceipt}
	provider := credentialpermissions.NewProvider(credentialMutationTargets{target: target}, owner, func(context.Context) (string, error) { return "owner-token", nil }, credentialpermissions.WithOwnerSessionProvider(func(context.Context) (credentialpermissions.OwnerSession, error) {
		return credentialpermissions.OwnerSession{Token: "owner-token", OwnerID: "operator-1"}, nil
	}))
	handler := NewConnectHandler(credentialRegistryService{provider: provider})
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-1", Verified: true})
	listed, err := handler.ListPermissions(ctx, connect.NewRequest(&permissionsv1.ListPermissionsRequest{Owner: credentialpermissions.Owner}))
	if err != nil {
		t.Fatal(err)
	}
	var revoke *permissionsv1.PermissionDescriptor
	for _, descriptor := range listed.Msg.GetPermissions() {
		if descriptor.GetId() == "bridge/credential-grant-action/revoke/Z3JhbnQtMQ" {
			revoke = descriptor
		}
	}
	if revoke == nil || !revoke.GetCanWrite() || revoke.GetCanRead() || revoke.GetTarget().GetNodeId() != target.NodeID || revoke.GetTarget().GetEnrollmentGeneration() != target.EnrollmentGeneration {
		t.Fatalf("revoke descriptor=%+v", revoke)
	}
	written, err := handler.WritePermission(ctx, connect.NewRequest(&permissionsv1.WritePermissionRequest{PermissionId: revoke.GetId(), Owner: credentialpermissions.Owner, Target: &permissionsv1.PermissionTarget{MachineId: target.MachineID, NodeId: target.NodeID, EnrollmentGeneration: target.EnrollmentGeneration}, ExpectedRevision: "3", IdempotencyKey: request.IdempotencyKey, ValueJson: request.Value}))
	if err != nil {
		t.Fatal(err)
	}
	if owner.writes != 1 || written.Msg.GetReceipt().GetActorId() != "operator-1" || written.Msg.GetReceipt().GetResultRevision() != "4" || written.Msg.GetReceipt().GetIdempotencyKey() != request.IdempotencyKey || written.Msg.GetReceipt().GetTarget().GetEnrollmentGeneration() != target.EnrollmentGeneration {
		t.Fatalf("owner writes=%d receipt=%+v", owner.writes, written.Msg.GetReceipt())
	}
}
