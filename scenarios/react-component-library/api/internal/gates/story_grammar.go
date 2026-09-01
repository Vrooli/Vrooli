package gates

import (
	"fmt"
	"os"
	"path/filepath"

	"react-component-library/internal/components"
)

func ValidateStoryGrammar(scope Scope) (Result, error) {
	root := scope.Root
	catalog, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	renderable := map[string]bool{}
	for _, asset := range catalog {
		switch asset.Asset.Kind {
		case "component", "primitive", "pattern", "page-template", "navigation":
			renderable[asset.Asset.ID] = true
		}
	}
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(paths)}
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return Result{}, readErr
		}
		_, diagnostics := components.ParseStoryContract(raw)
		contract, _ := components.ParseStoryContract(raw)
		for _, diagnostic := range components.StoryContractErrors(diagnostics) {
			if diagnostic.Rule != "raw_node_tag_name" {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.raw_node_tag_name", AssetID: implementationName(path), File: repoRel(root, path),
				Message: diagnostic.Detail, Remediation: "Replace the tag-name $text value with a $node and meaningful children, or move the React composition into story.tsx.", DocsRef: "docs/guides/asset-preview-composition.md",
			})
		}
		if contract == nil {
			continue
		}
		assetID := implementationName(path)
		if !renderable[assetID] {
			continue
		}
		anatomyCount := 0
		axes := map[string]bool{}
		for _, story := range contract.Stories {
			if story.Role == "anatomy" {
				anatomyCount++
			}
			if story.Role == "axis" {
				axes[story.Axis] = true
			}
			if len(story.Expect) == 0 {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.story_expectation_missing", AssetID: assetID, File: repoRel(root, path),
					Message:     fmt.Sprintf("story %q declares no expectation", story.ID),
					Remediation: "Add at least one rendered expectation to the story, such as a role, text, attribute, or layout assertion.",
					DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
				})
			}
		}
		if anatomyCount != 1 {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.story_anatomy_missing", AssetID: assetID, File: repoRel(root, path),
				Message:     fmt.Sprintf("contract has %d anatomy frames; exactly one is required", anatomyCount),
				Remediation: "Mark exactly one default rendered story with role=anatomy and keep its specimen free of matrix or boundary-only variation.",
				DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
			})
		}
		for _, field := range contract.Args.Fields {
			if field.Kind == components.StoryFieldEnum && !axes[field.Path] {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.story_axis_missing", AssetID: assetID, File: repoRel(root, path),
					Message:     fmt.Sprintf("enum axis %q has no axis frame", field.Path),
					Remediation: fmt.Sprintf("Add one story with role=axis, axis=%q, and covers listing the rendered options.", field.Path),
					DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
				})
			}
		}
	}
	return nonEmpty(result, "story-grammar"), nil
}

// ValidateStoryDistinctness rejects exact duplicate frames and the old
// one-specimen-per-option shape. Axis stories are intentionally allowed to
// share a specimen because their declared covers matrix is the variation.
