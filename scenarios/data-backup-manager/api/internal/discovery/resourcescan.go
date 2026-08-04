package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"data-backup-manager/internal/sources"

	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
)

// ResourceDataScanner derives suggestions from the canonical owner inventory.
// Despite the historical name, it intentionally scans every storage owner
// kind (scenarios, resources, tools, and safeguards). The inventory is
// read-only and retains declaration findings so malformed or incomplete
// manifests remain visible to the candidate consumer.
type ResourceDataScanner struct {
	repoRoot       string
	platform       storage.Platform
	platformSeams  storage.PlatformSeams
	maxScanEntries int
	loadInventory  func(storage.InventoryOptions) (storage.OwnerInventory, error)
}

// NewResourceDataScanner wires the repository and host platform authorities.
// A repository-root failure is deferred to Scan so API construction remains
// available for environments that do not run inside a checkout.
func NewResourceDataScanner() *ResourceDataScanner {
	root, _ := repocontract.FindRepoRootFromEnvOrCWD()
	return &ResourceDataScanner{
		repoRoot:       root,
		platform:       storage.NormalizePlatform(runtime.GOOS),
		maxScanEntries: defaultMaxScanEntries,
		loadInventory:  storage.LoadOwnerInventory,
	}
}

// NewResourceDataScannerWithRoot is the deterministic test seam. The loader
// remains injectable for exercising inventory failures without shelling out or
// changing process-wide environment state.
func NewResourceDataScannerWithRoot(root string, platform storage.Platform) *ResourceDataScanner {
	return &ResourceDataScanner{
		repoRoot:       root,
		platform:       storage.NormalizePlatform(string(platform)),
		maxScanEntries: defaultMaxScanEntries,
		loadInventory:  storage.LoadOwnerInventory,
	}
}

// WithPlatformSeams overrides OS directory lookups for deterministic foreign
// platform tests and controlled discovery environments.
func (s *ResourceDataScanner) WithPlatformSeams(seams storage.PlatformSeams) *ResourceDataScanner {
	s.platformSeams = seams
	return s
}

// WithInventoryLoader is a test seam for exercising inventory failures and
// synthetic owner sets without changing the production loader.
func (s *ResourceDataScanner) WithInventoryLoader(loader func(storage.InventoryOptions) (storage.OwnerInventory, error)) *ResourceDataScanner {
	s.loadInventory = loader
	return s
}

var _ TargetSourceScanner = (*ResourceDataScanner)(nil)

func (s *ResourceDataScanner) Scan(ctx context.Context) ([]TargetCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.repoRoot) == "" {
		return nil, fmt.Errorf("owner inventory: repository root is unavailable")
	}
	platform := s.platform
	if platform == "" {
		platform = storage.NormalizePlatform(runtime.GOOS)
	}
	load := s.loadInventory
	if load == nil {
		load = storage.LoadOwnerInventory
	}
	inventory, err := load(storage.InventoryOptions{
		RepoRoot:      s.repoRoot,
		Platform:      platform,
		PlatformSeams: s.platformSeams,
	})
	if err != nil {
		return nil, fmt.Errorf("load owner inventory: %w", err)
	}

	findingsByOwner := make(map[string][]storage.InventoryFinding)
	for _, finding := range inventory.Findings {
		key := ownerFindingKey(finding.OwnerKind, finding.OwnerID)
		findingsByOwner[key] = append(findingsByOwner[key], finding)
	}

	var out []TargetCandidate
	for _, owner := range inventory.Owners {
		if strings.TrimSpace(owner.ID) == "" {
			continue
		}
		entries := append([]storage.StorageEntry(nil), owner.StorageEntries...)
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if entry.Regenerable {
				continue
			}
			locator, resolveErr := storage.ResolveOwnerStoragePath(s.repoRoot, owner, entry, platform, s.platformSeams)
			if resolveErr != nil {
				if _, notApplicable := resolveErr.(*storage.NotApplicable); notApplicable {
					continue
				}
				// Inventory already records declaration-level failures. A path
				// that cannot be resolved at scan time is not a safe target.
				continue
			}
			candidate, ok := s.candidateFor(ctx, owner.ID, entry, locator)
			if !ok {
				continue
			}
			candidate.Findings = append(candidate.Findings, findingsByOwner[ownerFindingKey(owner.Kind, owner.ID)]...)
			out = append(out, candidate)
		}
	}
	return out, nil
}

func (s *ResourceDataScanner) candidateFor(ctx context.Context, owner string, entry storage.StorageEntry, locator string) (TargetCandidate, bool) {
	if strings.TrimSpace(locator) == "" || hasParentTraversal(filepath.ToSlash(locator)) {
		return TargetCandidate{}, false
	}
	info, err := os.Stat(locator)
	if err != nil {
		return TargetCandidate{}, false
	}
	var approx int64
	if strings.EqualFold(entry.Kind, "dir") {
		if !info.IsDir() {
			return TargetCandidate{}, false
		}
		entries, readErr := os.ReadDir(locator)
		if readErr != nil || len(entries) == 0 {
			return TargetCandidate{}, false
		}
		approx = boundedDirSize(ctx, locator, s.maxScanEntries)
	} else {
		if info.IsDir() || info.Size() == 0 {
			return TargetCandidate{}, false
		}
		approx = info.Size()
	}
	return TargetCandidate{
		Owner:       owner,
		Name:        entry.Name,
		SourceKind:  ownerSourceKind(entry),
		Locator:     locator,
		Rationale:   ownerRationale(owner, entry),
		ApproxBytes: approx,
		Sensitive:   entry.Sensitive,
	}, true
}

func ownerSourceKind(entry storage.StorageEntry) sources.SourceKind {
	if strings.EqualFold(entry.Format, "sqlite") {
		return sources.KindSQLite
	}
	return sources.KindFilesystem
}

func ownerRationale(owner string, entry storage.StorageEntry) string {
	if rationale := strings.TrimSpace(entry.Rationale); rationale != "" {
		return rationale
	}
	return fmt.Sprintf("Durable %s data declared by the %s owner.", entry.Name, owner)
}

func ownerFindingKey(kind storage.OwnerKind, id string) string {
	return string(kind) + "\x00" + id
}
