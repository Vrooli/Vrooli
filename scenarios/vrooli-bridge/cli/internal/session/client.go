// Package session supplies the bridge CLI's owner-session-aware Connect
// transport. A single expired owner call gets one refresh attempt, then the
// original request is replayed once with the new access token.
package session

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	sharedsession "github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/cli-core/cliapp"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity"
	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity/identity_v1connect"
)

type client struct {
	app      *cliapp.ScenarioApp
	base     connect.HTTPClient
	baseURL  string
	exchange func(context.Context) (string, string, error)
	mu       sync.Mutex
}

// NewConnectHTTPClient returns the standard Bridge transport. Enrolled
// operators mint a short-lived local session for each request; enrollment is
// the only path that may contact the authenticator.
func NewConnectHTTPClient(app *cliapp.ScenarioApp) (connect.HTTPClient, string) {
	return NewConnectHTTPClientWithTimeout(app, 0)
}

// NewConnectHTTPClientWithTimeout is NewConnectHTTPClient with an explicit
// transport timeout. Long-running Bridge workflows (such as onboarding) use
// this to keep the client attached while the server-owned operation blocks.
// A zero timeout preserves cli-core's normal application-configured timeout.
func NewConnectHTTPClientWithTimeout(app *cliapp.ScenarioApp, timeout time.Duration) (connect.HTTPClient, string) {
	var base connect.HTTPClient
	var baseURL string
	if timeout > 0 {
		base, baseURL = cliapp.NewConnectHTTPClientWithTimeout(app, timeout)
	} else {
		base, baseURL = cliapp.NewConnectHTTPClient(app)
	}
	return &client{app: app, base: base, baseURL: baseURL, exchange: ExchangeLocal}, baseURL
}

// EnrollLocalWithToken completes the one-time enrollment using an access token
// already obtained by an explicit login flow. The access token is used only
// for this RPC and is not persisted by this helper.
func EnrollLocalWithToken(ctx context.Context, app *cliapp.ScenarioApp, access string) error {
	if app == nil || strings.TrimSpace(access) == "" {
		return errors.New("operator enrollment requires a Bridge app and access token")
	}
	base, baseURL := cliapp.NewConnectHTTPClient(app)
	private, err := sharedsession.GenerateKey()
	if err != nil {
		return err
	}
	defer clearBytes(private)
	public, err := sharedsession.PublicKey(private)
	if err != nil {
		return err
	}
	c := &client{app: app, base: base, baseURL: baseURL, exchange: ExchangeLocal}
	if _, err := c.enrollWithAccess(ctx, private, public, access); err != nil {
		return err
	}
	return nil
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	if token, err := BreakGlassTokenFile(); err != nil {
		return nil, err
	} else if strings.TrimSpace(token) != "" {
		return c.doBreakGlass(req, token)
	}
	// Identity RPCs precede enrollment and must remain callable to perform the
	// one-time password/token exchange. They do not inherit a stale local
	// session and never trigger an authenticator call through this transport.
	if skipEnrollment(req.URL.Path) {
		return c.base.Do(req)
	}

	// An enrolled client mints a short-lived local credential without calling
	// the authenticator. This is also the expired-session refresh path: minting
	// a new local credential is enough, so no bearer or refresh token is needed.
	var enrollmentErr error
	if c.app != nil {
		if localToken, _, localErr := mintLocalSession(time.Now()); localErr == nil {
			if err := c.clearLegacyConfig(); err != nil {
				return nil, err
			}
			return c.doLocal(req, localToken)
		} else {
			enrollmentErr = localErr
		}
		// A deliberately supplied token (for example a one-time token file or a
		// caller that just completed login) may establish the enrollment. It is
		// consumed for this one RPC and then removed from the saved config.
		if access := strings.TrimSpace(c.app.Config.Token); access != "" {
			if localToken, enrollErr := c.enrollWithAccess(req.Context(), nil, nil, access); enrollErr == nil {
				return c.doLocal(req, localToken)
			} else {
				enrollmentErr = diagnoseCredential(enrollErr)
			}
		} else if localToken, enrollErr := c.enrollLocal(req.Context()); enrollErr == nil {
			return c.doLocal(req, localToken)
		} else {
			enrollmentErr = enrollErr
		}
	}

	resp, err := c.base.Do(req)
	if err != nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}
	if enrollmentErr != nil {
		return nil, diagnoseEnrollment(enrollmentErr)
	}
	return resp, err
}

func (c *client) clearLegacyConfig() error {
	if c.app == nil || (strings.TrimSpace(c.app.Config.Token) == "" && strings.TrimSpace(c.app.Config.RefreshToken) == "") {
		return nil
	}
	previous := c.app.Config
	c.app.Config.Token = ""
	c.app.Config.RefreshToken = ""
	if err := c.app.SaveConfig(); err != nil {
		c.app.Config = previous
		return fmt.Errorf("clear legacy owner session after local resolution: %w", err)
	}
	return nil
}

func (c *client) doLocal(req *http.Request, token string) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.app == nil || c.app.HTTPClient == nil {
		return nil, fmt.Errorf("local owner session client is not configured")
	}
	previous := c.app.Config.Token
	defer func() {
		c.app.Config.Token = previous
		c.app.HTTPClient.SetToken(previous)
	}()
	// cli-core's transport applies Bearer from Config.Token. Clear that
	// transient value and set the explicit scheme after request construction so
	// the local credential cannot be mistaken for an authenticator JWT.
	c.app.Config.Token = ""
	c.app.HTTPClient.SetToken("")
	req.Header.Set("Authorization", sharedsession.LocalSessionScheme+" "+token)
	return c.base.Do(req)
}

