package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthBoundaryLoopbackDoesNotNeedToken(t *testing.T) {
	called := false
	handler := onboardingMutationAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v2/apply", strings.NewReader("{}"))
	request.RemoteAddr = "127.0.0.1:9876"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || !called {
		t.Fatalf("loopback request = %d, called=%v, body=%s", response.Code, called, response.Body.String())
	}
}

func TestAuthBoundaryRejectsRemoteWithoutToken(t *testing.T) {
	handler := onboardingMutationAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("remote request reached protected handler")
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v2/apply", strings.NewReader("{}"))
	request.RemoteAddr = "198.51.100.10:9876"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), "missing bearer token") {
		t.Fatalf("remote request = %d: %s", response.Code, response.Body.String())
	}
}

func TestAuthBoundaryAcceptsCredentialAuthorityToken(t *testing.T) {
	previous := onboardingExpectedToken
	onboardingExpectedToken = func(context.Context) (string, error) { return "authority-token", nil }
	t.Cleanup(func() { onboardingExpectedToken = previous })

	handler := onboardingMutationAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v2/apply", strings.NewReader("{}"))
	request.RemoteAddr = "203.0.113.10:9876"
	request.Header.Set("Authorization", "Bearer authority-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("authorized remote request = %d: %s", response.Code, response.Body.String())
	}
}

func TestAuthBoundaryNeverEchoesPresentedToken(t *testing.T) {
	const presented = "secret-that-must-not-appear"
	handler := onboardingMutationAuth(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("invalid token reached protected handler")
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v2/apply", strings.NewReader("{}"))
	request.RemoteAddr = "203.0.113.10:9876"
	request.Header.Set("Authorization", "Bearer "+presented)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || strings.Contains(response.Body.String(), presented) {
		t.Fatalf("token disclosure in auth response: %d %s", response.Code, response.Body.String())
	}
}
