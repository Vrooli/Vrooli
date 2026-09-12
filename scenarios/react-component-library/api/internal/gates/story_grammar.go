package gates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/librarywalk"

	"react-component-library/internal/components"
)

func ValidateStoryGrammar(scope Scope) (Result, error) {
	root := scope.Root
	// The catalog map is needed to classify every implementation path as
	// renderable before the caller's asset scope is applied. Loading it through
	// the selected library id would omit the corresponding catalog id
	// (for example react-component-library:Badge vs primitives.badge).
	catalog, err := loadAssets(Scope{Root: scope.Root})
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
	paths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(paths)}
	result.Inspected = 0
	for _, path := range paths {
		assetID := implementationName(path)
		if !scopeReportsAsset(scope, assetID) {
			continue
		}
		retired, err := isRetiredVersion(path)
		if err != nil {
			return Result{}, err
		}
		if retired {
			continue
		}
		result.Inspected++
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
			if storyNeedsExpectation(story) {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.story_tautological_expectation", AssetID: assetID, File: repoRel(root, path),
					Message:     fmt.Sprintf("story %q declares no expectation", story.ID),
					Remediation: fmt.Sprintf("Asset %s must assert something the component itself renders; add a role, text, attribute, count, or layout expectation.", assetID),
					DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
				})
			}
			for expectationIndex, expectation := range story.Expect {
				selector := strings.TrimSpace(expectation.Selector)
				if (expectation.Kind == "visible" || expectation.Kind == "exists") && (selector == "body" || selector == "html" || selector == ":root") {
					result.Findings = append(result.Findings, Finding{
						Code: "catalog.story_tautological_expectation", AssetID: assetID, File: repoRel(root, path),
						Message:     fmt.Sprintf("story %q expectation %d selects %q, which is visible in every document", story.ID, expectationIndex, selector),
						Remediation: fmt.Sprintf("Asset %s must assert something the component itself renders; replace %q with a component-owned selector, role, or text.", assetID, selector),
						DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
					})
				}
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
		for _, boundary := range contract.Stories {
			if boundary.Role != "boundary" {
				continue
			}
			var boundaryArgs map[string]json.RawMessage
			if json.Unmarshal(boundary.Args, &boundaryArgs) != nil {
				continue
			}
			// A boundary may use an axis value as part of a meaningful
			// condition, such as long content with a warning tone. It is
			// redundant only when its complete argument set consists of axis
			// values; the coverage gate owns that same axis-only rule for the
			// latest version.
			if !storyGrammarBoundaryArgsAreAxisOnly(contract, boundaryArgs) || storyGrammarBoundaryHasExplicitState(boundary) {
				continue
			}
			for _, axis := range contract.Stories {
				if axis.Role != "axis" {
					continue
				}
				for fieldPath, values := range axis.Covers {
					value, ok := boundaryArgs[fieldPath]
					if !ok || !containsRawValue(values, value) {
						continue
					}
					result.Findings = append(result.Findings, Finding{
						Code: "catalog.story_boundary_redundant", AssetID: assetID, File: repoRel(root, path),
						Message:     fmt.Sprintf("boundary story %q repeats enum value for axis %q", boundary.ID, fieldPath),
						Remediation: fmt.Sprintf("Move the %s value into the axis story's covers matrix, or make boundary story %q render a non-enum condition.", fieldPath, boundary.ID),
						DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
					})
				}
			}
		}
	}
	return nonEmpty(result, "story-grammar"), nil
}

func isEnumField(contract *components.StoryContract, path string) bool {
	for _, field := range contract.Args.Fields {
		if field.Path == path {
			return field.Kind == components.StoryFieldEnum
		}
	}
	return false
}

func storyGrammarBoundaryArgsAreAxisOnly(contract *components.StoryContract, args map[string]json.RawMessage) bool {
	if len(args) == 0 {
		return false
	}
	for path := range args {
		if !isEnumField(contract, path) {
			return false
		}
	}
	return true
}

func storyGrammarBoundaryHasExplicitState(story components.StoryDefinition) bool {
	return len(story.States) > 0 || len(story.Interactions) > 0
}

func storyNeedsExpectation(story components.StoryDefinition) bool {
	return len(story.Expect) == 0 && !story.RendersNothing
}

func containsRawValue(values []json.RawMessage, target json.RawMessage) bool {
	for _, value := range values {
		if bytes.Equal(bytes.TrimSpace(value), bytes.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func scopeReportsAsset(scope Scope, assetID string) bool {
	if len(scope.Assets) == 0 {
		return true
	}
	candidate := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(assetID), "react-component-library:"))
	candidateShort := candidate
	if index := strings.LastIndex(candidateShort, "."); index >= 0 {
		candidateShort = candidateShort[index+1:]
	}
	for _, allowed := range scope.Assets {
		allowed = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(allowed), "react-component-library:"))
		allowedShort := allowed
		if index := strings.LastIndex(allowedShort, "."); index >= 0 {
			allowedShort = allowedShort[index+1:]
		}
		if allowed == candidate || allowedShort == candidate || allowed == candidateShort || allowedShort == candidateShort {
			return true
		}
	}
	return false
}

// ValidateStoryDistinctness rejects exact duplicate frames and the old
// one-specimen-per-option shape. Axis stories are intentionally allowed to
// share a specimen because their declared covers matrix is the variation.
