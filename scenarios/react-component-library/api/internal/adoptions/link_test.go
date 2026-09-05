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

func TestLinkSynchronisesMissingConsumerTokensBeforeRecordingAdoption(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-button": {
				ID: "cmp-button", LibraryID: "react-component-library:Button", Slug: "Button",
				LatestVersion: "1.2.0", Version: "1.2.0", AssetKind: components.AssetKindComponent,
			},
		},
		body: map[string]string{"cmp-button": "export function Button() { return null }"},
		versions: map[string]components.ComponentVersion{
			"cmp-button@1.2.0": {
				ComponentID: "cmp-button", LibraryID: "react-component-library:Button", Version: "1.2.0",
				Status: components.VersionStatusReleased, SourcePath: "Button.tsx",
				RequiredTokens: []string{"--space-sm"},
			},
		},
	}
	baseFiles := &fakeFiles{bytes: map[string][]byte{
		"target::ui/package.json":                    []byte(`{"name":"target-ui","dependencies":{}}`),
		"target::ui/src/i18n/locales/en.json":        []byte(`{}`),
		"target::ui/src/consts/selectors.library.ts": []byte(`export const librarySelectors = {} as const;`),
		"target::ui/src/consts/selectors.ts":         []byte(`const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);`),
		"target::ui/src/main.tsx":                    []byte("import ReactDOM from \"react-dom/client\";\nReactDOM.createRoot(document.body).render(<div />);\n"),
		"target::ui/src/design-tokens.css":           []byte(":root { /* rcl:tokens:begin */ /* rcl:tokens:end */ }"),
	}}
	files := &importedRampFiles{fakeFiles: baseFiles, imports: []components.LibraryPackageSpecifier{{Name: "Button", RequestedVersion: "1.2.0"}}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))
	adoptions.SetTokenNamespaceReader(svc, rampTokenInventory{files: baseFiles})

	result, err := svc.Link(context.Background(), adoptions.LinkInput{ComponentID: "cmp-button", Scenario: "target", Version: "1.2.0"})
	require.NoError(t, err)
	require.Contains(t, result.UpdatedFiles, "ui/src/design-tokens.css")
	require.Contains(t, string(baseFiles.bytes["target::ui/src/design-tokens.css"]), "--space-sm")
	rows, listErr := repo.List(context.Background(), adoptions.ListQuery{Scenario: "target", Limit: 10})
	require.NoError(t, listErr)
	require.Len(t, rows, 1)
}

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
