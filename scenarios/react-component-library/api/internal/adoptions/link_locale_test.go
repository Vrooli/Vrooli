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

// stringsLibrary adds a `.strings.ts` companion to the fake library.
type stringsLibrary struct {
	*fakeLibrary
	companions map[string]string // path -> body
}

func (l *stringsLibrary) GetVersionContentAt(_ context.Context, _, _, path string) (components.Content, error) {
	body, ok := l.companions[path]
	if !ok {
		return components.Content{}, components.ErrComponentNotFound{IDOrLibraryID: path}
	}
	return components.Content{Body: body, SourcePath: path}, nil
}

func TestLinkMergesOnlyStringsCompanionIntoLocaleCatalogue(t *testing.T) {
	repo := adoptmocks.NewFakeRepository()
	lib := &stringsLibrary{
		fakeLibrary: &fakeLibrary{
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
					Content: "const label = { \"variant\": \"primary\" };",
					Files: []components.ComponentVersionFile{
						{Path: "story.json", Content: `{"$schema": "../story.schema.json", "id": "default", "title": "Button 2.2"}`},
						{Path: "dependencies.json", Content: `{"libraryId": "react-component-library:Button", "version": "1.2.0"}`},
					},
				},
			},
		},
		companions: map[string]string{
			"Button.strings.ts": `export const ButtonStrings = defineStrings("react-component-library:Button", {
  "controls.button.loading": "Working…",
});`,
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		"target::ui/package.json":                    []byte(`{"name":"target-ui","dependencies":{}}`),
		"target::ui/src/i18n/locales/en.json":        []byte("{\n  \"app\": {\n    \"title\": \"Target\"\n  }\n}\n"),
		"target::ui/src/consts/selectors.library.ts": []byte(`export const librarySelectors = {} as const;`),
		"target::ui/src/consts/selectors.ts":         []byte(`const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);`),
		"target::ui/src/main.tsx":                    []byte("import ReactDOM from \"react-dom/client\";\nReactDOM.createRoot(document.body).render(<div />);\n"),
		"target::ui/src/design-tokens.css":           []byte(":root { /* rcl:tokens:begin */ /* rcl:tokens:end */ }"),
	}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))

	_, err := svc.Link(context.Background(), adoptions.LinkInput{ComponentID: "cmp-button", Scenario: "target", Version: "1.2.0"})
	require.NoError(t, err)

	var catalogue map[string]any
	require.NoError(t, json.Unmarshal(files.bytes["target::ui/src/i18n/locales/en.json"], &catalogue))
	require.Equal(t, map[string]any{
		"app":      map[string]any{"title": "Target"},
		"controls": map[string]any{"button": map[string]any{"loading": "Working…"}},
	}, catalogue, "only defineStrings entries may be merged; story.json, dependencies.json and source literals must not leak")

	selectors := string(files.bytes["target::ui/src/consts/selectors.ts"])
	require.Contains(t, selectors, "createSelectorRegistry({ library: librarySelectors, ...literalSelectors }, dynamicSelectorDefinitions);")
	require.NotContains(t, selectors, ",,")
}
