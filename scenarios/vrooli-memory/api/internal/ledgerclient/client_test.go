package ledgerclient

import (
	"testing"

	"github.com/stretchr/testify/require"
	sourcerecall "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	memoryrecall "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
)

func TestTranslatePreservesWireCompatibleRecallContract(t *testing.T) {
	out := &sourcerecall.RecallRequest{}
	require.NoError(t, TranslateWithScope(&memoryrecall.RecallRequest{Query: "durable", Limit: 4}, out, ""))
	require.Equal(t, "durable", out.GetQuery())
	require.Equal(t, int32(4), out.GetLimit())
	require.Equal(t, DefaultScope, out.GetScope())
}

func TestNormalizeScopeDefaultsOnlyBlankValues(t *testing.T) {
	require.Equal(t, DefaultScope, NormalizeScope(""))
	require.Equal(t, "custom", NormalizeScope(" custom "))
}
