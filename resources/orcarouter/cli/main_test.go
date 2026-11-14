package main

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/ai-provider-go/conformance"
	"github.com/vrooli/ai-provider-go/policy"
)

func TestProviderConfigIdentity(t *testing.T) {
	config := providerConfig()
	if config.Name != "orcarouter" || config.APIKeyEnv != "ORCAROUTER_API_KEY" {
		t.Fatalf("config = %+v", config)
	}
	if config.CredentialIdentity != "vrooli/orcarouter" {
		t.Fatalf("credential identity = %q", config.CredentialIdentity)
	}
}

func TestShippedPolicyConforms(t *testing.T) {
	path := filepath.Join("..", "model-policy.json")
	if err := conformance.CheckPolicyFile(path, policy.CloudOptions()); err != nil {
		t.Fatalf("policy conformance: %v", err)
	}
}
