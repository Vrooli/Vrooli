package gates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"react-component-library/internal/components"
)

func ValidateStoryDistinctness(scope Scope) (Result, error) {
	root := scope.Root
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
		contract, diagnostics := components.ParseStoryContract(raw)
		if contract == nil || len(components.StoryContractErrors(diagnostics)) > 0 || len(contract.Stories) < 2 {
			continue
		}
		seen := map[string]string{}
		specimenExports := map[string]int{}
		for _, story := range contract.Stories {
			if story.Composition != nil && story.Composition.Specimen != nil {
				specimenExports[story.Composition.Specimen.Export]++
			}
			fingerprintStory := story
			fingerprintStory.ID = ""
			fingerprintStory.Name = ""
			canonical, marshalErr := json.Marshal(fingerprintStory)
			if marshalErr != nil {
				return Result{}, marshalErr
			}
			hash := sha256.Sum256(canonical)
			key := hex.EncodeToString(hash[:])
			if previous, exists := seen[key]; exists {
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.story_sibling_duplicate", AssetID: implementationName(path), File: repoRel(root, path),
					Message:     fmt.Sprintf("story %q duplicates sibling story %q", story.ID, previous),
					Remediation: "Give each sibling frame a distinct rendered question, or collapse the duplicate into the matrix story that declares its covers.",
					DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
				})
			} else {
				seen[key] = story.ID
			}
		}
		if len(contract.Stories) >= 3 && len(specimenExports) == 1 {
			for exportName, count := range specimenExports {
				if exportName == "" || count != len(contract.Stories) {
					continue
				}
				result.Findings = append(result.Findings, Finding{
					Code: "catalog.story_sibling_duplicate", AssetID: implementationName(path), File: repoRel(root, path),
					Message:     fmt.Sprintf("all %d sibling frames reuse specimen %q", count, exportName),
					Remediation: "Give anatomy, axis, and boundary questions distinct specimen compositions; an axis matrix may share one specimen only when it is the sole frame for that axis.",
					DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
				})
			}
		}
		if len(seen) == 1 && contract.Stories[0].Role == "" {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.story_sibling_duplicate", AssetID: implementationName(path), File: repoRel(root, path),
				Message:     "multiple stories render one indistinguishable legacy specimen",
				Remediation: "Migrate the contract to role=anatomy plus explicit axis or boundary frames with distinct rendered coverage.",
				DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
			})
		}
	}
	return nonEmpty(result, "story-distinctness"), nil
}

// Evidence freshness is a filesystem gate so the catalog can distinguish a
// current story contract from an older component-test observation.
