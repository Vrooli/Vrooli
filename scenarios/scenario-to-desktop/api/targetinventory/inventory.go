// Package targetinventory exposes the provider-neutral target capability
// snapshot consumed by desktop validation. It reports what this host can
// actually prove; it does not claim that bridge dispatch provides a desktop
// stream.
package targetinventory

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type Target struct {
	Descriptor   *domainv1.ValidationTargetDescriptor `json:"descriptor"`
	Kind         string                               `json:"kind"`
	NodeID       string                               `json:"node_id,omitempty"`
	OS           string                               `json:"os"`
	Architecture string                               `json:"architecture"`
	Mode         string                               `json:"mode"`
	Reason       string                               `json:"reason,omitempty"`
	Health       TargetHealth                         `json:"health"`
	BridgeTrust  *BridgeTrust                         `json:"bridge_trust,omitempty"`
}

type TargetHealth struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type BridgeTrust struct {
	Registered         bool   `json:"registered"`
	Online             bool   `json:"online"`
	DispatchAuthorized bool   `json:"dispatch_authorized"`
	Reason             string `json:"reason,omitempty"`
}

type Inventory struct {
	Targets []Target `json:"targets"`
}

// BridgeSource supplies trusted-node inventory without making this package
// depend on bridge's generated transport. A source may return an explicit
// unavailable target when the control plane cannot be reached.
type BridgeSource interface {
	Discover(context.Context) ([]Target, error)
}

type LookPath func(string) (string, error)

// LocalProbe is deliberately capability-based. A target is unavailable when
// the host cannot provide the display/evidence prerequisites; callers never
// infer support merely from the operating-system name.
type LocalProbe struct {
	LookPath LookPath
	Getenv   func(string) string
}

func (p LocalProbe) Probe(context.Context) Target {
	lookPath := p.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	getenv := p.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}

	capabilities := []domainv1.ValidationTargetCapability{
		domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_PROCESS_METRICS,
	}
	missing := make([]string, 0, 3)
	if runtime.GOOS == "linux" {
		if _, err := lookPath("xvfb-run"); err == nil || strings.TrimSpace(getenv("DISPLAY")) != "" {
			capabilities = append(capabilities, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_ELECTRON_CDP)
		} else {
			missing = append(missing, "xvfb-run or DISPLAY")
		}
		if _, err := lookPath("xdotool"); err == nil {
			capabilities = append(capabilities, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NATIVE_WINDOW)
		}
	} else {
		missing = append(missing, "supported Linux display runtime")
	}
	if _, err := lookPath("ffmpeg"); err != nil {
		missing = append(missing, "ffmpeg evidence recorder")
	}

	available := len(missing) == 0
	reason := "local target is ready for declared capabilities"
	if !available {
		reason = fmt.Sprintf("local target prerequisites unavailable: %s", strings.Join(missing, ", "))
	}
	targetID := fmt.Sprintf("local-%s-%s", runtime.GOOS, runtime.GOARCH)
	return Target{
		Descriptor: &domainv1.ValidationTargetDescriptor{
			TargetId: targetID, DisplayName: "Local host", Capabilities: capabilities,
			Available: available, Reason: stringPtr(reason),
		},
		Kind:         "local",
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Mode:         "native",
		Reason:       reason,
		Health:       TargetHealth{Status: map[bool]string{true: "healthy", false: "unavailable"}[available], Reason: reason},
		BridgeTrust:  nil,
	}
}

func Discover(ctx context.Context, probe LocalProbe, bridgeSources ...BridgeSource) Inventory {
	result := Inventory{Targets: []Target{probe.Probe(ctx)}}
	for _, source := range bridgeSources {
		if source == nil {
			continue
		}
		targets, err := source.Discover(ctx)
		if err != nil {
			targets = []Target{Target{Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "bridge:unavailable", DisplayName: "Bridge fleet", Available: false, Reason: stringPtr("bridge inventory unavailable")}, Kind: "bridge", Mode: "remote", Reason: "bridge inventory unavailable"}}
		}
		result.Targets = append(result.Targets, targets...)
	}
	return result
}

func stringPtr(value string) *string { return &value }
