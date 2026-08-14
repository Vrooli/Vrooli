// Package androidprobe owns Android host and device capability detection for
// scenario-to-android. It observes device-control's provider-neutral inventory
// and never reaches around that service with a direct adb call.
package androidprobe

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type LookPath func(string) (string, error)

type RunCommand func(context.Context, string, ...string) ([]byte, error)

type DeviceObservation struct {
	ID           string
	Label        string
	NodeID       string
	Serial       string
	OS           string
	Architecture string
	Transport    deliveryramp.Transport
	ADBTransport string
	Capabilities []string
	Available    bool
	Reason       string
}

// DeviceInventory is intentionally smaller than device-control's generated
// API. The scenario consumes observations, while device-control retains
// ownership of discovery, leases, and provider-specific verbs.
type DeviceInventory interface {
	List(context.Context) ([]DeviceObservation, error)
}

type Prober struct {
	LookPath LookPath
	Getenv   func(string) string
	Run      RunCommand
	KVM      func() (present bool, writable bool, reason string)
	Devices  DeviceInventory
	Now      func() time.Time
}

var _ deliveryramp.Prober = Prober{}

func (p Prober) Probe(ctx context.Context, request deliveryramp.ProbeRequest) (deliveryramp.Inventory, error) {
	lookPath := p.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	run := p.Run
	if run == nil {
		run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		}
	}
	kvm := p.KVM
	if kvm == nil {
		kvm = hostKVM
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}

	local := probeLocalEmulator(ctx, lookPath, run, kvm)
	targets := []deliveryramp.Target{local}
	if p.Devices != nil {
		observations, err := p.Devices.List(ctx)
		if err != nil {
			return deliveryramp.Inventory{}, fmt.Errorf("list device-control Android targets: %w", err)
		}
		for _, observation := range observations {
			targets = append(targets, targetFromObservation(observation, request.RequiredCapability))
		}
	} else {
		targets = append(targets, unavailablePhysical("device-control inventory is not configured", "device-control"))
	}

	for index := range targets {
		if targets[index].Available && !matchesRequired(targets[index], request.RequiredCapability) {
			targets[index].Available = false
			targets[index].MissingCapability = missingRequired(targets[index], request.RequiredCapability)
			targets[index].NextAction = "provide the required Android capability and probe again"
			targets[index].Reason = "Android target does not provide all requested capabilities"
		}
		if err := targets[index].Validate(); err != nil {
			return deliveryramp.Inventory{}, err
		}
	}
	return deliveryramp.Inventory{Targets: targets, Observed: now().UTC()}, nil
}

func probeLocalEmulator(ctx context.Context, lookPath LookPath, run RunCommand, kvm func() (bool, bool, string)) deliveryramp.Target {
	missing := make([]string, 0, 4)
	for _, tool := range []string{"adb", "emulator", "avdmanager", "sdkmanager"} {
		if _, err := lookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if _, err := lookPath("ffmpeg"); err != nil {
		missing = append(missing, "ffmpeg evidence recorder")
	}
	if len(missing) == 0 {
		present, writable, _ := kvm()
		if !present {
			missing = append(missing, "/dev/kvm")
		} else if !writable {
			missing = append(missing, "/dev/kvm writable access")
		} else if output, err := run(ctx, "emulator", "-list-avds"); err != nil || strings.TrimSpace(string(output)) == "" {
			missing = append(missing, "configured Android AVD")
		}
	}

	capabilities := []string{deliveryramp.CapabilityDeviceControl, deliveryramp.CapabilityScreenRecording}
	available := len(missing) == 0
	reason := "local accelerated Android emulator is ready"
	if !available {
		reason = fmt.Sprintf("local Android emulator prerequisites unavailable: %s", strings.Join(missing, ", "))
	}
	nextAction := ""
	if !available {
		nextAction = "install android-sdk, create an accelerated AVD, and probe again"
	}
	return deliveryramp.Target{
		ID: "android:emulator:local", Ramp: "scenario-to-android", Label: "Local Android emulator",
		Platform: "android", OS: "Android", Architecture: "host", DeviceKind: "emulator",
		Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "local", Trust: "host", Available: available},
		Mode:      "emulator", Capabilities: capabilities, Available: available, Reason: reason,
		MissingCapability: strings.Join(missing, ", "), NextAction: nextAction,
		Health: deliveryramp.TargetHealth{Status: "observed", Reason: reason},
	}
}

func targetFromObservation(observation DeviceObservation, required []string) deliveryramp.Target {
	if observation.ID == "" {
		observation.ID = "android:physical:unknown"
	}
	available := observation.Available
	missing := missingRequiredValues(observation.Capabilities, required)
	if !available && strings.TrimSpace(missing) == "" {
		missing = "device-control target"
	}
	if observation.Transport.Kind == "" {
		observation.Transport.Kind = deliveryramp.TransportBridge
	}
	nextAction := ""
	if !available {
		nextAction = "restore the device-control target and probe again"
	}
	deviceKind := "physical"
	mode := "physical"
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(observation.Serial)), "emulator-") {
		deviceKind, mode = "emulator", "emulator"
	}
	return deliveryramp.Target{
		ID: observation.ID, Ramp: "scenario-to-android", Label: observation.Label,
		Platform: "android", OS: observation.OS, Architecture: observation.Architecture,
		DeviceKind: deviceKind, Transport: observation.Transport, NodeID: observation.NodeID,
		Mode: mode, Capabilities: observation.Capabilities, Available: available,
		Reason: observation.Reason, MissingCapability: missing, NextAction: nextAction,
		Health: deliveryramp.TargetHealth{Status: healthStatus(available), Reason: observation.Reason},
	}
}

func unavailablePhysical(reason, missing string) deliveryramp.Target {
	return deliveryramp.Target{
		ID: "android:physical:unavailable", Ramp: "scenario-to-android", Label: "Physical Android device",
		Platform: "android", OS: "Android", DeviceKind: "physical", Mode: "physical",
		Transport: deliveryramp.Transport{Kind: deliveryramp.TransportBridge, ID: "device-control", Available: false, Reason: reason},
		Available: false, Reason: reason, MissingCapability: missing,
		NextAction: "start device-control and attach a trusted Android device",
		Health:     deliveryramp.TargetHealth{Status: "unavailable", Reason: reason},
	}
}

func hostKVM() (bool, bool, string) {
	info, err := os.Stat("/dev/kvm")
	if err != nil {
		return false, false, err.Error()
	}
	if info.Mode()&os.ModeDevice == 0 {
		return true, false, "/dev/kvm is not a device node"
	}
	file, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return true, false, err.Error()
	}
	_ = file.Close()
	return true, true, ""
}

func matchesRequired(target deliveryramp.Target, required []string) bool {
	if !target.Available {
		return false
	}
	for _, capability := range required {
		if !target.Supports(capability) {
			return false
		}
	}
	return true
}

func missingRequired(target deliveryramp.Target, required []string) string {
	return missingRequiredValues(target.Capabilities, required)
}

func missingRequiredValues(capabilities, required []string) string {
	missing := make([]string, 0, len(required))
	for _, capability := range required {
		found := false
		for _, observed := range capabilities {
			if strings.EqualFold(strings.TrimSpace(capability), strings.TrimSpace(observed)) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, capability)
		}
	}
	return strings.Join(missing, ", ")
}

func healthStatus(available bool) string {
	if available {
		return "ready"
	}
	return "unavailable"
}
