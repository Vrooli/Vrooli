package adoptions_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
)

func TestLinkAddsGovernedPackageDependencyWithoutCopyingSource(t *testing.T) {
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
		"money-ledger::ui/package.json":                    []byte(`{"name":"money-ledger-ui","dependencies":{"react":"18.3.1"}}`),
		"money-ledger::ui/src/i18n/index.ts":               []byte(`export const i18n = { t: (key: string, options?: { defaultValue?: string }) => options?.defaultValue ?? key };`),
		"money-ledger::ui/src/consts/selectors.library.ts": []byte(`export const librarySelectors = {} as const;`),
		"money-ledger::ui/src/consts/selectors.ts":         []byte(`const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);`),
		"money-ledger::ui/src/i18n/locales/en.json":        []byte(`{}`),
		"money-ledger::ui/src/main.tsx":                    []byte("import ReactDOM from \"react-dom/client\";\nReactDOM.createRoot(document.body).render(\n  <div />\n  );\n"),
	}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))

	result, err := svc.Link(context.Background(), adoptions.LinkInput{ComponentID: "cmp-button", Scenario: "money-ledger"})
	require.NoError(t, err)
	require.Equal(t, adoptions.AdoptionModeLinked, result.Adoption.Mode)
	require.Equal(t, "./Button/1.2.0", result.ImportSubpath)
	require.Contains(t, result.UpdatedFiles, "ui/package.json")
	require.Contains(t, result.UpdatedFiles, "ui/src/consts/selectors.library.ts")
	require.Contains(t, result.UpdatedFiles, "ui/src/consts/selectors.ts")
	require.Contains(t, result.UpdatedFiles, "ui/src/main.tsx")
	require.Contains(t, string(files.bytes["money-ledger::ui/src/main.tsx"]), "LibraryStringsProvider")
	require.Contains(t, string(files.bytes["money-ledger::ui/src/main.tsx"]), "i18n.t")
	require.NotContains(t, string(files.bytes["money-ledger::ui/src/i18n/index.ts"]), "vrooli:library-locale-bridge")
	require.Contains(t, string(files.bytes["money-ledger::ui/src/consts/selectors.library.ts"]), "librarySelectors")
	_, exists := files.bytes["money-ledger::ui/src/components/Button.tsx"]
	require.False(t, exists, "link must not copy a source file")

	var manifest map[string]any
	require.NoError(t, json.Unmarshal(files.bytes["money-ledger::ui/package.json"], &manifest))
	dependencies := manifest["dependencies"].(map[string]any)
	require.Equal(t, "file:../../../packages/react-component-library", dependencies["@vrooli/react-component-library"])
}

func TestEjectRequiresReason(t *testing.T) {
	svc := adoptions.NewService(adoptmocks.NewFakeRepository(), &fakeLibrary{}, &fakeFiles{bytes: map[string][]byte{}}, scheduletest.New(time.Unix(0, 0)))
	_, err := svc.Eject(context.Background(), adoptions.EjectInput{})
	require.ErrorContains(t, err, "reason")
}
