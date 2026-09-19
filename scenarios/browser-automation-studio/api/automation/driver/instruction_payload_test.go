package driver

import (
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

func clickInstruction() contracts.CompiledInstruction {
	return contracts.CompiledInstruction{
		Index:  3,
		NodeID: "click-thing",
		Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK},
	}
}

// buildInstructionPayload assembles the driver wire format field by field, so a
// new field on CompiledInstruction is dropped silently instead of failing to
// compile. That is exactly how the telemetry directive went missing: every
// component tested green in isolation while nothing reached the driver.
func TestBuildInstructionPayloadCarriesTelemetryDirective(t *testing.T) {
	instruction := clickInstruction()
	instruction.Telemetry = &basexecution.StepTelemetryDirective{
		Screenshot: basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ON_FAILURE,
	}

	payload, err := buildInstructionPayload(instruction)
	if err != nil {
		t.Fatalf("buildInstructionPayload: %v", err)
	}

	wire, ok := payload["instruction"].(map[string]any)
	if !ok {
		t.Fatalf("payload has no instruction object: %#v", payload)
	}
	telemetry, ok := wire["telemetry"].(map[string]any)
	if !ok {
		t.Fatalf("telemetry directive missing from wire payload: %#v", wire)
	}
	// UseEnumNumbers keeps this an integer on the wire; the driver's lenient
	// proto parse accepts that form.
	got, ok := telemetry["screenshot"].(float64)
	if !ok {
		t.Fatalf("screenshot policy is not a JSON number: %#v", telemetry["screenshot"])
	}
	want := float64(basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ON_FAILURE)
	if got != want {
		t.Fatalf("screenshot policy = %v, want %v", got, want)
	}
}

// An instruction with no directive must not emit the key at all, so the driver
// falls back to its own defaults and older drivers see an unchanged payload.
func TestBuildInstructionPayloadOmitsAbsentTelemetryDirective(t *testing.T) {
	payload, err := buildInstructionPayload(clickInstruction())
	if err != nil {
		t.Fatalf("buildInstructionPayload: %v", err)
	}

	wire := payload["instruction"].(map[string]any)
	if _, present := wire["telemetry"]; present {
		t.Fatalf("expected no telemetry key, got %#v", wire["telemetry"])
	}
}
