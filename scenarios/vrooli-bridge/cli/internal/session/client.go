// Package session supplies the bridge CLI's owner-session-aware Connect
// transport. A single expired owner call gets one refresh attempt, then the
// original request is replayed once with the new access token.
package session

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	identityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity"
	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/identity/identity_v1connect"
)

type client struct {
	app     *cliapp.ScenarioApp
	base    connect.HTTPClient
	baseURL string
}

// NewConnectHTTPClient returns the standard bridge CLI transport with one
// transparent owner-token renewal on an unauthenticated response.
func NewConnectHTTPClient(app *cliapp.ScenarioApp) (connect.HTTPClient, string) {
	base, baseURL := cliapp.NewConnectHTTPClient(app)
	return &client{app: app, base: base, baseURL: baseURL}, baseURL
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	if token, err := BreakGlassTokenFile(); err != nil {
		return nil, err
	} else if strings.TrimSpace(token) != "" {
		return c.doBreakGlass(req, token)
	}
	if c.app != nil && strings.TrimSpace(c.app.Config.Token) == "" {
		// Local exchange is the password-free acquisition path. Failure is not
		// an authorization decision here: the request continues anonymously so
		// public RPCs and the interactive login fallback retain their normal
		// behavior.
		if token, refresh, err := ExchangeLocal(req.Context()); err == nil {
			_ = c.persist(token, refresh)
		} else if token, tokenErr := TokenFile(); tokenErr == nil {
			c.app.Config.Token = token
		}
	}
	resp, err := c.base.Do(req)
	if err != nil || resp == nil || resp.StatusCode != http.StatusUnauthorized || skipRefresh(req.URL.Path) {
		return resp, err
	}
	if strings.TrimSpace(c.app.Config.RefreshToken) == "" {
		return resp, err
	}

	newToken, newRefresh, refreshErr := c.refresh(req.Context())
	if refreshErr != nil {
		// Preserve the original response. Connect will report the original
		// unauthenticated operation, never the failed recovery attempt.
		return resp, err
	}
	if err := c.persist(newToken, newRefresh); err != nil {
		// A local config write failure must not replace the original
		// unauthenticated response with an implementation detail.
		return resp, nil
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}
	retry, cloneErr := replayRequest(req)
	if cloneErr != nil {
		return resp, cloneErr
	}
	return c.base.Do(retry)
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

func (c *client) refresh(ctx context.Context) (string, string, error) {
	refreshClient := identityconnect.NewIdentityServiceClient(c.base, c.baseURL)
	resp, err := refreshClient.Refresh(ctx, connect.NewRequest(&identityv1.RefreshRequest{
		RefreshToken: c.app.Config.RefreshToken,
	}))
	if err != nil {
		return "", "", err
	}
	if resp == nil || resp.Msg == nil || strings.TrimSpace(resp.Msg.Token) == "" || strings.TrimSpace(resp.Msg.RefreshToken) == "" {
		return "", "", fmt.Errorf("refresh returned an incomplete owner session")
	}
	return resp.Msg.Token, resp.Msg.RefreshToken, nil
}

func (c *client) persist(token, refresh string) error {
	previous := c.app.Config
	c.app.Config.Token = token
	c.app.Config.RefreshToken = refresh
	if err := c.app.SaveConfig(); err != nil {
		c.app.Config = previous
		return fmt.Errorf("save refreshed owner session: %w", err)
	}
	return nil
}

func replayRequest(req *http.Request) (*http.Request, error) {
	if req == nil || req.GetBody == nil {
		return nil, fmt.Errorf("cannot retry an owner request whose body is not replayable")
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	retry := req.Clone(req.Context())
	retry.Body = body
	if closer, ok := body.(io.ReadCloser); ok {
		retry.Body = closer
	}
	return retry, nil
}

func skipRefresh(path string) bool {
	path = strings.TrimRight(path, "/")
	return strings.HasSuffix(path, "IdentityService/Login") ||
		strings.HasSuffix(path, "IdentityService/Register") ||
		strings.HasSuffix(path, "IdentityService/Refresh")
}
