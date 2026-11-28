package credentialgrant

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

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
	deliveryMu       sync.Mutex
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
	grant, found, err := h.service.Active(ctx, req.Msg.GetNodeId(), req.Msg.GetLogicalId(), req.Msg.GetField())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !found {
		grant, found, err = h.service.ReactivateEphemeral(ctx, req.Msg.GetNodeId(), req.Msg.GetLogicalId(), req.Msg.GetField(), internalgrant.Class(req.Msg.GetClass()))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if !found {
		grant, err = h.service.Create(ctx, internalgrant.CreateInput{NodeID: req.Msg.GetNodeId(), LogicalID: req.Msg.GetLogicalId(), Field: req.Msg.GetField(), Class: internalgrant.Class(req.Msg.GetClass()), Retention: internalgrant.Retention(req.Msg.GetRetention()), Generation: 1})
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else if grant.Retention == internalgrant.RetentionEphemeral {
		// Ephemeral grants are one-shot. Re-answering the same address advances
		// its generation in place; the address is unique and cannot have two
		// active rows.
		grant, err = h.service.RefreshEphemeral(ctx, grant)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("refresh ephemeral credential grant: %w", err))
		}
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

// ListGrantMetadata is the read-only owner projection used by permission
// inventory. It never calls deliverGrant, reads credential values, or touches
// the presence hub.
func (h *handler) ListGrantMetadata(ctx context.Context, req *connect.Request[grantv1.ListGrantMetadataRequest]) (*connect.Response[grantv1.ListGrantMetadataResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	if owner.OwnerID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authenticated owner identity missing"))
	}
	nodeID, generation := strings.TrimSpace(req.Msg.GetNodeId()), req.Msg.GetEnrollmentGeneration()
	if nodeID == "" || generation <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("node_id and positive enrollment_generation are required"))
	}
	grants, err := h.service.ListMetadata(ctx, nodeID, generation)
	if err != nil {
		var stale internalgrant.ErrStaleNodeGeneration
		if errors.As(err, &stale) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("target enrollment generation changed"))
		}
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("credential grant metadata is unavailable"))
	}
	response := &grantv1.ListGrantMetadataResponse{NodeId: nodeID, EnrollmentGeneration: generation, Grants: make([]*grantv1.CredentialGrantMetadata, 0, len(grants))}
	for _, grant := range grants {
		response.Grants = append(response.Grants, toMetadataProto(grant))
	}
	return connect.NewResponse(response), nil
}

func (h *handler) GetGrantMetadata(ctx context.Context, req *connect.Request[grantv1.GetGrantMetadataRequest]) (*connect.Response[grantv1.CredentialGrantMetadata], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	if owner.OwnerID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authenticated owner identity missing"))
	}
	id, nodeID, generation := strings.TrimSpace(req.Msg.GetId()), strings.TrimSpace(req.Msg.GetNodeId()), req.Msg.GetEnrollmentGeneration()
	if id == "" || nodeID == "" || generation <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id, node_id, and positive enrollment_generation are required"))
	}
	grants, err := h.service.ListMetadata(ctx, nodeID, generation)
	if err != nil {
		var stale internalgrant.ErrStaleNodeGeneration
		if errors.As(err, &stale) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("target enrollment generation changed"))
		}
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("credential grant metadata is unavailable"))
	}
	for _, grant := range grants {
		if grant.ID == id {
			return connect.NewResponse(toMetadataProto(grant)), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, errors.New("credential grant metadata not found"))
}

