// Package targets owns iOS target observation for the delivery ramp.
package targets

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

const (
	CapabilityMacOSBridge = "macos-bridge-node"
	CapabilityXcode       = "xcodebuild"
	CapabilitySimctl      = "simctl"
	CapabilityWebDriver   = "ios-webdriver"
)

// LookPath is injectable so target probing remains deterministic in tests.
type LookPath func(string) (string, error)

// Prober observes host tooling and registered macOS bridge capabilities. It
// never infers Apple readiness from the host OS alone.
type Prober struct {
	LookPath LookPath
	GOOS     string
	Now      func() time.Time
	Bridge   []deliveryramp.Target
}

var _ deliveryramp.Prober = Prober{}

// Probe returns an explicit unsupported simulator target on Linux and an
// unavailable bridge target when no qualifying macOS node is registered.
func (p Prober) Probe(_ context.Context, request deliveryramp.ProbeRequest) (deliveryramp.Inventory, error) {
	goos := p.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	lookPath := p.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	targets := make([]deliveryramp.Target, 0, len(p.Bridge)+2)
	if goos == "darwin" {
		targets = append(targets, localMacTarget(lookPath, request.RequiredCapability))
	} else {
		targets = append(targets, deliveryramp.Target{
			ID: "ios:simulator:linux", Ramp: "scenario-to-ios", Label: "iOS simulator (Linux host)",
			Platform: "ios", OS: "iOS", Architecture: runtime.GOARCH, DeviceKind: "emulator", Mode: "simulator",
			Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "linux", Available: false},
			Available: false, Reason: "iOS Simulator requires Apple tooling and cannot run on Linux",
			MissingCapability: "macOS iOS Simulator runtime", NextAction: "register a macOS bridge node",
			Health: deliveryramp.TargetHealth{Status: "unsupported", Reason: "Apple simulator is unsupported on Linux"},
		})
	}
	if len(p.Bridge) == 0 {
		targets = append(targets, unavailableBridgeTarget())
	} else {
		for _, target := range p.Bridge {
			targets = append(targets, normalizeBridgeTarget(target, request.RequiredCapability))
		}
	}
	for i := range targets {
		if err := targets[i].Validate(); err != nil {
			return deliveryramp.Inventory{}, fmt.Errorf("validate iOS target %q: %w", targets[i].ID, err)
		}
	}
	return deliveryramp.Inventory{Targets: targets, Observed: now().UTC()}, nil
}

func localMacTarget(lookPath LookPath, required []string) deliveryramp.Target {
	missing := make([]string, 0, 2)
	for _, tool := range []string{"xcodebuild", "xcrun"} {
		if _, err := lookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	available := len(missing) == 0 && supports(required, []string{CapabilityXcode, CapabilitySimctl})
	if len(missing) == 0 && !available {
		missing = append(missing, "requested iOS capability")
	}
	return deliveryramp.Target{
		ID: "ios:macos:local", Ramp: "scenario-to-ios", Label: "Local macOS iOS toolchain", Platform: "ios", OS: "macOS", Architecture: runtime.GOARCH,
		DeviceKind: "host", Mode: "xcode", Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "local-macos", Available: available},
		Capabilities: []string{CapabilityXcode, CapabilitySimctl, CapabilityWebDriver}, Available: available,
		Reason: availabilityReason(available, missing), MissingCapability: strings.Join(missing, ", "), NextAction: nextAction(available, "install Xcode and expose simctl, then probe again"),
		Health: deliveryramp.TargetHealth{Status: healthStatus(available)},
	}
}

func unavailableBridgeTarget() deliveryramp.Target {
	return deliveryramp.Target{
		ID: "ios:macos:bridge-unavailable", Ramp: "scenario-to-ios", Label: "macOS bridge node", Platform: "ios", OS: "macOS", DeviceKind: "host", Mode: "bridge",
		Transport: deliveryramp.Transport{Kind: deliveryramp.TransportBridge, ID: "vrooli-bridge", Available: false}, Available: false,
		Reason: "no registered macOS bridge node is available", MissingCapability: CapabilityMacOSBridge, NextAction: "register a trusted macOS bridge node and probe again",
		Health: deliveryramp.TargetHealth{Status: "unavailable", Reason: "no macOS bridge node"},
	}
}

func normalizeBridgeTarget(target deliveryramp.Target, required []string) deliveryramp.Target {
	target.Ramp, target.Platform = "scenario-to-ios", "ios"
	if target.MissingCapability == "" && !target.Available {
		target.MissingCapability = CapabilityMacOSBridge
	}
	if target.NextAction == "" && !target.Available {
		target.NextAction = "restore the macOS bridge node and probe again"
	}
	if target.Available && !supports(required, target.Capabilities) {
		target.Available = false
		target.MissingCapability = "required iOS capability"
		target.NextAction = "advertise the required iOS capability and probe again"
	}
	return target
}

func supports(required, observed []string) bool {
	for _, want := range required {
		found := false
		for _, got := range observed {
			if strings.EqualFold(strings.TrimSpace(want), strings.TrimSpace(got)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func availabilityReason(available bool, missing []string) string {
	if available {
		return "macOS iOS toolchain is present"
	}
	return "macOS iOS toolchain is unavailable: " + strings.Join(missing, ", ")
}

func nextAction(available bool, action string) string {
	if available {
		return "no action required"
	}
	return action
}

func healthStatus(available bool) string {
	if available {
		return "ready"
	}
	return "unavailable"
}
