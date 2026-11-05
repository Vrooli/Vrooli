package contextcapture

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

type failingBlobs struct {
	blobstore.BlobStore
	failPut, failDelete bool
}

func (b *failingBlobs) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if err := b.BlobStore.Put(ctx, key, r, mime); err != nil {
		return err
	}
	if b.failPut {
		return errors.New("injected lost blob write reply")
	}
	return nil
}

func (b *failingBlobs) Delete(ctx context.Context, key string) error {
	if b.failDelete {
		return errors.New("injected deletion failure")
	}
	return b.BlobStore.Delete(ctx, key)
}

func TestDurableContextOwnershipExpiryAndReconstruction(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	blobs := blobstore.NewFilesystemBlobStore(t.TempDir())
	input, now := captureFixture(t)
	svc := NewService(repo, blobs, func() time.Time { return now })
	doc, err := svc.Import(ctx, "alice", input, time.Hour)
	require.NoError(t, err)
	// Recreate both adapters over the durable database and existing blob owner.
	svc = NewService(NewSQLiteRepository(db), blobs, func() time.Time { return now })
	read, pixels, err := svc.Read(ctx, "alice", doc.ID)
	require.NoError(t, err)
	require.Equal(t, doc, read)
	require.NotEmpty(t, pixels)
	_, pixels, err = svc.Read(ctx, "bob", doc.ID)
	require.ErrorIs(t, err, ErrUnavailable)
	require.Nil(t, pixels)
	require.ErrorIs(t, svc.Delete(ctx, "bob", doc.ID), ErrUnavailable)
	_, _, err = svc.Read(ctx, "alice", doc.ID)
	require.NoError(t, err)
	now = doc.ExpiresAt
	_, pixels, err = svc.Read(ctx, "alice", doc.ID)
	require.ErrorIs(t, err, ErrUnavailable)
	require.Nil(t, pixels)
	require.NoError(t, svc.Reap(ctx, 10))
	_, _, err = blobs.Get(ctx, imageKey(doc.ID))
	require.Error(t, err)
	var rows int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM context_capsules").Scan(&rows))
	require.Zero(t, rows)
}

func TestFailedImportKeepsTrackedStagingUntilCleanup(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	base := blobstore.NewMemoryBlobStore()
	blobs := &failingBlobs{BlobStore: base, failPut: true, failDelete: true}
	input, now := captureFixture(t)
	svc := NewService(repo, blobs, func() time.Time { return now })
	doc, err := svc.Import(ctx, "alice", input, time.Minute)
	require.Error(t, err)
	require.Empty(t, doc.ID)
	var id, state string
	require.NoError(t, db.QueryRow("SELECT id,state FROM context_capsules").Scan(&id, &state))
	require.Equal(t, "staging", state)
	_, pixels, err := svc.Read(ctx, "alice", id)
	require.ErrorIs(t, err, ErrUnavailable)
	require.Nil(t, pixels)
	now = now.Add(time.Minute)
	require.Error(t, svc.Reap(ctx, 10))
	blobs.failDelete = false
	require.NoError(t, svc.Reap(ctx, 10))
	_, _, err = base.Get(ctx, imageKey(id))
	require.Error(t, err)
}

func TestStagedCrashAndTamperedBlobCannotBeRead(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	blobs := blobstore.NewMemoryBlobStore()
	input, now := captureFixture(t)
	doc, pixels, err := Prepare("alice", input, now, time.Hour)
	require.NoError(t, err)
	require.NoError(t, repo.Reserve(ctx, doc, int64(len(pixels))))
	require.NoError(t, blobs.Put(ctx, imageKey(doc.ID), bytes.NewReader(pixels), "image/png"))
	svc := NewService(repo, blobs, func() time.Time { return now })
	_, _, err = svc.Read(ctx, "alice", doc.ID)
	require.ErrorIs(t, err, ErrUnavailable)
	require.NoError(t, repo.WithRecord(ctx, "alice", doc.ID, func(Record) (Action, error) { return Publish, nil }))
	pixels[0] ^= 1
	require.NoError(t, blobs.Put(ctx, imageKey(doc.ID), bytes.NewReader(pixels), "image/png"))
	_, returned, err := svc.Read(ctx, "alice", doc.ID)
	require.ErrorIs(t, err, ErrUnavailable)
	require.Nil(t, returned)
	require.NoError(t, svc.Delete(ctx, "alice", doc.ID))
}

