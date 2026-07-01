package cliapp

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
)

type scenarioConnectHTTPClient struct {
	app    *ScenarioApp
	client *http.Client
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
	return &scenarioConnectHTTPClient{app: app, client: httpClient}, baseURL
}

func (c *scenarioConnectHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c == nil || c.app == nil {
		return nil, fmt.Errorf("connect http client is not configured")
	}
	if req == nil {
		return nil, fmt.Errorf("connect request is nil")
	}
	if c.app.HTTPClient != nil {
		c.app.HTTPClient.SetToken(strings.TrimSpace(c.app.tokenSource()))
		c.app.HTTPClient.ApplyRequestHeaders(req)
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
