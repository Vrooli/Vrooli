package portability

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/deployability"
)

func TestFleetReportsTheManifestRootItComputedAgainst(t *testing.T) {
	readout, err := liveReader(t).Fleet(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if readout.ManifestRoot != repoRoot(t) {
		t.Fatalf("fleet reports manifest root %q, want %q", readout.ManifestRoot, repoRoot(t))
	}
	if readout.ComputedAt.IsZero() {
		t.Fatal("fleet carries no computed_at")
	}
	if readout.DesktopBundling.Resources == 0 {
		t.Fatal("fleet counted zero resources; the desktop bundling verdict would be vacuous")
	}
	if readout.DesktopBundling.Reason == "" {
		t.Fatal("the desktop bundling verdict carries no reason")
	}
}

// TestFleetPeerlessAgreesWithTheGrid pins the reason the fleet view reuses the
// grid instead of resolving capabilities a second time: two resolutions of the
// same manifests can disagree, and then neither is the instrument's answer.
func TestFleetPeerlessAgreesWithTheGrid(t *testing.T) {
	reader := liveReader(t)
	now := time.Now()
	grid, err := reader.Grid(now)
	if err != nil {
		t.Fatal(err)
	}
	readout, err := reader.Fleet(now)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range readout.Peerless {
		for _, capability := range item.Capabilities {
			entry, ok := grid.Capability(capability)
			if !ok {
				t.Errorf("fleet reports %s peerless on %s, but the grid has no such capability", capability, item.HostOS)
				continue
			}
			platform, ok := entry.Platform(item.HostOS)
			if !ok {
				t.Errorf("grid row %s has no %s entry", capability, item.HostOS)
				continue
			}
			if platform.Status != deployability.CapabilityPeerless {
				t.Errorf("fleet reports %s peerless on %s but the grid resolves it %s", capability, item.HostOS, platform.Status)
			}
		}
	}
}

func TestServiceRefusesAnUnusableRootRatherThanServingAnEmptyReadout(t *testing.T) {
	service := NewService(t.TempDir(), nil)
	if _, err := service.Grid(context.Background()); err == nil {
		t.Fatal("Grid returned a readout for a root with no capability vocabulary")
	} else if !IsUnresolvedRoot(err) {
		t.Fatalf("Grid returned %T, want UnresolvedRootError", err)
	}
	if _, err := service.Fleet(context.Background()); err == nil {
		t.Fatal("Fleet returned a readout for a root with no capability vocabulary")
	} else if !IsUnresolvedRoot(err) {
		t.Fatalf("Fleet returned %T, want UnresolvedRootError", err)
	}
}
