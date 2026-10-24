package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
)

func TestValidateTransportRejectsUnregisteredREST(t *testing.T) {
	if err := validateTransport([]module.EndpointDescriptor{{ID: "bad", Path: "/api/v1/bad"}}); err == nil {
		t.Fatal("expected an untagged REST endpoint to be rejected")
	}
}

func TestGenerateCheckRequiresAllModuleMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "endpoints.json")
	if err := os.WriteFile(path, []byte(`{"endpoints":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := generate(path, true); err == nil {
		t.Fatal("expected missing module endpoint metadata to fail")
	}
}
