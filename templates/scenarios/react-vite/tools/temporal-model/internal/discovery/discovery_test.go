package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindContractsIgnoresGeneratedAndDependencyDirs(t *testing.T) {
	root := t.TempDir()
	writeFlow(t, root, "api/example.flow.json", "example.visible")
	writeFlow(t, root, ".git/hidden.flow.json", "example.git")
	writeFlow(t, root, "node_modules/hidden.flow.json", "example.node")
	writeFlow(t, root, "dist/hidden.flow.json", "example.dist")
	writeFlow(t, root, "build/hidden.flow.json", "example.build")
	writeFlow(t, root, "coverage/hidden.flow.json", "example.coverage")
	writeFlow(t, root, "_apalache-out/hidden.flow.json", "example.apalache")

	contracts, err := FindContracts(root)
	if err != nil {
		t.Fatalf("FindContracts() error = %v", err)
	}
	if got, want := len(contracts), 1; got != want {
		t.Fatalf("contracts = %d, want %d", got, want)
	}
	if contracts[0].FlowID != "example.visible" {
		t.Fatalf("flow id = %s", contracts[0].FlowID)
	}
}

func writeFlow(t *testing.T, root string, rel string, flowID string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
  "schemaVersion": 2,
  "flowId": "` + flowID + `",
  "domain": "example",
  "description": "Example.",
  "model": {
    "module": "Example",
    "seed": "1",
    "maxSteps": 1,
    "traceCount": 1,
    "verify": { "invariants": ["TypeOK"] }
  },
  "outputs": { "modelPath": "model.qnt", "artifactPath": "model.formal.generated.json" },
  "states": [{ "id": "idle", "quint": "Idle", "initial": true }],
  "events": [{ "id": "tick", "quint": "Tick" }],
  "transitionDefaults": { "invalid": { "to": "self", "wantError": true } },
  "transitions": [{ "from": "idle", "event": "tick", "to": "self", "wantError": true }],
  "invariants": [{ "id": "type_ok", "quint": "TypeOK", "description": "Type OK." }],
  "traces": [{ "name": "idle", "initial": "idle", "steps": [] }]
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
