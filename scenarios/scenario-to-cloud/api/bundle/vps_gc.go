package bundle

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

const (
	// DefaultVPSBundleKeepLatest is the default retention policy for the
	// target release store: keep the current release and the rollback
	// predecessor, and prune older staged releases to bound disk growth.
	DefaultVPSBundleKeepLatest = 2
)

// PlanVPSBundleGC decides which inventory rows to keep vs delete.
//
// Inputs:
// - rows are the releases present on the target (any scenario).
// - scenarioID optionally scopes the plan to a single scenario.
// - keepLatest keeps N newest rows (by mod time) per scenario.
// - protectSHA256 are bundle digests that must not be deleted.
//
// Output invariants:
// - kept and deleted are disjoint
// - (kept U deleted) equals the considered set (filtered by scenarioID if provided)
// - a row whose Role is active or previous is always kept
func PlanVPSBundleGC(
	rows []domain.VPSBundleInfo,
	scenarioID string,
	keepLatest int,
	protectSHA256 []string,
) (kept, deleted []domain.VPSBundleInfo, deletedBytes int64) {
	if keepLatest <= 0 {
		keepLatest = DefaultVPSBundleKeepLatest
	}

	shouldConsider := func(b domain.VPSBundleInfo) bool {
		if scenarioID == "" {
			return true
		}
		return b.ScenarioID == scenarioID
	}

	protected := make(map[string]bool, len(protectSHA256))
	for _, sha := range protectSHA256 {
		sha = strings.TrimSpace(sha)
		if sha != "" {
			protected[sha] = true
		}
	}

	byScenario := make(map[string][]domain.VPSBundleInfo)
	for _, b := range rows {
		if !shouldConsider(b) {
			continue
		}
		byScenario[b.ScenarioID] = append(byScenario[b.ScenarioID], b)
	}

	keepKey := make(map[string]bool)
	for _, group := range byScenario {
		sort.Slice(group, func(i, j int) bool {
			return group[i].ModTime > group[j].ModTime // RFC3339 sortable; produced by the owner
		})
		for i := 0; i < len(group) && i < keepLatest; i++ {
			keepKey[group[i].Filename] = true
		}
		for _, b := range group {
			if protected[b.Sha256] || b.Role == "active" || b.Role == "previous" || b.State == "staging" {
				keepKey[b.Filename] = true
			}
		}
	}

	for _, b := range rows {
		if !shouldConsider(b) {
			continue
		}
		if keepKey[b.Filename] {
			kept = append(kept, b)
			continue
		}
		deleted = append(deleted, b)
		deletedBytes += b.SizeBytes
	}

	sort.Slice(kept, func(i, j int) bool { return kept[i].ModTime > kept[j].ModTime })
	sort.Slice(deleted, func(i, j int) bool { return deleted[i].ModTime > deleted[j].ModTime })
	return kept, deleted, deletedBytes
}

// GCTargetReleases prunes retired releases on the target according to the
// retention policy: the owner's listing is read, the plan is computed with
// lease protection, and the owner is asked to prune the named digests
// (unless DryRun). The owner keeps the final say over active and previous.
func GCTargetReleases(
	ctx context.Context,
	r reach.Reach,
	target identity.TargetRef,
	deploymentID string,
	scenarioID string,
	req domain.VPSBundleGCRequest,
) domain.VPSBundleGCResponse {
	now := time.Now().UTC().Format(time.RFC3339)
	if req.KeepLatest <= 0 {
		req.KeepLatest = DefaultVPSBundleKeepLatest
	}

	listing, err := ListTargetReleases(ctx, r, target, deploymentID)
	if err != nil {
		return domain.VPSBundleGCResponse{OK: false, DryRun: req.DryRun, Error: fmt.Sprintf("list releases: %v", err), Timestamp: now}
	}
	before, beforeTotal := InventoryFromListing(listing, scenarioID)

	// Releases the cloud side still owns (active, rollback predecessor) hold
	// artifact leases; their bundles are protected regardless of age. A lease
	// read failure is a refusal, never a silent "nothing protected".
	leased, err := releaseProtectionFn(time.Now().UTC())
	if err != nil {
		return domain.VPSBundleGCResponse{OK: false, DryRun: req.DryRun, Error: fmt.Sprintf("read release leases: %v", err), Timestamp: now}
	}
	protect := append(append([]string(nil), req.ProtectSHA256...), leased...)

	kept, toDelete, deletedBytes := PlanVPSBundleGC(before, req.ScenarioID, req.KeepLatest, protect)

	resp := domain.VPSBundleGCResponse{
		OK:               true,
		DryRun:           req.DryRun,
		BundlesBefore:    before,
		Deleted:          toDelete,
		Kept:             kept,
		DeletedCount:     len(toDelete),
		DeletedBytes:     deletedBytes,
		TotalBeforeBytes: beforeTotal,
		Timestamp:        now,
	}

	if req.DryRun || len(toDelete) == 0 {
		resp.BundlesAfter = before
		resp.TotalAfterBytes = beforeTotal
		if len(toDelete) == 0 {
			resp.Message = "No target releases needed cleanup"
		} else {
			resp.Message = fmt.Sprintf("Would prune %d target release(s)", len(toDelete))
		}
		return resp
	}

	digests := make([]string, 0, len(toDelete))
	for _, b := range toDelete {
		digests = append(digests, b.Filename)
	}
	report, err := PruneTargetReleases(ctx, r, target, deploymentID, digests)
	if err != nil {
		resp.OK = false
		resp.Error = fmt.Sprintf("prune releases: %v", err)
		resp.Message = "Target release cleanup failed"
		return resp
	}
	if len(report.Refused) > 0 {
		// The owner refused some of the plan; the cloud reports the owner's
		// reasons and keeps those rows in the kept set.
		refused := make([]string, 0, len(report.Refused))
		var stillDeleted []domain.VPSBundleInfo
		for _, b := range toDelete {
			if reason, ok := report.Refused[b.Filename]; ok {
				refused = append(refused, b.Filename[:12]+" ("+reason+")")
				kept = append(kept, b)
				continue
			}
			stillDeleted = append(stillDeleted, b)
		}
		resp.Deleted = stillDeleted
		resp.Kept = kept
		resp.DeletedCount = len(stillDeleted)
		resp.DeletedBytes = report.ReclaimedBytes
		resp.Message = fmt.Sprintf("Pruned %d target release(s); owner refused %s", len(stillDeleted), strings.Join(refused, ", "))
	} else {
		resp.DeletedBytes = report.ReclaimedBytes
		resp.Message = fmt.Sprintf("Pruned %d target release(s)", len(report.Deleted))
	}

	after, err := ListTargetReleases(ctx, r, target, deploymentID)
	if err != nil {
		resp.OK = false
		resp.Error = fmt.Sprintf("relist releases: %v", err)
		resp.Message = "Pruned releases but failed to re-list the release store"
		return resp
	}
	resp.BundlesAfter, resp.TotalAfterBytes = InventoryFromListing(after, scenarioID)
	return resp
}