func (h *handler) CreateGrantWithReceipt(ctx context.Context, req *connect.Request[grantv1.CreateGrantWithReceiptRequest]) (*connect.Response[grantv1.CredentialGrantMutationReceipt], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	if owner.OwnerID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authenticated owner identity missing"))
	}
	m := req.Msg
	if strings.TrimSpace(m.GetNodeId()) == "" || m.GetEnrollmentGeneration() <= 0 || strings.TrimSpace(m.GetRequestId()) == "" || m.GetExpectedRevision() != 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("node, enrollment generation, request ID, and expected revision 0 are required"))
	}
	digest := grantMutationDigest(struct {
		Operation, NodeID, LogicalID, Field, Class, Retention string
		EnrollmentGeneration, Generation, ExpectedRevision    int64
	}{"create", m.GetNodeId(), m.GetLogicalId(), m.GetField(), m.GetClass(), m.GetRetention(), m.GetEnrollmentGeneration(), m.GetGeneration(), m.GetExpectedRevision()})
	grant, receipt, err := h.service.CreateWithReceipt(ctx, internalgrant.CreateInput{NodeID: m.GetNodeId(), LogicalID: m.GetLogicalId(), Field: m.GetField(), Class: internalgrant.Class(m.GetClass()), Retention: internalgrant.Retention(m.GetRetention()), Generation: m.GetGeneration()}, internalgrant.MutationBinding{RequestID: m.GetRequestId(), ActorID: owner.OwnerID, NodeID: m.GetNodeId(), EnrollmentGeneration: m.GetEnrollmentGeneration(), Subject: internalgrant.CreateGrantSubject, ExpectedRevision: m.GetExpectedRevision(), PayloadDigest: digest})
	if err != nil {
		return nil, credentialMutationError(err)
	}
	// A durable mutation receipt proves that Bridge accepted the request, not
	// that the target applied it. Replays therefore resume an unconfirmed
	// delivery. The target deduplicates grant generations, and once Bridge has
	// persisted the target receipt we avoid sending the same effect again.
	if receipt.DeliveryState != "confirmed" && receipt.DeliveryState != "refused" {
		receipt.DeliveryState = h.dispatchGrantMutation(ctx, receipt.RequestID, grant)
	}
	return connect.NewResponse(toMutationReceiptProto(receipt, grant)), nil
}

func (h *handler) RevokeGrantWithReceipt(ctx context.Context, req *connect.Request[grantv1.RevokeGrantWithReceiptRequest]) (*connect.Response[grantv1.CredentialGrantMutationReceipt], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	if owner.OwnerID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authenticated owner identity missing"))
	}
	m := req.Msg
	id, nodeID := strings.TrimSpace(m.GetId()), strings.TrimSpace(m.GetNodeId())
	if id == "" || nodeID == "" || m.GetEnrollmentGeneration() <= 0 || m.GetExpectedRevision() <= 0 || strings.TrimSpace(m.GetRequestId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("grant, node, enrollment generation, expected revision, and request ID are required"))
	}
	digest := grantMutationDigest(struct {
		Operation, ID, NodeID                  string
		EnrollmentGeneration, ExpectedRevision int64
	}{"revoke", id, nodeID, m.GetEnrollmentGeneration(), m.GetExpectedRevision()})
	grant, receipt, err := h.service.RevokeWithReceipt(ctx, id, internalgrant.MutationBinding{RequestID: m.GetRequestId(), ActorID: owner.OwnerID, NodeID: nodeID, EnrollmentGeneration: m.GetEnrollmentGeneration(), Subject: internalgrant.RevokeGrantSubject + ":" + id, ExpectedRevision: m.GetExpectedRevision(), PayloadDigest: digest})
	if err != nil {
		return nil, credentialMutationError(err)
	}
	// Replayed revocations must retry a missing purge acknowledgement. Agent
	// tombstones make repeated purges idempotent; a durable Bridge purge receipt
	// is the point after which the effect is no longer dispatched.
	if receipt.DeliveryState != "confirmed" && receipt.DeliveryState != "refused" {
		receipt.DeliveryState = h.dispatchPurgeMutation(ctx, receipt.RequestID, grant)
	}
	return connect.NewResponse(toMutationReceiptProto(receipt, grant)), nil
}

