package reconcile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"experience-manager/internal/spec"
)

// Component source checks cover the small set of experience properties that
// an accessibility tree cannot carry: CSS spacing and interaction treatment.
// They are intentionally narrow, tied to typed claims, and supplementary to
// live BAS reconciliation. A source check never claims that a browser render
// passed; it only prevents a known contract from becoming ungrounded when the
// capture producer omits computed styles.
var (
	relativeImportRE         = regexp.MustCompile(`(?:from\s+|import\s*)["'](\.[^"']+)["']`)
	libraryImportRE          = regexp.MustCompile(`(?:from\s+|import\s*)["']@vrooli/react-component-library/([A-Za-z0-9]+)/([0-9]+)["']`)
	comfortableGapUtilityRE  = regexp.MustCompile(`(?s)comfortable\s*:\s*["'][^"']*gap-(?:[1-9][0-9]*(?:\.[0-9]+)?|\[[^]]+\])[^"]*["']`)
	positiveGapDeclarationRE = regexp.MustCompile(`(?i)gap\s*:\s*(?:[1-9][0-9]*(?:\.[0-9]+)?|0?\.[0-9]+|\[[^]]+\])`)
	semanticGapDeclarationRE = regexp.MustCompile(`(?is)gap\s*:\s*[^;{}]*var\(--space-[^)]+\)`)
)

func componentSourceFindings(report spec.Report, loc string, component spec.ComponentDocument) []spec.Finding {
	claims := make([]spec.Claim, 0)
	for _, claim := range component.Claims {
		if claim.Tier == "machine" && (claim.Type == "spacing" || claim.Type == "state-contrast" || claim.Type == "size-parity") {
			claims = append(claims, claim)
		}
	}
	if len(claims) == 0 {
		return nil
	}
	source, err := readComponentSourceGraph(report, component)
	if err != nil {
		return []spec.Finding{{
			Code:       spec.CodeClaimUnverifiable,
			Severity:   spec.SeverityWarning,
			Message:    fmt.Sprintf("component source claims cannot be grounded: %v", err),
			Locations:  []string{loc},
			Suggestion: "Restore the catalog story/source reference or add a capture producer that exposes the required computed evidence.",
		}}
	}
	var findings []spec.Finding
	for _, claim := range claims {
		if sourceClaimPasses(claim, source) {
			continue
		}
		findings = append(findings, spec.Finding{
			Code:       spec.CodeClaimFailed,
			Severity:   spec.SeverityError,
			Message:    fmt.Sprintf("component claim %q failed source-level contract proof", claim.ID),
			Locations:  []string{loc},
			Suggestion: "Restore the declared spacing, interaction treatment, or size contract before promoting the component.",
		})
	}
	return findings
}

func sourceClaimPasses(claim spec.Claim, source string) bool {
	switch claim.Type {
	case "spacing":
		// Tailwind's zero gap is not a valid proof. Requiring a positive gap
		// utility catches the exact icon/text edge regression while accepting
		// arbitrary positive spacing declarations used by catalog authors.
		return comfortableGapUtilityRE.MatchString(source) || positiveGapDeclarationRE.MatchString(source) || semanticGapDeclarationRE.MatchString(source)
	case "state-contrast":
		// A state-contrast claim must be backed by an interaction-state rule
		// and a semantic surface/background token. A classless hover prop is
		// not sufficient evidence because it can disappear against the base
		// surface. Native in-house sources use CSS custom properties while
		// older adopted sources use the app-* utility vocabulary; both are
		// valid semantic proofs.
		hasSemanticSurface := strings.Contains(source, "bg-app-") ||
			(strings.Contains(source, "background") && strings.Contains(source, "--color-"))
		hasInteraction := strings.Contains(source, "hover:") || strings.Contains(source, ":hover")
		return hasInteraction && hasSemanticSurface
	case "size-parity":
		legacySizeScale := strings.Contains(source, "sizeClasses") && strings.Contains(source, "min-h-") && strings.Contains(source, "min-w-")
		semanticSizeScale := strings.Contains(source, "sizeStyles") && strings.Contains(source, "min-height: var(--tap-target-min)") && strings.Contains(source, "min-width: var(--tap-target-min)") && strings.Contains(source, "touch-target")
		return legacySizeScale || semanticSizeScale
	default:
		return true
	}
}

