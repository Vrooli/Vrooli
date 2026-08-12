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
	if a.display == "" {
		next := "Start a usable DISPLAY session (or configure WAYLAND_DISPLAY) for host-desktop."
		return strategy.UnavailableDeclaration(a.ID(), "Local Linux desktop", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next), nil
	}
	if _, err := exec.LookPath("import"); err != nil {
		next := "Install ImageMagick and expose import on PATH for host-desktop screenshots."
		return strategy.UnavailableDeclaration(a.ID(), "Local Linux desktop", []strategy.Capability{{Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next), nil
	}
	if _, err := exec.LookPath("xdotool"); err != nil {
		next := "Install xdotool and expose it on PATH for host-desktop input."
		return strategy.UnavailableDeclaration(a.ID(), "Local Linux desktop", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Status: strategy.StatusAvailable, ProbeEvidence: "ImageMagick import probe"}}, next), nil
	}
	caps := map[string]strategy.Capability{strategy.CapInput: strategy.ProbeCapability(strategy.CapInput, true, "", "", "DISPLAY probe"), strategy.CapScreenshot: strategy.ProbeCapability(strategy.CapScreenshot, true, "", "", "DISPLAY probe")}
	d := strategy.Declaration{StrategyID: a.ID(), Description: "The local host desktop through the configured display", Status: strategy.StatusAvailable, Capabilities: caps, Promotable: true}
	d.Tiers = strategy.Tiers(d)
	return d, nil
}

func (a *Adapter) Observe(ctx context.Context) (strategy.Frame, error) {
	if a.display == "" {
		return strategy.Frame{}, &strategy.AvailabilityError{Reason: "no display session", NextAction: "Start a usable DISPLAY session for host-desktop."}
	}
	// Import is present on the supported desktop images. If it is not, report a
	// precise unavailable result instead of fabricating a successful capture.
	data, err := exec.CommandContext(ctx, "import", "-window", "root", "png:-").Output()
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("capture host display: %w", err)
	}
	cfg, _, err := image.DecodeConfig(strings.NewReader(string(data)))
	if err != nil {
		return strategy.Frame{}, err
	}
	return strategy.Frame{Width: cfg.Width, Height: cfg.Height, Scale: 1, Timestamp: time.Now().UTC(), MediaType: "image/png", Bytes: data}, nil
}

func (a *Adapter) Actuate(ctx context.Context, event strategy.Actuation) error {
	if a.display == "" {
		return &strategy.AvailabilityError{Reason: "no display session", NextAction: "Start a usable DISPLAY session for host-desktop."}
	}
	if event.Pointer == nil {
		return fmt.Errorf("host-desktop currently accepts pointer events only")
	}
	return exec.CommandContext(ctx, "xdotool", "mousemove", fmt.Sprintf("%d", int(event.Pointer.X)), fmt.Sprintf("%d", int(event.Pointer.Y)), "click", "1").Run()
}
