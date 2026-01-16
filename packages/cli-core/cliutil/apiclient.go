package cliutil

import (
	"net/url"
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