func TestQuotaCountsReservationsAndReleasesOnlyAfterRemoval(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	input, now := captureFixture(t)
	doc, _, err := Prepare("alice", input, now, time.Hour)
	require.NoError(t, err)
	first := ""
	for i := 0; i < 8; i++ {
		doc.ID = uuid.NewString()
		doc.RequestID = uuid.NewString()
		if i == 0 {
			first = doc.ID
		}
		require.NoError(t, repo.Reserve(ctx, doc, MaxImageBytes))
	}
	doc.ID = uuid.NewString()
	doc.RequestID = uuid.NewString()
	require.ErrorIs(t, repo.Reserve(ctx, doc, 1), ErrQuota)
	require.Error(t, repo.WithRecord(ctx, "alice", first, func(Record) (Action, error) { return Remove, errors.New("cleanup failed") }))
	require.ErrorIs(t, repo.Reserve(ctx, doc, 1), ErrQuota)
	require.NoError(t, repo.WithRecord(ctx, "alice", first, func(Record) (Action, error) { return Remove, nil }))
	require.NoError(t, repo.Reserve(ctx, doc, 1))
}

func TestImportRetryPreservesIdentityRetentionAndDeletion(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	blobs := &failingBlobs{BlobStore: blobstore.NewMemoryBlobStore()}
	input, now := captureFixture(t)
	input.RequestID = uuid.NewString()
	svc := NewService(repo, blobs, func() time.Time { return now })
	first, err := svc.Import(ctx, "alice", input, time.Hour)
	require.NoError(t, err)
	blobs.failPut = true // Any repeated blob write would now fail.
	// A lost reply can be resolved after source-capture freshness has elapsed.
	now = now.Add(time.Minute)
	svc = NewService(NewSQLiteRepository(db), blobs, func() time.Time { return now })
	retry, err := svc.Import(ctx, "alice", input, time.Hour)
	require.NoError(t, err)
	require.Equal(t, first, retry)
	var rows int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM context_capsules").Scan(&rows))
	require.Equal(t, 1, rows)
	changed := input
	changed.Region.X = 0
	_, err = svc.Import(ctx, "alice", changed, time.Hour)
	require.ErrorIs(t, err, ErrConflict)
	_, err = svc.Import(ctx, "alice", input, 2*time.Hour)
	require.ErrorIs(t, err, ErrConflict)
	require.NoError(t, svc.Delete(ctx, "alice", first.ID))
	_, err = svc.Import(ctx, "alice", input, time.Hour)
	require.ErrorIs(t, err, ErrUnavailable)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM context_capsules").Scan(&rows))
	require.Zero(t, rows)
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM context_import_intents").Scan(&rows))
	require.Equal(t, 1, rows)
	now = first.CreatedAt.Add(24 * time.Hour)
	require.NoError(t, svc.Reap(ctx, 64))
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM context_import_intents").Scan(&rows))
	require.Zero(t, rows)
	_, err = svc.Import(ctx, "alice", input, time.Hour)
	require.ErrorIs(t, err, ErrInvalid, "old capture cannot be imported after receipt expiry")
}

func TestReconcileImportReportsPublicationWithoutBlobAccess(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	input, now := captureFixture(t)
	input.RequestID = uuid.NewString()
	doc, pixels, err := Prepare("alice", input, now, time.Hour)
	require.NoError(t, err)
	// A nil blob adapter makes any accidental blob access fail this test.
	svc := NewService(repo, nil, func() time.Time { return now })
	check := func(owner, state string, wantDoc bool) {
		t.Helper()
		status, err := svc.ReconcileImport(ctx, owner, input.RequestID)
		require.NoError(t, err)
		require.Equal(t, state, status.State)
		if wantDoc {
			require.Equal(t, &doc, status.Document)
		} else {
			require.Nil(t, status.Document)
		}
	}
	check("alice", "absent", false)
	require.NoError(t, repo.Reserve(ctx, doc, int64(len(pixels))))
	check("alice", "staging", false)
	check("bob", "absent", false)
	require.NoError(t, repo.WithRecord(ctx, "alice", doc.ID, func(Record) (Action, error) { return Publish, nil }))
	now = now.Add(time.Minute)
	check("alice", "ready", true)
	now = doc.ExpiresAt
	check("alice", "unavailable", false)
	require.NoError(t, repo.WithRecord(ctx, "alice", doc.ID, func(Record) (Action, error) { return Remove, nil }))
	check("alice", "unavailable", false)
	_, err = svc.ReconcileImport(ctx, "alice", "bad-id")
	require.ErrorIs(t, err, ErrInvalid)
	require.NoError(t, db.Close())
	_, err = svc.ReconcileImport(ctx, "alice", input.RequestID)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrUnavailable)
}

