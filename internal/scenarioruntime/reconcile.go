package scenarioruntime

import (
	"fmt"
	"strconv"
	"time"
)

const (
	ReconcileVerifiedRunning   ReconcileClassification = "verified_running"
	ReconcileStaleInstance     ReconcileClassification = "stale_instance"
	ReconcileStaleClaim        ReconcileClassification = "stale_claim"
	ReconcileOrphanListener    ReconcileClassification = "orphan_listener"
	ReconcileAdoptionCandidate ReconcileClassification = "adoption_candidate"
	ReconcileUnverified        ReconcileClassification = "unverified"
)

type ReconcileClassification string

type ProcessEvidence struct {
	Known   bool
	Running bool
}

type ListenerEvidence struct {
	Known        bool
	Listening    bool
	PID          *int
	ProcessLabel string
}

type PIDRunningFunc func(pid int) bool

type PortListenerFunc func(port int) ListenerEvidence

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
	Classification ReconcileClassification
	Reason         string
	Authoritative  bool
}

type ReconcileResult struct {
	Instance       Instance
	Classification ReconcileClassification
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
	// Heartbeat-only staleness is no longer sufficient: long-running setup
	// or develop phases (e.g. web-console's UI build at 4400+ vite modules)
	// routinely run past the 30s heartbeat TTL while the owner process is
	// still alive and making forward progress. Treating those as stale
	// caused parallel-restart invocations and on-demand reapers to release
	// in-flight startup's port claims, surfacing as bind UNIQUE/"claim is
	// no longer reservable" errors. Require corroborating evidence — owner
	// PID known-dead or missing — before classifying as stale. This
	// mirrors the conservative classifier used by ports.preemptFixedPortConflict.
	if in.Instance.HeartbeatDeadlineAt != nil && !in.Instance.HeartbeatDeadlineAt.After(in.Now) &&
		isOwnerKnownDead(in.Instance.OwnerPID, in.Processes) {
		switch in.Instance.Status {
		case StatusStarting:
			return result.fail(ReconcileStaleInstance, "starting lease heartbeat deadline has expired (owner pid known-dead)")
		case StatusRunning:
			if in.Instance.SupervisorID != "" {
				return result.fail(ReconcileStaleInstance, "supervised running lease heartbeat deadline has expired (owner pid known-dead)")
			}
		}
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

func ProcessEvidenceFromRefs(refs []ProcessRef, isRunning PIDRunningFunc) map[string]ProcessEvidence {
	out := make(map[string]ProcessEvidence)
	if isRunning == nil {
		return out
	}
	for _, ref := range refs {
		if ref.PID == nil {
			continue
		}
		pid := *ref.PID
		out[strconv.Itoa(pid)] = ProcessEvidence{Known: true, Running: isRunning(pid)}
	}
	return out
}

func ListenerEvidenceFromClaims(claims []PortClaim, refs []ProcessRef, inspect PortListenerFunc) map[int]ListenerEvidence {
	if inspect == nil {
		return nil
	}
	out := make(map[int]ListenerEvidence)
	for _, claim := range claims {
		if !IsDiscoverablePortClaimStatus(claim.Status) || claim.Port <= 0 {
			continue
		}
		out[claim.Port] = inspect(claim.Port)
	}
	return out
}

func ListenerObservationFromEvidence(checkedAt time.Time, evidence ListenerEvidence) ListenerObservation {
	status := ListenerStatusUnknown
	if evidence.Known {
		if evidence.Listening {
			status = ListenerStatusListening
		} else {
			status = ListenerStatusNotListening
		}
	}
	return ListenerObservation{
		CheckedAt:    checkedAt,
		Status:       status,
		PID:          evidence.PID,
		ProcessLabel: evidence.ProcessLabel,
	}
}

func (r ReconcileResult) fail(classification ReconcileClassification, reason string) ReconcileResult {
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
		claimAuthoritative := authoritative && IsDiscoverablePortClaimStatus(claim.Status)
		claimReason := reason
		classification := ReconcileVerifiedRunning
		if !claimAuthoritative {
			classification = ReconcileStaleClaim
			if claimReason == "" {
				claimReason = fmt.Sprintf("claim status is %q", claim.Status)
			}
		}
		out = append(out, ReconciledClaim{
			Claim:          claim,
			Classification: classification,
			Reason:         claimReason,
			Authoritative:  claimAuthoritative,
		})
	}
	return out
}

// isOwnerKnownDead returns true only when the input.Processes map has
// positive evidence that the owner PID is no longer running. Absence of
// evidence is NOT enough: the orchestrator/runner PID typically has no
// process_ref entry, so the Processes map will not contain it, and we
// must NOT treat that absence as a stale signal — doing so caused the
// in-flight-startup release bug. Boot mismatch and dead known refs
// remain separate stale signals upstream of this check.
func isOwnerKnownDead(ownerPID *int, processes map[string]ProcessEvidence) bool {
	if ownerPID == nil || *ownerPID <= 0 {
		return true
	}
	key := fmt.Sprintf("%d", *ownerPID)
	evidence, ok := processes[key]
	if !ok || !evidence.Known {
		return false
	}
	return !evidence.Running
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
		if !IsDiscoverablePortClaimStatus(claim.Status) {
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
