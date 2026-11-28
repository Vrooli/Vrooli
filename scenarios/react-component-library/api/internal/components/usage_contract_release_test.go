package components_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
	"react-component-library/internal/components/mocks"
)

func TestDraftPublishRewritesUsageContractToReleaseVersion(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	svc := components.NewServiceWithContent(mocks.NewFakeRepository(), components.NewFSContentStore(root))
	created, err := svc.InitializeComponent(ctx, components.InitializeComponentInput{
		LibraryID: "react-component-library:ContractedPanel", Slug: "ContractedPanel", DisplayName: "Contracted Panel",
		InitialVersion: "1.0.0", InitialSource: `export function ContractedPanel() { return <div />; }`, ScaffoldExamples: true,
	})
	require.NoError(t, err)
	latestContract := filepath.Join(root, "components", "ContractedPanel", "versions", "1.0.0", "usage-contract.json")
	require.NoError(t, os.WriteFile(latestContract, []byte(`{"componentId":"react-component-library:ContractedPanel","version":"1.0.0"}`), 0o600))

	authoring := svc.(components.AuthoringService)
	draft, err := authoring.BeginComponentVersion(ctx, components.BeginComponentVersionInput{Component: created.Component.LibraryID, Bump: "patch"})
	require.NoError(t, err)
	require.Equal(t, "1.0.1-draft.1", draft.Version.Version)
	draftContractPath := filepath.Join(root, "components", "ContractedPanel", "versions", draft.Version.Version, "usage-contract.json")
	draftContractBytes, err := os.ReadFile(draftContractPath)
	require.NoError(t, err)
	var draftContract struct {
		ComponentID string `json:"componentId"`
		Version     string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(draftContractBytes, &draftContract))
	require.Equal(t, created.Component.LibraryID, draftContract.ComponentID)
	require.Equal(t, draft.Version.Version, draftContract.Version, "draft-begin must re-version the inherited contract")

	published, err := authoring.PublishComponentVersion(ctx, components.PublishComponentVersionInput{Component: created.Component.LibraryID})
	require.NoError(t, err)
	require.Equal(t, "1.0.1", published.Version.Version)
	contractPath := filepath.Join(root, "components", "ContractedPanel", "versions", published.Version.Version, "usage-contract.json")
	contractBytes, err := os.ReadFile(contractPath)
	require.NoError(t, err)
	var contract struct {
		ComponentID string `json:"componentId"`
		Version     string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(contractBytes, &contract))
	require.Equal(t, created.Component.LibraryID, contract.ComponentID)
	require.Equal(t, published.Version.Version, contract.Version)
}

func TestDraftCheckValidatesContractIntroducedByLegacyAsset(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	svc := components.NewServiceWithContent(mocks.NewFakeRepository(), components.NewFSContentStore(root))
	created, err := svc.InitializeComponent(ctx, components.InitializeComponentInput{
		LibraryID: "react-component-library:LegacyPanel", Slug: "LegacyPanel", DisplayName: "Legacy Panel",
		InitialVersion: "1.0.0", InitialSource: `export function LegacyPanel() { return <div />; }`, ScaffoldExamples: true,
	})
	require.NoError(t, err)
	authoring := svc.(components.AuthoringService)
	draft, err := authoring.BeginComponentVersion(ctx, components.BeginComponentVersionInput{Component: created.Component.LibraryID, Bump: "patch"})
	require.NoError(t, err)

	contractPath := filepath.Join(root, "components", "LegacyPanel", "versions", draft.Version.Version, "usage-contract.json")
	require.NoError(t, os.WriteFile(contractPath, []byte(`{"componentId":"react-component-library:LegacyPanel","version":"1.0.0"}`), 0o600))
	checked, err := authoring.CheckComponentVersion(ctx, created.Component.LibraryID, draft.Version.Version)
	require.NoError(t, err)
	require.False(t, checked.Passed)
	require.Equal(t, "failed", checkByStage(t, checked.Checks, "usage-contract").Verdict)
	writeValidDraftContract(t, root, "LegacyPanel", created.Component.LibraryID, draft.Version.Version)
	checked, err = authoring.CheckComponentVersion(ctx, created.Component.LibraryID, draft.Version.Version)
	require.NoError(t, err)
	require.True(t, checked.Passed, checked.Checks)
	require.Equal(t, "passed", checkByStage(t, checked.Checks, "usage-contract").Verdict)
	published, err := authoring.PublishComponentVersion(ctx, components.PublishComponentVersionInput{Component: created.Component.LibraryID})
	require.NoError(t, err)
	require.Equal(t, "1.0.1", published.Version.Version)
	releasedContract, err := os.ReadFile(filepath.Join(root, "components", "LegacyPanel", "versions", published.Version.Version, "usage-contract.json"))
	require.NoError(t, err)
	require.JSONEq(t, `{"componentId":"react-component-library:LegacyPanel","version":"1.0.1"}`, string(releasedContract))
}

