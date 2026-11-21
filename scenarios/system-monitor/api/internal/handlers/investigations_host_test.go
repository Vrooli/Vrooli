package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
)

func TestResolveAPIBaseURLUsesFirstForwardedProxyHop(t *testing.T) {
	handler := &InvestigationHandler{config: &config.Config{}}
	request := httptest.NewRequest("GET", "http://127.0.0.1:8080/", nil)
	request.Header.Set("X-Forwarded-Host", "web-console.example.com, inner.proxy")
	request.Header.Set("X-Forwarded-Proto", "https, http")
	if got := handler.resolveAPIBaseURL(request); got != "https://web-console.example.com" {
		t.Fatalf("API base URL = %q, want first forwarded browser origin", got)
	}
}
