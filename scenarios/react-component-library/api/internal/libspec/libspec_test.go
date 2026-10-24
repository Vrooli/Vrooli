package libspec

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoSecondGrammar(t *testing.T) {
	root, err := filepath.Abs("../../../../../")
	require.NoError(t, err)
	require.NoError(t, assertGrammarOwners(root, os.ReadFile))
}

func TestGrammarInventoryRecognizesGoAndEscapedJavaScript(t *testing.T) {
	root := grammarFixture(t)

	owners, err := grammarOwners(root, os.ReadFile)
	require.NoError(t, err)
	require.Equal(t, expectedGrammarOwners, owners)
	require.NoError(t, assertGrammarOwners(root, os.ReadFile))
}

func TestGrammarInventoryRejectsDuplicateActiveDefinition(t *testing.T) {
	root := grammarFixture(t)
	writeGrammarFixture(t, root, "packages/other/libspec.go", goGrammarFixture)

	err := assertGrammarOwners(root, os.ReadFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "packages/other/libspec.go")
}

func TestGrammarInventorySkipsArchivedCopiesBeforeReading(t *testing.T) {
	root := grammarFixture(t)
	archivedPath := filepath.Join("packages", "react-component-library", "archive", "old", "libspec.go")
	writeGrammarFixture(t, root, archivedPath, goGrammarFixture)
	var readPaths []string
	readFile := func(path string) ([]byte, error) {
		readPaths = append(readPaths, path)
		return os.ReadFile(path)
	}

	require.NoError(t, assertGrammarOwners(root, readFile))
	require.NotContains(t, readPaths, filepath.Join(root, archivedPath))
}

var expectedGrammarOwners = []string{
	"packages/react-component-library/libspec/libspec.go",
	"packages/react-component-library/tooling/libspec.mjs",
}

const goGrammarFixture = "var grammar = regexp.MustCompile(`^@vrooli/react-component-library/([A-Za-z][A-Za-z0-9-]*)(?:/(\\d+))?`)"
const jsGrammarFixture = "const grammar = /^@vrooli\\/react-component-library\\/([A-Za-z][A-Za-z0-9-]*)(?:\\/(\\d+))?$/;"

func grammarFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeGrammarFixture(t, root, "packages/react-component-library/libspec/libspec.go", goGrammarFixture)
	writeGrammarFixture(t, root, "packages/react-component-library/tooling/libspec.mjs", jsGrammarFixture)
	return root
}

func writeGrammarFixture(t *testing.T, root, path, source string) {
	t.Helper()
	fullPath := filepath.Join(root, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(source), 0o644))
}

func assertGrammarOwners(root string, readFile func(string) ([]byte, error)) error {
	owners, err := grammarOwners(root, readFile)
	if err != nil {
		return err
	}
	if strings.Join(owners, "\n") != strings.Join(expectedGrammarOwners, "\n") {
		return fmt.Errorf("expected active grammar owners %v, found %v", expectedGrammarOwners, owners)
	}
	return nil
}

func grammarOwners(root string, readFile func(string) ([]byte, error)) ([]string, error) {
	packagesRoot := filepath.Join(root, "packages")
	var owners []string
	err := filepath.WalkDir(packagesRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if isExcludedGrammarDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".mjs" {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".test.mjs") {
			return nil
		}
		data, readErr := readFile(path)
		if readErr != nil {
			return readErr
		}
		matches := strings.Count(normalizeGrammarSource(string(data)), grammarDefinitionMarker())
		for range matches {
			relative, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			owners = append(owners, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(owners)
	return owners, nil
}

func isExcludedGrammarDirectory(name string) bool {
	switch name {
	case ".git", "archive", "archives", "baseline", "baseline-original", "dist", "evidence", "node_modules":
		return true
	default:
		return false
	}
}

func normalizeGrammarSource(source string) string {
	return strings.ReplaceAll(source, `\/`, "/")
}

func grammarDefinitionMarker() string {
	return strings.Join([]string{"^@vrooli", "/react-component-library/", "([A-Za-z][A-Za-z0-9-]*)", "(?:/"}, "")
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
