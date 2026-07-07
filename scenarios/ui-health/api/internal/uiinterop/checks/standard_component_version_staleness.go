/*
Rule: Component Version Staleness
ID: standard_component_version_staleness
Description: Vendored react-component-library copies should match catalog latest
  unless the catalog marks the local version as acceptable.
Why: Templates and scaffolded scenarios can carry @vrooliComponentVersion
  headers without adoption records. If ui-health only checks DB-backed
  adoptions, stale copied primitives silently spread to every future scenario.
Category: standards
Severity: medium
Slot: [D]
SlotFile: ui/src/components/ui
TechStack: React
Recommendation: Refresh stale vendored component copies from
  react-component-library so @vrooliComponentVersion matches catalog latest,
  and replace deprecated versions immediately.
Standard: vrooli-ui-component-canon-v1

GoodExample:
    // ui/src/components/ui/data-table.tsx
    // @vrooliComponentSource react-component-library:DataTable
    // @vrooliComponentVersion 1.10.0

BadExample:
    // ui/src/components/ui/data-table.tsx
    // @vrooliComponentSource react-component-library:DataTable
    // @vrooliComponentVersion 1.0.0

<test-case id="component-version-current-clean" should-fail="false">
  <description>Vendored copy at catalog latest is clean</description>
  <input>
    [go.work]
    go 1.24
    [scenarios/react-component-library/library/components/DataTable/component.json]
    {"libraryId":"react-component-library:DataTable","latest":"1.10.0","draft":"","deprecatedVersions":[]}
    [ui/package.json]
    {"dependencies":{"react":"^18.3.1"}}
    [ui/src/components/ui/data-table.tsx]
    // @vrooliComponentSource react-component-library:DataTable
    // @vrooliComponentVersion 1.10.0
    export function DataTable() { return <table />; }
  </input>
</test-case>

<test-case id="component-version-behind" should-fail="true">
  <description>Record-less vendored copy behind catalog latest is flagged</description>
  <input>
    [go.work]
    go 1.24
    [scenarios/react-component-library/library/components/DataTable/component.json]
    {"libraryId":"react-component-library:DataTable","latest":"1.10.0","draft":"","deprecatedVersions":[]}
    [ui/package.json]
    {"dependencies":{"react":"^18.3.1"}}
    [ui/src/components/ui/data-table.tsx]
    // @vrooliComponentSource react-component-library:DataTable
    // @vrooliComponentVersion 1.0.0
    export function DataTable() { return <table />; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>behind catalog latest 1.10.0</expected-message>
</test-case>

<test-case id="component-version-deprecated" should-fail="true">
  <description>Deprecated vendored versions are flagged even if they are not semver-behind</description>
  <input>
    [go.work]
    go 1.24
    [scenarios/react-component-library/library/components/Button/component.json]
    {"libraryId":"react-component-library:Button","latest":"1.1.0","draft":"","deprecatedVersions":["1.0.0"]}
    [ui/package.json]
    {"dependencies":{"react":"^18.3.1"}}
    [ui/src/components/ui/button.tsx]
    // @vrooliComponentSource react-component-library:Button
    // @vrooliComponentVersion 1.0.0
    export function Button() { return <button />; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>deprecated</expected-message>
</test-case>
*/

package checks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_component_version_staleness", checkComponentVersionStaleness)
}

type componentCatalogEntry struct {
	Name               string
	LibraryID          string
	Latest             string
	Draft              string
	DeprecatedVersions map[string]bool
}

func checkComponentVersionStaleness(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_component_version_staleness"

	files := walkUISource(ctx.ScenarioRoot, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no ui/src directory found",
			Message:    "no ui/src directory found; skipping",
		}
	}
	catalog, err := loadRCLCatalog(ctx.ScenarioRoot)
	if err != nil {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: err.Error(),
			Message:    "react-component-library catalog unavailable; skipping vendored component staleness",
		}
	}

	var violations []uiinterop.Violation
	for _, f := range files {
		libraryID := provenanceField(f.content, "@vrooliComponentSource")
		version := provenanceField(f.content, "@vrooliComponentVersion")
		if libraryID == "" && version == "" {
			continue
		}
		component, ok := catalog[libraryID]
		if !ok {
			violations = append(violations, componentVersionViolation(ruleID, f, libraryID, version, "component removed from react-component-library catalog"))
			continue
		}
		status := componentVersionStatus(component, version)
		if status == "" {
			continue
		}
		violations = append(violations, componentVersionViolation(ruleID, f, libraryID, version, status))
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "stale or deprecated vendored component copies found",
			Violations: violations,
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "vendored component copies match the catalog",
	}
}

