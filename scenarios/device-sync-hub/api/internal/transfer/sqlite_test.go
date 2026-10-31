package transfer_test

import (
	"context"
	"testing"
	"time"

	"device-sync-hub/internal/transfer"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "device-sync-hub/internal/database"
)

// newRepo returns a sqlite-backed transfer.Repository with the production schema
// applied — the canonical repository-test compose pattern.
func newRepo(t *testing.T) (transfer.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(transfer.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))
	return transfer.NewSQLiteRepository(d, clk), clk
}

func TestSQLiteItemRoundTrip(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, transfer.Item{
		OwnerID:        "owner-1",
		OriginDeviceID: "dev-a",
		Kind:           transfer.KindText,
		Name:           "note",
		MIME:           "text/plain",
		SizeBytes:      5,
		Text:           "hello",
		Retention:      transfer.RetentionHeld,
		ExpiresAt:      clk.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	got, err := repo.GetVisible(ctx, "owner-1", "dev-a", created.ID)
	require.NoError(t, err)
	assert.Equal(t, "hello", got.Text)
	assert.Equal(t, transfer.RetentionHeld, got.Retention)
	assert.False(t, got.ExpiresAt.IsZero())
	assert.Equal(t, int64(5), got.SizeBytes)
}

func TestSQLiteVisibilityACL(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	// Broadcast item from dev-a.
	bcast, err := repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "dev-a", Kind: transfer.KindText, Text: "b", Retention: transfer.RetentionPinned})
	require.NoError(t, err)
	// Directed item from dev-a to dev-b.
	directed, err := repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "dev-a", Kind: transfer.KindText, Text: "d", Retention: transfer.RetentionPinned, TargetDeviceID: "dev-b"})
	require.NoError(t, err)

	// dev-c (a third trusted device) sees the broadcast but NOT the directed item.
	listC, err := repo.ListVisible(ctx, "o", "dev-c", transfer.ListFilter{})
	require.NoError(t, err)
	require.Len(t, listC, 1)
	assert.Equal(t, bcast.ID, listC[0].ID)

	_, err = repo.GetVisible(ctx, "o", "dev-c", directed.ID)
	assert.ErrorAs(t, err, &transfer.ErrItemNotFound{})

	// dev-b (the target) sees both.
	listB, err := repo.ListVisible(ctx, "o", "dev-b", transfer.ListFilter{})
	require.NoError(t, err)
	assert.Len(t, listB, 2)

	// The origin always sees its own directed item.
	gotA, err := repo.GetVisible(ctx, "o", "dev-a", directed.ID)
	require.NoError(t, err)
	assert.Equal(t, directed.ID, gotA.ID)

	// Cross-owner isolation: a different owner sees nothing.
	listOther, err := repo.ListVisible(ctx, "other-owner", "dev-a", transfer.ListFilter{})
	require.NoError(t, err)
	assert.Empty(t, listOther)
}

func TestSQLiteListFilters(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, _ = repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "d", Kind: transfer.KindText, Name: "Report", Text: "quarterly numbers", Retention: transfer.RetentionPinned})
	_, _ = repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "d", Kind: transfer.KindFile, Name: "photo.png", MIME: "image/png", SizeBytes: 10, BlobKey: "k", Retention: transfer.RetentionPinned})

	files, err := repo.ListVisible(ctx, "o", "d", transfer.ListFilter{Kind: transfer.KindFile})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "photo.png", files[0].Name)

	// Case-insensitive substring over name OR text.
	q, err := repo.ListVisible(ctx, "o", "d", transfer.ListFilter{Query: "QUARTERLY"})
	require.NoError(t, err)
	require.Len(t, q, 1)
	assert.Equal(t, "Report", q[0].Name)
}

func TestSQLiteUsage(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, _ = repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "dev-a", Kind: transfer.KindFile, SizeBytes: 100, BlobKey: "k1", Retention: transfer.RetentionPinned})
	_, _ = repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "dev-b", Kind: transfer.KindFile, SizeBytes: 250, BlobKey: "k2", Retention: transfer.RetentionPinned})

	owner, err := repo.UsageByOwner(ctx, "o")
	require.NoError(t, err)
	assert.Equal(t, int64(350), owner)

	devA, err := repo.UsageByDevice(ctx, "o", "dev-a")
	require.NoError(t, err)
	assert.Equal(t, int64(100), devA)
}

func TestSQLiteDueForPurge(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()
	now := clk.Now()

	// Pinned: never due.
	pinned, _ := repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "d", Kind: transfer.KindText, Text: "p", Retention: transfer.RetentionPinned})
	// Held, already expired.
	expired, _ := repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "d", Kind: transfer.KindText, Text: "e", Retention: transfer.RetentionHeld, ExpiresAt: now.Add(-time.Minute)})
	// Held, not yet expired.
	fresh, _ := repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "d", Kind: transfer.KindText, Text: "f", Retention: transfer.RetentionHeld, ExpiresAt: now.Add(time.Hour)})
	// Live, delivered → due regardless of expiry.
	live, _ := repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "d", Kind: transfer.KindText, Text: "l", Retention: transfer.RetentionLive, Delivered: true, ExpiresAt: now.Add(time.Hour)})

	due, err := repo.DueForPurge(ctx, now)
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, i := range due {
		ids[i.ID] = true
	}
	assert.True(t, ids[expired.ID], "expired Held is due")
	assert.True(t, ids[live.ID], "delivered Live is due")
	assert.False(t, ids[fresh.ID], "fresh Held is not due")
	assert.False(t, ids[pinned.ID], "Pinned is never due")
}

func TestSQLiteDeleteReturnsKeys(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	created, _ := repo.Create(ctx, transfer.Item{OwnerID: "o", OriginDeviceID: "d", Kind: transfer.KindFile, SizeBytes: 1, BlobKey: "blob/x", ThumbKey: "blob/x.thumb", Retention: transfer.RetentionPinned})

	deleted, err := repo.Delete(ctx, "o", created.ID)
	require.NoError(t, err)
	assert.Equal(t, "blob/x", deleted.BlobKey)
	assert.Equal(t, "blob/x.thumb", deleted.ThumbKey)

	_, err = repo.GetByOwner(ctx, "o", created.ID)
	assert.ErrorAs(t, err, &transfer.ErrItemNotFound{})

	// Deleting a missing item is ErrItemNotFound.
	_, err = repo.Delete(ctx, "o", "nope")
	assert.ErrorAs(t, err, &transfer.ErrItemNotFound{})
}
