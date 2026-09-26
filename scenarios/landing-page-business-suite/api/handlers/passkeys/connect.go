package passkeys

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/go-webauthn/webauthn/webauthn"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	authadmin "landing-page-business-suite-api/internal/administration"
	internal "landing-page-business-suite-api/internal/passkeys"
)

type Throttle interface {
	Allow(context.Context, string, authadmin.ThrottleRule) (bool, error)
}

type Dependencies struct {
	Service          *internal.Service
	Challenges       internal.ChallengeWriter
	Credentials      *internal.Credentials
	CurrentUser      func(context.Context) (internal.User, error)
	DiscoverableUser func(context.Context, []byte, []byte) (internal.User, error)
	EstablishSession func(context.Context, internal.User, string, *connect.Response[lpbsv1.FinishPasskeyAuthenticationResponse]) error
	IssueNativeGrant func(context.Context, string, string, string, string) (string, error)
	Throttle         Throttle
	Notify           func(context.Context, string, string, string) error
	PublicOrigin     string
}
type Handler struct{ deps Dependencies }

func New(deps Dependencies) *Handler { return &Handler{deps: deps} }

func (h *Handler) user(ctx context.Context) (internal.User, error) {
	if h.deps.CurrentUser == nil {
		return internal.User{}, errors.New("customer session is not authenticated")
	}
	return h.deps.CurrentUser(ctx)
}
func save(ctx context.Context, store internal.ChallengeWriter, id, purpose, subject, binding string, session *webauthn.SessionData) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return store.Create(ctx, internal.Challenge{ID: id, Value: session.Challenge, Purpose: purpose, Subject: subject, BindingHash: internal.Hash(binding), SessionData: data})
}

func ceremonyBinding(headers http.Header) (string, error) {
	binding := internal.NormalizeBinding(headers.Get("X-Lpbs-Browser-Binding"))
	if len(binding) < 32 || len(binding) > 128 {
		return "", errors.New("browser binding is required")
	}
	return binding, nil
}

func (h *Handler) BeginRegistration(ctx context.Context, req *connect.Request[lpbsv1.BeginPasskeyRegistrationRequest]) (*connect.Response[lpbsv1.BeginPasskeyRegistrationResponse], error) {
	if h.deps.Service == nil || h.deps.Challenges == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("passkey service unavailable"))
	}
	u, err := h.user(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("customer session is not authenticated"))
	}
	// Connect request headers are the browser binding input; do not bind a
	// ceremony merely to the account identifier.
	// The generated request is intentionally opaque, so binding is read from
	// the transport request in the route-level middleware when available.
	binding, err := ceremonyBinding(req.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.deps.Throttle != nil {
		allowed, throttleErr := h.deps.Throttle.Allow(ctx, "passkey-register-user:"+u.ID, authadmin.ThrottleRule{Limit: 10, Window: 24 * time.Hour})
		if throttleErr != nil {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("passkey service unavailable"))
		}
		if !allowed {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("too many passkey registrations"))
		}
	}
	opts, session, err := h.deps.Service.BeginRegistration(u)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to begin passkey registration"))
	}
	id := internal.NewChallengeID()
	if err := save(ctx, h.deps.Challenges, id, "registration", u.ID, binding, session); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to save passkey ceremony"))
	}
	return connect.NewResponse(&lpbsv1.BeginPasskeyRegistrationResponse{OptionsJson: string(opts), CeremonyId: id}), nil
}

func (h *Handler) BeginAuthentication(ctx context.Context, req *connect.Request[lpbsv1.BeginPasskeyAuthenticationRequest]) (*connect.Response[lpbsv1.BeginPasskeyAuthenticationResponse], error) {
	if h.deps.Service == nil || h.deps.Challenges == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("passkey service unavailable"))
	}
	opts, session, err := h.deps.Service.BeginAuthentication()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to begin passkey authentication"))
	}
	binding, err := ceremonyBinding(req.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	id := internal.NewChallengeID()
	if err := save(ctx, h.deps.Challenges, id, "authentication", "discoverable", binding, session); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to save passkey ceremony"))
	}
	return connect.NewResponse(&lpbsv1.BeginPasskeyAuthenticationResponse{OptionsJson: string(opts), CeremonyId: id}), nil
}

func (h *Handler) FinishRegistration(ctx context.Context, req *connect.Request[lpbsv1.FinishPasskeyRegistrationRequest]) (*connect.Response[lpbsv1.FinishPasskeyRegistrationResponse], error) {
	if h.deps.Service == nil || h.deps.Challenges == nil || h.deps.Credentials == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("passkey service unavailable"))
	}
	u, err := h.user(ctx)
	if err != nil || strings.TrimSpace(req.Msg.GetCeremonyId()) == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("passkey ceremony is not valid"))
	}
	binding, err := ceremonyBinding(req.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	c, err := h.deps.Challenges.Consume(ctx, req.Msg.GetCeremonyId(), "registration", internal.Hash(binding))
	if err != nil || c.Subject != u.ID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("passkey ceremony is not valid or has expired"))
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(c.SessionData, &session); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("passkey ceremony is corrupt"))
	}
	credential, err := h.deps.Service.FinishRegistration(u, session, []byte(req.Msg.GetCredentialJson()), h.deps.PublicOrigin)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("passkey registration was rejected"))
	}
	id, err := h.deps.Credentials.Save(ctx, u.ID, session.RelyingPartyID, req.Msg.GetNickname(), *credential)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to save passkey"))
	}
	if h.deps.Notify != nil {
		_ = h.deps.Notify(ctx, u.Email, "passkey_added", "A passkey was added from "+req.Header().Get("User-Agent"))
	}
	return connect.NewResponse(&lpbsv1.FinishPasskeyRegistrationResponse{Id: id, Nickname: req.Msg.GetNickname()}), nil
}

