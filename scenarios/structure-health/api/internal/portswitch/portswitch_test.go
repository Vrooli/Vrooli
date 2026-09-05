package portswitch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeScenario writes a minimal .vrooli/service.json for a scenario under root
// and returns the scenario root dir.
func writeScenario(t *testing.T, scenariosRoot, name, serviceJSON string) string {
	t.Helper()
	dir := filepath.Join(scenariosRoot, name, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(scenariosRoot, name)
}

func uiPort(t *testing.T, scenarioRoot string) (port int, hasRange bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(scenarioRoot, ".vrooli", "service.json"))
	if err != nil {
		t.Fatal(err)
	}
	var svc struct {
		Ports map[string]struct {
			Port  int    `json:"port"`
			Range string `json:"range"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(raw, &svc); err != nil {
		t.Fatal(err)
	}
	return svc.Ports["ui"].Port, svc.Ports["ui"].Range != ""
}

const rangedUI = `{
  "name": "target",
  "ports": {
    "ui": { "env_var": "UI_PORT", "range": "20000-24999" }
  }
}
`

func TestAssignFixed_PicksFreeInBandPortAvoidingSiblings(t *testing.T) {
	root := t.TempDir()
	target := writeScenario(t, root, "target", rangedUI)
	// A sibling already pins the lowest band port (20000) — assign must skip it.
	writeScenario(t, root, "sibling", `{"name":"sibling","ports":{"ui":{"env_var":"UI_PORT","port":20000}}}`)

	res, err := AssignFixed(target, "ui", true, nil)
	if err != nil {
		t.Fatalf("AssignFixed: %v", err)
	}
	if !res.Changed {
		t.Fatalf("expected a change, got no-op (%s)", res.Message)
	}
	if res.AssignedPort != 20001 {
		t.Errorf("assigned port = %d, want 20001 (lowest free in band, 20000 taken)", res.AssignedPort)
	}
	// Applied to disk: now fixed, range dropped.
	port, hasRange := uiPort(t, target)
	if port != 20001 || hasRange {
		t.Errorf("on-disk ui port=%d hasRange=%v, want 20001 / false", port, hasRange)
	}
}

func TestAssignFixed_AvoidsLiveListeners(t *testing.T) {
	root := t.TempDir()
	target := writeScenario(t, root, "target", rangedUI)
	// Simulate 20000 and 20001 already listening.
	listening := func(p int) bool { return p == 20000 || p == 20001 }

	res, err := AssignFixed(target, "ui", false, listening)
	if err != nil {
		t.Fatalf("AssignFixed: %v", err)
	}
	if res.AssignedPort != 20002 {
		t.Errorf("assigned port = %d, want 20002 (20000/20001 listening)", res.AssignedPort)
	}
	if res.Applied {
		t.Error("preview must not write")
	}
}

func TestAssignFixed_PreviewMatchesApply(t *testing.T) {
	root := t.TempDir()
	target := writeScenario(t, root, "target", rangedUI)

	preview, err := AssignFixed(target, "ui", false, nil)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Applied {
		t.Fatal("preview must not apply")
	}
	// Disk unchanged after preview.
	if p, hasRange := uiPort(t, target); p != 0 || !hasRange {
		t.Fatalf("preview mutated disk: port=%d hasRange=%v", p, hasRange)
	}
	apply, err := AssignFixed(target, "ui", true, nil)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if preview.After != apply.After {
		t.Errorf("preview/apply diverged:\npreview=%s\napply=%s", preview.After, apply.After)
	}
	if !apply.Applied {
		t.Error("apply must write")
	}
}

func TestAssignFixed_IdempotentWhenAlreadyFixed(t *testing.T) {
	root := t.TempDir()
	target := writeScenario(t, root, "target", `{"name":"target","ports":{"ui":{"env_var":"UI_PORT","port":21242}}}`)

	res, err := AssignFixed(target, "ui", true, nil)
	if err != nil {
		t.Fatalf("AssignFixed: %v", err)
	}
	if res.Changed {
		t.Errorf("already-fixed port must be a no-op, got change to %d", res.AssignedPort)
	}
	if res.AssignedPort != 21242 {
		t.Errorf("reported port = %d, want existing 21242", res.AssignedPort)
	}
}

func TestReleaseFixed_RevertsToRange(t *testing.T) {
	root := t.TempDir()
	target := writeScenario(t, root, "target", `{"name":"target","ports":{"ui":{"env_var":"UI_PORT","port":21242}}}`)

	res, err := ReleaseFixed(target, "ui", true)
	if err != nil {
		t.Fatalf("ReleaseFixed: %v", err)
	}
	if !res.Changed || res.PreviousPort != 21242 {
		t.Errorf("expected release of 21242, got changed=%v prev=%d", res.Changed, res.PreviousPort)
	}
	port, hasRange := uiPort(t, target)
	if port != 0 || !hasRange {
		t.Errorf("after release: port=%d hasRange=%v, want 0 / true (ranged)", port, hasRange)
	}
}

func TestReleaseFixed_IdempotentWhenAlreadyRanged(t *testing.T) {
	root := t.TempDir()
	target := writeScenario(t, root, "target", rangedUI)
	res, err := ReleaseFixed(target, "ui", true)
	if err != nil {
		t.Fatalf("ReleaseFixed: %v", err)
	}
	if res.Changed {
		t.Error("already-ranged port must be a no-op")
	}
}
