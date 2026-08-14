package androidrelease

import (
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestEvaluateGateRejectsUnavailableRequiredCell(t *testing.T) {
	target := deliveryramp.Target{ID: "android:emulator:local", Platform: "android", Available: false, MissingCapability: "KVM", NextAction: "enable KVM", Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal}}
	if _, err := EvaluateGate([]CellResult{{ID: "smoke", Required: true, Disposition: deliveryramp.DispositionUnavailable, Target: target}}, "scenario-to-android", "run-1", "smoke"); err == nil {
		t.Fatal("expected unavailable cell to fail gate")
	}
}

func TestEvaluateGateRequiresReferenceOnlyEvidence(t *testing.T) {
	target := deliveryramp.Target{ID: "android:emulator:local", Platform: "android", Available: true, Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal}}
	if _, err := EvaluateGate([]CellResult{{ID: "smoke", Required: true, Disposition: deliveryramp.DispositionPass, Target: target}}, "scenario-to-android", "run-1", "smoke"); err == nil {
		t.Fatal("expected pass without evidence to fail gate")
	}
}
