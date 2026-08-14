// Package androidrelease owns Android channel and Google readiness models.
// It reports what the ramp can prove and leaves account, identity, and Play
// Console actions to the operator.
package androidrelease

import (
	"context"
	"fmt"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
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

type Channel struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ArtifactKey string `json:"artifact_key"`
	Available   bool   `json:"available"`
	Reason      string `json:"reason"`
	NextAction  string `json:"next_action"`
}

type Distributor struct {
	HasDeveloperVerification bool
	HasSigningReference      bool
	ADB                      func(context.Context, deliveryramp.DistributionRequest) (deliveryramp.DistributionTarget, error)
}

type CellResult struct {
	ID          string
	Required    bool
	Disposition deliveryramp.Disposition
	Target      deliveryramp.Target
	References  []deliveryramp.EvidenceReference
}

// EvaluateGate requires every required cell to pass with producer-owned,
// reference-only evidence. Missing and unavailable cells stay terminal.
func EvaluateGate(cells []CellResult, producer, runID, detail string) (deliveryramp.Disposition, error) {
	if len(cells) == 0 {
		return deliveryramp.DispositionFailed, fmt.Errorf("Android release gate has no validation cells")
	}
	for _, cell := range cells {
		if !cell.Required {
			continue
		}
		if cell.Disposition != deliveryramp.DispositionPass {
			return deliveryramp.DispositionFailed, fmt.Errorf("required Android cell %q is %s", cell.ID, cell.Disposition)
		}
		if _, err := deliveryramp.NewTargetVerdict(deliveryramp.TargetVerdictInput{Producer: producer, Target: cell.Target, Disposition: cell.Disposition, RunID: runID, Detail: detail, References: cell.References}); err != nil {
			return deliveryramp.DispositionFailed, fmt.Errorf("required Android cell %q has invalid evidence: %w", cell.ID, err)
		}
	}
	return deliveryramp.DispositionPass, nil
}

var _ deliveryramp.Distributor = Distributor{}

func (d Distributor) Distribute(ctx context.Context, request deliveryramp.DistributionRequest) (deliveryramp.DistributionResult, error) {
	if strings.TrimSpace(request.Artifact.ImmutableRef) == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "artifact has no immutable identity"}, nil
	}
	targets := []deliveryramp.DistributionTarget{
		{ID: "play", Kind: "play", Available: d.HasDeveloperVerification && d.HasSigningReference, Reason: channelReason(d.HasDeveloperVerification && d.HasSigningReference, "Play requires developer verification and a secrets-manager signing reference")},
		{ID: "verified-sideload", Kind: "verified-sideload", Available: d.HasDeveloperVerification, Reason: channelReason(d.HasDeveloperVerification, "verified sideload requires developer verification")},
	}
	if d.ADB == nil {
		targets = append(targets, deliveryramp.DistributionTarget{ID: "adb-internal", Kind: "adb-internal", Available: false, Reason: "ADB distribution client is not configured"})
	} else {
		target, err := d.ADB(ctx, request)
		if err != nil {
			return deliveryramp.DistributionResult{}, err
		}
		targets = append(targets, target)
	}
	for _, target := range targets {
		if target.Available {
			return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionPass, Targets: targets}, nil
		}
	}
	return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Targets: targets, Reason: "no Android distribution channel is currently available"}, nil
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

func channelReason(available bool, reason string) string {
	if available {
		return "channel prerequisites are present"
	}
	return reason
}
