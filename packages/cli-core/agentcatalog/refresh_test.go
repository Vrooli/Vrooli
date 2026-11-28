package agentcatalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/usagemode"
)

const claudeBakedCatalog = `xx{id:"claude-opus-5",family:"opus",display_name:"Opus 5"},{id:"claude-opus-5-5",family:"opus",display_name:"Opus 5.5"},` +
	`{id:"claude-sonnet-5",family:"sonnet",display_name:"Sonnet 5"},{id:"claude-sonnet-5-5",family:"sonnet",display_name:"Sonnet 5.5"},` +
	`{id:"claude-fable-5-1",family:"fable",display_name:"Fable 5.1"},{id:"claude-haiku-5-5",family:"haiku",display_name:"Haiku 5.5"}],` +
	`aliases:{opus:{default:"claude-opus-5-5",per_provider:{bedrock:"claude-opus-5-5"}},sonnet:{default:"claude-sonnet-5-5",per_provider:{bedrock:"claude-sonnet-5"}},haiku:{default:"claude-haiku-5-5"},fable:{default:"claude-fable-5-1",per_provider:{gateway:"claude-fable-5"}}},defaults:{}`

const oldClaudePolicy = `{
  "schema_version": "v1",
  "runner": "claude-code",
  "excluded_models": ["gpt-6-astra"],
  "provenance": {"source": "old", "observed_at": "2026-01-01"},
  "usage_tiers": {
    "model_map": {"fable": "opus", "opus": "sonnet", "claude-fable-5-1": "claude-opus-5", "claude-opus-5": "claude-sonnet-5"},
    "session_model": "sonnet"
  },
  "model_aliases": {
    "sonnet": {"canonical_model": "anthropic/claude-sonnet-5", "provider": "anthropic"},
    "opus": {"canonical_model": "anthropic/claude-opus-5", "provider": "anthropic"},
    "claude-opus-5": {"canonical_model": "anthropic/claude-opus-5", "provider": "anthropic"},
    "claude-sonnet-5": {"canonical_model": "anthropic/claude-sonnet-5", "provider": "anthropic"},
    "haiku": {"canonical_model": "anthropic/claude-haiku-5-5", "provider": "anthropic"},
    "fable": {"canonical_model": "anthropic/claude-fable-5-1", "provider": "anthropic"},
    "claude-fable-5-1": {"canonical_model": "anthropic/claude-fable-5-1", "provider": "anthropic"},
    "openrouter/x": {"canonical_model": "openrouter/x", "provider": "openrouter"}
  },
  "roles": {
    "code.default": {"models": [{"model": "sonnet", "source": "claude-subscription", "effort": null}, {"model": "openrouter/x", "source": "openrouter", "effort": null}], "model": "sonnet", "canonical_model": "anthropic/claude-sonnet-5", "fallbacks": ["claude-sonnet-5"], "description": "d", "capabilities": ["code"]},
    "code.fast": {"model": "sonnet", "canonical_model": "anthropic/claude-sonnet-5", "fallbacks": ["haiku"], "description": "d", "capabilities": ["code"]},
    "code.smart": {"model": "fable", "canonical_model": "anthropic/claude-fable-5-1", "fallbacks": ["claude-fable-5-1", "claude-opus-5"], "description": "d", "capabilities": ["code"]},
    "code.cheap": {"model": "haiku", "canonical_model": "anthropic/claude-haiku-5-5", "description": "d", "capabilities": ["code"]}
  }
}
`