func readComponentSourceGraph(report spec.Report, component spec.ComponentDocument) (string, error) {
	ref := strings.TrimSpace(component.Component.StoryRef)
	if ref == "" {
		ref = strings.TrimSpace(component.Component.ExamplesRef)
	}
	if ref == "" {
		return "", fmt.Errorf("component has no catalog story or examples reference")
	}
	refPath := filepath.Clean(filepath.Join(report.Spec.ExperienceDir, "components", filepath.FromSlash(ref)))
	versionDir := filepath.Dir(refPath)
	if !catalogPathWithinScenarioOrRCL(report.TargetPath, versionDir) {
		return "", fmt.Errorf("catalog reference %q escapes the scenario tree", ref)
	}
	entry, err := versionEntrySource(versionDir)
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	var parts []string
	var visit func(string) error
	visit = func(path string) error {
		path = filepath.Clean(path)
		if seen[path] {
			return nil
		}
		seen[path] = true
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		parts = append(parts, string(body))
		// A release composes other library components through major-version
		// imports; their treatment (hover rules, surfaces) is part of what the
		// release renders, so follow each to the exact version it pinned.
		for _, match := range libraryImportRE.FindAllStringSubmatch(string(body), -1) {
			dependency, ok, err := resolveLibraryImport(filepath.Dir(path), match[1], match[2])
			if err != nil {
				return err
			}
			if !ok || !catalogPathWithinScenarioOrRCL(report.TargetPath, dependency) {
				continue
			}
			if err := visit(dependency); err != nil {
				return err
			}
		}
		for _, match := range relativeImportRE.FindAllStringSubmatch(string(body), -1) {
			if len(match) < 2 {
				continue
			}
			candidate := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(match[1])))
			for _, suffix := range []string{"", ".tsx", ".ts"} {
				resolved := candidate + suffix
				if _, statErr := os.Stat(resolved); statErr == nil {
					if err := visit(resolved); err != nil {
						return err
					}
					break
				}
			}
		}
		return nil
	}
	if err := visit(entry); err != nil {
		return "", fmt.Errorf("read catalog source graph: %w", err)
	}
	return strings.Join(parts, "\n"), nil
}

// versionEntrySource returns a catalog release's component source: its one
// non-story TSX file.
func versionEntrySource(versionDir string) (string, error) {
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return "", fmt.Errorf("read catalog version: %w", err)
	}
	for _, item := range entries {
		if item.IsDir() || item.Name() == "story.tsx" || !strings.HasSuffix(item.Name(), ".tsx") {
			continue
		}
		return filepath.Join(versionDir, item.Name()), nil
	}
	return "", fmt.Errorf("catalog version has no TSX source")
}

// resolveLibraryImport maps `@vrooli/react-component-library/<name>/<major>`
// from a release directory to the source of the version that release pinned in
// its dependencies.json. A directory without a pin (a scenario-local source)
// or an unpinned dependency resolves to nothing.
func resolveLibraryImport(versionDir, name, major string) (string, bool, error) {
	raw, err := os.ReadFile(filepath.Join(versionDir, "dependencies.json"))
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read catalog dependencies: %w", err)
	}
	var pins struct {
		Dependencies []struct {
			LibraryID string `json:"libraryId"`
			Major     int    `json:"major"`
			Observed  string `json:"observed"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(raw, &pins); err != nil {
		return "", false, fmt.Errorf("parse catalog dependencies %s: %w", versionDir, err)
	}
	for _, pin := range pins.Dependencies {
		if pin.LibraryID != "react-component-library:"+name || fmt.Sprint(pin.Major) != major || pin.Observed == "" {
			continue
		}
		componentsDir := filepath.Dir(filepath.Dir(filepath.Dir(versionDir)))
		entry, err := versionEntrySource(filepath.Join(componentsDir, name, "versions", pin.Observed))
		if err != nil {
			return "", false, fmt.Errorf("resolve %s/%s: %w", name, major, err)
		}
		return entry, true, nil
	}
	return "", false, nil
}

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func catalogPathWithinScenarioOrRCL(scenarioDir, candidate string) bool {
	if pathWithin(scenarioDir, candidate) {
		return true
	}
	return pathWithin(filepath.Join(filepath.Dir(scenarioDir), "react-component-library"), candidate)
}