func TestCancelImportFencesUnknownAndPreparedRequests(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	input, now := captureFixture(t)
	input.RequestID = uuid.NewString()
	prepared, pixels, err := Prepare("alice", input, now, time.Hour)
	require.NoError(t, err)
	svc := NewService(repo, nil, func() time.Time { return now })
	require.NoError(t, svc.CancelImport(ctx, "alice", input.RequestID))
	receipt, err := repo.LookupIntent(ctx, "alice", input.RequestID)
	require.NoError(t, err)
	require.Empty(t, receipt.DocumentID)
	require.ErrorIs(t, repo.Reserve(ctx, prepared, int64(len(pixels))), ErrIntentExists)
	_, err = svc.Import(ctx, "alice", input, time.Hour)
	require.ErrorIs(t, err, ErrUnavailable)
	status, err := svc.ReconcileImport(ctx, "alice", input.RequestID)
	require.NoError(t, err)
	require.Equal(t, "unavailable", status.State)
	status, err = svc.ReconcileImport(ctx, "bob", input.RequestID)
	require.NoError(t, err)
	require.Equal(t, "absent", status.State)
	now = now.Add(time.Minute)
	require.NoError(t, svc.CancelImport(ctx, "alice", input.RequestID))
	repeated, err := repo.LookupIntent(ctx, "alice", input.RequestID)
	require.NoError(t, err)
	require.Equal(t, receipt, repeated)
}

func TestCancelImportRemovesStagedAndReadyPixelsAndKeepsFailedCleanupTracked(t *testing.T) {
	for _, publish := range []bool{false, true} {
		t.Run(map[bool]string{false: "staging", true: "ready"}[publish], func(t *testing.T) {
			ctx := context.Background()
			db := databasetest.NewSQLite(t)
			require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
			repo := NewSQLiteRepository(db)
			input, now := captureFixture(t)
			input.RequestID = uuid.NewString()
			doc, pixels, err := Prepare("alice", input, now, time.Hour)
			require.NoError(t, err)
			require.NoError(t, repo.Reserve(ctx, doc, int64(len(pixels))))
			base := blobstore.NewMemoryBlobStore()
			blobs := &failingBlobs{BlobStore: base, failDelete: true}
			require.NoError(t, base.Put(ctx, imageKey(doc.ID), bytes.NewReader(pixels), "image/png"))
			if publish {
				require.NoError(t, repo.WithRecord(ctx, "alice", doc.ID, func(Record) (Action, error) { return Publish, nil }))
			}
			svc := NewService(repo, blobs, func() time.Time { return now })
			require.Error(t, svc.CancelImport(ctx, "alice", input.RequestID))
			require.NoError(t, repo.WithRecord(ctx, "alice", doc.ID, func(Record) (Action, error) { return Keep, nil }))
			blobs.failDelete = false
			require.NoError(t, svc.CancelImport(ctx, "alice", input.RequestID))
			_, _, err = base.Get(ctx, imageKey(doc.ID))
			require.Error(t, err)
			require.ErrorIs(t, repo.WithRecord(ctx, "alice", doc.ID, func(Record) (Action, error) { return Publish, nil }), ErrUnavailable)
			_, err = svc.Import(ctx, "alice", input, time.Hour)
			require.ErrorIs(t, err, ErrUnavailable)
			require.NoError(t, svc.CancelImport(ctx, "alice", input.RequestID))
		})
	}
}
