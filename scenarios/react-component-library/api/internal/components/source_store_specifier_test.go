package components

import (
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
