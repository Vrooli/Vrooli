package iosxcuitest

import (
	"context"
	"fmt"

	"device-control/strategy"
)

type Adapter struct{}

func New() *Adapter           { return &Adapter{} }
func (a *Adapter) ID() string { return "ios-xcuitest" }
func (a *Adapter) Describe(context.Context) (strategy.Declaration, error) {
	if unsupported, ok := strategy.ResolveHostSupport(a.ID(), "Physical iPhones through devicectl and WebDriverAgent", []string{"darwin"}); ok {
		return unsupported, nil
	}
	actions := []string{"Enroll in the Apple Developer Program and configure code signing for WebDriverAgent.", "Attach an iPhone to a trusted macOS host node and authorize the device."}
	return strategy.WithSupportedHostOS(strategy.UnavailableDeclaration(a.ID(), "Physical iPhones through devicectl and WebDriverAgent", []strategy.Capability{{Name: strategy.CapSemanticTree, Status: strategy.StatusUnavailable, Prerequisite: actions[0], NextAction: actions[0]}}, actions...), "darwin"), nil
}

func (a *Adapter) Observe(context.Context) (strategy.Frame, error) {
	return strategy.Frame{}, &strategy.AvailabilityError{Reason: "Apple enrollment and an attached iPhone are required", NextAction: "Enroll in the Apple Developer Program and attach an iPhone to a macOS host node."}
}

func (a *Adapter) Actuate(context.Context, strategy.Actuation) error {
	return fmt.Errorf("ios-xcuitest unavailable until code signing and device attachment are complete")
}
