package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunListAndExplainUseDiscoveredContracts(t *testing.T) {
	root := t.TempDir()
	writeFlow(t, root, "api/example.flow.json", "example.visible")
	writeBinding(t, root, "api/example_test.go", "TestExample_ReplaysFormalModelArtifacts")

	var stdout bytes.Buffer
	if err := Run(context.Background(), []string{"list", "--root", root}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(list) error = %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "example.visible" {
		t.Fatalf("list output = %q", got)
	}

	stdout.Reset()
	if err := Run(context.Background(), []string{"explain", "--root", root, "--flow", "example.visible"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(explain) error = %v", err)
	}
	for _, want := range []string{
		"flow: example.visible",
		"Generated files:",
		"Runtime:",
		"Topology:",
		"states: 1 (initial idle; terminal none)",
		"events: 1",
		"expanded transitions: 1",
		"Replay bindings:",
		"Coverage requirements:",
		"named traces: 1",
		"Commands:",
		"Hand-authored follow-up files:",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("explain output = %q, want %q", stdout.String(), want)
		}
	}
}

func TestRunRejectsUnknownFlow(t *testing.T) {
	root := t.TempDir()
	writeFlow(t, root, "api/example.flow.json", "example.visible")
	writeBinding(t, root, "api/example_test.go", "TestExample_ReplaysFormalModelArtifacts")

	err := Run(context.Background(), []string{"list", "--root", root, "--flow", "missing.flow"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown flow id missing.flow") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestParseFlagsRequiresValues(t *testing.T) {
	if _, err := parseFlags([]string{"--root"}); err == nil {
		t.Fatal("expected --root without value to fail")
	}
	if _, err := parseFlags([]string{"--flow"}); err == nil {
		t.Fatal("expected --flow without value to fail")
	}
}

func writeFlow(t *testing.T, root string, rel string, flowID string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{
  "schemaVersion": 3,
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
  "outputs": { "modelPath": "model.qnt", "artifactPath": "model.formal.generated.json", "declarationsPath": "api/example.generated.go" },
  "states": [{ "id": "idle", "quint": "Idle", "initial": true }],
  "events": [{ "id": "tick", "quint": "Tick" }],
  "transitionDefaults": { "invalid": { "to": "self", "wantError": true } },
  "transitions": [{ "from": "idle", "event": "tick", "to": "self", "wantError": true }],
  "invariants": [{ "id": "type_ok", "quint": "TypeOK", "description": "Type OK." }],
  "traces": [{ "name": "idle", "initial": "idle", "steps": [] }],
  "runtime": {
    "go": { "package": "api", "statusType": "Status", "eventType": "Event", "constantPrefix": "Example" }
  },
  "replay": {
    "bindings": [{ "kind": "go-test", "path": "api/example_test.go", "assertion": "TestExample_ReplaysFormalModelArtifacts" }]
  }
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeBinding(t *testing.T, root string, rel string, marker string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := marker + "\nAssertFormalArtifactFresh\nAssertFormalTransitionsReplay\nAssertFormalTracesReplay\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
