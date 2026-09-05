package upstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestParsesManifestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manifests/linux_amd64.json" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"version":"1.0.13","url":"https://example/cli.tar.gz","sha512":"deadbeef"}`))
	}))
	defer srv.Close()

	got, err := FetchLatest(context.Background(), srv.Client(), srv.URL, "linux_amd64")
	if err != nil {
		t.Fatalf("FetchLatest() error = %v", err)
	}
	if got != "1.0.13" {
		t.Fatalf("FetchLatest() = %q, want %q", got, "1.0.13")
	}
}

func TestFetchLatestEmptyPlatformDefaultsToHost(t *testing.T) {
	if HostPlatform() == "" {
		t.Skip("no Antigravity build for this host platform; default-to-host path not exercised")
	}
	var requested string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = r.URL.Path
		_, _ = w.Write([]byte(`{"version":"2.0.0"}`))
	}))
	defer srv.Close()

	if _, err := FetchLatest(context.Background(), srv.Client(), srv.URL, "  "); err != nil {
		t.Fatalf("FetchLatest() error = %v", err)
	}
	if want := "/manifests/" + HostPlatform() + ".json"; requested != want {
		t.Fatalf("requested path = %q, want %q", requested, want)
	}
}

func TestFetchLatestErrorsOnHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := FetchLatest(context.Background(), srv.Client(), srv.URL, "linux_amd64"); err == nil {
		t.Fatal("FetchLatest() expected error on HTTP failure")
	}
}

func TestFetchLatestErrorsOnMissingVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"url":"https://example/cli.tar.gz"}`))
	}))
	defer srv.Close()

	if _, err := FetchLatest(context.Background(), srv.Client(), srv.URL, "linux_amd64"); err == nil {
		t.Fatal("FetchLatest() expected error when manifest has no version")
	}
}

func TestHostPlatformShape(t *testing.T) {
	got := HostPlatform()
	if got == "" {
		return // unsupported host arch — acceptable
	}
	switch got {
	case "linux_amd64", "linux_arm64", "darwin_amd64", "darwin_arm64", "windows_amd64", "windows_arm64":
	default:
		t.Fatalf("HostPlatform() = %q, not a known Antigravity platform key", got)
	}
}

func TestHandlersWireManifestSource(t *testing.T) {
	h := Handlers("antigravity", "1.0.13")
	if h == nil {
		t.Fatal("Handlers() returned nil")
	}
	if h.Cfg.SourceKind != SourceKind {
		t.Fatalf("SourceKind = %q, want %q", h.Cfg.SourceKind, SourceKind)
	}
	if h.LatestFetcher == nil {
		t.Fatal("Handlers() left LatestFetcher nil")
	}
}
