package targetinventory

import (
	"context"
	"errors"
	"testing"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

func TestLocalProbeReportsReadyLinuxCapabilities(t *testing.T) {
	probe := LocalProbe{
		LookPath: func(name string) (string, error) {
			switch name {
			case "xvfb-run", "xdotool", "ffmpeg":
				return "/usr/bin/" + name, nil
			default:
				return "", errors.New("missing")
			}
		},
		Getenv: func(string) string { return "" },
	}
	result := probe.Probe(context.Background())
	if result.Descriptor == nil || !result.Descriptor.GetAvailable() {
		t.Fatalf("expected ready target: %+v", result)
	}
	if result.Health.Status != "healthy" || result.BridgeTrust != nil {
		t.Fatalf("local target health/trust = %#v/%#v", result.Health, result.BridgeTrust)
	}
	for _, capability := range []domainv1.ValidationTargetCapability{
		domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_ELECTRON_CDP,
		domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NATIVE_WINDOW,
		domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_PROCESS_METRICS,
	} {
		if !hasCapability(result.Descriptor.GetCapabilities(), capability) {
			t.Fatalf("missing capability %s: %+v", capability, result.Descriptor.GetCapabilities())
		}
	}
}

func TestLocalProbeReportsUnavailableEvidencePrerequisites(t *testing.T) {
	result := (LocalProbe{
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
		Getenv:   func(string) string { return "" },
	}).Probe(context.Background())
	if result.Descriptor == nil || result.Descriptor.GetAvailable() {
		t.Fatalf("expected unavailable target: %+v", result)
	}
	if result.Descriptor.GetReason() == "" {
		t.Fatal("unavailable target should explain missing prerequisites")
	}
	if result.Health.Status != "unavailable" {
		t.Fatalf("health = %q, want unavailable", result.Health.Status)
	}
}

func hasCapability(values []domainv1.ValidationTargetCapability, wanted domainv1.ValidationTargetCapability) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
