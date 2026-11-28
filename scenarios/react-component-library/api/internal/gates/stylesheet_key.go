package gates

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/librarywalk"
)

var (
	styleSheetLiteral    = regexp.MustCompile(`(?s)<StyleSheet\b[^>]*?\bname\s*=`)
	legacyStyleSheetHook = regexp.MustCompile(`useLibraryStyleSheet\s*\(\s*["'][^"']+["']\s*,`)
	styleSheetVersionArg = regexp.MustCompile(`^\s*["']\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?["']\s*,`)
	styleSheetLibraryID  = regexp.MustCompile(`@libraryId\s+([^\s*]+)`)
	styleSheetVersion    = regexp.MustCompile(`@version\s+([^\s*]+)`)
	styleSheetReleaseDir = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

type stylesheetSource struct {
	path       string
	libraryID  string
	version    string
	assetID    string
	versionDir string
	// released marks immutable history: a release-shaped version of an asset
	// that has published a release, other than its active draft. Everything
	// else (the draft, or an asset with no release yet) can still change and
	// is judged.
	released bool
	// current marks the version a consumer gets today: the latest release or
	// the active draft (or any version of an asset with no release yet).
	current bool
}

// ValidateStylesheetKey rejects the legacy literal key at each injection site.
// The identity must come from the source header (or its manifest for older
// adopted files), so the runtime key cannot drift from the published asset.
//
// Only the active draft is judged: it is the one version that can still
// change. Released versions are immutable history, so their legacy keys are
// reported as information instead of failing the gate on bytes nobody may
// edit; a new release fixes them, and the store stamps every version it writes.
func ValidateStylesheetKey(scope Scope) (Result, error) {
	sources, err := stylesheetSources(scope)
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: len(sources)}
	for _, source := range sources {
		data, err := os.ReadFile(source.path)
		if err != nil {
			return Result{}, err
		}
		text := string(data)
		var findings []Finding
		for _, match := range styleSheetLiteral.FindAllStringIndex(text, -1) {
			findings = append(findings, Finding{
				Code: "catalog.stylesheet-key", AssetID: source.assetID, File: repoRel(scope.Root, source.path), Line: lineAt(data, match[0]),
				Message:     "StyleSheet still passes a literal name instead of the exact asset identity",
				Remediation: fmt.Sprintf("Replace the literal name with libraryId=%q and version=%q, or use the stylesheet identity codemod.", source.libraryID, source.version),
				DocsRef:     "docs/reference/style-ownership.md",
			})
		}
		// The key's version literal must be this version's own. A draft copied
		// from its release keeps the release's literal unless the store
		// rewrites it, and then two versions share one page-global key with
		// different CSS (FullPageDrawer 1.4.5-1.5.0 all used 1.4.3).
		for _, literal := range components.SelfStyleSheetVersions(text, source.libraryID) {
			if literal.Version == source.version {
				continue
			}
			findings = append(findings, Finding{
				Code: "catalog.stylesheet-key-version", AssetID: source.assetID, File: repoRel(scope.Root, source.path), Line: lineAt(data, literal.Offset),
				Message:     fmt.Sprintf("stylesheet key passes version %q but this version is %q", literal.Version, source.version),
				Remediation: fmt.Sprintf("Pass version %q; draft-begin and draft-publish stamp it automatically.", source.version),
				DocsRef:     "docs/reference/style-ownership.md",
			})
		}
		for _, match := range legacyStyleSheetHook.FindAllStringIndex(text, -1) {
			if styleSheetVersionArg.MatchString(text[match[1]:]) {
				continue
			}
			findings = append(findings, Finding{
				Code: "catalog.stylesheet-key", AssetID: source.assetID, File: repoRel(scope.Root, source.path), Line: lineAt(data, match[0]),
				Message:     "useLibraryStyleSheet still passes a legacy literal key",
				Remediation: fmt.Sprintf("Pass libraryId %q and version %q as the first two arguments.", source.libraryID, source.version),
				DocsRef:     "docs/reference/style-ownership.md",
			})
		}
		if source.released {
			result.InformationalFindings = append(result.InformationalFindings, findings...)
		} else {
			result.Findings = append(result.Findings, findings...)
		}
	}
	return nonEmpty(result, "stylesheet-key"), nil
}

