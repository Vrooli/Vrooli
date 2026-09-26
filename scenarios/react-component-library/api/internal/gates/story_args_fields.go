package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/librarywalk"
)

// ValidateStoryArgsFields makes hand-authored args.fields visible while the
// index projection serves the source-derived fields to consumers.
func ValidateStoryArgsFields(scope Scope) (Result, error) {
	paths, err := librarywalk.Glob(filepath.Join(scope.Root, "scenarios", "react-component-library", "library", "*", "*", "component.json"))
	if err != nil {
		return Result{}, err
	}
	sort.Strings(paths)
	result := Result{}
	for _, manifestPath := range paths {
		if strings.Contains(filepath.ToSlash(manifestPath), "/library/.retired/") {
			continue
		}
		assetID := implementationName(manifestPath)
		if !scopeReportsAsset(scope, assetID) {
			continue
		}
		var manifest struct {
			Latest string `json:"latest"`
		}
		data, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return Result{}, readErr
		}
		if err := json.Unmarshal(data, &manifest); err != nil || manifest.Latest == "" {
			continue
		}
		versionDir := filepath.Join(filepath.Dir(manifestPath), "versions", manifest.Latest)
		storyPath := filepath.Join(versionDir, "story.json")
		storyData, readErr := os.ReadFile(storyPath)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return Result{}, readErr
		}
		contract, diagnostics := components.ParseStoryContract(storyData)
		if contract == nil || len(components.StoryContractErrors(diagnostics)) > 0 {
			continue
		}
		derived := components.DeriveStoryFields(implementationText(versionDir))
		result.Inspected++
		if len(contract.Args.Fields) == 0 {
			continue
		}
		if storyFieldsEquivalent(contract.Args.Fields, derived) {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.args_fields_hand_authored", AssetID: assetID, File: repoRel(scope.Root, storyPath),
			Message:     fmt.Sprintf("story.json declares %d args.fields but source derivation produced %d", len(contract.Args.Fields), len(derived)),
			Remediation: "Remove the hand-authored args.fields copy; the indexed read model derives fields from the component props type.",
			DocsRef:     "docs/concepts/STORY-CONTRACT.md#derived-requirements",
		})
	}
	return nonEmpty(result, "story-args-fields"), nil
}

func storyFieldsEquivalent(left, right []components.StoryField) bool {
	type comparableField struct {
		Path     string                    `json:"path"`
		Kind     components.StoryFieldKind `json:"kind"`
		Required bool                      `json:"required"`
		Options  []json.RawMessage         `json:"options,omitempty"`
	}
	project := func(fields []components.StoryField) []comparableField {
		out := make([]comparableField, 0, len(fields))
		for _, field := range fields {
			out = append(out, comparableField{Path: field.Path, Kind: field.Kind, Required: field.Required, Options: field.Options})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
		return out
	}
	leftJSON, _ := json.Marshal(project(left))
	rightJSON, _ := json.Marshal(project(right))
	return string(leftJSON) == string(rightJSON)
}
