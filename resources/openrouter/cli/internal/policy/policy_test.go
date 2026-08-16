package policy_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"resource-openrouter/cli/internal/policy"
	"resource-openrouter/cli/internal/policytest"
)

func writeFixture(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "model-policy.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadFixture(t *testing.T) policy.Policy {
	t.Helper()
	p, err := policy.LoadFile(writeFixture(t, policytest.FixturePolicyJSON))
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return p
}

func TestResolveRole(t *testing.T) {
	p := loadFixture(t)
	got, err := p.ResolveRole("image.generate.logo")
	if err != nil {
		t.Fatalf("ResolveRole: %v", err)
	}
	if got.Model != "vendor/img-vec" {
		t.Fatalf("model = %q", got.Model)
	}
	if got.Endpoint != "images" {
		t.Fatalf("endpoint = %q", got.Endpoint)
	}
	if got.Source != "role" {
		t.Fatalf("source = %q", got.Source)
	}
	if got.RequestDefaults == nil || got.RequestDefaults.OutputFormat != "png" || got.RequestDefaults.Background != "transparent" {
		t.Fatalf("request_defaults not propagated: %+v", got.RequestDefaults)
	}
	if got.RoleProvenance == nil {
		t.Fatal("role provenance missing")
	}
}

func TestResolveRoleUnknown(t *testing.T) {
	p := loadFixture(t)
	if _, err := p.ResolveRole("nope.role"); err == nil {
		t.Fatal("expected unknown role error")
	}
}

func TestResolveModel(t *testing.T) {
	p := loadFixture(t)
	got, err := p.ResolveModel("vendor/chat-a")
	if err != nil {
		t.Fatalf("ResolveModel: %v", err)
	}
	if got.Source != "model" || got.Endpoint != "chat" {
		t.Fatalf("unexpected resolve: %+v", got)
	}
	if _, err := p.ResolveModel("vendor/missing"); err == nil {
		t.Fatal("expected unknown model error")
	}
}

func TestResolveRequestRoles(t *testing.T) {
	p := loadFixture(t)
	res, err := p.Resolve(policy.ResolveRequest{ModelRoles: []policy.RoleRequest{{Role: "chat.default"}, {Role: "image.generate.logo"}}})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(res.Roles) != 2 {
		t.Fatalf("roles = %d", len(res.Roles))
	}
	_, err = p.Resolve(policy.ResolveRequest{ModelRoles: []policy.RoleRequest{{Role: "bad"}}})
	if err == nil {
		t.Fatal("expected unknown role in Resolve")
	}
}

// validation failure cases — each mutation must make LoadFile reject.
func TestValidateRejections(t *testing.T) {
	cases := map[string]func(string) string{
		"fallback missing capability": func(s string) string {
			// give the logo role's model a chat-only fallback that lacks image_output
			return strings.Replace(s, `"fallbacks": [],`, `"fallbacks": ["vendor/chat-a"],`, 1)
		},
		"endpoint mismatch": func(s string) string {
			return strings.Replace(s, `"endpoint": "images",`, `"endpoint": "chat",`, 1)
		},
		"capability outside vocabulary": func(s string) string {
			return strings.Replace(s, `"required_capabilities": ["chat"],`, `"required_capabilities": ["telepathy"],`, 1)
		},
		"bad provenance source_kind": func(s string) string {
			return strings.Replace(s, `"source_kind": "manual_policy", "confidence": "high", "source": "fixture"`, `"source_kind": "made_up", "confidence": "high", "source": "fixture"`, 1)
		},
		"out of range temperature": func(s string) string {
			return strings.Replace(s, `"temperature": 0.7,`, `"temperature": 9.9,`, 1)
		},
		"unknown sampling support state": func(s string) string {
			return strings.Replace(s, `"sampling_support": {"temperature": "honored"},`, `"sampling_support": {"temperature": "probably"},`, 1)
		},
		"unknown sampling support parameter": func(s string) string {
			return strings.Replace(s, `"sampling_support": {"temperature": "honored"},`, `"sampling_support": {"temperatue": "honored"},`, 1)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			body := mutate(policytest.FixturePolicyJSON)
			if _, err := policy.LoadFile(writeFixture(t, body)); err == nil {
				t.Fatalf("expected validation failure for %q", name)
			}
		})
	}
}

