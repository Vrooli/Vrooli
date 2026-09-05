package components_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
	"react-component-library/internal/components/mocks"
)

func TestService_UpsertRejectsBlankLibraryID(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)

	_, err := svc.Upsert(context.Background(), components.UpsertInput{LibraryID: "   "})
	var bad components.ErrInvalidHeader
	require.True(t, errors.As(err, &bad))
	require.Equal(t, int64(0), repo.UpsertCalls.Load())
}

func TestPublishFreezesBareLibrarySpecifierAndDependencyLock(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	for _, asset := range []string{"Stack", "Panel"} {
		_, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
			LibraryID: "react-component-library:" + asset, Slug: asset, DisplayName: asset,
			InitialVersion: "1.0.0", InitialSource: "export function " + asset + "() { return null; }", ScaffoldExamples: true,
		})
		require.NoError(t, err)
	}
	authoring := svc.(components.AuthoringService)
	draft, err := authoring.BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{Component: "react-component-library:Panel", Bump: "patch"})
	require.NoError(t, err)
	writer := svc.(interface {
		UpdateVersionContentAt(context.Context, string, string, string, components.WriteContentInput) (components.Content, error)
	})
	_, err = writer.UpdateVersionContentAt(context.Background(), "react-component-library:Panel", draft.Version.Version, "Panel.tsx", components.WriteContentInput{
		Body: `import { Stack } from "@vrooli/react-component-library/Stack"; export function Panel() { return <Stack />; }`,
	})
	require.NoError(t, err)
	released, err := authoring.PublishComponentVersion(context.Background(), components.PublishComponentVersionInput{Component: "react-component-library:Panel"})
	require.NoError(t, err)

	source, err := os.ReadFile(filepath.Join(root, "components", "Panel", "versions", released.Version.Version, "Panel.tsx"))
	require.NoError(t, err)
	require.Contains(t, string(source), "@vrooli/react-component-library/Stack/1")
	lock, err := os.ReadFile(filepath.Join(root, "components", "Panel", "versions", released.Version.Version, "dependencies.json"))
	require.NoError(t, err)
	require.JSONEq(t, `{"schemaVersion":2,"libraryId":"react-component-library:Panel","version":"1.0.1","resolvedAt":"`+extractResolvedAt(t, lock)+`","dependencies":[{"libraryId":"react-component-library:Stack","major":1,"observed":"1.0.0","rank":4}]}`, string(lock))
	ledger, err := os.ReadFile(filepath.Join(root, "released-version-hashes.json"))
	require.NoError(t, err)
	require.Contains(t, string(ledger), "components/Panel/versions/1.0.1/Panel.tsx")
}

func TestPublishPreservesExplicitLibraryPinAndOlderReleaseBytes(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	for _, asset := range []string{"Stack", "Panel"} {
		_, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
			LibraryID: "react-component-library:" + asset, Slug: asset, DisplayName: asset,
			InitialVersion: "1.0.0", InitialSource: "export function " + asset + "() { return null; }", ScaffoldExamples: true,
		})
		require.NoError(t, err)
	}
	authoring := svc.(components.AuthoringService)
	draft, err := authoring.BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{Component: "react-component-library:Panel", Bump: "patch"})
	require.NoError(t, err)
	writer := svc.(interface {
		UpdateVersionContentAt(context.Context, string, string, string, components.WriteContentInput) (components.Content, error)
	})
	_, err = writer.UpdateVersionContentAt(context.Background(), "react-component-library:Panel", draft.Version.Version, "Panel.tsx", components.WriteContentInput{
		Body: `import { Stack } from "@vrooli/react-component-library/Stack/1.0.0"; export function Panel() { return <Stack />; }`,
	})
	require.NoError(t, err)
	_, err = authoring.PublishComponentVersion(context.Background(), components.PublishComponentVersionInput{Component: "react-component-library:Panel"})
	require.NoError(t, err)
	panelBefore, err := os.ReadFile(filepath.Join(root, "components", "Panel", "versions", "1.0.1", "Panel.tsx"))
	require.NoError(t, err)
	require.Contains(t, string(panelBefore), "Stack/1")

	stackDraft, err := authoring.BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{Component: "react-component-library:Stack", Bump: "patch"})
	require.NoError(t, err)
	_, err = authoring.PublishComponentVersion(context.Background(), components.PublishComponentVersionInput{Component: "react-component-library:Stack", DraftVersion: stackDraft.Version.Version})
	require.NoError(t, err)
	panelAfter, err := os.ReadFile(filepath.Join(root, "components", "Panel", "versions", "1.0.1", "Panel.tsx"))
	require.NoError(t, err)
	require.Equal(t, panelBefore, panelAfter)
}

func extractResolvedAt(t *testing.T, raw []byte) string {
	t.Helper()
	var value struct {
		ResolvedAt string `json:"resolvedAt"`
	}
	require.NoError(t, json.Unmarshal(raw, &value))
	return value.ResolvedAt
}

