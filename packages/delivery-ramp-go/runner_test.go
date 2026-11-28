package deliveryramp

import (
	"context"
	"errors"
	"testing"
)

type fakeJourneyDriver struct {
	result JourneyResult
	err    error
}

func (f fakeJourneyDriver) Execute(context.Context, DriverRequest) (JourneyResult, error) {
	return f.result, f.err
}

func readyJourneyRequest() JourneyExecutionRequest {
	return JourneyExecutionRequest{
		RunID: "run-1", Plan: JourneyPlan{SchemaVersion: JourneyEvidenceVersion, ID: "plan-1", Capability: CapabilityCDP, Profile: "normal-review"},
		Target: Target{ID: "local-linux-amd64", Platform: "desktop", Transport: Transport{Kind: TransportLocal}, Capabilities: []string{CapabilityCDP}, Available: true},
	}
}

func TestJourneyRunnerNormalizesDriverResult(t *testing.T) {
	request := readyJourneyRequest()
	result := (JourneyRunner{Driver: fakeJourneyDriver{result: JourneyResult{Disposition: DispositionPass}}}).Run(context.Background(), request)
	if result.Disposition != DispositionPass || result.SchemaVersion != JourneySchemaVersion || result.EvidenceVersion != JourneyEvidenceVersion || result.TargetID != request.Target.ID {
		t.Fatalf("normalized journey result = %+v", result)
	}
}

func TestJourneyRunnerFailsClosedForMissingCapabilityAndCancellation(t *testing.T) {
	request := readyJourneyRequest()
	request.Plan.Capability = CapabilityNativeWindow
	result := (JourneyRunner{Driver: fakeJourneyDriver{}}).Run(context.Background(), request)
	if result.Disposition != DispositionUnavailable {
		t.Fatalf("missing capability disposition = %q", result.Disposition)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request = readyJourneyRequest()
	result = (JourneyRunner{Driver: fakeJourneyDriver{err: errors.New("cancelled")}}).Run(ctx, request)
	if result.Disposition != DispositionNotRun {
		t.Fatalf("cancelled disposition = %q", result.Disposition)
	}
}

func TestJourneyConfigurationPreservesExistingEnvironmentNames(t *testing.T) {
	config := ReadJourneyConfiguration(func(key string) string {
		if key == JourneyCapabilityEnv {
			return " hello-desktop "
		}
		return " normal-review "
	})
	if config.Capability != "hello-desktop" || config.Profile != "normal-review" {
		t.Fatalf("config = %+v", config)
	}
}