func TestRealPolicyValid(t *testing.T) {
	// The shipped policy must load and resolve every role.
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "model-policy.json")
	if _, statErr := os.Stat(path); statErr != nil {
		t.Skipf("shipped policy not found: %v", statErr)
	}
	p, err := policy.LoadFile(path)
	if err != nil {
		t.Fatalf("shipped policy invalid: %v", err)
	}
	for _, r := range p.RoleNames() {
		if _, err := p.ResolveRole(r); err != nil {
			t.Fatalf("resolve %s: %v", r, err)
		}
	}
}

func TestResolveGeneratePrecedence(t *testing.T) {
	t.Parallel()

	roleTemperature := 0.0
	roleMaxTokens := 512
	defaults := &policy.RequestDefaults{Temperature: &roleTemperature, MaxTokens: &roleMaxTokens}

	cases := []struct {
		name            string
		defaults        *policy.RequestDefaults
		temperatureFlag float64
		maxTokensFlag   int
		wantTemperature *float64
		wantMaxTokens   int
	}{
		{
			name:            "absent flag adopts the role default",
			defaults:        defaults,
			temperatureFlag: -1,
			maxTokensFlag:   0,
			wantTemperature: floatPtr(0),
			wantMaxTokens:   512,
		},
		{
			name:            "explicit flag overrides the role default",
			defaults:        defaults,
			temperatureFlag: 1.2,
			maxTokensFlag:   4096,
			wantTemperature: floatPtr(1.2),
			wantMaxTokens:   4096,
		},
		{
			name:            "no flag and no default omits the parameter",
			defaults:        nil,
			temperatureFlag: -1,
			maxTokensFlag:   0,
			wantTemperature: nil,
			wantMaxTokens:   0,
		},
		{
			name:            "explicit zero is a request, not an absence",
			defaults:        nil,
			temperatureFlag: 0,
			maxTokensFlag:   0,
			wantTemperature: floatPtr(0),
			wantMaxTokens:   0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotTemperature, gotMaxTokens := tc.defaults.ResolveGenerate(tc.temperatureFlag, tc.maxTokensFlag)
			switch {
			case tc.wantTemperature == nil && gotTemperature != nil:
				t.Fatalf("temperature = %v, want unset", *gotTemperature)
			case tc.wantTemperature != nil && gotTemperature == nil:
				t.Fatalf("temperature = unset, want %v", *tc.wantTemperature)
			case tc.wantTemperature != nil && *gotTemperature != *tc.wantTemperature:
				t.Fatalf("temperature = %v, want %v", *gotTemperature, *tc.wantTemperature)
			}
			if gotMaxTokens != tc.wantMaxTokens {
				t.Fatalf("max_tokens = %d, want %d", gotMaxTokens, tc.wantMaxTokens)
			}
		})
	}
}

func floatPtr(v float64) *float64 { return &v }

func TestResolveRoleCarriesSamplingSupport(t *testing.T) {
	t.Parallel()

	p, err := policy.LoadFile(writeFixture(t, policytest.FixturePolicyJSON))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := p.ResolveRole("chat.default")
	if err != nil {
		t.Fatal(err)
	}
	if got := resolved.SamplingSupport["temperature"]; got != policy.SamplingHonored {
		t.Fatalf("sampling_support[temperature] = %q, want %q", got, policy.SamplingHonored)
	}
	// A role that declares nothing must resolve to "unknown", not to an error
	// and not to a fabricated promise.
	if got := p.Roles["image.generate.logo"].SupportFor("temperature"); got != policy.SamplingUnknown {
		t.Fatalf("undeclared support = %q, want %q", got, policy.SamplingUnknown)
	}
}