func TestInitializeComponentCreatesDependencyLockBeforeIndexing(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))

	created, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
		LibraryID: "react-component-library:ViewportEnvironment",
		Slug:      "ViewportEnvironment", DisplayName: "Viewport Environment",
		InitialVersion: "1.0.0", InitialSource: "export function useViewportEnvironment() { return null; }",
		ScaffoldExamples: true,
	})
	require.NoError(t, err)

	lock, err := os.ReadFile(filepath.Join(root, "components", "ViewportEnvironment", "versions", "1.0.0", "dependencies.json"))
	require.NoError(t, err)
	require.JSONEq(t, `{"schemaVersion":2,"libraryId":"react-component-library:ViewportEnvironment","version":"1.0.0","resolvedAt":"`+extractResolvedAt(t, lock)+`","dependencies":[]}`, string(lock))
	require.Equal(t, "1.0.0", created.Component.LatestVersion)
}

func TestService_UpdateVersionContentAtFormatsDraftJSONWithoutMutatingRelease(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	_, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
		LibraryID: "react-component-library:Button", Slug: "Button", DisplayName: "Button",
		InitialVersion: "1.0.0", InitialSource: "export function Button() { return <button />; }",
		ScaffoldExamples: true,
	})
	require.NoError(t, err)
	authoring := svc.(components.AuthoringService)
	draft, err := authoring.BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{
		Component: "react-component-library:Button", Bump: "patch",
	})
	require.NoError(t, err)
	writer := svc.(interface {
		UpdateVersionContentAt(context.Context, string, string, string, components.WriteContentInput) (components.Content, error)
	})
	written, err := writer.UpdateVersionContentAt(context.Background(), "react-component-library:Button", draft.Version.Version, "story.json", components.WriteContentInput{
		Body: `{"schemaVersion": 5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[]}`,
	})
	require.NoError(t, err)
	require.Contains(t, written.Body, "\n  \"schemaVersion\": 5")

	released, err := os.ReadFile(filepath.Join(root, "components", "Button", "versions", "1.0.0", "story.json"))
	require.NoError(t, err)
	require.Contains(t, string(released), "\"schemaVersion\": 5")
	draftStory, err := os.ReadFile(filepath.Join(root, "components", "Button", "versions", draft.Version.Version, "story.json"))
	require.NoError(t, err)
	require.Equal(t, written.Body, string(draftStory))
}

func TestService_UpdateContentTargetsActiveDraftAndPreservesRelease(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	_, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
		LibraryID: "react-component-library:Button", Slug: "Button", DisplayName: "Button",
		InitialVersion: "1.0.0", InitialSource: "export function Button() { return <button>released</button>; }",
		ScaffoldExamples: true,
	})
	require.NoError(t, err)
	draft, err := svc.(components.AuthoringService).BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{
		Component: "react-component-library:Button", Bump: "patch",
	})
	require.NoError(t, err)

	written, err := svc.UpdateContent(context.Background(), "react-component-library:Button", components.WriteContentInput{
		Body: "export function Button() { return <button>draft</button>; }",
	})
	require.NoError(t, err)
	require.Contains(t, written.SourcePath, draft.Version.Version)

	released, err := os.ReadFile(filepath.Join(root, "components", "Button", "versions", "1.0.0", "Button.tsx"))
	require.NoError(t, err)
	require.Contains(t, string(released), "released")
	draftSource, err := os.ReadFile(filepath.Join(root, "components", "Button", "versions", draft.Version.Version, "Button.tsx"))
	require.NoError(t, err)
	require.Contains(t, string(draftSource), "draft")
}

func TestService_UpdateVersionContentAtRefusesReleasedVersion(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	_, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
		LibraryID: "react-component-library:Button", Slug: "Button", DisplayName: "Button",
		InitialVersion: "1.0.0", InitialSource: "export function Button() { return <button />; }",
		ScaffoldExamples: true,
	})
	require.NoError(t, err)

	writer := svc.(interface {
		UpdateVersionContentAt(context.Context, string, string, string, components.WriteContentInput) (components.Content, error)
	})
	_, err = writer.UpdateVersionContentAt(context.Background(), "react-component-library:Button", "1.0.0", "story.json", components.WriteContentInput{
		Body: `{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[]}`,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "released component versions are immutable; create a draft first")
}

func TestService_ListAppliesDefaultLimit(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)
	ctx := context.Background()

	_, err := svc.Upsert(ctx, components.UpsertInput{LibraryID: "a"})
	require.NoError(t, err)
	_, err = svc.Upsert(ctx, components.UpsertInput{LibraryID: "b"})
	require.NoError(t, err)

	got, err := svc.List(ctx, components.SearchQuery{Limit: 0})
	require.NoError(t, err)
	require.Len(t, got, 2, "default limit should fetch all seeded rows")
}

func TestService_GetByLibraryIDPropagatesNotFound(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)
	_, err := svc.GetByLibraryID(context.Background(), "missing")
	var nf components.ErrComponentNotFound
	require.True(t, errors.As(err, &nf))
}

func TestService_GetAcceptsLibraryIDOrInternalID(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)

	created, err := svc.Upsert(context.Background(), components.UpsertInput{LibraryID: "react-component-library:Button"})
	require.NoError(t, err)

	byID, err := svc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	byLibraryID, err := svc.Get(context.Background(), created.LibraryID)
	require.NoError(t, err)
	require.Equal(t, byID.ID, byLibraryID.ID)
}

func TestService_GetAcceptsCatalogID(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)

	created, err := svc.Upsert(context.Background(), components.UpsertInput{
		LibraryID: "react-component-library:Slider",
		CatalogID: "controls.slider",
	})
	require.NoError(t, err)

	byCatalogID, err := svc.Get(context.Background(), "controls.slider")
	require.NoError(t, err)
	require.Equal(t, created.ID, byCatalogID.ID)
}

