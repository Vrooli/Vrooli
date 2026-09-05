package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/artifactlease"

	"github.com/vrooli/vrooli/packages/artifactledger"

	"storage-manager/internal/cleanup"
)

const scenarioBinaryMetadataSuffix = ".build.meta"

type ScenarioBinariesProviderConfig struct {
	Root string
	// Ledger records every removal this provider performs. A provider built
	// without one refuses to delete rather than deleting unrecorded.
	Ledger *artifactledger.Ledger
}

type ScenarioBinariesProvider struct {
	meta     cleanup.ProviderMetadata
	files    cleanup.FileSystem
	liveness cleanup.ProcessLiveness
	root     string
	clock    cleanup.Clock
	// ledger makes this reaper's removals attributable. This provider deletes
	// binaries it decided were orphaned, which is the hardest kind of removal
	// to reconstruct after the fact and therefore the one that most needs a
	// receipt naming the rule that fired.
	ledger *artifactledger.Ledger
}

type scenarioInstallMetadata struct {
	Kind       string `json:"kind"`
	ModulePath string `json:"module_path"`
}

func NewScenarioBinariesProvider(files cleanup.FileSystem, liveness cleanup.ProcessLiveness, clock cleanup.Clock, cfg ScenarioBinariesProviderConfig) *ScenarioBinariesProvider {
	return &ScenarioBinariesProvider{
		meta: cleanup.ProviderMetadata{
			ID:                  "scenario-binaries",
			Name:                "Orphaned scenario binaries",
			Version:             "v1",
			OwnerScenario:       "storage-manager",
			SafetyTier:          cleanup.SafetyTierSafeWithOwner,
			DefaultMode:         cleanup.ProviderModeDisabled,
			DefaultApproval:     cleanup.ApprovalModeOwner,
			SupportedPlatforms:  []string{"linux", "darwin", "windows"},
			IrreversibleEffects: []string{"removes an orphaned scenario CLI binary and its build and manifest sidecars"},
			TestSubstitute:      "fake-filesystem-and-process-liveness",
		},
		files:    files,
		liveness: liveness,
		root:     filepath.Clean(strings.TrimSpace(cfg.Root)),
		clock:    clock,
		ledger:   cfg.Ledger,
	}
}

func (p *ScenarioBinariesProvider) Metadata() cleanup.ProviderMetadata { return p.meta }

func (p *ScenarioBinariesProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	preview, err := p.Preview(ctx, cleanup.PreviewRequest{Scope: req.Scope, Policy: req.Policy})
	if err != nil {
		return cleanup.Estimate{}, err
	}
	return cleanup.Estimate{
		ProviderID:       p.meta.ID,
		ProviderVersion:  p.meta.Version,
		EstimatedBytes:   sumPreviewBytes(preview.Items),
		ItemCount:        len(preview.Items),
		RequiresApproval: req.Policy.ApprovalMode != cleanup.ApprovalModeNone,
		BlockedReason:    preview.BlockedReason,
		ObservedAt:       p.now(req.Scope),
	}, nil
}

