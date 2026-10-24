package adoptions

import (
	"path"
	"strings"
)

// ScenarioPaths are the consumer-side files a governed adoption reads or
// writes. They are resolved from the target template's `ui/manifest.json`
// `files` declarations when the manifest loader is wired, and fall back to the
// react-vite layout every generated scenario has carried so far.
type ScenarioPaths struct {
	// TokenRamp is the design-token file carrying the rcl:tokens managed region.
	TokenRamp string
	// TokenRampBegin and TokenRampEnd delimit the managed region.
	TokenRampBegin string
	TokenRampEnd   string
	// LocaleCatalogue is the default-locale i18next catalogue library strings merge into.
	LocaleCatalogue string
	// SelectorRegistry is the scenario's typed selector registry.
	SelectorRegistry string
	// LibrarySelectors is the file the link writes derived selector ids into.
	LibrarySelectors string
	// AppEntry is the mount point that hosts the library strings provider.
	AppEntry string
}

const (
	defaultTokenRampPath        = "ui/src/design-tokens.css"
	defaultLocaleCataloguePath  = "ui/src/i18n/locales/en.json"
	defaultSelectorRegistryPath = "ui/src/consts/selectors.ts"
	defaultLibrarySelectorsPath = "ui/src/consts/selectors.library.ts"
	defaultAppEntryPath         = "ui/src/main.tsx"
)

// DefaultScenarioPaths is the layout assumed when a template declares nothing.
func DefaultScenarioPaths() ScenarioPaths {
	return ScenarioPaths{
		TokenRamp:        defaultTokenRampPath,
		TokenRampBegin:   tokenRampBegin,
		TokenRampEnd:     tokenRampEnd,
		LocaleCatalogue:  defaultLocaleCataloguePath,
		SelectorRegistry: defaultSelectorRegistryPath,
		LibrarySelectors: defaultLibrarySelectorsPath,
		AppEntry:         defaultAppEntryPath,
	}
}

// scenarioPaths resolves the consumer file layout for one scenario. A missing
// or undeclared manifest is not an error here: the defaults are the contract
// every pre-`files` template satisfied, and the obligations and gate checks
// report the outcome rather than this resolver.
func (s *service) scenarioPaths(scenario string) ScenarioPaths {
	paths := DefaultScenarioPaths()
	if s == nil || s.manifestLoader == nil {
		return paths
	}
	manifest, err := s.manifestLoader.Load(scenario)
	if err != nil {
		return paths
	}
	paths.TokenRamp = manifest.ResolveFile("designTokens", paths.TokenRamp)
	paths.TokenRampBegin, paths.TokenRampEnd = manifest.ManagedRegionMarkers("designTokens", paths.TokenRampBegin, paths.TokenRampEnd)
	paths.LocaleCatalogue = manifest.ResolveLocaleFile("localeCatalogue", "", paths.LocaleCatalogue)
	paths.SelectorRegistry = manifest.ResolveFile("selectorRegistry", paths.SelectorRegistry)
	paths.LibrarySelectors = manifest.ResolveFile("librarySelectors", paths.LibrarySelectors)
	paths.AppEntry = manifest.ResolveFile("appEntry", paths.AppEntry)
	return paths
}

// relativeImport returns the module specifier `from` uses to import `to`
// (both scenario-relative file paths), without the TypeScript extension.
func relativeImport(from, to string) string {
	fromDir := path.Dir(path.Clean(from))
	target := strings.TrimSuffix(strings.TrimSuffix(path.Clean(to), ".tsx"), ".ts")
	rel, err := relPath(fromDir, target)
	if err != nil || rel == "" {
		return "./" + path.Base(target)
	}
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel
}

func relPath(fromDir, target string) (string, error) {
	fromParts := strings.Split(strings.Trim(fromDir, "/"), "/")
	targetParts := strings.Split(strings.Trim(target, "/"), "/")
	common := 0
	for common < len(fromParts) && common < len(targetParts) && fromParts[common] == targetParts[common] {
		common++
	}
	var out []string
	for i := common; i < len(fromParts); i++ {
		out = append(out, "..")
	}
	out = append(out, targetParts[common:]...)
	return strings.Join(out, "/"), nil
}
