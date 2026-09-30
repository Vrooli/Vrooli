// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"react-component-library/internal/librarywalk"

	"react-component-library/internal/themes"
)

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// isRetiredVersion reports whether a materialized version is no longer part
// of an asset's live surface. Deprecated and evicted versions remain on disk
// for reproducibility, but must not make an asset-scoped live validation fail.
func isRetiredVersion(path string) (bool, error) {
	versionDir := filepath.Clean(path)
	if info, err := os.Stat(versionDir); err == nil && !info.IsDir() {
		versionDir = filepath.Dir(versionDir)
	}
	version := filepath.Base(versionDir)
	if filepath.Base(filepath.Dir(versionDir)) != "versions" {
		return false, nil
	}
	manifestPath := filepath.Join(filepath.Dir(filepath.Dir(versionDir)), "component.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		// Support modules use the canonical versioned source layout but are
		// intentionally cataloged through their support metadata rather than a
		// component.json manifest. They still participate in liveness checks;
		// absence of the component manifest is not an unreadable input.
		if os.IsNotExist(err) && strings.Contains(filepath.ToSlash(manifestPath), "/library/support/") {
			return false, nil
		}
		return false, err
	}
	var manifest struct {
		Deprecated []string `json:"deprecatedVersions"`
		Evicted    []string `json:"evictedVersions"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return false, err
	}
	return containsString(manifest.Deprecated, version) || containsString(manifest.Evicted, version), nil
}

func semverLikeLess(left, right string) bool {
	parse := func(value string) [3]int {
		parts := strings.Split(value, ".")
		var out [3]int
		for i := 0; i < len(parts) && i < 3; i++ {
			out[i], _ = strconv.Atoi(parts[i])
		}
		return out
	}
	l, r := parse(left), parse(right)
	for i := range l {
		if l[i] != r[i] {
			return l[i] < r[i]
		}
	}
	return left < right
}

func versionSources(versionDir string) []string {
	var matches []string
	for _, extension := range []string{"*.tsx", "*.ts"} {
		found, _ := librarywalk.Glob(filepath.Join(versionDir, extension))
		matches = append(matches, found...)
	}
	sort.Strings(matches)
	return matches
}

var (
	pxValue = regexp.MustCompile(`--space-[a-z0-9-]+\s*:\s*([0-9.]+)px`)
	// literalDimension splits into two families because they have different
	// fixes. Box spacing has a token ramp to point at; sizing does not — the
	// ramp publishes no icon-size property, so telling an author to "use a
	// semantic token" for w-4/h-4 sends them looking for something that does
	// not exist. Sizing belongs to the Icon primitive's size scale instead.
	literalSpacing = regexp.MustCompile(`\b(?:p|px|py|pt|pr|pb|pl|m|mx|my|mt|mr|mb|ml|gap)-[0-9]+(?:\.[0-9]+)?\b`)
	literalSizing  = regexp.MustCompile(`\b[wh]-[0-9]+(?:\.[0-9]+)?\b`)
	arbitraryPx    = regexp.MustCompile(`\[[0-9.]+px\]`)
)

// literalDimensionFindings classifies a raw dimension by the fix it needs.
// Each family names a different remediation because each has a different
// correct answer; a single shared message would be wrong for two of the three.
func literalDimensionFindings(root, path string, data []byte) []Finding {
	assetID := implementationName(path)
	file := repoRel(root, path)
	var out []Finding
	for _, loc := range literalSpacing.FindAllIndex(data, -1) {
		match := string(data[loc[0]:loc[1]])
		out = append(out, Finding{
			Code: "catalog.tokens_literal", AssetID: assetID, File: file, Line: lineAt(data, loc[0]),
			Message:     fmt.Sprintf("box spacing %q is a raw value, not a ramp step", match),
			Remediation: fmt.Sprintf("Replace %q with the matching gap-space-*/p-space-*/m-space-* utility. The ramp publishes space-3xs through space-2xl on a 4px grid; a raw step does not move when the ramp is retuned, so this element drifts out of rhythm with every tokenized neighbour the first time density changes.", match),
			DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
		})
	}
	for _, loc := range literalSizing.FindAllIndex(data, -1) {
		// Do not treat the sizing suffix in layout utilities such as
		// min-w-0/max-w-0 as a standalone raw width utility.
		if loc[0] > 0 && data[loc[0]-1] == '-' {
			continue
		}
		match := string(data[loc[0]:loc[1]])
		out = append(out, Finding{
			Code: "catalog.tokens_literal", AssetID: assetID, File: file, Line: lineAt(data, loc[0]),
			Message:     fmt.Sprintf("element sized with raw dimension %q", match),
			Remediation: "Size icons through the Icon primitive's size scale (size=\"sm\" | \"md\" | \"lg\") rather than raw width/height utilities. The canonical ramp deliberately publishes no icon-size custom property, so there is no token to substitute here — the scale lives in the primitive's API. For non-icon boxes that genuinely need an intrinsic size, prefer a layout constraint (flex/grid sizing) over a fixed one.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
		})
	}
	for _, loc := range arbitraryPx.FindAllIndex(data, -1) {
		match := string(data[loc[0]:loc[1]])
		out = append(out, Finding{
			Code: "catalog.tokens_literal", AssetID: assetID, File: file, Line: lineAt(data, loc[0]),
			Message:     fmt.Sprintf("arbitrary pixel value %q bypasses the ramp entirely", match),
			Remediation: fmt.Sprintf("Remove the arbitrary value %q. If the nearest ramp step is visually acceptable, use it; if no step fits, the ramp is missing a rung — add it in ui/src/design-tokens.css so every consumer inherits it, rather than encoding the exception at one callsite where nothing can find it later.", match),
			DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
		})
	}
	return out
}

// ValidateTokens checks the shared ramp contract in every design kit and
// rejects non-grid spacing declarations.
func compatibilityGateResult(census Census, affinityOnly bool) Result {
	result := Result{Inspected: census.ComponentsScanned}
	if affinityOnly {
		for _, overclaim := range census.AffinityOverclaims {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.affinity_compatible", AssetID: strings.TrimPrefix(overclaim.LibraryID, "react-component-library:"),
				Message:     fmt.Sprintf("declared affinity %s is broader than derived kit compatibility", overclaim.StyleID),
				Remediation: "Remove the aesthetic-fit claim until the named kit supplies every required token.",
				DocsRef:     "docs/design/TOKEN-DICTIONARY.md",
			})
		}
		return result
	}
	for _, component := range census.Components {
		if component.Verdict != CompatibilityUndefinedVocabulary && component.Verdict != CompatibilityUnsatisfiable {
			continue
		}
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.kit_compatibility", AssetID: strings.TrimPrefix(component.LibraryID, "react-component-library:"),
			Message:     fmt.Sprintf("derived kit compatibility is %s for required tokens %s", component.Verdict, strings.Join(component.RequiredTokens, ", ")),
			Remediation: "Publish the missing semantic vocabulary through the shared base or a deliberate kit override; do not broaden affinity metadata.",
			DocsRef:     "docs/design/TOKEN-DICTIONARY.md",
		})
	}
	return result
}

func tokenValue(tokens []themes.DesignToken, property string) string {
	for _, token := range tokens {
		if token.Name == property {
			return token.Value
		}
	}
	return ""
}

func normalizeTokenValue(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func parseTokenFallbacks(source string) []tokenFallback {
	var result []tokenFallback
	for offset := 0; offset < len(source); {
		relative := strings.Index(source[offset:], "var(")
		if relative < 0 {
			break
		}
		start := offset + relative
		depth, comma, end := 0, -1, -1
		for i := start + len("var("); i < len(source); i++ {
			switch source[i] {
			case '(':
				depth++
			case ')':
				if depth == 0 {
					end = i
				} else {
					depth--
				}
			case ',':
				if depth == 0 && comma < 0 {
					comma = i
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			break
		}
		if comma > 0 {
			property := strings.TrimSpace(source[start+len("var(") : comma])
			if strings.HasPrefix(property, "--") && !strings.Contains(property, "${") {
				result = append(result, tokenFallback{Property: property, Value: strings.TrimSpace(source[comma+1 : end]), Offset: start})
			}
		}
		offset = end + 1
	}
	return result
}

// ValidateLifecycle performs conservative static checks over hook/service/
// adapter/generator sources. It deliberately prefers a finding over a green
// result when cleanup evidence is absent.
func isStorySource(path string) bool {
	base := filepath.Base(path)
	return base == "story.ts" || base == "story.tsx"
}

func isTestSource(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".spec.tsx")
}

// hasBrowserAccessOutsideEffects keeps the static SSR check conservative while
// understanding the one React lifecycle boundary that is guaranteed not to
// execute during server rendering. Browser access in render, module scope, or
// an arbitrary exported callback still requires an explicit guard.
func hasBrowserAccessOutsideEffects(text string) bool {
	remaining := []byte(text)
	for _, start := range effectCallbackRanges(text) {
		for index := start[0]; index < start[1] && index < len(remaining); index++ {
			remaining[index] = ' '
		}
	}
	textWithoutEffects := string(remaining)
	return (strings.Contains(textWithoutEffects, "window.") && !strings.Contains(textWithoutEffects, "typeof window")) ||
		(strings.Contains(textWithoutEffects, "document.") && !strings.Contains(textWithoutEffects, "typeof document"))
}

func effectCallbackRanges(text string) [][2]int {
	var ranges [][2]int
	for offset := 0; offset < len(text); {
		match := strings.Index(text[offset:], "useEffect")
		if match < 0 {
			break
		}
		start := offset + match
		after := start + len("useEffect")
		if after < len(text) && isIdentifierPart(text[after]) {
			offset = after
			continue
		}
		for after < len(text) && (text[after] == ' ' || text[after] == '\n' || text[after] == '\r' || text[after] == '	') {
			after++
		}
		if after >= len(text) || text[after] != '(' {
			offset = after
			continue
		}
		arrow := strings.Index(text[after:], "=>")
		if arrow < 0 {
			break
		}
		arrow += after + 2
		body := arrow
		for body < len(text) && (text[body] == ' ' || text[body] == '\n' || text[body] == '\r' || text[body] == '	') {
			body++
		}
		if body >= len(text) || text[body] != '{' {
			offset = arrow
			continue
		}
		end, ok := matchingBrace(text, body)
		if !ok {
			break
		}
		ranges = append(ranges, [2]int{body, end + 1})
		offset = end + 1
	}
	return ranges
}

func matchingBrace(text string, open int) (int, bool) {
	depth := 0
	for index := open; index < len(text); index++ {
		switch text[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index, true
			}
		}
	}
	return 0, false
}

func isIdentifierPart(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '_'
}

func implementationName(path string) string {
	path = filepath.Clean(path)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		path = filepath.Dir(path)
	}
	if filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path)))) == "support" {
		assetName := filepath.Base(filepath.Dir(filepath.Dir(path)))
		for current := filepath.Dir(path); current != filepath.Dir(current); current = filepath.Dir(current) {
			assetPaths, _ := librarywalk.Glob(filepath.Join(current, "catalog", "assets", "*", "*.json"))
			for _, assetPath := range assetPaths {
				data, readErr := os.ReadFile(assetPath)
				if readErr != nil {
					continue
				}
				var doc struct {
					Asset struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"asset"`
				}
				if json.Unmarshal(data, &doc) == nil && compactIdentity(doc.Asset.Name) == compactIdentity(assetName) {
					return doc.Asset.ID
				}
			}
		}
	}
	for {
		data, err := os.ReadFile(filepath.Join(path, "component.json"))
		if err == nil {
			var manifest struct {
				CatalogID string `json:"catalogId"`
			}
			if json.Unmarshal(data, &manifest) == nil && manifest.CatalogID != "" {
				return manifest.CatalogID
			}
		}
		parent := filepath.Dir(path)
		if parent == path {
			return ""
		}
		path = parent
	}
}

func compactIdentity(value string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(value) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			out.WriteRune(r)
		}
	}
	return out.String()
}
