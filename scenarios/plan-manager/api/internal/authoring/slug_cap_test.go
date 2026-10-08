package authoring_test

import (
	"context"
	"strings"
	"testing"

	"plan-manager/internal/authoring"

	"github.com/stretchr/testify/require"
)

// TestSessionSlugCappedAtDerivation: a very long title derives a slug ≤ 60
// chars, word-boundary truncated, and the session resolves by that handle.
func TestSessionSlugCappedAtDerivation(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	title := "Plan manager authoring form not wizard honest finalize and batch submission overhaul for agents"
	sess, _, err := svc.StartSession(ctx, title, "", "")
	require.NoError(t, err)
	require.LessOrEqual(t, len(sess.Slug), 60, "derived slug must be capped: %q", sess.Slug)
	require.False(t, strings.HasSuffix(sess.Slug, "-"))

	resolved, _, err := svc.GetSession(ctx, sess.Slug)
	require.NoError(t, err)
	require.Equal(t, sess.ID, resolved.ID)
}

// TestSessionSlugCollisionSuffixAfterTruncation: two sessions with the same
// long title truncate to the same base then disambiguate with a suffix — the
// suffix may exceed the cap only by its own length.
func TestSessionSlugCollisionSuffixAfterTruncation(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	title := strings.Repeat("collision ", 12) + "tail" // > 60 chars slugified
	first, _, err := svc.StartSession(ctx, title, "", "")
	require.NoError(t, err)
	second, _, err := svc.StartSession(ctx, title, "", "")
	require.NoError(t, err)
	require.NotEqual(t, first.Slug, second.Slug)
	require.True(t, strings.HasPrefix(second.Slug, first.Slug), "suffix applies after truncation: %q vs %q", first.Slug, second.Slug)
	require.LessOrEqual(t, len(second.Slug), 60+3, "suffix may exceed the cap by its own length only")
}
