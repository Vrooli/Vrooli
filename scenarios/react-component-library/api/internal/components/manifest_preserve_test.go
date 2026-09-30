package components

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFSContentStore_UpdateManifestPreservesAllFields pins the fix for bug
// c71f56c0: UpdateManifest must preserve every field it does not intentionally
// change — designStyles, fileSlots, and any author-added key — instead of
// dropping them through a lossy struct round-trip.
func TestFSContentStore_UpdateManifestPreservesAllFields(t *testing.T) {
	root := t.TempDir()
	manifestRel := filepath.ToSlash(filepath.Join("components", "data-table", "component.json"))
	abs := filepath.Join(root, manifestRel)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	original := `{
  "libraryId": "react-component-library:DataTable",
  "displayName": "Data Table",
  "description": "Dense table.",
  "tags": [
    "data",
    "table"
  ],
  "slot": "ui-pattern",
  "category": "data-display",
  "fileSlots": {
    "useTableState.ts": "hook"
  },
  "designStyles": [
    {
      "styleId": "vrooli-default",
      "affinity": "native"
    },
    {
      "styleId": "vrooli-command-display",
      "affinity": "discouraged"
    }
  ],
  "customAuthorField": "keep-me",
  "x-supplementalJustification": "temporary",
  "latest": "1.1.1",
  "draft": "",
  "deprecatedVersions": []
}
`
	require.NoError(t, os.WriteFile(abs, []byte(original), 0o600))

	store := NewFSContentStore(root)
	c := Component{
		Slug:          "data-table",
		ManifestPath:  manifestRel,
		LibraryID:     "react-component-library:DataTable",
		DisplayName:   "Data Table",
		Description:   "Dense table.",
		Tags:          []string{"data", "table"},
		LatestVersion: "1.1.1",
	}
	// Simulate a release version-create bumping latest 1.1.1 -> 1.1.2.
	require.NoError(t, store.UpdateManifest(context.Background(), c, UpdateComponentManifestInput{
		ComponentID: c.ID, DisplayName: c.DisplayName, Description: c.Description,
		Tags: c.Tags, LatestVersion: "1.1.2", CatalogID: "data-display.table",
		ReplacedBy: []string{"data-display.grid"}, ClearSupplementalJustification: true,
	}))

	raw, err := os.ReadFile(abs)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))

	require.Equal(t, "1.1.2", got["latest"], "the managed version pointer must advance")

	require.Contains(t, got, "designStyles", "designStyles must survive version-create")
	styles, ok := got["designStyles"].([]any)
	require.True(t, ok)
	require.Len(t, styles, 2)
	require.Equal(t, "vrooli-default", styles[0].(map[string]any)["styleId"])
	require.Equal(t, "discouraged", styles[1].(map[string]any)["affinity"])

	require.Contains(t, got, "fileSlots")
	require.Equal(t, "hook", got["fileSlots"].(map[string]any)["useTableState.ts"])
	require.Equal(t, "keep-me", got["customAuthorField"], "unknown author keys must survive")
	require.Equal(t, "data-display.table", got["catalogId"])
	require.Equal(t, []any{"data-display.grid"}, got["replacedBy"])
	require.NotContains(t, got, "x-supplementalJustification")
	require.Contains(t, got, "draft", "an explicit empty draft key is preserved")
	require.Equal(t, "ui-pattern", got["slot"])
	require.Equal(t, "data-display", got["category"])

	require.NoError(t, store.UpdateManifest(context.Background(), c, UpdateComponentManifestInput{
		LatestVersion: "1.1.2", ClearCatalogID: true,
	}))
	raw, err = os.ReadFile(abs)
	require.NoError(t, err)
	got = map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &got))
	require.NotContains(t, got, "catalogId")

	// Field order is preserved: designStyles and the author field still precede latest.
	rawStr := string(raw)
	require.Less(t, strings.Index(rawStr, `"designStyles"`), strings.Index(rawStr, `"latest"`))
	require.Less(t, strings.Index(rawStr, `"customAuthorField"`), strings.Index(rawStr, `"latest"`))
}

func TestFSContentStore_UpdateManifestSetsExplicitEntry(t *testing.T) {
	root := t.TempDir()
	manifestRel := filepath.ToSlash(filepath.Join("components", "markdown-renderer", "component.json"))
	abs := filepath.Join(root, manifestRel)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, []byte(`{
  "libraryId": "react-component-library:markdown-renderer",
  "latest": "0.4.2",
  "draft": "",
  "deprecatedVersions": []
}
`), 0o600))

	store := NewFSContentStore(root)
	require.NoError(t, store.UpdateManifest(context.Background(), Component{
		Slug: "markdown-renderer", ManifestPath: manifestRel,
		LibraryID: "react-component-library:markdown-renderer", LatestVersion: "0.4.2",
	}, UpdateComponentManifestInput{Entry: "markdown-renderer.tsx"}))

	raw, err := os.ReadFile(abs)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, "markdown-renderer.tsx", got["entry"])
}