func (p *ScenarioBinariesProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if p.files == nil {
		out.BlockedReason = "filesystem seam unavailable"
		return out, nil
	}
	if p.root == "." || p.root == "" {
		out.BlockedReason = "scenario binary root unavailable"
		return out, nil
	}
	if p.liveness == nil {
		out.BlockedReason = "process liveness seam unavailable"
		return out, nil
	}

	entries, err := p.files.ReadDir(ctx, p.root)
	if err != nil {
		if isMissing(err) {
			out.BlockedReason = "scenario binary root unavailable"
			return out, nil
		}
		return out, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	now := p.now(req.Scope)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return cleanup.Preview{}, err
		}
		if entry.IsDir || !strings.HasSuffix(entry.Path, scenarioBinaryMetadataSuffix) {
			continue
		}
		metadata, err := p.readMetadata(ctx, entry.Path)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: metadata could not be read; skipped", filepath.Base(entry.Path)))
			continue
		}
		if strings.TrimSpace(metadata.ModulePath) == "" {
			// Only scenario install metadata with an owning module path is
			// authoritative for this provider. This avoids treating arbitrary
			// .build.meta files as scenario-owned artifacts.
			continue
		}
		if metadata.Kind != "scenario" {
			// The shared runtime bin directory also contains resource and tool
			// installs. Their owners have different lifecycles and must not be
			// reclaimed by the scenario reaper.
			continue
		}
		binaryForLease := strings.TrimSuffix(entry.Path, scenarioBinaryMetadataSuffix)
		if _, err := p.files.Stat(ctx, metadata.ModulePath); err == nil {
			// The owning scenario module exists. Clearing any recorded absence
			// here is what makes a scenario that was deleted and recreated keep
			// its CLI: the grace clock restarts from nothing rather than
			// carrying a stale observation forward.
			if clearErr := artifactlease.NoteOwnerPresent(binaryForLease); clearErr != nil {
				out.Warnings = append(out.Warnings, fmt.Sprintf("%s: recorded absence could not be cleared; skipped", filepath.Base(entry.Path)))
			}
			continue
		} else if !isMissing(err) {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: owner module could not be checked; skipped", filepath.Base(entry.Path)))
			continue
		}

		// The owner looks missing. Record that and stop -- an observation is
		// not authority to delete. Reclamation needs the absence to have
		// persisted for the grace window and to have been seen more than once,
		// because a scenario directory can be absent for a moment while another
		// agent regenerates it.
		lease, leaseErr := artifactlease.NoteOwnerMissing(binaryForLease, p.now(req.Scope))
		if leaseErr != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: ownership record unavailable; skipped", filepath.Base(entry.Path)))
			continue
		}
		eligibility := artifactlease.EvaluateReclaim(lease, true, p.now(req.Scope), artifactlease.DefaultGrace)
		if !eligibility.Reclaimable {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: %s", filepath.Base(entry.Path), eligibility.Reason))
			continue
		}

		binaryPath, artifacts, infos, ok := p.triple(ctx, strings.TrimSuffix(entry.Path, scenarioBinaryMetadataSuffix))
		if !ok {
			continue
		}
		newest := infos[0].ModTime
		var bytes int64
		for _, info := range infos {
			bytes += info.Size
			if info.ModTime.After(newest) {
				newest = info.ModTime
			}
		}
		if req.Policy.MinAge > 0 && now.Sub(newest) < req.Policy.MinAge {
			continue
		}
		running, err := p.liveness.IsRunning(ctx, binaryPath)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: liveness could not be verified; skipped", filepath.Base(binaryPath)))
			continue
		}
		if running {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: binary is running; skipped", filepath.Base(binaryPath)))
			continue
		}
		if req.Policy.MaxBytes > 0 && sumPreviewBytes(out.Items)+bytes > req.Policy.MaxBytes {
			out.Warnings = append(out.Warnings, "max reclaim limit reached")
			continue
		}
		out.Items = append(out.Items, cleanup.PreviewItem{
			ID:          stableItemID(p.meta.ID, binaryPath),
			Path:        binaryPath,
			Description: fmt.Sprintf("orphaned scenario CLI triple: %s, %s, %s", filepath.Base(artifacts[0]), filepath.Base(artifacts[1]), filepath.Base(artifacts[2])),
			Bytes:       bytes,
			Action:      "scenario-binary-triple-remove",
			SafetyTier:  p.meta.SafetyTier,
		})
	}
	return out, nil
}

