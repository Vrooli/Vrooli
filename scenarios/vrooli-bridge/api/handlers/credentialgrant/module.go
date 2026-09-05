package credentialgrant

import (
	"context"
	"errors"
	"fmt"
	"log"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/channelsign"
	internalgrant "vrooli-bridge/internal/credentialgrant"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/presence"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	grantv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	grantconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant/credentialgrant_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ModuleDeps struct {
	Service          *internalgrant.Service
	Presence         *presence.Hub
	Signer           channelsign.Signer
	SealingPublicKey func(context.Context, string) ([]byte, error)
	ResolveValue     func(context.Context, string, string) (string, error)
	ProvisionValue   func(context.Context, string, string, string) error
	NodeVerifier     *nodeauth.Verifier
	Logger           *log.Logger
}

type handler struct {
	service          *internalgrant.Service
	presence         *presence.Hub
	signer           channelsign.Signer
	sealingPublicKey func(context.Context, string) ([]byte, error)
	resolveValue     func(context.Context, string, string) (string, error)
	provisionValue   func(context.Context, string, string, string) error
	nodeVerifier     *nodeauth.Verifier
	logger           *log.Logger
}

func NewHandler(deps ModuleDeps) *handler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &handler{service: deps.Service, presence: deps.Presence, signer: deps.Signer, sealingPublicKey: deps.SealingPublicKey, resolveValue: deps.ResolveValue, provisionValue: deps.ProvisionValue, nodeVerifier: deps.NodeVerifier, logger: deps.Logger}
}

func Module(h *handler) module.Module {
	path, connectHandler := grantconnect.NewCredentialGrantServiceHandler(h)
	return module.Module{
		Name: "credentialgrant",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func (h *handler) CreateGrant(ctx context.Context, req *connect.Request[grantv1.CreateGrantRequest]) (*connect.Response[grantv1.CredentialGrant], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	grant, err := h.service.Create(ctx, internalgrant.CreateInput{
		NodeID: req.Msg.GetNodeId(), LogicalID: req.Msg.GetLogicalId(), Field: req.Msg.GetField(),
		Class: internalgrant.Class(req.Msg.GetClass()), Retention: internalgrant.Retention(req.Msg.GetRetention()), Generation: req.Msg.GetGeneration(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.deliverGrant(ctx, grant); err != nil {
		h.logger.Printf("credential grant %q delivery deferred: %v", grant.ID, err)
	}
	return connect.NewResponse(toProto(grant)), nil
}

func (h *handler) AnswerSecret(ctx context.Context, req *connect.Request[grantv1.AnswerSecretRequest]) (*connect.Response[grantv1.CredentialGrant], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if h.provisionValue == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("credential provisioning is unavailable"))
	}
	if req.Msg.GetValue() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("secret value is required"))
	}
	grant, err := h.service.Create(ctx, internalgrant.CreateInput{NodeID: req.Msg.GetNodeId(), LogicalID: req.Msg.GetLogicalId(), Field: req.Msg.GetField(), Class: internalgrant.Class(req.Msg.GetClass()), Retention: internalgrant.Retention(req.Msg.GetRetention()), Generation: 1})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	value := req.Msg.GetValue()
	defer func() { value = "" }()
	if err := h.provisionValue(ctx, grant.LogicalID, grant.Field, value); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("credential authority rejected the answer: %w", err))
	}
	if err := h.deliverGrant(ctx, grant); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("sealed credential delivery failed: %w", err))
	}
	return connect.NewResponse(toProto(grant)), nil
}

func (h *handler) ListGrants(ctx context.Context, req *connect.Request[grantv1.ListGrantsRequest]) (*connect.Response[grantv1.ListGrantsResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	grants, err := h.service.List(ctx, req.Msg.GetNodeId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &grantv1.ListGrantsResponse{Grants: make([]*grantv1.CredentialGrant, 0, len(grants))}
	for _, grant := range grants {
		response.Grants = append(response.Grants, toProto(grant))
	}
	// The response reconciles metadata; the delivery below reconciles the
	// current value.  Keeping both under the same authenticated node request
	// means an agent that was offline during rotation converges on reconnect
	// without a second operator action.  Values remain sealed for the node and
	// are never included in the RPC response or its logs.
	for _, grant := range grants {
		if err := h.deliverGrant(ctx, grant); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("deliver credential grant %q after sync: %w", grant.ID, err))
		}
	}
	return connect.NewResponse(response), nil
}

