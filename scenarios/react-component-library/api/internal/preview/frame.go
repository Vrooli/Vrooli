package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/deps"
)

type frameCatalogDocument struct {
	Kind  string `json:"kind"`
	Asset struct {
		ID      string   `json:"id"`
		Kind    string   `json:"kind"`
		Targets []string `json:"targets"`
	} `json:"asset"`
	Regions []struct {
		ID      string `json:"id"`
		Accepts string `json:"accepts"`
	} `json:"regions"`
	Expects []struct {
		Capability    string   `json:"capability"`
		TypeArguments []string `json:"typeArguments"`
	} `json:"expects"`
	Fixture struct {
		DataShapes  []string       `json:"dataShapes"`
		RecordCount map[string]int `json:"recordCount"`
		Satisfies   *struct {
			Capability    string   `json:"capability"`
			TypeArguments []string `json:"typeArguments"`
		} `json:"satisfies"`
	} `json:"fixture"`
}

func (s *service) bundleFrame(ctx context.Context, frame *components.StoryFrame) (string, string, string, []deps.Declaration, error) {
	if frame == nil {
		return "", "", "", nil, nil
	}
	frameDoc, err := s.readCatalogFrame(frame.Asset)
	if err != nil {
		return "", "", "", nil, err
	}
	if frameDoc.Kind != "catalog-asset" || frameDoc.Asset.ID != frame.Asset {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame asset is not a catalog asset")
	}
	if !contains(frameDoc.Asset.Targets, "react-vite") {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame asset does not target react-vite")
	}
	regionDeclared := false
	for _, region := range frameDoc.Regions {
		if region.ID == frame.Region {
			regionDeclared = true
			break
		}
	}
	if !regionDeclared {
		return "", "", "", nil, frameBundleError(frame.Asset, fmt.Sprintf("frame region %q is not declared", frame.Region))
	}
	for _, region := range frameDoc.Regions {
		if region.ID == frame.Region && region.Accepts != "" && region.Accepts != frame.Capability {
			return "", "", "", nil, frameBundleError(frame.Asset, fmt.Sprintf("frame region %q requires subject capability %q", frame.Region, region.Accepts))
		}
	}
	var fixtureDoc frameCatalogDocument
	if strings.TrimSpace(frame.Fixture) != "" {
		fixtureDoc, err = s.readCatalogFrame(frame.Fixture)
		if err != nil {
			return "", "", "", nil, err
		}
		if fixtureDoc.Asset.Kind != "fixture" {
			return "", "", "", nil, frameBundleError(frame.Fixture, "frame fixture must be a fixture asset")
		}
	} else if frameRequiresFixture(frameDoc.Expects) {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame requires a fixture")
	}
	for _, expect := range frameDoc.Expects {
		if expect.Capability != "data-source" {
			continue
		}
		if fixtureDoc.Fixture.Satisfies == nil || fixtureDoc.Fixture.Satisfies.Capability != "data-source" || !compatibleFrameTypes(expect.TypeArguments, fixtureDoc.Fixture.Satisfies.TypeArguments) {
			return "", "", "", nil, frameBundleError(frame.Fixture, "frame fixture does not satisfy the frame asset's data-source port")
		}
	}
	componentsList, err := s.components.List(ctx, components.SearchQuery{Limit: 10000})
	if err != nil {
		return "", "", "", nil, err
	}
	var implementations []components.Component
	for index := range componentsList {
		candidate, getErr := s.components.Get(ctx, componentsList[index].ID)
		if getErr == nil && candidate.CatalogID == frame.Asset {
			implementations = append(implementations, candidate)
		}
	}
	if len(implementations) == 0 {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame asset has no react-vite implementation")
	}
	if len(implementations) > 1 {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame asset has ambiguous react-vite implementations")
	}
	implementation := &implementations[0]
	version := strings.TrimSpace(frame.Version)
	if version == "" {
		// Schema-v3 stories predate immutable frame references. Keep them
		// renderable during migration, but make the fallback explicit and
		// deterministic for the current catalog projection.
		version = implementation.LatestVersion
	}
	if len(implementation.Dependencies) > 0 {
		if _, err := components.ResolveDependencyClosure(ctx, s.components, implementation.ID, version); err != nil {
			return "", "", "", nil, err
		}
	}
	content, err := s.components.GetVersionContent(ctx, implementation.ID, version)
	if err != nil {
		return "", "", "", nil, err
	}
	js, _, err := s.bundler.BuildBundle(ctx, stampPreviewSource(content.Body, content.SourcePath, frame.Asset, version), content.SourcePath)
	if err != nil {
		return "", "", "", nil, err
	}
	var declarations []deps.Declaration
	if s.deps != nil {
		var depErr error
		declarations, depErr = s.deps.ListForComponentVersion(ctx, implementation.ID, version)
		if depErr != nil {
			return "", "", "", nil, depErr
		}
	}
	fixtureJSON := []byte(nil)
	if strings.TrimSpace(frame.Fixture) != "" {
		fixtureJSON, err = json.Marshal(struct {
			Asset       string   `json:"asset"`
			DataShapes  []string `json:"dataShapes,omitempty"`
			RecordCount any      `json:"recordCount,omitempty"`
		}{Asset: fixtureDoc.Asset.ID, DataShapes: fixtureDoc.Fixture.DataShapes, RecordCount: fixtureDoc.Fixture.RecordCount})
		if err != nil {
			return "", "", "", nil, err
		}
	}
	return js, string(fixtureJSON), content.SourcePath, declarations, nil
}

func frameRequiresFixture(expects []struct {
	Capability    string   `json:"capability"`
	TypeArguments []string `json:"typeArguments"`
}) bool {
	for _, expect := range expects {
		if expect.Capability == "data-source" {
			return true
		}
	}
	return false
}

func (s *service) readCatalogFrame(id string) (frameCatalogDocument, error) {
	if strings.TrimSpace(s.repoRoot) == "" {
		return frameCatalogDocument{}, frameBundleError(id, "preview repository root is not configured")
	}
	parts := strings.Split(strings.TrimSpace(id), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(id, `/\\`) {
		return frameCatalogDocument{}, frameBundleError(id, "invalid catalog asset id")
	}
	path := filepath.Join(s.repoRoot, "scenarios", "react-component-library", "catalog", "assets", parts[0], parts[1]+".json")
	if _, err := os.Stat(path); err != nil {
		return frameCatalogDocument{}, frameBundleError(id, "catalog asset was not found")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return frameCatalogDocument{}, fmt.Errorf("read catalog asset %q: %w", id, err)
	}
	var doc frameCatalogDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return frameCatalogDocument{}, fmt.Errorf("parse catalog asset %q: %w", id, err)
	}
	return doc, nil
}

func frameBundleError(sourcePath, detail string) error {
	return ErrBundle{SourcePath: sourcePath, Messages: []string{detail}}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func compatibleFrameTypes(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	for index := range expected {
		if strings.HasPrefix(expected[index], "T") {
			continue
		}
		if expected[index] != actual[index] {
			return false
		}
	}
	return true
}
