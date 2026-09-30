package recall

import (
	"testing"

	"source-ledger/internal/policy"

	"github.com/stretchr/testify/require"
)

func TestConfigFromPolicyCarriesDualUnitDefaults(t *testing.T) {
	config := ConfigFromPolicy(policy.BuiltInDefaults())
	require.Equal(t, policy.DefaultFrontierTarget, config.FrontierTarget)
	require.Equal(t, DefaultWakeBudget, config.WakeBudget)
	require.Equal(t, DefaultWakeBudgetChars, config.WakeBudgetChars)
	require.Equal(t, DefaultMaxEntryChars, config.MaxEntryChars)
}
