package components

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/themes"
)

var (
	cssVariableReferenceRE   = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)`)
	cssVariableDeclarationRE = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:`)
	dynamicTokenPatternRE    = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)-\$\{`)
)

// ExtractRequiredTokens derives the external CSS custom-property contract for
// one version. A property is required when the version references it through
// var(--name) but does not declare --name anywhere in its own source files.
// TS/TSX template literals are scanned as source text because the library
// stores several stylesheets in those literals.
func ExtractRequiredTokens(files []ComponentVersionFile) []string {
	referenced := make(map[string]struct{})
	declared := make(map[string]struct{})
	dynamicPrefixes := make(map[string]struct{})
	for _, file := range files {
		for _, match := range dynamicTokenPatternRE.FindAllStringSubmatch(file.Content, -1) {
			if len(match) > 1 {
				dynamicPrefixes[match[1]+"-"] = struct{}{}
			}
		}
		for _, match := range cssVariableReferenceRE.FindAllStringSubmatch(file.Content, -1) {
			if len(match) > 1 {
				referenced[match[1]] = struct{}{}
			}
		}
		for _, match := range cssVariableDeclarationRE.FindAllStringSubmatch(file.Content, -1) {
			if len(match) > 1 {
				declared[match[1]] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(referenced))
	for property := range referenced {
		_, dynamic := dynamicPrefixes[property]
		if _, defined := declared[property]; !defined && !dynamic {
			result = append(result, property)
		}
	}
	sort.Strings(result)
	return result
}

// ExtractRequiredTokenPatterns derives dynamic CSS custom-property families
// such as --space-* from a version's source. The concrete suffix is selected
// at runtime, so it cannot be represented by RequiredTokens alone.
func ExtractRequiredTokenPatterns(files []ComponentVersionFile) []string {
	patterns := make(map[string]struct{})
	for _, file := range files {
		for _, match := range dynamicTokenPatternRE.FindAllStringSubmatch(file.Content, -1) {
			if len(match) == 2 {
				patterns[match[1]+"-*"] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(patterns))
	for pattern := range patterns {
		result = append(result, pattern)
	}
	sort.Strings(result)
	return result
}

type KitCompatibilityVerdict string

const (
	KitCompatibilityUniversal           KitCompatibilityVerdict = "universal"
	KitCompatibilityRestricted          KitCompatibilityVerdict = "restricted"
	KitCompatibilityUnsatisfiable       KitCompatibilityVerdict = "unsatisfiable"
	KitCompatibilityUndefinedVocabulary KitCompatibilityVerdict = "undefined-vocabulary"
)

type ComponentKitCompatibility struct {
	Verdict               KitCompatibilityVerdict
	CompatibleKitIDs      []string
	UnsatisfiedProperties []string
}

// DeriveKitCompatibility is the single compatibility rule shared by indexing,
// census, and gates. Host runtime properties are intentionally outside the
// design-kit contract.
func DeriveKitCompatibility(required []string, kits map[string][]string) ComponentKitCompatibility {
	kitIDs := make([]string, 0, len(kits))
	publishedAny := map[string]bool{}
	sets := make(map[string]map[string]bool, len(kits))
	for kitID, properties := range kits {
		kitIDs = append(kitIDs, kitID)
		set := make(map[string]bool, len(properties))
		for _, property := range properties {
			set[property] = true
			publishedAny[property] = true
		}
		sets[kitID] = set
	}
	sort.Strings(kitIDs)
	filtered := make([]string, 0, len(required))
	seen := map[string]bool{}
	for _, property := range required {
		if strings.HasPrefix(property, "--rcl-") || seen[property] {
			continue
		}
		seen[property] = true
		filtered = append(filtered, property)
	}
	sort.Strings(filtered)
	result := ComponentKitCompatibility{}
	for _, property := range filtered {
		if !publishedAny[property] {
			result.UnsatisfiedProperties = append(result.UnsatisfiedProperties, property)
		}
	}
	if len(result.UnsatisfiedProperties) > 0 {
		result.Verdict = KitCompatibilityUndefinedVocabulary
		return result
	}
	for _, kitID := range kitIDs {
		complete := true
		for _, property := range filtered {
			if !sets[kitID][property] {
				complete = false
				break
			}
		}
		if complete {
			result.CompatibleKitIDs = append(result.CompatibleKitIDs, kitID)
		}
	}
	switch {
	case len(result.CompatibleKitIDs) == len(kitIDs):
		result.Verdict = KitCompatibilityUniversal
	case len(result.CompatibleKitIDs) > 0:
		result.Verdict = KitCompatibilityRestricted
	default:
		result.Verdict = KitCompatibilityUnsatisfiable
	}
	return result
}

func loadKitTokenVocabularies(repoRoot string) (map[string][]string, error) {
	metadata, err := filepath.Glob(filepath.Join(repoRoot, "templates", "design", "*", "metadata.json"))
	if err != nil {
		return nil, err
	}
	if len(metadata) == 0 {
		return nil, nil
	}
	result := make(map[string][]string, len(metadata))
	for _, path := range metadata {
		kitID := filepath.Base(filepath.Dir(path))
		tokens, resolveErr := themes.ResolveKitTokens(repoRoot, kitID)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve design kit %s: %w", kitID, resolveErr)
		}
		properties := make([]string, 0, len(tokens))
		for _, token := range tokens {
			properties = append(properties, token.Name)
		}
		result[kitID] = properties
	}
	return result, nil
}

func repositoryRootFromLibrary(libraryRoot string) string {
	root := filepath.Clean(filepath.Join(libraryRoot, "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "templates", "design", "_base", "tokens.css")); err != nil {
		return ""
	}
	return root
}
