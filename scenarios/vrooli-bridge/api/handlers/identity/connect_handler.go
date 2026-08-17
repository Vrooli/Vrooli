// Package identity is the HTTP/Connect transport edge for the same-origin owner
// sign-in / registration facade. It is intentionally thin: decode the request,
// call internal/identity.Forwarder (which forwards to scenario-authenticator via
// api-core/discovery), and translate the result or typed error to Connect. No
// credential logic lives here or in the forwarder — only relay.
package identity

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	sharedsession "github.com/vrooli/api-core/operatorsession"
	"google.golang.org/protobuf/types/known/timestamppb"
	"vrooli-bridge/internal/auth"
	internalidentity "vrooli-bridge/internal/identity"
	internaloperatorsession "vrooli-bridge/internal/operatorsession"

	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity"
)

// Forwarder is the seam the handler depends on (internal/identity.Forwarder in
// production; a fake in tests).
type Forwarder interface {
	Login(ctx context.Context, c internalidentity.Credentials) (internalidentity.Owner, error)
	Register(ctx context.Context, r internalidentity.Registration) (internalidentity.Owner, error)
	Refresh(ctx context.Context, refreshToken string) (internalidentity.Owner, error)
}

// Deps wires the seams the Connect identity handler needs.
type Deps struct {
	Forwarder   Forwarder
	Enrollments internaloperatorsession.Store
	Logger      *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect handler for the identity service.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Login(ctx context.Context, req *connect.Request[identityv1.LoginRequest]) (*connect.Response[identityv1.LoginResponse], error) {
	owner, err := h.deps.Forwarder.Login(ctx, internalidentity.Credentials{
		Email:    req.Msg.GetEmail(),
		Password: req.Msg.GetPassword(),
	})
	if err != nil {
		// Invalid credentials is an expected, non-noteworthy outcome; only log
		// the unexpected (unavailable/internal) cases to avoid log spam from
		// failed sign-in attempts.
		if !errors.Is(err, internalidentity.ErrInvalidCredentials) {
			h.deps.Logger.Printf("identity.Login: %v", err)
		}
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&identityv1.LoginResponse{
		Token:        owner.Token,
		Email:        owner.Email,
		UserId:       owner.UserID,
		RefreshToken: owner.RefreshToken,
	}), nil
}

func (h *connectHandler) Register(ctx context.Context, req *connect.Request[identityv1.RegisterRequest]) (*connect.Response[identityv1.RegisterResponse], error) {
	owner, err := h.deps.Forwarder.Register(ctx, internalidentity.Registration{
		Email:    req.Msg.GetEmail(),
		Password: req.Msg.GetPassword(),
		Username: req.Msg.GetUsername(),
	})
	if err != nil {
		if !errors.Is(err, internalidentity.ErrEmailTaken) && !errors.Is(err, internalidentity.ErrInvalidInput) {
			h.deps.Logger.Printf("identity.Register: %v", err)
		}
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&identityv1.RegisterResponse{
		Token:        owner.Token,
		Email:        owner.Email,
		UserId:       owner.UserID,
		RefreshToken: owner.RefreshToken,
	}), nil
}

func (h *connectHandler) Refresh(ctx context.Context, req *connect.Request[identityv1.RefreshRequest]) (*connect.Response[identityv1.RefreshResponse], error) {
	owner, err := h.deps.Forwarder.Refresh(ctx, req.Msg.GetRefreshToken())
	if err != nil {
		if !errors.Is(err, internalidentity.ErrInvalidCredentials) && !errors.Is(err, internalidentity.ErrInvalidInput) {
			h.deps.Logger.Printf("identity.Refresh: %v", err)
		}
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&identityv1.RefreshResponse{
		Token: owner.Token, RefreshToken: owner.RefreshToken,
	}), nil
}

func (h *connectHandler) EnrollOperatorSession(ctx context.Context, req *connect.Request[identityv1.EnrollOperatorSessionRequest]) (*connect.Response[identityv1.EnrollOperatorSessionResponse], error) {
	if h.deps.Enrollments == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("operator session enrollment is unavailable"))
	}
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	mode := sharedsession.Mode(strings.TrimSpace(req.Msg.GetMode()))
	if mode == "" {
		mode = sharedsession.ModePersonal
	}
	if mode != sharedsession.ModePersonal && mode != sharedsession.ModeShared && mode != sharedsession.ModeHosted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported operator session mode"))
	}
	ceiling := append([]string(nil), owner.Scopes...)
	requested := req.Msg.GetRequestedScopes()
	if len(requested) > 0 {
		if !sharedsession.ContainsAll(ceiling, requested) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("requested operator session scope exceeds the owner ceiling"))
		}
		ceiling = append([]string(nil), requested...)
	}
	now := time.Now().UTC()
	record := internaloperatorsession.Record{Reference: uuid.NewString(), OperatorID: owner.OwnerID, Mode: mode, PublicKey: append([]byte(nil), req.Msg.GetPublicKey()...), Scopes: ceiling, EnrolledAt: now}
	if err := internaloperatorsession.Validate(record); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if _, err := h.deps.Enrollments.Enroll(ctx, record); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&identityv1.EnrollOperatorSessionResponse{EnrollmentReference: record.Reference, OperatorId: record.OperatorID, IdentityProvider: "scenario-authenticator", Mode: string(record.Mode), ScopeCeiling: record.Scopes, EnrolledAt: timestamppb.New(record.EnrolledAt), SessionTtlSeconds: int64(sharedsession.LocalSessionTTL / time.Second)}), nil
}

// toConnectError maps forwarder errors to Connect codes the UI can branch on.
// The message is preserved so an input-validation failure (weak password, bad
// email) surfaces verbatim.
func toConnectError(err error) error {
	switch {
	case errors.Is(err, internalidentity.ErrInvalidCredentials):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, internalidentity.ErrEmailTaken):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, internalidentity.ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, internalidentity.ErrAuthUnavailable):
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
