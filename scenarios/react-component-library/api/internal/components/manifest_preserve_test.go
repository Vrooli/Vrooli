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
		Tags: c.Tags, LatestVersion: "1.1.2",
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
	require.Contains(t, got, "draft", "an explicit empty draft key is preserved")
	require.Equal(t, "ui-pattern", got["slot"])
	require.Equal(t, "data-display", got["category"])

	// Field order is preserved: designStyles and the author field still precede latest.
	rawStr := string(raw)
	require.Less(t, strings.Index(rawStr, `"designStyles"`), strings.Index(rawStr, `"latest"`))
	require.Less(t, strings.Index(rawStr, `"customAuthorField"`), strings.Index(rawStr, `"latest"`))
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
