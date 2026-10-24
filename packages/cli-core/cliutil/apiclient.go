package cliutil

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"
)

// APIClient wraps HTTPClient and applies API base resolution and token wiring.
type APIClient struct {
	client       *HTTPClient
	baseResolver func() APIBaseOptions
	tokenSource  func() string
}

func NewAPIClient(client *HTTPClient, baseResolver func() APIBaseOptions, tokenSource func() string) *APIClient {
	return &APIClient{
		client:       client,
		baseResolver: baseResolver,
		tokenSource:  tokenSource,
	}
}

// WithToken returns an isolated request client with an explicitly selected
// in-memory credential. It preserves transport/provenance and never changes or
// persists the receiver's authentication. Authentication policy belongs to the
// caller; this method must not be used as an implicit privilege fallback.
func (c *APIClient) WithToken(token string) *APIClient {
	if c == nil {
		return nil
	}
	clone := *c
	base := c.client
	if base == nil {
		base = NewHTTPClient(HTTPClientOptions{})
	}
	copyClient := *base
	clone.client = &copyClient
	clone.tokenSource = func() string { return token }
	return &clone
}

// WithoutRedirects returns a request-local transport that refuses every redirect.
// Authentication, attribution, transport, cookies and timeout are preserved;
// the receiver and ordinary clients keep their original redirect policy.
func (c *APIClient) WithoutRedirects() *APIClient {
	if c == nil {
		return nil
	}
	clone := *c
	base := c.client
	if base == nil {
		base = NewHTTPClient(HTTPClientOptions{})
	}
	client := *base
	client.rejectRedirectResponses = true
	raw := base.client
	if raw == nil {
		raw = &http.Client{}
	}
	protected := *raw
	protected.CheckRedirect = func(*http.Request, []*http.Request) error {
		return errors.New("protected request redirect refused")
	}
	client.client = &protected
	clone.client = &client
	return &clone
}

// WithTimeout returns a copy of this client whose requests use timeout instead
// of the CLI default. Base resolution, token wiring, and provenance headers are
// preserved, so an operator-initiated maintenance call that legitimately runs
// for minutes does not have to choose between timing out and losing its
// attribution. The receiver is unchanged.
// WithHeaders returns an isolated transport with additional request headers.
// Existing application and invocation provenance sources are preserved.
func (c *APIClient) WithHeaders(headers map[string]string) *APIClient {
	if c == nil {
		return nil
	}
	clone := *c
	client := c.client
	if client == nil {
		client = NewHTTPClient(HTTPClientOptions{})
	}
	copyClient := *client
	previous := copyClient.headerSource
	pinned := make(map[string]string, len(headers))
	for k, v := range headers {
		pinned[k] = v
	}
	copyClient.headerSource = func() map[string]string {
		merged := make(map[string]string)
		if previous != nil {
			for k, v := range previous() {
				merged[k] = v
			}
		}
		for k, v := range pinned {
			merged[k] = v
		}
		return merged
	}
	clone.client = &copyClient
	return &clone
}

func (c *APIClient) WithTimeout(timeout time.Duration) *APIClient {
	if c == nil || timeout <= 0 {
		return c
	}
	clone := *c
	clone.client = c.client.CloneWithTimeout(timeout)
	return &clone
}

// WithoutTimeout creates an explicit server-owned wait attachment. It keeps
// base resolution, authentication, provenance and transport configuration,
// without changing the ordinary client's deadline or retrying the request.
func (c *APIClient) WithoutTimeout() *APIClient {
	if c == nil {
		return nil
	}
	clone := *c
	client := c.client
	if client == nil {
		client = NewHTTPClient(HTTPClientOptions{})
	}
	clone.client = client.CloneWithoutTimeout()
	return &clone
}

func (c *APIClient) Get(path string, query url.Values) ([]byte, error) {
	return c.Request("GET", path, query, nil)
}

func (c *APIClient) Request(method, path string, query url.Values, body interface{}) ([]byte, error) {
	if c.client == nil {
		c.client = NewHTTPClient(HTTPClientOptions{})
	}
	if c.baseResolver != nil {
		c.client.SetBaseOptions(c.baseResolver())
	}
	if c.tokenSource != nil {
		c.client.SetToken(c.tokenSource())
	}
	return c.client.Do(method, path, query, body)
}

// BaseURL returns the resolved API base URL.
func (c *APIClient) BaseURL() string {
	if c.baseResolver == nil {
		return ""
	}
	opts := c.baseResolver()
	base, _ := ValidateAPIBase(opts)
	return base
}

// AuthHeaders returns a map of authentication headers.
func (c *APIClient) AuthHeaders() map[string]string {
	headers := make(map[string]string)
	if c.tokenSource != nil {
		token := c.tokenSource()
		if token != "" {
			headers["Authorization"] = "Bearer " + token
		}
	}
	return headers
}

// WithPinnedAgentCredential carries one protected existing credential through
// a paused exact-action signing flow. Each wire request rechecks its authority;
// refusals and any credential change are terminal. No attach or token creation.
func (c *APIClient) WithPinnedAgentCredential(token string, resolve func(context.Context) (string, error)) *APIClient {
	if c == nil || token == "" || resolve == nil {
		return nil
	}
	clone := *c
	base := c.client
	if base == nil {
		base = NewHTTPClient(HTTPClientOptions{})
	}
	client := *base
	client.identityPin = &agentCredentialPin{token: token, resolve: resolve}
	// Keep redirect and wire custody policy private to this pinned flow. Ordinary
	// clients retain their exact transport, cookie jar, timeout and redirect policy.
	raw := base.client
	if raw == nil {
		raw = &http.Client{}
	}
	protected := *raw
	protected.CheckRedirect = func(*http.Request, []*http.Request) error {
		return errors.New("pinned exact request redirect refused")
	}
	next := raw.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	protected.Transport = identityForwardingTransport{next: next}
	client.client = &protected
	clone.client = &client
	clone.tokenSource = func() string { return "" }
	return &clone
}

// WithPinnedAgentCredentials pins the exact baseline/operation pair used by a
// protected request. Both proofs are re-resolved before every wire send.
func (c *APIClient) WithPinnedAgentCredentials(credentials AgentCredentials, resolve func(context.Context) (AgentCredentials, error)) *APIClient {
	if c == nil || resolve == nil || credentials.BaselineToken == "" || credentials.OperationToken == "" {
		return nil
	}
	clone := *c
	base := c.client
	if base == nil {
		base = NewHTTPClient(HTTPClientOptions{})
	}
	client := *base
	client.identityPin = &agentCredentialPin{token: credentials.BaselineToken, operationToken: credentials.OperationToken, resolvePair: resolve}
	raw := base.client
	if raw == nil {
		raw = &http.Client{}
	}
	protected := *raw
	protected.CheckRedirect = func(*http.Request, []*http.Request) error {
		return errors.New("pinned exact request redirect refused")
	}
	next := raw.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	protected.Transport = identityForwardingTransport{next: next}
	client.client = &protected
	clone.client = &client
	clone.tokenSource = func() string { return "" }
	return &clone
}
