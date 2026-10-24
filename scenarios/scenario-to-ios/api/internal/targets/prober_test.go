package targets

import (
	"context"
	"errors"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestProbeLinuxSimulatorIsUnsupported(t *testing.T) {
	inventory, err := (Prober{GOOS: "linux", LookPath: func(string) (string, error) { return "", errors.New("missing") }}).Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Targets[0].Available || inventory.Targets[0].MissingCapability == "" || inventory.Targets[0].NextAction == "" {
		t.Fatalf("invalid Linux disposition: %+v", inventory.Targets[0])
	}
	if inventory.Targets[0].Reason != "iOS Simulator requires Apple tooling and cannot run on Linux" {
		t.Fatalf("reason = %q", inventory.Targets[0].Reason)
	}
}

func TestProbeWithoutMacBridgeIsUnavailable(t *testing.T) {
	inventory, err := (Prober{GOOS: "linux", LookPath: func(string) (string, error) { return "", errors.New("missing") }}).Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	bridge := inventory.Targets[1]
	if bridge.Available || bridge.MissingCapability != CapabilityMacOSBridge || bridge.NextAction == "" {
		t.Fatalf("bridge disposition = %+v", bridge)
	}
}

func TestProbeMacToolchainUsesInjectedTools(t *testing.T) {
	inventory, err := (Prober{GOOS: "darwin", LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil }}).Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !inventory.Targets[0].Available {
		t.Fatalf("expected local toolchain available: %+v", inventory.Targets[0])
	}
}
