package adoptions

import "testing"

func TestLibrarySelectorsImportPresentAcceptsGeneratedExtensions(t *testing.T) {
	for _, source := range []string{
		`import { librarySelectors } from "./selectors.library";`,
		`import { librarySelectors } from "./selectors.library.js";`,
		`import { librarySelectors } from './selectors.library.ts';`,
	} {
		if !librarySelectorsImportPresent(source) {
			t.Errorf("library selector import not recognized: %s", source)
		}
	}
	if librarySelectorsImportPresent(`import { other } from "./selectors.library-extra";`) {
		t.Error("unrelated selector import was recognized")
	}
}
