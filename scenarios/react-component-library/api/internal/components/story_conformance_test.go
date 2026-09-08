package components

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestCatalogStoryConformance scans the authored corpus directly, so an
// omitted file cannot disappear merely because it was not projected into
// storage.
func TestCatalogStoryConformance(t *testing.T) {
	t.Helper()
	scenarioRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	libraryRoot := filepath.Join(scenarioRoot, "library")

	var failures []string
	storyCount := 0
	err := filepath.WalkDir(libraryRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "story.json" {
			return nil
		}
		storyCount++
		versionDir := filepath.Dir(path)
		prefix := strings.TrimPrefix(versionDir, libraryRoot+string(filepath.Separator))
		rawStory, readErr := os.ReadFile(path)
		if readErr != nil {
			failures = append(failures, prefix+": unreadable story.json ("+readErr.Error()+")")
			return nil
		}
		contract, diagnostics := ParseStoryContract(rawStory)
		if contract == nil || len(StoryContractErrors(diagnostics)) > 0 {
			for _, diagnostic := range StoryContractErrors(diagnostics) {
				failures = append(failures, prefix+": story.json "+diagnostic.Error())
			}
			return nil
		}
		if contract.SchemaVersion != 5 {
			failures = append(failures, prefix+": story schemaVersion must be 5")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan story corpus: %v", err)
	}
	if storyCount == 0 {
		t.Fatal("story corpus is empty")
	}
	sort.Strings(failures)
	if len(failures) > 0 {
		t.Fatalf("story-contract conformance failed:\n%s", strings.Join(failures, "\n"))
	}
}
