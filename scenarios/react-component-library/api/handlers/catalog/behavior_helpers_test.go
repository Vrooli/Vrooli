package catalog

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/gates"
)

func TestCatalogFindingProjectionHelpers(t *testing.T) {
	tests := []struct {
		name      string
		finding   gates.Finding
		wantID    string
		wantScope catalogv1.FindingScope
	}{
		{"asset uses catalog identity", gates.Finding{AssetID: "library:Card", CatalogID: "react-component-library:Card"}, "react-component-library:Card", catalogv1.FindingScope_FINDING_SCOPE_ASSET},
		{"corpus scope omits asset identity", gates.Finding{AssetID: "Card", Scope: gates.FindingScopeCorpus}, "", catalogv1.FindingScope_FINDING_SCOPE_CORPUS},
		{"corpus prefix marks corpus", gates.Finding{AssetID: "__corpus__:catalog"}, "", catalogv1.FindingScope_FINDING_SCOPE_CORPUS},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantID, findingCatalogID(tc.finding))
			require.Equal(t, tc.wantScope, findingScope(tc.finding))
		})
	}
	require.Equal(t, catalogv1.FindingSeverity_FINDING_SEVERITY_INFO, findingSeverity("info"))
	require.Equal(t, catalogv1.FindingSeverity_FINDING_SEVERITY_WARNING, findingSeverity("warning"))
	require.Equal(t, catalogv1.FindingSeverity_FINDING_SEVERITY_BLOCKING, findingSeverity("error"))
}

func TestSameAssetIdentityAcceptsCatalogLibraryAndScopedNames(t *testing.T) {
	for _, pair := range [][2]string{
		{" Card ", "react-component-library:card"},
		{"library.Card", "Card"},
		{"react-component-library:Card", "Card"},
	} {
		require.True(t, sameAssetIdentity(pair[0], pair[1]), "%q should match %q", pair[0], pair[1])
	}
	require.False(t, sameAssetIdentity("Card", ""))
	require.False(t, sameAssetIdentity("Card", "Button"))
}

func TestGetScoreHistoryWithoutEvidenceStoreIsEmpty(t *testing.T) {
	h := &handler{}
	response, err := h.GetScoreHistory(context.Background(), connect.NewRequest(&catalogv1.GetScoreHistoryRequest{}))
	require.NoError(t, err)
	require.NotNil(t, response.Msg)
	require.Empty(t, response.Msg.GetPoints())
}

func TestEnsureQuarantinesSkipsRepeatedCalibration(t *testing.T) {
	h := &handler{quarantined: map[string]bool{"existing": true}, quarantineKnown: true}
	require.NoError(t, h.ensureQuarantines([]catalogcoverage.GateDefinition{{ID: "unknown", Blocking: true}}))
	require.Equal(t, map[string]bool{"existing": true}, h.quarantined)
}
