package adminpasskeys

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/go-webauthn/webauthn/webauthn"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	internal "landing-page-business-suite-api/internal/passkeys"
)

type Dependencies struct {
	Service        *internal.Service
	Challenges     internal.ChallengeWriter
	Credentials    *internal.AdminCredentials
	AdminEmail     func(*http.Request) (string, bool)
	PublicOrigin   string
	MFARequired    func() bool
	SecurityEvents interface {
		Record(context.Context, string, string, string, string, string, map[string]any) error
	}
	Notifier interface {
		Notify(context.Context, string, string, string) error
	}
}
type Handler struct{ deps Dependencies }

func New(deps Dependencies) *Handler { return &Handler{deps: deps} }
func (h *Handler) user(ctx context.Context, email string) (internal.AdminUser, error) {
	if h.deps.Credentials == nil || strings.TrimSpace(email) == "" {
		return internal.AdminUser{}, errors.New("admin session is not authenticated")
	}
	return h.deps.Credentials.User(ctx, email)
}
func webUser(u internal.AdminUser) internal.User {
	return internal.User{ID: fmt.Sprint(u.ID), Handle: u.Handle, Email: u.Email, Credentials: u.Credentials}
}
func (h *Handler) BeginRegistration(ctx context.Context, req *connect.Request[lpbsv1.BeginAdminPasskeyRegistrationRequest]) (*connect.Response[lpbsv1.BeginAdminPasskeyRegistrationResponse], error) {
	email := req.Header().Get("X-Lpbs-Admin-Email")
	u, err := h.user(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	opts, session, err := h.deps.Service.BeginRegistration(webUser(u))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to begin passkey registration"))
	}
	data, err := internal.EncodeSessionData(session)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to save passkey ceremony"))
	}
	id := internal.NewChallengeID()
	if err := h.deps.Challenges.Create(ctx, internal.Challenge{ID: id, Value: session.Challenge, Purpose: "admin_register", Subject: email, BindingHash: internal.Hash(internal.NormalizeBinding(req.Header().Get("X-Lpbs-Browser-Binding"))), SessionData: data}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to save passkey ceremony"))
	}
	return connect.NewResponse(&lpbsv1.BeginAdminPasskeyRegistrationResponse{OptionsJson: string(opts), CeremonyId: id}), nil
}
func (h *Handler) FinishRegistration(ctx context.Context, req *connect.Request[lpbsv1.FinishAdminPasskeyRegistrationRequest]) (*connect.Response[lpbsv1.FinishAdminPasskeyRegistrationResponse], error) {
	email := req.Header().Get("X-Lpbs-Admin-Email")
	u, err := h.user(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	c, err := h.deps.Challenges.Consume(ctx, req.Msg.GetCeremonyId(), "admin_register", internal.Hash(internal.NormalizeBinding(req.Header().Get("X-Lpbs-Browser-Binding"))))
	if err != nil || c.Subject != email {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("passkey ceremony is not valid"))
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(c.SessionData, &session); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("passkey ceremony is corrupt"))
	}
	credential, err := h.deps.Service.FinishRegistration(webUser(u), session, []byte(req.Msg.GetCredentialJson()), h.deps.PublicOrigin)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("passkey registration was rejected"))
	}
	id, err := h.deps.Credentials.Save(ctx, u.ID, h.deps.Service.RP.Config.RPID, req.Msg.GetNickname(), *credential)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to save passkey"))
	}
	if h.deps.SecurityEvents != nil {
		_ = h.deps.SecurityEvents.Record(ctx, "passkey_added", email, "", "", req.Header().Get("User-Agent"), map[string]any{"credential_id": id})
	}
	if h.deps.Notifier != nil {
		_ = h.deps.Notifier.Notify(ctx, email, "passkey_added", "An administrator passkey was added to your account.")
	}
	return connect.NewResponse(&lpbsv1.FinishAdminPasskeyRegistrationResponse{Id: id, Nickname: req.Msg.GetNickname()}), nil
}
func (h *Handler) ListPasskeys(ctx context.Context, req *connect.Request[lpbsv1.ListAdminPasskeysRequest]) (*connect.Response[lpbsv1.ListAdminPasskeysResponse], error) {
	u, err := h.user(ctx, req.Header().Get("X-Lpbs-Admin-Email"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("admin session is not authenticated"))
	}
	records, err := h.deps.Credentials.List(ctx, u.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*lpbsv1.AdminPasskey, 0, len(records))
	for _, v := range records {
		p := &lpbsv1.AdminPasskey{Id: v.ID, Nickname: v.Nickname, CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339)}
		if v.LastUsedAt.Valid {
			p.LastUsedAt = v.LastUsedAt.Time.UTC().Format(time.RFC3339)
		}
		out = append(out, p)
	}
	return connect.NewResponse(&lpbsv1.ListAdminPasskeysResponse{Passkeys: out}), nil
}
func (h *Handler) RenamePasskey(ctx context.Context, req *connect.Request[lpbsv1.RenameAdminPasskeyRequest]) (*connect.Response[lpbsv1.RenameAdminPasskeyResponse], error) {
	u, err := h.user(ctx, req.Header().Get("X-Lpbs-Admin-Email"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if err = h.deps.Credentials.Rename(ctx, u.ID, req.Msg.GetId(), req.Msg.GetNickname()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("passkey could not be renamed"))
	}
	return connect.NewResponse(&lpbsv1.RenameAdminPasskeyResponse{Passkey: &lpbsv1.AdminPasskey{Id: req.Msg.GetId(), Nickname: req.Msg.GetNickname()}}), nil
}
func (h *Handler) RevokePasskey(ctx context.Context, req *connect.Request[lpbsv1.RevokeAdminPasskeyRequest]) (*connect.Response[lpbsv1.RevokeAdminPasskeyResponse], error) {
	u, err := h.user(ctx, req.Header().Get("X-Lpbs-Admin-Email"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if h.deps.MFARequired != nil && h.deps.MFARequired() {
		active, countErr := h.deps.Credentials.List(ctx, u.ID)
		totp, totpErr := h.deps.Credentials.HasTOTP(ctx, u.ID)
		if countErr != nil || totpErr != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("unable to verify administrator factors"))
		}
		if len(active) <= 1 && !totp {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("keep at least one administrator second factor enrolled"))
		}
	}
	if err = h.deps.Credentials.Revoke(ctx, u.ID, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("passkey not found"))
	}
	if h.deps.SecurityEvents != nil {
		_ = h.deps.SecurityEvents.Record(ctx, "passkey_removed", u.Email, "", "", req.Header().Get("User-Agent"), map[string]any{"credential_id": req.Msg.GetId()})
	}
	if h.deps.Notifier != nil {
		_ = h.deps.Notifier.Notify(ctx, u.Email, "passkey_removed", "An administrator passkey was removed from your account.")
	}
	return connect.NewResponse(&lpbsv1.RevokeAdminPasskeyResponse{Revoked: true}), nil
}
func (h *Handler) BeginSecondFactor(ctx context.Context, req *connect.Request[lpbsv1.BeginAdminSecondFactorRequest]) (*connect.Response[lpbsv1.BeginAdminSecondFactorResponse], error) {
	u, err := h.user(ctx, req.Msg.GetEmail())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("passkey authentication is not available"))
	}
	opts, session, err := h.deps.Service.BeginUserAuthentication(webUser(u))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("passkey authentication is not available"))
	}
	data, err := internal.EncodeSessionData(session)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to save passkey ceremony"))
	}
	id := internal.NewChallengeID()
	if err := h.deps.Challenges.Create(ctx, internal.Challenge{ID: id, Value: session.Challenge, Purpose: "admin_second_factor", Subject: u.Email, BindingHash: internal.Hash(internal.NormalizeBinding(req.Header().Get("X-Lpbs-Browser-Binding"))), SessionData: data}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to save passkey ceremony"))
	}
	return connect.NewResponse(&lpbsv1.BeginAdminSecondFactorResponse{OptionsJson: string(opts), CeremonyId: id}), nil
}

func (h *Handler) VerifyLogin(ctx context.Context, email, ceremony string, assertion []byte, browserBinding, _ string) error {
	u, err := h.user(ctx, email)
	if err != nil {
		return err
	}
	// Login's Connect request supplies the binding through the context-free
	// verifier seam; the ceremony itself is still one-use and email-scoped.
	c, err := h.deps.Challenges.Consume(ctx, ceremony, "admin_second_factor", internal.Hash(internal.NormalizeBinding(browserBinding)))
	if err != nil {
		return err
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(c.SessionData, &session); err != nil {
		return err
	}
	credential, err := h.deps.Service.FinishLogin(webUser(u), session, assertion, h.deps.PublicOrigin)
	if err != nil {
		return err
	}
	return h.deps.Credentials.RecordUse(ctx, *credential)
}

var _ lpbsconnect.AdminPasskeyServiceHandler = (*Handler)(nil)
