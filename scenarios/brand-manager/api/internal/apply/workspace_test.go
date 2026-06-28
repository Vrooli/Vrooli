package apply_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"brand-manager/internal/apply"

	"github.com/stretchr/testify/require"
)

func TestFSWorkspace_RoundTrip(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "web-console"), 0o750))
	ws := apply.NewFSWorkspace(root)
	ctx := context.Background()

	exists, err := ws.ScenarioExists(ctx, "web-console")
	require.NoError(t, err)
	require.True(t, exists)

	missing, err := ws.ScenarioExists(ctx, "ghost")
	require.NoError(t, err)
	require.False(t, missing)

	// Reading an absent file yields (nil, nil) so callers can merge-or-create.
	data, err := ws.ReadFile(ctx, "web-console", "ui/public/manifest.json")
	require.NoError(t, err)
	require.Nil(t, data)

	require.NoError(t, ws.WriteFile(ctx, "web-console", "ui/src/styles/brand.css", []byte(":root{}")))
	got, err := ws.ReadFile(ctx, "web-console", "ui/src/styles/brand.css")
	require.NoError(t, err)
	require.Equal(t, ":root{}", string(got))

	// The file really landed under the scenario tree.
	onDisk, err := os.ReadFile(filepath.Join(root, "web-console", "ui/src/styles/brand.css"))
	require.NoError(t, err)
	require.Equal(t, ":root{}", string(onDisk))
}

func TestFSWorkspace_RejectsEscapingPaths(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "web-console"), 0o750))
	ws := apply.NewFSWorkspace(root)
	ctx := context.Background()

	// A relative path that climbs out of the scenario dir is rejected.
	err := ws.WriteFile(ctx, "web-console", "../../escape.txt", []byte("nope"))
	var invalid apply.ErrInvalidApply
	require.ErrorAs(t, err, &invalid)

	// A scenario name with separators is rejected.
	_, err = ws.ScenarioExists(ctx, "../etc")
	require.ErrorAs(t, err, &invalid)
}
