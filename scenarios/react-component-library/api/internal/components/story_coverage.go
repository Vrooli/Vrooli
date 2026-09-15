package components

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ErrStoryCoverageRequired prevents an authored release from being promoted
// while its public enum surface has unrendered values. Drafts intentionally do
// not use this error so authors can stage an incomplete contract.
type ErrStoryCoverageRequired struct {
	LibraryID string
	Version   string
	Gaps      []StoryCoverageGap
}

func (e ErrStoryCoverageRequired) Error() string {
	return fmt.Sprintf("%s@%s has %d story coverage gap(s); add a story for every enum value", e.LibraryID, e.Version, len(e.Gaps))
}

func releaseStoryCoverage(root string, c Component, version string) error {
	manifestDir := filepath.Dir(c.ManifestPath)
	if manifestDir == "." || manifestDir == "" {
		manifestDir = "components"
		if c.AssetKind == AssetKindHook {
			manifestDir = "hooks"
		}
	}
	path := filepath.Join(root, manifestDir, "versions", version, "story.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	contract, diagnostics := ParseStoryContract(raw)
	if len(StoryContractErrors(diagnostics)) > 0 || contract == nil {
		return nil
	}
	if gaps := StoryCoverageGaps(contract); len(gaps) > 0 {
		return ErrStoryCoverageRequired{LibraryID: c.LibraryID, Version: version, Gaps: gaps}
	}
	return nil
}

// StoryCoverageFromJSON is the transport-independent check used by the
// component-tests provider and promotion tests.
func StoryCoverageFromJSON(raw []byte) ([]StoryCoverageGap, error) {
	var contract StoryContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		return nil, err
	}
	return StoryCoverageGaps(&contract), nil
}
