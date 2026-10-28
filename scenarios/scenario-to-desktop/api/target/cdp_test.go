package target

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoverElectronRendererSelectsExactPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/list" {
			t.Fatalf("path = %q, want /json/list", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
            {"id":"splash","type":"page","title":"Splash","url":"file:///splash","webSocketDebuggerUrl":"ws://127.0.0.1/devtools/splash"},
            {"id":"main","type":"page","title":"Example","url":"http://127.0.0.1","webSocketDebuggerUrl":"ws://127.0.0.1/devtools/main"}
        ]`))
	}))
	defer server.Close()

	page, err := DiscoverElectronRenderer(context.Background(), server.URL, "main")
	if err != nil {
		t.Fatal(err)
	}
	if page.ID != "main" || page.Title != "Example" {
		t.Fatalf("page = %#v", page)
	}
}

func TestDiscoverElectronRendererFailsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"main","type":"background_page"}]`))
	}))
	defer server.Close()

	_, err := DiscoverElectronRenderer(context.Background(), server.URL, "main")
	if err == nil || !strings.Contains(err.Error(), "attachable") {
		t.Fatalf("error = %v, want attachability failure", err)
	}
	_, err = DiscoverElectronRenderer(context.Background(), server.URL, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v, want missing renderer failure", err)
	}
}

func TestSelectElectronRendererRejectsAmbiguousPages(t *testing.T) {
	_, err := SelectElectronRenderer([]CDPPage{
		{ID: "one", Type: "page", WebSocketDebuggerURL: "ws://one"},
		{ID: "two", Type: "page", WebSocketDebuggerURL: "ws://two"},
	}, RendererExpectation{})
	if err == nil || !strings.Contains(err.Error(), "identity is required") {
		t.Fatalf("error = %v, want missing identity refusal", err)
	}
}

func TestSelectElectronRendererRequiresIdentity(t *testing.T) {
	_, err := SelectElectronRenderer([]CDPPage{
		{ID: "splash", Type: "page", URL: "file:///splash.html", WebSocketDebuggerURL: "ws://splash"},
	}, RendererExpectation{})
	if err == nil || !strings.Contains(err.Error(), "identity is required") {
		t.Fatalf("error = %v, want missing identity refusal", err)
	}
}

func TestSelectElectronRendererUsesStableURLIdentity(t *testing.T) {
	page, err := SelectElectronRenderer([]CDPPage{
		{ID: "splash", Type: "page", URL: "file:///splash", WebSocketDebuggerURL: "ws://splash"},
		{ID: "main", Type: "page", URL: "http://127.0.0.1:4400", WebSocketDebuggerURL: "ws://main"},
	}, RendererExpectation{URLPrefix: "http://127.0.0.1:4400"})
	if err != nil || page.ID != "main" {
		t.Fatalf("page = %#v, err = %v", page, err)
	}
}