func (c *client) enrollLocal(ctx context.Context) (string, error) {
	private, err := sharedsession.GenerateKey()
	if err != nil {
		return "", err
	}
	defer clearBytes(private)
	public, err := sharedsession.PublicKey(private)
	if err != nil {
		return "", err
	}
	access, _, err := c.exchangeLocal(ctx)
	if err != nil {
		return "", diagnoseEnrollment(err)
	}
	return c.enrollWithAccess(ctx, private, public, access)
}

func (c *client) enrollWithAccess(ctx context.Context, private ed25519.PrivateKey, public ed25519.PublicKey, access string) (string, error) {
	if private == nil || public == nil {
		var err error
		private, err = sharedsession.GenerateKey()
		if err != nil {
			return "", err
		}
		defer clearBytes(private)
		public, err = sharedsession.PublicKey(private)
		if err != nil {
			return "", err
		}
	}
	previous := c.app.Config.Token
	c.app.Config.Token = access
	enrolled := false
	defer func() {
		if !enrolled {
			c.app.Config.Token = previous
		}
	}()
	identityClient := identityconnect.NewIdentityServiceClient(c.base, c.baseURL)
	resp, err := identityClient.EnrollOperatorSession(ctx, connect.NewRequest(&identityv1.EnrollOperatorSessionRequest{PublicKey: public, Mode: string(sharedsession.ModePersonal)}))
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Msg == nil || strings.TrimSpace(resp.Msg.EnrollmentReference) == "" || strings.TrimSpace(resp.Msg.OperatorId) == "" {
		return "", errors.New("operator enrollment returned an incomplete record")
	}
	enrolledAt := time.Now().UTC()
	if resp.Msg.EnrolledAt != nil {
		enrolledAt = resp.Msg.EnrolledAt.AsTime()
	}
	enrollment := sharedsession.Enrollment{OperatorID: resp.Msg.OperatorId, IdentityProvider: resp.Msg.IdentityProvider, Mode: sharedsession.Mode(resp.Msg.Mode), Reference: resp.Msg.EnrollmentReference, EnrolledAt: enrolledAt, ScopeCeiling: append([]string(nil), resp.Msg.ScopeCeiling...)}
	if err := saveLocalEnrollment(private, enrollment); err != nil {
		return "", err
	}
	// Enrollment supersedes legacy bearer config. Clear it in memory and on
	// disk immediately after the durable local binding succeeds.
	c.app.Config.Token = ""
	c.app.Config.RefreshToken = ""
	if err := c.app.SaveConfig(); err != nil {
		return "", fmt.Errorf("clear legacy owner session after enrollment: %w", err)
	}
	enrolled = true
	return sharedsession.Mint(private, enrollment.Reference, enrollment.OperatorID, enrollment.ScopeCeiling, time.Now(), sharedsession.LocalSessionTTL)
}

func diagnoseEnrollment(cause error) error {
	if cause == nil {
		cause = errors.New("operator enrollment failed")
	}
	if connect.CodeOf(cause) == connect.CodeUnauthenticated {
		return sharedsession.EnrollmentRequired(cause, sharedsession.ModePersonal)
	}
	return sharedsession.ProviderUnavailable(cause, sharedsession.ModePersonal)
}

func diagnoseCredential(cause error) error {
	if cause == nil {
		cause = errors.New("owner credential was rejected")
	}
	if connect.CodeOf(cause) == connect.CodeUnauthenticated {
		return sharedsession.Diagnosis{Kind: sharedsession.DiagnosisUnauthenticated, Provider: sharedsession.IdentityProviderAuthenticator, Mode: sharedsession.ModePersonal, Reason: cause.Error(), Recovery: "authenticate again and enroll this machine"}
	}
	return cause
}

func (c *client) exchangeLocal(ctx context.Context) (string, string, error) {
	if c.exchange == nil {
		return "", "", fmt.Errorf("local owner exchange is not configured")
	}
	return c.exchange(ctx)
}

func (c *client) doBreakGlass(req *http.Request, token string) (*http.Response, error) {
	if c.app == nil || req == nil {
		return nil, fmt.Errorf("break-glass client is not configured")
	}
	resolveRelativeURL(c.app, req)
	if err := validateURL(req.URL); err != nil {
		return nil, err
	}
	if c.app.HTTPClient != nil {
		c.app.HTTPClient.SetToken(token)
		c.app.HTTPClient.ApplyRequestHeaders(req)
	}
	req.Header.Set("Authorization", "BreakGlass "+token)
	timeout := 0 * time.Second
	if c.app.HTTPClient != nil {
		timeout = c.app.HTTPClient.Timeout()
	}
	return (&http.Client{Timeout: timeout}).Do(req)
}

func resolveRelativeURL(app *cliapp.ScenarioApp, req *http.Request) {
	if app == nil || req == nil || req.URL == nil || (req.URL.Scheme != "" && req.URL.Host != "") {
		return
	}
	base, err := url.Parse(app.APIRootBase())
	if err != nil || base.Scheme == "" || base.Host == "" {
		return
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(req.URL.Path, "/")
	base.RawPath = ""
	base.RawQuery = req.URL.RawQuery
	base.Fragment = req.URL.Fragment
	req.URL = base
}

func validateURL(value *url.URL) error {
	if value == nil || value.Scheme == "" || value.Host == "" {
		return fmt.Errorf("connect request URL must be absolute")
	}
	return nil
}

func skipEnrollment(path string) bool {
	path = strings.TrimRight(path, "/")
	return strings.HasSuffix(path, "IdentityService/Login") ||
		strings.HasSuffix(path, "IdentityService/Register") ||
		strings.HasSuffix(path, "IdentityService/Refresh") ||
		strings.HasSuffix(path, "IdentityService/EnrollOperatorSession")
}
