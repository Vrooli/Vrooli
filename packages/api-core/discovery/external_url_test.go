package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBrowserURLForHostMatchesAPIBase pins this resolver to the same answers
// packages/api-base/src/server/embedded.ts gives. The cases are transcribed
// from that package's own suite so a divergence fails here rather than showing
// up as a link that works in one scenario and 404s in another.
func TestBrowserURLForHostMatchesAPIBase(t *testing.T) {
	cases := []struct {
		name     string
		host     string
		scenario string
		port     int
		want     string
	}{
		{"localhost with port", "localhost:3000", "my-scenario", 9000, "http://localhost:9000"},
		{"loopback address", "127.0.0.1:3000", "my-scenario", 9000, "http://localhost:9000"},
		{"unspecified address", "0.0.0.0:3000", "my-scenario", 9000, "http://localhost:9000"},
		{"missing host", "", "my-scenario", 9000, "http://localhost:9000"},
		{"tunnel swaps first label", "git-control-tower.example.com", "my-scenario", 9000, "https://my-scenario.example.com"},
		{"deep domain swaps first label", "app.staging.example.com", "test-app", 9000, "https://test-app.staging.example.com"},
		{"two-label domain falls back", "example.com", "my-scenario", 9000, "http://localhost:9000"},
		{"single word host falls back", "myhost", "my-scenario", 9000, "http://localhost:9000"},
		{"port stripped before analysis", "app.example.com:8080", "my-scenario", 9000, "https://my-scenario.example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BrowserURLForHost(tc.host, tc.scenario, tc.port); got != tc.want {
				t.Fatalf("BrowserURLForHost(%q, %q, %d) = %q, want %q", tc.host, tc.scenario, tc.port, got, tc.want)
			}
		})
	}
}

// TestBrowserURLForHostHandlesIPv6 covers a shape the Express original never
// meets, because Go's own server hands IPv6 hosts back in bracket form.
func TestBrowserURLForHostHandlesIPv6(t *testing.T) {
	if got := BrowserURLForHost("[::1]:3000", "my-scenario", 9000); got != "http://localhost:9000" {
		t.Fatalf("bracketed IPv6 loopback = %q, want the localhost URL", got)
	}
}

// TestBrowserURLForHostWithoutScenarioStaysLocal keeps an empty slug from
// producing "https://.example.com".
func TestBrowserURLForHostWithoutScenarioStaysLocal(t *testing.T) {
	if got := BrowserURLForHost("app.example.com", "", 9000); got != "http://localhost:9000" {
		t.Fatalf("empty scenario slug = %q, want the localhost URL", got)
	}
}

func TestExternalHostPrefersTheForwardedValue(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:16382/", nil)
	request.Header.Set("X-Forwarded-Host", "web-console.example.com")
	if got := ExternalHost(request); got != "web-console.example.com" {
		t.Fatalf("ExternalHost = %q, want the forwarded host", got)
	}
}

func TestExternalHostReadsTheFirstForwardedHop(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:16382/", nil)
	request.Header.Set("X-Forwarded-Host", "web-console.example.com, inner.proxy")
	if got := ExternalHost(request); got != "web-console.example.com" {
		t.Fatalf("ExternalHost = %q, want the first forwarded hop", got)
	}
}

func TestExternalHostFallsBackToRequestHost(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://web-console.example.com/", nil)
	if got := ExternalHost(request); got != "web-console.example.com" {
		t.Fatalf("ExternalHost = %q, want the request host", got)
	}
}

func TestExternalHostMiddlewareCarriesTheHostToTheHandler(t *testing.T) {
	var seen string
	handler := ExternalHostMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = ExternalHostFromContext(r.Context())
	}))
	request := httptest.NewRequest(http.MethodGet, "http://app.staging.example.com/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if seen != "app.staging.example.com" {
		t.Fatalf("context host = %q, want the request host", seen)
	}
}

func TestExternalHostFromContextIsEmptyWithoutTheMiddleware(t *testing.T) {
	if got := ExternalHostFromContext(context.Background()); got != "" {
		t.Fatalf("bare context host = %q, want empty", got)
	}
}

// TestResolveExternalURLUsesTheUIPort is the regression this whole seam exists
// for: a caller that asks for a browser link must not receive the API port.
func TestResolveExternalURLUsesTheUIPort(t *testing.T) {
	var askedFor string
	resolver := NewResolver(ResolverConfig{
		CommandRunner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			askedFor = args[len(args)-1]
			return []byte("22054\n"), nil
		},
	})
	url, err := resolver.ResolveExternalURL(context.Background(), "vrooli-bridge", "localhost:16382")
	if err != nil {
		t.Fatalf("ResolveExternalURL: %v", err)
	}
	if askedFor != UIPortKey {
		t.Fatalf("resolved port key %q, want %q", askedFor, UIPortKey)
	}
	if url != "http://localhost:22054" {
		t.Fatalf("ResolveExternalURL = %q, want the UI port", url)
	}
}

func TestResolveExternalURLReportsAnUnresolvableScenario(t *testing.T) {
	resolver := NewResolver(ResolverConfig{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, context.DeadlineExceeded
		},
	})
	if _, err := resolver.ResolveExternalURL(context.Background(), "vrooli-bridge", ""); err == nil {
		t.Fatal("expected an error when the scenario port cannot be resolved")
	}
}
