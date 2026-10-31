package preview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-component-library/internal/components"
	"react-component-library/internal/deps"
)

type compositionCatalog struct {
	*fakeComponentsService
	assets   map[string]components.Component
	versions map[string]components.ComponentVersion
}

func (c *compositionCatalog) Get(_ context.Context, id string) (components.Component, error) {
	a, ok := c.assets[id]
	if !ok {
		return a, fmt.Errorf("missing asset %s", id)
	}
	return a, nil
}

func (c *compositionCatalog) GetVersion(_ context.Context, id, version string) (components.ComponentVersion, error) {
	v, ok := c.versions[id+"@"+version]
	if !ok {
		return v, fmt.Errorf("missing version")
	}
	return v, nil
}

func compositionFixture(t *testing.T) (*service, Composition) {
	t.Helper()
	root := t.TempDir()
	catalog := &compositionCatalog{fakeComponentsService: &fakeComponentsService{}, assets: map[string]components.Component{}, versions: map[string]components.ComponentVersion{}}
	for slug, source := range map[string]string{"Frame": "export function Frame({data}:{data?:{inspector?:unknown}}){return <main>{data?.inspector}</main>}", "Card": "export function Card({label}:{label:string}){return <button>{label}</button>}"} {
		id := "test." + strings.ToLower(slug)
		dir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", slug, "versions", "1.0.0")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, slug+".tsx")
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		asset := components.Component{ID: id, CatalogID: id, LibraryID: id, Slug: slug}
		catalog.assets[id] = asset
		catalog.versions[id+"@1.0.0"] = components.ComponentVersion{ComponentID: id, Version: "1.0.0", Status: components.VersionStatusReleased, DependencyLockPresent: true, Content: source, ContentSHA256: digest(source), SourcePath: path}
	}
	svc := &service{components: catalog, bundler: NewEsbuilder(), repoRoot: root}
	input := Composition{Revision: "revision-a", Template: CompositionAsset{CatalogID: "test.frame", Version: "1.0.0", Export: "Frame"}, Regions: []CompositionRegion{{ID: "inspector", Slot: []string{"data", "inspector"}, Required: true, Asset: &CompositionAsset{CatalogID: "test.card", Version: "1.0.0", Export: "Card"}}}}
	return svc, input
}

func TestCompositionUsesOnePinnedLayoutAndRevisionIdentity(t *testing.T) {
	svc, input := compositionFixture(t)
	result, err := svc.GetCompositionBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"/Card/1.0.0", `React.ComponentProps<typeof Asset1>`, `bindings["$template"]?.["data"]`, `data-design-region="inspector"`, `<Asset1 {...bindings["inspector"]}`} {
		if !strings.Contains(result.Source, needle) {
			t.Fatalf("generated layout omitted %q", needle)
		}
	}
	if len(result.Gaps) != 0 || result.SHA256 == "" || !strings.Contains(result.JS, "button") {
		t.Fatal("real asset composition not bundled")
	}
	repeat, err := svc.GetCompositionBundle(context.Background(), input)
	if err != nil || repeat.SHA256 != result.SHA256 {
		t.Fatal("same inputs changed identity")
	}
	input.Revision = "revision-b"
	next, err := svc.GetCompositionBundle(context.Background(), input)
	if err != nil || next.SHA256 == result.SHA256 {
		t.Fatal("new revision reused stale identity")
	}
}

func TestCompositionGapsAndExportValidation(t *testing.T) {
	svc, input := compositionFixture(t)
	input.Regions[0].Asset = nil
	result, err := svc.GetCompositionBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Gaps) != 1 || !result.Gaps[0].Required || !strings.Contains(result.Source, `data-design-render="wireframe"`) {
		t.Fatalf("missing required asset concealed: %+v", result.Gaps)
	}
	input.Regions[0].Asset = &CompositionAsset{CatalogID: "test.card", Version: "1.0.0", Export: "MissingExport"}
	missingExport, err := svc.GetCompositionBundle(context.Background(), input)
	if err != nil || len(missingExport.Gaps) != 1 || !strings.Contains(missingExport.Gaps[0].Message, "selected export is unavailable") {
		t.Fatalf("missing export did not remain a scoped gap: %v %+v", err, missingExport.Gaps)
	}
	if strings.Contains(missingExport.Source, "MissingExport as") {
		t.Fatal("unresolved export entered generated layout")
	}
	input.Template.Version = "latest"
	if _, err := svc.GetCompositionBundle(context.Background(), input); err == nil {
		t.Fatal("unpinned template accepted")
	}
}