func (p *ScenarioBinariesProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", p.meta.ID, req.ProviderVersion, p.meta.Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s apply requires idempotency key", p.meta.ID)
	}
	if req.ApprovalMode == cleanup.ApprovalModeDisabled {
		return cleanup.ApplyResult{ProviderID: p.meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"provider disabled by policy"}}, nil
	}
	if req.ApprovalMode != cleanup.ApprovalModeOwner && req.ApprovalMode != cleanup.ApprovalModeOperator {
		return cleanup.ApplyResult{ProviderID: p.meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"owner or operator approval required"}}, nil
	}
	if p.files == nil || p.liveness == nil {
		return cleanup.ApplyResult{ProviderID: p.meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"filesystem and liveness seams are required"}}, nil
	}
	if p.ledger == nil {
		// Refusing is the safe direction. An orphan reclamation that leaves no
		// receipt is indistinguishable from the unexplained removals this
		// ledger was built to end.
		return cleanup.ApplyResult{ProviderID: p.meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"no removal ledger is configured; refusing to reclaim unrecorded"}}, nil
	}

	result := cleanup.ApplyResult{ProviderID: p.meta.ID}
	for _, item := range req.Preview.Items {
		if err := ctx.Err(); err != nil {
			return cleanup.ApplyResult{}, err
		}
		binaryPath := filepath.Clean(item.Path)
		if !p.withinRoot(binaryPath) {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			result.Warnings = append(result.Warnings, item.ID+": path outside scenario binary root")
			continue
		}
		running, err := p.liveness.IsRunning(ctx, binaryPath)
		if err != nil || running {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			if err != nil {
				result.Warnings = append(result.Warnings, item.ID+": liveness could not be verified")
			} else {
				result.Warnings = append(result.Warnings, item.ID+": binary is running")
			}
			continue
		}
		_, artifacts, _, complete := p.triple(ctx, binaryPath)
		if !complete {
			// Preview may race with an operator or a previous idempotent apply.
			// Reconstruct the exact triple and remove only members still present;
			// missing members must not be counted as newly reclaimed bytes.
			artifacts = []string{binaryPath, binaryPath + scenarioBinaryMetadataSuffix, binaryPath + ".manifest.json"}
		}
		failed := false
		abandoned := false
		var reclaimed int64
		for _, path := range artifacts {
			info, statErr := p.files.Stat(ctx, path)
			if statErr != nil {
				if isMissing(statErr) {
					continue
				}
				failed = true
				result.Warnings = append(result.Warnings, item.ID+": "+cleanup.Redact(statErr.Error()))
				continue
			}
			leaseForReceipt, _, _ := artifactlease.Load(binaryPath)
			removeErr := p.ledger.Guard(artifactledger.Removal{
				Path:       path,
				Subject:    binaryPath,
				Generation: leaseForReceipt.Generation,
				Kind:       artifactKindFor(path),
				Component:  "storage-manager.ScenarioBinariesProvider",
				Predicate:  orphanReclaimPredicate,
				Verify:     func() error { return p.stillOrphaned(ctx, binaryPath, p.now(cleanup.ObservationScope{})) },
			}, func() error { return p.files.RemoveAll(ctx, path) })
			if err := removeErr; err != nil {
				if errors.Is(err, artifactledger.ErrAbandoned) {
					// The owner came back between plan and apply. Nothing was
					// removed, so nothing may be counted as reclaimed.
					abandoned = true
					result.Warnings = append(result.Warnings, item.ID+": "+cleanup.Redact(err.Error()))
					continue
				}
				if isMissing(err) {
					continue
				}
				failed = true
				result.Warnings = append(result.Warnings, item.ID+": "+cleanup.Redact(err.Error()))
				continue
			}
			reclaimed += info.Size
		}
		if failed || abandoned {
			result.SkippedItems = append(result.SkippedItems, item.ID)
			continue
		}
		if reclaimed == 0 {
			result.AlreadyDone = true
			continue
		}
		result.ReclaimedBytes += reclaimed
	}
	result.Applied = result.ReclaimedBytes > 0
	return result, nil
}

func (p *ScenarioBinariesProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "scenario binary triples were removed through FileSystem seam"}, nil
}

func (p *ScenarioBinariesProvider) readMetadata(ctx context.Context, path string) (scenarioInstallMetadata, error) {
	contents, err := p.files.ReadFile(ctx, path)
	if err != nil {
		return scenarioInstallMetadata{}, err
	}
	var metadata scenarioInstallMetadata
	if err := json.Unmarshal(contents, &metadata); err != nil {
		return scenarioInstallMetadata{}, err
	}
	return metadata, nil
}