func TestFSContentStore_UpdateManifestRejectsUnsafeEntry(t *testing.T) {
	root := t.TempDir()
	manifestRel := filepath.ToSlash(filepath.Join("components", "card", "component.json"))
	abs := filepath.Join(root, manifestRel)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, []byte(`{"libraryId":"react-component-library:Card","latest":"1.0.0"}`), 0o600))

	err := NewFSContentStore(root).UpdateManifest(context.Background(), Component{
		Slug: "card", ManifestPath: manifestRel, LibraryID: "react-component-library:Card", LatestVersion: "1.0.0",
	}, UpdateComponentManifestInput{Entry: "../Card.tsx"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "entry")
}

func TestFSContentStore_UpdateManifestClearsStaleEvictedMarkerForMaterializedVersion(t *testing.T) {
	root := t.TempDir()
	manifestRel := filepath.ToSlash(filepath.Join("components", "card", "component.json"))
	base := filepath.Join(root, "components", "card")
	require.NoError(t, os.MkdirAll(filepath.Join(base, "versions", "1.2.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "component.json"), []byte(`{
  "libraryId": "react-component-library:Card",
  "displayName": "Card",
  "description": "Surface.",
  "latest": "1.2.1",
  "draft": "",
  "deprecatedVersions": ["1.2.0"],
  "evictedVersions": ["1.1.0", "1.2.0"]
}
`), 0o600))

	store := NewFSContentStore(root)
	c := Component{Slug: "card", ManifestPath: manifestRel, LibraryID: "react-component-library:Card", LatestVersion: "1.2.1"}
	require.NoError(t, store.UpdateManifest(context.Background(), c, UpdateComponentManifestInput{
		Description: "Surface.", ReconcileEvictedVersions: true, EvictedVersions: []string{"1.1.0", "1.2.0"},
	}))

	raw, err := os.ReadFile(filepath.Join(base, "component.json"))
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, []any{"1.1.0"}, got["evictedVersions"])
	require.Equal(t, []any{"1.2.0"}, got["deprecatedVersions"], "retirement status remains independent of placement")
}

// TestFSContentStore_CreateVersionPreservesDesignStyles proves the end-to-end
// version-create flow (which calls UpdateManifest) no longer strips affinities.
func TestFSContentStore_CreateVersionPreservesDesignStyles(t *testing.T) {
	root := t.TempDir()
	slug := "data-table"
	base := filepath.Join(root, "components", slug)
	require.NoError(t, os.MkdirAll(filepath.Join(base, "versions", "1.1.1"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "versions", "1.1.1", "DataTable.tsx"), []byte("export const DataTable = () => null;\n"), 0o600))
	manifest := `{
  "libraryId": "react-component-library:DataTable",
  "displayName": "Data Table",
  "description": "Dense table.",
  "tags": [
    "data"
  ],
  "slot": "ui-pattern",
  "category": "data-display",
  "designStyles": [
    {
      "styleId": "vrooli-default",
      "affinity": "native"
    }
  ],
  "latest": "1.1.1",
  "draft": "",
  "deprecatedVersions": []
}

`
	require.NoError(t, os.WriteFile(filepath.Join(base, "component.json"), []byte(manifest), 0o600))

	store := NewFSContentStore(root)
	c := Component{
		Slug:          slug,
		ManifestPath:  filepath.ToSlash(filepath.Join("components", slug, "component.json")),
		SourcePath:    filepath.ToSlash(filepath.Join("components", slug, "versions", "1.1.1", "DataTable.tsx")),
		LibraryID:     "react-component-library:DataTable",
		DisplayName:   "Data Table",
		Description:   "Dense table.",
		Tags:          []string{"data"},
		LatestVersion: "1.1.1",
	}
	_, err := store.CreateVersion(context.Background(), c, CreateComponentVersionInput{
		Version: "1.1.2", FileName: "DataTable.tsx", Source: "export const DataTable = () => <div/>;\n",
	})
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(base, "component.json"))
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, "1.1.2", got["latest"])
	require.Contains(t, got, "designStyles", "designStyles must survive a real version cut")
	require.Len(t, got["designStyles"].([]any), 1)
}

func TestEnsureHeaderFieldsMergesDependencyMetadataAndRemovesDuplicates(t *testing.T) {
	source := `/**
 * @libraryId react-component-library:markdown-renderer
 * @version 0.2.0
 */
/**
 * @libraryId react-component-library:markdown-renderer
 * @deps {"react":"^18","react-markdown":"^10.1.0"}
 */
export function MarkdownRenderer() { return null; }
`
	got := ensureHeaderFields(source, "react-component-library:markdown-renderer", "Markdown Renderer", "Renderer", "0.3.0", []string{"markdown"})
	require.Equal(t, 1, strings.Count(got, "@libraryId"))
	require.Contains(t, got, `@deps {"react":"^18","react-markdown":"^10.1.0"}`)
	require.Contains(t, got, "export function MarkdownRenderer")
}

func TestFSContentStore_MajorAliasRetirement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "components", "Panel", "component.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(`{"libraryId":"react-component-library:Panel","latest":"2.0.0","custom":"keep"}`), 0644))
	store := NewFSContentStore(root)
	c := Component{Slug: "Panel", LibraryID: "react-component-library:Panel", LatestVersion: "2.0.0"}
	require.NoError(t, store.UpdateManifest(t.Context(), c, UpdateComponentManifestInput{RetiredMajorAliases: []string{"1"}}))
	read := func() map[string]any {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(data, &got))
		return got
	}
	require.Equal(t, []any{"1"}, read()["retiredMajorAliases"])
	require.NoError(t, store.UpdateManifest(t.Context(), c, UpdateComponentManifestInput{Description: "Updated"}))
	require.Equal(t, []any{"1"}, read()["retiredMajorAliases"])
	for _, majors := range [][]string{{"2"}, {"01"}, {"x"}} {
		require.Error(t, store.UpdateManifest(t.Context(), c, UpdateComponentManifestInput{RetiredMajorAliases: majors}))
		require.Equal(t, []any{"1"}, read()["retiredMajorAliases"])
	}
	require.Error(t, store.UpdateManifest(t.Context(), c, UpdateComponentManifestInput{LatestVersion: "1.0.0"}))
	require.NoError(t, store.UpdateManifest(t.Context(), c, UpdateComponentManifestInput{RetiredMajorAliases: []string{}}))
	require.Empty(t, read()["retiredMajorAliases"])
	require.Equal(t, "keep", read()["custom"])
}

func TestFSContentStore_ArchiveComponentPreservesReleasedBytes(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "components", "Panel")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "versions", "1.0.0"), 0755))
	manifest := []byte(`{"libraryId":"react-component-library:Panel","latest":"1.0.0"}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.json"), manifest, 0644))
	source := []byte("// immutable release\nexport const Panel = () => null;\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "versions", "1.0.0", "Panel.tsx"), source, 0644))
	store := NewFSContentStore(root)
	c := Component{LibraryID: "react-component-library:Panel", ManifestPath: "components/Panel/component.json"}
	wrong := c
	wrong.LibraryID = "react-component-library:Other"
	_, err := store.ArchiveComponentSource(wrong)
	require.Error(t, err)
	require.FileExists(t, filepath.Join(dir, "component.json"))
	archive, err := store.ArchiveComponentSource(c)
	require.NoError(t, err)
	require.NoDirExists(t, dir)
	got, err := os.ReadFile(filepath.Join(archive, "versions", "1.0.0", "Panel.tsx"))
	require.NoError(t, err)
	require.Equal(t, source, got)
	got, err = os.ReadFile(filepath.Join(archive, "component.json"))
	require.NoError(t, err)
	require.Equal(t, manifest, got)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.json"), manifest, 0644))
	_, err = store.ArchiveComponentSource(c)
	require.Error(t, err)
	require.FileExists(t, filepath.Join(dir, "component.json"))
}

func TestFSContentStore_ArchiveComponentRejectsSymlinkPaths(t *testing.T) {
	for _, location := range []string{"kind", "asset", "manifest", "archive"} {
		t.Run(location, func(t *testing.T) {
			root := t.TempDir()
			outside := t.TempDir()
			dir := filepath.Join(root, "components", "Panel")
			require.NoError(t, os.MkdirAll(dir, 0755))
			manifest := []byte(`{"libraryId":"react-component-library:Panel","latest":"1.0.0"}`)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "component.json"), manifest, 0644))
			var link, target string
			switch location {
			case "kind":
				link = filepath.Join(root, "components")
				target = filepath.Join(outside, "components")
			case "asset":
				link = dir
				target = filepath.Join(outside, "Panel")
			case "manifest":
				link = filepath.Join(dir, "component.json")
				target = filepath.Join(outside, "component.json")
			case "archive":
				link = filepath.Join(root, ".retired")
				target = outside
			}
			if location != "archive" {
				require.NoError(t, os.Rename(link, target))
			}
			require.NoError(t, os.Symlink(target, link))
			_, err := NewFSContentStore(root).ArchiveComponentSource(Component{LibraryID: "react-component-library:Panel", ManifestPath: "components/Panel/component.json"})
			require.Error(t, err)
			got, err := os.ReadFile(filepath.Join(dir, "component.json"))
			require.NoError(t, err)
			require.Equal(t, manifest, got)
		})
	}
}

