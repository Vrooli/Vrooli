package apihttp

import (
	"net/http"

	"github.com/vrooli/api-core/database"
)

// TestModeTransport propagates the request-scoped test-mode marker to an
// outbound scenario call. It is opt-in so ordinary clients retain their
// existing transport and request bytes.
type TestModeTransport struct {
	Base http.RoundTripper
}

func (t TestModeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if request == nil || !database.IsTestMode(request.Context()) {
		return base.RoundTrip(request)
	}

	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	clone.Header.Set(TestModeHeader, TestModeValue)
	return base.RoundTrip(clone)
}

// NewTestModeClient returns an HTTP client that carries test-mode context to
// downstream scenario requests while leaving non-test-mode requests alone.
func NewTestModeClient(base *http.Client) *http.Client {
	if base == nil {
		base = http.DefaultClient
	}
	clone := *base
	transport := base.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clone.Transport = TestModeTransport{Base: transport}
	return &clone
}
