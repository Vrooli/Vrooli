package failures_test

import (
	"context"
	"errors"
	"testing"

	"data-backup-manager/internal/failures"
	"github.com/stretchr/testify/require"
)

func TestClassify_UsesStableRedactedVocabulary(t *testing.T) {
	cause := failures.Classify(errors.New("credential is not configured: secret=do-not-leak"))

	require.Equal(t, failures.CredentialMissing, cause.Code)
	require.Equal(t, failures.CategoryCredential, cause.Category)
	require.Equal(t, failures.ScopeDestination, cause.Scope)
	require.NotContains(t, cause.Message, "do-not-leak")
	require.NotContains(t, cause.NextAction, "do-not-leak")
}

func TestClassify_ContextCancellationIsInterruptible(t *testing.T) {
	cause := failures.Classify(context.Canceled)

	require.Equal(t, failures.ProcessInterrupted, cause.Code)
	require.Equal(t, failures.CategoryExecution, cause.Category)
	require.Contains(t, cause.Message, "interrupted")
}

func TestClassify_ReadOnlySecretNamedSourceIsDestinationFailure(t *testing.T) {
	cause := failures.Classify(errors.New("unable to write /tmp/secrets.enc.json: read-only file system"))

	require.Equal(t, failures.DestinationReadOnly, cause.Code)
	require.Equal(t, failures.CategoryDestination, cause.Category)
	require.NotEqual(t, failures.CredentialUnreadable, cause.Code)
}