func TestCompositionRejectsSlotOverlapUnsafePathsAndExcessDepth(t *testing.T) {
	_, input := compositionFixture(t)
	for _, path := range [][]string{{"data", "__proto__"}, {"a", "b", "c", "d", "e"}, {}} {
		input.Regions[0].Slot = path
		if validateComposition(input) == nil {
			t.Fatalf("invalid slot accepted: %v", path)
		}
	}
	input.Regions = []CompositionRegion{{ID: "a", Slot: []string{"data"}}, {ID: "b", Slot: []string{"data", "inspector"}}}
	if validateComposition(input) == nil {
		t.Fatal("overlapping slots accepted")
	}
	input.Regions = []CompositionRegion{{ID: `bad"id`, Slot: []string{"content"}}}
	if validateComposition(input) == nil {
		t.Fatal("invalid region identity accepted")
	}
}

func TestCompositionNestedRegionsCompileAndReportHiddenRequiredChildren(t *testing.T) {
	svc, input := compositionFixture(t)
	child := CompositionRegion{ID: "start", Parent: "inspector", Slot: []string{"actions"}, Required: true, Asset: &CompositionAsset{CatalogID: "test.card", Version: "1.0.0", Export: "Card"}}
	// Child-first declarations must not depend on authoring order.
	input.Regions = append([]CompositionRegion{child}, input.Regions...)
	result, err := svc.GetCompositionBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Gaps) != 0 || !strings.Contains(result.Source, `"actions": regions["start"]`) {
		t.Fatalf("nested action was not lowered into its owning asset: %v\n%s", result.Gaps, result.Source)
	}
	input.Regions[1].Asset = nil
	input.Regions[1].Required = false
	result, err = svc.GetCompositionBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, gap := range result.Gaps {
		if gap.Region == "start" && gap.Code == "parent_unresolved" && gap.Required {
			found = true
		}
	}
	if !found {
		t.Fatal("required child hidden by an optional unresolved parent was not reported")
	}
}

func TestCompositionRejectsInvalidParentGraphs(t *testing.T) {
	for _, parents := range [][]string{{"missing", ""}, {"a", ""}, {"b", "a"}} {
		_, input := compositionFixture(t)
		input.Regions = []CompositionRegion{{ID: "a", Parent: parents[0], Slot: []string{"children"}}, {ID: "b", Parent: parents[1], Slot: []string{"children"}}}
		if validateComposition(input) == nil {
			t.Fatalf("invalid parent graph accepted: %v", parents)
		}
	}
}

type compositionRuntimeDeps struct{ deps.Service }

func (compositionRuntimeDeps) ListForComponentVersion(context.Context, string, string) ([]deps.Declaration, error) {
	return nil, nil
}

