package derivation

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

func TestRegistryLoadsAndP0HandlersAreNonContainer(t *testing.T) {
	registry, err := LoadRegistry()
	require.NoError(t, err)
	require.NotEmpty(t, registry.Handlers)
	for _, handler := range registry.Handlers {
		if handler.Tier <= int(documentpb.Tier_TIER_TWO) {
			require.NotEqual(t, "docker", strings.ToLower(handler.Runtime), "P0 chain cannot require a container")
		}
	}
}

func TestRegistryDeclaresTheDocumentMatrixHandlers(t *testing.T) { // [REQ:DOC-P0-005]
	data, err := os.ReadFile("../../../docs/reference/format-matrix.md")
	require.NoError(t, err)
	text := string(data)
	for _, name := range []string{"doc-parse", "doc-ocr", "native-markup"} {
		require.Contains(t, text, name)
	}
	registry, err := LoadRegistry()
	require.NoError(t, err)
	for _, name := range []string{"doc-parse", "doc-ocr", "native-markup"} {
		found := false
		for _, handler := range registry.Handlers {
			found = found || handler.ID == name
		}
		require.True(t, found, "registry must project %s", name)
	}
}

func TestBlockedHandlerIsDistinctFromMissingHandler(t *testing.T) {
	registry := testRegistry()
	registry.Handlers = append(registry.Handlers, Handler{ID: "heavy", Formats: []string{"application/pdf"}, Tier: 2})
	_, err := registry.Match("application/pdf", documentpb.Tier_TIER_ONE)
	require.ErrorIs(t, err, ErrBlockedByPolicy)
	_, err = registry.Match("application/x-unknown", documentpb.Tier_TIER_ONE)
	require.ErrorIs(t, err, ErrNoHandler)
}
