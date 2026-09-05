package cliutil

import (
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

// WithTimeout returns a copy of this client whose requests use timeout instead
// of the CLI default. Base resolution, token wiring, and provenance headers are
// preserved, so an operator-initiated maintenance call that legitimately runs
// for minutes does not have to choose between timing out and losing its
// attribution. The receiver is unchanged.
func (c *APIClient) WithTimeout(timeout time.Duration) *APIClient {
	if c == nil || timeout <= 0 {
		return c
	}
	clone := *c
	clone.client = c.client.CloneWithTimeout(timeout)
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
