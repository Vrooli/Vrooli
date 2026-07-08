package providers

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cleanup-manager/internal/cleanup"
)

type FileProviderConfig struct {
	ID          string
	Name        string
	Roots       []string
	Description string
}

type FileProvider struct {
	meta        cleanup.ProviderMetadata
	files       cleanup.FileSystem
	clock       cleanup.Clock
	roots       []string
	description string
	action      string
}

func NewTrashProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig) *FileProvider {
	return newFileProvider(files, clock, cfg, cleanup.SafetyTierSafeWithOwner, cleanup.ProviderModeDisabled, cleanup.ApprovalModeOwner, "trash-remove")
}

func NewTmpProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig) *FileProvider {
	return newFileProvider(files, clock, cfg, cleanup.SafetyTierSafe, cleanup.ProviderModeDisabled, cleanup.ApprovalModeOperator, "tmp-remove")
}

func NewCacheProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig) *FileProvider {
	return newFileProvider(files, clock, cfg, cleanup.SafetyTierConditional, cleanup.ProviderModeDisabled, cleanup.ApprovalModeOperator, "cache-remove")
}

func newFileProvider(files cleanup.FileSystem, clock cleanup.Clock, cfg FileProviderConfig, tier cleanup.SafetyTier, mode cleanup.ProviderMode, approval cleanup.ApprovalMode, action string) *FileProvider {
	return &FileProvider{
		meta: cleanup.ProviderMetadata{
			ID:                  cfg.ID,
			Name:                cfg.Name,
			Version:             "v1",
			OwnerScenario:       "cleanup-manager",
			SafetyTier:          tier,
			DefaultMode:         mode,
			DefaultApproval:     approval,
			SupportedPlatforms:  []string{"linux", "darwin"},
			IrreversibleEffects: []string{"filesystem entries are removed from configured cleanup roots"},
			TestSubstitute:      "fake-filesystem",
		},
		files:       files,
		clock:       clock,
		roots:       cleanRoots(cfg.Roots),
		description: cfg.Description,
		action:      action,
	}
}

func (p *FileProvider) Metadata() cleanup.ProviderMetadata { return p.meta }

func (p *FileProvider) Estimate(ctx context.Context, req cleanup.EstimateRequest) (cleanup.Estimate, error) {
	preview, err := p.preview(ctx, req.Scope, req.Policy)
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

func (p *FileProvider) Preview(ctx context.Context, req cleanup.PreviewRequest) (cleanup.Preview, error) {
	return p.preview(ctx, req.Scope, req.Policy)
}

func (p *FileProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	if req.ProviderVersion != p.meta.Version {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s version mismatch: got %q want %q", p.meta.ID, req.ProviderVersion, p.meta.Version)
	}
	if req.IdempotencyKey == "" {
		return cleanup.ApplyResult{}, fmt.Errorf("provider %s apply requires idempotency key", p.meta.ID)
	}
	if req.ApprovalMode == cleanup.ApprovalModeDisabled {
		return cleanup.ApplyResult{ProviderID: p.meta.ID, Applied: false, SkippedItems: previewItemIDs(req.Preview.Items), Warnings: []string{"provider disabled by policy"}}, nil
	}
	var reclaimed int64
	var skipped []string
	for _, item := range req.Preview.Items {
		if !p.withinConfiguredRoot(item.Path) {
			skipped = append(skipped, item.ID)
			continue
		}
		if err := p.files.RemoveAll(ctx, item.Path); err != nil {
			return cleanup.ApplyResult{}, err
		}
		reclaimed += item.Bytes
	}
	return cleanup.ApplyResult{ProviderID: p.meta.ID, Applied: reclaimed > 0, ReclaimedBytes: reclaimed, SkippedItems: skipped}, nil
}

func (p *FileProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true, Message: "filesystem provider applied through FileSystem seam"}, nil
}

func (p *FileProvider) preview(ctx context.Context, scope cleanup.ObservationScope, policy cleanup.ProviderPolicy) (cleanup.Preview, error) {
	out := cleanup.Preview{ProviderID: p.meta.ID, ProviderVersion: p.meta.Version}
	if !policy.Enabled {
		out.BlockedReason = "provider disabled by policy"
		return out, nil
	}
	if p.files == nil {
		out.BlockedReason = "filesystem seam unavailable"
		return out, nil
	}

	roots := p.roots
	if len(scope.RootPaths) > 0 {
		roots = intersectRoots(p.roots, cleanRoots(scope.RootPaths))
	}
	now := p.now(scope)
	for _, root := range roots {
		err := p.files.Walk(ctx, root, func(info cleanup.FileInfo) error {
			if info.Path == root || info.IsDir || !p.withinConfiguredRoot(info.Path) || activePath(info.Path) {
				return nil
			}
			if policy.MinAge > 0 && now.Sub(info.ModTime) < policy.MinAge {
				return nil
			}
			if policy.MaxBytes > 0 && sumPreviewBytes(out.Items)+info.Size > policy.MaxBytes {
				out.Warnings = append(out.Warnings, "max reclaim limit reached")
				return nil
			}
			out.Items = append(out.Items, cleanup.PreviewItem{
				ID:          stableItemID(p.meta.ID, info.Path),
				Path:        info.Path,
				Description: p.description,
				Bytes:       info.Size,
				Action:      p.action,
				SafetyTier:  p.meta.SafetyTier,
			})
			return nil
		})
		if err != nil {
			return cleanup.Preview{}, err
		}
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].Path < out.Items[j].Path })
	return out, nil
}

func (p *FileProvider) now(scope cleanup.ObservationScope) time.Time {
	if !scope.Now.IsZero() {
		return scope.Now
	}
	if p.clock != nil {
		return p.clock.Now()
	}
	return time.Time{}
}

func (p *FileProvider) withinConfiguredRoot(path string) bool {
	cleanPath := filepath.Clean(path)
	for _, root := range p.roots {
		if cleanPath == root || strings.HasPrefix(cleanPath, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func cleanRoots(roots []string) []string {
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		if strings.TrimSpace(root) != "" {
			out = append(out, filepath.Clean(root))
		}
	}
	sort.Strings(out)
	return out
}

func intersectRoots(configured []string, scoped []string) []string {
	var out []string
	for _, root := range configured {
		for _, scope := range scoped {
			if root == scope || strings.HasPrefix(root, scope+string(filepath.Separator)) || strings.HasPrefix(scope, root+string(filepath.Separator)) {
				out = append(out, root)
				break
			}
		}
	}
	return out
}

func activePath(path string) bool {
	name := filepath.Base(path)
	return strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, ".sock") || strings.Contains(path, string(filepath.Separator)+"proc"+string(filepath.Separator))
}

func stableItemID(providerID, path string) string {
	id := strings.NewReplacer(string(filepath.Separator), "-", " ", "-", ".", "-").Replace(strings.Trim(filepath.Clean(path), string(filepath.Separator)))
	return providerID + ":" + id
}

func sumPreviewBytes(items []cleanup.PreviewItem) int64 {
	var total int64
	for _, item := range items {
		total += item.Bytes
	}
	return total
}

func previewItemIDs(items []cleanup.PreviewItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}
