package resourcecli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/resources"
)

// TestWriteListJSONContract pins the `vrooli resource list --json` wire shape so
// a change to the vrooli.cli.v1 proto (or the producer mapping) is caught here
// rather than silently breaking out-of-process consumers (swarm-manager, UI).
func TestWriteListJSONContract(t *testing.T) {
	items := []resources.Resource{
		// A fully-populated resource.
		{
			Name:         "redis",
			Path:         "/r/redis",
			Exists:       true,
			Registered:   true,
			Enabled:      true,
			Required:     true,
			DeclaresCLI:  true,
			CLIInstalled: true,
			Config:       resources.ConfigEntry{Enabled: true, Required: true, Description: "cache"},
			ControlMode:  "manifest-native",
			Driver:       "managed-service",
			ManifestPath: "/r/redis/resource.json",
		},
		// A sparse resource: every optional field empty. EmitUnpopulated means
		// the contract is fully specified — these still appear, as "" / false.
		{Name: "gone", Path: "/r/gone"},
	}
	failures := []discovery.Failure{
		{Kind: "resource", Name: "broken", Path: "/r/broken", Stage: "load", Error: "boom"},
	}

	var buf bytes.Buffer
	if err := WriteList(&buf, cliout.FormatJSON, items, failures); err != nil {
		t.Fatalf("WriteList: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}

	// Envelope.
	if got["success"] != true {
		t.Errorf("success: want true, got %v", got["success"])
	}
	res, ok := got["resources"].([]any)
	if !ok || len(res) != 2 {
		t.Fatalf("resources: want 2 entries, got %v", got["resources"])
	}

	// snake_case field names (UseProtoNames=true) — a camelCase regression here
	// would silently break every consumer that reads the CLI state/control fields.
	first := res[0].(map[string]any)
	for _, key := range []string{
		"name", "path", "exists", "registered", "enabled", "required",
		"declares_cli", "cli_installed", "cli_state_reason", "config", "control_mode", "driver", "template",
		"portability_tier", "manifest_path",
	} {
		if _, ok := first[key]; !ok {
			t.Errorf("first resource missing key %q (camelCase regression?)", key)
		}
	}
	if first["declares_cli"] != true || first["cli_installed"] != true || first["name"] != "redis" {
		t.Errorf("first resource value mismatch: %v", first)
	}
	cfg, ok := first["config"].(map[string]any)
	if !ok || cfg["description"] != "cache" {
		t.Errorf("nested config not mapped: %v", first["config"])
	}

	// Fully-specified output: the sparse resource still carries every field.
	sparse := res[1].(map[string]any)
	if sparse["driver"] != "" || sparse["enabled"] != false {
		t.Errorf("EmitUnpopulated regression: sparse resource dropped zero-valued fields: %v", sparse)
	}

	// Discovery failures are carried through.
	fails, ok := got["discovery_failures"].([]any)
	if !ok || len(fails) != 1 {
		t.Fatalf("discovery_failures: want 1, got %v", got["discovery_failures"])
	}
	if f := fails[0].(map[string]any); f["error"] != "boom" || f["name"] != "broken" {
		t.Errorf("failure not mapped: %v", f)
	}
}
