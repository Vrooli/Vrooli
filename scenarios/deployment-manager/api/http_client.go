package main

import (
	"context"
	"net/http"
)

// HTTPClient defines the interface for making HTTP requests.
// This seam allows tests to substitute real HTTP calls with mocks.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// httpClientKey is the context key for injecting an HTTP client.
type httpClientKey struct{}

// WithHTTPClient returns a new context with the given HTTP client.
func WithHTTPClient(ctx context.Context, client HTTPClient) context.Context {
	return context.WithValue(ctx, httpClientKey{}, client)
}

// GetHTTPClient retrieves the HTTP client from context, falling back to http.DefaultClient.
func GetHTTPClient(ctx context.Context) HTTPClient {
	if client, ok := ctx.Value(httpClientKey{}).(HTTPClient); ok && client != nil {
		return client
	}
	return http.DefaultClient
}
