package iosmirror

import (
	"context"
	"device-control/strategy"
	"fmt"
)

type Adapter struct{}

func New() *Adapter           { return &Adapter{} }
func (a *Adapter) ID() string { return "ios-mirror" }
func (a *Adapter) Describe(context.Context) (strategy.Declaration, error) {
	next := "Pair iPhone Mirroring on a macOS node and grant Accessibility and Screen Recording permissions."
	d := strategy.UnavailableDeclaration(a.ID(), "Physical iPhones through iPhone Mirroring and OCR", []strategy.Capability{{Name: strategy.CapInput, Prerequisite: next, NextAction: next}, {Name: strategy.CapScreenshot, Prerequisite: next, NextAction: next}}, next)
	d.Promotable = false
	d.EvidenceClass = "advisory-ocr"
	return d, nil
}
func (a *Adapter) Observe(context.Context) (strategy.Frame, error) {
	return strategy.Frame{}, &strategy.AvailabilityError{Reason: "iPhone Mirroring is not paired", NextAction: "Pair iPhone Mirroring on a macOS node and grant Accessibility and Screen Recording permissions."}
}
func (a *Adapter) Actuate(context.Context, strategy.Actuation) error {
	return fmt.Errorf("ios-mirror unavailable until pairing and permissions are complete")
}
