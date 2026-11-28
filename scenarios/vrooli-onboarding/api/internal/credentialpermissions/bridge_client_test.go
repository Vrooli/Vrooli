package credentialpermissions

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	grantv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeMetadataRPC struct {
	listed        *grantv1.ListGrantMetadataRequest
	got           *grantv1.GetGrantMetadataRequest
	list          *grantv1.ListGrantMetadataResponse
	item          *grantv1.CredentialGrantMetadata
	created       *grantv1.CreateGrantWithReceiptRequest
	revoked       *grantv1.RevokeGrantWithReceiptRequest
	createReceipt *grantv1.CredentialGrantMutationReceipt
	revokeReceipt *grantv1.CredentialGrantMutationReceipt
	err           error
}

func (f *fakeMetadataRPC) ListGrantMetadata(_ context.Context, r *connect.Request[grantv1.ListGrantMetadataRequest]) (*connect.Response[grantv1.ListGrantMetadataResponse], error) {
	f.listed = r.Msg
	if r.Header().Get("Authorization") != "Bearer owner-token" {
		return nil, connect.NewError(connect.CodeUnauthenticated, connect.NewError(connect.CodeUnauthenticated, nil))
	}
	return connect.NewResponse(f.list), f.err
}
func (f *fakeMetadataRPC) GetGrantMetadata(_ context.Context, r *connect.Request[grantv1.GetGrantMetadataRequest]) (*connect.Response[grantv1.CredentialGrantMetadata], error) {
	f.got = r.Msg
	if r.Header().Get("Authorization") != "Bearer owner-token" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	return connect.NewResponse(f.item), f.err
}
func (f *fakeMetadataRPC) CreateGrantWithReceipt(_ context.Context, r *connect.Request[grantv1.CreateGrantWithReceiptRequest]) (*connect.Response[grantv1.CredentialGrantMutationReceipt], error) {
	f.created = r.Msg
	if r.Header().Get("Authorization") != "Bearer owner-token" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	return connect.NewResponse(f.createReceipt), f.err
}
func (f *fakeMetadataRPC) RevokeGrantWithReceipt(_ context.Context, r *connect.Request[grantv1.RevokeGrantWithReceiptRequest]) (*connect.Response[grantv1.CredentialGrantMutationReceipt], error) {
	f.revoked = r.Msg
	if r.Header().Get("Authorization") != "Bearer owner-token" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	return connect.NewResponse(f.revokeReceipt), f.err
}

func TestConnectGrantClientUsesMetadataRPCAndBindsGeneration(t *testing.T) {
	target := permissions.TargetBinding{MachineID: "machine-a", NodeID: "node-a", EnrollmentGeneration: 7}
	metadata := &grantv1.CredentialGrantMetadata{Id: "grant-a", NodeId: "node-a", LogicalId: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 4, AckedGeneration: 4, GrantedAt: timestamppb.Now(), ReceiptAccepted: true, Version: "v4"}
	rpc := &fakeMetadataRPC{list: &grantv1.ListGrantMetadataResponse{NodeId: "node-a", EnrollmentGeneration: 7, Grants: []*grantv1.CredentialGrantMetadata{metadata}}, item: metadata}
	client := NewConnectGrantClient(rpc)
	got, err := client.ListGrantMetadata(context.Background(), "Bearer owner-token", target)
	if err != nil {
		t.Fatal(err)
	}
	if rpc.listed.GetNodeId() != "node-a" || rpc.listed.GetEnrollmentGeneration() != 7 || len(got) != 1 || got[0].ID != "grant-a" || got[0].Version != "v4" {
		t.Fatalf("list binding=%+v grant=%+v", rpc.listed, got)
	}
	read, err := client.GetGrantMetadata(context.Background(), "Bearer owner-token", target, "grant-a")
	if err != nil {
		t.Fatal(err)
	}
	if rpc.got.GetNodeId() != "node-a" || rpc.got.GetEnrollmentGeneration() != 7 || rpc.got.GetId() != "grant-a" || read.ID != "grant-a" {
		t.Fatalf("get binding=%+v grant=%+v", rpc.got, read)
	}
}

func TestConnectGrantClientRejectsMismatchedSnapshotAndEmptyTarget(t *testing.T) {
	target := permissions.TargetBinding{MachineID: "machine-a", NodeID: "node-a", EnrollmentGeneration: 7}
	rpc := &fakeMetadataRPC{list: &grantv1.ListGrantMetadataResponse{NodeId: "node-a", EnrollmentGeneration: 8}}
	client := NewConnectGrantClient(rpc)
	if _, err := client.ListGrantMetadata(context.Background(), "Bearer owner-token", target); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale metadata generation=%v", err)
	}
	if _, err := client.ListGrantMetadata(context.Background(), "Bearer owner-token", permissions.TargetBinding{}); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("empty target=%v", err)
	}
}

