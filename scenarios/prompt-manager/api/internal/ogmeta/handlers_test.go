package ogmeta

import "testing"

func TestNewHandlersInitializesClientAndCache(t *testing.T) {
	handlers := NewHandlers()
	if handlers == nil {
		t.Fatal("expected handlers")
	}
	if handlers.client == nil {
		t.Fatal("expected http client")
	}
	if handlers.cache == nil || handlers.cache.entries == nil {
		t.Fatal("expected initialized metadata cache")
	}
}

func TestExtractTitleFallsBackToPageTitle(t *testing.T) {
	html := `<html><head><title>Example Page</title></head></html>`
	if got := extractTitle(html); got != "Example Page" {
		t.Fatalf("expected title fallback, got %q", got)
	}
}
