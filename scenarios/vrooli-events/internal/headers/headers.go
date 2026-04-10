package headers

import "net/http"

const (
	// HeaderSourceScenario is the header name for source scenario identification.
	HeaderSourceScenario = "X-Source-Scenario"
)

// InjectSource adds the X-Source-Scenario header to the request.
func InjectSource(req *http.Request, scenario string) {
	req.Header.Set(HeaderSourceScenario, scenario)
}

// ExtractSource reads the X-Source-Scenario header from the request.
func ExtractSource(req *http.Request) string {
	return req.Header.Get(HeaderSourceScenario)
}

// SourceTransport wraps an http.RoundTripper to inject X-Source-Scenario on all requests.
type SourceTransport struct {
	Inner    http.RoundTripper
	Scenario string
}

// RoundTrip implements http.RoundTripper, injecting the source scenario header.
func (t *SourceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	InjectSource(clone, t.Scenario)
	inner := t.Inner
	if inner == nil {
		inner = http.DefaultTransport
	}
	return inner.RoundTrip(clone)
}

// NewClient returns an *http.Client that injects X-Source-Scenario on every request.
func NewClient(scenario string) *http.Client {
	return &http.Client{
		Transport: &SourceTransport{Scenario: scenario},
	}
}
