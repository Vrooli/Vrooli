/*
Rule: Component Kit Affinity
ID: standard_component_kit_affinity
Description: A governed component must not reference a design-kit-private utility
  unless its manifest declares native affinity for that kit.
Why: Shared components can use the semantic contract in every kit. Kit-private
  utilities are valid only for a component that was authored and verified for
  that kit; a compatible declaration is not evidence that private tokens work.
Category: standards
Severity: warning
Slot: [D]
SlotFile: ui/src/components
TechStack: React
Recommendation: Replace the private utility with a shared semantic token or
  declare native affinity and verify the component for the referenced kit.
Standard: vrooli-ui-design-kit-contract-v1

<test-case id="kit-affinity-shared-contract" should-fail="false">
  <description>Shared semantic tokens are portable without native kit affinity</description>
  <input>
    [ui/src/components/Card.tsx]
    // @vrooliComponentSource react-component-library:Card
    export const Card = () =&gt; &lt;div className="rounded-panel bg-app-surface" /&gt;;
  </input>
</test-case>

<test-case id="kit-affinity-private-without-manifest" should-fail="true">
  <description>A private token is not portable without native affinity</description>
  <input>
    [ui/src/components/Card.tsx]
    // @vrooliComponentSource react-component-library:Card
    export const Card = () =&gt; &lt;div className="display-glow" /&gt;;
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>does not declare native affinity</expected-message>
</test-case>
*/

package checks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_component_kit_affinity", checkComponentKitAffinity)
}

type componentKitManifest struct {
	LibraryID   string `json:"libraryId"`
	DesignStyle []struct {
		StyleID  string `json:"styleId"`
		Affinity string `json:"affinity"`
	} `json:"designStyles"`
}

func checkComponentKitAffinity(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_component_kit_affinity"
	files := sourceFiles(ctx, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{RuleID: ruleID, Skipped: true, SkipReason: "no UI source files", Message: "no UI source files; skipping component kit affinity"}
	}
	repoRoot := findRepoRoot(ctx.ScenarioRoot)
	kitPrivateTokenPatterns, patternsErr := loadKitPrivateTokenPatterns(repoRoot)
	if patternsErr != nil {
		return uiinterop.RuleResult{RuleID: ruleID, Skipped: true, SkipReason: patternsErr.Error(), Message: "design-kit metadata unavailable; skipping component kit affinity"}
	}
	violations := []uiinterop.Violation{}
	for _, source := range files {
		libraryID := provenanceField(source.Content, "@vrooliComponentSource")
		if libraryID == "" {
			continue
		}
		componentName := strings.TrimPrefix(libraryID, "react-component-library:")
		var data []byte
		var err error
		if repoRoot != "" {
			manifestPath := filepath.Join(repoRoot, "scenarios", "react-component-library", "library", "components", componentName, "component.json")
			data, err = os.ReadFile(manifestPath)
		} else {
			err = os.ErrNotExist
		}
		var manifest componentKitManifest
		if err == nil {
			_ = json.Unmarshal(data, &manifest)
		}
		native := map[string]bool{}
		for _, style := range manifest.DesignStyle {
			native[style.StyleID] = strings.EqualFold(style.Affinity, "native")
		}
		for kit, pattern := range kitPrivateTokenPatterns {
			if native[kit] {
				continue
			}
			matches := uniqueStrings(trimRegexMatches(pattern.FindAllString(source.Content, -1)))
			if len(matches) == 0 {
				continue
			}
			sort.Strings(matches)
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "warning",
				Title:          "Component uses a foreign kit-private token",
				Description:    fmt.Sprintf("%s references %s-private token(s) %s but does not declare native affinity for %s", source.RelPath, kit, strings.Join(matches, ", "), kit),
				FilePath:       source.RelPath,
				Line:           lineOf(source.Content, matches[0]),
				Recommendation: "Use shared app-* / rounded-* / ramp utilities, or declare native affinity and verify the component against that kit.",
			})
		}
	}
	if len(violations) == 0 {
		return uiinterop.RuleResult{RuleID: ruleID, Passed: true, Message: "governed components use only shared or natively-affined kit tokens"}
	}
	return uiinterop.RuleResult{RuleID: ruleID, Passed: false, Message: "component kit affinity contract violated", Violations: violations}
}

func loadKitPrivateTokenPatterns(repoRoot string) (map[string]*regexp.Regexp, error) {
	designRoot := filepath.Join(repoRoot, "templates", "design")
	if _, err := os.Stat(designRoot); err != nil {
		_, source, _, ok := runtime.Caller(0)
		if !ok {
			return nil, fmt.Errorf("design-kit registry unavailable")
		}
		designRoot = filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "..", "..", "..", "..", "templates", "design"))
	}
	paths, err := filepath.Glob(filepath.Join(designRoot, "*", "metadata.json"))
	if err != nil {
		return nil, err
	}
	patterns := map[string]*regexp.Regexp{}
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var metadata struct {
			ID                   string   `json:"id"`
			PrivateTokens        []string `json:"privateTokens"`
			PrivateTokenPrefixes []string `json:"privateTokenPrefixes"`
		}
		if decodeErr := json.Unmarshal(raw, &metadata); decodeErr != nil {
			return nil, decodeErr
		}
		var alternatives []string
		for _, token := range metadata.PrivateTokens {
			alternatives = append(alternatives, regexp.QuoteMeta(token))
		}
		for _, prefix := range metadata.PrivateTokenPrefixes {
			alternatives = append(alternatives, regexp.QuoteMeta(prefix)+`[A-Za-z0-9_-]+`)
		}
		if metadata.ID != "" && len(alternatives) > 0 {
			patterns[metadata.ID] = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])(?:` + strings.Join(alternatives, "|") + `)(?:$|[^A-Za-z0-9_-])`)
		}
	}
	return patterns, nil
}
