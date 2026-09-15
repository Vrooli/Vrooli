package discovery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"brand-manager/internal/discovery"

	"github.com/stretchr/testify/require"
)

func TestFSScanner_ReadsAndListsUnderScenarioRoot(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "web-console", "ui", "public")
	require.NoError(t, os.MkdirAll(scenarioDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(scenarioDir, "logo.png"), []byte("PNG"), 0o600))

	sc := discovery.NewFSScanner(root)
	ctx := context.Background()

	exists, err := sc.ScenarioExists(ctx, "web-console")
	require.NoError(t, err)
	require.True(t, exists)

	missing, err := sc.ScenarioExists(ctx, "ghost")
	require.NoError(t, err)
	require.False(t, missing)

	data, err := sc.ReadFile(ctx, "web-console", "ui/public/logo.png")
	require.NoError(t, err)
	require.Equal(t, []byte("PNG"), data)

	// A missing file is (nil, nil), never an error.
	absent, err := sc.ReadFile(ctx, "web-console", "ui/public/nope.png")
	require.NoError(t, err)
	require.Nil(t, absent)

	entries, err := sc.ListDir(ctx, "web-console", "ui/public")
	require.NoError(t, err)
	require.Equal(t, []string{"logo.png"}, entries)
}

func TestFSScanner_RejectsEscapingPaths(t *testing.T) {
	sc := discovery.NewFSScanner(t.TempDir())
	ctx := context.Background()

	_, err := sc.ReadFile(ctx, "web-console", "../../etc/passwd")
	var invalid discovery.ErrInvalidDiscovery
	require.ErrorAs(t, err, &invalid)

	_, err = sc.ScenarioExists(ctx, "../other")
	require.ErrorAs(t, err, &invalid)

	_, err = sc.ScenarioExists(ctx, "")
	require.ErrorAs(t, err, &invalid)
}
