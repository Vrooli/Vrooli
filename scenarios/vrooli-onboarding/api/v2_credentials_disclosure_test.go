package main

import (
	"context"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"net/http"
	"strings"
	"testing"
)

func TestCredentialProvisionDoesNotDiscloseValue(t *testing.T) {
	const secret = "credential-value-must-not-leak"
	previous := credentialProvisionCommand
	credentialProvisionCommand = func(_ context.Context, logicalID, field, value string) (credentialclient.ProvisionResponse, error) {
		if logicalID != "vrooli/demo" || field != "api-key" || value != secret {
			t.Fatalf("provision command received %q/%q/%q", logicalID, field, value)
		}
		return credentialclient.ProvisionResponse{Version: "opaque-version"}, nil
	}
	t.Cleanup(func() { credentialProvisionCommand = previous })

	response := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/ProvisionCredential", `{"target":"local","logicalId":"vrooli/demo","field":"api-key","value":"`+secret+`"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), secret) {
		t.Fatalf("credential value disclosed in response or request URL: %s", response.Body.String())
	}
}
