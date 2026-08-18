package components

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// TestCatalogStoryConformance is the deterministic cutover report.  It reads
// the committed migration inventory rather than the index, so an omitted file
// cannot disappear merely because it was not projected into storage.
func TestCatalogStoryConformance(t *testing.T) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate conformance test")
	}
	scenarioRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	libraryRoot := filepath.Join(scenarioRoot, "library")

	rawInventory, err := os.ReadFile(filepath.Join(libraryRoot, "story-migration-inventory.json"))
	if err != nil {
		t.Fatalf("read story migration inventory: %v", err)
	}
	var inventory struct {
		SchemaVersion int `json:"schemaVersion"`
		Assets        []struct {
			Kind     string   `json:"kind"`
			ID       string   `json:"id"`
			Versions []string `json:"versions"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(rawInventory, &inventory); err != nil {
		t.Fatalf("decode story migration inventory: %v", err)
	}
	if inventory.SchemaVersion != 1 || len(inventory.Assets) == 0 {
		t.Fatalf("invalid story migration inventory: schemaVersion=%d assets=%d", inventory.SchemaVersion, len(inventory.Assets))
	}

	var failures []string
	for _, asset := range inventory.Assets {
		if asset.Kind != string(StoryKindComponent) && asset.Kind != string(StoryKindHook) {
			failures = append(failures, "inventory "+asset.ID+": unsupported kind "+asset.Kind)
			continue
		}
		for _, version := range asset.Versions {
			versionDir := filepath.Join(libraryRoot, asset.Kind+"s", asset.ID, "versions", version)
			storyPath := filepath.Join(versionDir, "story.json")
			rawStory, err := os.ReadFile(storyPath)
			prefix := asset.Kind + "/" + asset.ID + "@" + version
			if err != nil {
				failures = append(failures, prefix+": missing readable story.json ("+err.Error()+")")
				continue
			}
			contract, diagnostics := ParseStoryContract(rawStory)
			if contract == nil || len(diagnostics) > 0 {
				for _, diagnostic := range diagnostics {
					failures = append(failures, prefix+": story.json "+diagnostic.Error())
				}
				continue
			}
			if string(contract.Kind) != asset.Kind {
				failures = append(failures, prefix+": story kind "+string(contract.Kind)+" does not match inventory kind "+asset.Kind)
			}
			if contract.SchemaVersion != 3 {
				failures = append(failures, prefix+": story schemaVersion must be 3")
			}
			liveSource := false
			if _, statErr := os.Stat(filepath.Join(versionDir, "story.tsx")); statErr == nil {
				liveSource = true
			} else if !os.IsNotExist(statErr) {
				failures = append(failures, prefix+": inspect story.tsx: "+statErr.Error())
			}
			for storyIndex, story := range contract.Stories {
				if story.Mode != StoryModeLive && story.Mode != StoryModePinned {
					failures = append(failures, fmt.Sprintf("%s: story %d must declare mode live or pinned", prefix, storyIndex))
					continue
				}
				if liveSource && story.Mode != StoryModeLive {
					failures = append(failures, fmt.Sprintf("%s: story %q has story.tsx but is not live", prefix, story.ID))
				}
				if !liveSource && story.Mode != StoryModePinned {
					failures = append(failures, fmt.Sprintf("%s: story %q has no story.tsx but is not pinned", prefix, story.ID))
				}
			}
			for _, legacy := range []string{"examples.json", "test-contract.json", "setup.json", "controls.json"} {
				if _, err := os.Stat(filepath.Join(versionDir, legacy)); err == nil {
					failures = append(failures, prefix+": legacy metadata must be removed: "+legacy)
				} else if !os.IsNotExist(err) {
					failures = append(failures, prefix+": inspect legacy metadata "+legacy+": "+err.Error())
				}
			}
			if info, err := os.Stat(filepath.Join(versionDir, "controls")); err == nil && info.IsDir() {
				failures = append(failures, prefix+": legacy controls directory must be removed")
			}
		}
	}
	sort.Strings(failures)
	if len(failures) > 0 {
		t.Fatalf("story-contract conformance failed:\n%s", strings.Join(failures, "\n"))
	}
}