func TestService_UpdateManifestRepairsAuthoredComponentMissingFromIndex(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	assetRoot := filepath.Join(root, "components", "ControlBase")
	require.NoError(t, os.MkdirAll(filepath.Join(assetRoot, "versions", "1.0.0"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(assetRoot, "versions", "1.1.0"), 0o755))
	manifest := `{
  "libraryId": "react-component-library:ControlBase",
  "displayName": "Control Base",
  "description": "Shared control primitive.",
  "kind": "control",
  "latest": "1.0.0",
  "draft": "",
  "deprecatedVersions": ["1.0.0"],
  "tags": ["control"]
}
`
	require.NoError(t, os.WriteFile(filepath.Join(assetRoot, "component.json"), []byte(manifest), 0o600))
	for _, version := range []string{"1.0.0", "1.1.0"} {
		source := fmt.Sprintf(`/**
 * @libraryId react-component-library:ControlBase
 * @displayName Control Base
 * @description Shared control primitive.
 * @version %s
 * @tags ["control"]
 */
export function ControlBase() { return null; }
`, version)
		require.NoError(t, os.WriteFile(filepath.Join(assetRoot, "versions", version, "ControlBase.tsx"), []byte(source), 0o600))
	}

	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	updated, err := svc.UpdateComponentManifest(context.Background(), components.UpdateComponentManifestInput{
		ComponentID:        "react-component-library:ControlBase",
		LatestVersion:      "1.1.0",
		DeprecatedVersions: []string{"1.0.0"},
	})
	require.NoError(t, err)
	require.Equal(t, "1.1.0", updated.LatestVersion)

	raw, err := os.ReadFile(filepath.Join(assetRoot, "component.json"))
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, "1.1.0", got["latest"])
}

func TestAuthoringWorkflowBeginsChecksAndPublishesByLibraryID(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	authoring := svc.(components.AuthoringService)

	created, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
		LibraryID:        "react-component-library:Button",
		Slug:             "Button",
		DisplayName:      "Button",
		InitialVersion:   "1.0.0",
		InitialSource:    `export function Button() { return <button>Save</button>; }`,
		ScaffoldExamples: true,
	})
	require.NoError(t, err)
	baselineDir := filepath.Join(root, "components", "Button", "versions", "1.0.0")
	companions := map[string][]byte{
		"story.tsx":                []byte("export function ButtonStory() { return null; }\n"),
		"experience-contract.json": []byte("{\"kind\":\"experience-component\"}\n"),
		"styles.css":               []byte(".button { display: inline-flex; }\n"),
		"focus-ring.svg":           []byte("<svg><circle r=\"2\" /></svg>\n"),
		"LICENSE":                  []byte("Button fixture license\n"),
	}
	for name, body := range companions {
		require.NoError(t, os.WriteFile(filepath.Join(baselineDir, name), body, 0o600))
	}

	baselineStory, err := os.ReadFile(filepath.Join(baselineDir, "story.json"))
	require.NoError(t, err)
	draft, err := authoring.BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{
		Component: created.Component.LibraryID,
		Bump:      "minor",
	})
	require.NoError(t, err)
	require.Equal(t, "1.1.0-draft.1", draft.Version.Version)
	require.Equal(t, "1.1.0-draft.1", draft.Component.DraftVersion)
	draftStoryPath := filepath.Join(root, "components", "Button", "versions", draft.Version.Version, "story.json")
	draftStory, err := os.ReadFile(draftStoryPath)
	require.NoError(t, err)
	require.Equal(t, baselineStory, draftStory, "begin must preserve authored companion artifacts")
	for name, body := range companions {
		copied, readErr := os.ReadFile(filepath.Join(root, "components", "Button", "versions", draft.Version.Version, name))
		require.NoError(t, readErr)
		if name == "experience-contract.json" {
			require.JSONEq(t, string(body), string(copied), "begin must preserve %s semantics", name)
			require.Equal(t, "\n", string(copied[len(copied)-1:]), "formatted contracts end with a newline")
		} else {
			require.Equal(t, body, copied, "begin must preserve %s byte-for-byte", name)
		}
	}

	authoredStory := []byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"primary","name":"Primary","args":{},"expect":[]}]
}
`)
	require.NoError(t, os.WriteFile(draftStoryPath, authoredStory, 0o600))
	checked, err := authoring.CheckComponentVersion(context.Background(), created.Component.LibraryID, draft.Version.Version)
	require.NoError(t, err)
	require.True(t, checked.Passed, checked.Checks)

	released, err := authoring.PublishComponentVersion(context.Background(), components.PublishComponentVersionInput{Component: created.Component.LibraryID})
	require.NoError(t, err)
	require.Equal(t, "1.1.0", released.Version.Version)
	require.Equal(t, "1.1.0", released.Component.LatestVersion)
	require.Empty(t, released.Component.DraftVersion)
	require.NoDirExists(t, filepath.Join(root, "components", "Button", "versions", draft.Version.Version))
	releaseStory, err := os.ReadFile(filepath.Join(root, "components", "Button", "versions", "1.1.0", "story.json"))
	require.NoError(t, err)
	require.Equal(t, authoredStory, releaseStory, "publish must preserve the exact checked story contract")
	for name, body := range companions {
		copied, readErr := os.ReadFile(filepath.Join(root, "components", "Button", "versions", "1.1.0", name))
		require.NoError(t, readErr)
		if name == "experience-contract.json" {
			require.JSONEq(t, string(body), string(copied), "publish must preserve %s semantics", name)
		} else {
			require.Equal(t, body, copied, "publish must preserve %s byte-for-byte", name)
		}
	}
}

func TestAuthoringCheckRefreshesSelectedManifestBeforeResolvingDependencies(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	authoring := svc.(components.AuthoringService)

	created, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
		LibraryID:        "react-component-library:EmptyState",
		Slug:             "EmptyState",
		DisplayName:      "Empty State",
		InitialVersion:   "1.0.0",
		InitialSource:    `export function EmptyState() { return <section />; }`,
		ScaffoldExamples: true,
	})
	require.NoError(t, err)
	draft, err := authoring.BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{
		Component: created.Component.LibraryID,
		Bump:      "patch",
	})
	require.NoError(t, err)

	// Simulate an editor adding a version-local dependency after the draft was
	// indexed. Without the focused refresh, the stale registry row still
	// reports an empty dependency closure and the check incorrectly passes.
	lockPath := filepath.Join(root, "components", "EmptyState", "versions", draft.Version.Version, "dependencies.json")
	require.NoError(t, os.WriteFile(lockPath, []byte(`{"schemaVersion":1,"libraryId":"react-component-library:EmptyState","version":"`+draft.Version.Version+`","resolvedAt":"2026-08-27T00:00:00Z","dependencies":[{"libraryId":"react-component-library:MissingPrimitive","version":"1.0.0","rank":3}]}
