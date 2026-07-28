package sources

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"signal-inbox/internal/clock"
	localdb "signal-inbox/internal/database"
	"signal-inbox/internal/signals"
	"signal-inbox/internal/testutil/db"
	"signal-inbox/internal/testutil/mocks"
)

type fakeCapture struct{ seen map[string]bool }

func (f *fakeCapture) Capture(_ context.Context, input signals.CaptureInput) (signals.CaptureResult, error) {
	if f.seen == nil {
		f.seen = map[string]bool{}
	}
	if f.seen[input.URL] {
		return signals.CaptureResult{Duplicate: true}, nil
	}
	f.seen[input.URL] = true
	return signals.CaptureResult{}, nil
}

type adapter struct {
	descriptor Descriptor
	err        error
	calls      int
}

func (a *adapter) Descriptor() Descriptor { return a.descriptor }
func (a *adapter) Parse(_ context.Context, _ io.Reader) ([]signals.CaptureInput, error) {
	a.calls++
	if a.err != nil {
		return nil, a.err
	}
	return []signals.CaptureInput{{URL: "https://example.test/a"}}, nil
}

func testService(t *testing.T, adapter Adapter) (*Service, *sqliteRepository, *mocks.FakeClock) {
	t.Helper()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(Schema)))
	clk := mocks.NewFakeClock(time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC))
	repo := NewSQLiteRepository(database).(*sqliteRepository)
	service, err := NewService(repo, &fakeCapture{}, clk, adapter)
	require.NoError(t, err)
	return service, repo, clk
}

func TestRegistryRejectsUndeclaredRiskTier(t *testing.T) {
	t.Log("[REQ:SIG-P0-014]")
	_, err := NewService(nil, nil, clock.System{}, &adapter{descriptor: Descriptor{ID: "bad", Kind: "test"}})
	require.ErrorAs(t, err, new(ErrInvalidDescriptor))
}

func TestHigherRiskAdapterStartsDisabledAndPersists(t *testing.T) {
	t.Log("[REQ:SIG-P0-014]")
	a := &adapter{descriptor: Descriptor{ID: "network", Kind: "official_api", RiskTier: RiskTier1}}
	service, repo, clk := testService(t, a)
	state, err := service.State(context.Background(), "network")
	require.NoError(t, err)
	require.False(t, state.Enabled)
	_, err = service.Import(context.Background(), "network", bytes.NewReader(nil))
	require.ErrorAs(t, err, new(ErrAdapterDisabled))
	second, err := NewService(repo, &fakeCapture{}, clk, a)
	require.NoError(t, err)
	restored, err := second.State(context.Background(), "network")
	require.NoError(t, err)
	require.False(t, restored.Enabled)
}

func TestAnomalyDisablesWithoutRetryAndSurvivesRestart(t *testing.T) {
	t.Log("[REQ:SIG-P0-014]")
	for _, reason := range []string{"HTTP 429 rate limited", "HTTP 403 forbidden", "challenge page detected"} {
		t.Run(reason, func(t *testing.T) {
			a := &adapter{descriptor: Descriptor{ID: "network", Kind: "official_api", RiskTier: RiskTier1}, err: ErrAnomalousResponse{Reason: reason}}
			service, repo, clk := testService(t, a)
			_, err := service.SetEnabled(context.Background(), "network", true)
			require.NoError(t, err)
			_, err = service.Import(context.Background(), "network", bytes.NewReader(nil))
			require.Error(t, err)
			require.Equal(t, 1, a.calls, "the runner has no retry path")
			state, err := service.State(context.Background(), "network")
			require.NoError(t, err)
			require.False(t, state.Enabled)
			require.Equal(t, reason, state.DisabledReason)
			second, err := NewService(repo, &fakeCapture{}, clk, a)
			require.NoError(t, err)
			restored, err := second.State(context.Background(), "network")
			require.NoError(t, err)
			require.False(t, restored.Enabled)
		})
	}
}

func TestChromeImportIsIdempotentAndRejectsWrongShape(t *testing.T) {
	t.Log("[REQ:SIG-P0-008]")
	service, _, _ := testService(t, ChromeBookmarksAdapter{})
	export := []byte(`<!DOCTYPE NETSCAPE-Bookmark-file-1><DL><p><DT><A HREF="https://example.test/a">A</A><DT><A HREF="https://example.test/b">B</A></DL><p>`)
	first, err := service.Import(context.Background(), ChromeBookmarksAdapterID, bytes.NewReader(export))
	require.NoError(t, err)
	require.Equal(t, 2, first.Created)
	second, err := service.Import(context.Background(), ChromeBookmarksAdapterID, bytes.NewReader(export))
	require.NoError(t, err)
	require.Zero(t, second.Created)
	require.Equal(t, 2, second.Duplicated)
	_, err = service.Import(context.Background(), ChromeBookmarksAdapterID, bytes.NewReader([]byte("not bookmarks")))
	require.Error(t, err)
	require.False(t, errors.Is(err, context.Canceled))
}

func TestChromeImportPreservesMeasuredFolderPathAsTags(t *testing.T) {
	t.Log("[REQ:SIG-P0-008] [REQ:SIG-P0-009]")
	adapter := ChromeBookmarksAdapter{}
	entries, err := adapter.Parse(context.Background(), bytes.NewReader([]byte(`<!DOCTYPE NETSCAPE-Bookmark-file-1><DL><p><DT><H3>Bookmarks bar</H3><DL><p><DT><H3>Research</H3><DL><p><DT><A HREF="https://example.test/a">A</A></DL><p></DL><p></DL><p>`)))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, []string{"Bookmarks bar", "Research"}, entries[0].Tags)
}
