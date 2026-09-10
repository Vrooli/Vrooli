package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestCredentialProvisionUsesMetadataOnlyResponse(t *testing.T) {
	prior := credentialProvisionCommand
	var received string
	credentialProvisionCommand = func(_ context.Context, logicalID, field, value string) error {
		received = logicalID + "/" + field + "/" + value
		return nil
	}
	t.Cleanup(func() { credentialProvisionCommand = prior })

	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/ProvisionCredential", `{"target":"local","logicalId":"vrooli/demo","field":"api-key","value":"test-value"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if received != "vrooli/demo/api-key/test-value" {
		t.Fatalf("provision input = %q", received)
	}
	if strings.Contains(w.Body.String(), "test-value") {
		t.Fatalf("credential value leaked in response: %s", w.Body.String())
	}
}

func TestCredentialProvisionRejectsMissingValue(t *testing.T) {
	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/ProvisionCredential", `{"target":"local","logicalId":"vrooli/demo","field":"api-key"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}

func TestCredentialDoctorRelaysMetadataOnly(t *testing.T) {
	prior := credentialDoctorCommand
	credentialDoctorCommand = func(context.Context) ([]byte, error) {
		return []byte(`{"provider":{"backend":"libsecret","condition":"available"},"credentials":[{"logical_id":"vrooli/demo","field":"api-key","label":"Demo key","description":"Key for the demo provider.","obtain_url":"https://example.test/demo-key","provisioning":"operator"}]}`), nil
	}
	t.Cleanup(func() { credentialDoctorCommand = prior })

	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/DiagnoseCredentials", `{"target":"local"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"backend":"libsecret"`) || !strings.Contains(w.Body.String(), `"obtainUrl":"https://example.test/demo-key"`) || !strings.Contains(w.Body.String(), `"provisioning":"operator"`) {
		t.Fatalf("status/body = %d/%s", w.Code, w.Body.String())
	}
}

func TestCredentialDoctorHidesRelayFailureDetails(t *testing.T) {
	prior := credentialDoctorCommand
	credentialDoctorCommand = func(context.Context) ([]byte, error) {
		return nil, errors.New("private host detail")
	}
	t.Cleanup(func() { credentialDoctorCommand = prior })

	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/DiagnoseCredentials", `{"target":"local"}`)
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "private host detail") {
		t.Fatalf("status/body = %d/%s", w.Code, w.Body.String())
	}
}
