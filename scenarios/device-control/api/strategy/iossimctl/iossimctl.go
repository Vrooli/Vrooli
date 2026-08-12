package iossimctl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"device-control/strategy"
)

type Adapter struct{}

func New() *Adapter           { return &Adapter{} }
func (a *Adapter) ID() string { return "ios-simctl" }
func (a *Adapter) Describe(ctx context.Context) (strategy.Declaration, error) {
	if _, err := exec.LookPath("xcrun"); err != nil {
		next := "Install Xcode and an iOS Simulator runtime on a macOS host node."
		return strategy.UnavailableDeclaration(a.ID(), "iOS Simulator through simctl", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next), nil
	}
	if err := exec.CommandContext(ctx, "xcrun", "simctl", "list", "devices", "available").Run(); err != nil {
		next := "Install an available iOS Simulator runtime in Xcode."
		return strategy.UnavailableDeclaration(a.ID(), "iOS Simulator through simctl", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}}, next), nil
	}
	udid := strings.TrimSpace(os.Getenv("IOS_SIMULATOR_UDID"))
	if udid == "" {
		next := "Select a booted simulator by setting IOS_SIMULATOR_UDID before using ios-simctl."
		return strategy.UnavailableDeclaration(a.ID(), "iOS Simulator through simctl", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next), nil
	}
	caps := map[string]strategy.Capability{}
	for _, n := range []string{strategy.CapInput, strategy.CapScreenshot, strategy.CapSemanticTree, strategy.CapScreenRecording} {
		caps[n] = strategy.ProbeCapability(n, true, "", "", "simctl probe")
	}
	d := strategy.Declaration{StrategyID: a.ID(), Description: "iOS simulators through simctl and XCUITest", Status: strategy.StatusAvailable, Capabilities: caps, Promotable: true}
	d.Tiers = strategy.Tiers(d)
	return d, nil
}

func (a *Adapter) Observe(context.Context) (strategy.Frame, error) {
	return strategy.Frame{}, &strategy.AvailabilityError{Reason: "simulator capture requires a selected simulator UDID", NextAction: "Boot an iOS simulator on a macOS host node."}
}

func (a *Adapter) Actuate(context.Context, strategy.Actuation) error {
	return fmt.Errorf("simulator is unavailable without a selected macOS simulator")
}

var (
	_ = strings.TrimSpace
	_ = time.Now
)