`), 0o600))

	checked, err := authoring.CheckComponentVersion(context.Background(), created.Component.LibraryID, draft.Version.Version)
	require.NoError(t, err)
	require.False(t, checked.Passed)
	var dependencyFailure *components.ComponentVersionCheck
	for index := range checked.Checks {
		if checked.Checks[index].Stage == "dependencies" {
			dependencyFailure = &checked.Checks[index]
			break
		}
	}
	require.NotNil(t, dependencyFailure)
	require.Equal(t, "failed", dependencyFailure.Verdict)
}

func TestAuthoringDraftRewritesInheritedDependencyLockIdentity(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	created, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
		LibraryID:        "react-component-library:Stack",
		Slug:             "Stack",
		DisplayName:      "Stack",
		InitialVersion:   "1.0.0",
		InitialSource:    `export function Stack() { return <div />; }`,
		ScaffoldExamples: true,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Stack", "versions", "1.0.0", "dependencies.json"), []byte(`{
  "schemaVersion": 1,
  "libraryId": "react-component-library:Stack",
  "version": "1.0.0",
  "resolvedAt": "2026-08-27T00:00:00Z",
  "dependencies": []
}
`), 0o600))

	draft, err := svc.(components.AuthoringService).BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{
		Component: created.Component.LibraryID,
		Bump:      "patch",
	})
	require.NoError(t, err)
	lockBytes, err := os.ReadFile(filepath.Join(root, "components", "Stack", "versions", draft.Version.Version, "dependencies.json"))
	require.NoError(t, err)
	var lock struct {
		LibraryID string `json:"libraryId"`
		Version   string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(lockBytes, &lock))
	require.Equal(t, created.Component.LibraryID, lock.LibraryID)
	require.Equal(t, draft.Version.Version, lock.Version)
}

func TestAuthoringVersionCopiesRewriteExperienceStoryRef(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	authoring := svc.(components.AuthoringService)

	created, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
		LibraryID:        "react-component-library:Panel",
		Slug:             "Panel",
		DisplayName:      "Panel",
		InitialVersion:   "1.0.0",
		InitialSource:    `export function Panel() { return <section />; }`,
		ScaffoldExamples: true,
	})
	require.NoError(t, err)
	baselineContract := filepath.Join(root, "components", "Panel", "versions", "1.0.0", "experience-contract.json")
	require.NoError(t, os.WriteFile(baselineContract, []byte(`{
  "kind": "experience-component",
  "component": {"storyRef": "../../library/components/Panel/versions/1.0.0/story.json"}
}
`), 0o600))

	draft, err := authoring.BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{
		Component: created.Component.LibraryID,
		Bump:      "patch",
	})
	require.NoError(t, err)
	draftContract, err := os.ReadFile(filepath.Join(root, "components", "Panel", "versions", draft.Version.Version, "experience-contract.json"))
	require.NoError(t, err)
	require.Contains(t, string(draftContract), "/versions/1.0.1-draft.1/story.json")
	require.NotContains(t, string(draftContract), "/versions/1.0.0/story.json")

	released, err := authoring.PublishComponentVersion(context.Background(), components.PublishComponentVersionInput{Component: created.Component.LibraryID})
	require.NoError(t, err)
	releaseContract, err := os.ReadFile(filepath.Join(root, "components", "Panel", "versions", released.Version.Version, "experience-contract.json"))
	require.NoError(t, err)
	require.Contains(t, string(releaseContract), "/versions/1.0.1/story.json")
	require.NotContains(t, string(releaseContract), "/versions/1.0.1-draft.1/story.json")
}

func TestAuthoringBeginRollsBackFilesystemAndManifestWhenIndexingFails(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	authoring := svc.(components.AuthoringService)
	created, err := svc.InitializeComponent(context.Background(), components.InitializeComponentInput{
		LibraryID: "react-component-library:Button", Slug: "Button", DisplayName: "Button", InitialVersion: "1.0.0", ScaffoldExamples: true,
	})
	require.NoError(t, err)

	repo.UpsertErr = errors.New("injected index failure")
	_, err = authoring.BeginComponentVersion(context.Background(), components.BeginComponentVersionInput{Component: created.Component.LibraryID, Bump: "minor"})
	require.ErrorContains(t, err, "injected index failure")
	require.NoDirExists(t, filepath.Join(root, "components", "Button", "versions", "1.1.0-draft.1"))
	manifest, readErr := os.ReadFile(filepath.Join(root, "components", "Button", "component.json"))
	require.NoError(t, readErr)
	require.Contains(t, string(manifest), `"latest": "1.0.0"`)
	require.NotContains(t, string(manifest), `1.1.0-draft.1`)
}

func TestService_ValidateStyleFitFoldsAffinityVerdicts(t *testing.T) {
	tests := []struct {
		name     string
		style    string
		wantKind components.StyleFitVerdictKind
		wantAff  components.DesignAffinity
		wantText string
	}{
		{
			name:     "native is ok",
			style:    "vrooli-default",
			wantKind: components.StyleFitVerdictOK,
			wantAff:  components.DesignAffinityNative,
			wantText: "token-native baseline",
		},
		{
			name:     "compatible is ok with info detail",
			style:    "vrooli-conversion-landing",
			wantKind: components.StyleFitVerdictOK,
			wantAff:  components.DesignAffinityCompatible,
			wantText: "compatible",
		},
		{
			name:     "discouraged is warn",
			style:    "vrooli-data-dense",
			wantKind: components.StyleFitVerdictWarn,
			wantAff:  components.DesignAffinityDiscouraged,
			wantText: "too sparse",
		},
		{
			name:     "undeclared style is info",
			style:    "vrooli-editorial",
			wantKind: components.StyleFitVerdictInfo,
			wantText: "declares no affinity",
		},
		{
			name:     "missing scenario style is warn",
			style:    "",
			wantKind: components.StyleFitVerdictWarn,
			wantText: "does not declare generation.design.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewFakeRepository()
			c := seedStyleFitComponent(t, repo)
			svc := components.NewServiceWithScenarioReader(repo, serviceJSONReaderFunc(func(context.Context, string) ([]byte, error) {
				if tt.style == "" {
					return []byte(`{"generation":{"design":{}}}`), nil
				}
				return []byte(`{"generation":{"design":{"id":"` + tt.style + `"}}}`), nil
			}))

			got, err := svc.ValidateStyleFit(context.Background(), c.ID, "1.0.0", "demo")
			require.NoError(t, err)
			require.Equal(t, tt.wantKind, got.Kind)
			require.Equal(t, tt.wantAff, got.Affinity)
			require.Equal(t, c.ID, got.ComponentID)
			require.Equal(t, "1.0.0", got.Version)
			require.Equal(t, "demo", got.Scenario)
			require.Equal(t, tt.style, got.ScenarioStyle)
			require.Contains(t, got.Detail, tt.wantText)
		})
	}
}

func TestService_ValidateStyleFitRequiresScenarioReader(t *testing.T) {
	repo := mocks.NewFakeRepository()
	c := seedStyleFitComponent(t, repo)
	svc := components.NewService(repo)

	_, err := svc.ValidateStyleFit(context.Background(), c.ID, "1.0.0", "demo")
	require.ErrorContains(t, err, "service.json reader not configured")
}

func TestFSServiceJSONReaderGuardsTraversal(t *testing.T) {
	reader := components.NewFSServiceJSONReader(t.TempDir())
	_, err := reader.Read(context.Background(), "../demo")
	require.ErrorContains(t, err, "invalid scenario name")
}

// TestFSServiceJSONReaderResolvesTemplateScenarioKey covers the template
// adoption key form "../templates/scenarios/<id>": it must resolve next to
// the scenarios root (so reapply against a vendored template copy can run
// style-fit) while still rejecting traversal inside the template id.
func TestFSServiceJSONReaderResolvesTemplateScenarioKey(t *testing.T) {
	repoRoot := t.TempDir()
	scenariosRoot := filepath.Join(repoRoot, "scenarios")
	serviceDir := filepath.Join(repoRoot, "templates", "scenarios", "react-vite", ".vrooli")
	require.NoError(t, os.MkdirAll(scenariosRoot, 0o755))
	require.NoError(t, os.MkdirAll(serviceDir, 0o755))
	payload := []byte(`{"generation":{"design":{"id":"vrooli-default"}}}`)
	require.NoError(t, os.WriteFile(filepath.Join(serviceDir, "service.json"), payload, 0o600))

	reader := components.NewFSServiceJSONReader(scenariosRoot)

	got, err := reader.Read(context.Background(), "../templates/scenarios/react-vite")
	require.NoError(t, err)
	require.Equal(t, payload, got)

	_, err = reader.Read(context.Background(), "../templates/scenarios/../../secrets")
	require.ErrorContains(t, err, "invalid scenario name")
	_, err = reader.Read(context.Background(), "../templates/scenarios/react-vite/nested")
	require.ErrorContains(t, err, "invalid scenario name")
}

func TestService_IngestComponentCreatesIndexedDraftAndReportsFindings(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(_ context.Context, scenario, sourceFile string) ([]byte, error) {
		require.Equal(t, "web-console", scenario)
		require.Equal(t, "ui/src/components/DrawerShell.tsx", sourceFile)
		return []byte(`import { useNavigate } from "react-router-dom";
