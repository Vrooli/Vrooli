package privacyconsent

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
	_ "modernc.org/sqlite"
)

func newConsentFixture(t *testing.T) (*Store, *sql.DB, *time.Time) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:consent-test?mode=memory&cache=shared")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	store, err := New(db, func() time.Time { return now })
	require.NoError(t, err)
	return store, db, &now
}

func TestGrantIsScopedExpiringAndRevocable(t *testing.T) {
	store, _, now := newConsentFixture(t)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "verified-user-7", Verified: true})
	otherUser := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "different-user", Verified: true})
	receipt, grant, err := store.Grant(ctx, "verified-user-7", "own-key", 24*time.Hour)
	require.NoError(t, err)
	require.NotEmpty(t, receipt)
	require.NoError(t, store.Verify(ctx, receipt, "own-key"))
	require.ErrorIs(t, store.Verify(otherUser, receipt, "own-key"), ErrInvalidGrant)
	require.ErrorIs(t, store.Verify(context.Background(), receipt, "own-key"), ErrInvalidGrant)
	require.ErrorIs(t, store.Verify(ctx, receipt, "vrooli-credits"), ErrInvalidGrant)
	require.ErrorIs(t, store.Verify(ctx, "forged-receipt-that-is-long-enough-1234567890", "own-key"), ErrInvalidGrant)
	require.NoError(t, store.Revoke(ctx, "verified-user-7", grant.ID))
	require.ErrorIs(t, store.Verify(ctx, receipt, "own-key"), ErrInvalidGrant)
	rows, err := store.List(ctx, "verified-user-7")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].RevokedAt)

	receipt, _, err = store.Grant(ctx, "verified-user-7", "vrooli-credits", time.Hour)
	require.NoError(t, err)
	*now = now.Add(time.Hour)
	require.ErrorIs(t, store.Verify(ctx, receipt, "vrooli-credits"), ErrInvalidGrant)
}

func TestGrantRejectsInvalidSubjectSourceAndLifetime(t *testing.T) {
	store, _, _ := newConsentFixture(t)
	for _, tc := range []struct {
		user, source string
		lifetime     time.Duration
	}{
		{"", "own-key", time.Hour},
		{"user", "local", time.Hour},
		{"user", "own-key", 0},
		{"user", "own-key", MaxLifetime + time.Second},
	} {
		_, _, err := store.Grant(context.Background(), tc.user, tc.source, tc.lifetime)
		require.Error(t, err)
	}
}

func TestRevocationRequiresOwningSubject(t *testing.T) {
	store, _, _ := newConsentFixture(t)
	_, grant, err := store.Grant(context.Background(), "user-a", "own-key", time.Hour)
	require.NoError(t, err)
	require.Error(t, store.Revoke(context.Background(), "user-b", grant.ID))
}

func TestAudioToolsConsentIsSourceBoundAndExpires(t *testing.T) {
	store, _, now := newConsentFixture(t)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "audio-owner", Verified: true})
	receipt, _, err := store.Grant(ctx, "audio-owner", "audio-tools-openai-gpt-live", time.Minute)
	require.NoError(t, err)
	require.NoError(t, store.Verify(ctx, receipt, "audio-tools-openai-gpt-live"))
	require.ErrorIs(t, store.Verify(ctx, receipt, "own-key"), ErrInvalidGrant)
	*now = now.Add(time.Minute)
	require.ErrorIs(t, store.Verify(ctx, receipt, "audio-tools-openai-gpt-live"), ErrInvalidGrant)
}
