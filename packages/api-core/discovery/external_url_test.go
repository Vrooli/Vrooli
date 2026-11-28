package discovery

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed testdata/external-url-cases.json
var externalURLCasesJSON []byte

// TestBrowserURLForHostMatchesAPIBase uses the same conformance fixture as
// packages/api-base so cross-language behavior cannot drift through copied cases.
func TestBrowserURLForHostMatchesAPIBase(t *testing.T) {
	cases := []struct {
		Name          string `json:"name"`
		Host          string `json:"host"`
		ForwardedHost string `json:"forwardedHost"`
		Scenario      string `json:"scenario"`
		Port          int    `json:"port"`
		Want          string `json:"want"`
	}{}
	if err := json.Unmarshal(externalURLCasesJSON, &cases); err != nil {
		t.Fatalf("decode cross-language URL fixture: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://fallback.invalid/", nil)
			request.Host = tc.Host
			if tc.ForwardedHost != "" {
				request.Header.Set("X-Forwarded-Host", tc.ForwardedHost)
			}
			host := ExternalHost(request)
			if got := BrowserURLForHost(host, tc.Scenario, tc.Port); got != tc.Want {
				t.Fatalf("BrowserURLForHost(%q, %q, %d) = %q, want %q", host, tc.Scenario, tc.Port, got, tc.Want)
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

func TestExternalHostUsesFirstForwardedHeaderValue(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:16382/", nil)
	request.Header.Add("X-Forwarded-Host", "first.example.com")
	request.Header.Add("X-Forwarded-Host", "second.example.com")
	if got := ExternalHost(request); got != "first.example.com" {
		t.Fatalf("ExternalHost = %q, want the first forwarded header value", got)
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

func TestBrowserURLPreservesPrivateLANHost(t *testing.T) {
	for host, want := range map[string]string{
		"192.168.1.173:16000": "http://192.168.1.173:22054",
		"[fd00::173]:16000":   "http://[fd00::173]:22054",
		"localhost:16000":     "http://localhost:22054",
		"127.0.0.1:16000":     "http://localhost:22054",
	} {
		if got := BrowserURLForHost(host, "nutrition-planner", 22054); got != want {
			t.Errorf("host %s: got %s, want %s", host, got, want)
		}
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
