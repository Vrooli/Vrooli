package main

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/ai-provider-go/conformance"
	"github.com/vrooli/ai-provider-go/policy"
)

func TestProviderConfigIdentity(t *testing.T) {
	config := providerConfig()
	if config.Name != "abliteration" || config.APIKeyEnv != "ABLIT_KEY" {
		t.Fatalf("config = %+v", config)
	}
	if config.CredentialIdentity != "vrooli/abliteration" {
		t.Fatalf("credential identity = %q", config.CredentialIdentity)
	}
}

func TestShippedPolicyConforms(t *testing.T) {
	path := filepath.Join("..", "model-policy.json")
	if err := conformance.CheckPolicyFile(path, policy.CloudOptions()); err != nil {
		t.Fatalf("policy conformance: %v", err)
	}
}

func TestShippedPolicyContextAndPricing(t *testing.T) {
	catalog, err := policy.LoadFile(filepath.Join("..", "model-policy.json"), policy.CloudOptions())
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	cases := map[string]struct {
		context int
		input   float64
		output  float64
	}{
		"abliterated-model":             {context: 262144, input: 1.0, output: 3.0},
		"abliterated-model-large-flash": {context: 1048576, input: 1.0, output: 2.0},
		"abliterated-model-large":       {context: 1000000, input: 3.0, output: 5.0},
		"abliterated-model-large-v2":    {context: 1000000, input: 3.0, output: 5.0},
	}
	for name, want := range cases {
		model, ok := catalog.Models[name]
		if !ok {
			t.Fatalf("missing model %q", name)
		}
		if model.ContextWindowTokens != want.context {
			t.Errorf("%s context = %d, want %d", name, model.ContextWindowTokens, want.context)
		}
		if model.Pricing == nil {
			t.Errorf("%s pricing missing", name)
			continue
		}
		if model.Pricing.InputPerMTok != want.input || model.Pricing.OutputPerMTok != want.output {
			t.Errorf("%s pricing = %v/%v, want %v/%v", name, model.Pricing.InputPerMTok, model.Pricing.OutputPerMTok, want.input, want.output)
		}
	}
}