export default function DrawerShell({ className }) { const navigate = useNavigate(); return <div className={className} onClick={() => navigate("/")} />; }`), nil
	}))

	got, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell",
		DisplayName: "Drawer Shell", Tags: []string{"overlay"},
	})
	require.NoError(t, err)
	require.Equal(t, "0.1.0-draft.1", got.DraftVersion)
	require.Equal(t, "react-component-library:drawer-shell", got.Component.LibraryID)
	require.Equal(t, got.DraftVersion, got.Component.DraftVersion)
	require.FileExists(t, filepath.Join(root, got.ManifestPath))
	require.FileExists(t, filepath.Join(root, got.SourcePath))
	require.Contains(t, got.ChecklistPath, "de-scenario-ification")
	require.Len(t, got.Findings, 1)
}

func TestService_IngestComponentRefusesUtilityClasses(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	source := `export default function Panel() { return <div className="z-wc-drawer" />; }`
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(context.Context, string, string) ([]byte, error) {
		return []byte(source), nil
	}))

	_, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/Panel.tsx", Slug: "panel",
	})
	require.ErrorContains(t, err, "z-wc-drawer")
	require.ErrorContains(t, err, "DrawerShell/1.1.2")

	source = `export default function Panel({ className }) { return <div className={className} style={{ zIndex: "var(--layer-drawer)" }} />; }`
	_, err = svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/Panel.tsx", Slug: "panel",
	})
	require.NoError(t, err)
}

// TestService_IngestScaffoldsCatalogMetadataContract asserts a fresh harvest
// lands catalog-complete: the manifest carries slot, category, and tags (slot
// and category defaulted when the harvester omits them), and every created
// version folder ships a story.json stub. This is the contract that keeps
// harvested drafts indistinguishable from authored components at ingest time.
func TestService_IngestScaffoldsCatalogMetadataContract(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(_ context.Context, _, _ string) ([]byte, error) {
		return []byte(`export default function Panel({ className }) { return <div className={className}>Panel</div>; }`), nil
	}))

	got, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/Panel.tsx", Slug: "panel",
		DisplayName: "Panel", Tags: []string{"surface"},
		// Slot and Category deliberately omitted: the scaffold must default them.
	})
	require.NoError(t, err)

	manifestRaw, err := os.ReadFile(filepath.Join(root, got.ManifestPath))
	require.NoError(t, err)
	var mf struct {
		Slot     string   `json:"slot"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}
	require.NoError(t, json.Unmarshal(manifestRaw, &mf))
	require.Equal(t, "ui-pattern", mf.Slot)
	require.Equal(t, "uncategorized", mf.Category)
	require.Equal(t, []string{"surface"}, mf.Tags)

	// Both the released baseline and the working draft carry the story stub.
	for _, version := range []string{"0.1.0", got.DraftVersion} {
		storyRaw, err := os.ReadFile(filepath.Join(root, "components", "panel", "versions", version, "story.json"))
		require.NoError(t, err, "story.json missing for version %s", version)
		require.Contains(t, string(storyRaw), `"id": "default"`)
	}
}

