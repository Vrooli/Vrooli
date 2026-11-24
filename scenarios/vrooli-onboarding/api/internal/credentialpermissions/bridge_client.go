package credentialpermissions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	grantv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

type MetadataRPC interface {
	ListGrantMetadata(context.Context, *connect.Request[grantv1.ListGrantMetadataRequest]) (*connect.Response[grantv1.ListGrantMetadataResponse], error)
	GetGrantMetadata(context.Context, *connect.Request[grantv1.GetGrantMetadataRequest]) (*connect.Response[grantv1.CredentialGrantMetadata], error)
	CreateGrantWithReceipt(context.Context, *connect.Request[grantv1.CreateGrantWithReceiptRequest]) (*connect.Response[grantv1.CredentialGrantMutationReceipt], error)
	RevokeGrantWithReceipt(context.Context, *connect.Request[grantv1.RevokeGrantWithReceiptRequest]) (*connect.Response[grantv1.CredentialGrantMutationReceipt], error)
}

// ConnectGrantClient adapts Bridge's side-effect-free metadata projection.
// It intentionally does not implement or call the legacy ListGrants method.
type ConnectGrantClient struct{ rpc MetadataRPC }

func NewConnectGrantClient(rpc MetadataRPC) *ConnectGrantClient {
	return &ConnectGrantClient{rpc: rpc}
}

func (c *ConnectGrantClient) ListGrantMetadata(ctx context.Context, token string, target permissions.TargetBinding) ([]GrantMetadata, error) {
	if err := validateTarget(target); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if c == nil || c.rpc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Bridge metadata client is unavailable"))
	}
	req := connect.NewRequest(&grantv1.ListGrantMetadataRequest{NodeId: target.NodeID, EnrollmentGeneration: int64(target.EnrollmentGeneration)})
	req.Header().Set("Authorization", token)
	resp, err := c.rpc.ListGrantMetadata(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Msg == nil {
		return nil, connect.NewError(connect.CodeDataLoss, fmt.Errorf("Bridge returned no credential metadata response"))
	}
	if resp.Msg.GetNodeId() != target.NodeID || resp.Msg.GetEnrollmentGeneration() != int64(target.EnrollmentGeneration) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("Bridge metadata target binding changed"))
	}
	out := make([]GrantMetadata, 0, len(resp.Msg.GetGrants()))
	for _, wire := range resp.Msg.GetGrants() {
		grant, err := metadataFromProto(wire)
		if err != nil {
			return nil, err
		}
		if grant.NodeID != target.NodeID {
			return nil, connect.NewError(connect.CodeDataLoss, fmt.Errorf("Bridge returned cross-target grant metadata"))
		}
		out = append(out, grant)
	}
	return out, nil
}

