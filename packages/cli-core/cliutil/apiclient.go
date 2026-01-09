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