func TestService_IngestTransfersExplicitExperienceContractIntoReleasedVersion(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	contract := `{"kind":"experience-component","component":{"id":"panel"},"states":[]}`
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(_ context.Context, _, sourceFile string) ([]byte, error) {
		switch sourceFile {
		case "ui/src/components/Panel.tsx":
			return []byte(`export default function Panel() { return <div>Panel</div>; }`), nil
		case "experience/components/panel.json":
			return []byte(contract), nil
		default:
			return nil, os.ErrNotExist
		}
	}))

	ingested, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "demo", SourceFile: "ui/src/components/Panel.tsx", Slug: "panel", DisplayName: "Panel",
		ExperienceContractPath: "experience/components/panel.json",
	})
	require.NoError(t, err)
	draftPath := filepath.Join(root, "components", "panel", "versions", ingested.DraftVersion, "experience-contract.json")
	require.FileExists(t, draftPath)
	draftContract, err := os.ReadFile(draftPath)
	require.NoError(t, err)
	require.JSONEq(t, contract, string(draftContract))
	require.Equal(t, "\n", string(draftContract[len(draftContract)-1:]))

	component, err := svc.GetByLibraryID(context.Background(), ingested.Component.LibraryID)
	require.NoError(t, err)
	_, err = svc.CreateComponentVersion(context.Background(), components.CreateComponentVersionInput{ComponentID: component.ID, Version: "1.0.0", FromVersion: ingested.DraftVersion, Intent: components.VersionIntentRelease})
	require.NoError(t, err)
	releasePath := filepath.Join(root, "components", "panel", "versions", "1.0.0", "experience-contract.json")
	releaseContract, err := os.ReadFile(releasePath)
	require.NoError(t, err)
	require.JSONEq(t, contract, string(releaseContract))
	require.Equal(t, "\n", string(releaseContract[len(releaseContract)-1:]))
}

