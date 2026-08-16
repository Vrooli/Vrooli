package cliutil

import (
	"errors"
	"testing"
)

func clearInstanceOverrides(t *testing.T) {
	t.Helper()
	instanceOverridesMu.Lock()
	instanceOverrides = map[string]string{}
	instanceOverridesMu.Unlock()
	t.Cleanup(func() {
		instanceOverridesMu.Lock()
		instanceOverrides = map[string]string{}
		instanceOverridesMu.Unlock()
	})
}

func TestSplitInstance(t *testing.T) {
	cases := []struct {
		in      string
		base    string
		variant string
	}{
		{"swarm-manager", "swarm-manager", ""},
		{"swarm-manager@shadow", "swarm-manager", "shadow"},
		{" swarm-manager @ shadow ", "swarm-manager", "shadow"},
		{"a@b@c", "a@b", "c"},
		{"", "", ""},
	}
	for _, tc := range cases {
		base, variant := SplitInstance(tc.in)
		if base != tc.base || variant != tc.variant {
			t.Errorf("SplitInstance(%q) = (%q,%q), want (%q,%q)", tc.in, base, variant, tc.base, tc.variant)
		}
	}
}

func TestSplitAddress(t *testing.T) {
	cases := []struct {
		name          string
		input         string
		wantNode      string
		wantScenario  string
		wantVariant   string
		wantErrorCode string
	}{
		{name: "bare name", input: "web-search", wantScenario: "web-search"},
		{name: "name with variant", input: "web-search@shadow", wantScenario: "web-search", wantVariant: "shadow"},
		{name: "node with name", input: "minimouse/web-search", wantNode: "minimouse", wantScenario: "web-search"},
		{name: "node with name and variant", input: "minimouse/web-search@shadow", wantNode: "minimouse", wantScenario: "web-search", wantVariant: "shadow"},
		{name: "empty scenario after node", input: "minimouse/", wantErrorCode: "empty_scenario"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node, scenario, variant, err := SplitAddress(tc.input)
			if tc.wantErrorCode != "" {
				if err == nil {
					t.Fatalf("SplitAddress(%q) returned no error", tc.input)
				}
				var parseErr *AddressParseError
				if !errors.As(err, &parseErr) {
					t.Fatalf("SplitAddress(%q) error = %T, want *AddressParseError", tc.input, err)
				}
				if parseErr.Code != tc.wantErrorCode {
					t.Fatalf("SplitAddress(%q) error code = %q, want %q", tc.input, parseErr.Code, tc.wantErrorCode)
				}
				return
			}
			if err != nil {
				t.Fatalf("SplitAddress(%q): %v", tc.input, err)
			}
			if node != tc.wantNode || scenario != tc.wantScenario || variant != tc.wantVariant {
				t.Fatalf("SplitAddress(%q) = (%q,%q,%q), want (%q,%q,%q)", tc.input, node, scenario, variant, tc.wantNode, tc.wantScenario, tc.wantVariant)
			}
		})
	}
}

func TestSplitAddressDoesNotReadAmbientNodeState(t *testing.T) {
	t.Setenv("VROOLI_NODE", "minimouse")
	node, scenario, variant, err := SplitAddress("web-search")
	if err != nil {
		t.Fatalf("SplitAddress: %v", err)
	}
	if node != "" || scenario != "web-search" || variant != "" {
		t.Fatalf("SplitAddress read ambient node state: (%q,%q,%q)", node, scenario, variant)
	}
}

func TestInstanceSlugRoundTrip(t *testing.T) {
	cases := []struct {
		scenario string
		variant  string
		want     string
	}{
		{"swarm-manager", "", "swarm-manager"},
		{"swarm-manager", "live", "swarm-manager"},
		{"swarm-manager", "LIVE", "swarm-manager"},
		{"swarm-manager", "shadow", "swarm-manager@shadow"},
		{"swarm-manager", "Shadow", "swarm-manager@shadow"},
		{"swarm-manager@shadow", "shadow", "swarm-manager@shadow"},
	}
	for _, tc := range cases {
		got := InstanceSlug(tc.scenario, tc.variant)
		if got != tc.want {
			t.Errorf("InstanceSlug(%q,%q) = %q, want %q", tc.scenario, tc.variant, got, tc.want)
		}
		// Round-trip: the slug splits back to (base, variant) matching live=bare.
		base, variant := SplitInstance(got)
		if InstanceSlug(base, variant) != got {
			t.Errorf("round-trip of %q broke: base=%q variant=%q", got, base, variant)
		}
	}
}

