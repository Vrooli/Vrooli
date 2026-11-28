package ai

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrivacyConsentProxyFailsClosedWhenGatewayIsUnavailable(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ai/privacy-consent", nil)
	privacyConsentProxy().ServeHTTP(response, request)
	if response.Code < http.StatusBadRequest {
		t.Fatalf("status = %d, want authority error", response.Code)
	}
}
