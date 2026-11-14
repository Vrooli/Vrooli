// Package browsersession is the same-origin browser edge to the installation
// identity provider. It owns no passwords, token issuance, or permission grants.
package browsersession

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

const accessCookie = "vrooli_access_token"
const refreshCookie = "vrooli_refresh_token"

type accountsClient interface {
	Login(context.Context, *connect.Request[accountsv1.LoginRequest]) (*connect.Response[accountsv1.LoginResponse], error)
	Refresh(context.Context, *connect.Request[accountsv1.RefreshRequest]) (*connect.Response[accountsv1.RefreshResponse], error)
	Logout(context.Context, *connect.Request[accountsv1.LogoutRequest]) (*connect.Response[accountsv1.LogoutResponse], error)
}
type handler struct {
	client func(context.Context) (accountsClient, error)
}

// Register discovers an already configured provider lazily. A stopped provider
// produces an explicit unavailable response, never an implicit lifecycle action.
func Register(mux *http.ServeMux) {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	h := &handler{client: func(ctx context.Context) (accountsClient, error) {
		base, err := resolver.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
		if err != nil {
			return nil, err
		}
		return accountsconnect.NewAccountsServiceClient(client, base), nil
	}}
	mux.Handle("/api/session", h)
	mux.Handle("/api/session/", h)
}

type sessionView struct {
	Authenticated bool   `json:"authenticated"`
	Email         string `json:"email,omitempty"`
	Subject       string `json:"subject,omitempty"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
	MFARequired   bool   `json:"mfaRequired,omitempty"`
	MFAChallenge  string `json:"mfaChallenge,omitempty"`
	Message       string `json:"message,omitempty"`
}

func reply(w http.ResponseWriter, status int, view sessionView) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(view)
}
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Path == "/api/session" && r.Method == http.MethodGet {
		view := sessionView{}
		if p, ok := identity.PrincipalFromContext(r.Context()); ok && p.Verified && p.Subject != "" && !p.ExpiresAt.IsZero() && time.Now().Before(p.ExpiresAt) {
			view = sessionView{Authenticated: true, Email: p.Email, Subject: p.Subject, ExpiresAt: p.ExpiresAt.UTC().Format(time.RFC3339)}
		}
		reply(w, 200, view)
		return
	}
	if r.URL.Path != "/api/session/login" && r.URL.Path != "/api/session/refresh" && r.URL.Path != "/api/session/logout" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		reply(w, 405, sessionView{Message: "Use POST."})
		return
	}
	// A custom header makes non-browser callers explicit and forces cross-origin
	// browser requests through preflight. Also reject hostile Origin/Fetch metadata.
	if r.Header.Get("X-Vrooli-Session") != "1" || !sameOrigin(r) {
		reply(w, 403, sessionView{Message: "A same-origin session request is required."})
		return
	}
	if r.URL.Path == "/api/session/logout" {
		clearCookies(w, r)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	client, err := h.client(ctx)
	if err != nil {
		if r.URL.Path == "/api/session/logout" {
			reply(w, 503, sessionView{Message: "Signed out locally. The installation provider is unavailable; server session revocation was not confirmed."})
		} else {
			reply(w, 503, sessionView{Message: "The installation sign-in provider is unavailable."})
		}
		return
	}
	switch r.URL.Path {
	case "/api/session/login":
		if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
			reply(w, 415, sessionView{Message: "Use a JSON sign-in request."})
			return
		}
		var body struct {
			Email        string `json:"email"`
			Password     string `json:"password"`
			TOTPCode     string `json:"totpCode"`
			RecoveryCode string `json:"recoveryCode"`
			MFAChallenge string `json:"mfaChallenge"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&body) != nil || decoder.Decode(&struct{}{}) != io.EOF || body.Email == "" || body.Password == "" {
			reply(w, 400, sessionView{Message: "Enter your email and password."})
			return
		}
		req := connect.NewRequest(&accountsv1.LoginRequest{Email: body.Email, Password: body.Password, Resource: "lighting-control", TotpCode: body.TOTPCode, RecoveryCode: body.RecoveryCode, MfaChallenge: body.MFAChallenge})
		forwardMetadata(req.Header(), r)
		res, err := client.Login(ctx, req)
		if err != nil {
			authError(w, err)
			return
		}
		if res.Msg.GetMfaRequired() {
			reply(w, 200, sessionView{MFARequired: true, MFAChallenge: res.Msg.GetMfaChallenge()})
			return
		}
		if res.Msg.GetTokens().GetAccessToken() == "" {
			reply(w, 502, sessionView{Message: "The sign-in provider did not establish a session."})
			return
		}
		if !copyCookies(w, r, res.Header()) {
			reply(w, 502, sessionView{Message: "The sign-in provider did not establish a browser session."})
			return
		}
		reply(w, 200, sessionView{Message: "Signed in. Checking verified session."})
	case "/api/session/refresh":
		token, err := r.Cookie(refreshCookie)
		if err != nil {
			reply(w, 401, sessionView{Message: "Sign in to continue."})
			return
		}
		req := connect.NewRequest(&accountsv1.RefreshRequest{RefreshToken: token.Value})
		forwardMetadata(req.Header(), r)
		res, err := client.Refresh(ctx, req)
		if err != nil {
			authError(w, err)
			return
		}
		if !copyCookies(w, r, res.Header()) {
			reply(w, 502, sessionView{Message: "The sign-in provider did not renew the browser session."})
			return
		}
		reply(w, 200, sessionView{Message: "Session renewed."})
	case "/api/session/logout":
		token, err := r.Cookie(accessCookie)
		if err == nil && token.Value != "" {
			req := connect.NewRequest(&accountsv1.LogoutRequest{AccessToken: token.Value})
			forwardMetadata(req.Header(), r)
			if _, err := client.Logout(ctx, req); err != nil {
				reply(w, 503, sessionView{Message: "Signed out locally. Server session revocation was not confirmed; retry through the installation identity provider."})
				return
			}
		}
		reply(w, 200, sessionView{Message: "Signed out."})
	}
}
func sameOrigin(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	authority := r.Host
	peer, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip := net.ParseIP(peer); ip != nil && ip.IsLoopback() {
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwarded != "" && !strings.ContainsAny(forwarded, ",/\\") {
			authority = forwarded
		}
	}
	u, err := url.Parse(origin)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host == authority && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}

