package domains

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// UIFeaturesDir is the scenario-relative root the UIFeatureExtractor scans
// for feature folders.
const UIFeaturesDir = "ui/src/features"

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
type UIFeatureExtractor struct{}

// NewUIFeatureExtractor returns the production UI-feature extractor.
func NewUIFeatureExtractor() *UIFeatureExtractor { return &UIFeatureExtractor{} }

var _ DomainSourceExtractor = (*UIFeatureExtractor)(nil)

// Source identifies this (advisory) rung.
func (*UIFeatureExtractor) Source() Source { return SourceUIFeatures }

// Extract scans ui/src/features for feature folders. A missing directory
// returns an empty extraction.
func (e *UIFeatureExtractor) Extract(_ context.Context, scenarioDir string) (Extraction, error) {
	root := filepath.Join(scenarioDir, UIFeaturesDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Extraction{Source: SourceUIFeatures}, nil
		}
		return Extraction{}, fmt.Errorf("read %s: %w", UIFeaturesDir, err)
	}
	out := Extraction{Source: SourceUIFeatures}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		if _, skip := defaultNonFeatureUIDirs[name]; skip {
			continue
		}
		out.Domains = append(out.Domains, ExtractedDomain{
			Name:  name,
			Paths: []string{UIFeaturesDir + "/" + name + "/"},
		})
	}
	sort.Slice(out.Domains, func(i, j int) bool { return out.Domains[i].Name < out.Domains[j].Name })
	return out, nil
}
