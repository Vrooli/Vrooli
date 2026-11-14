package components

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncStyleSheetVersionRewritesOnlyTheOwnIdentity(t *testing.T) {
	source := `/** @libraryId react-component-library:FullPageDrawer */
useLibraryStyleSheet("react-component-library:FullPageDrawer", "1.4.3", styles);
useLibraryStyleSheet("react-component-library:Portal", "1.1.2", portalStyles);
const a = <StyleSheet libraryId="react-component-library:FullPageDrawer" version="1.4.3" css={styles} />;
const b = <StyleSheet
  version={"1.4.3"}
  libraryId="react-component-library:FullPageDrawer"
  css={styles}
/>;
const c = <StyleSheet libraryId="react-component-library:Icon" version="1.0.0" css={iconStyles} />;
const d = <div data-version="1.4.3" />;`
	got := SyncStyleSheetVersion(source, "react-component-library:FullPageDrawer", "1.5.1")

	literals := SelfStyleSheetVersions(got, "react-component-library:FullPageDrawer")
	if len(literals) != 3 {
		t.Fatalf("expected three own-identity literals, got %#v", literals)
	}
	for _, literal := range literals {
		if literal.Version != "1.5.1" {
			t.Fatalf("own literal not rewritten: %#v\n%s", literal, got)
		}
	}
	for _, untouched := range []string{`"react-component-library:Portal", "1.1.2"`, `libraryId="react-component-library:Icon" version="1.0.0"`, `data-version="1.4.3"`} {
		if !strings.Contains(got, untouched) {
			t.Fatalf("rewrote another identity or attribute (%s):\n%s", untouched, got)
		}
	}
}

// A version written by the store carries its own version in its stylesheet
// key, whatever the source it was copied from passed.
func TestCreateVersionStampsTheStylesheetKeyVersion(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "components", "Drawer")
	require.NoError(t, os.MkdirAll(filepath.Join(base, "versions", "1.4.3"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "versions", "1.4.3", "Drawer.tsx"),
		[]byte(`export const Drawer = () => { useLibraryStyleSheet("react-component-library:Drawer", "1.4.3", css); return null; };`+"\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(base, "component.json"), []byte(`{"libraryId":"react-component-library:Drawer","displayName":"Drawer","latest":"1.4.3","draft":"","deprecatedVersions":[]}`), 0o600))
	store := NewFSContentStore(root)
	c := Component{
		Slug: "Drawer", LibraryID: "react-component-library:Drawer", DisplayName: "Drawer", LatestVersion: "1.4.3",
		ManifestPath: "components/Drawer/component.json", SourcePath: "components/Drawer/versions/1.4.3/Drawer.tsx",
	}
	// The new version is copied from 1.4.3, the way draft-begin copies.
	_, err := store.CreateVersion(context.Background(), c, CreateComponentVersionInput{Version: "1.5.1", FileName: "Drawer.tsx"})
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(base, "versions", "1.5.1", "Drawer.tsx"))
	require.NoError(t, err)
	literals := SelfStyleSheetVersions(string(raw), "react-component-library:Drawer")
	require.Len(t, literals, 1)
	require.Equal(t, "1.5.1", literals[0].Version)
}

func TestStampStyleSheetVersionsCoversCompanionIdentities(t *testing.T) {
	source := `/**
 * @libraryId react-component-library:code-block
 */
export const CodeBlock = () => <StyleSheet libraryId="react-component-library:code-block" version="0.4.0" css={styles} />;`
	got := StampStyleSheetVersions(source, "react-component-library:markdown-renderer", "0.5.1")
	literals := SelfStyleSheetVersions(got, "react-component-library:code-block")
	require.Len(t, literals, 1)
	require.Equal(t, "0.5.1", literals[0].Version)
}
