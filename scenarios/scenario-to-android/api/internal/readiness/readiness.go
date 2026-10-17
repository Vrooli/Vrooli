// Package readiness owns the Google release-readiness ladder.
// It reports what the ramp can prove and leaves account, identity, and Play
// Console actions to the operator.
package readiness

import (
	"fmt"
	"strings"
)

type RungState string

const (
	RungReady       RungState = "ready"
	RungUnavailable RungState = "unavailable"
	RungPending     RungState = "pending"
)

type Rung struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	State       RungState `json:"state"`
	Owner       string    `json:"owner"`
	Automatable bool      `json:"automatable"`
	NextAction  string    `json:"next_action"`
	Obligation  string    `json:"obligation,omitempty"`
}

type Readiness struct {
	Rungs []Rung `json:"rungs"`
}

func GoogleReadiness(hasAccount, verifiedDeveloper, hasSigningReference, targetAPICompliant, internalTrackReady, listingReady bool) Readiness {
	return Readiness{Rungs: []Rung{
		{ID: "play-console-registration", Title: "Play Console registration", State: boolState(hasAccount), Owner: "operator", Automatable: false, NextAction: stateAction(hasAccount, "register the developer account")},
		{ID: "developer-verification", Title: "Developer verification", State: boolState(verifiedDeveloper), Owner: "operator", Automatable: false, NextAction: stateAction(verifiedDeveloper, "complete Google developer verification"), Obligation: "Enforcement begins 30 September 2026 in Brazil, Indonesia, Singapore, and Thailand, and extends globally through 2027."},
		{ID: "play-app-signing", Title: "Signing key and Play App Signing", State: boolState(hasSigningReference), Owner: "operator", Automatable: false, NextAction: stateAction(hasSigningReference, "provision a signing identity in secrets-manager")},
		{ID: "target-api", Title: "Target API compliance", State: boolState(targetAPICompliant), Owner: "ramp", Automatable: true, NextAction: stateAction(targetAPICompliant, "build with targetSdk 36 or higher"), Obligation: "New Android apps and updates must target API 35 by 31 August 2026; eligible extensions run through 1 November 2026."},
		{ID: "internal-testing", Title: "Internal testing track", State: boolState(internalTrackReady), Owner: "operator", Automatable: false, NextAction: stateAction(internalTrackReady, "create an internal testing release")},
		{ID: "production-listing", Title: "Production listing", State: boolState(listingReady), Owner: "operator", Automatable: false, NextAction: stateAction(listingReady, "complete the production listing and review")},
	}}
}

func (r Readiness) Validate() error {
	if len(r.Rungs) != 6 {
		return fmt.Errorf("Google readiness requires 6 rungs")
	}
	seen := make(map[string]bool, len(r.Rungs))
	for _, rung := range r.Rungs {
		if strings.TrimSpace(rung.ID) == "" || seen[rung.ID] || strings.TrimSpace(rung.NextAction) == "" {
			return fmt.Errorf("readiness rung %q is incomplete or duplicated", rung.ID)
		}
		seen[rung.ID] = true
	}
	return nil
}

func boolState(value bool) RungState {
	if value {
		return RungReady
	}
	return RungUnavailable
}

func stateAction(value bool, action string) string {
	if value {
		return "no action required"
	}
	return action
}