// ValidateStylesheetKeyUniqueness fails when the version a consumer gets today
// (the latest release or the active draft) injects different CSS under the
// same page-global stylesheet key as another version: whichever mounts first
// wins and the other renders with the wrong styles. The key is computed from
// the version literal the source actually passes, not from its header,
// because copied drafts used to keep their release's literal.
//
// A key shared by versions whose CSS is identical breaks nothing and is
// information, as is a shared key whose CSS cannot be read statically. A
// collision confined to older releases is information too: released history
// is immutable, and once the latest keys by its own version (the store stamps
// it on release) nothing current can collide.
func ValidateStylesheetKeyUniqueness(scope Scope) (Result, error) {
	sources, err := stylesheetSources(scope)
	if err != nil {
		return Result{}, err
	}
	type injection struct {
		source stylesheetSource
		css    string
		known  bool
	}
	byKey := map[string][]injection{}
	siblings := map[string][]string{}
	for _, source := range sources {
		data, err := os.ReadFile(source.path)
		if err != nil {
			return Result{}, err
		}
		text := string(data)
		literals := components.SelfStyleSheetVersions(text, source.libraryID)
		if len(literals) == 0 {
			// A source that keys by header identity or a variable keeps the
			// header version; it can still collide with a copied literal.
			key := libraryStyleSheetKey(source.libraryID, source.version)
			byKey[key] = append(byKey[key], injection{source: source})
			continue
		}
		for _, literal := range literals {
			if _, ok := siblings[source.versionDir]; !ok {
				siblings[source.versionDir] = versionSourceTexts(source.versionDir)
			}
			css, known := components.StyleSheetConstText(literal.CSSName, append([]string{text}, siblings[source.versionDir]...)...)
			key := libraryStyleSheetKey(source.libraryID, literal.Version)
			byKey[key] = append(byKey[key], injection{source: source, css: strings.Join(strings.Fields(css), " "), known: known})
		}
	}
	result := Result{Inspected: len(sources)}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entries := byKey[key]
		versionDirs := map[string]bool{}
		cssByDir := map[string]string{}
		unknown := false
		current := false
		for _, entry := range entries {
			versionDirs[entry.source.versionDir] = true
			current = current || entry.source.current
			if !entry.known {
				unknown = true
				continue
			}
			cssByDir[entry.source.versionDir] = entry.css
		}
		if len(versionDirs) < 2 {
			continue
		}
		distinct := map[string]bool{}
		for _, css := range cssByDir {
			distinct[css] = true
		}
		paths := make([]string, 0, len(versionDirs))
		for versionDir := range versionDirs {
			paths = append(paths, repoRel(scope.Root, versionDir))
		}
		sort.Strings(paths)
		collision := len(distinct) > 1
		for _, entry := range entries {
			finding := Finding{
				Code: "catalog.stylesheet-key-duplicate", AssetID: entry.source.assetID, File: repoRel(scope.Root, entry.source.versionDir),
				DocsRef: "docs/reference/style-ownership.md",
			}
			switch {
			case collision && current:
				finding.Message = fmt.Sprintf("stylesheet key %q injects different CSS from version directories: %s", key, strings.Join(paths, ", "))
				finding.Remediation = "Release a new version; the store stamps the release's own version into its stylesheet key."
				result.Findings = append(result.Findings, finding)
			case collision:
				finding.Code = "catalog.stylesheet-key-duplicate-history"
				finding.Message = fmt.Sprintf("older releases share stylesheet key %q with different CSS: %s", key, strings.Join(paths, ", "))
				finding.Remediation = "Nothing current collides; mounting two of these historical releases on one page would."
				result.InformationalFindings = append(result.InformationalFindings, finding)
			case unknown:
				finding.Code = "catalog.stylesheet-key-shared"
				finding.Message = fmt.Sprintf("stylesheet key %q is shared by version directories (%s); their CSS could not be compared statically", key, strings.Join(paths, ", "))
				finding.Remediation = "Nothing breaks unless the CSS differs; the next release keys by its own version."
				result.InformationalFindings = append(result.InformationalFindings, finding)
			default:
				finding.Code = "catalog.stylesheet-key-shared"
				finding.Message = fmt.Sprintf("stylesheet key %q is shared by version directories with identical CSS: %s", key, strings.Join(paths, ", "))
				finding.Remediation = "Nothing breaks; the next release keys by its own version."
				result.InformationalFindings = append(result.InformationalFindings, finding)
			}
		}
	}
	return nonEmpty(result, "stylesheet-key-duplicate"), nil
}

