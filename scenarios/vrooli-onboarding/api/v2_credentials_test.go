package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestV2CredentialProvisionUsesMetadataOnlyResponse(t *testing.T) {
	prior := credentialProvisionCommand
	var received string
	credentialProvisionCommand = func(_ context.Context, logicalID, field, value string) error {
		received = logicalID + "/" + field + "/" + value
		return nil
	}
	t.Cleanup(func() { credentialProvisionCommand = prior })

	w := doPost(t, NewServer(), "/api/v2/credentials/provision", `{"logical_id":"vrooli/demo","field":"api-key","value":"test-value"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if received != "vrooli/demo/api-key/test-value" {
		t.Fatalf("provision input = %q", received)
	}
	if strings.Contains(w.Body.String(), "test-value") {
		t.Fatalf("credential value leaked in response: %s", w.Body.String())
	}
}

func TestV2CredentialProvisionRejectsMissingValue(t *testing.T) {
	w := doPost(t, NewServer(), "/api/v2/credentials/provision", `{"logical_id":"vrooli/demo","field":"api-key"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}
