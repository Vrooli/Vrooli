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

	"storage-manager/internal/cleanup"
)

const scenarioBinaryMetadataSuffix = ".build.meta"

type ScenarioBinariesProviderConfig struct {
	Root string
}

type ScenarioBinariesProvider struct {
	meta     cleanup.ProviderMetadata
	files    cleanup.FileSystem
	liveness cleanup.ProcessLiveness
	root     string
	clock    cleanup.Clock
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
		if _, err := p.files.Stat(ctx, metadata.ModulePath); err == nil {
			// The owning scenario module still exists; this is not an orphan.
			continue
		} else if !isMissing(err) {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: owner module could not be checked; skipped", filepath.Base(entry.Path)))
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
			if err := p.files.RemoveAll(ctx, path); err != nil {
				if isMissing(err) {
					continue
				}
				failed = true
				result.Warnings = append(result.Warnings, item.ID+": "+cleanup.Redact(err.Error()))
				continue
			}
			reclaimed += info.Size
		}
		if failed {
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
