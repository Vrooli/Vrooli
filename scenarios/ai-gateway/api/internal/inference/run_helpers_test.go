package inference

import (
	"testing"

	"ai-gateway/internal/providers"
)

func TestSourceAllowsCandidate(t *testing.T) {
	tests := []struct {
		name, source, provider string
		allowed                bool
		wantErr                bool
	}{
		{name: "unspecified accepts any provider", provider: providers.ProviderMetered, allowed: true},
		{name: "local accepts ollama", source: "local", provider: providers.ProviderOllama, allowed: true},
		{name: "local excludes hosted", source: "local", provider: providers.ProviderMetered},
		{name: "own key excludes metered", source: "own-key", provider: providers.ProviderMetered},
		{name: "credits accepts metered", source: "vrooli-credits", provider: providers.ProviderMetered, allowed: true},
		{name: "unknown source fails", source: "unknown", provider: providers.ProviderOllama, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := sourceAllowsCandidate(test.source, test.provider)
			if got != test.allowed {
				t.Fatalf("allowed = %v, want %v", got, test.allowed)
			}
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
