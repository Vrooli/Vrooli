package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"react-component-library/internal/components"
	"react-component-library/internal/librarywalk"
)

type requiredStory struct {
	Name       string
	Role       string
	Capability string
	Reason     string
}

// ValidateStoryCoverage computes a small, explainable story floor from the
// component's derived props. Every requirement has a capability reason in the
// finding, so a missing tile can be repaired from evidence rather than from a
// hidden classifier decision.
func ValidateStoryCoverage(scope Scope) (Result, error) {
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
			AssetKind string `json:"assetKind"`
			Latest    string `json:"latest"`
		}
		data, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			return Result{}, readErr
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return Result{}, err
		}
		if !renderableStoryAssetKind(manifest.AssetKind) {
			continue
		}
		if strings.TrimSpace(manifest.Latest) == "" {
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
		result.Inspected++
		source := implementationText(versionDir)
		fields := components.DeriveStoryFields(source)
		required := requiredStorySet(assetID, source, fields)
		observed := observedStorySet(contract.Stories)
		for _, wanted := range required {
			if storySetContains(observed, wanted) {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.story_coverage_missing", AssetID: assetID, File: repoRel(scope.Root, storyPath),
				Message:     fmt.Sprintf("required %s story %q is missing; capability %s (%s)", wanted.Role, wanted.Name, wanted.Capability, wanted.Reason),
				Remediation: fmt.Sprintf("Add a %s story named %q with args that render the required state.", wanted.Role, wanted.Name),
				DocsRef:     "docs/concepts/STORY-CONTRACT.md#story-roles",
			})
		}
		result.Findings = append(result.Findings, boundaryAxisFindings(scope.Root, storyPath, assetID, contract, fields)...)
	}
	return nonEmpty(result, "story-coverage"), nil
}

func renderableStoryAssetKind(assetKind string) bool {
	switch strings.ToLower(strings.TrimSpace(assetKind)) {
	case "primitive", "component", "pattern", "page-template", "navigation":
		return true
	default:
		return false
	}
}

func implementationText(versionDir string) string {
	entries, _ := os.ReadDir(versionDir)
	var builder strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".tsx") && !strings.HasSuffix(entry.Name(), ".ts")) || strings.HasPrefix(entry.Name(), "story") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(versionDir, entry.Name()))
		if err == nil {
			builder.Write(data)
		}
	}
	return builder.String()
}

func requiredStorySet(assetID, source string, fields []components.StoryField) []requiredStory {
	text := strings.ToLower(assetID + "\n" + source)
	seen := map[string]requiredStory{}
	add := func(name, role, capability, reason string) {
		key := storyKey(role, name)
		if _, exists := seen[key]; !exists {
			seen[key] = requiredStory{Name: name, Role: role, Capability: capability, Reason: reason}
		}
	}
	add("anatomy", "anatomy", "renderable", "every renderable asset needs a representative specimen")
	for _, field := range fields {
		if field.Kind == components.StoryFieldEnum {
			add(field.Path, "axis", "enum:"+field.Path, "derived enum field has one story for its declared options")
		}
	}
	content := false
	disableable := false
	collection := false
	for _, field := range fields {
		switch {
		case field.Kind == components.StoryFieldText:
			content = true
		case strings.EqualFold(field.Path, "disabled") && field.Kind == components.StoryFieldBoolean:
			disableable = true
		case field.Kind == components.StoryFieldArray && (strings.Contains(text, strings.ToLower(field.Path)+".map") || strings.Contains(text, "map(")):
			collection = true
		}
	}
	if content {
		add("empty", "boundary", "content-bearing", "derived text or children field can render no content")
		add("overflow", "boundary", "content-bearing", "derived text or children field can render long content")
	}
	async := strings.Contains(text, "loading") || strings.Contains(text, "pending") || strings.Contains(text, "async") || strings.Contains(text, "query") || strings.Contains(text, "request")
	if async {
		add("loading", "boundary", "async", "source or contract names an asynchronous lifecycle")
		add("error", "boundary", "async", "source or contract names an asynchronous lifecycle")
	}
	if disableable {
		add("disabled", "boundary", "disableable", "derived boolean disabled field is public")
	}
	if collection {
		add("empty", "boundary", "collection", "derived array field is mapped into a collection")
		add("one", "boundary", "collection", "derived array field is mapped into a collection")
		add("many", "boundary", "collection", "derived array field is mapped into a collection")
	}
	result := make([]requiredStory, 0, len(seen))
	for _, value := range seen {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Role == result[j].Role {
			return result[i].Name < result[j].Name
		}
		return result[i].Role < result[j].Role
	})
	return result
}

func observedStorySet(stories []components.StoryDefinition) map[string]bool {
	observed := map[string]bool{}
	for _, story := range stories {
		role := strings.ToLower(strings.TrimSpace(story.Role))
		values := []string{story.ID, story.Name, story.Description}
		values = append(values, story.States...)
		for _, value := range values {
			name := normalizeStoryName(value)
			if role != "" && name != "" {
				observed[storyKey(role, name)] = true
				if role == "anatomy" {
					observed[storyKey(role, "anatomy")] = true
				}
			}
		}
	}
	return observed
}

func storySetContains(observed map[string]bool, wanted requiredStory) bool {
	if observed[storyKey(wanted.Role, wanted.Name)] {
		return true
	}
	wantedName := normalizeStoryName(wanted.Name)
	for key := range observed {
		if strings.HasPrefix(key, normalizeStoryName(wanted.Role)+":") && strings.Contains(key, wantedName) {
			return true
		}
	}
	return false
}

func storyKey(role, name string) string {
	return normalizeStoryName(role) + ":" + normalizeStoryName(name)
}

func normalizeStoryName(value string) string {
	var builder strings.Builder
	previousWasSeparator := true
	for _, r := range strings.TrimSpace(value) {
		if r == '-' || r == '_' || unicode.IsSpace(r) {
			if !previousWasSeparator {
				builder.WriteByte('-')
				previousWasSeparator = true
			}
			continue
		}
		if unicode.IsUpper(r) && !previousWasSeparator {
			builder.WriteByte('-')
		}
		builder.WriteRune(unicode.ToLower(r))
		previousWasSeparator = false
	}
	return strings.Trim(builder.String(), "-")
}

func boundaryAxisFindings(root, storyPath, assetID string, contract *components.StoryContract, fields []components.StoryField) []Finding {
	enums := map[string]bool{}
	for _, field := range fields {
		if field.Kind == components.StoryFieldEnum {
			enums[field.Path] = true
		}
	}
	if len(enums) == 0 {
		return nil
	}
	var findings []Finding
	for _, story := range contract.Stories {
		if strings.ToLower(story.Role) != "boundary" {
			continue
		}
		var args map[string]json.RawMessage
		if json.Unmarshal(story.Args, &args) != nil || len(args) == 0 {
			continue
		}
		allEnum := true
		for path := range args {
			if !enums[path] {
				allEnum = false
				break
			}
		}
		if !allEnum {
			continue
		}
		findings = append(findings, Finding{Code: "catalog.story_boundary_is_axis_value", AssetID: assetID, File: repoRel(root, storyPath), Message: fmt.Sprintf("boundary story %q only sets enum field values", story.ID), Remediation: "Convert this story to an axis story, or add a non-enum condition that makes it a boundary state.", DocsRef: "docs/concepts/STORY-CONTRACT.md#story-roles"})
	}
	return findings
}