func TestDraftBeginRefusesSecondActiveDraftWithoutWritingAnotherVersion(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	svc := components.NewServiceWithContent(mocks.NewFakeRepository(), components.NewFSContentStore(root))
	created, err := svc.InitializeComponent(ctx, components.InitializeComponentInput{
		LibraryID: "react-component-library:Panel", Slug: "Panel", DisplayName: "Panel",
		InitialVersion: "1.0.0", InitialSource: `export function Panel() { return <div />; }`, ScaffoldExamples: true,
	})
	require.NoError(t, err)
	authoring := svc.(components.AuthoringService)
	first, err := authoring.BeginComponentVersion(ctx, components.BeginComponentVersionInput{Component: created.Component.LibraryID, Bump: "patch"})
	require.NoError(t, err)
	_, err = authoring.BeginComponentVersion(ctx, components.BeginComponentVersionInput{Component: created.Component.LibraryID, Bump: "minor"})
	var invalid components.ErrInvalidHeader
	require.ErrorAs(t, err, &invalid)
	require.Contains(t, invalid.Reason, "active draft")
	require.NoDirExists(t, filepath.Join(root, "components", "Panel", "versions", "1.1.0-draft.1"))
	current, err := svc.Get(ctx, created.Component.LibraryID)
	require.NoError(t, err)
	require.Equal(t, first.Version.Version, current.DraftVersion)
}

func TestDraftBeginRefusesNonObjectUsageContractWithoutWriting(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	svc := components.NewServiceWithContent(mocks.NewFakeRepository(), components.NewFSContentStore(root))
	created, err := svc.InitializeComponent(ctx, components.InitializeComponentInput{
		LibraryID: "react-component-library:MalformedContract", Slug: "MalformedContract", DisplayName: "Malformed Contract",
		InitialVersion: "1.0.0", InitialSource: `export function MalformedContract() { return <div />; }`, ScaffoldExamples: true,
	})
	require.NoError(t, err)
	latestContract := filepath.Join(root, "components", "MalformedContract", "versions", "1.0.0", "usage-contract.json")
	require.NoError(t, os.WriteFile(latestContract, []byte("null\n"), 0o600))

	_, err = svc.(components.AuthoringService).BeginComponentVersion(ctx, components.BeginComponentVersionInput{Component: created.Component.LibraryID, Bump: "patch"})
	require.ErrorContains(t, err, "expected a JSON object")
	current, err := svc.Get(ctx, created.Component.LibraryID)
	require.NoError(t, err)
	require.Empty(t, current.DraftVersion)
	require.NoDirExists(t, filepath.Join(root, "components", "MalformedContract", "versions", "1.0.1-draft.1"))
}

func TestLegacyAssetCannotPublishUntilDraftIntroducesUsageContract(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	svc := components.NewServiceWithContent(mocks.NewFakeRepository(), components.NewFSContentStore(root))
	created, err := svc.InitializeComponent(ctx, components.InitializeComponentInput{
		LibraryID: "react-component-library:LegacyPanel", Slug: "LegacyPanel", DisplayName: "Legacy Panel",
		InitialVersion: "1.0.0", InitialSource: `export function LegacyPanel() { return <div />; }`, ScaffoldExamples: true,
	})
	require.NoError(t, err)
	authoring := svc.(components.AuthoringService)
	draft, err := authoring.BeginComponentVersion(ctx, components.BeginComponentVersionInput{Component: created.Component.LibraryID, Bump: "patch"})
	require.NoError(t, err)

	checked, err := authoring.CheckComponentVersion(ctx, created.Component.LibraryID, draft.Version.Version)
	require.NoError(t, err)
	require.False(t, checked.Passed)
	require.Equal(t, "failed", checkByStage(t, checked.Checks, "usage-contract").Verdict)
	require.Contains(t, checkByStage(t, checked.Checks, "usage-contract").Message, "usage-contract.json is required")

	_, err = authoring.PublishComponentVersion(ctx, components.PublishComponentVersionInput{Component: created.Component.LibraryID})
	var failed components.ErrVersionCheckFailed
	require.ErrorAs(t, err, &failed)
	require.Equal(t, draft.Version.Version, failed.Version)
	current, err := svc.Get(ctx, created.Component.LibraryID)
	require.NoError(t, err)
	require.Equal(t, "1.0.0", current.LatestVersion)
	require.Equal(t, draft.Version.Version, current.DraftVersion)
	require.NoDirExists(t, filepath.Join(root, "components", "LegacyPanel", "versions", "1.0.1"))
}

// writeValidDraftContract keeps lifecycle success fixtures explicit about the
// contract required by the registered release gate.
func writeValidDraftContract(t *testing.T, root, slug, libraryID, version string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"componentId": libraryID, "version": version})
	require.NoError(t, err)
	path := filepath.Join(root, "components", slug, "versions", version, "usage-contract.json")
	require.NoError(t, os.WriteFile(path, body, 0o600))
}
