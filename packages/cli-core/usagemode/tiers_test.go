package usagemode

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func readPolicy(t *testing.T, resource string) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../../resources/" + resource + "/model-policy.json")
	if err != nil {
		t.Skip("repo resource policies not available: ", err)
	}
	return raw
}

// aliasModel resolves a policy alias to the bare model ID the runner is given
// (the canonical model without its provider prefix), so shipped-policy tests
// follow model bumps in the policy instead of pinning yesterday's IDs.
func aliasModel(t *testing.T, policy []byte, alias string) string {
	t.Helper()
	var doc struct {
		ModelAliases map[string]struct {
			CanonicalModel string `json:"canonical_model"`
		} `json:"model_aliases"`
	}
	if err := json.Unmarshal(policy, &doc); err != nil {
		t.Fatal(err)
	}
	canonical := doc.ModelAliases[alias].CanonicalModel
	if canonical == "" {
		t.Fatalf("policy has no alias %q", alias)
	}
	return canonical[strings.LastIndex(canonical, "/")+1:]
}

func TestShippedCodexPolicy(t *testing.T) {
	tiers, err := LoadTiers(readPolicy(t, "codex"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewMode(tiers, NewModeOptions{Now: t0, Duration: 6 * time.Hour, Scope: Scope{Runner: "codex"}})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := m.SettingsFor("codex")
	if s.SessionModel != "gpt-6-luna" || s.ModelMap["gpt-6.1-terra"] != "gpt-6-luna" {
		t.Fatalf("%+v", s)
	}
	if m.Until.Kind != UntilTime || m.ExpiresAt == nil || !m.ExpiresAt.Equal(t0.Add(6*time.Hour)) {
		t.Fatalf("%+v", m)
	}
	for _, role := range []string{"code.fallback", "code.review", "judgment.supervision"} {
		if _, ok := resolveFrom([]Mode{m}, Query{Runner: "codex", Role: role, Now: t0}); ok {
			t.Errorf("%s not exempt", role)
		}
	}
}

func TestShippedClaudePolicy(t *testing.T) {
	policy := readPolicy(t, "claude-code")
	tiers, err := LoadTiers(policy)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewMode(tiers, NewModeOptions{Now: t0, Duration: time.Hour, ContextLimitTokens: 150000})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := m.SettingsFor("claude-code")
	// The session runs on the current Sonnet; Opus steps down to it and
	// Fable steps down to the current Opus.
	sonnet, opus := aliasModel(t, policy, "sonnet"), aliasModel(t, policy, "opus")
	if s.SessionModel != sonnet || s.ModelMap["opus"] != sonnet || s.ModelMap["fable"] != opus {
		t.Fatalf("settings %+v, want session and opus -> %s, fable -> %s", s, sonnet, opus)
	}
	if _, has := s.ModelMap["sonnet"]; has {
		t.Fatal("sonnet is the floor")
	}
	if _, err := NewMode(tiers, NewModeOptions{Now: t0, Duration: time.Hour, ContextLimitTokens: 90000}); err == nil {
		t.Fatal("context limit below 100k accepted")
	}
	if err := tiers.ValidateTarget("openrouter/deepseek/deepseek-v4.1-flash"); err == nil || !strings.Contains(err.Error(), "metered") {
		t.Fatalf("metered target: %v", err)
	}
	if err := tiers.ValidateTarget("haiku"); err != nil {
		t.Fatal(err)
	}
}

const policyFixture = `{"runner":"codex","excluded_models":["bad"],"restricted_models":{"sol":["code.fallback"]},
"billing":{"mode":"subscription","sources":[{"id":"paid","billing_mode":"metered"}]},
"model_aliases":{"low":{"canonical_model":"codex/low","provider":"codex"},"bad":{"canonical_model":"codex/bad","provider":"codex"},
"sol":{"canonical_model":"codex/sol","provider":"codex"},"paid-m":{"canonical_model":"paid/m","provider":"paid"},"high":{"canonical_model":"codex/high","provider":"codex"}},
"usage_tiers":{"model_map":{"high":"%s"},"session_model":"low"}}`

func TestNewModeRejectsBadTargets(t *testing.T) {
	for target, want := range map[string]string{"bad": "excluded", "sol": "restricted", "paid-m": "metered", "ghost": "model_aliases"} {
		tiers, err := LoadTiers([]byte(strings.Replace(policyFixture, "%s", target, 1)))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := NewMode(tiers, NewModeOptions{Now: t0, Duration: time.Hour}); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", target, err, want)
		}
	}
	tiers, _ := LoadTiers([]byte(strings.Replace(policyFixture, "%s", "low", 1)))
	if _, err := NewMode(tiers, NewModeOptions{Now: t0}); err != nil {
		t.Fatalf("off mode: %v", err)
	}
	if _, err := LoadTiers([]byte(`{"runner":"codex"}`)); err == nil {
		t.Fatal("no usage_tiers must error")
	}
	exp := t0.Add(-time.Minute)
	if _, err := NewMode(tiers, NewModeOptions{Now: t0, ExpiresAt: &exp}); err == nil {
		t.Fatal("past expiry accepted")
	}
}

func TestTierRankFollowsTheModelMap(t *testing.T) {
	policy := readPolicy(t, "claude-code")
	tiers, err := LoadTiers(policy)
	if err != nil {
		t.Fatal(err)
	}
	currentSonnet := aliasModel(t, policy, "sonnet")
	for _, c := range []struct {
		candidate, replaced string
		lower               bool
	}{
		{"sonnet", "opus", true},
		{currentSonnet, "fable", true},
		{"opus", "fable", true},
		{"opus", "opus", false},
		{"fable", "sonnet", false},
		{"opus", "sonnet", false},
		{"haiku", "opus", false},            // not in the tier order: cannot be proven lower
		{"claude-sonnet-5", "fable", false}, // a retired pin outside the tier order is not provably lower
	} {
		if got := tiers.LowerThan(c.candidate, c.replaced); got != c.lower {
			t.Errorf("LowerThan(%s, %s) = %v, want %v", c.candidate, c.replaced, got, c.lower)
		}
	}
}

func TestOverrideModelOnlyLowers(t *testing.T) {
	tiers, err := LoadTiers(readPolicy(t, "claude-code"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tiers.OverrideModel("sonnet"); err != nil {
		t.Fatalf("lowering to the floor must work: %v", err)
	}
	for _, model := range []string{"opus", "fable", "claude-fable-5-1", "haiku"} {
		if _, err := tiers.OverrideModel(model); err == nil {
			t.Errorf("--model %s was accepted but is not below every model it replaces", model)
		}
	}
	codex, err := LoadTiers(readPolicy(t, "codex"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codex.OverrideModel("gpt-6.1-sol"); err == nil {
		t.Fatal("codex override to an unranked higher model was accepted")
	}
}

func TestNewModeRefusesAMapThatRaises(t *testing.T) {
	policy := []byte(`{"runner":"claude-code","model_aliases":{"a":{"canonical_model":"p/a"},"b":{"canonical_model":"p/b"},"c":{"canonical_model":"p/c"}},` +
		`"usage_tiers":{"model_map":{"a":"b","b":"c","c":"a"}}}`)
	tiers, err := LoadTiers(policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewMode(tiers, NewModeOptions{Now: t0, Duration: time.Hour}); err == nil {
		t.Fatal("a cyclic tier map that raises a model was accepted")
	}
}
