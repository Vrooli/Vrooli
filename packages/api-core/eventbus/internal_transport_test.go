package eventbus

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

type recordingTransport struct{ request *http.Request }

func (t *recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.request = r
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
}

func TestInternalScenarioTransportDoesNotLeakIdentity(t *testing.T) {
	origin, _ := url.Parse("http://plan-manager.internal:19834")
	base := &recordingTransport{}
	transport := InternalScenarioTransport{Base: base, InternalOrigin: origin, IdentityToken: func(*http.Request) string { return "verified-token" }}
	for _, tc := range []struct {
		url  string
		want string
	}{{"http://plan-manager.internal:19834/api", "verified-token"}, {"https://example.com/api", ""}} {
		req, _ := http.NewRequest(http.MethodGet, tc.url, nil)
		if _, err := transport.RoundTrip(req); err != nil {
			t.Fatal(err)
		}
		if got := base.request.Header.Get(cliutil.HeaderAgentIdentityToken); got != tc.want {
			t.Fatalf("%s token=%q want=%q", tc.url, got, tc.want)
		}
	}
}
