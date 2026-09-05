package resources

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLivePlacementOnThisHost records what the placement verifier observes for
// the accelerated resources actually running on this machine. It is evidence
// capture, not an assertion about any particular host: it is skipped unless
// VROOLI_ACCEL_LIVE is set, so it never fails on a host with no accelerator or
// no running resources.
func TestLivePlacementOnThisHost(t *testing.T) {
	if os.Getenv("VROOLI_ACCEL_LIVE") == "" {
		t.Skip("set VROOLI_ACCEL_LIVE to capture live placement evidence")
	}
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	root = root + "/../.."
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("home directory: %v", err)
	}
	controller := NewController(root, home)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, name := range []string{"ollama", "whisper", "reranker", "kokoro"} {
		manifest, err := controller.ResourceManifest(name)
		if err != nil {
			t.Logf("%s: manifest unavailable: %v", name, err)
			continue
		}
		spec, accelerated := accelSpecFor(manifest)
		if !accelerated {
			t.Logf("%s: declares no accelerator (backends=%v)", name, spec.Backends)
			continue
		}
		placement, err := observePlacement(ctx, controller, manifest)
		if err != nil {
			t.Logf("%s: verify failed: %v", name, err)
			continue
		}
		if placement == nil {
			t.Logf("%s: nothing to verify", name)
			continue
		}
		t.Logf("%s: declared=%s observed=%s state=%s target=%s reason=%s",
			name, placement.Declared, placement.Observed, placement.State, placement.Target, placement.Reason)
	}
}
