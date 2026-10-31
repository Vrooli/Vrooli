package snippets

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterLoadsEverySnippetBinding(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(nil, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(group.Subcommands), 4; got != want {
		t.Fatalf("subcommands = %d, want %d", got, want)
	}
}

func TestUnknownBindingFailsManifestLoad(t *testing.T) {
	manifest := map[string]any{
		"groups": []any{map[string]any{
			"name": GroupName,
			"commands": []any{map[string]any{
				"name":       "unknown",
				"binding":    map[string]any{"kind": "connect-rpc", "service": "SnippetsService", "method": "Unknown"},
				"governance": map[string]any{"effect": "read", "run_eligible": true},
			}},
		}},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cliapp.LoadFromManifest(raw, GroupName, map[string]func(cliapp.RunContext) error{}); err == nil {
		t.Fatal("expected unknown binding to fail")
	}
}

func TestBodyPresentation(t *testing.T) {
	body := "Check {{scenario}} then {{ scenario }} and {{owner_name}}.\nContinue."
	if got := distinctVariableCount(body); got != 2 {
		t.Fatalf("distinctVariableCount() = %d, want 2", got)
	}
	if got := bodyPreview("one\ntwo", 60); got != "one two" {
		t.Fatalf("bodyPreview() = %q", got)
	}
}
