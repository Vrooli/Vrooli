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
	if attr == nil {
		attr = unknownAttributor{}
	}
	active := make([]CapacityClaim, 0, len(ledger))
	for _, c := range ledger {
		if IsActiveClaimStatus(c.Status) && c.ResourceKind == ResourceKindVRAM {
			active = append(active, c)
		}
	}

	var findings []Finding
	for _, proc := range snapshot.GPUProcesses {
		observed := int64(proc.UsedBytes)
		if observed < policy.TrackingThreshold {
			continue
		}
		a := attr.Attribute(ctx, proc.PID)
		idx := proc.GPUIndex
		finding := Finding{
			ResourceKind:  ResourceKindVRAM,
			GPUIndex:      &idx,
			PID:           proc.PID,
			ProcessName:   proc.ProcessName,
			ObservedBytes: observed,
			OwnerID:       observedOwnerLabel(a, proc),
		}

		claim, matched := matchClaim(active, a, proc)
		if !matched {
			finding.Class = FindingUnclaimed
			finding.Severity = "warn"
			finding.Message = fmt.Sprintf("unclaimed GPU consumer %q (pid %d) using %s on gpu %d holds no capacity claim",
				finding.OwnerID, proc.PID, humanBytes(observed), proc.GPUIndex)
			findings = append(findings, finding)
			continue
		}

		finding.ClaimID = claim.ClaimID
		finding.ClaimedBytes = claim.AmountBytes
		finding.ObservedPeakBytes = claim.ObservedPeakBytes
		finding.OwnerKind = claim.OwnerKind
		finding.OwnerID = claim.OwnerID
		if observed > claim.AmountBytes+policy.ReconcileWarnThreshold {
			finding.Class = FindingOverClaim
			finding.Severity = "warn"
			finding.Message = fmt.Sprintf("owner %q uses %s on gpu %d but claimed only %s (over-claim drift > %s)",
				claim.OwnerID, humanBytes(observed), proc.GPUIndex, humanBytes(claim.AmountBytes), humanBytes(policy.ReconcileWarnThreshold))
		} else {
			finding.Class = FindingClaimed
			finding.Severity = "info"
			finding.Message = fmt.Sprintf("owner %q claim %s covers %s on gpu %d",
				claim.OwnerID, claim.ClaimID, humanBytes(observed), proc.GPUIndex)
		}
		findings = append(findings, finding)
	}
	return findings
}

// matchClaim finds an active claim on the same GPU whose owner matches the
// attribution. Matching is intentionally loose (exact, or either name contains
// the other) because container names and declared owner ids use different
// conventions.
func matchClaim(active []CapacityClaim, a Attribution, proc hostinventory.GPUProcess) (CapacityClaim, bool) {
	candidates := []string{a.OwnerID, NormalizeOwnerName(a.ContainerName), strings.TrimPrefix(a.ContainerName, "/"), NormalizeProcessOwner(proc.ProcessName)}
	for _, c := range active {
		if !sameGPU(c.GPUIndex, &proc.GPUIndex) {
			continue
		}
		owner := strings.TrimSpace(c.OwnerID)
		if owner == "" {
			continue
		}
		for _, cand := range candidates {
			cand = strings.TrimSpace(cand)
			if cand == "" || cand == OwnerUnknown {
				continue
			}
			if ownerMatches(owner, cand) {
				return c, true
			}
		}
	}
	return CapacityClaim{}, false
}

// ownerMatches compares a claim owner id against an attributed candidate. An
// op-scoped owner ("image-tools:job-123") matches its scenario prefix
// ("image-tools").
func ownerMatches(claimOwner, candidate string) bool {
	if claimOwner == candidate {
		return true
	}
	base := claimOwner
	if i := strings.IndexByte(base, ':'); i > 0 {
		base = base[:i]
	}
	if base == candidate {
		return true
	}
	return strings.Contains(candidate, base) || strings.Contains(base, candidate)
}

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
