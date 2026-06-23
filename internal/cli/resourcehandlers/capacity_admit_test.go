package resourcehandlers

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestAdmitResourceCapacityCLIGate verifies only start/restart admit; other
// actions are a pure no-op that never reaches the ledger.
func TestAdmitResourceCapacityCLIGate(t *testing.T) {
	for _, action := range []string{"stop", "logs", "status", "install"} {
		if _, ok := actionsAdmittingCapacity[action]; ok {
			t.Fatalf("action %q must not admit capacity", action)
		}
		var buf bytes.Buffer
		// A non-admitting action must short-circuit before touching disk/ledger,
		// so even a bogus root produces no output and no panic.
		admitResourceCapacityCLI("/nonexistent-root", "whatever", action, &buf)
		if buf.Len() != 0 {
			t.Fatalf("action %q produced output %q, want silent no-op", action, buf.String())
		}
	}
	for _, action := range []string{"start", "restart"} {
		if _, ok := actionsAdmittingCapacity[action]; !ok {
			t.Fatalf("action %q must admit capacity", action)
		}
	}
}

// TestAdmitResourceCapacityCLINoBlockIsSilent verifies that admitting a resource
// with no `capacity` block is a silent no-op — AdmitResource returns Skipped
// before opening the ledger, so the byte-identical-safety guarantee holds and no
// capacity.db is materialized for a non-adopter resource.
func TestAdmitResourceCapacityCLINoBlockIsSilent(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "no-capacity")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(`{"name":"no-capacity"}`), 0o644); err != nil {
		t.Fatalf("write resource.json: %v", err)
	}

	var buf bytes.Buffer
	admitResourceCapacityCLI(root, "no-capacity", "start", &buf)
	if buf.Len() != 0 {
		t.Fatalf("no-capacity-block admit produced output %q, want silent no-op", buf.String())
	}
}

// TestAdmitResourceCapacityCLIEnforceOffIsSilent verifies the explicit
// enforce=off escape hatch short-circuits before any disk/ledger access even
// when the resource declares a capacity block.
func TestAdmitResourceCapacityCLIEnforceOffIsSilent(t *testing.T) {
	t.Setenv("VROOLI_CAPACITY_ENFORCE", "off")
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "gpu-resident")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := `{"name":"gpu-resident","capacity":{"resource_kind":"vram","gpu_index":0,"preferred_bytes":1073741824,"priority":"service"}}`
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write resource.json: %v", err)
	}

	var buf bytes.Buffer
	admitResourceCapacityCLI(root, "gpu-resident", "restart", &buf)
	if buf.Len() != 0 {
		t.Fatalf("enforce=off admit produced output %q, want silent no-op", buf.String())
	}
}
