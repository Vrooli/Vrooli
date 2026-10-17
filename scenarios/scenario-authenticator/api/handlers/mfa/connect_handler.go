// Package mfa implements the authenticator-owned MFAService transport.
package mfa

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	mfav1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/mfa"

	"scenario-authenticator/internal/accounts"
	auditlog "scenario-authenticator/internal/audit"
	intmfa "scenario-authenticator/internal/mfa"
)

type Deps struct {
	Accounts *accounts.Service
	Store    *intmfa.Store
	Audit    auditlog.Logger
	Logger   *log.Logger
}

type connectHandler struct{ deps Deps }

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) BeginEnrollment(ctx context.Context, req *connect.Request[mfav1.BeginEnrollmentRequest]) (*connect.Response[mfav1.BeginEnrollmentResponse], error) {
	identity, err := h.identity(ctx, req.Msg.GetAccessToken())
	if err != nil {
		return nil, err
	}
	enrollment, err := h.deps.Store.BeginEnrollment(ctx, identity.UserID, identity.Realm, identity.Email)
	if err != nil {
		return nil, h.internal("BeginEnrollment", err)
	}
	h.log(ctx, identity.UserID, identity.Realm, "mfa.enrollment.started", true)
	return connect.NewResponse(&mfav1.BeginEnrollmentResponse{
		EnrollmentId: enrollment.ID, ProvisioningUri: enrollment.ProvisioningURI,
		ExpiresAt: timestamppb.New(enrollment.ExpiresAt),
	}), nil
}

func (h *connectHandler) ConfirmEnrollment(ctx context.Context, req *connect.Request[mfav1.ConfirmEnrollmentRequest]) (*connect.Response[mfav1.ConfirmEnrollmentResponse], error) {
	identity, err := h.identity(ctx, req.Msg.GetAccessToken())
	if err != nil {
		return nil, err
	}
	codes, err := h.deps.Store.ConfirmEnrollment(ctx, identity.UserID, req.Msg.GetEnrollmentId(), req.Msg.GetTotpCode())
	if err != nil {
		success := false
		h.log(ctx, identity.UserID, identity.Realm, "mfa.enrollment.confirmed", success)
		switch {
		case errors.Is(err, intmfa.ErrEnrollmentExpired):
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("MFA enrollment expired"))
		case errors.Is(err, intmfa.ErrCodeInvalid):
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid MFA code"))
		default:
			return nil, h.internal("ConfirmEnrollment", err)
		}
	}
	h.log(ctx, identity.UserID, identity.Realm, "mfa.enrollment.confirmed", true)
	return connect.NewResponse(&mfav1.ConfirmEnrollmentResponse{RecoveryCodes: codes, EnrolledAt: timestamppb.New(timeNow())}), nil
}

func (h *connectHandler) RemoveEnrollment(ctx context.Context, req *connect.Request[mfav1.RemoveEnrollmentRequest]) (*connect.Response[mfav1.RemoveEnrollmentResponse], error) {
	identity, err := h.identity(ctx, req.Msg.GetAccessToken())
	if err != nil {
		return nil, err
	}
	if err := h.deps.Store.RemoveEnrollment(ctx, identity.UserID); err != nil {
		return nil, h.internal("RemoveEnrollment", err)
	}
	h.log(ctx, identity.UserID, identity.Realm, "mfa.enrollment.removed", true)
	return connect.NewResponse(&mfav1.RemoveEnrollmentResponse{}), nil
}

type identity struct {
	UserID string
	Email  string
	Realm  string
}

func (h *connectHandler) identity(ctx context.Context, token string) (identity, error) {
	validated, ok, err := h.deps.Accounts.Validate(ctx, token)
	if err != nil {
		return identity{}, h.internal("Validate", err)
	}
	if !ok {
		return identity{}, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired token"))
	}
	return identity{UserID: validated.UserID, Email: validated.Email, Realm: validated.Realm}, nil
}

func (h *connectHandler) internal(op string, err error) error {
	h.deps.Logger.Printf("mfa.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

func (h *connectHandler) log(ctx context.Context, userID, realmID, action string, success bool) {
	if h.deps.Audit != nil {
		_ = h.deps.Audit.Log(ctx, auditlog.Event{UserID: userID, RealmID: realmID, Action: action, Success: success})
	}
}

// timeNow is a narrow seam for the wire timestamp; enrollment persistence owns
// the authoritative clock and this timestamp is informational.
var timeNow = func() time.Time { return time.Now().UTC() }
