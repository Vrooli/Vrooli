package readiness

import "testing"

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
