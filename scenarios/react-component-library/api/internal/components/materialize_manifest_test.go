package components_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
)

// Restoring an evicted release must also take it off the manifest's
// evictedVersions list. The package export map skips listed versions, so a
// release restored on disk but still listed stays unexportable and the
// imports that asked for it remain broken.
func TestEnsureMaterializedClearsTheEvictedMarker(t *testing.T) {
	d, repo, _ := newComponentsRawDB(t)
	ctx := context.Background()
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	}
	write("components/Button/component.json", manifest("react-component-library:Button", "Button", `[]`))
	write("components/Button/versions/0.9.0/Button.tsx", buttonTSX)
	write("components/Button/versions/1.0.0/Button.tsx", buttonTSX)
	_, err := components.NewIndexer(repo, root, nil).Run(ctx)
	require.NoError(t, err)
	component, err := repo.GetByLibraryID(ctx, "react-component-library:Button")
	require.NoError(t, err)

	// Evict 0.9.0 the way the archive transition leaves it.
	_, err = d.ExecContext(ctx, `UPDATE component_versions SET presence='evicted' WHERE component_id=? AND version='0.9.0'`, component.ID)
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(filepath.Join(root, "components/Button/versions/0.9.0")))
	var doc map[string]any
	raw, err := os.ReadFile(filepath.Join(root, "components/Button/component.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &doc))
	// 0.8.0 is a release kept on disk as recovery source while deliberately
	// unexported; restoring 0.9.0 must not re-export it.
	write("components/Button/versions/0.8.0/Button.tsx", buttonTSX)
	doc["evictedVersions"] = []string{"0.8.0", "0.9.0"}
	raw, err = json.Marshal(doc)
	require.NoError(t, err)
	write("components/Button/component.json", string(raw))

	materializer, ok := components.NewServiceWithContent(repo, components.NewFSContentStore(root)).(components.Materializer)
	require.True(t, ok)
	_, err = materializer.EnsureMaterialized(ctx, component.ID, "0.9.0", "")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(root, "components/Button/versions/0.9.0/Button.tsx"))
	require.NoError(t, err, "the release is back on disk")
	raw, err = os.ReadFile(filepath.Join(root, "components/Button/component.json"))
	require.NoError(t, err)
	var after struct {
		Latest          string   `json:"latest"`
		EvictedVersions []string `json:"evictedVersions"`
	}
	require.NoError(t, json.Unmarshal(raw, &after))
	require.Equal(t, []string{"0.8.0"}, after.EvictedVersions, "only the restored release leaves the cold list")
	require.Equal(t, "1.0.0", after.Latest, "the version pointers are preserved")
	var afterDoc map[string]any
	require.NoError(t, json.Unmarshal(raw, &afterDoc))
	delete(doc, "evictedVersions")
	delete(afterDoc, "evictedVersions")
	require.Equal(t, doc, afterDoc, "every other manifest field is left as it was")
}
