package audit

import (
	"testing"

	"quality-health/internal/contracts"

	"github.com/stretchr/testify/require"
)

func TestContractToProtoPreservesRuleIDs(t *testing.T) {
	contract, ok := contracts.ByRule(contracts.RuleTSConfigStrict)
	require.True(t, ok)

	got := contractToProto(contract)

	require.Equal(t, contract.ID, got.GetId())
	require.Contains(t, got.GetRuleIds(), contracts.RuleTSConfigStrict)
	require.True(t, got.GetAutofixAvailable())
}
