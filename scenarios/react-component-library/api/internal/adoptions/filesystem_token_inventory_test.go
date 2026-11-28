package adoptions_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
)

func TestFSScenarioFileReaderDerivesTokenInventoryFromConsumerSources(t *testing.T) {
	root := t.TempDir()
	write := func(relative, content string) {
		t.Helper()
		path := filepath.Join(root, "demo", "ui", filepath.FromSlash(relative))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	write("tailwind.config.ts", `export default { theme: { colors: { slate: {} } } }`)
	write("styles/tokens.css", `:root { --color-primary: #fff; --color-surface: #000; }`)
	write("src/theme.ts", `document.documentElement.style.setProperty('--color-runtime', '#fff');`)
	write("src/panel.tsx", `import { Button } from "@vrooli/react-component-library/Button"; import { Card } from "@vrooli/react-component-library/Card/2"; import { Button as Again } from "@vrooli/react-component-library/Button";`)
	write("node_modules/ignored.css", `:root { --ignored-token: red; }`)

	reader := adoptions.NewFSScenarioFileReader(root)
	namespace, err := reader.TokenNamespace(context.Background(), "demo")
	require.NoError(t, err)
	require.Equal(t, "slate", namespace)
	declared, err := reader.DeclaredTokens(context.Background(), "demo")
	require.NoError(t, err)
	require.Equal(t, []string{"--color-primary", "--color-runtime", "--color-surface"}, declared)
	written, err := reader.RuntimeWrittenTokens(context.Background(), "demo")
	require.NoError(t, err)
	require.Equal(t, []string{"--color-runtime"}, written)
	imports, err := reader.ImportedLibrarySpecifiers(context.Background(), "demo")
	require.NoError(t, err)
	require.Equal(t, []components.LibraryPackageSpecifier{{Name: "Button"}, {Name: "Card", RequestedVersion: "2"}}, imports)
}

func TestFSScenarioFileReaderTokenNamespaceDefaultsAndHonorsCancellation(t *testing.T) {
	root := t.TempDir()
	reader := adoptions.NewFSScenarioFileReader(root)
	namespace, err := reader.TokenNamespace(context.Background(), "missing")
	require.NoError(t, err)
	require.Equal(t, "app", namespace)

	path := filepath.Join(root, "demo", "ui", "tailwind.config.ts")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(`export default { theme: { colors: { wc: {}, slate: {} } } }`), 0o600))
	namespace, err = reader.TokenNamespace(context.Background(), "demo")
	require.NoError(t, err)
	require.Equal(t, "wc", namespace, "the explicit wc namespace takes precedence when config mentions both aliases")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = reader.DeclaredTokens(ctx, "demo")
	require.ErrorIs(t, err, context.Canceled)
}

func TestFSScenarioFileReaderImportedSpecifiersAreScenarioScopedAndDeduplicated(t *testing.T) {
	root := t.TempDir()
	for scenario, source := range map[string]string{
		"demo":  `import { A } from "@vrooli/react-component-library/A"; import { B } from "@vrooli/react-component-library/B/1";`,
		"other": `import { C } from "@vrooli/react-component-library/C";`,
	} {
		path := filepath.Join(root, scenario, "ui", "src", "index.tsx")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
	}
	got, err := adoptions.NewFSScenarioFileReader(root).ImportedLibrarySpecifiers(context.Background(), "demo")
	require.NoError(t, err)
	want := []components.LibraryPackageSpecifier{{Name: "A"}, {Name: "B", RequestedVersion: "1"}}
	require.Equal(t, want, got)
}
