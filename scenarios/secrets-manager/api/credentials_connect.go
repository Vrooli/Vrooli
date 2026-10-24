package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	credentials "github.com/vrooli/vrooli/packages/proto/gen/go/secrets-manager/v1/credentials"
	credentialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/secrets-manager/v1/credentials/credentials_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// credentialBrokerConnectHandler adapts the typed runtime contract to the
// existing PasswordManager authority. It owns transport translation only;
// grant evaluation, session custody, origin pinning, and redaction remain in
// the password-manager domain.
type credentialBrokerConnectHandler struct {
	credentialsconnect.UnimplementedCredentialBrokerServiceHandler
	passwordManager *passwordManagerHandlers
}

func newCredentialBrokerConnectHandler(passwordManager *passwordManagerHandlers) *credentialBrokerConnectHandler {
	return &credentialBrokerConnectHandler{passwordManager: passwordManager}
}

func (h *credentialBrokerConnectHandler) authorize(ctx context.Context, headers http.Header) (context.Context, string, string, error) {
	request := (&http.Request{Header: headers}).WithContext(ctx)
	if err := h.passwordManager.authorize(request); err != nil {
		return nil, "", "", credentialBrokerConnectError(err)
	}
	workspace, actor := requestIdentity(request)
	return request.Context(), workspace, actor, nil
}

func (h *credentialBrokerConnectHandler) RequestAccess(ctx context.Context, req *connect.Request[credentials.RequestAccessRequest]) (*connect.Response[credentials.AccessRequest], error) {
	ctx, workspace, actor, err := h.authorize(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	request, err := h.passwordManager.manager.createAccessRequest(ctx, workspace, actor, createAccessRequestInput{
		GrantID: req.Msg.GetGrantId(), ItemID: req.Msg.GetItemId(), Operation: req.Msg.GetOperation(), DurationSec: int(req.Msg.GetDurationSeconds()),
	})
	if err != nil {
		return nil, credentialBrokerConnectError(err)
	}
	return connect.NewResponse(accessRequestMessage(request)), nil
}

func (h *credentialBrokerConnectHandler) GetAccessRequest(ctx context.Context, req *connect.Request[credentials.GetAccessRequestRequest]) (*connect.Response[credentials.AccessRequest], error) {
	ctx, workspace, _, err := h.authorize(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	request, err := h.passwordManager.manager.getAccessRequest(ctx, workspace, req.Msg.GetRequestId())
	if err != nil {
		return nil, credentialBrokerConnectError(err)
	}
	return connect.NewResponse(accessRequestMessage(request)), nil
}

func (h *credentialBrokerConnectHandler) WaitAccessRequest(ctx context.Context, req *connect.Request[credentials.WaitAccessRequestRequest]) (*connect.Response[credentials.WaitAccessRequestResponse], error) {
	ctx, workspace, _, err := h.authorize(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(req.Msg.GetTimeoutSeconds()) * time.Second
	request, timedOut, err := h.passwordManager.manager.waitForAccessRequest(ctx, workspace, req.Msg.GetRequestId(), timeout)
	if err != nil {
		return nil, credentialBrokerConnectError(err)
	}
	return connect.NewResponse(&credentials.WaitAccessRequestResponse{Request: accessRequestMessage(request), TimedOut: timedOut}), nil
}

func (h *credentialBrokerConnectHandler) CreateBrokerSession(ctx context.Context, req *connect.Request[credentials.CreateBrokerSessionRequest]) (*connect.Response[credentials.BrokerSession], error) {
	ctx, workspace, actor, err := h.authorize(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	input := brokerSessionInput{
		GrantID: req.Msg.GetGrantId(), ItemID: req.Msg.GetItemId(), TargetOrigin: req.Msg.GetTargetOrigin(), AllowInternal: req.Msg.GetAllowInternal(), DurationSec: int(req.Msg.GetDurationSeconds()), OneUse: req.Msg.GetOneUse(),
	}
	if auth, ok := ctx.Value(ownerAuthContextKey{}).(ownerAuthContext); ok && auth.Role == "agent" {
		input.RunID = auth.RunID
	}
	session, token, err := h.passwordManager.manager.createBrokerSession(ctx, workspace, actor, input)
	if err != nil {
		return nil, credentialBrokerConnectError(err)
	}
	return connect.NewResponse(&credentials.BrokerSession{
		SessionId: session.ID, SessionToken: token, ExpiresAt: timestamppb.New(session.Expires), TargetOrigin: session.Origin, SecretExposure: "brokered",
	}), nil
}

func (h *credentialBrokerConnectHandler) ExecuteBrokerOperation(ctx context.Context, req *connect.Request[credentials.ExecuteBrokerOperationRequest]) (*connect.Response[credentials.BrokerOperationResult], error) {
	ctx, workspace, actor, err := h.authorize(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	result, err := h.passwordManager.manager.brokerOperation(ctx, workspace, actor, req.Msg.GetSessionId(), req.Msg.GetSessionToken(), brokerOperationInput{
		Method: req.Msg.GetMethod(), Path: req.Msg.GetPath(), Headers: req.Msg.GetHeaders(), Body: req.Msg.GetBody(),
	})
	if err != nil {
		return nil, credentialBrokerConnectError(err)
	}
	return connect.NewResponse(&credentials.BrokerOperationResult{
		Status: int32(result.Status), Headers: result.Headers, Body: result.Body, BodyTruncated: result.BodyTruncated, Projection: result.Projection,
	}), nil
}

func (h *credentialBrokerConnectHandler) RevokeBrokerSession(ctx context.Context, req *connect.Request[credentials.RevokeBrokerSessionRequest]) (*connect.Response[credentials.RevokeBrokerSessionResponse], error) {
	ctx, workspace, actor, err := h.authorize(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	if err := h.passwordManager.manager.revokeBrokerSession(ctx, workspace, actor, req.Msg.GetSessionId()); err != nil {
		return nil, credentialBrokerConnectError(err)
	}
	return connect.NewResponse(&credentials.RevokeBrokerSessionResponse{SessionId: req.Msg.GetSessionId(), Status: "revoked"}), nil
}

func accessRequestMessage(request accessRequest) *credentials.AccessRequest {
	message := &credentials.AccessRequest{
		Id: request.ID, WorkspaceId: request.WorkspaceID, GrantId: request.GrantID, ItemId: request.ItemID,
		Operation: request.Operation, RequestDigest: request.RequestDigest, Status: request.Status, RequestedBy: request.RequestedBy,
		ExpiresAt: timestamppb.New(request.ExpiresAt), CreatedAt: timestamppb.New(request.CreatedAt),
	}
	return message
}

func credentialBrokerConnectError(err error) error {
	switch {
	case errors.Is(err, errAccessDenied):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Is(err, errAssuranceRequired), errors.Is(err, errVaultLocked):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, sql.ErrNoRows):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
}
