package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-component-library/internal/librarywalk"
)

// ValidateHarnessManifest keeps preview harness registrations on the governed
// catalog-gate path. Harnesses are preview-only, but their registry and story
// references are part of the catalog contract.
func ValidateHarnessManifest(scope Scope) (Result, error) {
	root := scope.Root
	harnessRoot := filepath.Join(root, "scenarios", "react-component-library", "harnesses")
	manifestPath := filepath.Join(harnessRoot, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Result{}, err
	}
	var manifest harnessManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Result{}, err
	}
	result := Result{Inspected: 1}
	fail := func(message, file string) {
		result.Findings = append(result.Findings, Finding{Code: "catalog.harness_manifest", AssetID: "__corpus__.harnesses", File: repoRel(root, file), Message: message, Remediation: "Update the preview harness registry and its referenced story contracts.", DocsRef: "docs/guides/asset-preview-composition.md#shared-preview-harnesses"})
	}
	if manifest.SchemaVersion != 1 {
		fail("manifest schemaVersion must be 1", manifestPath)
	}
	if manifest.Kind != "preview-harness-registry" {
		fail("manifest kind is invalid", manifestPath)
	}
	if manifest.Ownership != "preview-only" {
		fail("manifest ownership must be preview-only", manifestPath)
	}
	if len(manifest.Families) == 0 {
		fail("manifest must declare at least one family", manifestPath)
	}

	stableVersion := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	identifier := regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
	forbidden := []*regexp.Regexp{regexp.MustCompile(`from\s+["'](?:\.\./)*components/`), regexp.MustCompile(`from\s+["'](?:\.\./)*primitives/`), regexp.MustCompile(`from\s+["'](?:\.\./)*hooks/`), regexp.MustCompile(`\b(fetch|XMLHttpRequest|WebSocket)\s*\(`), regexp.MustCompile(`\b(localStorage|sessionStorage|indexedDB)\b`), regexp.MustCompile(`document\.cookie`)}
	registrations := map[string]harnessFamily{}
	for _, family := range manifest.Families {
		key := fmt.Sprintf("preview.%s@%s:%s", family.ID, family.Version, family.Export)
		if _, exists := registrations[key]; exists {
			fail("duplicate family registration "+key, manifestPath)
		}
		registrations[key] = family
		if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(family.ID) {
			fail(family.ID+": family id must be a lowercase slug", manifestPath)
		}
		if !stableVersion.MatchString(family.Version) {
			fail(family.ID+": version must be semantic", manifestPath)
		}
		if !identifier.MatchString(family.Export) {
			fail(family.ID+": export must be a JS identifier", manifestPath)
		}
		if len(family.Archetypes) == 0 || len(family.SubjectKinds) == 0 {
			fail(family.ID+": archetypes and subjectKinds are required", manifestPath)
		}
		sourcePath := filepath.Join(harnessRoot, family.ID, "versions", family.Version, family.Export+".tsx")
		source, readErr := os.ReadFile(sourcePath)
		if readErr != nil {
			fail(family.ID+": registered implementation is missing", sourcePath)
			continue
		}
		for _, pattern := range forbidden {
			if pattern.Match(source) {
				fail(family.ID+": forbidden side effect or import", sourcePath)
			}
		}
		if family.ID != "showcase" && !strings.Contains(string(source), "PreviewShowcase") {
			fail(family.ID+": family implementation must use PreviewShowcase", sourcePath)
		}
	}

	referencePath := filepath.Join(harnessRoot, manifest.ReferenceStories)
	referenceRaw, readErr := os.ReadFile(referencePath)
	if readErr != nil {
		fail("manifest must point to an existing referenceStories file", manifestPath)
	} else {
		var references struct {
			Stories []harnessReference `json:"stories"`
		}
		if json.Unmarshal(referenceRaw, &references) != nil {
			fail("reference stories are not valid JSON", referencePath)
		} else {
			covered := map[string]bool{}
			for _, reference := range references.Stories {
				for _, candidate := range manifest.Families {
					if candidate.ID == reference.Family {
						if !containsHarness(candidate.SubjectKinds, reference.SubjectKind) {
							fail(reference.ID+": subject kind is not allowed", referencePath)
						}
						covered[reference.Family] = true
					}
				}
				if !covered[reference.Family] {
					fail(reference.ID+": reference story family is not registered", referencePath)
				}
			}
			for _, family := range manifest.Families {
				if !covered[family.ID] {
					fail(family.ID+": a reference story is required", referencePath)
				}
			}
		}
	}

	storyPaths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return Result{}, err
	}
	result.Inspected += len(storyPaths)
	for _, storyPath := range storyPaths {
		var contract struct {
			Stories []struct {
				ID          string `json:"id"`
				Composition struct {
					Harness *struct {
						Asset   string         `json:"asset"`
						Version string         `json:"version"`
						Export  string         `json:"export"`
						Config  map[string]any `json:"config"`
					} `json:"harness"`
					Specimen any `json:"specimen"`
				} `json:"composition"`
			} `json:"stories"`
		}
		data, readErr := os.ReadFile(storyPath)
		if readErr != nil {
			return Result{}, readErr
		}
		if json.Unmarshal(data, &contract) != nil {
			continue
		}
		for _, story := range contract.Stories {
			if story.Composition.Harness == nil {
				continue
			}
			ref := story.Composition.Harness
			key := fmt.Sprintf("%s@%s:%s", ref.Asset, ref.Version, ref.Export)
			if _, ok := registrations[key]; !ok {
				fail(storyPath+"#"+story.ID+": composition harness is not registered", storyPath)
			}
			if story.Composition.Specimen != nil {
				fail(storyPath+"#"+story.ID+": story cannot declare both specimen and composition harness", storyPath)
			}
		}
	}
	return nonEmpty(result, "harness-manifest"), nil
}

type harnessManifest struct {
	SchemaVersion    int             `json:"schemaVersion"`
	Kind             string          `json:"kind"`
	Ownership        string          `json:"ownership"`
	ReferenceStories string          `json:"referenceStories"`
	Families         []harnessFamily `json:"families"`
}
type harnessFamily struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Export       string   `json:"export"`
	Archetypes   []string `json:"archetypes"`
	SubjectKinds []string `json:"subjectKinds"`
}
type harnessReference struct {
	ID          string `json:"id"`
	Family      string `json:"family"`
	SubjectKind string `json:"subjectKind"`
}

func containsHarness(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
