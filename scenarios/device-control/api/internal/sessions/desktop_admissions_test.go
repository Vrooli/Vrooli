package sessions

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/localprincipal"
)

func TestDesktopAdmissionMetadataIsImmutableAndActorScoped(t *testing.T) {
	_, repo, _, _, lease := desktopFixture(t)
	ctx := context.Background()
	store, err := NewSQLiteDesktopAdmissions(ctx, repo.db)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	grant := DesktopGrant{ID: "original", Principal: principal, Lease: lease, Operations: []string{"stop"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	require.NoError(t, store.Put(ctx, grant))
	require.NoError(t, store.Put(ctx, grant))
	reconstructed, err := NewSQLiteDesktopAdmissions(ctx, repo.db)
	require.NoError(t, err)
	got, err := reconstructed.Get(ctx, lease.Ref, lease.Actor)
	require.NoError(t, err)
	require.Equal(t, grant.ID, got.ID)
	require.True(t, sameDesktopLease(grant.Lease, got.Lease))
	_, err = reconstructed.Get(ctx, lease.Ref, "another-actor")
	require.ErrorIs(t, err, sql.ErrNoRows)
	changed := grant
	changed.ID = "replacement"
	require.ErrorIs(t, store.Put(ctx, changed), ErrDesktopAdmission)
	changed = grant
	changed.Lease.Actor = "another-actor"
	require.ErrorIs(t, store.Put(ctx, changed), ErrDesktopAdmission)
	changed = grant
	changed.Lease.Epoch++
	require.ErrorIs(t, store.Put(ctx, changed), ErrDesktopAdmission)
	wrong := lease.Ref
	wrong.DesktopSessionID = "other-desktop"
	_, err = reconstructed.Get(ctx, wrong, lease.Actor)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	got, err = store.Get(ctx, lease.Ref, lease.Actor)
	require.NoError(t, err)
	require.Equal(t, grant.ID, got.ID)
}

func TestDesktopAdmissionDiscoverySurvivesRestartAndScopesEveryPage(t *testing.T) {
	_, repo, _, _, lease := desktopFixture(t)
	ctx := context.Background()
	store, err := NewSQLiteDesktopAdmissions(ctx, repo.db)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	// Historical metadata remains discoverable without refreshing its authority.
	lease.ExpiresAt = time.Now().Add(-time.Minute)
	grant := DesktopGrant{ID: "historical", Principal: principal, Lease: lease, Operations: []string{"stop"}, IssuedAt: lease.ExpiresAt.Add(-time.Minute), ExpiresAt: lease.ExpiresAt}
	for i := 0; i < 5; i++ {
		g := grant
		g.Lease.Ref.SessionID = fmt.Sprintf("admission-%d", i)
		if i == 1 {
			g.Lease.Actor = "other-actor"
		}
		if i == 2 {
			g.Lease.Ref.Surface.Target.HostNodeID = "other-host"
		}
		require.NoError(t, store.Put(ctx, g))
	}
	page, err := store.List(ctx, lease.Ref.Surface, lease.Actor, "", 1)
	require.NoError(t, err)
	require.Len(t, page.Grants, 1)
	require.Equal(t, "admission-0", page.Grants[0].Lease.Ref.SessionID)
	require.NotEmpty(t, page.NextPageToken)
	reconstructed, err := NewSQLiteDesktopAdmissions(ctx, repo.db)
	require.NoError(t, err)
	// Insertions after the first page are visited without replaying earlier rows.
	grant.Lease.Ref.SessionID = "admission-later"
	require.NoError(t, reconstructed.Put(ctx, grant))
	page, err = reconstructed.List(ctx, lease.Ref.Surface, lease.Actor, page.NextPageToken, 2)
	require.NoError(t, err)
	require.Len(t, page.Grants, 2)
	require.Equal(t, "admission-3", page.Grants[0].Lease.Ref.SessionID)
	require.Equal(t, "admission-4", page.Grants[1].Lease.Ref.SessionID)
	require.NotEmpty(t, page.NextPageToken)
	page, err = reconstructed.List(ctx, lease.Ref.Surface, lease.Actor, page.NextPageToken, 2)
	require.NoError(t, err)
	require.Len(t, page.Grants, 1)
	require.Equal(t, "admission-later", page.Grants[0].Lease.Ref.SessionID)
	require.Empty(t, page.NextPageToken)
	page, err = reconstructed.List(ctx, lease.Ref.Surface, "unknown-actor", "", 0)
	require.NoError(t, err)
	require.Empty(t, page.Grants)
	require.Empty(t, page.NextPageToken)
	for _, token := range []string{"0", "-1", "+1", "01", " 1", "9223372036854775808", "x"} {
		_, err := store.List(ctx, lease.Ref.Surface, lease.Actor, token, 1)
		require.ErrorIs(t, err, ErrDesktopAdmission, token)
	}
	for _, size := range []int{-1, 101} {
		_, err := store.List(ctx, lease.Ref.Surface, lease.Actor, "", size)
		require.ErrorIs(t, err, ErrDesktopAdmission)
	}
}

func TestDesktopOpenAttemptIsActorScopedImmutableAndForwardedOnce(t *testing.T) {
	ctx := context.Background()
	_, repo, _, _, lease := desktopFixture(t)
	store, err := NewSQLiteDesktopAdmissions(ctx, repo.db)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	grant := DesktopGrant{ID: "attempt-grant", Principal: principal, Lease: lease, Operations: []string{"stop"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	intent := DesktopOpenIntent{Surface: lease.Ref.Surface, Control: lease.Control, TTLSeconds: 120}
	id := "85cbd2b6-b42b-4934-bd09-9006cf3dfb2c"
	reserved, created, err := store.ReserveOpen(ctx, id, lease.Actor, intent, time.Now().Add(10*time.Second))
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "reserved", reserved.State)
	duplicate, created, err := store.ReserveOpen(ctx, id, lease.Actor, intent, time.Now().Add(20*time.Second))
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, reserved, duplicate, "duplicate request must not extend its execution deadline")
	changed := intent
	changed.Control = !changed.Control
	_, _, err = store.ReserveOpen(ctx, id, lease.Actor, changed, time.Now().Add(10*time.Second))
	require.ErrorIs(t, err, ErrDesktopAdmission)
	_, err = store.ReadOpen(ctx, id, "another-actor")
	require.ErrorIs(t, err, sql.ErrNoRows)
	claimed, err := store.BindOpen(ctx, id, lease.Actor, grant)
	require.NoError(t, err)
	require.True(t, claimed)
	var seq int
	var name, path string
	require.NoError(t, repo.db.QueryRowContext(ctx, "PRAGMA database_list").Scan(&seq, &name, &path))
	secondDB, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer secondDB.Close()
	reopened, err := NewSQLiteDesktopAdmissions(ctx, secondDB)
	require.NoError(t, err)
	bound, err := reopened.ReadOpen(ctx, id, lease.Actor)
	require.NoError(t, err)
	require.Equal(t, "forwarding", bound.State)
	require.NotNil(t, bound.Grant)
	require.True(t, sameDesktopLease(lease, bound.Grant.Lease))
	claimed, err = reopened.BindOpen(ctx, id, lease.Actor, grant)
	require.NoError(t, err)
	require.False(t, claimed, "helper admission must not execute twice after an unknown reply")
	grant.ID = "replacement"
	_, err = reopened.BindOpen(ctx, id, lease.Actor, grant)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.ErrorIs(t, reopened.RejectOpen(ctx, id, lease.Actor), ErrDesktopAdmission)
	require.NoError(t, reopened.ExpireOpenReservations(ctx, time.Now().Add(time.Hour)))
	bound, err = reopened.ReadOpen(ctx, id, lease.Actor)
	require.NoError(t, err)
	require.Equal(t, "forwarding", bound.State, "expiry cannot certify no admission after the helper boundary")
}

func TestDesktopOpenAttemptCancellationFencesDelayedForwarding(t *testing.T) {
	ctx := context.Background()
	_, repo, _, _, lease := desktopFixture(t)
	store, err := NewSQLiteDesktopAdmissions(ctx, repo.db)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	grant := DesktopGrant{ID: "delayed", Principal: principal, Lease: lease, Operations: []string{"stop"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	intent := DesktopOpenIntent{Surface: lease.Ref.Surface, Control: lease.Control, TTLSeconds: 120}
	for _, expiration := range []bool{false, true} {
		id := "358b3461-a8a5-499c-8a61-9f5ad854eaed"
		if expiration {
			id = "68edce76-d6e7-4f1a-acb9-b55c7a7c06b6"
		}
		deadline := time.Now().Add(10 * time.Second)
		_, created, err := store.ReserveOpen(ctx, id, lease.Actor, intent, deadline)
		require.NoError(t, err)
		require.True(t, created)
		if expiration {
			require.NoError(t, store.ExpireOpenReservations(ctx, deadline))
		} else {
			require.NoError(t, store.RejectOpen(ctx, id, lease.Actor))
		}
		closed, err := store.ReadOpen(ctx, id, lease.Actor)
		require.NoError(t, err)
		require.Equal(t, "not_admitted", closed.State)
		require.Nil(t, closed.Grant)
		claimed, err := store.BindOpen(ctx, id, lease.Actor, grant)
		require.ErrorIs(t, err, ErrDesktopAdmission)
		require.False(t, claimed)
		_, created, err = store.ReserveOpen(ctx, id, lease.Actor, intent, time.Now().Add(20*time.Second))
		require.NoError(t, err)
		require.False(t, created, "a cancelled request ID cannot reserve a new execution")
		require.NoError(t, store.RejectOpen(ctx, id, lease.Actor))
	}
}

func TestDesktopOpenBindingAndAdmissionMetadataCommitTogether(t *testing.T) {
	ctx := context.Background()
	_, repo, _, _, lease := desktopFixture(t)
	store, err := NewSQLiteDesktopAdmissions(ctx, repo.db)
	require.NoError(t, err)
	principal, err := localprincipal.Current()
	require.NoError(t, err)
	grant := DesktopGrant{ID: "original-binding", Principal: principal, Lease: lease, Operations: []string{"stop"}, IssuedAt: time.Now().Add(-time.Second), ExpiresAt: lease.ExpiresAt}
	require.NoError(t, store.Put(ctx, grant))
	request := DesktopOpenIntent{Surface: lease.Ref.Surface, Control: lease.Control, TTLSeconds: 120}
	id := "8655568e-265b-4503-b7fe-082d332ce2eb"
	_, _, err = store.ReserveOpen(ctx, id, lease.Actor, request, time.Now().Add(10*time.Second))
	require.NoError(t, err)
	conflicting := grant
	conflicting.ID = "conflicting-binding"
	claimed, err := store.BindOpen(ctx, id, lease.Actor, conflicting)
	require.ErrorIs(t, err, ErrDesktopAdmission)
	require.False(t, claimed)
	attempt, err := store.ReadOpen(ctx, id, lease.Actor)
	require.NoError(t, err)
	require.Equal(t, "reserved", attempt.State, "metadata rejection must roll back the forwarding claim")
	require.Nil(t, attempt.Grant)
	closed, err := store.ReconcileOpen(ctx, id, lease.Actor, request)
	require.NoError(t, err)
	require.Equal(t, "not_admitted", closed.State)
	original, err := store.Get(ctx, lease.Ref, lease.Actor)
	require.NoError(t, err)
	require.Equal(t, grant.ID, original.ID)
}