// A validated HTTPS Origin retains browser TLS through the same-origin UI
// proxy. Forwarded scheme headers alone never establish secure transport.
func secureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	u, err := url.Parse(r.Header.Get("Origin"))
	return err == nil && u.Scheme == "https" && sameOrigin(r)
}
func forwardMetadata(h http.Header, r *http.Request) {
	h.Set("User-Agent", r.UserAgent())
	if secureRequest(r) {
		h.Set("X-Forwarded-Proto", "https")
	}
}
func copyCookies(w http.ResponseWriter, r *http.Request, h http.Header) bool {
	cookies := (&http.Response{Header: h}).Cookies()
	var accepted []*http.Cookie
	access := false
	for _, c := range cookies {
		if c.Name != accessCookie && c.Name != refreshCookie {
			continue
		}
		if c.Value == "" {
			continue
		}
		c.Path = "/"
		c.Domain = ""
		c.HttpOnly = true
		c.SameSite = http.SameSiteLaxMode
		c.Secure = c.Secure || secureRequest(r)
		accepted = append(accepted, c)
		if c.Name == accessCookie {
			access = true
		}
	}
	if !access {
		return false
	}
	for _, c := range accepted {
		http.SetCookie(w, c)
	}
	return true
}
func clearCookies(w http.ResponseWriter, r *http.Request) {
	for _, name := range []string{accessCookie, refreshCookie} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secureRequest(r), SameSite: http.SameSiteLaxMode})
	}
}
func authError(w http.ResponseWriter, err error) {
	var ce *connect.Error
	if errors.As(err, &ce) {
		switch ce.Code() {
		case connect.CodeUnauthenticated, connect.CodePermissionDenied:
			reply(w, 401, sessionView{Message: "Sign-in or session verification failed."})
			return
		case connect.CodeResourceExhausted:
			reply(w, 429, sessionView{Message: "Too many sign-in attempts. Try again later."})
			return
		case connect.CodeInvalidArgument:
			reply(w, 400, sessionView{Message: "Check your sign-in details."})
			return
		}
	}
	reply(w, 503, sessionView{Message: "The installation sign-in provider is unavailable."})
}
