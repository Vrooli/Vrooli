package bundle

import (
	"context"
	"fmt"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"
	"sort"
	"strings"
	"time"
)

const (
	// DefaultVPSBundleKeepLatest is the default retention policy for VPS bundle cache.
	// Keep current + previous known-good, and delete older bundles to avoid unbounded disk growth.
	DefaultVPSBundleKeepLatest = 2
)

// PlanVPSBundleGC decides which bundles to keep vs delete.
//
// Inputs:
// - bundles must represent the bundles present on the VPS (any scenario).
// - scenarioID optionally scopes the plan to a single scenario.
// - keepLatest keeps N newest bundles (by mod time) per scenario.
// - protectSHA256 are hashes that must not be deleted.
//
// Output invariants:
// - kept and deleted are disjoint
// - (kept U deleted) equals the considered set (filtered by scenarioID if provided)
func PlanVPSBundleGC(
	bundles []domain.VPSBundleInfo,
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

	// Group bundles by scenario to keep N newest per scenario.
	byScenario := make(map[string][]domain.VPSBundleInfo)
	for _, b := range bundles {
		if !shouldConsider(b) {
			continue
		}
		byScenario[b.ScenarioID] = append(byScenario[b.ScenarioID], b)
	}

	keepKey := make(map[string]bool) // filename key (safe, unique enough within bundles dir)

	for _, group := range byScenario {
		sort.Slice(group, func(i, j int) bool {
			return group[i].ModTime > group[j].ModTime // RFC3339 sortable; produced by server
		})
		for i := 0; i < len(group) && i < keepLatest; i++ {
			keepKey[group[i].Filename] = true
		}
		for _, b := range group {
			if protected[b.Sha256] {
				keepKey[b.Filename] = true
			}
		}
	}

	for _, b := range bundles {
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

	// Stable ordering for UX and tests
	sort.Slice(kept, func(i, j int) bool { return kept[i].ModTime > kept[j].ModTime })
	sort.Slice(deleted, func(i, j int) bool { return deleted[i].ModTime > deleted[j].ModTime })
	return kept, deleted, deletedBytes
}

// GCVPSBundles deletes old bundles on the VPS according to the retention policy.
// It lists the remote bundles, computes a plan, then executes the deletion (unless DryRun).
func GCVPSBundles(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	req domain.VPSBundleGCRequest,
) domain.VPSBundleGCResponse {
	now := time.Now().UTC().Format(time.RFC3339)

	if req.KeepLatest <= 0 {
		req.KeepLatest = DefaultVPSBundleKeepLatest
	}

	bundlesPath := shellutil.SafeRemoteJoin(workdir, ".vrooli/cloud/bundles")

	// List remote bundles with size and mod time.
	beforeBundles, beforeTotal, err := listVPSBundlesByPath(ctx, sshRunner, cfg, bundlesPath)
	if err != nil {
		return domain.VPSBundleGCResponse{
			OK:        false,
			DryRun:    req.DryRun,
			Error:     fmt.Sprintf("list bundles: %v", err),
			Timestamp: now,
		}
	}

	kept, toDelete, deletedBytes := PlanVPSBundleGC(beforeBundles, req.ScenarioID, req.KeepLatest, req.ProtectSHA256)

	resp := domain.VPSBundleGCResponse{
		OK:               true,
		DryRun:           req.DryRun,
		BundlesBefore:    beforeBundles,
		Deleted:          toDelete,
		Kept:             kept,
		DeletedCount:     len(toDelete),
		DeletedBytes:     deletedBytes,
		TotalBeforeBytes: beforeTotal,
		Timestamp:        now,
	}

	if req.DryRun || len(toDelete) == 0 {
		resp.BundlesAfter = beforeBundles
		resp.TotalAfterBytes = beforeTotal
		if len(toDelete) == 0 {
			resp.Message = "No VPS bundles needed cleanup"
		} else {
			resp.Message = fmt.Sprintf("Would delete %d VPS bundle(s)", len(toDelete))
		}
		return resp
	}

	// Validate filenames to avoid traversal, then delete.
	var filenames []string
	for _, b := range toDelete {
		if !isSafeBundleFilename(b.Filename) {
			return domain.VPSBundleGCResponse{
				OK:        false,
				DryRun:    req.DryRun,
				Error:     fmt.Sprintf("refusing to delete unsafe filename %q", b.Filename),
				Timestamp: now,
			}
		}
		filenames = append(filenames, b.Filename)
	}

	// Build a single rm command for efficiency (no xargs; avoid shell quoting surprises).
	// Use "--" to prevent filenames starting with "-" from being interpreted as flags.
	// Batch deletions to avoid hitting shell argument length limits on long-lived VPSes.
	const batchSize = 50
	for i := 0; i < len(filenames); i += batchSize {
		end := i + batchSize
		if end > len(filenames) {
			end = len(filenames)
		}
		batch := filenames[i:end]
		quoted := make([]string, 0, len(batch))
		for _, f := range batch {
			quoted = append(quoted, shellutil.QuoteSingle(f))
		}
		deleteCmd := fmt.Sprintf("cd %s 2>/dev/null && rm -f -- %s", shellutil.QuoteSingle(bundlesPath), strings.Join(quoted, " "))
		if _, err := sshRunner.Run(ctx, cfg, deleteCmd, ssh.DefaultRunOptions()); err != nil {
			resp.OK = false
			resp.Error = fmt.Sprintf("delete bundles: %v", err)
			resp.Message = "VPS bundle cleanup failed"
			return resp
		}
	}

	afterBundles, afterTotal, listErr := listVPSBundlesByPath(ctx, sshRunner, cfg, bundlesPath)
	if listErr != nil {
		// Deletion may have succeeded; return plan + partial info.
		resp.OK = false
		resp.Error = fmt.Sprintf("relist bundles: %v", listErr)
		resp.Message = "Deleted bundles but failed to re-list bundle cache"
		return resp
	}

	resp.BundlesAfter = afterBundles
	resp.TotalAfterBytes = afterTotal
	resp.Message = fmt.Sprintf("Deleted %d VPS bundle(s)", len(toDelete))
	return resp
}

func isSafeBundleFilename(name string) bool {
	// Keep this strict: only allow the known bundle filename pattern.
	// Example: mini-vrooli_landing-page-business-suite_<sha256>.tar.gz
	if !strings.HasPrefix(name, "mini-vrooli_") || !strings.HasSuffix(name, ".tar.gz") {
		return false
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return false
	}
	return true
}