func (h *handler) RevokeGrant(ctx context.Context, req *connect.Request[grantv1.RevokeGrantRequest]) (*connect.Response[grantv1.CredentialGrant], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	grant, err := h.service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err := h.service.Revoke(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.deliverPurge(ctx, grant); err != nil {
		h.logger.Printf("credential grant %q purge delivery deferred: %v", grant.ID, err)
	}
	return connect.NewResponse(&grantv1.CredentialGrant{Id: req.Msg.GetId()}), nil
}

func (h *handler) RotateAddress(ctx context.Context, req *connect.Request[grantv1.RotateAddressRequest]) (*connect.Response[grantv1.RotationResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	generation, grants, err := h.service.Rotate(ctx, req.Msg.GetLogicalId(), req.Msg.GetField())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	response := &grantv1.RotationResponse{LogicalId: req.Msg.GetLogicalId(), Field: req.Msg.GetField(), Generation: generation, Grants: make([]*grantv1.CredentialGrant, 0, len(grants))}
	for _, grant := range grants {
		if grant.LogicalID != response.LogicalId || grant.Field != response.Field {
			continue
		}
		response.Grants = append(response.Grants, toProto(grant))
		if err := h.deliverGrant(ctx, grant); err != nil {
			h.logger.Printf("credential rotation %q delivery deferred: %v", grant.ID, err)
		}
	}
	return connect.NewResponse(response), nil
}

func (h *handler) SyncNodeGrants(ctx context.Context, req *connect.Request[grantv1.SyncNodeGrantsRequest]) (*connect.Response[grantv1.ListGrantsResponse], error) {
	nodeID := req.Msg.GetNodeId()
	if nodeID == "" || h.nodeVerifier == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("node authentication is not configured"))
	}
	proof, err := nodeauth.ParseHeaders(req.Header().Get(nodeauth.HeaderNode), req.Header().Get(nodeauth.HeaderTS), req.Header().Get(nodeauth.HeaderSig))
	if err != nil || proof.NodeID != nodeID {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("node authentication proof does not match request"))
	}
	if err := h.nodeVerifier.VerifyProof(ctx, proof); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	grants, err := h.service.List(ctx, nodeID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &grantv1.ListGrantsResponse{Grants: make([]*grantv1.CredentialGrant, 0, len(grants))}
	for _, grant := range grants {
		response.Grants = append(response.Grants, toProto(grant))
	}
	return connect.NewResponse(response), nil
}

// SyncNode sends all active grant metadata to a node after it reconnects. It
// is safe to retry because the node-side metadata store is keyed by address.
func (h *handler) SyncNode(ctx context.Context, nodeID string) error {
	if h.presence == nil || !h.presence.IsOnline(nodeID) {
		return nil
	}
	grants, err := h.service.List(ctx, nodeID)
	if err != nil {
		return err
	}
	for _, grant := range grants {
		if err := h.deliverGrant(ctx, grant); err != nil {
			return err
		}
	}
	return nil
}

// deliverGrant seals and pushes the node-bound credential value.
// DOC: scenarios/vrooli-bridge/docs/reference/credential-delivery.md
func (h *handler) deliverGrant(ctx context.Context, grant internalgrant.Grant) error {
	if h.presence == nil || h.signer == nil || h.presence.IsOnline(grant.NodeID) == false {
		return nil
	}
	payload, err := internalgrant.GrantFrame(h.signer, grant, grant.NodeID)
	if err != nil {
		return err
	}
	if h.presence.Push(grant.NodeID, payload) == 0 {
		return fmt.Errorf("node %q has no writable channel", grant.NodeID)
	}
	if h.resolveValue == nil || h.sealingPublicKey == nil {
		return nil
	}
	value, err := h.resolveValue(ctx, grant.LogicalID, grant.Field)
	if err != nil {
		// A grant may legitimately precede provisioning. The metadata consent
		// remains durable; the next rotation/reconnect retries the value push.
		return nil
	}
	public, err := h.sealingPublicKey(ctx, grant.NodeID)
	if err != nil {
		return err
	}
	sealed, err := internalgrant.SealPush(h.signer, grant, grant.NodeID, public, value)
	if err != nil {
		return err
	}
	if h.presence.Push(grant.NodeID, sealed) == 0 {
		return fmt.Errorf("node %q has no writable channel for credential push", grant.NodeID)
	}
	return nil
}

func (h *handler) deliverPurge(ctx context.Context, grant internalgrant.Grant) error {
	if h.presence == nil || h.signer == nil || !h.presence.IsOnline(grant.NodeID) {
		return nil
	}
	payload, err := internalgrant.PurgeFrame(h.signer, grant.NodeID, []string{grant.LogicalID + ":" + grant.Field})
	if err != nil {
		return err
	}
	if h.presence.Push(grant.NodeID, payload) == 0 {
		return fmt.Errorf("node %q has no writable channel", grant.NodeID)
	}
	return nil
}

func toProto(grant internalgrant.Grant) *grantv1.CredentialGrant {
	out := &grantv1.CredentialGrant{
		Id: grant.ID, NodeId: grant.NodeID, LogicalId: grant.LogicalID, Field: grant.Field,
		Class: string(grant.Class), Retention: string(grant.Retention), Generation: grant.Generation,
		AckedGeneration: grant.AckedGeneration,
	}
	if !grant.GrantedAt.IsZero() {
		out.GrantedAt = timestamppb.New(grant.GrantedAt)
	}
	if !grant.RevokedAt.IsZero() {
		out.RevokedAt = timestamppb.New(grant.RevokedAt)
	}
	if !grant.ReceiptAt.IsZero() {
		out.ReceiptAt = timestamppb.New(grant.ReceiptAt)
	}
	out.ReceiptAccepted = grant.ReceiptAccepted
	out.ReceiptReason = grant.ReceiptReason
	return out
}

var _ grantconnect.CredentialGrantServiceHandler = (*handler)(nil)
