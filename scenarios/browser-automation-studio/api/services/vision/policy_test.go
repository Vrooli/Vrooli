package vision

import "testing"

func TestCreditPolicy_ShouldChargeCredits(t *testing.T) {
	tests := []struct {
		name                  string
		policy                CreditPolicy
		provenance            CredentialProvenance
		hasResourceOpenrouter bool
		isLocalExecution      bool
		want                  bool
	}{
		{
			name:   "no credits required returns false",
			policy: CreditPolicy{RequiresCredits: false},
			want:   false,
		},
		{
			name:   "credits required with no bypass returns true",
			policy: CreditPolicy{RequiresCredits: true},
			want:   true,
		},
		{
			name: "credential provenance bypass",
			policy: CreditPolicy{
				RequiresCredits:  true,
				BypassConditions: []BypassCondition{BypassCredentialProvenance},
			},
			provenance: CredentialProvenanceAuthority,
			want:       false,
		},
		{
			name: "credential provenance absent",
			policy: CreditPolicy{
				RequiresCredits:  true,
				BypassConditions: []BypassCondition{BypassCredentialProvenance},
			},
			provenance: CredentialProvenanceNone,
			want:       true,
		},
		{
			name: "resource_openrouter bypass when true",
			policy: CreditPolicy{
				RequiresCredits:  true,
				BypassConditions: []BypassCondition{BypassResourceOpenrouter},
			},
			hasResourceOpenrouter: true,
			want:                  false,
		},
		{
			name: "resource_openrouter bypass when false",
			policy: CreditPolicy{
				RequiresCredits:  true,
				BypassConditions: []BypassCondition{BypassResourceOpenrouter},
			},
			hasResourceOpenrouter: false,
			want:                  true,
		},
		{
			name: "local_execution bypass when true",
			policy: CreditPolicy{
				RequiresCredits:  true,
				BypassConditions: []BypassCondition{BypassLocalExecution},
			},
			isLocalExecution: true,
			want:             false,
		},
		{
			name: "multiple bypass conditions - provenance matches",
			policy: CreditPolicy{
				RequiresCredits:  true,
				BypassConditions: []BypassCondition{BypassCredentialProvenance, BypassResourceOpenrouter},
			},
			provenance:            CredentialProvenanceAuthority,
			hasResourceOpenrouter: false,
			want:                  false,
		},
		{
			name: "multiple bypass conditions - openrouter matches",
			policy: CreditPolicy{
				RequiresCredits:  true,
				BypassConditions: []BypassCondition{BypassCredentialProvenance, BypassResourceOpenrouter},
			},
			provenance:            CredentialProvenanceNone,
			hasResourceOpenrouter: true,
			want:                  false,
		},
		{
			name: "multiple bypass conditions - none match",
			policy: CreditPolicy{
				RequiresCredits:  true,
				BypassConditions: []BypassCondition{BypassCredentialProvenance, BypassResourceOpenrouter},
			},
			provenance:            CredentialProvenanceNone,
			hasResourceOpenrouter: false,
			want:                  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.ShouldChargeCredits(tt.provenance, tt.hasResourceOpenrouter, tt.isLocalExecution)
			if got != tt.want {
				t.Errorf("ShouldChargeCredits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreditPolicy_ToInfo(t *testing.T) {
	policy := CreditPolicy{
		RequiresCredits:  true,
		CreditsPerStep:   2,
		BypassConditions: []BypassCondition{BypassCredentialProvenance, BypassResourceOpenrouter},
	}

	info := policy.ToInfo()

	if info.RequiresCredits != true {
		t.Errorf("ToInfo().RequiresCredits = %v, want true", info.RequiresCredits)
	}
	if info.CreditsPerStep != 2 {
		t.Errorf("ToInfo().CreditsPerStep = %d, want 2", info.CreditsPerStep)
	}
	if len(info.BypassConditions) != 2 {
		t.Errorf("ToInfo().BypassConditions length = %d, want 2", len(info.BypassConditions))
	}
}

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
