package engine

import "net/http"

// HTTPDoer is an interface for making HTTP requests.
// This abstraction enables testing the PlaywrightEngine without a real HTTP server.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Compile-time interface enforcement - http.Client satisfies HTTPDoer
var _ HTTPDoer = (*http.Client)(nil)
