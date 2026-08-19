package agentharness

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

func TestCodingPolicyCommandsResolveAndReportPosture(t *testing.T) {
	path := writeCodingCatalog(t, "codex")
	var stdout, stderr bytes.Buffer
	group := CodingPolicyCommands(CodingPolicyConfig{Runner: "codex", CatalogPath: path, Posture: EnforcementPosture{Permissions: "intent_only"}, Stdout: &stdout, Stderr: &stderr})
	if err := command(group, "resolve").Run([]string{"--role", "code.default", "--json"}); err != nil {
		t.Fatalf("resolve() error = %v; stderr=%s", err, stderr.String())
	}
	var got codingRoleResponse
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Runner != "codex" || got.Role != "code.default" || got.Model != "test-default" {
		t.Fatalf("unexpected resolution: %#v", got)
	}
	if got.CanonicalModel != "vendor/test-default" {
		t.Fatalf("canonical model = %q", got.CanonicalModel)
	}
	if got.Enforcement.Permissions != "intent_only" {
		t.Fatalf("posture = %#v", got.Enforcement)
	}
}

func TestModelResolveUsesResourceOwnedAlias(t *testing.T) {
	path := writeCodingCatalog(t, "future-runner")
	resolution, err := ResolveCatalogModel("future-runner", path, "future-model")
	if err != nil || resolution.CanonicalModel != "vendor/future-model" || resolution.Provider != "future-pricing" {
		t.Fatalf("resolution=%+v err=%v", resolution, err)
	}
	group := ModelDiscoveryCommands(ModelDiscoveryConfig{Runner: "future-runner", CatalogPath: path})
	if err := command(group, "resolve").Run([]string{"--model", "future-model", "--json"}); err != nil {
		t.Fatalf("model resolve: %v", err)
	}
}

func TestCodingPolicyCommandsRejectUnknownRoleAndMalformedCatalog(t *testing.T) {
	path := writeCodingCatalog(t, "codex")
	group := CodingPolicyCommands(CodingPolicyConfig{Runner: "codex", CatalogPath: path})
	if err := command(group, "resolve").Run([]string{"--role", "code.unknown"}); err == nil {
		t.Fatal("unknown role unexpectedly succeeded")
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":"v1","runner":"codex","roles":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command(group, "validate").Run(nil); err == nil {
		t.Fatal("malformed catalog unexpectedly validated")
	}
}

func TestCodingPolicyAllowsWritingNamespaceAndRejectsUnknownNamespace(t *testing.T) {
	path := writeCodingCatalog(t, "codex")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var catalog map[string]any
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	roles := catalog["roles"].(map[string]any)
	roles["write.default"] = map[string]any{"model": "writer", "description": "writer", "capabilities": []string{"writing"}}
	data, err = json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	group := CodingPolicyCommands(CodingPolicyConfig{Runner: "codex", CatalogPath: path})
	if err := command(group, "validate").Run(nil); err != nil {
		t.Fatalf("write namespace rejected: %v", err)
	}
	roles["image.default"] = map[string]any{"model": "image", "description": "image", "capabilities": []string{"image"}}
	data, err = json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command(group, "validate").Run(nil); err == nil {
		t.Fatal("unknown namespace unexpectedly validated")
	}
}

func TestCatalogStalenessBudgetAndLivePrimaryFailure(t *testing.T) {
	now := time.Date(2026, time.August, 4, 0, 0, 0, 0, time.UTC)
	warning := catalogStalenessFindings(CodingRoleCatalog{StalenessBudgetDays: 14, Provenance: CatalogProvenance{ObservedAt: "2026-07-20"}}, now, false)
	if len(warning) != 1 || warning[0].Severity != "warning" {
		t.Fatalf("warning findings = %#v", warning)
	}
	hard := catalogStalenessFindings(CodingRoleCatalog{StalenessBudgetDays: 14, Provenance: CatalogProvenance{ObservedAt: "2026-06-01"}}, now, true)
	if len(hard) != 1 || hard[0].Severity != "error" {
		t.Fatalf("hard findings = %#v", hard)
	}
	if _, err := parseObservedAt("not-a-date"); err == nil {
		t.Fatal("invalid observed_at accepted")
	}
	findings := liveCatalogFindings(CodingRoleCatalog{Roles: map[string]CodingRole{"code.default": {Model: "missing"}}}, LiveModelCatalog{Models: []string{"present"}})
	if len(findings) != 2 || findings[0].Type != "missing_primary_model" {
		t.Fatalf("live findings = %#v", findings)
	}
	if !errors.Is(&PolicyValidationError{Code: "discovery_unavailable", Err: ErrModelDiscoveryUnavailable}, ErrModelDiscoveryUnavailable) {
		t.Fatal("typed discovery error no longer unwraps")
	}
}

func writeCodingCatalog(t *testing.T, runner string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model-policy.json")
	roles := map[string]map[string]any{}
	for _, role := range []string{"code.default", "code.fast", "code.smart", "code.cheap"} {
		roles[role] = map[string]any{"model": "test-" + role[5:], "canonical_model": "vendor/test-" + role[5:], "description": role, "capabilities": []string{"code", "tools"}}
	}
	data, err := json.Marshal(map[string]any{"schema_version": "v1", "runner": runner, "provenance": map[string]string{"source": "test", "observed_at": "2026-07-10"}, "model_aliases": map[string]any{"future-model": map[string]string{"canonical_model": "vendor/future-model", "provider": "future-pricing"}}, "roles": roles})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func command(group cliapp.SubcommandGroup, name string) cliapp.Command {
	for _, candidate := range group.Subcommands {
		if candidate.Name == name {
			return candidate
		}
	}
	panic("missing command " + name)
}