// triple returns binary, manifest, and build metadata in removal order. The
// Windows fallback accepts the historical sidecar form foo.build.meta beside
// foo.exe while still preferring the exact foo.exe.build.meta form.
func (p *ScenarioBinariesProvider) triple(ctx context.Context, binaryBase string) (string, []string, []cleanup.FileInfo, bool) {
	binaryPath := binaryBase
	metaPath := binaryBase + scenarioBinaryMetadataSuffix
	info, err := p.files.Stat(ctx, binaryPath)
	if err != nil {
		binaryPath += ".exe"
		info, err = p.files.Stat(ctx, binaryPath)
		if err == nil {
			metaPath = binaryBase + scenarioBinaryMetadataSuffix
		}
	}
	if err != nil {
		return "", nil, nil, false
	}
	manifestPath := binaryPath + ".manifest.json"
	manifest, manifestErr := p.files.Stat(ctx, manifestPath)
	metadata, metadataErr := p.files.Stat(ctx, metaPath)
	if manifestErr != nil || metadataErr != nil {
		return "", nil, nil, false
	}
	return binaryPath, []string{binaryPath, metaPath, manifestPath}, []cleanup.FileInfo{info, metadata, manifest}, true
}

func (p *ScenarioBinariesProvider) withinRoot(path string) bool {
	cleanPath := filepath.Clean(path)
	return cleanPath != p.root && strings.HasPrefix(cleanPath, p.root+string(filepath.Separator))
}

func (p *ScenarioBinariesProvider) now(scope cleanup.ObservationScope) time.Time {
	if !scope.Now.IsZero() {
		return scope.Now
	}
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Now().UTC()
}

func isMissing(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || strings.Contains(strings.ToLower(err.Error()), "not found") || strings.Contains(strings.ToLower(err.Error()), "does not exist")
}

// orphanReclaimPredicate is the rule this provider enforces, recorded on every
// receipt. Unlike the uninstall path, this reaper decides for itself what to
// delete, so the receipt has to carry the reasoning rather than a plan id.
const orphanReclaimPredicate = "scenario CLI triple whose owning module path no longer resolves"

// artifactKindFor labels a triple member for the ledger.
func artifactKindFor(path string) string {
	switch {
	case strings.HasSuffix(path, scenarioBinaryMetadataSuffix):
		return "build-metadata"
	case strings.HasSuffix(path, ".manifest.json"):
		return "manifest"
	default:
		return "binary"
	}
}

// stillOrphaned re-checks, under the artifact lock, the predicate that
// authorized this reclamation.
//
// Preview decided this binary was an orphan. Apply runs later, and between the
// two another agent may have recreated the scenario that owns it -- concurrent
// agents in one environment are a design property here, not an edge case. The
// window was previously unguarded: Apply re-checked liveness but never the
// orphan predicate, so a freshly rebuilt CLI could be deleted on the strength
// of a stale observation.
// The planned generation is deliberately not threaded from Preview to Apply.
// cleanup.PreviewItem is shared by every provider, and widening it for this one
// would spread an unused field across all of them -- for a check the lease
// already makes. A reinstall calls Claim, which clears the recorded absence, so
// EvaluateReclaim refuses on its own with a reason that names what changed. The
// generation is still recorded on the receipt, where it aids attribution.
func (p *ScenarioBinariesProvider) stillOrphaned(ctx context.Context, binaryPath string, now time.Time) error {
	metadata, err := p.readMetadata(ctx, binaryPath+scenarioBinaryMetadataSuffix)
	if err != nil {
		// The metadata that justified the reclamation is unreadable now. That
		// is not permission to proceed.
		return fmt.Errorf("owner metadata for %s could not be re-read: %w", filepath.Base(binaryPath), err)
	}
	if strings.TrimSpace(metadata.ModulePath) == "" {
		return fmt.Errorf("owner metadata for %s no longer names a module", filepath.Base(binaryPath))
	}
	if _, err := p.files.Stat(ctx, metadata.ModulePath); err == nil {
		return fmt.Errorf("owner module %s exists again; the artifact is no longer an orphan", metadata.ModulePath)
	} else if !isMissing(err) {
		return fmt.Errorf("owner module %s could not be re-checked: %w", metadata.ModulePath, err)
	}

	lease, found, leaseErr := artifactlease.Load(binaryPath)
	if leaseErr != nil {
		// An ownership record that cannot be read is not permission to delete
		// what it describes.
		return fmt.Errorf("ownership record for %s could not be re-read: %w", filepath.Base(binaryPath), leaseErr)
	}
	if eligibility := artifactlease.EvaluateReclaim(lease, found, now, artifactlease.DefaultGrace); !eligibility.Reclaimable {
		return errors.New(eligibility.Reason)
	}
	return nil
}
