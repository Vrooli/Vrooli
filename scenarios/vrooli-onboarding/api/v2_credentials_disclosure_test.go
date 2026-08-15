package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCredentialProvisionDoesNotDiscloseValue(t *testing.T) {
	const secret = "credential-value-must-not-leak"
	previous := credentialProvisionCommand
	credentialProvisionCommand = func(_ context.Context, logicalID, field, value string) error {
		if logicalID != "vrooli/demo" || field != "api-key" || value != secret {
			t.Fatalf("provision command received %q/%q/%q", logicalID, field, value)
		}
		return nil
	}
	t.Cleanup(func() { credentialProvisionCommand = previous })

	request := httptest.NewRequest(http.MethodPost, "/api/v2/credentials/provision", strings.NewReader(`{"logical_id":"vrooli/demo","field":"api-key","value":"`+secret+`"}`))
	request.RemoteAddr = "127.0.0.1:12345"
	response := httptest.NewRecorder()
	NewServer().Handler().ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), secret) || strings.Contains(request.URL.String(), secret) {
		t.Fatalf("credential value disclosed in response or request URL: %s", response.Body.String())
	}
}