func (h *handler) dispatchGrantMutation(ctx context.Context, requestID string, grant internalgrant.Grant) string {
	if h.presence == nil || !h.presence.IsOnline(grant.NodeID) {
		if err := h.service.SetMutationDeliveryState(ctx, requestID, "retryable"); err != nil {
			h.logger.Printf("credential grant %q retry state deferred: %v", grant.ID, err)
		}
		return "retryable"
	}
	if err := h.service.SetMutationDeliveryState(ctx, requestID, "dispatched"); err != nil {
		h.logger.Printf("credential grant %q dispatch state unavailable: %v", grant.ID, err)
		return "retryable"
	}
	if err := h.deliverGrant(ctx, grant); err != nil {
		if stateErr := h.service.SetMutationDeliveryState(ctx, requestID, "retryable"); stateErr != nil {
			h.logger.Printf("credential grant %q retry state deferred: %v", grant.ID, stateErr)
		}
		h.logger.Printf("credential grant %q delivery deferred: %v", grant.ID, err)
		return "retryable"
	}
	return "dispatched"
}

func (h *handler) dispatchPurgeMutation(ctx context.Context, requestID string, grant internalgrant.Grant) string {
	if h.presence == nil || !h.presence.IsOnline(grant.NodeID) {
		if err := h.service.SetMutationDeliveryState(ctx, requestID, "retryable"); err != nil {
			h.logger.Printf("credential grant %q purge retry state deferred: %v", grant.ID, err)
		}
		return "retryable"
	}
	if err := h.service.SetMutationDeliveryState(ctx, requestID, "dispatched"); err != nil {
		h.logger.Printf("credential grant %q purge dispatch state unavailable: %v", grant.ID, err)
		return "retryable"
	}
	if err := h.deliverPurge(ctx, grant); err != nil {
		if stateErr := h.service.SetMutationDeliveryState(ctx, requestID, "retryable"); stateErr != nil {
			h.logger.Printf("credential grant %q purge retry state deferred: %v", grant.ID, stateErr)
		}
		h.logger.Printf("credential grant %q purge delivery deferred: %v", grant.ID, err)
		return "retryable"
	}
	return "dispatched"
}

func credentialMutationError(err error) error {
	var stale internalgrant.ErrStaleNodeGeneration
	switch {
	case errors.Is(err, internalgrant.ErrMutationConflict):
		return connect.NewError(connect.CodeAlreadyExists, errors.New("credential grant request ID is bound to another mutation"))
	case errors.Is(err, internalgrant.ErrGrantRevisionConflict):
		return connect.NewError(connect.CodeAborted, errors.New("credential grant revision changed"))
	case errors.As(err, &stale):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("target enrollment generation changed"))
	case errors.Is(err, sql.ErrNoRows):
		return connect.NewError(connect.CodeNotFound, errors.New("credential grant not found for target"))
	default:
		return connect.NewError(connect.CodeInvalidArgument, errors.New("credential grant mutation refused"))
	}
}

func grantMutationDigest(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
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
	grant.RevokedAt = time.Now().UTC()
	grant.PurgeState = "pending"
	return connect.NewResponse(toProto(grant)), nil
}

