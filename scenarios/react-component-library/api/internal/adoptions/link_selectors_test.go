package adoptions

import (
	"strings"
	"testing"
)

func TestDerivedSelectorIDsUseSemanticDottedNames(t *testing.T) {
	ids := derivedSelectorIDs(`<div data-testid="sidebar-shell" /><button data-testid="sidebar-shell-close" />`, "navigation.sidebar")
	if got, want := strings.Join(ids, ","), "navigation.sidebar,navigation.sidebar.close"; got != want {
		t.Fatalf("derived selector ids = %q, want %q", got, want)
	}
	entry := selectorEntry(`"navigation.sidebar"`, "navigation.sidebar", ids)
	if !strings.Contains(entry, `"root": "navigation.sidebar"`) || !strings.Contains(entry, `"close": "navigation.sidebar.close"`) {
		t.Fatalf("semantic selector entry missing expected fields:\n%s", entry)
	}
	if strings.Contains(entry, "id2") {
		t.Fatalf("semantic selector entry retained positional field:\n%s", entry)
	}
}

func TestMergeSelectorRegionReplacesExistingEntry(t *testing.T) {
	source := `// vrooli:library-selectors start
export const librarySelectors = {
  "navigation.sidebar": {
    "root": "navigation.sidebar",
    "id2": "sidebar-shell-close",
  },
} as const;
// vrooli:library-selectors end
`
	updated, changed := mergeSelectorRegion(source, "navigation.sidebar", []string{"navigation.sidebar", "navigation.sidebar.close"})
	if !changed {
		t.Fatal("mergeSelectorRegion did not replace the managed entry")
	}
	if strings.Contains(updated, "id2") || !strings.Contains(updated, `"close": "navigation.sidebar.close"`) {
		t.Fatalf("merged selector region is not semantic:\n%s", updated)
	}
}
