package signals

import (
	"context"
	"testing"
	"time"

	db "github.com/vrooli/api-core/databasetest"
	localdb "signal-inbox/internal/database"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/scheduletest"
)

func newTestService(t *testing.T) Service {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(Schema)))
	clk := scheduletest.New(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	return NewService(NewSQLiteRepository(database, clk), clk)
}

// [REQ:SIG-P0-001] Capture accepts one source kind and appends immutable evidence.
func TestCaptureTextAppendsSignal(t *testing.T) {
	t.Log("[REQ:SIG-P0-001] [REQ:SIG-P0-002]")
	svc := newTestService(t)
	result, err := svc.Capture(context.Background(), CaptureInput{Text: "  durable   external  signal ", CaptureNote: "review later"})
	require.NoError(t, err)
	require.False(t, result.Duplicate)
	require.Equal(t, SourceKindText, result.Signal.SourceKind)
	require.Equal(t, "durable external signal", result.Signal.ExtractedContent)
	require.Equal(t, "review later", result.Signal.CaptureNote)
	require.NotEmpty(t, result.Signal.ID)
}

// [REQ:SIG-P0-002] Content-equivalent capture returns the original record.
func TestCaptureURLTrackingVariantDeduplicates(t *testing.T) {
	t.Log("[REQ:SIG-P0-001] [REQ:SIG-P0-002]")
	svc := newTestService(t)
	first, err := svc.Capture(context.Background(), CaptureInput{URL: "HTTPS://Example.COM/path?utm_source=x&z=1#frag"})
	require.NoError(t, err)
	second, err := svc.Capture(context.Background(), CaptureInput{URL: "https://example.com/path?z=1&utm_campaign=y"})
	require.NoError(t, err)
	require.False(t, first.Duplicate)
	require.True(t, second.Duplicate)
	require.Equal(t, first.Signal.ID, second.Signal.ID)
	listed, err := svc.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, listed, 1)
}

func TestCaptureDifferentURLsDoNotDeduplicate(t *testing.T) {
	svc := newTestService(t)
	first, err := svc.Capture(context.Background(), CaptureInput{URL: "https://example.com/a"})
	require.NoError(t, err)
	second, err := svc.Capture(context.Background(), CaptureInput{URL: "https://example.com/b"})
	require.NoError(t, err)
	require.NotEqual(t, first.Signal.ID, second.Signal.ID)
}

func TestCaptureRejectsIncompatibleSourceCombinations(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Capture(context.Background(), CaptureInput{Text: "also text", ImagePayloadRef: "signals/uploads/image"})
	require.ErrorAs(t, err, new(ErrInvalidSignal))
}

func TestCaptureRetainsArchiveTextWithURL(t *testing.T) {
	svc := newTestService(t)
	result, err := svc.Capture(context.Background(), CaptureInput{URL: "https://x.com/i/web/status/1", Text: "archive post text"})
	require.NoError(t, err)
	require.Equal(t, SourceKindURL, result.Signal.SourceKind)
	require.Equal(t, "archive post text", result.Signal.ExtractedContent)
}

// [REQ:SIG-P0-002] The caller supplies only an image payload reference; the
// journal infers the image kind and keeps bytes outside the proto request.
func TestCaptureImageInfersKind(t *testing.T) {
	t.Log("[REQ:SIG-P0-002]")
	svc := newTestService(t)
	result, err := svc.Capture(context.Background(), CaptureInput{ImagePayloadRef: "signals/uploads/content-addressed-image"})
	require.NoError(t, err)
	require.Equal(t, SourceKindImage, result.Signal.SourceKind)
	require.True(t, result.Signal.NeedsAttention)
	require.Equal(t, "signals/uploads/content-addressed-image", result.Signal.RawPayloadRef)
}