func loadRCLCatalog(scenarioRoot string) (map[string]componentCatalogEntry, error) {
	repoRoot := findRepoRoot(scenarioRoot)
	if repoRoot == "" {
		return nil, fmt.Errorf("repo root not found")
	}
	catalogDir := filepath.Join(repoRoot, "scenarios", "react-component-library", "library", "components")
	entries, err := os.ReadDir(catalogDir)
	if err != nil {
		return nil, fmt.Errorf("catalog not found")
	}
	out := map[string]componentCatalogEntry{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, ok := readComponentCatalogEntry(filepath.Join(catalogDir, entry.Name(), "component.json"), entry.Name())
		if ok && meta.LibraryID != "" {
			out[meta.LibraryID] = meta
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("catalog is empty")
	}
	return out, nil
}

func readComponentCatalogEntry(path, fallbackName string) (componentCatalogEntry, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return componentCatalogEntry{}, false
	}
	var doc struct {
		LibraryID          string   `json:"libraryId"`
		Latest             string   `json:"latest"`
		Draft              string   `json:"draft"`
		DeprecatedVersions []string `json:"deprecatedVersions"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return componentCatalogEntry{}, false
	}
	deprecated := make(map[string]bool, len(doc.DeprecatedVersions))
	for _, version := range doc.DeprecatedVersions {
		deprecated[strings.TrimSpace(version)] = true
	}
	return componentCatalogEntry{
		Name:               fallbackName,
		LibraryID:          strings.TrimSpace(doc.LibraryID),
		Latest:             strings.TrimSpace(doc.Latest),
		Draft:              strings.TrimSpace(doc.Draft),
		DeprecatedVersions: deprecated,
	}, true
}

func componentVersionStatus(component componentCatalogEntry, version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "missing @vrooliComponentVersion header"
	}
	if component.DeprecatedVersions[version] {
		return fmt.Sprintf("version %s is deprecated", version)
	}
	if component.Draft != "" && version == component.Draft {
		return fmt.Sprintf("version %s is a draft catalog version", version)
	}
	if component.Latest == "" || version == component.Latest {
		return ""
	}
	cmp, ok := compareComponentVersions(version, component.Latest)
	if ok && cmp >= 0 {
		return ""
	}
	return fmt.Sprintf("version %s is behind catalog latest %s", version, component.Latest)
}

func componentVersionViolation(ruleID string, f uiSourceFile, libraryID, version, detail string) uiinterop.Violation {
	componentName := strings.TrimPrefix(libraryID, "react-component-library:")
	if componentName == "" {
		componentName = "component"
	}
	return uiinterop.Violation{
		RuleID:         ruleID,
		Severity:       "medium",
		Title:          "Vendored component copy is stale",
		Description:    fmt.Sprintf("%s vendors %s at %s: %s", f.relPath, componentName, emptyComponentVersion(version), detail),
		FilePath:       f.relPath,
		Line:           lineOf(f.content, "@vrooliComponentVersion"),
		Recommendation: "Refresh this copied component from react-component-library and keep the @vrooliComponentVersion header synced to catalog latest.",
	}
}

func emptyComponentVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "an unknown version"
	}
	return "version " + version
}

type componentSemver struct {
	major int
	minor int
	patch int
}

func parseComponentSemver(raw string) (componentSemver, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return componentSemver{}, fmt.Errorf("empty version")
	}
	if i := strings.IndexAny(raw, "-+"); i >= 0 {
		raw = raw[:i]
	}
	parts := strings.SplitN(raw, ".", 3)
	out := componentSemver{}
	dst := []*int{&out.major, &out.minor, &out.patch}
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "x" || part == "X" || part == "*" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return componentSemver{}, err
		}
		*dst[i] = value
	}
	return out, nil
}

func compareComponentVersions(a, b string) (int, bool) {
	av, err := parseComponentSemver(a)
	if err != nil {
		return 0, false
	}
	bv, err := parseComponentSemver(b)
	if err != nil {
		return 0, false
	}
	switch {
	case av.major != bv.major:
		if av.major < bv.major {
			return -1, true
		}
		return 1, true
	case av.minor != bv.minor:
		if av.minor < bv.minor {
			return -1, true
		}
		return 1, true
	case av.patch != bv.patch:
		if av.patch < bv.patch {
			return -1, true
		}
		return 1, true
	default:
		return 0, true
	}
}
