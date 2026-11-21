package preferences

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Manifest is the optional per-scenario `.vrooli/notifications.json` that lets a
// scenario declare its notification branding and categories without any runtime
// configuration. Every field except the category key is optional; the loader is
// lenient (unknown fields are ignored and category entries with an empty key are
// skipped).
type Manifest struct {
	DisplayName  string             `json:"displayName"`
	Icon         string             `json:"icon"`
	DeepLinkBase string             `json:"deepLinkBase"`
	Categories   []ManifestCategory `json:"categories"`
	// AppDelivery opts the scenario's own installed app into showing its
	// notifications instead of Chimesail. The scenario web server
	// (@vrooli/api-base/server) reads it; the hub decides per device.
	AppDelivery bool `json:"appDelivery"`
}

// ManifestCategory is one declared notification class.
type ManifestCategory struct {
	Key               string `json:"key"`
	DisplayName       string `json:"displayName"`
	Description       string `json:"description"`
	DefaultEnabled    bool   `json:"defaultEnabled"`
	DefaultMinUrgency string `json:"defaultMinUrgency"`
}

// LoadManifest reads and leniently parses a `.vrooli/notifications.json` file.
// Categories with an empty key are dropped. A read or JSON error is returned so
// the caller can log-and-skip; a missing file is signaled with os.ErrNotExist.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	m.DisplayName = strings.TrimSpace(m.DisplayName)
	m.Icon = strings.TrimSpace(m.Icon)
	m.DeepLinkBase = strings.TrimSpace(m.DeepLinkBase)
	kept := m.Categories[:0]
	for _, c := range m.Categories {
		c.Key = strings.TrimSpace(c.Key)
		if c.Key == "" {
			continue
		}
		c.DisplayName = strings.TrimSpace(c.DisplayName)
		c.Description = strings.TrimSpace(c.Description)
		c.DefaultMinUrgency = strings.TrimSpace(c.DefaultMinUrgency)
		kept = append(kept, c)
	}
	m.Categories = kept
	return m, nil
}

// serviceDisplayName reads `service.displayName` from a scenario's
// `.vrooli/service.json`. An absent file or field yields ("", false) without an
// error so inference can fall back to the slug.
func serviceDisplayName(serviceJSONPath string) (string, bool) {
	data, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return "", false
	}
	var doc struct {
		Service struct {
			DisplayName string `json:"displayName"`
		} `json:"service"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", false
	}
	name := strings.TrimSpace(doc.Service.DisplayName)
	return name, name != ""
}

// ResolveRepoRoot resolves the active repository root using the canonical
// VROOLI_ROOT/VROOLI_SOURCE_ROOT environment variables, then the current working
// directory, then the executable path. It is a thin, test-free pass-through so
// callers do not import the repo-contract package directly.
func ResolveRepoRoot() (string, error) {
	return repocontract.FindRepoRootFromEnvOrCWD()
}

// scenarioSlugs returns the set of installed scenario slugs (contract targets of
// kind scenario) plus the repo-relative root of each. A failure to resolve the
// repo root or load/enumerate the contract returns (nil, nil, err); callers
// treat that as "inference unavailable" and fall back to the observed path.
func scenarioSlugs(repoRoot string) (roots map[string]string, err error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return nil, errors.New("repo root is required")
	}
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil, err
	}
	targets, err := contract.EnumerateTargets(repoRoot)
	if err != nil {
		return nil, err
	}
	roots = make(map[string]string)
	for _, t := range targets {
		if t.Kind != repocontract.TargetKindScenario {
			continue
		}
		roots[t.ID] = t.Root
	}
	return roots, nil
}

// IngestDeclaredManifests scans every installed scenario for an optional
// `.vrooli/notifications.json` and upserts the declared source and its
// categories. Only scenarios that carry a manifest are ingested; this never
// enumerates sources for scenarios that neither declare nor are observed.
//
// It is best-effort: a per-scenario failure is logged and skipped, and a missing
// repo root or a contract/enumerate failure is logged once and returns nil so
// branding falls back to the observed path. It never aborts startup.
func (s *Storage) IngestDeclaredManifests(ctx context.Context, repoRoot, now string) error {
	roots, err := scenarioSlugs(repoRoot)
	if err != nil {
		slog.Warn("notification catalog: declared-manifest scan skipped; relying on observed branding", "error", err)
		return nil
	}
	for slug, rel := range roots {
		manifestPath := filepath.Join(repoRoot, filepath.FromSlash(rel), ".vrooli", "notifications.json")
		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				slog.Warn("notification catalog: skip invalid manifest", "scenario", slug, "path", manifestPath, "error", err)
			}
			continue
		}
		displayName := manifest.DisplayName
		if displayName == "" {
			if svcName, ok := serviceDisplayName(filepath.Join(repoRoot, filepath.FromSlash(rel), ".vrooli", "service.json")); ok {
				displayName = svcName
			}
		}
		if displayName == "" {
			displayName = slug
		}
		icon := manifest.Icon
		if icon == "" {
			icon = slug
		}
		if err := s.UpsertDeclaredSource(ctx, slug, displayName, icon, manifest.DeepLinkBase, now); err != nil {
			slog.Warn("notification catalog: upsert declared source failed", "scenario", slug, "error", err)
			continue
		}
		for _, c := range manifest.Categories {
			if err := s.UpsertDeclaredCategory(ctx, slug, c.Key, c.DisplayName, c.Description, c.DefaultEnabled, c.DefaultMinUrgency, now); err != nil {
				slog.Warn("notification catalog: upsert declared category failed", "scenario", slug, "category", c.Key, "error", err)
			}
		}
	}
	return nil
}

// InferObservedBranding resolves display branding for an observed source from
// its `.vrooli/service.json` and stores it via UpsertObservedBranding. If the
// slug does not resolve to a real installed scenario (no service.json), branding
// is left empty so ListCatalog filters the stray source out. It is best-effort:
// an unresolved repo root or contract failure is swallowed.
func (s *Storage) InferObservedBranding(ctx context.Context, repoRoot, source, now string) {
	source = strings.TrimSpace(source)
	if source == "" {
		return
	}
	roots, err := scenarioSlugs(repoRoot)
	if err != nil {
		return
	}
	rel, ok := roots[source]
	if !ok {
		return
	}
	displayName, ok := serviceDisplayName(filepath.Join(repoRoot, filepath.FromSlash(rel), ".vrooli", "service.json"))
	if !ok {
		displayName = source
	}
	if err := s.UpsertObservedBranding(ctx, source, displayName, source, now); err != nil {
		slog.Warn("notification catalog: infer observed branding failed", "scenario", source, "error", err)
	}
}

// installedScenarioSet returns the set of installed scenario slugs for catalog
// filtering. A failure returns (nil, false) so ListCatalog can decline to filter
// rather than hide real sources when the contract cannot be read.
func installedScenarioSet(repoRoot string) (map[string]struct{}, bool) {
	roots, err := scenarioSlugs(repoRoot)
	if err != nil {
		return nil, false
	}
	set := make(map[string]struct{}, len(roots))
	for slug := range roots {
		set[slug] = struct{}{}
	}
	return set, true
}
