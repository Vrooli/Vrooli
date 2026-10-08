package sttchain

import "testing"

func TestKyutaiProvider_DerivesResourceEndpoints(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		endpoint  string
		streamURL string
	}{
		{name: "http", baseURL: "http://kyutai:8092/", endpoint: "http://kyutai:8092/health", streamURL: "ws://kyutai:8092/v1/stream"},
		{name: "https", baseURL: "https://kyutai.example", endpoint: "https://kyutai.example/health", streamURL: "wss://kyutai.example/v1/stream"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := NewKyutaiProvider(tc.baseURL)
			if got := provider.endpoint("/health"); got != tc.endpoint {
				t.Fatalf("endpoint() = %q, want %q", got, tc.endpoint)
			}
			if got := provider.streamURL(); got != tc.streamURL {
				t.Fatalf("streamURL() = %q, want %q", got, tc.streamURL)
			}
		})
	}
}