func (h *Handler) FinishAuthentication(ctx context.Context, req *connect.Request[lpbsv1.FinishPasskeyAuthenticationRequest]) (*connect.Response[lpbsv1.FinishPasskeyAuthenticationResponse], error) {
	if h.deps.Service == nil || h.deps.Challenges == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("passkey service unavailable"))
	}
	if strings.TrimSpace(req.Msg.GetCeremonyId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("passkey ceremony is not valid"))
	}
	binding, err := ceremonyBinding(req.Header())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	c, err := h.deps.Challenges.Consume(ctx, req.Msg.GetCeremonyId(), "authentication", internal.Hash(binding))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("passkey ceremony is not valid or has expired"))
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(c.SessionData, &session); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("passkey ceremony is corrupt"))
	}
	if h.deps.DiscoverableUser == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("passkey authentication unavailable"))
	}
	user, credential, err := h.deps.Service.FinishDiscoverableLogin(func(rawID, userHandle []byte) (webauthn.User, error) {
		return h.deps.DiscoverableUser(ctx, rawID, userHandle)
	}, session, req.Msg.GetCredentialJson(), h.deps.PublicOrigin)
	if err != nil || user == nil || credential == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("passkey assertion was rejected"))
	}
	if h.deps.Credentials != nil {
		if err := h.deps.Credentials.RecordUse(ctx, *credential); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("passkey assertion was rejected"))
		}
	}
	passkeyUser, ok := user.(internal.User)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("passkey assertion was rejected"))
	}
	response := connect.NewResponse(&lpbsv1.FinishPasskeyAuthenticationResponse{Authenticated: true})
	if h.deps.EstablishSession != nil {
		if err := h.deps.EstablishSession(ctx, passkeyUser, req.Header().Get("User-Agent"), response); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("unable to establish customer session"))
		}
	}
	if h.deps.IssueNativeGrant != nil && strings.TrimSpace(req.Msg.GetContext()) != "" {
		code, err := h.deps.IssueNativeGrant(ctx, passkeyUser.ID, req.Msg.GetContext(), binding, req.Header().Get("User-Agent"))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("unable to continue native sign-in"))
		}
		var flow struct {
			RedirectURI string `json:"redirect_uri"`
			State       string `json:"state"`
			DesktopLink bool   `json:"desktop_link"`
		}
		if json.Unmarshal([]byte(req.Msg.GetContext()), &flow) == nil && flow.RedirectURI != "" && !flow.DesktopLink {
			separator := "?"
			if strings.Contains(flow.RedirectURI, "?") {
				separator = "&"
			}
			response.Msg.RedirectUrl = flow.RedirectURI + separator + "code=" + url.QueryEscape(code)
			if flow.State != "" {
				response.Msg.RedirectUrl += "&state=" + url.QueryEscape(flow.State)
			}
		}
	}
	return response, nil
}
func (h *Handler) ListPasskeys(ctx context.Context, _ *connect.Request[lpbsv1.ListPasskeysRequest]) (*connect.Response[lpbsv1.ListPasskeysResponse], error) {
	if h.deps.Credentials == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("passkey service unavailable"))
	}
	u, err := h.user(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("customer session is not authenticated"))
	}
	records, err := h.deps.Credentials.List(ctx, u.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unable to list passkeys"))
	}
	out := make([]*lpbsv1.Passkey, 0, len(records))
	for _, record := range records {
		p := &lpbsv1.Passkey{Id: record.ID, Nickname: record.Nickname, CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339)}
		if record.LastUsedAt.Valid {
			p.LastUsedAt = record.LastUsedAt.Time.UTC().Format(time.RFC3339)
		}
		if record.BackupState.Valid {
			p.BackupState = record.BackupState.String
		}
		out = append(out, p)
	}
	return connect.NewResponse(&lpbsv1.ListPasskeysResponse{Passkeys: out}), nil
}
func (h *Handler) RenamePasskey(ctx context.Context, req *connect.Request[lpbsv1.RenamePasskeyRequest]) (*connect.Response[lpbsv1.RenamePasskeyResponse], error) {
	if h.deps.Credentials == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("passkey service unavailable"))
	}
	u, err := h.user(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("customer session is not authenticated"))
	}
	if err := h.deps.Credentials.Rename(ctx, u.ID, req.Msg.GetId(), req.Msg.GetNickname()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("passkey could not be renamed"))
	}
	return connect.NewResponse(&lpbsv1.RenamePasskeyResponse{Passkey: &lpbsv1.Passkey{Id: req.Msg.GetId(), Nickname: strings.TrimSpace(req.Msg.GetNickname())}}), nil
}
func (h *Handler) RevokePasskey(ctx context.Context, req *connect.Request[lpbsv1.RevokePasskeyRequest]) (*connect.Response[lpbsv1.RevokePasskeyResponse], error) {
	if h.deps.Credentials == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("passkey service unavailable"))
	}
	u, err := h.user(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("customer session is not authenticated"))
	}
	if err := h.deps.Credentials.Revoke(ctx, u.ID, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("passkey not found"))
	}
	if h.deps.Notify != nil {
		_ = h.deps.Notify(ctx, u.Email, "passkey_removed", "A passkey was removed from "+req.Header().Get("User-Agent"))
	}
	return connect.NewResponse(&lpbsv1.RevokePasskeyResponse{Revoked: true}), nil
}

var _ lpbsconnect.PasskeyServiceHandler = (*Handler)(nil)
