package readiness

import (
	"context"
	"errors"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestAppleReadinessDerivesSixRungsAndUnavailability(t *testing.T) {
	ladder := FromProbe(Probe{})
	if err := ladder.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(ladder.Rungs) != 6 {
		t.Fatal(len(ladder.Rungs))
	}
	for _, rung := range ladder.Rungs {
		if rung.State != Unavailable || rung.NextAction == "" || rung.MissingCapability == "" {
			t.Fatalf("rung = %+v", rung)
		}
	}
}

func TestAppleReadinessMovesWhenProbeChanges(t *testing.T) {
	ladder := FromProbe(Probe{DeveloperProgram: true, VerifiedIdentity: true, MacOSBuildHost: true, SigningReference: true, TestFlightAccess: true, AppStoreListing: true})
	if err := ladder.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, rung := range ladder.Rungs {
		if rung.State != Ready {
			t.Fatalf("rung = %+v", rung)
		}
	}
}

// The build-host rung must reflect live fleet state. A remembered environment
// flag can claim a macOS host that is not there, which is what the ladder
// exists to prevent.
func TestBuildHostRungIsDerivedFromLiveInventory(t *testing.T) {
	tests := []struct {
		name      string
		inventory deliveryramp.Inventory
		err       error
		wantReady bool
	}{
		{
			name: "available ios target satisfies the rung",
			inventory: deliveryramp.Inventory{Targets: []deliveryramp.Target{
				{ID: "bridge:mac-1", Platform: "ios", Available: true},
			}},
			wantReady: true,
		},
		{
			name: "a reachable macOS node without a usable toolchain does not",
			inventory: deliveryramp.Inventory{Targets: []deliveryramp.Target{
				{ID: "bridge:mac-1", Platform: "ios", Available: false},
			}},
			wantReady: false,
		},
		{
			name: "an available non-ios target does not satisfy an iOS rung",
			inventory: deliveryramp.Inventory{Targets: []deliveryramp.Target{
				{ID: "bridge:linux-1", Platform: "android", Available: true},
			}},
			wantReady: false,
		},
		{name: "discovery failure is not readiness", err: errors.New("bridge unavailable"), wantReady: false},
		{name: "empty fleet is not readiness", wantReady: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := BuildHostObserver(func(context.Context) (deliveryramp.Inventory, error) {
				return tt.inventory, tt.err
			})
			ladder := FromProbeContext(context.Background(), Probe{ObserveBuildHost: observer})

			var rung Rung
			for _, candidate := range ladder.Rungs {
				if candidate.ID == "macos-build-host" {
					rung = candidate
				}
			}
			if tt.wantReady && rung.State != Ready {
				t.Fatalf("expected a ready build-host rung, got %#v", rung)
			}
			if !tt.wantReady {
				if rung.State != Unavailable {
					t.Fatalf("expected an unavailable build-host rung, got %#v", rung)
				}
				if rung.MissingCapability == "" || rung.NextAction == "" {
					t.Fatalf("an unavailable rung must stay actionable: %#v", rung)
				}
			}
		})
	}
}

// A live observation must win over a stale resolved flag, so the ladder cannot
// report a host that the fleet no longer has.
func TestLiveObservationOverridesRememberedBuildHostFlag(t *testing.T) {
	observer := BuildHostObserver(func(context.Context) (deliveryramp.Inventory, error) {
		return deliveryramp.Inventory{}, nil
	})
	ladder := FromProbeContext(context.Background(), Probe{MacOSBuildHost: true, ObserveBuildHost: observer})

	for _, rung := range ladder.Rungs {
		if rung.ID == "macos-build-host" && rung.State == Ready {
			t.Fatal("a remembered flag must not outrank a live observation")
		}
	}
}

func TestBuildHostObserverIsNilWithoutDiscovery(t *testing.T) {
	if BuildHostObserver(nil) != nil {
		t.Fatal("a nil discovery function must not produce an observer")
	}
}
