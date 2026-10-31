package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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

	got := (&Server{}).GetRoleModelChoices(t.Context(), "opencode", "local")
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
			if choice.LaunchModel != "ollama/"+choice.Model {
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

	got := (&Server{paidCodingProtocolQualified: true}).GetRoleModelChoices(t.Context(), "opencode", "local")
	for _, choice := range got.Choices {
		if choice.Source == "vrooli-credits" {
			if choice.Model != "coding" || choice.LaunchModel != "vrooli/coding" || choice.SourceLabel != "Vrooli credits" {
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

	stale := (&Server{}).GetRoleModelChoices(t.Context(), "opencode", "local")
	if stale.PolicyStatus != "stale" || stale.Availability != "unknown" || len(stale.Choices) == 0 {
		t.Fatalf("stale policy evidence was lost: %+v", stale)
	}
	remote := (&Server{}).GetRoleModelChoices(t.Context(), "opencode", "remote:one")
	if remote.PolicyStatus != "unsupported" || len(remote.Choices) != 0 {
		t.Fatalf("remote target should not inherit local choices: %+v", remote)
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

func TestNativeModelTranslationDoesNotAdvertiseUnqualifiedVrooliFunding(t *testing.T) {
	for _, runner := range []string{"codex", "claude-code", "opencode"} {
		sources, _, _, supported := nativeModelTranslation(runner)
		if !supported {
			t.Fatalf("runner %q should have a native model mapping", runner)
		}
		if _, ok := sources["vrooli-credits"]; ok {
			t.Errorf("runner %q exposes Vrooli-funded model choices before coding execution and accounting are qualified", runner)
		}
	}
}
