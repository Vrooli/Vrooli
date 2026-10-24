package cliapp

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
)

type scenarioConnectHTTPClient struct {
	app     *ScenarioApp
	client  *http.Client
	baseURL string
}

// NewConnectHTTPClient returns an HTTP client and root base URL for generated
// Connect-Go clients, for example:
//
//	client, baseURL := cliapp.NewConnectHTTPClient(app)
//	notes := notesconnect.NewNotesClient(client, baseURL)
func NewConnectHTTPClient(app *ScenarioApp) (connect.HTTPClient, string) {
	if app == nil {
		return &scenarioConnectHTTPClient{}, ""
	}
	var timeout time.Duration
	if app.HTTPClient != nil {
		timeout = app.HTTPClient.Timeout()
	}
	return NewConnectHTTPClientWithTimeout(app, timeout)
}

// NewConnectHTTPClientWithTimeout is NewConnectHTTPClient with an explicit
// per-client timeout. Use it for inherently long-running RPCs (e.g. a handler
// that synchronously triggers a multi-minute test suite) where the scenario's
// default client timeout would abort the call while the server is still
// working. A zero timeout means no client-side deadline.
func NewConnectHTTPClientWithTimeout(app *ScenarioApp, timeout time.Duration) (connect.HTTPClient, string) {
	if app == nil {
		return &scenarioConnectHTTPClient{}, ""
	}
	baseURL := ""
	if app.baseOptions != nil {
		baseURL = app.APIRootBase()
	}
	httpClient := &http.Client{Timeout: timeout}
	return &scenarioConnectHTTPClient{app: app, client: httpClient, baseURL: baseURL}, baseURL
}

func (c *scenarioConnectHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c == nil || c.app == nil {
		return nil, fmt.Errorf("connect http client is not configured")
	}
	if req == nil {
		return nil, fmt.Errorf("connect request is nil")
	}
	// Generated clients are often constructed before CLI flags are parsed.
	// Resolve requests rooted at that construction base through the invocation's
	// current selection, retaining explicit unrelated absolute request URLs.
	if initial, err := url.Parse(c.baseURL); err == nil && initial.Scheme != "" && req.URL != nil {
		prefix := strings.TrimRight(initial.Path, "/") + "/"
		if req.URL.Scheme == initial.Scheme && req.URL.Host == initial.Host && strings.HasPrefix(req.URL.Path, prefix) {
			req = req.Clone(req.Context())
			req.URL.Scheme, req.URL.Host, req.URL.User = "", "", nil
			req.URL.Path = "/" + strings.TrimPrefix(req.URL.Path, prefix)
			req.URL.RawPath = ""
			req.Host = ""
		}
	}
	if c.app.HTTPClient != nil {
		c.app.HTTPClient.SetToken(strings.TrimSpace(c.app.tokenSource()))
		c.app.HTTPClient.ApplyRequestHeaders(req)
	}
	// Connect clients use a private http.Client, so do not rely solely on the
	// process-wide identity forwarding transport. Apply the run token directly
	// when the caller has not already supplied one.
	if req.Header.Get(cliutil.HeaderAgentIdentityToken) == "" {
		if token := strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)); token != "" {
			req.Header.Set(cliutil.HeaderAgentIdentityToken, token)
		}
	}
	resolveRelativeConnectURL(c.app, req)
	if err := validateAbsoluteURL(req.URL); err != nil {
		return nil, err
	}
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

func resolveRelativeConnectURL(app *ScenarioApp, req *http.Request) {
	if app == nil || req == nil || req.URL == nil {
		return
	}
	if req.URL.Scheme != "" && req.URL.Host != "" {
		return
	}
	base := strings.TrimRight(strings.TrimSpace(app.APIRootBase()), "/")
	if base == "" {
		return
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return
	}
	rel := req.URL
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(rel.Path, "/")
	parsed.RawPath = ""
	parsed.RawQuery = rel.RawQuery
	parsed.Fragment = rel.Fragment
	req.URL = parsed
}

func validateAbsoluteURL(u *url.URL) error {
	if u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("connect request URL must be absolute")
	}
	return nil
}