func TestShadowedScenariosParsing(t *testing.T) {
	t.Setenv(EnvShadowScenarios, " agent-manager, swarm-manager  test-genie@shadow,, ")
	set := ShadowedScenarios()
	for _, want := range []string{"agent-manager", "swarm-manager", "test-genie"} {
		if _, ok := set[want]; !ok {
			t.Errorf("expected %q in shadowed set, got %v", want, set)
		}
	}
	if len(set) != 3 {
		t.Errorf("expected 3 entries, got %d: %v", len(set), set)
	}
	if !IsShadowed("swarm-manager") || !IsShadowed("swarm-manager@shadow") {
		t.Error("IsShadowed should match bare and suffixed names")
	}
	if IsShadowed("web-console") {
		t.Error("web-console should not be shadowed")
	}
}

func TestResolveVariantPrecedence(t *testing.T) {
	clearInstanceOverrides(t)
	t.Setenv(EnvShadowScenarios, "agent-manager")

	// Ambient: a named scenario resolves to shadow.
	if got := ResolveVariant("agent-manager"); got != ShadowVariant {
		t.Errorf("ambient agent-manager = %q, want shadow", got)
	}
	// Unnamed scenario stays live.
	if got := ResolveVariant("swarm-manager"); got != DefaultVariant {
		t.Errorf("swarm-manager = %q, want live", got)
	}
	// Explicit @suffix beats ambient (even back to live).
	if got := ResolveVariant("agent-manager@live"); got != DefaultVariant {
		t.Errorf("agent-manager@live = %q, want live (suffix wins)", got)
	}
	if got := ResolveVariant("swarm-manager@shadow"); got != ShadowVariant {
		t.Errorf("swarm-manager@shadow = %q, want shadow (suffix wins)", got)
	}

	// Override beats ambient. --instance live on an ambiently-shadowed scenario
	// must force live.
	SetInstanceOverride("agent-manager", "live")
	if got := ResolveVariant("agent-manager"); got != DefaultVariant {
		t.Errorf("override live on agent-manager = %q, want live", got)
	}
	// Override shadow on an unshadowed scenario.
	SetInstanceOverride("swarm-manager", "shadow")
	if got := ResolveVariant("swarm-manager"); got != ShadowVariant {
		t.Errorf("override shadow on swarm-manager = %q, want shadow", got)
	}
	// Clearing the override restores ambient behavior.
	SetInstanceOverride("agent-manager", "")
	if got := ResolveVariant("agent-manager"); got != ShadowVariant {
		t.Errorf("cleared override agent-manager = %q, want shadow (ambient)", got)
	}
}

func TestResolveShadowTargetAndNonLive(t *testing.T) {
	clearInstanceOverrides(t)
	t.Setenv(EnvShadowScenarios, "agent-manager")

	if got := ResolveShadowTarget("agent-manager"); got != "agent-manager@shadow" {
		t.Errorf("ResolveShadowTarget(agent-manager) = %q, want agent-manager@shadow", got)
	}
	if got := ResolveShadowTarget("swarm-manager"); got != "swarm-manager" {
		t.Errorf("ResolveShadowTarget(swarm-manager) = %q, want bare", got)
	}
	if !IsNonLiveTarget("agent-manager@shadow") {
		t.Error("agent-manager@shadow should be non-live")
	}
	if IsNonLiveTarget("agent-manager") || IsNonLiveTarget("agent-manager@live") {
		t.Error("bare and @live targets should be live")
	}
}
