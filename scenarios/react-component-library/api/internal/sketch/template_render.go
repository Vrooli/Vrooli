package sketch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"react-component-library/internal/components"
)

// ConfigureTemplateRender consumes exact published story metadata. It neither
// guesses prop paths from region names nor copies a preview-only JSX renderer.
func ConfigureTemplateRender(ctx context.Context, doc Document, reader components.DependencyReader) (Document, []string, error) {
	if doc.Template == nil || reader == nil {
		return doc, nil, fmt.Errorf("published template reader is required")
	}
	asset, contract, err := readPublishedStory(ctx, *doc.Template, reader)
	if err != nil {
		return doc, nil, err
	}
	props, err := components.ResolveStoryArgs(contract, "default")
	if err != nil {
		return doc, nil, err
	}
	settings := &RenderSettings{TemplateExport: asset.Slug, Bindings: map[string]any{"$template": props}}
	known := map[string]bool{}
	for _, region := range doc.Regions {
		known[region.ID] = false
	}
	for _, placement := range doc.Placements {
		known[placement.Region] = false
	}
	for _, field := range contract.Args.Fields {
		if field.Region == "" {
			continue
		}
		if _, exists := known[field.Region]; !exists {
			continue
		}
		settings.Regions = append(settings.Regions, RenderRegion{ID: field.Region, TemplateRegion: field.Region, Slot: strings.Split(field.Path, ".")})
		known[field.Region] = true
	}
	if len(settings.Regions) == 0 {
		return doc, nil, fmt.Errorf("published template declares no matching semantic region ports")
	}
	var obligations []string
	for id, mapped := range known {
		if !mapped {
			obligations = append(obligations, "Map authored region "+id+" to a compatible template port.")
		}
	}
	sort.Strings(obligations)
	// Retained region selections still need their own explicit named exports and
	// deterministic bindings; never reuse props from an unrelated selected asset.
	for _, p := range doc.Placements {
		if p.Fills.Asset != "" {
			obligations = append(obligations, "Configure exact export and fixture bindings for "+p.Region+" ("+p.Fills.Asset+").")
		}
	}
	doc.Render = settings
	return doc, obligations, ctx.Err()
}

func readPublishedStory(ctx context.Context, ref AssetRef, reader components.DependencyReader) (components.Component, *components.StoryContract, error) {
	asset, err := reader.Get(ctx, ref.Asset)
	if err != nil {
		return components.Component{}, nil, err
	}
	if asset.CatalogID != ref.Asset {
		return components.Component{}, nil, fmt.Errorf("template catalog identity mismatch")
	}
	version, err := reader.GetVersion(ctx, asset.ID, ref.Version)
	if err != nil {
		return components.Component{}, nil, err
	}
	if version.Version != ref.Version || version.ComponentID != asset.ID || !version.DependencyLockPresent || version.Status != components.VersionStatusReleased {
		return components.Component{}, nil, fmt.Errorf("template lacks exact released version evidence")
	}
	var storyBytes []byte
	for _, file := range version.Files {
		if file.Path == "story.json" {
			digest := sha256.Sum256([]byte(file.Content))
			if file.ContentSHA256 == "" || hex.EncodeToString(digest[:]) != file.ContentSHA256 {
				return components.Component{}, nil, fmt.Errorf("published story digest mismatch")
			}
			if storyBytes != nil {
				return components.Component{}, nil, fmt.Errorf("ambiguous published story")
			}
			storyBytes = []byte(file.Content)
		}
	}
	if storyBytes == nil {
		return components.Component{}, nil, fmt.Errorf("published template has no story contract")
	}
	contract, diagnostics := components.ParseStoryContract(storyBytes)
	if len(components.StoryContractErrors(diagnostics)) > 0 {
		return components.Component{}, nil, fmt.Errorf("published template story contract is invalid")
	}
	return asset, contract, nil
}
