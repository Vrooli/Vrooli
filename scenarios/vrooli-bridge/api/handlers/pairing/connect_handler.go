package pairing

import (
	"context"
	"errors"
	"log"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/pairing"
	"vrooli-bridge/pairingwords"

	"connectrpc.com/connect"

	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
)

// Deps wires the seams the PairingService handler needs.
type Deps struct {
	Service *pairing.Service
	// ControlPlanePublicKey is the standard-base64 CP Ed25519 public key handed
	// to nodes to pin (from internal/cpkeys).
	ControlPlanePublicKey string
	// DefaultScopes are the posture-selected scopes applied when the owner does
	// not provide a narrower grant at enrollment time.
	DefaultScopes     []string
	PermissionPresets []pairing.PermissionPreset
	Logger            *log.Logger
	NodeVerifier      *nodeauth.Verifier
}

func (h *connectHandler) RegisterEncryptionKey(ctx context.Context, req *connect.Request[pairingv1.RegisterEncryptionKeyRequest]) (*connect.Response[pairingv1.RegisterEncryptionKeyResponse], error) {
	if h.deps.NodeVerifier == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("node authentication is not configured"))
	}
	proof, err := nodeauth.ParseHeaders(req.Header().Get(nodeauth.HeaderNode), req.Header().Get(nodeauth.HeaderTS), req.Header().Get(nodeauth.HeaderSig))
	if err != nil || proof.NodeID != req.Msg.GetNodeId() {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("node authentication proof does not match request"))
	}
	if err := h.deps.NodeVerifier.VerifyProof(ctx, proof); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if err := h.deps.Service.RegisterEncryptionKeyAuthenticated(ctx, proof.NodeID, req.Msg.GetEncryptionPublicKey()); err != nil {
		return nil, h.toConnectError("RegisterEncryptionKey", err)
	}
	return connect.NewResponse(&pairingv1.RegisterEncryptionKeyResponse{NodeId: proof.NodeID, Algorithm: "x25519", Registered: true}), nil
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the PairingService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// IssuePairingCode is owner-gated: only the owner mints bootstrap codes.
func (h *connectHandler) IssuePairingCode(ctx context.Context, req *connect.Request[pairingv1.IssuePairingCodeRequest]) (*connect.Response[pairingv1.IssuePairingCodeResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	ttl := time.Duration(req.Msg.GetTtlSeconds()) * time.Second
	scopes := req.Msg.GetScopes()
	if len(scopes) == 0 {
		scopes = append([]string(nil), h.deps.DefaultScopes...)
	}
	issued, err := h.deps.Service.IssueCode(ctx, req.Msg.GetName(), scopes, ttl)
	if err != nil {
		return nil, h.toConnectError("IssuePairingCode", err)
	}
	return connect.NewResponse(&pairingv1.IssuePairingCodeResponse{
		Code:                  issued.Code,
		ControlPlanePublicKey: h.deps.ControlPlanePublicKey,
		ExpiresAt:             timeToProto(issued.ExpiresAt),
	}), nil
}

// RedeemPairingCode is the open, node-facing bootstrap call (authed by
// possession of the still-valid code, not an owner token).
func (h *connectHandler) RedeemPairingCode(ctx context.Context, req *connect.Request[pairingv1.RedeemPairingCodeRequest]) (*connect.Response[pairingv1.RedeemPairingCodeResponse], error) {
	nodeID, err := h.deps.Service.Redeem(ctx, req.Msg.GetCode(), req.Msg.GetNodePublicKey(), pairing.NodeFacts{
		Name:         req.Msg.GetName(),
		OS:           req.Msg.GetOs(),
		Arch:         req.Msg.GetArch(),
		MachineArch:  req.Msg.GetMachineArch(),
		BinaryArch:   req.Msg.GetBinaryArch(),
		Endpoint:     req.Msg.GetEndpoint(),
		Capabilities: req.Msg.GetCapabilities(),
	})
	if err != nil {
		return nil, h.toConnectError("RedeemPairingCode", err)
	}
	return connect.NewResponse(&pairingv1.RedeemPairingCodeResponse{
		NodeId:                nodeID,
		ControlPlanePublicKey: h.deps.ControlPlanePublicKey,
	}), nil
}

// RequestPairing is the open, no-code fallback enrollment ask.
func (h *connectHandler) RequestPairing(ctx context.Context, req *connect.Request[pairingv1.RequestPairingRequest]) (*connect.Response[pairingv1.RequestPairingResponse], error) {
	pr, err := h.deps.Service.RequestPairing(ctx, req.Msg.GetNodePublicKey(), pairing.NodeFacts{
		Name:         req.Msg.GetName(),
		OS:           req.Msg.GetOs(),
		Arch:         req.Msg.GetArch(),
		MachineArch:  req.Msg.GetMachineArch(),
		BinaryArch:   req.Msg.GetBinaryArch(),
		Endpoint:     req.Msg.GetEndpoint(),
		Capabilities: req.Msg.GetCapabilities(),
	})
	if err != nil {
		return nil, h.toConnectError("RequestPairing", err)
	}
	return connect.NewResponse(&pairingv1.RequestPairingResponse{
		RequestId:         pr.ID,
		Status:            statusToProto(pr.Status),
		ConfirmationWords: h.confirmationWords(pr),
		KeyFingerprint:    pairingwords.Fingerprint(pr.PublicKey),
	}), nil
}

// GetPairingRequest is the open polling leg of request/approve enrollment. It
// deliberately returns only the request's public enrollment state; an owner
// credential is not needed because possession of the request id is the
// bootstrap handle and no secret is exposed before approval.
func (h *connectHandler) GetPairingRequest(ctx context.Context, req *connect.Request[pairingv1.GetPairingRequestRequest]) (*connect.Response[pairingv1.GetPairingRequestResponse], error) {
	request, err := h.deps.Service.GetRequest(ctx, req.Msg.GetRequestId())
	if err != nil {
		return nil, h.toConnectError("GetPairingRequest", err)
	}
	return connect.NewResponse(&pairingv1.GetPairingRequestResponse{
		Request:               requestToProto(request, h.confirmationWords(request)),
		ControlPlanePublicKey: h.deps.ControlPlanePublicKey,
	}), nil
}

// ApprovePairing is owner-gated: only the owner decides pending requests.
func (h *connectHandler) ApprovePairing(ctx context.Context, req *connect.Request[pairingv1.ApprovePairingRequest]) (*connect.Response[pairingv1.ApprovePairingResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	var status pairing.RequestStatus
	var nodeID string
	var err error
	if req.Msg.GetApprove() {
		status, nodeID, err = h.deps.Service.ApproveWithConfirmation(ctx, req.Msg.GetRequestId(), true, req.Msg.GetScopes(), req.Msg.GetConfirmationWords())
	} else {
		status, nodeID, err = h.deps.Service.Approve(ctx, req.Msg.GetRequestId(), false, req.Msg.GetScopes())
	}
	if err != nil {
		return nil, h.toConnectError("ApprovePairing", err)
	}
	return connect.NewResponse(&pairingv1.ApprovePairingResponse{
		NodeId: nodeID,
		Status: statusToProto(status),
	}), nil
}

// ListPairingRequests is owner-gated.
func (h *connectHandler) ListPairingRequests(ctx context.Context, req *connect.Request[pairingv1.ListPairingRequestsRequest]) (*connect.Response[pairingv1.ListPairingRequestsResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	reqs, err := h.deps.Service.ListRequests(ctx, req.Msg.GetIncludeDecided())
	if err != nil {
		return nil, h.toConnectError("ListPairingRequests", err)
	}
	out := &pairingv1.ListPairingRequestsResponse{
		Requests: make([]*pairingv1.PairingRequest, 0, len(reqs)),
		Presets:  make([]*pairingv1.PermissionPreset, 0, len(h.deps.PermissionPresets)),
	}
	for _, r := range reqs {
		out.Requests = append(out.Requests, requestToProto(r, h.confirmationWords(r)))
	}
	for _, preset := range h.deps.PermissionPresets {
		out.Presets = append(out.Presets, &pairingv1.PermissionPreset{
			Name:        string(preset.Name),
			Description: preset.Description,
			Scopes:      append([]string(nil), preset.Scopes...),
			Withholds:   append([]string(nil), preset.Withholds...),
		})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) confirmationWords(r pairing.PairingRequest) []string {
	words, err := pairingwords.Derive(h.deps.ControlPlanePublicKey, r.PublicKey)
	if err != nil {
		// Test seams and degraded installations can omit a valid CP identity. Do
		// not make listing a request leak or crash in that state; production
		// approval still fails closed because its validator is configured below.
		return nil
	}
	return words
}

// toConnectError maps the pairing domain's typed sentinels to Connect codes. A
// bad/expired/used code is Unauthenticated (a node failing to authenticate),
// validation is InvalidArgument, missing request is NotFound, and an unexpected
// error is logged + Internal.
func (h *connectHandler) toConnectError(op string, err error) error {
	switch {
	case errors.Is(err, pairing.ErrCodeNotFound), errors.Is(err, pairing.ErrCodeExpired), errors.Is(err, pairing.ErrCodeUsed):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, pairing.ErrRequestNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, pairing.ErrRequestDecided):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	var invalid pairing.ErrInvalid
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	h.deps.Logger.Printf("pairing.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}
