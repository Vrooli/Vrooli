package gates

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// SurfaceVerdict is the reconciliation of the author-facing static oracle and
// the browser-owned computed-style capture oracle.
type SurfaceVerdict string

const (
	SurfacePass          SurfaceVerdict = "pass"
	SurfaceRenderedWrong SurfaceVerdict = "renders-wrong-elevation"
	SurfaceHardCoded     SurfaceVerdict = "correct-pixels-hard-coded-path"
	SurfaceBothMismatch  SurfaceVerdict = "static-and-rendered-mismatch"
	SurfaceUnmeasured    SurfaceVerdict = "unmeasured"
)

func classifySurface(staticApplied, captured, renderedMatches bool) SurfaceVerdict {
	if !captured {
		return SurfaceUnmeasured
	}
	if staticApplied && renderedMatches {
		return SurfacePass
	}
	if staticApplied {
		return SurfaceRenderedWrong
	}
	if renderedMatches {
		return SurfaceHardCoded
	}
	return SurfaceBothMismatch
}

type capturedSurfaceNode struct {
	DOM struct {
		Attributes map[string]string `json:"attributes"`
	} `json:"dom"`
	ComputedStyle map[string]string     `json:"computedStyle"`
	Children      []capturedSurfaceNode `json:"children"`
}

// ValidateSurfaceDiscipline reconciles the static ramp-use oracle with the
// latest BAS computed-style capture. Missing capture is explicitly
// unmeasured; it can never become a pass merely because source text mentions a
// token.
func ValidateSurfaceDiscipline(scope Scope) (Result, error) {
	root := scope.Root
	assets, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	captures, err := loadSurfaceCaptures(root)
	if err != nil {
		return Result{}, err
	}
	tokens, err := loadElevationTokens(root)
	if err != nil {
		return Result{}, err
	}
	result := Result{SurfaceCounts: map[string]int{}}
	for _, asset := range assets {
		manifest, source, found, sourceErr := implementationSource(root, asset.Asset.ID)
		if sourceErr != nil {
			return Result{}, sourceErr
		}
		if !found {
			continue
		}
		sourceData, err := os.ReadFile(source)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		result.InspectedAssets = appendUnique(result.InspectedAssets, asset.Asset.ID)
		staticApplied := sourceUsesSurfaceRamp(string(sourceData), asset.Asset.Surface)
		captured, ok := captures[asset.Asset.ID]
		if !ok {
			result.UnmeasuredAssets = append(result.UnmeasuredAssets, asset.Asset.ID)
		}
		verdict := classifySurface(staticApplied, ok, ok && renderedSurfaceMatches(captured, asset.Asset.Surface, tokens))
		result.SurfaceCounts[string(verdict)]++
		switch verdict {
		case SurfaceRenderedWrong:
			result.Findings = append(result.Findings, Finding{Code: "catalog.surface_wrong_elevation", AssetID: asset.Asset.ID, File: repoRel(root, manifest), Message: "renders the wrong elevation", Remediation: "Apply the declared SURFACE_ELEVATIONS value to the stamped asset root so its computed box-shadow matches the declared surface ramp."})
		case SurfaceHardCoded:
			result.Findings = append(result.Findings, Finding{Code: "catalog.surface_hard_coded", AssetID: asset.Asset.ID, File: repoRel(root, manifest), Message: "correct pixels, hard-coded path", Remediation: "Use SURFACE_ELEVATIONS in the rendered JSX instead of reproducing the ramp value through a raw class or style."})
		case SurfaceBothMismatch:
			result.Findings = append(result.Findings, Finding{Code: "catalog.surface_unreconciled", AssetID: asset.Asset.ID, File: repoRel(root, manifest), Message: "static ramp usage and rendered elevation both disagree with the declared surface", Remediation: "Declare the intended surface and make the rendered root consume the matching SURFACE_ELEVATIONS value."})
		}
	}
	if result.Inspected == 0 || len(captures) == 0 {
		result.Status = "unmeasured"
	}
	stampReport, stampErr := LoadStampReport(root)
	if stampErr != nil {
		return Result{}, stampErr
	}
	result.UnstampedAssets, result.UncapturedAssets = classifyUnmeasured(stampReport, result.UnmeasuredAssets)
	result.SurfaceCounts["unstamped"] = len(result.UnstampedAssets)
	result.SurfaceCounts["uncaptured"] = len(result.UncapturedAssets)
	return nonEmpty(result, "surface-discipline"), nil
}

func sourceUsesSurfaceRamp(source, surface string) bool {
	if !strings.Contains(source, "SURFACE_ELEVATIONS") {
		return false
	}
	name := surface
	if name == "base" {
		name = "flat"
	}
	if name == "none" || name == "" {
		return false
	}
	return strings.Contains(source, "SURFACE_ELEVATIONS."+name) || strings.Contains(source, "SURFACE_ELEVATIONS[\""+name+"\"]") || strings.Count(source, "SURFACE_ELEVATIONS") > 1
}

