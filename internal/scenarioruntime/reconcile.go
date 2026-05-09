package scenarioruntime

import (
	"fmt"
	"time"
)

const (
	ReconcileVerifiedRunning   = "verified_running"
	ReconcileStaleInstance     = "stale_instance"
	ReconcileStaleClaim        = "stale_claim"
	ReconcileOrphanListener    = "orphan_listener"
	ReconcileAdoptionCandidate = "adoption_candidate"
	ReconcileUnverified        = "unverified"
)

type ProcessEvidence struct {
	Known   bool
	Running bool
}

type ListenerEvidence struct {
	Known     bool
	Listening bool
}

type ReconcileInput struct {
	Now           time.Time
	CurrentBootID string
	Instance      Instance
	Claims        []PortClaim
	ProcessRefs   []ProcessRef
	Processes     map[string]ProcessEvidence
	Listeners     map[int]ListenerEvidence
}

type ReconciledClaim struct {
	Claim          PortClaim
	Classification string
	Reason         string
	Authoritative  bool
}

type ReconcileResult struct {
	Instance       Instance
	Classification string
	Reason         string
	Authoritative  bool
	Claims         []ReconciledClaim
}

func ReconcileRuntime(in ReconcileInput) ReconcileResult {
	result := ReconcileResult{
		Instance:       in.Instance,
		Classification: ReconcileVerifiedRunning,
		Authoritative:  true,
	}
	if in.Now.IsZero() {
		in.Now = time.Now().UTC()
	}
	result.Claims = reconcileClaims(in.Claims, true, "")

	if !IsActiveInstanceStatus(in.Instance.Status) {
		return result.fail(ReconcileStaleInstance, fmt.Sprintf("instance status is %q", in.Instance.Status))
	}
	if in.CurrentBootID == "" {
		return result.fail(ReconcileUnverified, "current host boot identity is unavailable")
	}
	if in.Instance.HostBootID == "" {
		return result.fail(ReconcileUnverified, "instance has no host boot identity")
	}
	if in.Instance.HostBootID != in.CurrentBootID {
		return result.fail(ReconcileStaleInstance, "instance was written by a previous host boot")
	}
	if in.Instance.Status == StatusStarting && in.Instance.HeartbeatDeadlineAt != nil && !in.Instance.HeartbeatDeadlineAt.After(in.Now) {
		return result.fail(ReconcileStaleInstance, "starting lease heartbeat deadline has expired")
	}

	deadKnownRefs, liveKnownRefs := processRefEvidence(in)
	if deadKnownRefs > 0 && liveKnownRefs == 0 {
		listening, unknown := listenerSummary(in)
		if !listening && !unknown {
			return result.fail(ReconcileStaleInstance, "all known process refs are dead and no active claim has a listener")
		}
		if !listening && unknown {
			return result.fail(ReconcileUnverified, "all known process refs are dead and listener evidence is unavailable")
		}
	}

	result.Claims = reconcileClaims(in.Claims, true, "")
	for i := range result.Claims {
		if result.Claims[i].Claim.Status == ClaimStatusBound && listenerKnownAbsent(result.Claims[i].Claim, in.Listeners) {
			result.Claims[i].Classification = ReconcileStaleClaim
			result.Claims[i].Authoritative = false
			result.Claims[i].Reason = "bound claim has no listener evidence"
		}
	}
	return result
}

func (r ReconcileResult) fail(classification, reason string) ReconcileResult {
	r.Classification = classification
	r.Reason = reason
	r.Authoritative = false
	r.Claims = reconcileClaims(r.claims(), false, reason)
	return r
}

func (r ReconcileResult) claims() []PortClaim {
	out := make([]PortClaim, 0, len(r.Claims))
	for _, claim := range r.Claims {
		out = append(out, claim.Claim)
	}
	return out
}

func reconcileClaims(claims []PortClaim, authoritative bool, reason string) []ReconciledClaim {
	out := make([]ReconciledClaim, 0, len(claims))
	for _, claim := range claims {
		classification := ReconcileVerifiedRunning
		if !authoritative {
			classification = ReconcileStaleClaim
		}
		out = append(out, ReconciledClaim{
			Claim:          claim,
			Classification: classification,
			Reason:         reason,
			Authoritative:  authoritative,
		})
	}
	return out
}

func processRefEvidence(in ReconcileInput) (deadKnown int, liveKnown int) {
	for _, ref := range in.ProcessRefs {
		if ref.PID == nil {
			continue
		}
		key := fmt.Sprintf("%d", *ref.PID)
		evidence, ok := in.Processes[key]
		if !ok || !evidence.Known {
			continue
		}
		if evidence.Running {
			liveKnown++
		} else {
			deadKnown++
		}
	}
	return deadKnown, liveKnown
}

func listenerSummary(in ReconcileInput) (listening bool, unknown bool) {
	for _, claim := range in.Claims {
		if claim.Status != ClaimStatusBound {
			continue
		}
		evidence, ok := in.Listeners[claim.Port]
		if !ok || !evidence.Known {
			unknown = true
			continue
		}
		if evidence.Listening {
			listening = true
		}
	}
	return listening, unknown
}

func listenerKnownAbsent(claim PortClaim, listeners map[int]ListenerEvidence) bool {
	if claim.Port <= 0 {
		return false
	}
	evidence, ok := listeners[claim.Port]
	return ok && evidence.Known && !evidence.Listening
}
