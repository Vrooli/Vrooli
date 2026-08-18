package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ValidateConformance reports design-system drift separately from defect
// floors. These are deliberately corpus-level measures: they describe the
// shape of the workbench's authored styling and do not claim that a raw value
// is a layout defect by itself.
func ValidateConformance(root string) (Result, error) {
	uiRoot := filepath.Join(root, "scenarios", "react-component-library", "ui", "src")
	if _, statErr := os.Stat(uiRoot); os.IsNotExist(statErr) {
		return nonEmpty(Result{}, "conformance"), nil
	} else if statErr != nil {
		return Result{}, statErr
	}
	var paths []string
	err := filepath.WalkDir(uiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".ts" || ext == ".tsx" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	sort.Strings(paths)
	result := Result{}
	counts := map[string]int{}
	first := map[string]Finding{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, err
		}
		result.Inspected++
		text := string(data)
		for _, match := range typeScalePattern.FindAllStringIndex(text, -1) {
			counts["type-scale"]++
			if _, ok := first["type-scale"]; !ok {
				first["type-scale"] = conformanceFinding(root, path, data, "conformance.type-scale", "text scale is concentrated in tiny labels instead of the published semantic type ramp")
			}
			_ = match
		}
		for _, match := range largeTypeScalePattern.FindAllStringIndex(text, -1) {
			counts["type-scale-large"]++
			_ = match
		}
		for _, match := range rawSpacingPattern.FindAllStringIndex(text, -1) {
			counts["ramp-adherence"]++
			if _, ok := first["ramp-adherence"]; !ok {
				first["ramp-adherence"] = conformanceFinding(root, path, data, "conformance.ramp-adherence", "spacing uses a raw utility instead of the published ramp")
			}
			_ = match
		}
		for _, match := range rawIconPattern.FindAllStringIndex(text, -1) {
			counts["icon-scale"]++
			if _, ok := first["icon-scale"]; !ok {
				first["icon-scale"] = conformanceFinding(root, path, data, "conformance.icon-scale", "icon dimensions bypass the Icon primitive size scale")
			}
			_ = match
		}
		for _, match := range arbitrarySurfacePattern.FindAllStringIndex(text, -1) {
			if strings.Contains(text[match[0]:match[1]], "var(--elev-") {
				continue
			}
			counts["elevation-radius"]++
			if _, ok := first["elevation-radius"]; !ok {
				first["elevation-radius"] = conformanceFinding(root, path, data, "conformance.elevation-radius", "surface radius or elevation is an arbitrary value instead of a declared role")
			}
			_ = match
		}
	}
	// A type-scale finding is about the distribution, not every individual
	// token. It becomes actionable only when the small-label share dominates.
	if counts["type-scale"] == 0 || counts["type-scale"]*100 <= (counts["type-scale"]+counts["type-scale-large"])*80 {
		delete(first, "type-scale")
	}
	for _, key := range []string{"type-scale", "ramp-adherence", "icon-scale", "elevation-radius"} {
		if finding, ok := first[key]; ok {
			finding.Message = fmt.Sprintf("%s (%d observed instances)", finding.Message, counts[key])
			result.Findings = append(result.Findings, finding)
		}
	}
	return nonEmpty(result, "conformance"), nil
}

func conformanceFinding(root, path string, data []byte, code, message string) Finding {
	return Finding{
		Code: code, Category: "conformance", AssetID: "workbench.conformance",
		File: repoRel(root, path), Line: lineOf(data, firstMatch(data, code)),
		Message:     message,
		Remediation: "Replace the local styling with the published semantic token, primitive API, or named elevation/radius role; keep a deliberate exception documented if no generalized asset can express it.",
		DocsRef:     "docs/concepts/ARCHITECTURE.md#design-tokens",
	}
}

func firstMatch(data []byte, code string) string {
	if strings.Contains(code, "type-scale") {
		return "text-xs"
	}
	if strings.Contains(code, "ramp-adherence") {
		return "gap-"
	}
	if strings.Contains(code, "icon-scale") {
		return "h-"
	}
	return "rounded-["
}

var (
	typeScalePattern        = regexp.MustCompile(`\btext-(?:xs|sm)\b`)
	largeTypeScalePattern   = regexp.MustCompile(`\btext-(?:base|lg|xl|2xl|display|title|heading|body)\b`)
	rawSpacingPattern       = regexp.MustCompile(`\b(?:p|px|py|pt|pr|pb|pl|m|mx|my|mt|mr|mb|ml|gap)-[0-9]+(?:\.[0-9]+)?\b`)
	rawIconPattern          = regexp.MustCompile(`\b[hw]-[1-9][0-9]*\b`)
	arbitrarySurfacePattern = regexp.MustCompile(`\b(?:rounded|shadow)-\[[^]]+\]`)
)
