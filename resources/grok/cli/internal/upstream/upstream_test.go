package upstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchLatestReturnsTrimmedPointer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stable" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte("0.2.72\n"))
	}))
	defer srv.Close()

	got, err := FetchLatest(context.Background(), srv.Client(), []string{srv.URL}, "stable")
	if err != nil {
		t.Fatalf("FetchLatest() error = %v", err)
	}
	if got != "0.2.72" {
		t.Fatalf("FetchLatest() = %q, want %q", got, "0.2.72")
	}
}

func TestFetchLatestFallsBackToSecondBase(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer down.Close()
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0.3.0"))
	}))
	defer up.Close()

	got, err := FetchLatest(context.Background(), up.Client(), []string{down.URL, up.URL}, "stable")
	if err != nil {
		t.Fatalf("FetchLatest() error = %v", err)
	}
	if got != "0.3.0" {
		t.Fatalf("FetchLatest() = %q, want %q", got, "0.3.0")
	}
}

func TestFetchLatestEmptyChannelDefaultsToStable(t *testing.T) {
	var requested string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = r.URL.Path
		_, _ = w.Write([]byte("1.0.0"))
	}))
	defer srv.Close()

	if _, err := FetchLatest(context.Background(), srv.Client(), []string{srv.URL}, "  "); err != nil {
		t.Fatalf("FetchLatest() error = %v", err)
	}
	if requested != "/stable" {
		t.Fatalf("requested path = %q, want %q", requested, "/stable")
	}
}

func TestFetchLatestAllSourcesFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := FetchLatest(context.Background(), srv.Client(), []string{srv.URL}, "stable"); err == nil {
		t.Fatal("FetchLatest() expected error when all sources fail")
	}
}

func TestHandlersWireGrokSource(t *testing.T) {
	h := Handlers("grok", "0.2.72")
	if h == nil {
		t.Fatal("Handlers() returned nil")
	}
	if h.Cfg.SourceKind != SourceKind {
		t.Fatalf("SourceKind = %q, want %q", h.Cfg.SourceKind, SourceKind)
	}
	if h.Cfg.SourceID != DefaultChannel {
		t.Fatalf("SourceID = %q, want %q", h.Cfg.SourceID, DefaultChannel)
	}
	if h.LatestFetcher == nil {
		t.Fatal("Handlers() left LatestFetcher nil")
	}
}