func writeFakeRunner(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func writePolicy(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

var refreshNow = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestRefreshMovesClaudeAliasesTiersAndProvenanceFromTheInstalledBinary(t *testing.T) {
	writeFakeRunner(t, "claude", "#!/bin/sh\necho '2.1.312 (Claude Code)'\nexit 0\n# "+claudeBakedCatalog+"\n")
	path := writePolicy(t, oldClaudePolicy)

	result, err := RefreshCodingRoleCatalog(context.Background(), "claude-code", path, refreshNow)
	if err != nil {
		t.Fatal(err)
	}
	catalog, raw, err := LoadCodingRoleCatalog("claude-code", path)
	if err != nil {
		t.Fatalf("the refreshed policy must still load strictly: %v\n%s", err, raw)
	}
	for alias, want := range map[string]string{
		"sonnet": "anthropic/claude-sonnet-5-5", "claude-sonnet-5-5": "anthropic/claude-sonnet-5-5",
		"opus": "anthropic/claude-opus-5-5", "claude-opus-5-5": "anthropic/claude-opus-5-5",
		"claude-opus-5": "anthropic/claude-opus-5", "openrouter/x": "openrouter/x",
	} {
		if got := catalog.ModelAliases[alias].CanonicalModel; got != want {
			t.Errorf("alias %s = %q, want %q", alias, got, want)
		}
	}
	defaultRole := catalog.Roles["code.default"]
	if defaultRole.Model != "sonnet" || defaultRole.CanonicalModel != "anthropic/claude-sonnet-5-5" || !slices.Equal(defaultRole.Fallbacks, []string{"claude-sonnet-5-5"}) {
		t.Fatalf("role was not moved: %+v", defaultRole)
	}
	if got := catalog.Roles["code.smart"].Fallbacks; !slices.Equal(got, []string{"claude-fable-5-1", "claude-opus-5-5"}) {
		t.Fatalf("fallbacks: %v", got)
	}
	if got := catalog.Roles["code.fast"].Fallbacks; !slices.Equal(got, []string{"haiku"}) {
		t.Fatalf("an alias fallback must stay as written: %v", got)
	}
	if len(defaultRole.Models) != 2 || defaultRole.Models[1].Source != "openrouter" || defaultRole.Models[0].Effort != nil {
		t.Fatalf("declared sources must survive: %+v", defaultRole.Models)
	}
	if !slices.Equal(catalog.ExcludedModels, []string{"gpt-6-astra"}) {
		t.Fatalf("exclusions changed: %v", catalog.ExcludedModels)
	}
	for from, want := range map[string]string{"fable": "opus", "opus": "sonnet", "claude-fable-5-1": "claude-opus-5-5", "claude-opus-5-5": "claude-sonnet-5-5", "claude-opus-5": "claude-sonnet-5-5"} {
		if got := catalog.UsageTiers.ModelMap[from]; got != want {
			t.Errorf("tier %s = %q, want %q", from, got, want)
		}
	}
	tiers, err := usagemode.LoadTiers(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !tiers.LowerThan("claude-sonnet-5-5", "claude-fable-5-1") || !tiers.LowerThan("sonnet", "opus") {
		t.Fatal("the refreshed tiers must still order fable over opus over sonnet")
	}
	if err := tiers.ValidateTarget("claude-sonnet-5-5"); err != nil {
		t.Fatalf("usage modes must accept the new target: %v", err)
	}
	if catalog.Provenance.ObservedAt != "2026-10-30" || !strings.Contains(catalog.Provenance.Source, "2.1.312") {
		t.Fatalf("provenance: %+v", catalog.Provenance)
	}
	if len(result.Changes) == 0 || result.Evidence == "" {
		t.Fatalf("result: %+v", result)
	}
	if strings.Contains(string(raw), `>`) {
		t.Fatal("the policy must not be HTML-escaped")
	}

	// A second refresh has nothing left to move.
	again, err := RefreshCodingRoleCatalog(context.Background(), "claude-code", path, refreshNow)
	if err != nil || len(again.Changes) != 0 {
		t.Fatalf("refresh is not idempotent: %+v, %v", again.Changes, err)
	}
}

func TestRefreshFailsWithoutWritingWhenEvidenceIsMissing(t *testing.T) {
	removed := strings.Replace(oldClaudePolicy, `"model": "haiku", "canonical_model"`, `"model": "claude-gone-1", "canonical_model"`, 1)
	for _, tc := range []struct {
		name, runner, policy, script string
	}{
		{"binary has no catalog", "claude", oldClaudePolicy, "#!/bin/sh\necho 1.0.0\n"},
		{"catalog disagrees with itself", "claude", oldClaudePolicy, "#!/bin/sh\necho 1\nexit 0\n# " + claudeBakedCatalog + ` aliases:{opus:{default:"claude-opus-5"}}` + "\n"},
		{"alias names an unlisted model", "claude", oldClaudePolicy, "#!/bin/sh\necho 1\nexit 0\n# " + strings.Replace(claudeBakedCatalog, `sonnet:{default:"claude-sonnet-5-5"`, `sonnet:{default:"claude-sonnet-9"`, 1) + "\n"},
		{"a role uses a model that is gone", "claude", removed, "#!/bin/sh\necho 1\nexit 0\n# " + claudeBakedCatalog + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeFakeRunner(t, tc.runner, tc.script)
			path := writePolicy(t, tc.policy)
			_, err := RefreshCodingRoleCatalog(context.Background(), "claude-code", path, refreshNow)
			if !errors.Is(err, ErrRefreshEvidenceUnavailable) {
				t.Fatalf("got %v", err)
			}
			if data, _ := os.ReadFile(path); string(data) != tc.policy {
				t.Fatal("a failed refresh changed the policy")
			}
		})
	}
	t.Run("claude is not installed", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		path := writePolicy(t, oldClaudePolicy)
		if _, err := RefreshCodingRoleCatalog(context.Background(), "claude-code", path, refreshNow); !errors.Is(err, ErrRefreshEvidenceUnavailable) {
			t.Fatalf("got %v", err)
		}
	})
}

func codexPolicy(defaultModel string) string {
	role := func(model string) string {
		return `{"model": "` + model + `", "fallbacks": ["gpt-old"], "description": "d", "capabilities": ["code"]}`
	}
	return `{"schema_version": "v1", "runner": "codex", "provenance": {"source": "old", "observed_at": "2026-01-01"},
		"model_aliases": {"gpt-a": {"canonical_model": "codex/gpt-a", "provider": "codex"}},
		"roles": {"code.default": ` + role(defaultModel) + `, "code.fast": ` + role("gpt-a") + `, "code.smart": ` + role("gpt-a") + `, "code.cheap": ` + role("gpt-a") + `}}`
}

func TestRefreshedProvenanceKeepsTheRationaleWithoutStackingRefreshes(t *testing.T) {
	first := refreshedProvenance("Operator policy: Terra is the default", "codex 1", "")
	second := refreshedProvenance(first, "codex 2", "")
	if second != "policy refresh from codex 2; earlier: Operator policy: Terra is the default" {
		t.Fatalf("got %q", second)
	}
	if got := refreshedProvenance("policy refresh from codex 1", "codex 2", ""); got != "policy refresh from codex 2" {
		t.Fatalf("got %q", got)
	}
}

func TestRefreshConfirmsCodexRolesAndNamesNewModels(t *testing.T) {
	t.Setenv("VROOLI_CODEX_MODELS", `{"models":[{"slug":"gpt-a"},{"slug":"gpt-new"}],"source":"codex debug models"}`)
	path := writePolicy(t, codexPolicy("gpt-a"))
	result, err := RefreshCodingRoleCatalog(context.Background(), "codex", path, refreshNow)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := ReadCodingRoleCatalog("codex", path)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.ModelAliases["gpt-new"]; got.CanonicalModel != "codex/gpt-new" || got.Provider != "codex" {
		t.Fatalf("new model: %+v", got)
	}
	if catalog.Provenance.ObservedAt != "2026-10-30" {
		t.Fatalf("provenance: %+v", catalog.Provenance)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("a fallback the runner does not list must be reported")
	}
}

func TestRefreshRefusesACodexRoleTheRunnerDoesNotList(t *testing.T) {
	t.Setenv("VROOLI_CODEX_MODELS", `{"models":[{"slug":"gpt-a"}],"source":"codex debug models"}`)
	policy := codexPolicy("gpt-gone")
	path := writePolicy(t, policy)
	if _, err := RefreshCodingRoleCatalog(context.Background(), "codex", path, refreshNow); !errors.Is(err, ErrRefreshEvidenceUnavailable) {
		t.Fatalf("got %v", err)
	}
	if data, _ := os.ReadFile(path); string(data) != policy {
		t.Fatal("a failed refresh changed the policy")
	}
}

func TestRefreshRefusesARunnerWithNoEvidenceReader(t *testing.T) {
	policy := strings.ReplaceAll(codexPolicy("gpt-a"), `"runner": "codex"`, `"runner": "grok"`)
	path := writePolicy(t, policy)
	if _, err := RefreshCodingRoleCatalog(context.Background(), "grok", path, refreshNow); !errors.Is(err, ErrRefreshEvidenceUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestModelContextWindowResolvesAnAliasToItsCanonicalModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "windows.json")
	t.Setenv(harness.ModelWindowsEnv, path)
	if err := harness.RecordModelWindow(path, "claude-opus-5-5", 1000000, refreshNow); err != nil {
		t.Fatal(err)
	}
	claude := harness.Descriptor{Features: harness.Features{ContextWindowSource: harness.WindowSourceClaudeStatusline}}
	catalog := CodingRoleCatalog{ModelAliases: map[string]ModelAlias{
		"opus":   {CanonicalModel: "anthropic/claude-opus-5-5"},
		"sonnet": {CanonicalModel: "anthropic/claude-sonnet-5-5"},
	}}
	for model, want := range map[string]int64{"opus": 1000000, "claude-opus-5-5": 1000000, "sonnet": 0, "unlisted": 0} {
		if got := ModelContextWindow(claude, catalog, model); got != want {
			t.Errorf("%s: got %d, want %d", model, got, want)
		}
	}
}

func TestRefreshProvenanceNamesTheActorAndDoesNotStackEarlierRefreshes(t *testing.T) {
	first := refreshedProvenance("Operator policy", "codex 1", "agent codex session t")
	if first != "policy refresh by agent codex session t from codex 1; earlier: Operator policy" {
		t.Fatalf("first = %q", first)
	}
	second := refreshedProvenance(first, "codex 2", "operator op@example.com")
	if second != "policy refresh by operator op@example.com from codex 2; earlier: Operator policy" {
		t.Fatalf("second = %q", second)
	}
}
