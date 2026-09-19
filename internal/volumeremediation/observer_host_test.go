package volumeremediation

import (
	"context"
	"os"
	"testing"
)

// TestObserveRealDevice runs the production observer against a real device on
// the current host. Fixture tests prove the parsing; only this proves the paths
// and formats the observer expects are the ones the host actually publishes.
//
// It is opt-in because it needs a specific device:
//
//	VROOLI_VOLUME_TEST_DEVICE=/dev/sda1 go test ./internal/volumeremediation/ -run TestObserveRealDevice -v
//
// It is strictly read-only and never executes a remediation action.
func TestObserveRealDevice(t *testing.T) {
	device := os.Getenv("VROOLI_VOLUME_TEST_DEVICE")
	if device == "" {
		t.Skip("set VROOLI_VOLUME_TEST_DEVICE to run the host observation check")
	}

	state, err := NewHostObserver("").Observe(context.Background(), device)
	if err != nil {
		t.Fatalf("Observe(%s): %v", device, err)
	}

	if state.Device.Path != device {
		t.Fatalf("device path = %q, want %q", state.Device.Path, device)
	}
	if state.Device.TotalBytes <= 0 {
		t.Errorf("size was not observed; the block layer should publish one for %s", device)
	}
	if !state.Device.StableIdentity() {
		t.Errorf("no UUID or serial observed for %s; a mutating action would be refused", device)
	}
	if state.Mounted && state.Device.Filesystem == "" {
		t.Errorf("%s is mounted but no filesystem was observed", device)
	}
	t.Logf("device=%s mounted=%v read_only=%v fs=%q dirty=%s uuid=%q serial=%q size=%d evidence=%q observations=%v",
		state.Device.Path, state.Mounted, state.ReadOnly, state.Device.Filesystem,
		state.Dirty, state.Device.UUID, state.Device.Serial, state.Device.TotalBytes,
		state.Evidence, state.Observations)
}