// versionSourceTexts reads every .ts/.tsx file of one version directory, so a
// CSS constant imported from a sibling module can be resolved.
func versionSourceTexts(versionDir string) []string {
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return nil
	}
	var texts []string
	for _, entry := range entries {
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if entry.IsDir() || (ext != ".ts" && ext != ".tsx") {
			continue
		}
		if data, err := os.ReadFile(filepath.Join(versionDir, entry.Name())); err == nil {
			texts = append(texts, string(data))
		}
	}
	return texts
}

func stylesheetSources(scope Scope) ([]stylesheetSource, error) {
	if scope.Context == nil {
		scope.Context = context.Background()
	}
	libraryRoot := filepath.Join(scope.Root, "scenarios", "react-component-library", "library")
	var sources []stylesheetSource
	err := librarywalk.WalkContext(scope.Context, libraryRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		slash := filepath.ToSlash(path)
		if entry.IsDir() {
			if entry.Name() == ".retired" || entry.Name() == "node_modules" || entry.Name() == "dist" || strings.Contains(slash, "/library/tests/") {
				return filepath.SkipDir
			}
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(path)); ext != ".ts" && ext != ".tsx" {
			return nil
		}
		if !strings.Contains(slash, "/versions/") || !sourceInScope(scope.Root, path, scope) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		if !strings.Contains(text, "<StyleSheet") && !strings.Contains(text, "useLibraryStyleSheet(") {
			return nil
		}
		libraryID, version, err := stylesheetIdentity(path, text)
		if err != nil {
			return err
		}
		assetDir := filepath.Dir(filepath.Dir(filepath.Dir(path)))
		if _, err := os.Stat(filepath.Join(assetDir, "component.json")); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		assetID, latest, draft, err := manifestIdentity(assetDir)
		if err != nil {
			return err
		}
		versionName := filepath.Base(filepath.Dir(path))
		sources = append(sources, stylesheetSource{
			path: path, libraryID: libraryID, version: version, assetID: assetID, versionDir: filepath.Dir(path),
			released: latest != "" && versionName != draft && styleSheetReleaseDir.MatchString(versionName),
			current:  latest == "" || versionName == latest || versionName == draft,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].path < sources[j].path })
	return sources, nil
}

func stylesheetIdentity(path, text string) (string, string, error) {
	libraryMatch := styleSheetLibraryID.FindStringSubmatch(text)
	versionMatch := styleSheetVersion.FindStringSubmatch(text)
	if len(libraryMatch) > 1 && len(versionMatch) > 1 {
		return libraryMatch[1], versionMatch[1], nil
	}
	assetDir := filepath.Dir(filepath.Dir(filepath.Dir(path)))
	libraryID, err := manifestLibraryID(assetDir)
	if err != nil {
		return "", "", err
	}
	return libraryID, filepath.Base(filepath.Dir(path)), nil
}

func manifestLibraryID(assetDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(assetDir, "component.json"))
	if err != nil {
		return "", err
	}
	var manifest struct {
		LibraryID string `json:"libraryId"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", err
	}
	if strings.TrimSpace(manifest.LibraryID) == "" {
		return "", fmt.Errorf("component manifest %s has no libraryId", assetDir)
	}
	return manifest.LibraryID, nil
}

// manifestIdentity returns the asset's finding identity, its latest release
// and its active draft version.
func manifestIdentity(assetDir string) (string, string, string, error) {
	data, err := os.ReadFile(filepath.Join(assetDir, "component.json"))
	if err != nil {
		return "", "", "", err
	}
	var manifest struct {
		CatalogID string `json:"catalogId"`
		LibraryID string `json:"libraryId"`
		Latest    string `json:"latest"`
		Draft     string `json:"draft"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", "", "", err
	}
	id := manifest.LibraryID
	if manifest.CatalogID != "" {
		id = manifest.CatalogID
	}
	return id, strings.TrimSpace(manifest.Latest), strings.TrimSpace(manifest.Draft), nil
}

func libraryStyleSheetKey(libraryID, version string) string {
	normalized := strings.ToLower(strings.TrimSpace(libraryID))
	normalized = regexp.MustCompile(`[^a-z0-9_-]+`).ReplaceAllString(normalized, "-")
	normalized = regexp.MustCompile(`-+`).ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	return normalized + "-" + strings.TrimSpace(version)
}
