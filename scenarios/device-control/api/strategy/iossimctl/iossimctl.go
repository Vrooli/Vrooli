package iossimctl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"device-control/strategy"
)

type Adapter struct{ udid string }

func New() *Adapter { return NewWithUDID("") }

func NewWithUDID(udid string) *Adapter { return &Adapter{udid: strings.TrimSpace(udid)} }
func (a *Adapter) ID() string          { return "ios-simctl" }

type simctlDeviceList struct {
	Devices map[string][]simctlDeviceInfo `json:"devices"`
}

type simctlDeviceInfo struct {
	UDID  string `json:"udid"`
	Name  string `json:"name"`
	State string `json:"state"`
}

func selectBootedDevice(listed simctlDeviceList, udid string) (string, error) {
	if strings.TrimSpace(udid) == "" {
		return "", &strategy.AvailabilityError{Reason: "no iOS simulator UDID is selected", NextAction: "Set IOS_SIMULATOR_UDID to the UDID of an already booted simulator on this macOS host."}
	}
	for _, devices := range listed.Devices {
		for _, device := range devices {
			if device.UDID != udid {
				continue
			}
			if device.State != "Booted" {
				return "", &strategy.AvailabilityError{Reason: fmt.Sprintf("selected iOS simulator %q is not booted (state %q)", device.Name, device.State), NextAction: "Boot the selected simulator through the authorized simulator workflow, then retry observation."}
			}
			return device.UDID, nil
		}
	}
	return "", &strategy.AvailabilityError{Reason: "selected iOS simulator UDID is not present in the available simulator inventory", NextAction: "Select an installed simulator UDID reported by xcrun simctl list devices available."}
}

// selectedBootedDevice is deliberately read-only: this adapter never boots,
// shuts down, or changes the selected simulator's configuration.
func (a *Adapter) selectedBootedDevice(ctx context.Context) (string, error) {
	if strings.TrimSpace(a.udid) == "" {
		return selectBootedDevice(simctlDeviceList{}, a.udid)
	}
	if _, err := exec.LookPath("xcrun"); err != nil {
		return "", &strategy.AvailabilityError{Reason: "Apple xcrun tooling is unavailable", NextAction: "Run Device Control on a macOS host with Xcode and an iOS Simulator runtime installed."}
	}
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "list", "devices", "available", "--json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("list available iOS simulators: %w", err)
	}
	var listed simctlDeviceList
	if err := json.Unmarshal(output, &listed); err != nil {
		return "", fmt.Errorf("decode iOS simulator inventory: %w", err)
	}
	return selectBootedDevice(listed, a.udid)
}

func (a *Adapter) Describe(ctx context.Context) (strategy.Declaration, error) {
	if unsupported, ok := strategy.ResolveHostSupport(a.ID(), "iOS Simulator through simctl", []string{"darwin"}); ok {
		return unsupported, nil
	}
	if _, err := a.selectedBootedDevice(ctx); err != nil {
		var unavailable *strategy.AvailabilityError
		if errors.As(err, &unavailable) {
			return strategy.WithSupportedHostOS(strategy.UnavailableDeclaration(a.ID(), "iOS Simulator through simctl", []strategy.Capability{{Name: strategy.CapInput, Status: strategy.StatusUnavailable, Prerequisite: unavailable.Reason, NextAction: unavailable.NextAction}, {Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Prerequisite: unavailable.Reason, NextAction: unavailable.NextAction}}, unavailable.NextAction), "darwin"), nil
		}
		return strategy.WithSupportedHostOS(strategy.UnavailableDeclaration(a.ID(), "iOS Simulator through simctl", []strategy.Capability{{Name: strategy.CapInput, Status: strategy.StatusUnavailable, Prerequisite: err.Error()}, {Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Prerequisite: err.Error()}}, "Check Xcode and the selected simulator runtime."), "darwin"), nil
	}
	caps := map[string]strategy.Capability{}
	caps[strategy.CapInput] = strategy.ProbeCapability(strategy.CapInput, true, "", "", "simctl openurl on selected simulator")
	caps[strategy.CapScreenshot] = strategy.ProbeCapability(strategy.CapScreenshot, true, "", "", "simctl screenshot on selected simulator")
	d := strategy.Declaration{StrategyID: a.ID(), Description: "Read-only screenshots and URL navigation through iOS Simulator simctl", Status: strategy.StatusAvailable, Capabilities: caps, Promotable: true, EvidenceClass: "release-grade"}
	d = strategy.WithSupportedHostOS(d, "darwin")
	d.Tiers = strategy.Tiers(d)
	return d, nil
}

func (a *Adapter) Observe(ctx context.Context) (strategy.Frame, error) {
	udid, err := a.selectedBootedDevice(ctx)
	if err != nil {
		return strategy.Frame{}, err
	}
	dir, err := os.MkdirTemp("", "device-control-ios-simctl-")
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("create temporary screenshot directory: %w", err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "screen.png")
	command := exec.CommandContext(ctx, "xcrun", "simctl", "io", udid, "screenshot", path)
	if output, err := command.CombinedOutput(); err != nil {
		return strategy.Frame{}, fmt.Errorf("capture selected iOS simulator: %w: %s", err, strings.TrimSpace(string(output)))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("read iOS simulator screenshot: %w", err)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("decode iOS simulator screenshot dimensions: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 {
		return strategy.Frame{}, fmt.Errorf("iOS simulator returned invalid screenshot dimensions %dx%d", config.Width, config.Height)
	}
	return strategy.Frame{Width: config.Width, Height: config.Height, Scale: 1, Timestamp: time.Now().UTC(), MediaType: "image/png", Bytes: data}, nil
}

func (a *Adapter) Actuate(ctx context.Context, event strategy.Actuation) error {
	udid, err := a.selectedBootedDevice(ctx)
	if err != nil {
		return err
	}
	if event.Action != "open-url" || event.Pointer != nil || event.Key != nil || event.Text != "" || event.Package != "" || event.Component != "" || event.LogTag != "" || event.LogPriority != "" || event.Permission != "" || event.Expected != "" || event.Output != nil {
		return &strategy.AvailabilityError{Reason: "simctl supports URL navigation but not arbitrary pointer, keyboard, or text input", NextAction: "Use the EX9 journey actuator for supported UI actions; use Action=open-url only for simulator deep-link navigation."}
	}
	urlValue := strings.TrimSpace(event.Value)
	if urlValue == "" || strings.ContainsAny(urlValue, "\r\n\x00") {
		return fmt.Errorf("open-url requires a non-empty URL without control characters")
	}
	parsed, err := url.Parse(urlValue)
	if err != nil {
		return fmt.Errorf("open-url requires an absolute URL: %w", err)
	}
	if !parsed.IsAbs() || parsed.Scheme == "" {
		return fmt.Errorf("open-url requires an absolute URL with a scheme")
	}
	if (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() == "" {
		return fmt.Errorf("open-url HTTP URL must include a host")
	}
	if parsed.Scheme == "file" || parsed.Scheme == "data" || parsed.Scheme == "javascript" {
		return fmt.Errorf("open-url does not allow the %q URL scheme", parsed.Scheme)
	}
	command := exec.CommandContext(ctx, "xcrun", "simctl", "openurl", udid, urlValue)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("navigate selected iOS simulator to URL: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
