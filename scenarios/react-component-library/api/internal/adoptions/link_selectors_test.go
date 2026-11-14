package adoptions

import (
	"strings"
	"testing"
)

func TestMergeSelectorRegionSeparatesEntriesWithoutTrailingComma(t *testing.T) {
	source := "// vrooli:library-selectors start\nexport const librarySelectors = {\n  \"old\": {}\n} as const;\n// vrooli:library-selectors end\n"
	updated, changed := mergeSelectorRegion(source, "new", []string{"new"})
	if !changed || !strings.Contains(updated, "\"old\": {},\n") {
		t.Fatalf("appended entry lacks a separator:\n%s", updated)
	}
	again, changed := mergeSelectorRegion(updated, "new", []string{"new"})
	if changed || again != updated {
		t.Fatal("selector merge must be idempotent")
	}
}

func TestDerivedSelectorIDsPreserveRenderedAttributes(t *testing.T) {
	ids := derivedSelectorIDs(`<div data-testid="sidebar-shell" /><button data-testid="sidebar-shell-close" />`, "navigation.sidebar")
	if got, want := strings.Join(ids, ","), "sidebar-shell,sidebar-shell-close"; got != want {
		t.Fatalf("derived selector ids = %q, want %q", got, want)
	}
	entry := selectorEntry(`"navigation.sidebar"`, "navigation.sidebar", ids)
	if !strings.Contains(entry, `"sidebarShell": "sidebar-shell"`) || !strings.Contains(entry, `"sidebarShellClose": "sidebar-shell-close"`) {
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

func TestDerivedSelectorsRecognizeDefaultPropsWithoutInventingRoots(t *testing.T) {
	source := "function Sidebar({testId = \"navigation.sidebar\"}) { return <aside data-testid={testId}><button data-testid={`${testId}-close`} /></aside> }"
	ids := derivedSelectorIDs(source, "catalog.unrelated")
	if got := strings.Join(ids, ","); got != "navigation.sidebar,navigation.sidebar-close" {
		t.Fatalf("default IDs: %s", got)
	}
	ids = derivedSelectorIDs(`<button data-testid={testId ?? "controls.button"} />`, "catalog.unrelated")
	if len(ids) != 1 || ids[0] != "controls.button" {
		t.Fatalf("fallback IDs: %v", ids)
	}
}
