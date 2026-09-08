package hostdesktop

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/exec"
	"strings"
	"time"

	"device-control/strategy"
)

type Adapter struct{ display string }

func New() *Adapter           { return &Adapter{display: strings.TrimSpace(os.Getenv("DISPLAY"))} }
func (a *Adapter) ID() string { return "host-desktop" }
func (a *Adapter) Describe(context.Context) (strategy.Declaration, error) {
	// Linux and macOS have implemented capture/input paths. Windows remains
	// explicitly unsupported until a real, permission-aware implementation is
	// available; a PowerShell executable alone does not prove that capability.
	supported := []string{"linux", "darwin"}
	if unsupported, ok := strategy.ResolveHostSupport(a.ID(), "The local host desktop through the configured display", supported); ok {
		return unsupported, nil
	}
	if strategy.HostOS == "darwin" {
		return strategy.WithSupportedHostOS(a.describeDarwin(), supported...), nil
	}
	if a.display == "" {
		next := "Start a usable DISPLAY session for host-desktop."
		return strategy.WithSupportedHostOS(strategy.UnavailableDeclaration(a.ID(), "The local host desktop through the configured display", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next), supported...), nil
	}
	if _, err := exec.LookPath("import"); err != nil {
		next := "Install ImageMagick and expose import on PATH for host-desktop screenshots."
		return strategy.WithSupportedHostOS(strategy.UnavailableDeclaration(a.ID(), "The local host desktop through the configured display", []strategy.Capability{{Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next), supported...), nil
	}
	if _, err := exec.LookPath("xdotool"); err != nil {
		next := "Install xdotool and expose it on PATH for host-desktop input."
		return strategy.WithSupportedHostOS(strategy.UnavailableDeclaration(a.ID(), "The local host desktop through the configured display", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Status: strategy.StatusAvailable, ProbeEvidence: "ImageMagick import probe"}}, next), supported...), nil
	}
	caps := map[string]strategy.Capability{strategy.CapInput: strategy.ProbeCapability(strategy.CapInput, true, "", "", "DISPLAY probe"), strategy.CapScreenshot: strategy.ProbeCapability(strategy.CapScreenshot, true, "", "", "DISPLAY probe")}
	d := strategy.Declaration{StrategyID: a.ID(), Description: "The local host desktop through the configured display", Status: strategy.StatusAvailable, Capabilities: caps, Promotable: true, EvidenceClass: "release-grade", MinimumUsefulFPS: 5}
	d = strategy.WithSupportedHostOS(d, supported...)
	d.Tiers = strategy.Tiers(d)
	return d, nil
}

func (a *Adapter) describeDarwin() strategy.Declaration {
	const description = "The local host desktop through the configured display"
	if _, err := exec.LookPath("screencapture"); err != nil {
		next := "Allow Screen Recording for the device-control process in macOS Privacy & Security settings."
		return strategy.UnavailableDeclaration(a.ID(), description, []strategy.Capability{{Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next)
	}
	if _, err := exec.LookPath("osascript"); err != nil {
		next := "Allow Accessibility for the device-control process in macOS Privacy & Security settings."
		return strategy.UnavailableDeclaration(a.ID(), description, []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Status: strategy.StatusAvailable}}, next)
	}
	caps := map[string]strategy.Capability{strategy.CapInput: strategy.ProbeCapability(strategy.CapInput, true, "", "", "osascript accessibility path"), strategy.CapScreenshot: strategy.ProbeCapability(strategy.CapScreenshot, true, "", "", "screencapture path")}
	d := strategy.Declaration{StrategyID: a.ID(), Description: description, Status: strategy.StatusAvailable, Capabilities: caps, Promotable: true, EvidenceClass: "release-grade", MinimumUsefulFPS: 5}
	d.Tiers = strategy.Tiers(d)
	return d
}

func (a *Adapter) Observe(ctx context.Context) (strategy.Frame, error) {
	if strategy.HostOS == "darwin" {
		data, err := exec.CommandContext(ctx, "screencapture", "-x", "-t", "png", "-").Output()
		if err != nil {
			return strategy.Frame{}, fmt.Errorf("capture macOS desktop: %w", err)
		}
		return decodeFrame(data)
	}
	if strategy.HostOS == "windows" {
		return strategy.Frame{}, &strategy.AvailabilityError{Reason: "Windows host-desktop capture is not implemented", NextAction: "Use a Linux or macOS host until the Windows capture path is verified."}
	}
	if a.display == "" {
		return strategy.Frame{}, &strategy.AvailabilityError{Reason: "no display session", NextAction: "Start a usable DISPLAY session for host-desktop."}
	}
	// Import is present on the supported desktop images. If it is not, report a
	// precise unavailable result instead of fabricating a successful capture.
	data, err := exec.CommandContext(ctx, "import", "-window", "root", "png:-").Output()
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("capture host display: %w", err)
	}
	return decodeFrame(data)
}

func decodeFrame(data []byte) (strategy.Frame, error) {
	cfg, _, err := image.DecodeConfig(strings.NewReader(string(data)))
	if err != nil {
		return strategy.Frame{}, err
	}
	return strategy.Frame{Width: cfg.Width, Height: cfg.Height, Scale: 1, Timestamp: time.Now().UTC(), MediaType: "image/png", Bytes: data}, nil
}

func (a *Adapter) Actuate(ctx context.Context, event strategy.Actuation) error {
	if strategy.HostOS == "darwin" {
		if event.Pointer == nil {
			return fmt.Errorf("host-desktop currently accepts pointer events only")
		}
		script := fmt.Sprintf("tell application \"System Events\" to click at {%d, %d}", int(event.Pointer.X), int(event.Pointer.Y))
		return exec.CommandContext(ctx, "osascript", "-e", script).Run()
	}
	if strategy.HostOS == "windows" {
		return &strategy.AvailabilityError{Reason: "Windows host-desktop input is not implemented", NextAction: "Use a Linux or macOS host until the Windows input path is verified."}
	}
	if a.display == "" {
		return &strategy.AvailabilityError{Reason: "no display session", NextAction: "Start a usable DISPLAY session for host-desktop."}
	}
	if event.Pointer == nil {
		return fmt.Errorf("host-desktop currently accepts pointer events only")
	}
	return exec.CommandContext(ctx, "xdotool", "mousemove", fmt.Sprintf("%d", int(event.Pointer.X)), fmt.Sprintf("%d", int(event.Pointer.Y)), "click", "1").Run()
}
