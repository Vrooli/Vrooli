package resources

import (
	"path/filepath"
	"testing"
)

func TestNativeOpenRGBOwnerUsesOperatorOnlyStorageWithoutCrossAccountGrant(t *testing.T) {
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		t.Fatal(err)
	}
	owner := openRGBNativeOwner{SchemaVersion: openRGBNativeOwnerVersion, OwnerPrincipal: "uid:501", RuntimePrincipal: "uid:501", NativeAccount: "uid:501"}
	for _, path := range []string{root, filepath.Join(root, "owner.json"), filepath.Join(root, "launch", "attempt", "capsule.json"), filepath.Join(root, "state", "managed-update", "ledger.json"), filepath.Join(root, "runtime", "initial", "attempt", "config", "OpenRGB.json")} {
		policy, err := owner.policyFor(path)
		if err != nil {
			t.Fatal(err)
		}
		if policy.OwnerPrincipal != owner.OwnerPrincipal || policy.RuntimePrincipal != "" || policy.RuntimeReadOnly {
			t.Fatalf("same-user storage must not assert ACL vendor isolation: %+v", policy)
		}
		if path == root && !policy.NoInherit {
			t.Fatal("native root must not inherit broader grants")
		}
	}
	if _, err := owner.policyFor(root + "-other/ledger.json"); err == nil {
		t.Fatal("sibling resource admitted")
	}
	for _, changed := range []openRGBNativeOwner{
		{SchemaVersion: openRGBNativeOwnerVersion, OwnerPrincipal: "uid:501", RuntimePrincipal: "uid:502", NativeAccount: "_vrooli_openrgb"},
		{SchemaVersion: openRGBNativeOwnerVersion, OwnerPrincipal: "uid:0", RuntimePrincipal: "uid:0", NativeAccount: "uid:0"},
		{SchemaVersion: openRGBNativeOwnerVersion, OwnerPrincipal: "sid:S-1-5-80-1-2-3-4-5", RuntimePrincipal: "sid:S-1-5-80-1-2-3-4-5", NativeAccount: "sid:S-1-5-80-1-2-3-4-5"},
	} {
		if changed.validate() == nil {
			t.Fatalf("old account separation or privileged owner admitted: %+v", changed)
		}
	}
}

func TestNativeServingEnvironmentComparesValuesAndRefusesAmbiguousKeys(t *testing.T) {
	if !sameOpenRGBEnvironment([]string{"A=1", "B=2"}, []string{"B=2", "A=1"}) {
		t.Fatal("native environment ordering changed its semantic contract")
	}
	for _, other := range [][]string{{"A=1", "B=3"}, {"A=1", "A=1", "B=2"}, {"A=1", "B"}, {"A=1", "=2"}} {
		if sameOpenRGBEnvironment([]string{"A=1", "B=2"}, other) {
			t.Fatalf("ambiguous or changed environment accepted: %v", other)
		}
	}
}
