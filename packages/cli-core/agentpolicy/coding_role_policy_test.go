package agentpolicy

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
	if got.Enforcement.Permissions != "intent_only" {
		t.Fatalf("posture = %#v", got.Enforcement)
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

func writeCodingCatalog(t *testing.T, runner string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model-policy.json")
	roles := map[string]map[string]any{}
	for _, role := range []string{"code.default", "code.fast", "code.smart", "code.cheap"} {
		roles[role] = map[string]any{"model": "test-" + role[5:], "description": role, "capabilities": []string{"code", "tools"}}
	}
	data, err := json.Marshal(map[string]any{"schema_version": "v1", "runner": runner, "provenance": map[string]string{"source": "test", "observed_at": "2026-07-10"}, "roles": roles})
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