func (h *handler) RotateAddress(ctx context.Context, req *connect.Request[grantv1.RotateAddressRequest]) (*connect.Response[grantv1.RotationResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	generation, grants, err := h.service.Rotate(ctx, req.Msg.GetLogicalId(), req.Msg.GetField())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	response := &grantv1.RotationResponse{LogicalId: req.Msg.GetLogicalId(), Field: req.Msg.GetField(), Generation: generation, Grants: make([]*grantv1.CredentialGrant, 0, len(grants)), RotationScope: "local_generation_only"}
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
	revoked, err := h.service.Revoked(ctx, nodeID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &grantv1.ListGrantsResponse{Grants: make([]*grantv1.CredentialGrant, 0, len(grants)+len(revoked))}
	for _, grant := range grants {
		response.Grants = append(response.Grants, toProto(grant))
	}
	for _, grant := range revoked {
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
	// The node's own store passphrase goes first: every durable value after it
	// is written into that store, which stays locked until the passphrase
	// arrives (2026-09-15: minimouse refused vrooli/openrouter on every
	// reconnect because its older grant was delivered first).
	for _, grant := range deliveryOrder(grants) {
		if err := h.deliverGrant(ctx, grant); err != nil {
			return err
		}
	}
	return nil
}

// nodeStorePassphrasePrefix names the grants that carry a node's own store
// passphrase (see internal/onboard.CredentialStoreEscrowNamespace).
const nodeStorePassphrasePrefix = "vrooli-bridge/node-credential-store/"

func opensNodeStore(grant internalgrant.Grant) bool {
	return strings.HasPrefix(grant.LogicalID, nodeStorePassphrasePrefix)
}

// deliveryOrder puts the grants that open the node's store first and keeps
// every other grant in its original order.
func deliveryOrder(grants []internalgrant.Grant) []internalgrant.Grant {
	ordered := append([]internalgrant.Grant(nil), grants...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return opensNodeStore(ordered[i]) && !opensNodeStore(ordered[j])
	})
	return ordered
}

// deliverGrant seals and pushes the node-bound credential value.
// DOC: docs/reference/credential-delivery.md
func (h *handler) deliverGrant(ctx context.Context, grant internalgrant.Grant) error {
	// A grant metadata frame must reach the node before its sealed push. Online
	// reconciliation, AnswerSecret, rotation, and reconnect hooks can all call
	// this method concurrently; serialize the complete metadata+value sequence
	// so separate calls cannot interleave those two frames on the same channel.
	h.deliveryMu.Lock()
	defer h.deliveryMu.Unlock()

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
	// Keep revocation frames ordered with grant delivery. A purge racing a
	// re-answer must not leave the node with an ambiguous local consent state.
	h.deliveryMu.Lock()
	defer h.deliveryMu.Unlock()

	if h.presence == nil || h.signer == nil || !h.presence.IsOnline(grant.NodeID) {
		return nil
	}
	payload, err := internalgrant.PurgeFrame(h.signer, grant.NodeID, grant.ID, grant.Generation, []string{grant.LogicalID + ":" + grant.Field})
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
	out.PurgeState = grant.PurgeState
	out.PurgeReceiptAccepted = grant.PurgeAccepted
	out.PurgeReceiptReason = grant.PurgeReason
	if !grant.PurgeReceiptAt.IsZero() {
		out.PurgeReceiptAt = timestamppb.New(grant.PurgeReceiptAt)
	}
	return out
}

func toMetadataProto(grant internalgrant.Grant) *grantv1.CredentialGrantMetadata {
	out := &grantv1.CredentialGrantMetadata{Id: grant.ID, NodeId: grant.NodeID, LogicalId: grant.LogicalID, Field: grant.Field, Class: string(grant.Class), Retention: string(grant.Retention), Generation: grant.Generation, AckedGeneration: grant.AckedGeneration, ReceiptAccepted: grant.ReceiptAccepted, PurgeState: grant.PurgeState, PurgeReceiptAccepted: grant.PurgeAccepted, Version: internalgrant.MetadataVersion(grant)}
	if !grant.GrantedAt.IsZero() {
		out.GrantedAt = timestamppb.New(grant.GrantedAt)
	}
	if !grant.RevokedAt.IsZero() {
		out.RevokedAt = timestamppb.New(grant.RevokedAt)
	}
	if !grant.ReceiptAt.IsZero() {
		out.ReceiptAt = timestamppb.New(grant.ReceiptAt)
	}
	if !grant.PurgeReceiptAt.IsZero() {
		out.PurgeReceiptAt = timestamppb.New(grant.PurgeReceiptAt)
	}
	return out
}

func toMutationReceiptProto(receipt internalgrant.MutationReceipt, grant internalgrant.Grant) *grantv1.CredentialGrantMutationReceipt {
	return &grantv1.CredentialGrantMutationReceipt{Id: "credential-grant-mutation:" + receipt.RequestID, Operation: strings.TrimSuffix(receipt.Subject, ":"+grant.ID), RequestId: receipt.RequestID, ActorId: receipt.ActorID, TargetNodeId: receipt.NodeID, EnrollmentGeneration: receipt.EnrollmentGeneration, Subject: receipt.Subject, ExpectedRevision: receipt.ExpectedRevision, ResultRevision: receipt.ResultRevision, PayloadDigest: receipt.PayloadDigest, Outcome: receipt.Outcome, CreatedAt: timestamppb.New(receipt.CreatedAt), Grant: toMetadataProto(grant), DeliveryState: receipt.DeliveryState}
}

var _ grantconnect.CredentialGrantServiceHandler = (*handler)(nil)
