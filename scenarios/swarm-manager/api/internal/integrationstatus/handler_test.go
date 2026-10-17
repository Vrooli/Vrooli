package integrationstatus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerProjectsProviderStatuses(t *testing.T) {
	provider := New(map[string]Checker{"agent-manager": fakeChecker{status: Status{Configured: true, Availability: Available, DegradedBehavior: "starts are blocked"}}})
	handler := NewHandler(provider)
	rec := httptest.NewRecorder()
	handler.List(rec, httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"id":"agent-manager"`) {
		t.Fatalf("response status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRejectsMissingProvider(t *testing.T) {
	handler := NewHandler(nil)
	rec := httptest.NewRecorder()
	handler.List(rec, httptest.NewRequest(http.MethodGet, "/api/v1/integrations", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}
