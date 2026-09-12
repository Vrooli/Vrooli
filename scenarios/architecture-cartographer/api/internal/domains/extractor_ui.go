package domains

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
)

// defaultNonFeatureUIDirs are ui/src/features/ subdirectories that are not
// product domains (cross-cutting UI scaffolding present in every scenario).
var defaultNonFeatureUIDirs = map[string]struct{}{
	"health": {},
}

// UIFeatureExtractor derives advisory domain coverage from
// ui/src/features/<feature>/ folders. It is NEVER authoritative — UI
// features only feed cross-surface convergence reporting (a feature that
// maps to no declared domain, or a domain with no UI feature). The ladder
// skips advisory sources when selecting the authority rung.
type UIFeatureExtractor struct {
	surfaces SurfaceProvider
}

// NewUIFeatureExtractor returns the production UI-feature extractor.
func NewUIFeatureExtractor() *UIFeatureExtractor {
	return NewUIFeatureExtractorWithSurfaceProvider(nil)
}

func NewUIFeatureExtractorWithSurfaceProvider(surfaces SurfaceProvider) *UIFeatureExtractor {
	if surfaces == nil {
		surfaces = NewLocalSurfaceProvider()
	}
	return &UIFeatureExtractor{surfaces: surfaces}
}

var _ DomainSourceExtractor = (*UIFeatureExtractor)(nil)

// Source identifies this (advisory) rung.
func (*UIFeatureExtractor) Source() Source { return SourceUIFeatures }

// Extract scans the code-facts UI surface for src/features folders. A missing
// UI surface or features directory returns an empty advisory extraction.
func (e *UIFeatureExtractor) Extract(ctx context.Context, scenarioDir string) (Extraction, error) {
	provider := e.surfaces
	if provider == nil {
		provider = NewLocalSurfaceProvider()
	}
	inv, err := inspectSurfaceProvider(ctx, provider, scenarioDir)
	if err != nil {
		return Extraction{}, fmt.Errorf("inspect surfaces: %w", err)
	}
	ui, ok := surfaceByID(inv, "ui")
	if !ok {
		return Extraction{Source: SourceUIFeatures, Warnings: append([]ExtractionWarning(nil), inv.Warnings...)}, nil
	}
	root := filepath.Join(ui.Path, "src", "features")
	entries, err := readChildDirs(root)
	if err != nil {
		return Extraction{}, err
	}
	out := Extraction{Source: SourceUIFeatures, Warnings: append([]ExtractionWarning(nil), inv.Warnings...)}
	for _, ent := range entries {
		name := ent.Name()
		if _, skip := defaultNonFeatureUIDirs[name]; skip {
			continue
		}
		rel := surfaceRel(scenarioDir, filepath.Join(root, name))
		if rel == "" {
			continue
		}
		out.Domains = append(out.Domains, ExtractedDomain{
			Name:  name,
			Paths: []string{rel + "/"},
		})
	}
	sort.Slice(out.Domains, func(i, j int) bool { return out.Domains[i].Name < out.Domains[j].Name })
	return out, nil
}
