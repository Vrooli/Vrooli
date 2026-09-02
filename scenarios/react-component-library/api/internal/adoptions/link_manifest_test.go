package adoptions_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/scheduletest"
	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	"react-component-library/internal/uimanifest"
)

type fakeManifestLoader struct{ manifest uimanifest.Manifest }

func (f fakeManifestLoader) Load(string) (uimanifest.Manifest, error)         { return f.manifest, nil }
func (f fakeManifestLoader) LoadTemplate(string) (uimanifest.Manifest, error) { return f.manifest, nil }

// A template that moves its catalogue, selector registry and entry point must
// see the link land in the declared places, with the registry importing the
// library selectors by the right relative specifier.
func TestLinkWritesToManifestDeclaredFiles(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-button": {
				ID: "cmp-button", LibraryID: "react-component-library:Button", Slug: "Button",
				LatestVersion: "1.2.0", Version: "1.2.0", AssetKind: components.AssetKindComponent,
			},
		},
		body: map[string]string{"cmp-button": "export function Button() { return null }"},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		"moved::ui/package.json":                 []byte(`{"name":"moved-ui","dependencies":{"react":"18.3.1"}}`),
		"moved::ui/src/app/i18n/index.ts":        []byte(`export const i18n = { t: (key: string, options?: { defaultValue?: string }) => options?.defaultValue ?? key };`),
		"moved::ui/src/app/selectors.ts":         []byte(`const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);`),
		"moved::ui/src/app/i18n/catalog/en.json": []byte(`{}`),
		"moved::ui/src/app/entry.tsx":            []byte("import ReactDOM from \"react-dom/client\";\nReactDOM.createRoot(document.body).render(\n  <div />\n  );\n"),
	}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))
	adoptions.SetManifestLoader(svc, fakeManifestLoader{manifest: uimanifest.Manifest{Files: map[string]uimanifest.FileDeclaration{
		"designTokens":     {Path: "ui/src/app/tokens.css"},
		"localeCatalogue":  {Path: "ui/src/app/i18n/catalog/{locale}.json", DefaultLocale: "en"},
		"selectorRegistry": {Path: "ui/src/app/selectors.ts"},
		"librarySelectors": {Path: "ui/src/app/generated/library-selectors.ts"},
		"appEntry":         {Path: "ui/src/app/entry.tsx"},
	}}})

	result, err := svc.Link(context.Background(), adoptions.LinkInput{ComponentID: "cmp-button", Scenario: "moved"})
	require.NoError(t, err)
	require.Contains(t, result.UpdatedFiles, "ui/src/app/generated/library-selectors.ts")
	require.Contains(t, result.UpdatedFiles, "ui/src/app/selectors.ts")
	require.Contains(t, result.UpdatedFiles, "ui/src/app/entry.tsx")
	require.Contains(t, string(files.bytes["moved::ui/src/app/selectors.ts"]), `from "./generated/library-selectors"`)
	require.Contains(t, string(files.bytes["moved::ui/src/app/entry.tsx"]), "LibraryStringsProvider")
	_, defaultRegistry := files.bytes["moved::ui/src/consts/selectors.library.ts"]
	require.False(t, defaultRegistry, "link must not fall back to the react-vite path when the manifest declares another")
	_, defaultEntry := files.bytes["moved::ui/src/main.tsx"]
	require.False(t, defaultEntry)
}
