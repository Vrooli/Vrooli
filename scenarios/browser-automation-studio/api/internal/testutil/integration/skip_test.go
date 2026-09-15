package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireEnvReturnsConfiguredValue(t *testing.T) {
	t.Setenv("BAS_TEST_INTEGRATION_ENV", " configured ")

	got := RequireEnv(t, "BAS_TEST_INTEGRATION_ENV", "unit test")

	if got != "configured" {
		t.Fatalf("expected trimmed environment value, got %q", got)
	}
}

func TestRequireAnyEnvReturnsFirstConfiguredValue(t *testing.T) {
	t.Setenv("BAS_TEST_INTEGRATION_SECOND", "second")

	got := RequireAnyEnv(t, []string{"BAS_TEST_INTEGRATION_FIRST", "BAS_TEST_INTEGRATION_SECOND"}, "unit test")

	if got != "second" {
		t.Fatalf("expected second environment value, got %q", got)
	}
}

func TestRequireCommandAcceptsAvailableExecutable(t *testing.T) {
	RequireCommand(t, "go", "unit test")
}

func TestRequireHTTPStatusOKAcceptsHealthyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	RequireHTTPStatusOK(t, server.Client(), server.URL, "unit test")
}