func TestService_IngestComponentCopiesRelativeImportClosureAsOneVersionUnit(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	sources := map[string]string{
		"ui/src/components/DrawerShell.tsx": `import { useFocusTrap } from "../hooks/useFocusTrap";
import { useEscapeKey } from "../hooks/useEscapeKey";
export function DrawerShell() { useFocusTrap(); useEscapeKey(); return <div role="dialog" aria-modal="true" />; }`,
		"ui/src/hooks/useFocusTrap.ts": `export function useFocusTrap() { window.addEventListener("keydown", () => {}); }`,
		"ui/src/hooks/useEscapeKey.ts": `export function useEscapeKey() { return undefined; }`,
	}
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(_ context.Context, scenario, sourceFile string) ([]byte, error) {
		require.Equal(t, "web-console", scenario)
		body, ok := sources[sourceFile]
		if !ok {
			return nil, errors.New("not found")
		}
		return []byte(body), nil
	}))

	got, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell", DisplayName: "DrawerShell",
	})
	require.NoError(t, err)
	draft, err := svc.GetVersion(context.Background(), got.Component.ID, got.DraftVersion)
	require.NoError(t, err)
	// story.json is a Preview contract and is exposed in the Files tab, but it
	// is not part of the production source/adoption closure. The parity report
	// remains limited to the three source files copied from the origin.
	require.Len(t, draft.Files, 4)
	require.NotNil(t, draft.ParityReport)
	require.Empty(t, draft.ParityReport.Findings)
	require.Equal(t, []string{"DrawerShell.tsx", "useEscapeKey.ts", "useFocusTrap.ts"}, draft.ParityReport.OriginFiles)
	require.Equal(t, []string{"DrawerShell.tsx", "useEscapeKey.ts", "useFocusTrap.ts", "story.json"}, []string{draft.Files[0].Path, draft.Files[1].Path, draft.Files[2].Path, draft.Files[3].Path})
	require.True(t, draft.Files[0].IsEntry)
	require.Contains(t, draft.Files[0].Content, `from "./useFocusTrap"`)
	require.FileExists(t, filepath.Join(root, "components", "drawer-shell", "versions", got.DraftVersion, "useFocusTrap.ts"))
	require.FileExists(t, filepath.Join(root, "components", "drawer-shell", "versions", got.DraftVersion, "useEscapeKey.ts"))

	reharvested, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{
		Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell", DisplayName: "DrawerShell", Version: "1.0.0",
	})
	require.NoError(t, err)
	require.Equal(t, "1.0.0-draft.1", reharvested.DraftVersion)
	require.Equal(t, got.Component.ID, reharvested.Component.ID)
	require.FileExists(t, filepath.Join(root, "components", "drawer-shell", "versions", reharvested.DraftVersion, "useFocusTrap.ts"))
}

func TestIngestBehaviorInventoryFlagsHistoricalFocusTrapLoss(t *testing.T) {
	origin := `import { useFocusTrap } from "../hooks/useFocusTrap";
export function Drawer() { return <div role="dialog" aria-modal="true" onKeyDown={() => {}} /> }`
	harvested := `export function Drawer() { return <div role="dialog" /> }`
	findings := components.BehaviorLossFindings(origin, harvested, "Drawer.tsx")
	require.Len(t, findings, 3)
	require.Contains(t, findings[0].Code, "behavior-lost")
}

// TestService_IngestBlocksBehaviorLossUnlessAccepted is the permanent
// planted-error calibration for the origin-parity gate. It reconstructs the
// historical DrawerShell failure: a focus-trap hook reachable only through an
// app-alias import the harvest cannot carry, so the listener behavior is
// dropped. The gate must fail the harvest naming the dropped listener, and
// must only proceed when the caller explicitly accepts the loss — recording
// the named losses as an acknowledged parity report.
func TestService_IngestBlocksBehaviorLossUnlessAccepted(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	sources := map[string]string{
		"ui/src/components/DrawerShell.tsx": `import { useFocusTrap } from "@/hooks/useFocusTrap";
export function DrawerShell() { useFocusTrap(); return <div role="dialog" aria-modal="true" />; }`,
		"ui/src/hooks/useFocusTrap.ts": `export function useFocusTrap() { window.addEventListener("keydown", () => {}); }`,
	}
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(_ context.Context, _ string, sourceFile string) ([]byte, error) {
		body, ok := sources[sourceFile]
		if !ok {
			return nil, errors.New("not found: " + sourceFile)
		}
		return []byte(body), nil
	}))

	in := components.IngestComponentInput{Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell", DisplayName: "Drawer Shell"}

	// (a) Without the override the harvest is blocked and names the loss.
	_, err := svc.IngestComponent(context.Background(), in)
	var loss components.ErrHarvestBehaviorLoss
	require.True(t, errors.As(err, &loss), "expected ErrHarvestBehaviorLoss, got %v", err)
	require.NotEmpty(t, loss.Findings)
	require.Contains(t, err.Error(), "addEventListener")
	require.Contains(t, err.Error(), "accept-behavior-loss")
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(components.ToConnectError(err)))
	// A blocked harvest creates nothing.
	_, notFoundErr := svc.GetByLibraryID(context.Background(), "react-component-library:drawer-shell")
	require.True(t, errors.As(notFoundErr, &components.ErrComponentNotFound{}))

	// (b) With the override the harvest proceeds and records the losses.
	accepted := in
	accepted.AcceptBehaviorLoss = true
	got, err := svc.IngestComponent(context.Background(), accepted)
	require.NoError(t, err)
	require.NotEmpty(t, got.ParityReport.Findings)
	require.True(t, got.ParityReport.Acknowledged)

	draft, err := svc.GetVersion(context.Background(), got.Component.ID, got.DraftVersion)
	require.NoError(t, err)
	require.NotNil(t, draft.ParityReport)
	require.True(t, draft.ParityReport.Acknowledged)
	require.NotEmpty(t, draft.ParityReport.Findings)
	require.Contains(t, draft.ParityReport.Findings[0].Message, "addEventListener")

	// The acceptance is durable on the version's parity.json audit trail.
	raw, err := os.ReadFile(filepath.Join(root, "components", "drawer-shell", "versions", got.DraftVersion, "parity.json"))
	require.NoError(t, err)
	var onDisk components.IngestParityReport
	require.NoError(t, json.Unmarshal(raw, &onDisk))
	require.True(t, onDisk.Acknowledged)
	require.NotEmpty(t, onDisk.Findings)
}

