package portability

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/internal/deployability"
)

type fixturePlatformSource struct {
	items DerivedPlatformFleet
	err   error
}

func (s fixturePlatformSource) ListPlatformFleet(context.Context) (DerivedPlatformFleet, error) {
	return s.items, s.err
}

func TestFleetUsesSDAPlatformVerdicts(t *testing.T) {
	service := NewService(repoRoot(t), nil, fixturePlatformSource{items: DerivedPlatformFleet{Scenarios: []DerivedScenarioPlatformVerdict{{
		Scenario: "sample", HostOS: deployability.HostOSWindows, Status: "blocked", Reason: "redis does not resolve", BlockingDependency: "redis",
	}}}})
	readout, err := service.Fleet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !readout.Available || len(readout.BlockedByOS) != 1 {
		t.Fatalf("fleet readout = %#v, want one SDA-derived block", readout)
	}
	if readout.BlockedByOS[0].Dependencies[0].Name != "redis" {
		t.Fatalf("block = %#v, want redis", readout.BlockedByOS[0])
	}
}

func TestFleetReportsSDAUnavailableWithoutRecomputing(t *testing.T) {
	service := NewService(repoRoot(t), nil, fixturePlatformSource{err: context.DeadlineExceeded})
	readout, err := service.Fleet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if readout.Available || readout.Reason == "" {
		t.Fatalf("fleet readout = %#v, want visible unavailable source", readout)
	}
}

func TestServiceRefusesAnUnusableRootRatherThanServingAnEmptyReadout(t *testing.T) {
	service := NewService(t.TempDir(), nil, fixturePlatformSource{})
	if _, err := service.Grid(context.Background()); err == nil {
		t.Fatal("Grid returned a readout for a root with no capability vocabulary")
	} else if !IsUnresolvedRoot(err) {
		t.Fatalf("Grid returned %T, want UnresolvedRootError", err)
	}
	if _, err := service.Fleet(context.Background()); err == nil {
		t.Fatal("Fleet returned a readout for an unusable root")
	} else if !IsUnresolvedRoot(err) {
		t.Fatalf("Fleet returned %T, want UnresolvedRootError", err)
	}
}
