package desktopprobe

import (
	"context"
	"errors"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestProberReportsOracleLinuxCapabilities(t *testing.T) {
	result, err := (Prober{
		GOOS: "linux", GOARCH: "amd64",
		LookPath: func(name string) (string, error) {
			return "/usr/bin/" + name, nil
		},
		Getenv: func(string) string { return "" },
	}).Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	target := result.Targets[0]
	if !target.Available || !target.Supports(deliveryramp.CapabilityCDP) || !target.Supports(deliveryramp.CapabilityNativeWindow) || !target.Supports(deliveryramp.CapabilityProcessMetrics) {
		t.Fatalf("unexpected target: %+v", target)
	}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestProberReportsUnavailableMissingCapabilityAndRecovery(t *testing.T) {
	result, err := (Prober{
		GOOS: "linux", GOARCH: "amd64",
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
		Getenv:   func(string) string { return "" },
	}).Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	target := result.Targets[0]
	if target.Available || target.MissingCapability == "" || target.NextAction == "" {
		t.Fatalf("unavailable target lacks recovery details: %+v", target)
	}
	if err := result.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestProberMarksUnsupportedHostUnavailable(t *testing.T) {
	result, err := (Prober{
		GOOS: "darwin", GOARCH: "arm64",
		LookPath: func(name string) (string, error) {
			if name == "ffmpeg" {
				return "/usr/bin/ffmpeg", nil
			}
			return "", errors.New("missing")
		},
	}).Probe(context.Background(), deliveryramp.ProbeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Targets[0].Available || result.Targets[0].MissingCapability != "supported Linux display runtime" {
		t.Fatalf("unexpected unsupported target: %+v", result.Targets[0])
	}
}
