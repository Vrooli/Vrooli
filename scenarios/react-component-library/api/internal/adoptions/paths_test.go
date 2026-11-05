package adoptions

import "testing"

func TestRelativeImportSameDirectory(t *testing.T) {
	if got := relativeImport("ui/src/consts/selectors.ts", "ui/src/consts/selectors.library.ts"); got != "./selectors.library" {
		t.Fatalf("got %q", got)
	}
}

func TestRelativeImportAcrossDirectories(t *testing.T) {
	if got := relativeImport("ui/src/consts/selectors.ts", "ui/src/generated/library/selectors.ts"); got != "../generated/library/selectors" {
		t.Fatalf("got %q", got)
	}
	if got := relativeImport("ui/src/selectors.ts", "ui/src/consts/selectors.library.ts"); got != "./consts/selectors.library" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultScenarioPathsMatchReactViteLayout(t *testing.T) {
	paths := DefaultScenarioPaths()
	if paths.TokenRamp != "ui/src/design-tokens.css" || paths.LocaleCatalogue != "ui/src/i18n/locales/en.json" || paths.SelectorRegistry != "ui/src/consts/selectors.ts" || paths.LibrarySelectors != "ui/src/consts/selectors.library.ts" || paths.AppEntry != "ui/src/main.tsx" {
		t.Fatalf("unexpected defaults: %+v", paths)
	}
}