func TestService_CreateComponentVersionRequiresExplicitParityWaiver(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(context.Context, string, string) ([]byte, error) {
		return []byte(`export function DrawerShell() { return <div /> }`), nil
	}))
	created, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{Scenario: "web-console", SourceFile: "ui/src/components/DrawerShell.tsx", Slug: "drawer-shell"})
	require.NoError(t, err)
	report := components.IngestParityReport{Findings: []components.IngestFinding{{Code: "behavior-lost", Message: "fixture loss"}}}
	raw, err := json.Marshal(report)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "drawer-shell", "versions", created.DraftVersion, "parity.json"), raw, 0o600))
	_, err = components.NewIndexer(repo, root, nil).Run(context.Background())
	require.NoError(t, err)
	c, err := svc.GetByLibraryID(context.Background(), created.Component.LibraryID)
	require.NoError(t, err)
	_, err = svc.CreateComponentVersion(context.Background(), components.CreateComponentVersionInput{ComponentID: c.ID, Version: "1.0.0", FromVersion: created.DraftVersion, Intent: components.VersionIntentRelease})
	var waiver components.ErrParityWaiverRequired
	require.True(t, errors.As(err, &waiver))
	got, err := svc.CreateComponentVersion(context.Background(), components.CreateComponentVersionInput{ComponentID: c.ID, Version: "1.0.0", FromVersion: created.DraftVersion, Intent: components.VersionIntentRelease, AcknowledgeParityWaiver: true})
	require.NoError(t, err)
	require.NotNil(t, got.Version.ParityReport)
	require.True(t, got.Version.ParityReport.Acknowledged)
}

func TestService_CreateComponentVersionPreservesOperatorParityReport(t *testing.T) {
	repo := mocks.NewFakeRepository()
	root := t.TempDir()
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(root))
	components.SetScenarioSourceReader(svc, scenarioSourceReaderFunc(func(context.Context, string, string) ([]byte, error) {
		return []byte(`export function MarkdownRenderer() { return <div /> }`), nil
	}))
	created, err := svc.IngestComponent(context.Background(), components.IngestComponentInput{Scenario: "react-component-library", SourceFile: "ui/src/MarkdownRenderer.tsx", Slug: "markdown-renderer"})
	require.NoError(t, err)
	want := &components.IngestParityReport{
		OriginFiles:    []string{"scenarios/web-console/ui/src/components/markdown/MarkdownRenderer.tsx"},
		HarvestedFiles: []string{"MarkdownRenderer.tsx"},
	}
	got, err := svc.CreateComponentVersion(context.Background(), components.CreateComponentVersionInput{
		ComponentID:  created.Component.ID,
		Version:      "1.0.0",
		FromVersion:  created.DraftVersion,
		Intent:       components.VersionIntentRelease,
		ParityReport: want,
	})
	require.NoError(t, err)
	require.Equal(t, want.OriginFiles, got.Version.ParityReport.OriginFiles)
	require.Equal(t, want.HarvestedFiles, got.Version.ParityReport.HarvestedFiles)
}

func seedStyleFitComponent(t *testing.T, repo *mocks.FakeRepository) components.Component {
	t.Helper()
	c, err := repo.UpsertManifest(context.Background(), components.IndexManifestInput{
		Manifest: components.ComponentManifest{
			LibraryID:     "react-component-library:Button",
			DisplayName:   "Button",
			LatestVersion: "1.0.0",
			DesignStyles: []components.ComponentDesignAffinity{
				{StyleID: "vrooli-default", Affinity: components.DesignAffinityNative, Reason: "token-native baseline"},
				{StyleID: "vrooli-conversion-landing", Affinity: components.DesignAffinityCompatible},
				{StyleID: "vrooli-data-dense", Affinity: components.DesignAffinityDiscouraged, Reason: "too sparse"},
			},
		},
		Versions: []components.ComponentVersion{{Version: "1.0.0"}},
	})
	require.NoError(t, err)
	return c
}

type serviceJSONReaderFunc func(context.Context, string) ([]byte, error)

func (f serviceJSONReaderFunc) Read(ctx context.Context, scenario string) ([]byte, error) {
	return f(ctx, scenario)
}

type scenarioSourceReaderFunc func(context.Context, string, string) ([]byte, error)

func (f scenarioSourceReaderFunc) Read(ctx context.Context, scenario, sourceFile string) ([]byte, error) {
	return f(ctx, scenario, sourceFile)
}
