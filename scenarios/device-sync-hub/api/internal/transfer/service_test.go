package transfer_test

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"device-sync-hub/internal/testutil/db"
	"device-sync-hub/internal/transfer"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"

	localdb "device-sync-hub/internal/database"
)

func strReader(s string) io.Reader { return strings.NewReader(s) }

// fakeTrust is a transfer.TrustChecker that trusts a fixed allow-list.
type fakeTrust struct{ trusted map[string]bool }

func (f fakeTrust) IsTrustedDevice(_ context.Context, _ /*ownerID*/, deviceID string) (bool, error) {
	return f.trusted[deviceID], nil
}

// captureNotifier records the events the service emits so tests assert fan-out.
type captureNotifier struct {
	mu      sync.Mutex
	arrived []transfer.Item
	deleted []transfer.Item
}

func (c *captureNotifier) ItemArrived(_ context.Context, i transfer.Item) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.arrived = append(c.arrived, i)
}

func (c *captureNotifier) ItemDeleted(_ context.Context, i transfer.Item) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleted = append(c.deleted, i)
}

type harness struct {
	svc   transfer.Service
	repo  transfer.Repository
	blobs *blobstore.MemoryBlobStore
	clk   *scheduletest.FakeClock
	notif *captureNotifier
}

func newService(t *testing.T, cfg transfer.Config) harness {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(transfer.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC))
	repo := transfer.NewSQLiteRepository(d, clk)
	blobs := blobstore.NewMemoryBlobStore()
	notif := &captureNotifier{}

	cfg.Repo = repo
	cfg.Blobs = blobs
	cfg.Clock = clk
	if cfg.Notif == nil {
		cfg.Notif = notif
	}
	return harness{svc: transfer.NewService(cfg), repo: repo, blobs: blobs, clk: clk, notif: notif}
}

func TestCreateText_DefaultsAndNotifies(t *testing.T) {
	h := newService(t, transfer.Config{})
	ctx := context.Background()

	item, err := h.svc.CreateText(ctx, transfer.CreateText{OwnerID: "o", OriginDeviceID: "d", Text: "hello"})
	require.NoError(t, err)
	assert.Equal(t, transfer.KindText, item.Kind)
	assert.Equal(t, transfer.RetentionHeld, item.Retention, "default retention is Held")
	// Held → expires 24h out from the fake clock.
	assert.Equal(t, h.clk.Now().Add(24*time.Hour), item.ExpiresAt.UTC())
	require.Len(t, h.notif.arrived, 1)
	assert.Equal(t, item.ID, h.notif.arrived[0].ID)
}

func TestCreateText_RejectsBlankAndOversized(t *testing.T) {
	h := newService(t, transfer.Config{MaxTextBytes: 8})
	ctx := context.Background()

	_, err := h.svc.CreateText(ctx, transfer.CreateText{OwnerID: "o", OriginDeviceID: "d", Text: "   "})
	assert.ErrorAs(t, err, &transfer.ErrInvalidItem{})

	_, err = h.svc.CreateText(ctx, transfer.CreateText{OwnerID: "o", OriginDeviceID: "d", Text: "way too long"})
	assert.ErrorAs(t, err, &transfer.ErrInvalidItem{})
}

func TestRetentionExpiry(t *testing.T) {
	h := newService(t, transfer.Config{LiveTTL: 5 * time.Minute, HeldTTL: time.Hour})
	ctx := context.Background()

	live, err := h.svc.CreateText(ctx, transfer.CreateText{OwnerID: "o", OriginDeviceID: "d", Text: "x", Retention: transfer.RetentionLive})
	require.NoError(t, err)
	assert.Equal(t, h.clk.Now().Add(5*time.Minute), live.ExpiresAt.UTC())

	pinned, err := h.svc.CreateText(ctx, transfer.CreateText{OwnerID: "o", OriginDeviceID: "d", Text: "y", Retention: transfer.RetentionPinned})
	require.NoError(t, err)
	assert.True(t, pinned.ExpiresAt.IsZero(), "Pinned never expires")
}

func TestQuotaEnforced(t *testing.T) {
	h := newService(t, transfer.Config{OwnerQuotaBytes: 100, DeviceQuotaBytes: 100})
	ctx := context.Background()

	// Store an 80-byte file via the file path.
	require.NoError(t, h.blobs.Put(ctx, "k1", strReader("12345678901234567890123456789012345678901234567890123456789012345678901234567890"), "application/octet-stream"))
	_, err := h.svc.CreateFile(ctx, transfer.CreateFile{OwnerID: "o", OriginDeviceID: "d", SizeBytes: 80, BlobKey: "k1", Name: "f"})
	require.NoError(t, err)

	// A further 30 bytes would breach the 100-byte owner quota.
	err = h.svc.CheckQuota(ctx, "o", "d", 30)
	var quota transfer.ErrQuotaExceeded
	require.ErrorAs(t, err, &quota)
	assert.Equal(t, "owner", quota.Scope)
}

func TestDirectedTargetValidation(t *testing.T) {
	h := newService(t, transfer.Config{Trust: fakeTrust{trusted: map[string]bool{"good": true}}})
	ctx := context.Background()

	_, err := h.svc.CreateText(ctx, transfer.CreateText{OwnerID: "o", OriginDeviceID: "d", Text: "x", TargetDeviceID: "stranger"})
	assert.ErrorAs(t, err, &transfer.ErrInvalidTarget{})

	ok, err := h.svc.CreateText(ctx, transfer.CreateText{OwnerID: "o", OriginDeviceID: "d", Text: "x", TargetDeviceID: "good"})
	require.NoError(t, err)
	assert.Equal(t, "good", ok.TargetDeviceID)
}

func TestDeleteRemovesBlobsAndNotifies(t *testing.T) {
	h := newService(t, transfer.Config{})
	ctx := context.Background()
	require.NoError(t, h.blobs.Put(ctx, "blob/k", strReader("data"), "text/plain"))
	item, err := h.svc.CreateFile(ctx, transfer.CreateFile{OwnerID: "o", OriginDeviceID: "d", SizeBytes: 4, BlobKey: "blob/k", Name: "f"})
	require.NoError(t, err)

	_, err = h.svc.Delete(ctx, "o", item.ID)
	require.NoError(t, err)

	_, _, getErr := h.blobs.Get(ctx, "blob/k")
	assert.Error(t, getErr, "blob removed on delete")
	require.Len(t, h.notif.deleted, 1)
}

func TestPurgeSweep(t *testing.T) {
	h := newService(t, transfer.Config{HeldTTL: time.Hour})
	ctx := context.Background()
	require.NoError(t, h.blobs.Put(ctx, "blob/e", strReader("data"), "text/plain"))
	_, err := h.svc.CreateFile(ctx, transfer.CreateFile{OwnerID: "o", OriginDeviceID: "d", SizeBytes: 4, BlobKey: "blob/e", Name: "f", Retention: transfer.RetentionHeld})
	require.NoError(t, err)

	// Nothing due yet.
	n, err := h.svc.Purge(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	// Advance past the Held window → the sweep removes it and its blob.
	h.clk.Advance(2 * time.Hour)
	n, err = h.svc.Purge(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	_, _, getErr := h.blobs.Get(ctx, "blob/e")
	assert.Error(t, getErr, "purged item's blob removed")
}
