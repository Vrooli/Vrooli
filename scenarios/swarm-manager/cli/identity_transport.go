package main

import "net/http"

const headerAgentIdentityToken = "X-Agent-Identity-Token"

// identityTransport wraps an http.RoundTripper to inject the agent identity
// token header on every outgoing request when a token is present.
type identityTransport struct {
	base  http.RoundTripper
	token string
}

func (t *identityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req = req.Clone(req.Context())
		req.Header.Set(headerAgentIdentityToken, t.token)
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
