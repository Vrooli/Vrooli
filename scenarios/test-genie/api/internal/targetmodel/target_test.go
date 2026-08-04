package targetmodel

import (
	"path/filepath"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestResolveCanonicalTargets(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..")
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		expression string
		kind       commonv1.ValidationTargetKind
		id         string
		root       string
	}{
		{expression: "storage-manager", kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, id: "storage-manager", root: "scenarios/storage-manager"},
		{expression: "package:api-core", kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE, id: "api-core", root: "packages/api-core"},
		{expression: "module:packages/api-core", kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE, id: "api-core", root: "packages/api-core"},
		{expression: "docs:docs", kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS, id: "docs", root: "docs"},
	} {
		got, err := Resolve(root, test.expression)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", test.expression, err)
		}
		if got.Kind != test.kind || got.ID != test.id || filepath.ToSlash(got.Root) != test.root {
			t.Fatalf("Resolve(%q) = %#v, want kind=%v id=%q root=%q", test.expression, got, test.kind, test.id, test.root)
		}
	}
}

func TestArtifactRootSeparatesNonScenarioTargets(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	target, err := Resolve(root, "package:api-core")
	if err != nil {
		t.Fatal(err)
	}
	artifactRoot, err := ArtifactRoot(root, target)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(artifactRoot) == filepath.Clean(target.Path) {
		t.Fatalf("ArtifactRoot(%#v) = %q; target evidence must be outside source", target, artifactRoot)
	}
	if filepath.Base(artifactRoot) != target.ID {
		t.Fatalf("ArtifactRoot = %q; want target id %q as final component", artifactRoot, target.ID)
	}
}

func TestResolveTeamAcceptsManifestRootAliasAndUsesOwnerID(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	target, err := Resolve(root, "team:docs/marketing")
	if err != nil {
		t.Fatal(err)
	}
	if target.ID != "marketing-crew" || target.Root != "docs/marketing" {
		t.Fatalf("target = %#v, want owner id marketing-crew and root docs/marketing", target)
	}
}
