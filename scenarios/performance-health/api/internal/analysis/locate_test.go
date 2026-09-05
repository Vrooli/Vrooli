package analysis

import (
	"path/filepath"
	"testing"
)

// [REQ:PH-ANALYSIS-002] SourceLocator resolves component names to file:line
// across the common React declaration forms, anchored on word boundaries.
func TestSourceLocatorResolvesComponents(t *testing.T) {
	abs, err := filepath.Abs("testdata/fixture-scenario")
	if err != nil {
		t.Fatal(err)
	}
	loc := SourceLocator{ScenarioRoot: abs}

	cases := map[string]string{
		// export const Foo = memo(function ...)
		"ProjectList": "ui/src/features/list/ProjectList.tsx:9",
		// export function Foo()
		"App": "ui/src/App.tsx:1",
	}
	for component, want := range cases {
		got, ok := loc.Locate("fixture-scenario", component)
		if !ok {
			t.Errorf("%s: expected to locate, got miss", component)
			continue
		}
		if got != want {
			t.Errorf("%s: got %q, want %q", component, got, want)
		}
	}
}

// A component that does not exist in the source tree returns a miss (so the
// finding still emits, with a "definition not located" note).
func TestSourceLocatorMissesUnknown(t *testing.T) {
	abs, _ := filepath.Abs("testdata/fixture-scenario")
	loc := SourceLocator{ScenarioRoot: abs}
	if def, ok := loc.Locate("fixture-scenario", "DoesNotExist"); ok {
		t.Fatalf("expected miss for unknown component, got %q", def)
	}
}

// Word-boundary anchoring: "List" must not match "ProjectList".
func TestSourceLocatorWordBoundary(t *testing.T) {
	abs, _ := filepath.Abs("testdata/fixture-scenario")
	loc := SourceLocator{ScenarioRoot: abs}
	if def, ok := loc.Locate("fixture-scenario", "List"); ok {
		t.Fatalf("'List' must not match 'ProjectList', got %q", def)
	}
}
