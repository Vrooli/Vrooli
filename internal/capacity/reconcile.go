package capacity

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// Reconcile diffs observed per-process GPU usage (attributed to a
// container/scenario/resource) against the active claim ledger and returns a
// finding per observed consumer above policy.TrackingThreshold (plan §7 Phase
// 2). It only observes and classifies — no enforcement. UNCLAIMED and
// OVER_CLAIM are warn-level; CLAIMED is info.
func Reconcile(ctx context.Context, snapshot hostinventory.Snapshot, ledger []CapacityClaim, attr Attributor, policy Policy) []Finding {
	observations := observeClaimUsage(ctx, ledger, snapshot, attr)
	var findings []Finding
	// A reservation covers the aggregate, not each worker independently.
	for _, c := range ledger {
		workers := observations.assigned[c.ClaimID]
		if len(workers) == 0 {
			continue
		}
		var observed int64
		var pids []int
		for _, proc := range workers {
			observed += int64(proc.UsedBytes)
			pids = append(pids, proc.PID)
		}
		if observed < policy.TrackingThreshold {
			continue
		}
		idx := gpuIndexOf(c)
		f := Finding{Class: FindingClaimed, OwnerID: c.OwnerID, OwnerKind: c.OwnerKind, ResourceKind: ResourceKindVRAM, GPUIndex: &idx, ClaimID: c.ClaimID, ClaimedBytes: c.AmountBytes, ObservedBytes: observed, ObservedPeakBytes: c.ObservedPeakBytes, Severity: "info", PIDs: pids}
		if len(workers) == 1 {
			f.PID = workers[0].PID
			f.ProcessName = workers[0].ProcessName
		}
		if observed > c.AmountBytes+policy.ReconcileWarnThreshold {
			f.Class = FindingOverClaim
			f.Severity = ActionWarn
			f.Message = fmt.Sprintf("owner %q workers %v use %s on gpu %d but reserve %s (aggregate drift > %s)", c.OwnerID, pids, humanBytes(observed), idx, humanBytes(c.AmountBytes), humanBytes(policy.ReconcileWarnThreshold))
		} else {
			f.Message = fmt.Sprintf("owner %q claim %s covers workers %v using %s on gpu %d", c.OwnerID, c.ClaimID, pids, humanBytes(observed), idx)
		}
		findings = append(findings, f)
	}
	for _, unbound := range observations.unbound {
		proc := unbound.proc
		if int64(proc.UsedBytes) < policy.TrackingThreshold {
			continue
		}
		idx := proc.GPUIndex
		findings = append(findings, Finding{Class: FindingUnclaimed, OwnerID: observedOwnerLabel(unbound.attr, proc), ResourceKind: ResourceKindVRAM, GPUIndex: &idx, PID: proc.PID, ProcessName: proc.ProcessName, ObservedBytes: int64(proc.UsedBytes), Severity: ActionWarn, Message: fmt.Sprintf("GPU consumer pid %d has no unique capacity claim association: %s", proc.PID, unbound.reason)})
	}
	return findings
}

type (
	unboundProcess struct {
		proc   hostinventory.GPUProcess
		attr   Attribution
		reason string
	}
	claimUsage struct {
		assigned  map[string][]hostinventory.GPUProcess
		uncertain ExpiryExclusions
		unbound   []unboundProcess
	}
)

