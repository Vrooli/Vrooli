package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONStrictRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok","unknown":"bad"}`))
	var payload struct {
		Name string `json:"name"`
	}

	if err := DecodeJSONStrict(req, &payload); err == nil {
		t.Fatal("expected DecodeJSONStrict to reject unknown fields")
	}
}

func TestDecodeJSONStrictRejectsTrailingContent(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"ok"}{"name":"extra"}`))
	var payload struct {
		Name string `json:"name"`
	}

	if err := DecodeJSONStrict(req, &payload); err == nil {
		t.Fatal("expected DecodeJSONStrict to reject trailing JSON")
	}
}
