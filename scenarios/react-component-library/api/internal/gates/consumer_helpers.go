// Package gates contains deterministic, browser-free catalog gate runners.
// Runners return findings for authored/implementation defects; they only return
// an error when their inputs cannot be read.
package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/packages/react-component-library/libspec"
	"react-component-library/internal/librarywalk"
)

func scenarioConsumerPins(root string) ([]consumerPin, error) {
	byKey := map[string]*consumerPin{}
	scenariosRoot := filepath.Join(root, "scenarios")
	err := librarywalk.Walk(scenariosRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".retired" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(scenariosRoot, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 4 || parts[1] != "ui" || parts[2] != "src" {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, specifier := range libspec.ParseAll(string(raw)) {
			if specifier.Selector == "" {
				continue
			}
			key := specifier.Name + "@" + specifier.Selector
			pin := byKey[key]
			if pin == nil {
				pin = &consumerPin{Asset: specifier.Name, Version: specifier.Selector, Scenarios: map[string]bool{}}
				byKey[key] = pin
			}
			pin.Scenarios[parts[0]] = true
			pin.Files = appendUnique(pin.Files, repoRel(root, path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]consumerPin, 0, len(keys))
	for _, key := range keys {
		result = append(result, *byKey[key])
	}
	return result, nil
}

func resolveConsumerPinVersion(manifest consumerPinManifest, pin string) (string, string) {
	versionsRoot := filepath.Join(manifest.Root, "versions")
	if strings.Count(pin, ".") == 0 {
		entries, _ := os.ReadDir(versionsRoot)
		var candidates []string
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), pin+".") {
				candidates = append(candidates, entry.Name())
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return semverParts(candidates[i]) > semverParts(candidates[j]) })
		if len(candidates) == 0 {
			return "", ""
		}
		pin = candidates[0]
	}
	path := filepath.Join(versionsRoot, pin)
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return "", ""
	}
	return pin, path
}

func semverParts(version string) int64 {
	parts := strings.Split(version, ".")
	var value int64
	for index := 0; index < 3; index++ {
		value *= 1_000_000
		if index < len(parts) {
			number, _ := strconv.ParseInt(parts[index], 10, 64)
			value += number
		}
	}
	return value
}

func versionMajor(version string) int {
	major, _ := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
	return major
}

func sortedStringMapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func allLibrarySources(scope Scope) ([]string, error) {
	root := scope.Root
	var sources []string
	err := librarywalk.Walk(filepath.Join(root, "scenarios", "react-component-library", "library"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".retired" {
				return filepath.SkipDir
			}
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(path)); ext == ".ts" || ext == ".tsx" {
			if !sourceInScope(root, path, scope) {
				return nil
			}
			sources = append(sources, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(sources)
	return sources, nil
}

func deprecatedLibraryVersions(root string) (map[string][]string, error) {
	result := map[string][]string{}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			var doc struct {
				LibraryID          string   `json:"libraryId"`
				CatalogID          string   `json:"catalogId"`
				DeprecatedVersions []string `json:"deprecatedVersions"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return nil, err
			}
			for _, key := range []string{filepath.Base(filepath.Dir(path)), strings.TrimPrefix(doc.LibraryID, "react-component-library:"), strings.TrimPrefix(doc.CatalogID, "react-component-library:")} {
				if key != "" {
					result[key] = append(result[key], doc.DeprecatedVersions...)
				}
			}
		}
	}
	return result, nil
}

// ValidateProvenanceStamp ensures a source marker describes a real adoption
// edge. Library files are checked against their owning manifest; UI files must
// import or render the stamped asset instead of copying a stale label.
func libraryManifestIdentities(sourcePath string) (string, string) {
	// sourcePath is .../<asset>/versions/<version>/<entry>.tsx. The owning
	// manifest is three directories above the source: <asset>/component.json.
	assetPath := filepath.Dir(filepath.Dir(filepath.Dir(sourcePath)))
	manifestPath := filepath.Join(assetPath, "component.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", ""
	}
	var doc struct {
		LibraryID string `json:"libraryId"`
		CatalogID string `json:"catalogId"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return "", ""
	}
	return doc.LibraryID, doc.CatalogID
}

var (
	stylePropRE          = regexp.MustCompile(`(?m)(?:^|[;{]\s*)style\?\s*:\s*([^;},\n]+)`)
	classNamePropRE      = regexp.MustCompile(`(?m)\bclassName\??\s*:`)
	classNameUseRE       = regexp.MustCompile(`\bclassName\b`)
	forwardRefRE         = regexp.MustCompile(`\bforwardRef\b`)
	refAttributeRE       = regexp.MustCompile(`\bref\s*=\s*\{[^}]*\b(?:ref|forwardedRef)\b[^}]*\}`)
	imperativeRefRE      = regexp.MustCompile(`\buseImperativeHandle\s*\(`)
	assignRefRE          = regexp.MustCompile(`\bassignRef\s*\(`)
	classNameBoundaryRE  = regexp.MustCompile(`\bwithClassName\s*\(`)
	exportedComponentRE  = regexp.MustCompile(`(?m)(?:^|[;\n])\s*export\s+(?:function|const|class)\s+[A-Z]`)
	jsxIdentifierStyleRE = regexp.MustCompile(`style\s*=\s*\{\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*\}`)
	moduleObjectRE       = regexp.MustCompile(`(?m)(?:^|[;\n])\s*(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*(?:[:][^=\n]+)?=\s*\{`)
)

func analyzeRestyleSource(source string) defect {
	if !exportedComponentRE.MatchString(source) {
		return ok()
	}
	hasClassName := classNamePropRE.MatchString(source) || classNameUseRE.MatchString(source) || strings.Contains(source, "HTMLAttributes<") || strings.Contains(source, "ComponentProps<") || classNameBoundaryRE.MatchString(source)
	if hasOverloadedStyleProp(source) && hasClassName {
		return defect{
			Message:     "overloads the standard style prop while exposing className",
			Remediation: "Remove the bespoke style prop; expose a named token selector and reserve style for consumer-owned computed values.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#restyle-contract",
		}
	}
	if hasClassName && hasInlineStyleOverride(source) && !strings.Contains(source, "/* computed-style-ok */") {
		return defect{
			Message:     "accepts className but also assigns an inline style object",
			Remediation: "Move token-derived declarations to a co-located stylesheet keyed by data attributes, then compose className with cn() on the root.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#restyle-contract",
		}
	}
	hasForwardedRef := forwardRefRE.MatchString(source) && refAttributeRE.MatchString(source)
	if classNameBoundaryRE.MatchString(source) || imperativeRefRE.MatchString(source) || assignRefRE.MatchString(source) {
		hasForwardedRef = true
	}
	if hasClassName && !hasForwardedRef {
		return defect{
			Message:     "does not forward a consumer ref to its root element",
			Remediation: "Wrap the exported component in forwardRef and pass ref to the outermost rendered element.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#restyle-contract",
		}
	}
	if !hasClassName {
		return defect{
			Message:     "does not expose a className pass-through on its public component surface",
			Remediation: "Add className?: string to the exported props, merge it with cn(), and forward it to the outermost rendered element.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#restyle-contract",
		}
	}
	return ok()
}

func hasOverloadedStyleProp(source string) bool {
	for _, match := range stylePropRE.FindAllStringSubmatch(source, -1) {
		if len(match) < 2 {
			continue
		}
		typeName := strings.TrimSpace(match[1])
		if typeName != "CSSProperties" && typeName != "React.CSSProperties" {
			return true
		}
	}
	return false
}

// hasInlineStyleOverride only considers one JSX opening element at a time.
// A component can legitimately use a computed inline value on a nested
// layout child while still exposing a className-controlled root. Matching the
// whole source would incorrectly reject that case (and even match
// data-* attributes such as data-text-style).
func hasInlineStyleOverride(source string) bool {
	objects := map[string]bool{}
	for _, match := range moduleObjectRE.FindAllStringSubmatch(source, -1) {
		if len(match) > 1 {
			objects[match[1]] = true
		}
	}
	for _, tag := range jsxOpeningTags(source) {
		inlineObject := jsxInlineStyleObjectRE.MatchString(tag)
		if !inlineObject {
			match := jsxIdentifierStyleRE.FindStringSubmatch(tag)
			inlineObject = len(match) > 1 && objects[match[1]]
		}
		if !inlineObject {
			continue
		}
		if jsxConsumerClassRE.MatchString(tag) || jsxSpreadRE.MatchString(tag) {
			return true
		}
	}
	// Generic TypeScript syntax can look like a JSX opening tag to a small
	// scanner (for example forwardRef<HTMLDivElement, Props>). Preserve the
	// same root-class requirement while covering that valid source shape.
	for _, match := range jsxIdentifierStyleRE.FindAllStringSubmatch(source, -1) {
		if len(match) > 1 && (objects[match[1]] || strings.Contains(source, "const "+match[1]+" = {") || strings.Contains(source, "let "+match[1]+" = {") || strings.Contains(source, "var "+match[1]+" = {")) && strings.Contains(source, "className={className}") && strings.Contains(source, "return") {
			return true
		}
	}
	return false
}

var (
	jsxSpreadRE            = regexp.MustCompile(`\.\.\.(?:props|rest)\b`)
	jsxConsumerClassRE     = regexp.MustCompile(`(?m)(?:^|\s)className\s*=\s*\{[^}]*\bclassName\b`)
	jsxInlineStyleObjectRE = regexp.MustCompile(`(?m)(?:^|\s)style\s*=\s*\{\s*\{`)
)
