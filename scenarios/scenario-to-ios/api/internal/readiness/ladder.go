// Package readiness owns the probed Apple delivery ladder.
package readiness

import (
	"fmt"
	"strings"
)

type State string

const (
	Ready       State = "ready"
	Unavailable State = "unavailable"
)

type (
	Rung struct {
		ID                string `json:"id"`
		Title             string `json:"title"`
		State             State  `json:"state"`
		Owner             string `json:"owner"`
		Automatable       bool   `json:"automatable"`
		NextAction        string `json:"next_action"`
		MissingCapability string `json:"missing_capability,omitempty"`
	}
	Probe struct {
		DeveloperProgram bool
		VerifiedIdentity bool
		MacOSBuildHost   bool
		SigningReference bool
		TestFlightAccess bool
		AppStoreListing  bool
	}
	Ladder struct {
		Rungs []Rung `json:"rungs"`
	}
)

// FromProbe derives all six rung states from current observations. No state is
// persisted as a literal, so enrollment changes the next action immediately.
func FromProbe(probe Probe) Ladder {
	return Ladder{Rungs: []Rung{
		rung("developer-program", "Apple Developer Program", probe.DeveloperProgram, "operator", false, "enroll the Apple Developer Program", "apple-developer-program"),
		rung("verified-identity", "Verified developer identity", probe.VerifiedIdentity, "operator", false, "complete Apple developer identity verification", "apple-developer-identity"),
		rung("macos-build-host", "macOS build host", probe.MacOSBuildHost, "operator", false, "register an online macOS bridge node with Xcode", "macos-bridge-node"),
		rung("signing-reference", "Signing identity reference", probe.SigningReference && probe.DeveloperProgram, "operator", false, "provision an App Store signing identity through secrets-manager", "apple-signing-identity"),
		rung("testflight-access", "TestFlight access", probe.TestFlightAccess && probe.SigningReference, "operator", false, "enable TestFlight for the enrolled app", "testflight-access"),
		rung("app-store-listing", "App Store listing", probe.AppStoreListing && probe.TestFlightAccess, "operator", false, "complete App Store Connect listing and review", "app-store-listing"),
	}}
}

func (l Ladder) Validate() error {
	if len(l.Rungs) != 6 {
		return fmt.Errorf("Apple readiness requires six rungs")
	}
	seen := map[string]bool{}
	for _, r := range l.Rungs {
		if r.ID == "" || seen[r.ID] || strings.TrimSpace(r.NextAction) == "" {
			return fmt.Errorf("invalid readiness rung %q", r.ID)
		}
		seen[r.ID] = true
		if r.State == Ready && r.NextAction != "no action required" {
			return fmt.Errorf("ready rung %q has an action", r.ID)
		}
	}
	return nil
}

func rung(id, title string, ready bool, owner string, automatable bool, action, missing string) Rung {
	if ready {
		return Rung{ID: id, Title: title, State: Ready, Owner: owner, Automatable: automatable, NextAction: "no action required"}
	}
	return Rung{ID: id, Title: title, State: Unavailable, Owner: owner, Automatable: automatable, NextAction: action, MissingCapability: missing}
}
