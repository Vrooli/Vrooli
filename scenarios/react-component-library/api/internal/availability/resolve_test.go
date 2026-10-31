package availability

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"react-component-library/internal/components"
)

type catalogFixture struct {
	assets   []components.Component
	versions map[string]components.ComponentVersion
	reads    int
}

func (f *catalogFixture) List(context.Context, components.SearchQuery) ([]components.Component, error) {
	return f.assets, nil
}
func (f *catalogFixture) Get(context.Context, string) (components.Component, error) {
	panic("snapshot must own identity reads")
}
func (f *catalogFixture) GetByLibraryID(context.Context, string) (components.Component, error) {
	panic("snapshot must own identity reads")
}
func (f *catalogFixture) GetVersion(_ context.Context, id, version string) (components.ComponentVersion, error) {
	f.reads++
	v, ok := f.versions[id+"@"+version]
	if !ok {
		return v, fmt.Errorf("missing version")
	}
	return v, nil
}
func fixture() *catalogFixture {
	source := "export const Card = () => null;"
	return &catalogFixture{assets: []components.Component{{ID: "uuid", CatalogID: "components.card", LibraryID: "card", LatestVersion: "1.0.0"}}, versions: map[string]components.ComponentVersion{"uuid@1.0.0": {ComponentID: "uuid", Version: "1.0.0", Status: components.VersionStatusReleased, DependencyLockPresent: true, SourcePath: "Card.tsx", Content: source, ContentSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(source)))}}}
}
func TestResolveRequiresPublishedImmutableBuild(t *testing.T) {
	cases := []struct {
		name, code string
		edit       func(*components.ComponentVersion)
		compile    Compile
	}{
		{name: "published", code: "published_build_resolved"},
		{name: "draft", code: "unpublished_version", edit: func(v *components.ComponentVersion) { v.Status = components.VersionStatusDraft }},
		{name: "retired", code: "retired_version", edit: func(v *components.ComponentVersion) { v.Status = "retired" }},
		{name: "unknown status", code: "unknown_lifecycle", edit: func(v *components.ComponentVersion) { v.Status = "" }},
		{name: "missing lock", code: "dependency_lock_missing", edit: func(v *components.ComponentVersion) { v.DependencyLockPresent = false }},
		{name: "missing source", code: "source_missing", edit: func(v *components.ComponentVersion) { v.Content = "" }},
		{name: "changed entry", code: "source_hash_mismatch", edit: func(v *components.ComponentVersion) { v.Content += "changed" }},
		{name: "changed companion", code: "source_hash_mismatch", edit: func(v *components.ComponentVersion) {
			v.Files = []components.ComponentVersionFile{{Path: "helper.ts", Content: "changed", ContentSHA256: "recorded"}}
		}},
		{name: "wrong version", code: "dependency_unresolved", edit: func(v *components.ComponentVersion) { v.Version = "2.0.0" }},
		{name: "missing dependency", code: "dependency_unresolved", edit: func(v *components.ComponentVersion) {
			v.Dependencies = []components.AssetDependency{{LibraryID: "missing", Version: "1.0.0"}}
		}},
		{name: "compile failure", code: "build_failed", compile: func(context.Context, string, string) (string, error) { return "", fmt.Errorf("unresolved export") }},
		{name: "empty build evidence", code: "build_evidence_missing", compile: func(context.Context, string, string) (string, error) { return "", nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := fixture()
			v := f.versions["uuid@1.0.0"]
			if tc.edit != nil {
				tc.edit(&v)
			}
			f.versions["uuid@1.0.0"] = v
			compile := tc.compile
			if compile == nil {
				compile = func(context.Context, string, string) (string, error) { return "bundle-digest", nil }
			}
			s, err := NewSnapshot(context.Background(), f, compile)
			if err != nil {
				t.Fatal(err)
			}
			got := s.Resolve(context.Background(), "components.card", "1.0.0")
			if got.ReasonCode != tc.code || got.IsBuilt() != (tc.name == "published") {
				t.Fatalf("unexpected evidence: %+v", got)
			}
		})
	}
}
func TestResolveIdentityCancellationAndSnapshotReuse(t *testing.T) {
	f := fixture()
	calls := 0
	s, err := NewSnapshot(context.Background(), f, func(_ context.Context, id, version string) (string, error) {
		calls++
		if id != "uuid" || version != "1.0.0" {
			t.Fatalf("wrong compile target %s@%s", id, version)
		}
		return "bundle", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if !s.Resolve(context.Background(), "components.card", "1.0.0").IsBuilt() {
			t.Fatal("published version unavailable")
		}
	}
	if f.reads != 1 || calls != 1 {
		t.Fatalf("repeated resolution reads=%d builds=%d", f.reads, calls)
	}
	for _, tc := range []struct{ id, version, code string }{{"card", "1.0.0", "catalog_identity_mismatch"}, {"components.card", "", "version_not_selected"}, {"components.cards", "1.0.0", "asset_unresolved"}} {
		if got := s.Resolve(context.Background(), tc.id, tc.version); got.ReasonCode != tc.code {
			t.Fatalf("%+v", got)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := s.Resolve(ctx, "components.card", "1.0.0"); got.ReasonCode != "cancelled" {
		t.Fatalf("cached result ignored cancellation: %+v", got)
	}
	f.assets = append(f.assets, components.Component{ID: "other", CatalogID: "components.card"})
	ambiguous, _ := NewSnapshot(context.Background(), f, nil)
	if got := ambiguous.Resolve(context.Background(), "components.card", "1.0.0"); got.ReasonCode != "asset_unresolved" {
		t.Fatalf("ambiguous identity accepted: %+v", got)
	}
}