func TestConnectGrantClientBindsDurableCreateAndRevokeReceipts(t *testing.T) {
	target := permissions.TargetBinding{MachineID: "machine-a", NodeID: "node-a", EnrollmentGeneration: 7}
	metadata := &grantv1.CredentialGrantMetadata{Id: "grant-a", NodeId: "node-a", LogicalId: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 1, Version: "2", RevokedAt: timestamppb.Now()}
	created := &grantv1.CredentialGrantMutationReceipt{Id: "credential-grant-mutation:create-1", Operation: "credential-grant.create", RequestId: "create-1", ActorId: "operator-a", TargetNodeId: "node-a", EnrollmentGeneration: 7, Subject: "credential-grant.create", ExpectedRevision: 0, ResultRevision: 1, PayloadDigest: "sha256:" + strings.Repeat("a", 64), Outcome: "created", DeliveryState: "pending", CreatedAt: timestamppb.Now(), Grant: metadata}
	revoked := &grantv1.CredentialGrantMutationReceipt{Id: "credential-grant-mutation:revoke-1", Operation: "credential-grant.revoke", RequestId: "revoke-1", ActorId: "operator-a", TargetNodeId: "node-a", EnrollmentGeneration: 7, Subject: "credential-grant.revoke:grant-a", ExpectedRevision: 1, ResultRevision: 2, PayloadDigest: "sha256:" + strings.Repeat("b", 64), Outcome: "revoked", DeliveryState: "retryable", CreatedAt: timestamppb.Now(), Grant: metadata}
	rpc := &fakeMetadataRPC{createReceipt: created, revokeReceipt: revoked}
	client := NewConnectGrantClient(rpc)
	create, err := client.CreateGrantWithReceipt(context.Background(), "Bearer owner-token", target, "create-1", CreateMutation{LogicalID: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	if rpc.created.GetNodeId() != target.NodeID || rpc.created.GetEnrollmentGeneration() != 7 || rpc.created.GetExpectedRevision() != 0 || rpc.created.GetRequestId() != "create-1" || create.ActorID != "operator-a" || create.ResultRevision != 1 || create.DeliveryState != "pending" {
		t.Fatalf("create binding=%+v receipt=%+v", rpc.created, create)
	}
	revoke, err := client.RevokeGrantWithReceipt(context.Background(), "Bearer owner-token", target, "grant-a", 1, "revoke-1")
	if err != nil {
		t.Fatal(err)
	}
	if rpc.revoked.GetNodeId() != target.NodeID || rpc.revoked.GetEnrollmentGeneration() != 7 || rpc.revoked.GetId() != "grant-a" || rpc.revoked.GetExpectedRevision() != 1 || rpc.revoked.GetRequestId() != "revoke-1" || revoke.Subject != "credential-grant.revoke:grant-a" || revoke.DeliveryState != "retryable" {
		t.Fatalf("revoke binding=%+v receipt=%+v", rpc.revoked, revoke)
	}
}

func TestConnectGrantClientRejectsUnknownDeliveryState(t *testing.T) {
	target := permissions.TargetBinding{MachineID: "machine-a", NodeID: "node-a", EnrollmentGeneration: 7}
	metadata := &grantv1.CredentialGrantMetadata{Id: "grant-a", NodeId: "node-a", LogicalId: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 1, Version: "1"}
	receipt := &grantv1.CredentialGrantMutationReceipt{Id: "receipt", Operation: "credential-grant.create", RequestId: "request-1", ActorId: "operator-a", TargetNodeId: "node-a", EnrollmentGeneration: 7, Subject: "credential-grant.create", ExpectedRevision: 0, ResultRevision: 1, PayloadDigest: "sha256:" + strings.Repeat("a", 64), Outcome: "created", DeliveryState: "completed", CreatedAt: timestamppb.Now(), Grant: metadata}
	client := NewConnectGrantClient(&fakeMetadataRPC{createReceipt: receipt})
	if _, err := client.CreateGrantWithReceipt(context.Background(), "Bearer owner-token", target, "request-1", CreateMutation{LogicalID: "service/api", Field: "token", Class: "user_prompt", Retention: "ephemeral", Generation: 1}); connect.CodeOf(err) != connect.CodeDataLoss {
		t.Fatalf("unknown delivery state accepted: %v", err)
	}
}
