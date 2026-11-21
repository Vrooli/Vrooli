package vision

import "testing"

func TestClientSourceFromHeader(t *testing.T) {
	tests := []struct {
		header string
		want   ClientSource
	}{
		{"cli", ClientSourceCLI},
		{"ui", ClientSourceUI},
		{"api", ClientSourceAPI},
		{"", ClientSourceAPI},        // Default for empty
		{"unknown", ClientSourceAPI}, // Default for unknown
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := ClientSourceFromHeader(tt.header)
			if got != tt.want {
				t.Errorf("ClientSourceFromHeader(%q) = %v, want %v", tt.header, got, tt.want)
			}
		})
	}
}

func TestClientSourcePolicy_IsAllowed(t *testing.T) {
	tests := []struct {
		name   string
		policy ClientSourcePolicy
		source ClientSource
		want   bool
	}{
		{
			name:   "empty allowed sources allows all - CLI",
			policy: ClientSourcePolicy{AllowedSources: nil},
			source: ClientSourceCLI,
			want:   true,
		},
		{
			name:   "empty allowed sources allows all - UI",
			policy: ClientSourcePolicy{AllowedSources: nil},
			source: ClientSourceUI,
			want:   true,
		},
		{
			name:   "empty allowed sources allows all - API",
			policy: ClientSourcePolicy{AllowedSources: nil},
			source: ClientSourceAPI,
			want:   true,
		},
		{
			name:   "CLI only - allows CLI",
			policy: ClientSourcePolicy{AllowedSources: []ClientSource{ClientSourceCLI}},
			source: ClientSourceCLI,
			want:   true,
		},
		{
			name:   "CLI only - denies UI",
			policy: ClientSourcePolicy{AllowedSources: []ClientSource{ClientSourceCLI}},
			source: ClientSourceUI,
			want:   false,
		},
		{
			name:   "CLI only - denies API",
			policy: ClientSourcePolicy{AllowedSources: []ClientSource{ClientSourceCLI}},
			source: ClientSourceAPI,
			want:   false,
		},
		{
			name:   "CLI and UI - allows CLI",
			policy: ClientSourcePolicy{AllowedSources: []ClientSource{ClientSourceCLI, ClientSourceUI}},
			source: ClientSourceCLI,
			want:   true,
		},
		{
			name:   "CLI and UI - allows UI",
			policy: ClientSourcePolicy{AllowedSources: []ClientSource{ClientSourceCLI, ClientSourceUI}},
			source: ClientSourceUI,
			want:   true,
		},
		{
			name:   "CLI and UI - denies API",
			policy: ClientSourcePolicy{AllowedSources: []ClientSource{ClientSourceCLI, ClientSourceUI}},
			source: ClientSourceAPI,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.IsAllowed(tt.source)
			if got != tt.want {
				t.Errorf("IsAllowed(%v) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestAllSourcesPolicy(t *testing.T) {
	policy := AllSourcesPolicy()

	if policy.AllowedSources != nil {
		t.Errorf("AllSourcesPolicy().AllowedSources = %v, want nil", policy.AllowedSources)
	}

	// Should allow all sources
	for _, source := range []ClientSource{ClientSourceCLI, ClientSourceUI, ClientSourceAPI} {
		if !policy.IsAllowed(source) {
			t.Errorf("AllSourcesPolicy().IsAllowed(%v) = false, want true", source)
		}
	}
}

func TestCLIOnlyPolicy(t *testing.T) {
	policy := CLIOnlyPolicy()

	if len(policy.AllowedSources) != 1 {
		t.Errorf("CLIOnlyPolicy().AllowedSources length = %d, want 1", len(policy.AllowedSources))
	}

	if !policy.IsAllowed(ClientSourceCLI) {
		t.Error("CLIOnlyPolicy().IsAllowed(CLI) = false, want true")
	}

	if policy.IsAllowed(ClientSourceUI) {
		t.Error("CLIOnlyPolicy().IsAllowed(UI) = true, want false")
	}

	if policy.IsAllowed(ClientSourceAPI) {
		t.Error("CLIOnlyPolicy().IsAllowed(API) = true, want false")
	}
}
