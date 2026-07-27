package enrichment

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"
	localdb "signal-inbox/internal/database"
	"signal-inbox/internal/signals"
	"signal-inbox/internal/testutil/db"
	"signal-inbox/internal/testutil/mocks"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (fn doerFunc) Do(request *http.Request) (*http.Response, error) { return fn(request) }

type runnerFunc func(context.Context, string) ([]byte, error)

func (fn runnerFunc) RunOCR(ctx context.Context, inputPath string) ([]byte, error) {
	return fn(ctx, inputPath)
}

func newServices(t *testing.T, doer HTTPDoer) (signals.Service, Repository) {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(signals.Schema),
		apidb.SchemaProviderFunc(Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	repo := NewSQLiteRepository(database)
	enricher := NewService(repo, clk, NewHTMLExtractor(doer))
	return signals.NewService(signals.NewSQLiteRepository(database, clk), clk, enricher), repo
}

// [REQ:SIG-P0-003] A client-rendered page with only navigation chrome must
// not become a blank document that looks successfully extracted.
func TestCaptureClientRenderedShellNeedsAttention(t *testing.T) {
	t.Log("[REQ:SIG-P0-003]")
	svc, repo := newServices(t, doerFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`<html><body><nav>Sign in</nav><script>hydrate()</script></body></html>`))}, nil
	}))

	result, err := svc.Capture(context.Background(), signals.CaptureInput{URL: "https://example.com/shared-chat"})
	require.NoError(t, err)
	require.False(t, result.Duplicate)
	require.True(t, result.Signal.NeedsAttention)
	require.Empty(t, result.Signal.ExtractedContent)

	record, found, err := repo.Latest(context.Background(), result.Signal.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Zero(t, record.ContentUnits)
	require.True(t, record.NeedsAttention)
	require.Empty(t, record.ExtractedContent)
}

func TestCaptureURLProjectsReadableHTMLAfterJournalAppend(t *testing.T) {
	svc, _ := newServices(t, doerFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "Mozilla/5.0 (compatible; VrooliSignalInbox/1.0; +https://vrooli.local)", request.Header.Get("User-Agent"))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewBufferString(`<html><body><nav>Navigation</nav><article><h1>Signal title</h1><p>Durable external material</p></article></body></html>`))}, nil
	}))

	result, err := svc.Capture(context.Background(), signals.CaptureInput{URL: "https://example.com/article"})
	require.NoError(t, err)
	require.False(t, result.Signal.NeedsAttention)
	require.Equal(t, "Signal title Durable external material", result.Signal.ExtractedContent)
}

func TestUnavailableExtractorLeavesCaptureDurableAndNeedsAttention(t *testing.T) {
	svc, repo := newServices(t, doerFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	}))

	result, err := svc.Capture(context.Background(), signals.CaptureInput{URL: "https://example.com/unavailable"})
	require.NoError(t, err)
	require.NotEmpty(t, result.Signal.ID)
	require.True(t, result.Signal.NeedsAttention)
	require.Empty(t, result.Signal.ExtractedContent)

	record, found, err := repo.Latest(context.Background(), result.Signal.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Contains(t, record.AttentionReason, "extraction unavailable")
}

// [REQ:SIG-P0-003] Image bytes are delegated to image-tools through its
// measured CLI operation; the journal carries only the retained payload ref.
func TestCaptureImageDelegatesOCRAndProjectsText(t *testing.T) {
	t.Log("[REQ:SIG-P0-003]")
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(signals.Schema),
		apidb.SchemaProviderFunc(Schema),
	))
	store := blobstore.NewMemoryBlobStore()
	require.NoError(t, store.Put(context.Background(), "signals/uploads/example", bytes.NewBufferString("image bytes"), "image/png"))
	clk := mocks.NewFakeClock(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	runner := runnerFunc(func(_ context.Context, inputPath string) ([]byte, error) {
		materialized, err := os.ReadFile(inputPath)
		require.NoError(t, err)
		require.Equal(t, []byte("image bytes"), materialized)
		return []byte(`{"ocr":{"fullText":"Text found in the saved image"}}`), nil
	})
	repo := NewSQLiteRepository(database)
	enricher := NewService(repo, clk, NewImageExtractor(store, runner))
	svc := signals.NewService(signals.NewSQLiteRepository(database, clk), clk, enricher)

	result, err := svc.Capture(context.Background(), signals.CaptureInput{ImagePayloadRef: "signals/uploads/example"})
	require.NoError(t, err)
	require.False(t, result.Signal.NeedsAttention)
	require.Equal(t, "Text found in the saved image", result.Signal.ExtractedContent)
}

func TestUnavailableImageToolsLeavesImageCapturedAndNeedsAttention(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(signals.Schema),
		apidb.SchemaProviderFunc(Schema),
	))
	store := blobstore.NewMemoryBlobStore()
	require.NoError(t, store.Put(context.Background(), "signals/uploads/unavailable", bytes.NewBufferString("image bytes"), "image/png"))
	clk := mocks.NewFakeClock(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	runner := runnerFunc(func(context.Context, string) ([]byte, error) { return nil, context.DeadlineExceeded })
	repo := NewSQLiteRepository(database)
	enricher := NewService(repo, clk, NewImageExtractor(store, runner))
	svc := signals.NewService(signals.NewSQLiteRepository(database, clk), clk, enricher)

	result, err := svc.Capture(context.Background(), signals.CaptureInput{ImagePayloadRef: "signals/uploads/unavailable"})
	require.NoError(t, err)
	require.NotEmpty(t, result.Signal.ID)
	require.True(t, result.Signal.NeedsAttention)
	require.Empty(t, result.Signal.ExtractedContent)

	record, found, err := repo.Latest(context.Background(), result.Signal.ID)
	require.NoError(t, err)
	require.True(t, found)
	require.Contains(t, record.AttentionReason, "extraction unavailable")
}