func loadElevationTokens(root string) (map[string]string, error) {
	data, err := os.ReadFile(root + "/scenarios/react-component-library/ui/src/design-tokens.css")
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	pattern := regexp.MustCompile(`(?m)--elev-([a-z-]+)\s*:\s*([^;]+);`)
	result := map[string]string{}
	for _, match := range pattern.FindAllStringSubmatch(string(data), -1) {
		result[match[1]] = strings.ToLower(strings.TrimSpace(match[2]))
	}
	return result, nil
}

func renderedSurfaceMatches(captured, declared string, tokens map[string]string) bool {
	actual := strings.ToLower(strings.TrimSpace(captured))
	if declared == "none" {
		return actual == "none"
	}
	name := declared
	if name == "base" {
		name = "flat"
	}
	expected, ok := tokens[name]
	return ok && equivalentBoxShadows(actual, expected)
}

// equivalentBoxShadows compares authored and computed CSS forms. Chromium
// serializes computed shadows with the color first and explicit zero spread,
// while the authored token uses the CSS grammar's offset-first form and may
// omit spread. Canonicalizing both forms keeps this oracle about the declared
// ramp rather than about browser serialization details.
func equivalentBoxShadows(actual, expected string) bool {
	if actual == expected {
		return true
	}
	actualLayers := canonicalShadowLayers(actual)
	expectedLayers := canonicalShadowLayers(expected)
	if len(actualLayers) != len(expectedLayers) {
		return false
	}
	for i := range actualLayers {
		if actualLayers[i] != expectedLayers[i] {
			return false
		}
	}
	return true
}

func canonicalShadowLayers(value string) []string {
	layers := splitCSSList(value)
	canonical := make([]string, 0, len(layers))
	for _, layer := range layers {
		layer = strings.TrimSpace(layer)
		if layer == "" {
			continue
		}
		tokens := splitCSSWords(layer)
		color := ""
		numbers := make([]string, 0, len(tokens))
		for _, token := range tokens {
			if isCSSColorToken(token) {
				color = normalizeCSSValue(token)
				continue
			}
			numbers = append(numbers, normalizeCSSNumber(token))
		}
		for len(numbers) < 4 {
			numbers = append(numbers, "0")
		}
		canonical = append(canonical, color+":"+strings.Join(numbers, ","))
	}
	return canonical
}

func splitCSSList(value string) []string {
	var out []string
	start, depth := 0, 0
	for i, r := range value {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				out = append(out, value[start:i])
				start = i + 1
			}
		}
	}
	return append(out, value[start:])
}

func splitCSSWords(value string) []string {
	var out []string
	start, depth := -1, 0
	flush := func(end int) {
		if start >= 0 {
			out = append(out, value[start:end])
			start = -1
		}
	}
	for i, r := range value {
		switch {
		case r == '(':
			depth++
			if start < 0 {
				start = i
			}
		case r == ')':
			if depth > 0 {
				depth--
			}
		case depth == 0 && (r == ' ' || r == '\t' || r == '\n' || r == '\r'):
			flush(i)
		default:
			if start < 0 {
				start = i
			}
		}
	}
	flush(len(value))
	return out
}

func isCSSColorToken(token string) bool {
	lower := strings.ToLower(token)
	return strings.HasPrefix(lower, "rgb(") || strings.HasPrefix(lower, "rgba(") ||
		strings.HasPrefix(lower, "hsl(") || strings.HasPrefix(lower, "hsla(") ||
		strings.HasPrefix(lower, "color(") || strings.HasPrefix(lower, "#")
}

func normalizeCSSNumber(value string) string {
	value = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(value), "px"))
	if value == "-0" || value == "0" {
		return "0"
	}
	return value
}

func normalizeCSSValue(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), "")
}

func loadSurfaceCaptures(root string) (map[string]string, error) {
	path := root + "/scenarios/experience-manager/data/experience-manager.db"
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	db, err := openGateDB(context.Background(), path)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(context.Background(), `SELECT ax_node_json FROM reconcile_evidence WHERE scenario = 'react-component-library' AND document_kind = 'component' AND ax_node_json LIKE '%data-rcl-asset%' ORDER BY checked_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var node capturedSurfaceNode
		if json.Unmarshal([]byte(raw), &node) != nil {
			continue
		}
		collectSurfaceNodes(node, result)
	}
	return result, rows.Err()
}

func collectSurfaceNodes(node capturedSurfaceNode, result map[string]string) {
	assetID := node.DOM.Attributes["data-rcl-asset"]
	if assetID != "" {
		if value := strings.TrimSpace(node.ComputedStyle["box-shadow"]); value != "" {
			if _, exists := result[assetID]; !exists {
				result[assetID] = value
			}
		}
	}
	for _, child := range node.Children {
		collectSurfaceNodes(child, result)
	}
}