type retirementSnapshotRepository struct {
	Repository
	fail bool
}

func (r retirementSnapshotRepository) ListVersions(context.Context, string, int) ([]ComponentVersion, error) {
	return []ComponentVersion{{Version: "1.0.0", Presence: "evicted"}}, nil
}
func (r retirementSnapshotRepository) GetVersion(context.Context, string, string) (ComponentVersion, error) {
	if r.fail {
		return ComponentVersion{}, os.ErrNotExist
	}
	return ComponentVersion{Version: "1.0.0", Presence: "evicted", Files: []ComponentVersionFile{{Path: "Panel.tsx", Content: "immutable"}}}, nil
}
func (r retirementSnapshotRepository) ListStories(context.Context, StoryQuery) ([]ComponentStory, error) {
	return []ComponentStory{{}}, nil
}
func TestComponentRetirementSnapshotRetainsEvictedFileMirror(t *testing.T) {
	snapshot, err := prepareComponentRetirementSnapshot(t.Context(), retirementSnapshotRepository{}, Component{ID: "panel"})
	require.NoError(t, err)
	require.Len(t, snapshot.Versions, 1)
	require.Equal(t, "evicted", snapshot.Versions[0].Presence)
	require.Equal(t, "immutable", snapshot.Versions[0].Files[0].Content)
	require.Len(t, snapshot.Stories, 1)
	_, err = prepareComponentRetirementSnapshot(t.Context(), retirementSnapshotRepository{fail: true}, Component{ID: "panel"})
	require.Error(t, err)
}

