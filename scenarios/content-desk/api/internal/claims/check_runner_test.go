package claims_test

import (
	"context"
	"errors"
	"testing"

	"content-desk/internal/claims"
	claimsmocks "content-desk/internal/claims/mocks"

	"github.com/stretchr/testify/require"
)

func TestLocalRunner_ComparesNormalizedOutput(t *testing.T) {
	t.Parallel()
	result, err := (claims.LocalRunner{}).Run(context.Background(), claims.EvidenceCheck{Command: "printf 'verified\\n'", ExpectedResult: "verified"})
	require.NoError(t, err)
	require.True(t, result.Matches)
	require.Equal(t, "verified", result.ActualResult)
}

func TestLocalRunner_ReturnsOutputWhenCommandFails(t *testing.T) {
	t.Parallel()
	result, err := (claims.LocalRunner{}).Run(context.Background(), claims.EvidenceCheck{Command: "printf 'failed check' >&2; exit 4", ExpectedResult: "verified"})
	require.Error(t, err)
	require.False(t, result.Matches)
	require.Equal(t, "failed check", result.ActualResult)
}

func TestFakeRunner_IsDeterministicClaimServiceSeam(t *testing.T) {
	t.Parallel()
	fake := &claimsmocks.FakeRunner{Result: claims.CheckResult{ActualResult: "verified", Matches: true}}
	result, err := fake.Run(context.Background(), claims.EvidenceCheck{Command: "ignored", ExpectedResult: "verified"})
	require.NoError(t, err)
	require.True(t, result.Matches)
	require.Len(t, fake.Checks, 1)
	fake.Err = errors.New("runner unavailable")
	_, err = fake.Run(context.Background(), claims.EvidenceCheck{Command: "ignored"})
	require.ErrorIs(t, err, fake.Err)
}
