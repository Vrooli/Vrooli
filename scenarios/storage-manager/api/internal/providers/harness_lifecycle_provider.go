package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/agentcatalog/harness"
	harnesslifecycle "github.com/vrooli/vrooli/packages/harness-lifecycle"
	"storage-manager/internal/cleanup"
)

// HarnessLifecycleProvider delegates ownership decisions and deletion to the
// shared harness lifecycle package. Storage Manager supplies preview, approval
// and audit surfaces; it does not scan or delete harness paths itself.
type HarnessLifecycleProvider struct {
	meta      cleanup.ProviderMetadata
	files     cleanup.FileSystem
	clock     cleanup.Clock
	root      string
	resources []string
}

func NewHarnessLifecycleProvider(files cleanup.FileSystem, clock cleanup.Clock, root string) *HarnessLifecycleProvider {
	resources := make([]string, 0)
	for _, descriptor := range harness.Active().All() {
		resources = append(resources, descriptor.Resource)
	}
	sort.Strings(resources)
	return &HarnessLifecycleProvider{
		meta:  cleanup.ProviderMetadata{ID: "coding-agent-lifecycle", Name: "Managed coding-agent copies", Version: "v1", OwnerScenario: "storage-manager", SafetyTier: cleanup.SafetyTierSafeWithOwner, DefaultMode: cleanup.ProviderModeDisabled, DefaultApproval: cleanup.ApprovalModeOwner, SupportedPlatforms: []string{"linux", "darwin", "windows"}, IrreversibleEffects: []string{"removes receipt-verified obsolete Vrooli-managed versions while retaining the active version and one rollback"}, TestSubstitute: "isolated-home-and-fixture-binaries"},
		files: files, clock: clock, root: root, resources: resources,
	}
}

func (p *HarnessLifecycleProvider) Metadata() cleanup.ProviderMetadata { return p.meta }

func (p *HarnessLifecycleProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	preview, err := p.Preview(ctx, cleanup.PreviewRequest{Scope: req.Scope, Policy: req.Policy})
	if err != nil {
		return cleanup.Estimate{}, err
	}
	var bytes int64
	for _, item := range preview.Items {
		bytes += item.Bytes
	}
	return cleanup.Estimate{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version, EstimatedBytes: bytes, ItemCount: len(preview.Items), RequiresApproval: req.Policy.ApprovalMode != cleanup.ApprovalModeNone, BlockedReason: preview.BlockedReason, ObservedAt: p.now(req.Scope)}, nil
}

func (p *HarnessLifecycleProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version}
	if !req.Policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if p.root == "" || p.files == nil {
		out.BlockedReason = "lifecycle root or filesystem seam unavailable"
		return out, nil
	}
	for _, resource := range p.resources {
		if err := ctx.Err(); err != nil {
			return cleanup.Preview{}, err
		}
		plan, err := harnesslifecycle.PreviewPruneOldManaged(p.root, resource, 2)
		if err != nil {
			out.Warnings = append(out.Warnings, resource+": preview unavailable")
			continue
		}
		identity, _ := json.Marshal(struct{ Resource, ID string }{resource, plan.ID})
		resourceItems := make([]cleanup.PreviewItem, 0, len(plan.Remove))
		measurementFailed := false
		for _, path := range plan.Remove {
			info, err := p.files.Stat(ctx, path)
			if err != nil {
				measurementFailed = true
				break
			}
			var size int64
			if info.IsDir {
				if err := p.files.Walk(ctx, path, func(item cleanup.FileInfo) error {
					if !item.IsDir {
						size += item.Size
					}
					return nil
				}); err != nil {
					measurementFailed = true
					break
				}
			} else {
				size = info.Size
			}
			resourceItems = append(resourceItems, cleanup.PreviewItem{ID: resource + ":" + path, Path: path, Description: "Receipt-verified managed coding-agent artifact", Bytes: size, Action: "remove_verified_harness_artifact", SafetyTier: cleanup.SafetyTierSafeWithOwner, Identity: string(identity), ModTime: info.ModTime})
		}
		if measurementFailed {
			out.Warnings = append(out.Warnings, resource+": an owned path could not be measured completely; no removal items were proposed")
		} else {
			out.Items = append(out.Items, resourceItems...)
		}
		for _, path := range plan.Preserved {
			out.Warnings = append(out.Warnings, resource+": found copy is preserved at "+path)
		}
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].Path < out.Items[j].Path })
	return out, nil
}

