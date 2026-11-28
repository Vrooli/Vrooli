package apihttp

import "net/http"

// Doer is the narrow outbound HTTP contract for callers that only need to
// issue requests. It keeps consumers independent of the full http.Client API.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

var _ Doer = (*http.Client)(nil)
