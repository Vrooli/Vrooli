// Package journeys owns the scenario-agnostic iOS conformance chapters.
package journeys

import (
	"context"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

const IOSCapability = "ios-native-runtime"

const capabilitySimctl = "simctl"

// Plan returns the twelve chapters required for a generated app's native
// contract. Each chapter names the capability that must be proven.
func Plan() deliveryramp.JourneyPlan {
	chapters := []struct{ id, purpose, action, capability string }{
		{"install-cold-start", "install and render the generated app", "install-and-launch", capabilitySimctl},
		{"permission-denial", "continue safely after a denied permission", "deny-permission", capabilitySimctl},
		{"background-resume", "restore state after backgrounding", "background-resume", capabilitySimctl},
		{"process-death-restore", "restore persisted state after process death", "kill-and-restore", capabilitySimctl},
		{"rotation", "preserve layout across rotation", "rotate", capabilitySimctl},
		{"keyboard-avoidance", "keep focused fields visible above the keyboard", "keyboard", capabilitySimctl},
		{"offline-transition", "report offline behavior without corrupting state", "offline", capabilitySimctl},
		{"deep-link", "open the declared deep link route", "deep-link", capabilitySimctl},
		{"notification-tap", "open the route selected by a notification", "notification-tap", capabilitySimctl},
		{"back-navigation", "return through native back navigation", "back", capabilitySimctl},
		{"update-migration", "retain state across an update", "update", capabilitySimctl},
		{"clean-uninstall", "remove app state on uninstall", "uninstall", capabilitySimctl},
	}
	readiness := deliveryramp.ReadinessPolicy{ID: "ios-bounded-ready", Reason: "bounded native readiness", Timeout: 30 * time.Second, PollInterval: 100 * time.Millisecond, StabilityCount: 2, Cancellation: "context cancellation"}
	settle := deliveryramp.SettlePolicy{ID: "ios-review-settle", Reason: "allow reviewed UI state to stabilize", Minimum: 250 * time.Millisecond, Maximum: 3 * time.Second, PollInterval: 50 * time.Millisecond, Cancellation: "context cancellation"}
	steps := make([]deliveryramp.JourneyStepSpec, 0, len(chapters))
	for _, chapter := range chapters {
		steps = append(steps, deliveryramp.JourneyStepSpec{ID: chapter.id, Purpose: chapter.purpose, Action: chapter.action, Capture: true, Readiness: readiness, Settle: settle, Arguments: map[string]string{"required_capability": chapter.capability}, Assertion: &deliveryramp.AssertionSpec{ID: chapter.id + "-assertion", Expected: "native chapter completed", Description: chapter.purpose}})
	}
	return deliveryramp.JourneyPlan{SchemaVersion: deliveryramp.JourneyEvidenceVersion, ID: "scenario-to-ios.conformance.v1", Capability: IOSCapability, Purpose: "prove generated scenario behavior on an Apple runtime", Profile: "normal-review", Steps: steps}
}

// Driver composes the platform journey. The actual device verbs remain owned
// by device-control; this Linux implementation reports honest unavailability.
type Driver struct{ GOOS string }

var _ deliveryramp.Driver = Driver{}

// Execute returns terminal unavailable steps when Apple runtime capability is
// absent. It never upgrades dispatch or compilation into a synthetic pass.
func (d Driver) Execute(ctx context.Context, request deliveryramp.DriverRequest) (deliveryramp.JourneyResult, error) {
	result := deliveryramp.JourneyResult{SchemaVersion: deliveryramp.JourneySchemaVersion, EvidenceVersion: deliveryramp.JourneyEvidenceVersion, SmokeTestID: request.RunID, PlanID: request.Plan.ID, Profile: request.Plan.Profile, Capability: request.Plan.Capability, Platform: "ios", Disposition: deliveryramp.DispositionUnavailable, DegradedReason: "Apple runtime is unavailable; no simulator or WebDriverAgent is registered"}
	if ctx == nil {
		result.DegradedReason = "journey context is nil"
		return result, nil
	}
	for _, spec := range request.Plan.Steps {
		result.Steps = append(result.Steps, deliveryramp.JourneyStep{ID: spec.ID, ChapterID: spec.ID, Name: spec.ID, Purpose: spec.Purpose, Action: spec.Action, Disposition: deliveryramp.StepUnavailable, Error: "missing capability: " + requiredCapability(spec) + "; next action: register a macOS node with simctl/WebDriverAgent", DegradedReason: "Apple runtime unavailable", Readiness: spec.Readiness, Settle: spec.Settle, ExpectedState: spec.Assertion.Expected})
	}
	return result, nil
}

func requiredCapability(spec deliveryramp.JourneyStepSpec) string {
	if value := strings.TrimSpace(spec.Arguments["required_capability"]); value != "" {
		return value
	}
	return capabilitySimctl
}
