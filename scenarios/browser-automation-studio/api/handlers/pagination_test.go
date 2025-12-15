//go:build legacydb
// +build legacydb

package handlers

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParsePaginationParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/?limit=250&offset=10", nil)
	limit, offset := parsePaginationParams(req, 50, 200)
	if limit != 200 {
		t.Fatalf("expected limit to be clamped to 200, got %d", limit)
	}
	if offset != 10 {
		t.Fatalf("expected offset 10, got %d", offset)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	limit, offset = parsePaginationParams(req2, 25, 100)
	if limit != 25 || offset != 0 {
		t.Fatalf("expected defaults (25,0), got (%d,%d)", limit, offset)
	}

	req3 := httptest.NewRequest("GET", "/", nil)
	req3.URL.RawQuery = url.Values{"limit": {"-5"}, "offset": {"-1"}}.Encode()
	limit, offset = parsePaginationParams(req3, 10, 100)
	if limit != 10 || offset != 0 {
		t.Fatalf("expected negative values to fall back to defaults, got (%d,%d)", limit, offset)
	}
}