func (c *ConnectGrantClient) GetGrantMetadata(ctx context.Context, token string, target permissions.TargetBinding, id string) (GrantMetadata, error) {
	if err := validateTarget(target); err != nil {
		return GrantMetadata{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if strings.TrimSpace(id) == "" || id != strings.TrimSpace(id) {
		return GrantMetadata{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("opaque grant id is required"))
	}
	if c == nil || c.rpc == nil {
		return GrantMetadata{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Bridge metadata client is unavailable"))
	}
	req := connect.NewRequest(&grantv1.GetGrantMetadataRequest{Id: id, NodeId: target.NodeID, EnrollmentGeneration: int64(target.EnrollmentGeneration)})
	req.Header().Set("Authorization", token)
	resp, err := c.rpc.GetGrantMetadata(ctx, req)
	if err != nil {
		return GrantMetadata{}, err
	}
	if resp == nil || resp.Msg == nil {
		return GrantMetadata{}, connect.NewError(connect.CodeDataLoss, fmt.Errorf("Bridge returned no credential metadata"))
	}
	grant, err := metadataFromProto(resp.Msg)
	if err != nil {
		return GrantMetadata{}, err
	}
	if grant.ID != id || grant.NodeID != target.NodeID {
		return GrantMetadata{}, connect.NewError(connect.CodeDataLoss, fmt.Errorf("Bridge returned mismatched credential metadata"))
	}
	return grant, nil
}

func (c *ConnectGrantClient) CreateGrantWithReceipt(ctx context.Context, token string, target permissions.TargetBinding, requestID string, value CreateMutation) (MutationReceipt, error) {
	if err := validateTarget(target); err != nil {
		return MutationReceipt{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if c == nil || c.rpc == nil || strings.TrimSpace(token) == "" || strings.TrimSpace(requestID) == "" {
		return MutationReceipt{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("Bridge credential mutation client is unavailable"))
	}
	req := connect.NewRequest(&grantv1.CreateGrantWithReceiptRequest{NodeId: target.NodeID, EnrollmentGeneration: int64(target.EnrollmentGeneration), LogicalId: value.LogicalID, Field: value.Field, Class: value.Class, Retention: value.Retention, Generation: value.Generation, ExpectedRevision: 0, RequestId: requestID})
	req.Header().Set("Authorization", token)
	resp, err := c.rpc.CreateGrantWithReceipt(ctx, req)
	if err != nil {
		return MutationReceipt{}, err
	}
	return mutationReceiptFromProto(resp, requestID, target)
}

func (c *ConnectGrantClient) RevokeGrantWithReceipt(ctx context.Context, token string, target permissions.TargetBinding, id string, revision int64, requestID string) (MutationReceipt, error) {
	if err := validateTarget(target); err != nil {
		return MutationReceipt{}, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if c == nil || c.rpc == nil || strings.TrimSpace(token) == "" || strings.TrimSpace(id) == "" || strings.TrimSpace(requestID) == "" || revision <= 0 {
		return MutationReceipt{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("Bridge credential revoke binding is invalid"))
	}
	req := connect.NewRequest(&grantv1.RevokeGrantWithReceiptRequest{Id: id, NodeId: target.NodeID, EnrollmentGeneration: int64(target.EnrollmentGeneration), ExpectedRevision: revision, RequestId: requestID})
	req.Header().Set("Authorization", token)
	resp, err := c.rpc.RevokeGrantWithReceipt(ctx, req)
	if err != nil {
		return MutationReceipt{}, err
	}
	return mutationReceiptFromProto(resp, requestID, target)
}

func mutationReceiptFromProto(resp *connect.Response[grantv1.CredentialGrantMutationReceipt], requestID string, target permissions.TargetBinding) (MutationReceipt, error) {
	if resp == nil || resp.Msg == nil || resp.Msg.GetGrant() == nil {
		return MutationReceipt{}, connect.NewError(connect.CodeDataLoss, fmt.Errorf("Bridge returned no credential mutation receipt"))
	}
	w := resp.Msg
	grant, err := metadataFromProto(w.GetGrant())
	if err != nil {
		return MutationReceipt{}, err
	}
	if w.GetRequestId() != requestID || w.GetTargetNodeId() != target.NodeID || w.GetEnrollmentGeneration() != int64(target.EnrollmentGeneration) || grant.NodeID != target.NodeID || strings.TrimSpace(w.GetActorId()) == "" || strings.TrimSpace(w.GetId()) == "" || w.GetResultRevision() <= 0 || w.GetExpectedRevision() < 0 || w.GetPayloadDigest() == "" || w.GetOutcome() == "" || !validDeliveryState(w.GetDeliveryState()) || w.GetCreatedAt() == nil || !w.GetCreatedAt().IsValid() {
		return MutationReceipt{}, connect.NewError(connect.CodeDataLoss, fmt.Errorf("Bridge credential mutation receipt binding mismatch"))
	}
	return MutationReceipt{ID: w.GetId(), Operation: w.GetOperation(), RequestID: w.GetRequestId(), ActorID: w.GetActorId(), NodeID: w.GetTargetNodeId(), EnrollmentGeneration: w.GetEnrollmentGeneration(), Subject: w.GetSubject(), ExpectedRevision: w.GetExpectedRevision(), ResultRevision: w.GetResultRevision(), PayloadDigest: w.GetPayloadDigest(), Outcome: w.GetOutcome(), DeliveryState: w.GetDeliveryState(), At: w.GetCreatedAt().AsTime().UTC(), Grant: grant}, nil
}

func validDeliveryState(state string) bool {
	switch state {
	case "pending", "dispatched", "retryable", "confirmed", "refused":
		return true
	default:
		return false
	}
}

func metadataFromProto(w *grantv1.CredentialGrantMetadata) (GrantMetadata, error) {
	if w == nil {
		return GrantMetadata{}, connect.NewError(connect.CodeDataLoss, fmt.Errorf("Bridge returned empty credential metadata"))
	}
	out := GrantMetadata{ID: w.GetId(), NodeID: w.GetNodeId(), LogicalID: w.GetLogicalId(), Field: w.GetField(), Class: w.GetClass(), Retention: w.GetRetention(), Generation: w.GetGeneration(), AckedGeneration: w.GetAckedGeneration(), ReceiptAccepted: w.GetReceiptAccepted(), PurgeState: w.GetPurgeState(), PurgeAccepted: w.GetPurgeReceiptAccepted(), Version: w.GetVersion()}
	for _, pair := range []struct {
		stamp interface{ AsTime() time.Time }
		dest  *time.Time
	}{{w.GetGrantedAt(), &out.GrantedAt}, {w.GetRevokedAt(), &out.RevokedAt}, {w.GetReceiptAt(), &out.ReceiptAt}, {w.GetPurgeReceiptAt(), &out.PurgeReceiptAt}} {
		if pair.stamp != nil {
			*pair.dest = pair.stamp.AsTime().UTC()
		}
	}
	if err := validateGrant(out, permissions.TargetBinding{NodeID: out.NodeID}); err != nil {
		return GrantMetadata{}, connect.NewError(connect.CodeDataLoss, err)
	}
	return out, nil
}
