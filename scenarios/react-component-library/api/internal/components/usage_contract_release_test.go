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
}
