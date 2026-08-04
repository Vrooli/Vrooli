package reconcile

import (
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
var relativeImportRE = regexp.MustCompile(`(?:from\s+|import\s*)["'](\.[^"']+)["']`)
var comfortableGapUtilityRE = regexp.MustCompile(`(?s)comfortable\s*:\s*["'][^"']*gap-(?:[1-9][0-9]*(?:\.[0-9]+)?|\[[^]]+\])[^"]*["']`)
var positiveGapDeclarationRE = regexp.MustCompile(`(?i)gap\s*:\s*(?:[1-9][0-9]*(?:\.[0-9]+)?|0?\.[0-9]+|\[[^]]+\])`)

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
		return comfortableGapUtilityRE.MatchString(source) || positiveGapDeclarationRE.MatchString(source)
	case "state-contrast":
		// A state-contrast claim must be backed by an interaction-state rule
		// and a semantic surface/background token. A classless hover prop is
		// not sufficient evidence because it can disappear against the base
		// surface.
		return strings.Contains(source, "hover:") && strings.Contains(source, "bg-app-")
	case "size-parity":
		return strings.Contains(source, "sizeClasses") && strings.Contains(source, "min-h-") && strings.Contains(source, "min-w-")
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
	if !pathWithin(report.TargetPath, versionDir) {
		return "", fmt.Errorf("catalog reference %q escapes the scenario tree", ref)
	}
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return "", fmt.Errorf("read catalog version: %w", err)
	}
	var entry string
	for _, item := range entries {
		if item.IsDir() || item.Name() == "story.tsx" || !strings.HasSuffix(item.Name(), ".tsx") {
			continue
		}
		entry = filepath.Join(versionDir, item.Name())
		break
	}
	if entry == "" {
		return "", fmt.Errorf("catalog version has no TSX source")
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

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
