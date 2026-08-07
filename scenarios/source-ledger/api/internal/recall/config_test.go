package recall

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnvUsesIndependentDefaultsAndOverrides(t *testing.T) {
	config, err := ConfigFromEnv(func(key string) (string, bool) {
		return map[string]string{FrontierTargetEnv: "17", WakeBudgetEnv: "53"}[key], true
	})
	require.NoError(t, err)
	require.Equal(t, 17, config.FrontierTarget)
	require.Equal(t, 53, config.WakeBudget)
}

func TestConfigFromEnvUsesCalibratedDefaults(t *testing.T) {
	config, err := ConfigFromEnv(func(string) (string, bool) { return "", false })
	require.NoError(t, err)
	require.Equal(t, 16, config.FrontierTarget)
	require.Equal(t, DefaultWakeBudget, config.WakeBudget)
}

func TestConfigFromEnvRejectsInvalidBudget(t *testing.T) {
	_, err := ConfigFromEnv(func(string) (string, bool) { return "0", true })
	require.ErrorContains(t, err, "positive integer")
}
