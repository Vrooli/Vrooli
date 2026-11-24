package adoptions_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"react-component-library/internal/adoptions"
)

func TestFSScenarioFileReaderFindsAndRewritesLocalImportSites(t *testing.T) {
	root := t.TempDir()
	reader := adoptions.NewFSScenarioFileReader(root)
	write := func(relative, source string) {
		t.Helper()
		path := filepath.Join(root, "demo", filepath.FromSlash(relative))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
	}
	write("ui/src/components/Button.tsx", `export const Button = () => null;`)
	write("ui/src/pages/One.tsx", `import { Button } from "../components/Button"; export { Button };`)
	write("ui/src/pages/Two.ts", `const Button = require('../components/Button');`)
	write("ui/src/pages/Unrelated.tsx", `import { Card } from "../components/Card";`)
	write("ui/src/node_modules/example/index.tsx", `import { Button } from "../components/Button";`)

	ctx := context.Background()
	sites, err := reader.FindImportSites(ctx, "demo", "ui/src/components/Button.tsx")
	require.NoError(t, err)
	require.Equal(t, []string{"ui/src/pages/One.tsx", "ui/src/pages/Two.ts"}, sites)

	updated, err := reader.RewriteImportSites(ctx, "demo", "ui/src/components/Button.tsx", "@app/design/Button")
	require.NoError(t, err)
	require.Equal(t, sites, updated)
	for _, site := range sites {
		body, readErr := os.ReadFile(filepath.Join(root, "demo", filepath.FromSlash(site)))
		require.NoError(t, readErr)
		require.Contains(t, string(body), `@app/design/Button`)
		require.NotContains(t, string(body), "../components/Button")
	}
}

func TestFSScenarioFileReaderUpdatesVersionedLibraryImportFamily(t *testing.T) {
	root := t.TempDir()
	reader := adoptions.NewFSScenarioFileReader(root)
	write := func(relative, source string) {
		t.Helper()
		path := filepath.Join(root, "demo", filepath.FromSlash(relative))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
	}
	write("ui/src/one.tsx", `import { Button } from "@vrooli/react-component-library/Button/1.0.0";`)
	write("ui/src/two.tsx", `import { Button } from '@vrooli/react-component-library/Button/0.9.0';`)
	write("ui/src/other.tsx", `import { Card } from "@vrooli/react-component-library/Card/1.0.0";`)
	write("ui/src/node_modules/ignored.tsx", `import { Button } from "@vrooli/react-component-library/Button/1.0.0";`)

	updated, err := reader.RewriteImportSites(context.Background(), "demo", "ui/src/components/Button.tsx", "@vrooli/react-component-library/Button/2.0.0")
	require.NoError(t, err)
	require.Equal(t, []string{"ui/src/one.tsx", "ui/src/two.tsx"}, updated)
	for _, relative := range updated {
		body, readErr := os.ReadFile(filepath.Join(root, "demo", filepath.FromSlash(relative)))
		require.NoError(t, readErr)
		require.Contains(t, string(body), `@vrooli/react-component-library/Button/2.0.0`)
		require.NotContains(t, string(body), `@vrooli/react-component-library/Button/1.0.0`)
	}
	card, err := os.ReadFile(filepath.Join(root, "demo", "ui", "src", "other.tsx"))
	require.NoError(t, err)
	require.Contains(t, string(card), `@vrooli/react-component-library/Card/1.0.0`)
}

func TestFSScenarioFileReaderRemoveIsIdempotentAndRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	reader := adoptions.NewFSScenarioFileReader(root)
	path := filepath.Join(root, "demo", "ui", "src", "temporary.tsx")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("temporary"), 0o600))
	ctx := context.Background()
	require.NoError(t, reader.Remove(ctx, "demo", "ui/src/temporary.tsx"))
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, reader.Remove(ctx, "demo", "ui/src/temporary.tsx"), "repeat remove should remain idempotent")
	require.ErrorContains(t, reader.Remove(ctx, "demo", "../../outside.tsx"), "escapes scenario root")
}
