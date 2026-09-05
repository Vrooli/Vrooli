package smoketest

import (
	"context"
	"os"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// runDesktopJourneyCapability is the scenario entry point for semantic
// evidence. The shared runner owns request validation and result normalization;
// this scenario supplies the desktop driver and its capability fixtures.
func (s *DefaultService) runDesktopJourneyCapability(ctx context.Context, smokeTestID, scenarioName, platform string, rec recordingState, capability string) deliveryramp.JourneyResult {
	config := deliveryramp.ReadJourneyConfiguration(os.Getenv)
	target := deliveryramp.Target{
		ID: "local-" + strings.TrimSpace(platform), Label: "Local target", Ramp: "scenario-to-desktop",
		Platform: platform, OS: platform, DeviceKind: "host", Available: true,
		Capabilities: []string{capability, deliveryramp.CapabilityCDP, deliveryramp.CapabilityNativeWindow, deliveryramp.CapabilityProcessMetrics},
		Transport:    deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "local-" + strings.TrimSpace(platform), Available: true},
	}
	request := deliveryramp.JourneyExecutionRequest{
		RunID:  smokeTestID,
		Cell:   deliveryramp.Cell{ID: "cell-" + smokeTestID, Target: target, Capability: capability, ProfileID: config.Profile},
		Target: target,
		Plan:   deliveryramp.JourneyPlan{ID: capability + ".plan", Capability: capability, Profile: config.Profile},
	}
	return deliveryramp.JourneyRunner{Driver: desktopJourneyDriver{service: s, scenarioName: scenarioName, platform: platform, recording: rec, capability: capability}}.Run(ctx, request)
}

type desktopJourneyDriver struct {
	service      *DefaultService
	scenarioName string
	platform     string
	recording    recordingState
	capability   string
}

var _ deliveryramp.Driver = desktopJourneyDriver{}

func (d desktopJourneyDriver) Execute(ctx context.Context, request deliveryramp.DriverRequest) (deliveryramp.JourneyResult, error) {
	return d.service.runDesktopJourneyCapabilityLegacy(ctx, request.RunID, d.scenarioName, d.platform, d.recording, d.capability), nil
}