func TestFSContentStore_RetirementSnapshotPersistsBeforeMutation(t *testing.T) {
	root := t.TempDir()
	store := NewFSContentStore(root)
	snapshot := componentRetirementSnapshot{Component: Component{LibraryID: "react-component-library:Panel"}, Versions: []ComponentVersion{{Version: "1.0.0", Presence: "evicted", Files: []ComponentVersionFile{{Path: "Panel.tsx", Content: "retained bytes"}}}}}
	path, err := store.PersistComponentRetirementSnapshot(snapshot)
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var got componentRetirementSnapshot
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, snapshot, got)
	repeated, err := store.PersistComponentRetirementSnapshot(snapshot)
	require.NoError(t, err)
	require.Equal(t, path, repeated)
	snapshot.Versions[0].Files[0].Content = "replacement"
	_, err = store.PersistComponentRetirementSnapshot(snapshot)
	require.Error(t, err)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, data, after)
	pending, err := filepath.Glob(filepath.Join(root, ".retired", ".snapshot-*"))
	require.NoError(t, err)
	require.Empty(t, pending)
}

func TestComponentArchiveRequiresPersistedHistory(t *testing.T) {
	for _, failure := range []string{"", "read", "persist", "cancel"} {
		t.Run(failure, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "components", "Panel")
			require.NoError(t, os.MkdirAll(dir, 0755))
			manifest := filepath.Join(dir, "component.json")
			require.NoError(t, os.WriteFile(manifest, []byte(`{"libraryId":"react-component-library:Panel","latest":"1.0.0"}`), 0644))
			store := NewFSContentStore(root)
			c := Component{ID: "panel", LibraryID: "react-component-library:Panel", ManifestPath: "components/Panel/component.json"}
			repo := retirementSnapshotRepository{fail: failure == "read"}
			if failure == "persist" {
				_, err := store.PersistComponentRetirementSnapshot(componentRetirementSnapshot{Component: c})
				require.NoError(t, err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if failure == "cancel" {
				cancel()
			}
			out, err := archiveComponentWithHistory(ctx, repo, store, c)
			if failure != "" {
				require.Error(t, err)
				require.FileExists(t, manifest)
				return
			}
			require.NoError(t, err)
			require.FileExists(t, out.SnapshotPath)
			require.FileExists(t, filepath.Join(out.SourceArchivePath, "component.json"))
			require.NoFileExists(t, manifest)
			data, err := os.ReadFile(out.SnapshotPath)
			require.NoError(t, err)
			var snapshot componentRetirementSnapshot
			require.NoError(t, json.Unmarshal(data, &snapshot))
			require.Equal(t, "immutable", snapshot.Versions[0].Files[0].Content)
		})
	}
}
