package ensure

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestLoopbackLocalhostPinsResolvedLoopbackAddress(t *testing.T) {
	var requests, lookups atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer target.Close()
	port := target.Listener.Addr().(*net.TCPAddr).Port
	lookup := func(_ context.Context, network, host string) ([]net.IP, error) {
		lookups.Add(1)
		if network != "ip" || host != "localhost" {
			t.Fatalf("lookup(%q, %q), want ip localhost", network, host)
		}
		return []net.IP{net.ParseIP("::1"), net.ParseIP("127.0.0.1")}, nil
	}
	client, err := newLoopbackClient("http://localhost:"+strconv.Itoa(port), lookup)
	if err != nil {
		t.Fatalf("newLoopbackClient(localhost): %v", err)
	}
	parsed, err := url.Parse(client.BaseURL)
	if err != nil || parsed.Hostname() != "127.0.0.1" || parsed.Port() != strconv.Itoa(port) {
		t.Fatalf("pinned endpoint = %q, parse error %v", client.BaseURL, err)
	}
	if _, err := client.ListTags(context.Background()); err != nil {
		t.Fatalf("ListTags via pinned localhost: %v", err)
	}
	if lookups.Load() != 1 || requests.Load() != 1 {
		t.Fatalf("lookups=%d requests=%d, want one each", lookups.Load(), requests.Load())
	}
	if strings.Contains(client.BaseURL, "localhost") {
		t.Fatalf("endpoint was not pinned: %q", client.BaseURL)
	}
}

func TestLoopbackLocalhostRejectsNonLoopbackAndMixedResolution(t *testing.T) {
	for _, ips := range [][]net.IP{
		{net.ParseIP("192.0.2.10")},
		{net.ParseIP("127.0.0.1"), net.ParseIP("192.0.2.10")},
	} {
		if got, err := validateLoopbackBaseURLWithLookup("http://localhost:11434", func(context.Context, string, string) ([]net.IP, error) {
			return ips, nil
		}); err == nil || got != "" {
			t.Fatalf("localhost resolution %v accepted as %q, want refusal", ips, got)
		}
	}
}

func TestLoopbackGenerateDoesNotFollowRedirectOrUseProxy(t *testing.T) {
	var targetCalls, proxyCalls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls.Add(1); w.WriteHeader(http.StatusOK) }))
	defer target.Close()
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { proxyCalls.Add(1); w.WriteHeader(http.StatusBadGateway) }))
	defer proxy.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/target", http.StatusFound)
	}))
	defer redirect.Close()
	t.Setenv("OLLAMA_BASE_URL", redirect.URL)
	t.Setenv("OLLAMA_HOST", "192.0.2.10:11434")
	t.Setenv("HTTP_PROXY", proxy.URL)
	t.Setenv("http_proxy", proxy.URL)
	client, err := NewLoopbackClient()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Generate(context.Background(), GenerateRequest{Model: "fixture", Prompt: "test"}); err == nil {
		t.Fatal("redirect response should not be treated as a completed generation")
	}
	if targetCalls.Load() != 0 {
		t.Fatalf("redirect target received %d requests", targetCalls.Load())
	}
	if proxyCalls.Load() != 0 {
		t.Fatalf("configured proxy received %d requests", proxyCalls.Load())
	}
}

func TestLoopbackUsesEffectiveHostOverrideAndDefault(t *testing.T) {
	t.Run("host override is validated", func(t *testing.T) {
		t.Setenv("OLLAMA_BASE_URL", "")
		t.Setenv("OLLAMA_HOST", "192.0.2.44:11434")
		if _, err := NewLoopbackClient(); err == nil {
			t.Fatal("remote OLLAMA_HOST override must be refused")
		}
	})
	t.Run("loopback default remains usable", func(t *testing.T) {
		t.Setenv("OLLAMA_BASE_URL", "")
		t.Setenv("OLLAMA_HOST", "")
		t.Setenv("OLLAMA_PORT", "")
		client, err := NewLoopbackClient()
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(client.BaseURL)
		if err != nil || u.Hostname() != "127.0.0.1" {
			t.Fatalf("unexpected effective local endpoint: %v", err)
		}
		if _, ok := client.HTTP.Transport.(*http.Transport); !ok {
			t.Fatal("loopback client must use its direct transport")
		}
	})
}