func (p *HarnessLifecycleProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider version mismatch")
	}
	if req.ApprovalMode != cleanup.ApprovalModeOwner && req.ApprovalMode != cleanup.ApprovalModeOperator {
		return cleanup.ApplyResult{ProviderID: p.meta.ID, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"owner or operator approval required"}}, nil
	}
	plans := map[string]string{}
	paths := map[string][]string{}
	for _, item := range req.Preview.Items {
		var value struct{ Resource, ID string }
		if json.Unmarshal([]byte(item.Identity), &value) != nil || value.Resource == "" || value.ID == "" {
			return cleanup.ApplyResult{}, fmt.Errorf("cleanup preview identity is invalid")
		}
		plans[value.Resource] = value.ID
		paths[value.Resource] = append(paths[value.Resource], item.Path)
	}
	result := cleanup.ApplyResult{ProviderID: p.meta.ID}
	for resource, id := range plans {
		if err := ctx.Err(); err != nil {
			return cleanup.ApplyResult{}, err
		}
		current, err := harnesslifecycle.PreviewPruneOldManaged(p.root, resource, 2)
		if err != nil {
			return cleanup.ApplyResult{}, err
		}
		plannedPaths := append([]string(nil), paths[resource]...)
		sort.Strings(plannedPaths)
		if current.ID != id || stringsJoin(current.Remove) != stringsJoin(plannedPaths) {
			return cleanup.ApplyResult{}, fmt.Errorf("cleanup preview for %s is stale or incomplete", resource)
		}
		// Pass the complete authoritative preview through the apply boundary.
		// ApplyPruneOldManaged rechecks every field under the lifecycle lock.
		if err := harnesslifecycle.ApplyPruneOldManaged(ctx, p.root, current); err != nil {
			return cleanup.ApplyResult{}, err
		}
		candidate := cleanup.ApplyResult{ProviderID: p.meta.ID, AppliedItems: append([]string(nil), paths[resource]...)}
		verification, err := p.Verify(ctx, cleanup.VerifyRequest{ApplyResult: candidate})
		if err != nil {
			return cleanup.ApplyResult{}, err
		}
		if !verification.Verified {
			return cleanup.ApplyResult{}, fmt.Errorf("cleanup apply for %s could not be verified: %s", resource, verification.Message)
		}
		result.AppliedItems = append(result.AppliedItems, candidate.AppliedItems...)
	}
	result.Applied = len(result.AppliedItems) > 0
	sort.Strings(result.AppliedItems)
	return result, nil
}

func stringsJoin(values []string) string { return strings.Join(values, "\x00") }

func (p *HarnessLifecycleProvider) Verify(ctx context.Context, req cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	if len(req.ApplyResult.SkippedItems) > 0 {
		return cleanup.VerifyResult{Verified: false, Message: "some managed harness artifacts were skipped"}, nil
	}
	for _, path := range req.ApplyResult.AppliedItems {
		if _, err := p.files.Stat(ctx, path); err == nil || !errors.Is(err, fs.ErrNotExist) {
			return cleanup.VerifyResult{Verified: false, Message: "a planned managed artifact remains"}, nil
		}
	}
	return cleanup.VerifyResult{Verified: true, Message: "all receipt-verified paths in the approved preview are absent"}, nil
}

func (p *HarnessLifecycleProvider) now(scope cleanup.ObservationScope) time.Time {
	if !scope.Now.IsZero() {
		return scope.Now
	}
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Now()
}

var _ cleanup.Provider = (*HarnessLifecycleProvider)(nil)