func TestCompositionIncludesSourceDeclaredRuntimeDependencies(t *testing.T) {
	svc, input := compositionFixture(t)
	svc.deps = compositionRuntimeDeps{}
	catalog := svc.components.(*compositionCatalog)
	v := catalog.versions["test.card@1.0.0"]
	v.Content = "/**\n * @deps {\"clsx\": \"^2.1.0\"}\n */\n" + v.Content
	v.ContentSHA256 = digest(v.Content)
	catalog.versions["test.card@1.0.0"] = v
	if err := os.WriteFile(v.SourcePath, []byte(v.Content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetCompositionBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range got.Dependencies {
		if d.DepName == "clsx" && d.VersionRange == "^2.1.0" {
			return
		}
	}
	t.Fatalf("composition lost published source dependency: %+v", got.Dependencies)
}

func TestCompositionRejectsMaterializedSourceDrift(t *testing.T) {
	for _, target := range []string{"template", "region", "companion"} {
		t.Run(target, func(t *testing.T) {
			svc, input := compositionFixture(t)
			catalog := svc.components.(*compositionCatalog)
			id := "test.card"
			if target == "template" {
				id = "test.frame"
			}
			v := catalog.versions[id+"@1.0.0"]
			path := v.SourcePath
			if target == "companion" {
				path = filepath.Join(filepath.Dir(path), "copy.ts")
				original := `export const label = "Published label";`
				v.Files = append(v.Files, components.ComponentVersionFile{Path: "copy.ts", Content: original, ContentSHA256: digest(original)})
				catalog.versions[id+"@1.0.0"] = v
			}
			// The index still contains the published bytes and digest. Only the
			// materialized source has changed after the last successful index.
			if err := os.WriteFile(path, []byte(`export function Card(){return <p>Unpublished change</p>} export function Frame(){return <main/>}`), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := svc.GetCompositionBundle(context.Background(), input)
			if target == "template" {
				if err == nil || !strings.Contains(err.Error(), "materialized source differs") {
					t.Fatalf("modified template produced a composition: %v", err)
				}
				return
			}
			if err != nil || len(got.Gaps) != 1 || !strings.Contains(got.Gaps[0].Message, "materialized source differs") {
				t.Fatalf("modified region did not remain an explicit gap: %v %+v", err, got.Gaps)
			}
			if strings.Contains(got.JS, "Unpublished change") {
				t.Fatal("modified release entered the composition bundle")
			}
		})
	}
}

type compositionMutationBundler struct {
	Bundler
	path string
}

func (b compositionMutationBundler) BuildBundle(ctx context.Context, source, path string) (string, []string, error) {
	js, warnings, err := b.Bundler.BuildBundle(ctx, source, path)
	if err == nil && filepath.Base(path) == "Composition.generated.tsx" {
		err = os.WriteFile(b.path, []byte(`export function Card(){return <p>Changed during compilation</p>}`), 0o600)
	}
	return js, warnings, err
}

func TestCompositionRejectsSourceChangedDuringCompilation(t *testing.T) {
	svc, input := compositionFixture(t)
	catalog := svc.components.(*compositionCatalog)
	svc.bundler = compositionMutationBundler{Bundler: svc.bundler, path: catalog.versions["test.card@1.0.0"].SourcePath}
	if _, err := svc.GetCompositionBundle(context.Background(), input); err == nil || !strings.Contains(err.Error(), "materialized source differs") {
		t.Fatalf("source changed during compilation produced evidence: %v", err)
	}
}

func TestCompositionSourceCarriesPortableVerifiedProvenance(t *testing.T) {
	svc, input := compositionFixture(t)
	result, err := svc.GetCompositionBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Source, `export const compositionProvenance = `) || !strings.Contains(result.Source, `"generatorVersion":"composition/2"`) || !strings.Contains(result.Source, `"revision":"revision-a"`) || !strings.Contains(result.Source, `"libraryId":"test.card"`) {
		t.Fatal("adopted source loses its origin")
	}
	if strings.Contains(result.Source, svc.repoRoot) {
		t.Fatal("generated provenance contains host paths")
	}
	replica, replicaInput := compositionFixture(t)
	repeated, err := replica.GetCompositionBundle(context.Background(), replicaInput)
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != repeated.Source {
		t.Fatal("same published inputs on another checkout changed source")
	}
	catalog := svc.components.(*compositionCatalog)
	version := catalog.versions["test.card@1.0.0"]
	oldHash := version.ContentSHA256
	version.Content += "\n// Different verified source content\n"
	version.ContentSHA256 = digest(version.Content)
	if err := os.WriteFile(version.SourcePath, []byte(version.Content), 0600); err != nil {
		t.Fatal(err)
	}
	catalog.versions["test.card@1.0.0"] = version
	changed, err := svc.GetCompositionBundle(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Source == result.Source || strings.Contains(changed.Source, oldHash) || !strings.Contains(changed.Source, version.ContentSHA256) {
		t.Fatal("provenance did not track verified source bytes")
	}
}

func TestCompositionProvenanceRejectsConflictingClosureAndPreservesGaps(t *testing.T) {
	c := Composition{Revision: "incomplete"}
	entry := components.ResolvedAsset{Asset: components.Component{LibraryID: "test.child"}, Version: components.ComponentVersion{Version: "1.0.0", ContentSHA256: "first"}}
	conflicting := entry
	conflicting.Version.ContentSHA256 = "second"
	if _, err := compositionProvenance(c, []components.ResolvedAsset{entry, conflicting}, nil); err == nil {
		t.Fatal("conflicting immutable sources accepted")
	}
	gaps := []CompositionGap{{Region: "detail", Code: "asset_unresolved", Message: "/private/checkout/source.tsx failed", Required: true}}
	raw, err := compositionProvenance(c, []components.ResolvedAsset{entry, entry}, gaps)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), `"libraryId":"test.child"`) != 1 || !strings.Contains(string(raw), "asset_unresolved") || strings.Contains(string(raw), "/private/") {
		t.Fatal("closure deduplication concealed unresolved obligation")
	}
}
