package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/agentcatalog/harness"
	"web-console/handlers/modelselection"
)

func TestRoleModelChoicesArePolicyDeclaredButAvailabilityUnknown(t *testing.T) {
	root := t.TempDir()
	policyDir := filepath.Join(root, "resources", "opencode")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	policy, err := os.ReadFile(filepath.Join("..", "..", "..", "resources", "opencode", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "model-policy.json"), policy, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_RESOURCE_ROOT", root)

	got := (&Server{}).GetRoleModelChoices(t.Context(), "opencode", "local", "")
	if got.PolicyStatus != "ready" || got.Availability != "unknown" {
		t.Fatalf("expected policy-only choices, got status=%q availability=%q detail=%q", got.PolicyStatus, got.Availability, got.Detail)
	}
	var ollamaFound bool
	var paidChoiceFound bool
	for _, choice := range got.Choices {
		if choice.Source == "vrooli-credits" {
			paidChoiceFound = true
		}
		if choice.Source == "ollama" {
			ollamaFound = true
			if !strings.HasPrefix(choice.LaunchModel, "ollama/") {
				t.Fatalf("launch model %q did not bind declared source", choice.LaunchModel)
			}
		}
	}
	if !ollamaFound {
		t.Fatal("expected the resource-declared local Ollama choice")
	}
	if paidChoiceFound {
		t.Fatal("unqualified server must not advertise Vrooli-funded coding")
	}
}

func TestQualifiedPaidCodingChoiceIsApplicationOwnedAndFixed(t *testing.T) {
	root := t.TempDir()
	policyDir := filepath.Join(root, "resources", "opencode")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	policy, err := os.ReadFile(filepath.Join("..", "..", "..", "resources", "opencode", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "model-policy.json"), policy, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_RESOURCE_ROOT", root)

	got := paidCodingAuthorityFixture(t).GetAccountRoleModelChoices(t.Context(), "opencode", "local", "", "account")
	for _, choice := range got.Choices {
		if choice.Source == "vrooli-credits" {
			if choice.Model != "coding" || choice.LaunchModel != "coding" || choice.SourceLabel != "Vrooli credits" {
				t.Fatalf("paid model must remain fixed to the qualified account-bound route: %+v", choice)
			}
			return
		}
	}
	t.Fatal("qualified local OpenCode policy did not advertise the application-owned paid route")
}

func TestRoleModelChoicesExposeStalePolicyAndRejectRemoteCatalogGuess(t *testing.T) {
	root := t.TempDir()
	policyDir := filepath.Join(root, "resources", "opencode")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	policy, err := os.ReadFile(filepath.Join("..", "..", "..", "resources", "opencode", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(policy, &doc); err != nil {
		t.Fatal(err)
	}
	provenance := doc["provenance"].(map[string]any)
	provenance["observed_at"] = "2025-01-01"
	policy, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "model-policy.json"), policy, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_RESOURCE_ROOT", root)

	stale := (&Server{}).GetRoleModelChoices(t.Context(), "opencode", "local", "")
	if stale.PolicyStatus != "stale" || stale.Availability != "unknown" || len(stale.Choices) == 0 {
		t.Fatalf("stale policy evidence was lost: %+v", stale)
	}
	if !stale.CatalogStale || stale.ModelFlag == "" {
		t.Fatalf("a stale list must warn and keep its choices launchable: %+v", stale)
	}
	remote := (&Server{}).GetRoleModelChoices(t.Context(), "opencode", "remote:one", "")
	if remote.PolicyStatus != "unsupported" || len(remote.Choices) != 0 {
		t.Fatalf("remote target should not inherit local choices: %+v", remote)
	}
}

func TestOpenCodeOffersEveryHostedProviderModelItsResourceDeclares(t *testing.T) {
	useRepoPolicy(t, "opencode", nil)
	result := (&Server{}).GetRoleModelChoices(t.Context(), "opencode", "local", "")
	offered := map[string]string{}
	for _, choice := range result.Choices {
		offered[choice.LaunchModel] = choice.DisplayName
	}
	rawAliases, err := os.ReadFile(filepath.Join("..", "..", "..", "resources", "opencode", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var aliasDoc struct {
		ModelAliases map[string]struct {
			DisplayName string `json:"display_name"`
		} `json:"model_aliases"`
	}
	if err := json.Unmarshal(rawAliases, &aliasDoc); err != nil {
		t.Fatal(err)
	}
	for _, provider := range []string{"abliteration", "orcarouter"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "resources", provider, "model-policy.json"))
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Models map[string]struct {
				DisplayName  string   `json:"display_name"`
				Endpoints    []string `json:"endpoints"`
				Capabilities []string `json:"capabilities"`
			} `json:"models"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		for model, meta := range doc.Models {
			if !slices.Contains(meta.Endpoints, "chat") && !slices.Contains(meta.Capabilities, "chat") {
				continue
			}
			launchModel := provider + "/" + model
			displayName, ok := offered[launchModel]
			if !ok {
				t.Errorf("%s is declared but not offered by the OpenCode launcher", launchModel)
				continue
			}
			if meta.DisplayName != "" && displayName != meta.DisplayName {
				t.Errorf("%s label = %q, provider name = %q", launchModel, displayName, meta.DisplayName)
			}
			if aliasDoc.ModelAliases[launchModel].DisplayName == "" {
				t.Errorf("%s has no display name in the launcher aliases", launchModel)
			}
		}
	}
}

func TestOpenCodeUsesProviderDisplayNames(t *testing.T) {
	useRepoPolicy(t, "opencode", nil)
	result := (&Server{}).GetRoleModelChoices(t.Context(), "opencode", "local", "")
	names := map[string]string{}
	for _, choice := range result.Choices {
		names[choice.LaunchModel] = choice.DisplayName
	}
	for model, want := range map[string]string{
		"abliteration/abliterated-model-large-v2": "Abliterated Large v2",
		"openrouter/deepseek/deepseek-v4.1-flash": "DeepSeek V4.1 Flash",
		"opencode-go/longcat-2.0":                 "LongCat-2.0",
		"orcarouter/openai/gpt-4o-mini":           "GPT-4o mini",
		"ollama/gemma4:12b":                       "Gemma 4 12B",
	} {
		if names[model] != want {
			t.Errorf("%s label = %q, want provider name %q", model, names[model], want)
		}
	}
}

func TestSafeNativeModelArgRejectsShellSyntax(t *testing.T) {
	for _, model := range []string{"provider/model-v2:12b", "gemma4:12b"} {
		if !safeNativeModelArg.MatchString(model) {
			t.Errorf("safe model id %q was rejected", model)
		}
	}
	for _, model := range []string{"model;touch /tmp/pwned", "model$(id)", "model with spaces", "--flag"} {
		if safeNativeModelArg.MatchString(model) {
			t.Errorf("unsafe model id %q was accepted", model)
		}
	}
}

func TestModelSelectionDescriptorsDoNotAdvertiseUnqualifiedVrooliFunding(t *testing.T) {
	supported := 0
	for _, descriptor := range harness.Builtin().All() {
		if descriptor.Features.ModelSelection == nil {
			continue
		}
		supported++
		if slices.Contains(descriptor.Features.ModelSelection.Sources, "vrooli-credits") {
			t.Errorf("%q exposes Vrooli-funded model choices before coding execution and accounting are qualified", descriptor.Resource)
		}
	}
	if supported != 3 {
		t.Fatalf("expected codex, claude-code and opencode to have a native model mapping, got %d", supported)
	}
}

// useRepoPolicy points the resource root at a copy of one resource's real policy,
// edited as a generic document so a test states only what it changes.
func useRepoPolicy(t *testing.T, resource string, edit func(map[string]any)) {
	t.Helper()
	policy, err := os.ReadFile(filepath.Join("..", "..", "..", "resources", resource, "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(policy, &doc); err != nil {
		t.Fatal(err)
	}
	doc["provenance"].(map[string]any)["observed_at"] = time.Now().UTC().Format("2006-01-02")
	if edit != nil {
		edit(doc)
	}
	if policy, err = json.Marshal(doc); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	dir := filepath.Join(root, "resources", resource)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model-policy.json"), policy, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_RESOURCE_ROOT", root)
}

func choiceNamed(t *testing.T, result modelselection.Result, model string) modelselection.Choice {
	t.Helper()
	for _, choice := range result.Choices {
		if choice.LaunchModel == model {
			return choice
		}
	}
	t.Fatalf("no choice %q in %+v", model, result.Choices)
	return modelselection.Choice{}
}

func TestClaudeChoicesOfferOnlyLimitsTheReportedWindowAllows(t *testing.T) {
	useRepoPolicy(t, "claude-code", nil)
	windows := filepath.Join(t.TempDir(), "windows.json")
	t.Setenv(harness.ModelWindowsEnv, windows)

	unknown := choiceNamed(t, (&Server{}).GetRoleModelChoices(t.Context(), "claude", "local", ""), "sonnet")
	if unknown.ContextWindowTokens != 0 || !slices.Equal(unknown.ContextLimitTokens, []int64{1000000, 300000, 150000}) {
		t.Fatalf("an unknown window is stated as zero and offers what Claude accepts: %+v", unknown)
	}
	for _, tc := range []struct {
		tokens int64
		want   []int64
	}{
		{1000000, []int64{1000000, 300000, 150000}},
		{200000, []int64{150000}},
		{120000, nil},
	} {
		if err := harness.RecordModelWindow(windows, "claude-sonnet-5-5", tc.tokens, time.Now()); err != nil {
			t.Fatal(err)
		}
		got := choiceNamed(t, (&Server{}).GetRoleModelChoices(t.Context(), "claude", "local", ""), "sonnet")
		if got.ContextWindowTokens != tc.tokens || !slices.Equal(got.ContextLimitTokens, tc.want) {
			t.Fatalf("window %d: got %+v, want limits %v", tc.tokens, got, tc.want)
		}
	}
}

func TestCodexChoicesReadWindowsThroughTheSharedPackage(t *testing.T) {
	useRepoPolicy(t, "codex", nil)
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	for _, tc := range []struct {
		window int64
		limits []int64
	}{
		{500000, []int64{300000, 150000}},
		{272000, []int64{150000}},
	} {
		cache := fmt.Sprintf(`{"models":[{"slug":"gpt-6.1-terra","context_window":%d}]}`, tc.window)
		if err := os.WriteFile(filepath.Join(home, "models_cache.json"), []byte(cache), 0o600); err != nil {
			t.Fatal(err)
		}
		got := choiceNamed(t, (&Server{}).GetRoleModelChoices(t.Context(), "codex", "local", ""), "gpt-6.1-terra")
		if got.ContextWindowTokens != tc.window || !slices.Equal(got.ContextLimitTokens, tc.limits) {
			t.Errorf("window %d: got %+v", tc.window, got)
		}
		if got.Metered {
			t.Errorf("%s bills through a subscription", got.Model)
		}
	}
}

func TestHarnessWithoutContextLimitGetsNoLimitsAndMeteredChoicesAreMarked(t *testing.T) {
	useRepoPolicy(t, "opencode", nil)
	result := paidCodingAuthorityFixture(t).GetAccountRoleModelChoices(t.Context(), "opencode", "local", "", "account")
	if result.CatalogStale || result.PolicyStatus != "ready" {
		t.Fatalf("a fresh list must not read as stale: %+v", result)
	}
	metered := map[string]bool{}
	for _, choice := range result.Choices {
		if choice.ContextWindowTokens != 0 || len(choice.ContextLimitTokens) != 0 {
			t.Errorf("%s offered a limit OpenCode cannot honour: %+v", choice.Model, choice)
		}
		metered[choice.Source] = choice.Metered
	}
	if !metered["vrooli-credits"] || !metered["openrouter"] || metered["ollama"] {
		t.Fatalf("metered by source: %v", metered)
	}
}

func TestShippedClaudeCatalogNamesTheCurrentModels(t *testing.T) {
	useRepoPolicy(t, "claude-code", nil)
	policy, err := os.ReadFile(filepath.Join(os.Getenv("VROOLI_RESOURCE_ROOT"), "resources", "claude-code", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		ModelAliases map[string]struct {
			CanonicalModel string `json:"canonical_model"`
		} `json:"model_aliases"`
	}
	if err := json.Unmarshal(policy, &doc); err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"claude-opus-5-5", "claude-sonnet-5-5"} {
		if doc.ModelAliases[model].CanonicalModel != "anthropic/"+model {
			t.Errorf("%s is missing from the policy", model)
		}
	}
}

func TestClaudeChoicesOfferEveryCatalogModelOnceWithTheDefaultMarkedOnce(t *testing.T) {
	useRepoPolicy(t, "claude-code", nil)
	result := (&Server{}).GetRoleModelChoices(t.Context(), "claude", "local", "")
	names := map[string]string{}
	defaults := 0
	for _, choice := range result.Choices {
		if choice.Source != "claude-subscription" {
			t.Errorf("claude offers only its declared sources, got %+v", choice)
		}
		if prior, dup := names[choice.DisplayName]; dup {
			t.Fatalf("%q offered twice: %s and %s", choice.DisplayName, prior, choice.Model)
		}
		names[choice.DisplayName] = choice.Model
		if choice.IsDefault {
			defaults++
		}
		if choice.Metered {
			t.Errorf("%s bills through a subscription, so it is not metered", choice.DisplayName)
		}
	}
	if defaults != 1 || !result.Choices[0].IsDefault {
		t.Fatalf("the default comes first and exactly once: %+v", result.Choices)
	}
	for name, model := range map[string]string{"Sonnet 5.5": "sonnet", "Opus 5.5": "opus", "Fable 5.1": "fable", "Haiku 5.5": "haiku"} {
		if names[name] != model {
			t.Errorf("%s is %q, want %q; got %v", name, names[name], model, names)
		}
	}
}

func TestCodexChoicesLeaveOutExcludedModels(t *testing.T) {
	useRepoPolicy(t, "codex", nil)
	result := (&Server{}).GetRoleModelChoices(t.Context(), "codex", "local", "")
	got := map[string]bool{}
	for _, choice := range result.Choices {
		got[choice.Model] = true
		if choice.DisplayName == "" || choice.DisplayName == choice.Model {
			t.Errorf("%s has no friendly name: %q", choice.Model, choice.DisplayName)
		}
	}
	for _, excluded := range []string{"gpt-6-astra", "gpt-6-sol", "gpt-5.5", "gpt-5.6-luna", "gpt-5.6-sol", "gpt-5.6-terra", "codex-auto-review", "gpt-reserve"} {
		if got[excluded] {
			t.Errorf("%s is excluded and must not be offered", excluded)
		}
	}
	// gpt-6.1-sol is restricted to the fallback role, so a launcher does not offer it.
	if got["gpt-6.1-sol"] {
		t.Error("gpt-6.1-sol is restricted and must not be offered")
	}
	for _, offered := range []string{"gpt-6.1-terra", "gpt-6-luna"} {
		if !got[offered] {
			t.Errorf("%s should be offered, got %v", offered, got)
		}
	}
}

func TestChoicesSayWhenACommandIsTheOperatorsOwn(t *testing.T) {
	useRepoPolicy(t, "claude-code", nil)
	s := &Server{}
	if !s.GetRoleModelChoices(t.Context(), "claude", "local", "my-wrapper claude").CommandLiteral {
		t.Error("a wrapper must be reported literal")
	}
	if s.GetRoleModelChoices(t.Context(), "claude", "local", "claude --dangerously-skip-permissions").CommandLiteral {
		t.Error("a plain start takes settings")
	}
}

func TestClaudeChoicesOrderByTierThenNewest(t *testing.T) {
	useRepoPolicy(t, "claude-code", nil)
	var got []string
	for _, choice := range (&Server{}).GetRoleModelChoices(t.Context(), "claude", "local", "").Choices {
		got = append(got, choice.DisplayName)
	}
	want := []string{"Sonnet 5.5", "Fable 5.1", "Fable 5", "Opus 5.5", "Opus 5", "Sonnet 5", "Haiku 5.5", "Haiku 4.5"}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestSplitModelName(t *testing.T) {
	for name, want := range map[string]struct {
		family  string
		version []int
	}{
		"Opus 5.5":            {"Opus", []int{5, 5}},
		"DeepSeek V4.1 Flash": {"DeepSeek", []int{4, 1}},
		"Grok Build":          {"Grok Build", nil},
	} {
		family, version := splitModelName(name)
		if family != want.family || !slices.Equal(version, want.version) {
			t.Errorf("%s: %q %v", name, family, version)
		}
	}
}

func TestModelDisplayNames(t *testing.T) {
	catalog := agentcatalog.CodingRoleCatalog{ModelAliases: map[string]agentcatalog.ModelAlias{
		"haiku":    {CanonicalModel: "anthropic/claude-haiku-4-5-20251001"},
		"friendly": {CanonicalModel: "vendor/internal-id", DisplayName: "Provider Friendly Name"},
	}}
	for model, want := range map[string]string{
		"friendly":      "Provider Friendly Name",
		"haiku":         "Haiku 4.5",
		"gpt-6.1-terra": "GPT 6.1 Terra",
		"openrouter/deepseek/deepseek-v4.1-flash": "DeepSeek V4.1 Flash",
		"opencode-go/mimo-v2.5":                   "MiMo V2.5",
		"gemma4:12b":                              "Gemma4 12b",
		"default":                                 "Default",
		"grok-build":                              "Grok Build",
	} {
		if got := modelDisplayName(catalog, model); got != want {
			t.Errorf("%s = %q, want %q", model, got, want)
		}
	}
}
