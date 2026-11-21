package failures_test

import (
	"context"
	"errors"
	"strings"
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

func TestClassifyCapture_KeepsARealCauseInsteadOfGenericFailure(t *testing.T) {
	cause := failures.ClassifyCapture(errors.New("open /etc/sudoers.d/vrooli-autoheal: permission denied"))

	require.Equal(t, failures.SourceUnreadable, cause.Code)
	require.Equal(t, failures.CategorySource, cause.Category)
	require.Equal(t, failures.ScopeTarget, cause.Scope)
	require.Contains(t, cause.Message, "permission denied")

	unknown := failures.ClassifyCapture(errors.New("copy: unexpected end of tar stream"))
	require.Equal(t, failures.SourceCaptureFailed, unknown.Code)
	require.Contains(t, unknown.Message, "unexpected end of tar stream")
	require.NotEqual(t, failures.Unknown, unknown.Code)
}

func TestBoundedDetail_TruncatesAndCollapsesWhitespace(t *testing.T) {
	detailed := failures.BoundedDetail(errors.New("a\n\tb  c"))
	require.Equal(t, "a b c", detailed)

	long := failures.BoundedDetail(errors.New(strings.Repeat("x", 500)))
	require.Less(t, len(long), 500)
	require.True(t, strings.HasSuffix(long, "…"))
}

func TestClassify_ReadOnlySecretNamedSourceIsDestinationFailure(t *testing.T) {
	cause := failures.Classify(errors.New("unable to write /tmp/secrets.enc.json: read-only file system"))

	require.Equal(t, failures.DestinationReadOnly, cause.Code)
	require.Equal(t, failures.CategoryDestination, cause.Category)
	require.NotEqual(t, failures.CredentialUnreadable, cause.Code)
}
