package eventbus

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// InternalScenarioTransport propagates an opaque, already-authorized identity
// only to one discovered internal scenario origin. It is intentionally an
// opt-in client transport: it never mutates http.DefaultTransport or leaks a
// token when a caller follows an external redirect or supplies another URL.
type InternalScenarioTransport struct {
	Base           http.RoundTripper
	InternalOrigin *url.URL
	IdentityToken  func(*http.Request) string
}

func (t InternalScenarioTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if request == nil || t.InternalOrigin == nil || !sameEndpointOrigin(request.URL, t.InternalOrigin) || t.IdentityToken == nil {
		return base.RoundTrip(request)
	}
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	if token := strings.TrimSpace(t.IdentityToken(request)); token != "" {
		clone.Header.Set(cliutil.HeaderAgentIdentityToken, token)
	}
	return base.RoundTrip(clone)
}

func sameEndpointOrigin(target, origin *url.URL) bool {
	return target != nil && strings.EqualFold(target.Scheme, origin.Scheme) && strings.EqualFold(target.Host, origin.Host)
}

func NewInternalScenarioClient(origin string, token func(*http.Request) string) (*http.Client, error) {
	parsed, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, &url.Error{Op: "parse", URL: origin, Err: errInvalidInternalOrigin{}}
	}
	return &http.Client{Transport: InternalScenarioTransport{InternalOrigin: parsed, IdentityToken: token}}, nil
}

type errInvalidInternalOrigin struct{}

func (errInvalidInternalOrigin) Error() string { return "internal scenario origin must be absolute" }
