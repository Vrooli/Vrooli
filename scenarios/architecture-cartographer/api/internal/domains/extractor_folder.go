package domains

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
)

// defaultNonDomainFolders are api/internal/ subdirectories that are
// cross-cutting infrastructure, never product domains. The control surface
// (Phase 6) lets a scenario extend this set; the default mirrors the
// Non-Domains list every cartographer-shaped scenario shares.
var defaultNonDomainFolders = map[string]struct{}{
	"server":     {},
	"module":     {},
	"modules":    {},
	"database":   {},
	"testutil":   {},
	"middleware": {},
	"clock":      {},
	"git":        {},
	"httpc":      {},
	"httpx":      {},
	// Shared search substrate that recurs identically across every
	// search-provider scenario, owned by the scenario's `search` product
	// domain rather than being products themselves:
	//   - aisearch: the thin adapter over the shared packages/ai-go/search
	//     engine (embedding/vector-store/reconcile binding).
	//   - searchcontrol: the transport handler for the SHARED
	//     search-hub.v1.control.SearchControlService (reindex + config-write),
	//     identical wherever a provider exposes the control plane.
	"aisearch":      {},
	"searchcontrol": {},
}

// FolderExtractor derives de-facto domains from api/internal/<domain>/
// package folders. It is a lower ladder rung used for convergence against
// the authoritative source, and the zero-config authority for scenarios
// that ship no DOMAINS.md.
type FolderExtractor struct {
	// nonDomains is the set of folder names treated as infrastructure. When
	// nil the package default is used.
	nonDomains map[string]struct{}
	surfaces   SurfaceProvider
}

// NewFolderExtractor returns the production folder extractor using the
// default non-domain set.
func NewFolderExtractor() *FolderExtractor {
	return NewFolderExtractorWithSurfaceProvider(nil, nil)
}

// NewFolderExtractorWithExemptions returns a folder extractor that treats
// the given folder names as non-domains in addition to the defaults.
func NewFolderExtractorWithExemptions(extra []string) *FolderExtractor {
	return NewFolderExtractorWithSurfaceProvider(extra, nil)
}

// NewFolderExtractorWithSurfaceProvider returns a folder extractor that uses
// code-facts surface inventory when available. The local provider is used
// only as an explicit degraded fallback.
func NewFolderExtractorWithSurfaceProvider(extra []string, surfaces SurfaceProvider) *FolderExtractor {
	set := make(map[string]struct{}, len(defaultNonDomainFolders)+len(extra))
	for k := range defaultNonDomainFolders {
		set[k] = struct{}{}
	}
	for _, name := range extra {
		set[name] = struct{}{}
	}
	if surfaces == nil {
		surfaces = NewLocalSurfaceProvider()
	}
	return &FolderExtractor{nonDomains: set, surfaces: surfaces}
}

// Source identifies this rung.
func (*FolderExtractor) Source() Source { return SourceAPIFolders }

// Extract scans code-facts' API surface for domain package roots and unions
// their names. A scenario with no API surface returns an empty extraction.
// Folders in the non-domain set (infrastructure) are skipped.
func (e *FolderExtractor) Extract(ctx context.Context, scenarioDir string) (Extraction, error) {
	exempt := e.nonDomains
	if exempt == nil {
		exempt = defaultNonDomainFolders
	}
	provider := e.surfaces
	if provider == nil {
		provider = NewLocalSurfaceProvider()
	}
	inv, err := provider.Inspect(ctx, scenarioDir)
	if err != nil {
		return Extraction{}, fmt.Errorf("inspect surfaces: %w", err)
	}

	names := map[string]struct{}{}
	for _, root := range apiDomainRoots(scenarioDir, inv) {
		entries, err := readChildDirs(root)
		if err != nil {
			return Extraction{}, err
		}
		for _, ent := range entries {
			name := ent.Name()
			if _, skip := exempt[name]; skip {
				continue
			}
			names[name] = struct{}{}
		}
	}

	out := Extraction{Source: SourceAPIFolders, Warnings: append([]ExtractionWarning(nil), inv.Warnings...)}
	for name := range names {
		out.Domains = append(out.Domains, ExtractedDomain{
			Name:  name,
			Paths: folderPaths(scenarioDir, apiDomainRoots(scenarioDir, inv), name),
		})
	}
	sort.Slice(out.Domains, func(i, j int) bool { return out.Domains[i].Name < out.Domains[j].Name })
	return out, nil
}

func apiDomainRoots(scenarioDir string, inv SurfaceInventory) []string {
	api, ok := surfaceByID(inv, "api")
	if !ok {
		return nil
	}
	roots := []string{}
	for _, rel := range []string{"internal", "handlers"} {
		root := filepath.Join(api.Path, rel)
		if entries, err := readChildDirs(root); err == nil && len(entries) > 0 {
			roots = append(roots, root)
		}
	}
	return roots
}

// folderPaths returns the repo-relative path prefixes a folder-derived
// domain owns: whichever of its api/internal/ and api/handlers/ packages
// exist on disk.
func folderPaths(scenarioDir string, roots []string, name string) []string {
	var paths []string
	for _, root := range roots {
		full := filepath.Join(root, name)
		if dirExists(full) {
			rel := surfaceRel(scenarioDir, full)
			if rel != "" {
				paths = append(paths, rel+"/")
			}
		}
	}
	return paths
}
