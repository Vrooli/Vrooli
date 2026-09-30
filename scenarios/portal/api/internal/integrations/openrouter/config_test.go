package openrouter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveConfigRequiresAPIKey(t *testing.T) {
	t.Setenv(apiKeyEnv, "")

	got, err := ResolveConfig()

	require.ErrorIs(t, err, ErrAPIKeyMissing)
	require.Empty(t, got.APIKey)
}

func TestResolveConfigTrimsAPIKey(t *testing.T) {
	t.Setenv(apiKeyEnv, " test-key ")

	got, err := ResolveConfig()

	require.NoError(t, err)
	require.Equal(t, "test-key", got.APIKey)
}

func TestCheckConfigReportsMissingKeyWithoutPanic(t *testing.T) {
	t.Setenv(apiKeyEnv, "")

	got := CheckConfig()

	require.False(t, got.Configured)
	require.Equal(t, ErrAPIKeyMissing.Error(), got.Error)
}
