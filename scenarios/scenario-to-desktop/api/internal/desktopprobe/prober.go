// Package desktopprobe owns desktop-host capability detection for the
// scenario-to-desktop ramp. The delivery spine consumes its observations
// through deliveryramp.Prober and never branches on operating-system names.
package desktopprobe

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type LookPath func(string) (string, error)

type Prober struct {
	LookPath            LookPath
	Getenv              func(string) string
	GOOS                string
	GOARCH              string
	Now                 func() time.Time
	ProbeOfflineNetwork func(context.Context, string) error
}

var _ deliveryramp.Prober = Prober{}

func (p Prober) Probe(_ context.Context, request deliveryramp.ProbeRequest) (deliveryramp.Inventory, error) {
	lookPath := p.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	getenv := p.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	goos := p.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := p.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}

	capabilities := []string{deliveryramp.CapabilityProcessMetrics}
	missing := make([]string, 0, 2)
	if goos == "linux" {
		if _, err := lookPath("xvfb-run"); err == nil || strings.TrimSpace(getenv("DISPLAY")) != "" {
			capabilities = append(capabilities, deliveryramp.CapabilityCDP)
		} else {
			missing = append(missing, "xvfb-run or DISPLAY")
		}
		if _, err := lookPath("xdotool"); err == nil {
			capabilities = append(capabilities, deliveryramp.CapabilityNativeWindow)
		}
		bwrapPath, bwrapErr := lookPath("bwrap")
		_, pythonErr := lookPath("python3")
		if bwrapErr == nil && pythonErr == nil {
			probeIsolation := p.ProbeOfflineNetwork
			if probeIsolation == nil {
				probeIsolation = probeBubblewrapNetworkNamespace
			}
			probeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			probeErr := probeIsolation(probeCtx, bwrapPath)
			cancel()
			if probeErr == nil {
				capabilities = append(capabilities, deliveryramp.CapabilityOfflineNetwork)
			}
		}
	} else {
		missing = append(missing, "supported Linux display runtime")
	}
	if _, err := lookPath("ffmpeg"); err != nil {
		missing = append(missing, "ffmpeg evidence recorder")
	}

	available := len(missing) == 0
	reason := "local target is ready for declared capabilities"
	missingCapability := ""
	nextAction := ""
	if !available {
		reason = fmt.Sprintf("local target prerequisites unavailable: %s", strings.Join(missing, ", "))
		missingCapability = strings.Join(missing, ", ")
		nextAction = "install the missing desktop prerequisites and probe again"
	}
	target := deliveryramp.Target{
		ID:                fmt.Sprintf("local-%s-%s", goos, goarch),
		Ramp:              "scenario-to-desktop",
		Label:             "Local host",
		Platform:          "desktop",
		OS:                goos,
		Architecture:      goarch,
		DeviceKind:        "desktop",
		Transport:         deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "local", Trust: "host", Available: true},
		Capabilities:      capabilities,
		Available:         available,
		Reason:            reason,
		MissingCapability: missingCapability,
		NextAction:        nextAction,
	}
	if err := target.Validate(); err != nil {
		return deliveryramp.Inventory{}, err
	}
	if !matchesRequired(target, request.RequiredCapability) {
		target.Available = false
		target.MissingCapability = missingRequired(target, request.RequiredCapability)
		target.NextAction = "provide the required capability and probe again"
		target.Reason = "local target does not provide all requested capabilities"
	}
	return deliveryramp.Inventory{Targets: []deliveryramp.Target{target}, Observed: now().UTC()}, nil
}

func probeBubblewrapNetworkNamespace(ctx context.Context, bwrapPath string) error {
	command := exec.CommandContext(ctx, bwrapPath, "--unshare-net", "--ro-bind", "/", "/", "--", "/usr/bin/true")
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("bubblewrap network namespace probe failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func matchesRequired(target deliveryramp.Target, required []string) bool {
	for _, capability := range required {
		if !target.Supports(capability) {
			return false
		}
	}
	return true
}

func missingRequired(target deliveryramp.Target, required []string) string {
	missing := make([]string, 0, len(required))
	for _, capability := range required {
		if !target.Supports(capability) {
			missing = append(missing, capability)
		}
	}
	return strings.Join(missing, ", ")
}
