package libspec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoSecondGrammar(t *testing.T) {
	root, err := filepath.Abs("../../../../../")
	require.NoError(t, err)
	count := 0
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "dist" || entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".mjs" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), "react-component-library/([A-Za-z][A-Za-z0-9-]*)(?:") {
			count++
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, count, "only the shared Go grammar and JS binding may define the parser regex")
}

func TestParseAllUsesOneGrammar(t *testing.T) {
	got := Sorted(ParseAll(`import "@vrooli/react-component-library/Button"; import "@vrooli/react-component-library/Panel/2"; import "@vrooli/react-component-library/Panel/2.1.0"`))
	require.Equal(t, []Specifier{{Name: "Button"}, {Name: "Panel", Selector: "2"}, {Name: "Panel", Selector: "2.1.0"}}, got)
	_, ok, err := Parse("@vrooli/react-component-library/Panel/2.1")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestWalkHonoursAssetScopeAndExclusions(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"Button/Button.tsx", "Other/Other.tsx", "Button/node_modules/hidden.tsx", ".retired/old.tsx"} {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, path), nil, 0o644))
	}
	paths := []string{}
	require.NoError(t, Walk(root, Scope{Assets: map[string]bool{"Button": true}}, func(path string) error { paths = append(paths, path); return nil }))
	require.Len(t, paths, 1)
	require.Equal(t, filepath.Join(root, "Button/Button.tsx"), paths[0])
}
