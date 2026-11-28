package components

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLibraryPackageSpecifierResolutionMatchesExportSelection(t *testing.T) {
	source := `
import { Bare } from "@vrooli/react-component-library/Bare";
import { Partial } from "@vrooli/react-component-library/Partial/2";
import { Exact } from "@vrooli/react-component-library/Exact/1.4.2";
`
	wantSpecifiers := []LibraryPackageSpecifier{
		{Name: "Bare"},
		{Name: "Partial", RequestedVersion: "2"},
		{Name: "Exact", RequestedVersion: "1.4.2"},
	}
	if got := LibraryPackageSpecifiers(source); !reflect.DeepEqual(got, wantSpecifiers) {
		t.Fatalf("LibraryPackageSpecifiers() = %#v, want %#v", got, wantSpecifiers)
	}
	versions := []string{"1.4.2", "2.0.0", "2.3.1", "2.3.1-draft.1"}
	for _, tc := range []struct{ requested, want string }{{"", "2.3.1"}, {"2", "2.3.1"}, {"1.4.2", "1.4.2"}} {
		got, ok := SelectActivePackageVersion(versions, tc.requested)
		if !ok || got != tc.want {
			t.Fatalf("SelectActivePackageVersion(%q) = %q, %v; want %q", tc.requested, got, ok, tc.want)
		}
	}
}

func TestResolveSourceDependencyRejectsAmbiguousAssetDeterministically(t *testing.T) {
	root := t.TempDir()
	for kind, libraryID := range map[string]string{"components": "library:ComponentCopy", "foundations": "library:FoundationCopy"} {
		base := filepath.Join(root, kind, "Shared")
		if err := os.MkdirAll(filepath.Join(base, "versions", "1.0.0"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(base, "component.json"), []byte(`{"libraryId":"`+libraryID+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := NewFSContentStore(root)
	for i := 0; i < 10; i++ {
		_, err := store.resolveSourceDependency("Shared", "")
		if err == nil || err.Error() != "ambiguous intra-library asset Shared: found in components, foundations" {
			t.Fatalf("resolveSourceDependency() error = %v; want stable ambiguity error", err)
		}
	}
}
