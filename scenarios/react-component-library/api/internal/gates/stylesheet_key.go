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

	"react-component-library/internal/librarywalk"
)

var (
	styleSheetLiteral    = regexp.MustCompile(`(?s)<StyleSheet\b[^>]*?\bname\s*=`)
	legacyStyleSheetHook = regexp.MustCompile(`useLibraryStyleSheet\s*\(\s*["'][^"']+["']\s*,`)
	styleSheetVersionArg = regexp.MustCompile(`^\s*["']\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?["']\s*,`)
	styleSheetLibraryID  = regexp.MustCompile(`@libraryId\s+([^\s*]+)`)
	styleSheetVersion    = regexp.MustCompile(`@version\s+([^\s*]+)`)
)

type stylesheetSource struct {
	path       string
	libraryID  string
	version    string
	assetID    string
	versionDir string
}

// ValidateStylesheetKey rejects the legacy literal key at each injection site.
// The identity must come from the source header (or its manifest for older
// adopted files), so the runtime key cannot drift from the published asset.
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
		for _, match := range styleSheetLiteral.FindAllStringIndex(text, -1) {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.stylesheet-key", AssetID: source.assetID, File: repoRel(scope.Root, source.path), Line: lineAt(data, match[0]),
				Message:     "StyleSheet still passes a literal name instead of the exact asset identity",
				Remediation: fmt.Sprintf("Replace the literal name with libraryId=%q and version=%q, or use the stylesheet identity codemod.", source.libraryID, source.version),
				DocsRef:     "docs/reference/style-ownership.md",
			})
		}
		for _, match := range legacyStyleSheetHook.FindAllStringIndex(text, -1) {
			if styleSheetVersionArg.MatchString(text[match[1]:]) {
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.stylesheet-key", AssetID: source.assetID, File: repoRel(scope.Root, source.path), Line: lineAt(data, match[0]),
				Message:     "useLibraryStyleSheet still passes a legacy literal key",
				Remediation: fmt.Sprintf("Pass libraryId %q and version %q as the first two arguments.", source.libraryID, source.version),
				DocsRef:     "docs/reference/style-ownership.md",
			})
		}
	}
	return nonEmpty(result, "stylesheet-key"), nil
}

// ValidateStylesheetKeyUniqueness proves that two reachable version
// directories cannot compute the same page-global stylesheet key.
func ValidateStylesheetKeyUniqueness(scope Scope) (Result, error) {
	sources, err := stylesheetSources(scope)
	if err != nil {
		return Result{}, err
	}
	byKey := map[string][]stylesheetSource{}
	for _, source := range sources {
		key := libraryStyleSheetKey(source.libraryID, source.version)
		byKey[key] = append(byKey[key], source)
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
		for _, entry := range entries {
			versionDirs[entry.versionDir] = true
		}
		if len(versionDirs) < 2 {
			continue
		}
		paths := make([]string, 0, len(versionDirs))
		for versionDir := range versionDirs {
			paths = append(paths, repoRel(scope.Root, versionDir))
		}
		sort.Strings(paths)
		for _, entry := range entries {
			result.Findings = append(result.Findings, Finding{
				Code: "catalog.stylesheet-key-duplicate", AssetID: entry.assetID, File: repoRel(scope.Root, entry.versionDir),
				Message:     fmt.Sprintf("computed stylesheet key %q is shared by version directories: %s", key, strings.Join(paths, ", ")),
				Remediation: "Keep the asset libraryId and exact version from the source header so every version computes a unique stylesheet key.",
				DocsRef:     "docs/reference/style-ownership.md",
			})
		}
	}
	return nonEmpty(result, "stylesheet-key-duplicate"), nil
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
		assetID, err := manifestIdentity(assetDir)
		if err != nil {
			return err
		}
		sources = append(sources, stylesheetSource{path: path, libraryID: libraryID, version: version, assetID: assetID, versionDir: filepath.Dir(path)})
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

func manifestIdentity(assetDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(assetDir, "component.json"))
	if err != nil {
		return "", err
	}
	var manifest struct {
		CatalogID string `json:"catalogId"`
		LibraryID string `json:"libraryId"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", err
	}
	if manifest.CatalogID != "" {
		return manifest.CatalogID, nil
	}
	return manifest.LibraryID, nil
}

func libraryStyleSheetKey(libraryID, version string) string {
	normalized := strings.ToLower(strings.TrimSpace(libraryID))
	normalized = regexp.MustCompile(`[^a-z0-9_-]+`).ReplaceAllString(normalized, "-")
	normalized = regexp.MustCompile(`-+`).ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	return normalized + "-" + strings.TrimSpace(version)
}
