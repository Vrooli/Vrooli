package gates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ValidateStress(scope Scope) (Result, error) {
	root := scope.Root
	const stressDocs = "docs/internal/TESTING.md"
	return validateActiveSources(scope, "stress", func(asset assetDoc, source string) defect {
		_ = source
		manifest, _, found, err := implementationSource(root, asset.Asset.ID)
		if err != nil || !found {
			return defect{
				Message:     "no active implementation is available to the stress runner",
				Remediation: "Point this catalog asset at a library implementation, or drop the stress gate from its appliesTo list. The stress runner has nothing to drive without one, and a gate that inspects nothing must not report a pass.",
				DocsRef:     stressDocs,
			}
		}
		data, readErr := os.ReadFile(manifest)
		if readErr != nil {
			return defect{
				Message:     fmt.Sprintf("implementation manifest could not be read: %v", readErr),
				Remediation: fmt.Sprintf("Restore or repair %s. Without a readable manifest the runner cannot resolve which version to stress.", repoRel(root, manifest)),
				DocsRef:     stressDocs,
			}
		}
		var doc struct {
			Latest string `json:"latest"`
		}
		if json.Unmarshal(data, &doc) != nil || doc.Latest == "" {
			return defect{
				Message:     "implementation manifest declares no released version",
				Remediation: fmt.Sprintf("Set \"latest\" in %s to the version this asset publishes. The stress runner resolves its specimens through the released version; with none declared there is nothing to stress.", repoRel(root, manifest)),
				DocsRef:     stressDocs,
			}
		}
		storyPath := filepath.Join(filepath.Dir(manifest), "versions", doc.Latest, "story.json")
		story, readErr := os.ReadFile(storyPath)
		if readErr != nil || len(bytes.TrimSpace(story)) == 0 {
			return defect{
				Message:     fmt.Sprintf("released version %s has no non-empty story contract", doc.Latest),
				Remediation: fmt.Sprintf("Author %s with the adversarial specimens this asset must survive: long strings, empty collections, disabled states, and large numeric values. The story contract is the stress fixture boundary — it is the only place those specimens are version-pinned, so an asset without one is never driven past its happy path.", repoRel(root, storyPath)),
				DocsRef:     stressDocs,
			}
		}
		return ok()
	})
}

// ValidateIntegration checks the source-level integration boundary shared by
// every released renderable asset. The actual manager/browser integration is
// recorded by component-test and Experience Manager evidence; this runner
// prevents a source-only asset from receiving an integration pass.