// observeClaimUsage assigns each physical process once against the full active
// cohort. Exact normalized identities are evidence; family/prefix names are not
// an operation identity. Ambiguity is unknown presence, never proven absence.
func observeClaimUsage(ctx context.Context, claims []CapacityClaim, snapshot hostinventory.Snapshot, attr Attributor) claimUsage {
	if attr == nil {
		attr = unknownAttributor{}
	}
	result := claimUsage{assigned: map[string][]hostinventory.GPUProcess{}, uncertain: ExpiryExclusions{}}
	for _, proc := range snapshot.GPUProcesses {
		a := attr.Attribute(ctx, proc.PID)
		ids := ownerCandidates(a, proc)
		matches := []CapacityClaim{}
		for _, c := range claims {
			if !IsActiveClaimStatus(c.Status) || c.ResourceKind != ResourceKindVRAM || !sameGPU(c.GPUIndex, &proc.GPUIndex) {
				continue
			}
			if ids[strings.TrimSpace(c.OwnerID)] {
				matches = append(matches, c)
			}
		}
		for _, c := range claims {
			if c.OwnerKind == OwnerKindOp && IsActiveClaimStatus(c.Status) && sameGPU(c.GPUIndex, &proc.GPUIndex) {
				if colon := strings.IndexByte(c.OwnerID, ':'); colon > 0 && ids[c.OwnerID[:colon]] && !ids[c.OwnerID] {
					result.uncertain[c.ClaimID] = "resource family does not identify an operation"
				}
			}
		}
		if len(ids) > 1 {
			reason := "conflicting canonical owner identities"
			for _, c := range matches {
				result.uncertain[c.ClaimID] = reason
			}
			result.unbound = append(result.unbound, unboundProcess{proc: proc, attr: a, reason: reason})
			continue
		}
		if len(matches) == 1 {
			c := matches[0]
			result.assigned[c.ClaimID] = append(result.assigned[c.ClaimID], proc)
			continue
		}
		reason := "no exact owner binding"
		if len(matches) > 1 {
			reason = "ambiguous owner binding"
			for _, c := range matches {
				result.uncertain[c.ClaimID] = reason
			}
		}
		if len(ids) == 0 {
			reason = "process ownership is unknown"
			for _, c := range claims {
				if claimRefreshable(c) && sameGPU(c.GPUIndex, &proc.GPUIndex) {
					result.uncertain[c.ClaimID] = reason
				}
			}
		}
		result.unbound = append(result.unbound, unboundProcess{proc: proc, attr: a, reason: reason})
	}
	return result
}

func ownerCandidates(a Attribution, proc hostinventory.GPUProcess) map[string]bool {
	result := map[string]bool{}
	for _, id := range []string{a.OwnerID, NormalizeOwnerName(a.ContainerName), NormalizeProcessOwner(proc.ProcessName)} {
		id = strings.TrimSpace(id)
		if id != "" && id != OwnerUnknown {
			result[id] = true
		}
	}
	return result
}

// matchClaim also serves declared-resident adoption; ambiguity must not select
// an arbitrary existing row. Sampling and sweep use one full-cohort assignment.
func matchClaim(active []CapacityClaim, a Attribution, proc hostinventory.GPUProcess) (CapacityClaim, bool) {
	ids := ownerCandidates(a, proc)
	if len(ids) != 1 {
		return CapacityClaim{}, false
	}
	var found CapacityClaim
	count := 0
	for _, c := range active {
		if IsActiveClaimStatus(c.Status) && sameGPU(c.GPUIndex, &proc.GPUIndex) && ids[strings.TrimSpace(c.OwnerID)] {
			found = c
			count++
		}
	}
	return found, count == 1
}

func ownerMatches(claimOwner, candidate string) bool { return claimOwner == candidate }

func observedOwnerLabel(a Attribution, proc hostinventory.GPUProcess) string {
	if a.OwnerID != "" && a.OwnerID != OwnerUnknown {
		return a.OwnerID
	}
	if name := strings.TrimPrefix(strings.TrimSpace(a.ContainerName), "/"); name != "" {
		return name
	}
	if proc.ProcessName != "" {
		if owner := NormalizeProcessOwner(proc.ProcessName); owner != "" {
			return owner
		}
		return proc.ProcessName
	}
	return OwnerUnknown
}

// unknownAttributor is the null attributor (every PID is unknown).
type unknownAttributor struct{}

func (unknownAttributor) Attribute(_ context.Context, pid int) Attribution {
	return Attribution{PID: pid, OwnerID: OwnerUnknown}
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
