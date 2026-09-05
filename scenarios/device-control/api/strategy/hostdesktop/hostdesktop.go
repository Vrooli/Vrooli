package hostdesktop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"time"

	"device-control/strategy"
)

const (
	captureTimeout   = 5 * time.Second
	maxCaptureBytes  = 32 << 20
	maxCapturePixels = 64 << 20
)

type Adapter struct {
	display string
	capture func(context.Context) (strategy.Frame, error)
}

func New() *Adapter {
	a := &Adapter{display: strings.TrimSpace(os.Getenv("DISPLAY"))}
	a.capture = a.Observe
	return a
}
func (a *Adapter) ID() string { return "host-desktop" }
func (a *Adapter) Describe(ctx context.Context) (strategy.Declaration, error) {
	const description = "The local host desktop through the configured display"
	supported := []string{"linux", "darwin"}
	if unsupported, ok := strategy.ResolveHostSupport(a.ID(), description, supported); ok {
		return unsupported, nil
	}
	// Discovery never injects input to test permission. Executable discovery is
	// not evidence of permission or a binding to an authenticated user session.
	inputNext := "Complete user-session input admission before controlling this desktop."
	captureNext := "Check the desktop session and screen capture permission, then retry capture."
	d := strategy.UnavailableDeclaration(a.ID(), description, []strategy.Capability{
		{Name: strategy.CapInput, Reason: "user-session input admission is unverified", NextAction: inputNext},
		{Name: strategy.CapScreenshot, Reason: "capture has not succeeded", NextAction: captureNext},
	}, captureNext, inputNext)
	d = strategy.WithSupportedHostOS(d, supported...)
	probeCtx, cancel := context.WithTimeout(ctx, captureTimeout)
	defer cancel()
	capture := a.capture
	if capture == nil {
		capture = a.Observe
	}
	frame, err := capture(probeCtx)
	if err != nil || probeCtx.Err() != nil || frame.Width <= 0 || frame.Height <= 0 || len(frame.Bytes) == 0 {
		// Do not expose command stderr, desktop pixels, or environment details in
		// inventory. Failure is not sufficient evidence to infer permission denial.
		d.Capabilities[strategy.CapScreenshot] = strategy.Capability{Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Reason: "desktop capture probe failed", NextAction: captureNext}
		return d, nil
	}
	d.Status = strategy.StatusAvailable
	d.NextActions = []string{inputNext}
	d.Capabilities[strategy.CapScreenshot] = strategy.ProbeCapability(strategy.CapScreenshot, true, "", "", fmt.Sprintf("decoded PNG %dx%d at %s", frame.Width, frame.Height, frame.Timestamp.UTC().Format(time.RFC3339Nano)))
	d.Tiers = strategy.Tiers(d)
	return d, nil
}

func (a *Adapter) Observe(ctx context.Context) (strategy.Frame, error) {
	if strategy.HostOS == "darwin" {
		data, err := a.captureCommand(ctx, "screencapture", "-x", "-t", "png", "-")
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
	data, err := a.captureCommand(ctx, "import", "-display", a.display, "-window", "root", "png:-")
	if err != nil {
		return strategy.Frame{}, fmt.Errorf("capture host display: %w", err)
	}
	return decodeFrame(data)
}

// boundedCapture avoids buffering unbounded native output. Returning an error
// terminates the pipe copy; CommandContext also imposes a fixed deadline.
type boundedCapture struct{ bytes.Buffer }

func (b *boundedCapture) Write(p []byte) (int, error) {
	if len(p) > maxCaptureBytes-b.Len() {
		return 0, errors.New("capture exceeds byte limit")
	}
	return b.Buffer.Write(p)
}
func (a *Adapter) captureCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, captureTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = time.Second
	var output boundedCapture
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
func decodeFrame(data []byte) (strategy.Frame, error) {
	if len(data) == 0 || len(data) > maxCaptureBytes {
		return strategy.Frame{}, errors.New("invalid capture byte size")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return strategy.Frame{}, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxCapturePixels {
		return strategy.Frame{}, errors.New("capture dimensions exceed limit")
	}
	// A valid header alone is not a successful capture: reject truncated pixels
	// and invalid checksums before reporting readiness.
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
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
