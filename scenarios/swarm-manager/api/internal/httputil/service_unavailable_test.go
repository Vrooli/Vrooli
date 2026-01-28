package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServiceUnavailable(t *testing.T) {
	w := httptest.NewRecorder()
	ServiceUnavailable(w, "[test]", "maintenance")

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	body := strings.TrimSpace(w.Body.String())
	if body != "maintenance" {
		t.Fatalf("expected body %q, got %q", "maintenance", body)
	}
}
