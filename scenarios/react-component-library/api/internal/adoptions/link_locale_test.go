package adoptions_test

import (
	"context"
	"fmt"
	"os"
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
		return components.Content{}, fmt.Errorf("read component companion: %w", &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist})
	}
	return components.Content{Body: body, SourcePath: path}, nil
}

func TestLinkLeavesLocaleCatalogueToTheAdopter(t *testing.T) {
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
			"selectors.json": `{"primaryAction":"real-button-id"}`,
			"Button.strings.ts": `export const ButtonStrings = defineStrings("react-component-library:Button", {
  "controls.button.loading": "Working…",
});`,
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{
		"target::ui/package.json":                    []byte(`{"name":"target-ui","dependencies":{"@vrooli/ui-selectors":"file:../../../packages/ui-selectors"}}`),
		"target::ui/src/i18n/locales/en.json":        []byte("{\n  \"app\": {\n    \"title\": \"Target\"\n  }\n}\n"),
		"target::ui/src/consts/selectors.library.ts": []byte(`export const librarySelectors = {} as const;`),
		"target::ui/src/consts/selectors.ts": []byte(`import { createSelectorRegistry } from "@vrooli/ui-selectors";
const registry = createSelectorRegistry({ library: librarySelectors, ...literalSelectors }, dynamicSelectorDefinitions, librarySelectors);`),
		"target::ui/src/main.tsx":          []byte("import ReactDOM from \"react-dom/client\";\nReactDOM.createRoot(document.body).render(<div />);\n"),
		"target::ui/src/design-tokens.css": []byte(":root { /* rcl:tokens:begin */ /* rcl:tokens:end */ }"),
	}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))

	_, err := svc.Link(context.Background(), adoptions.LinkInput{ComponentID: "cmp-button", Scenario: "target", Version: "1.2.0"})
	require.NoError(t, err)

	// Library strings resolve through LibraryStringsProvider with their
	// defineStrings defaults; the adopter's catalogue is never rewritten.
	require.Equal(t, "{\n  \"app\": {\n    \"title\": \"Target\"\n  }\n}\n", string(files.bytes["target::ui/src/i18n/locales/en.json"]),
		"linking must not copy library defaults (or story/dependency fields) into the adopter's catalogue")
	require.Contains(t, string(files.bytes["target::ui/src/main.tsx"]), "<LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>")

	selectors := string(files.bytes["target::ui/src/consts/selectors.ts"])
	require.Contains(t, selectors, "createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);")
	require.NotContains(t, selectors, ",,")
	require.Contains(t, string(files.bytes["target::ui/src/consts/selectors.library.ts"]), `"primaryAction": "real-button-id"`)
	first := string(files.bytes["target::ui/src/consts/selectors.library.ts"])
	_, err = svc.Link(context.Background(), adoptions.LinkInput{ComponentID: "cmp-button", Scenario: "target", Version: "1.2.0"})
	require.NoError(t, err)
	require.Equal(t, first, string(files.bytes["target::ui/src/consts/selectors.library.ts"]), "versioned selector contract must link idempotently")
}
