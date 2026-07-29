package sources

import (
	"archive/zip"
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
	require.Equal(t, uint32(2), first.Created)
	second, err := service.Import(context.Background(), ChromeBookmarksAdapterID, bytes.NewReader(export))
	require.NoError(t, err)
	require.Zero(t, second.Created)
	require.Equal(t, uint32(2), second.Duplicated)
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

func TestRedditSavedArchiveImportsOnlySavedMaterialAndIsIdempotent(t *testing.T) {
	t.Log("[REQ:SIG-P0-008]")
	service, _, _ := testService(t, RedditSavedArchiveAdapter{})
	export := redditArchive(t, map[string]string{
		"saved_posts.csv":    "id,permalink\npost-1,https://www.reddit.com/r/example/comments/post-1/saved/\n",
		"saved_comments.csv": "id,permalink\ncomment-1,https://www.reddit.com/r/example/comments/post-2/saved/comment-1/\n",
		"comments.csv":       "id,permalink,body\nprivate,https://www.reddit.com/r/example/comments/private/,operator-authored\n",
	})
	first, err := service.Import(context.Background(), RedditSavedArchiveAdapterID, bytes.NewReader(export))
	require.NoError(t, err)
	require.Equal(t, uint32(2), first.Created)
	second, err := service.Import(context.Background(), RedditSavedArchiveAdapterID, bytes.NewReader(export))
	require.NoError(t, err)
	require.Zero(t, second.Created)
	require.Equal(t, uint32(2), second.Duplicated)

	entries, err := (RedditSavedArchiveAdapter{}).Parse(context.Background(), bytes.NewReader(export))
	require.NoError(t, err)
	require.Equal(t, []string{"reddit", "saved"}, entries[0].Tags)
}

func TestRedditSavedArchiveRejectsWrongShape(t *testing.T) {
	_, err := (RedditSavedArchiveAdapter{}).Parse(context.Background(), bytes.NewReader(redditArchive(t, map[string]string{"posts.csv": "id,permalink\npost,https://example.test/\n"})))
	require.ErrorContains(t, err, "saved_posts.csv and saved_comments.csv are absent")
}

func TestRedditSavedArchiveBoundsCompressedAndExpandedInput(t *testing.T) {
	archive := redditArchive(t, map[string]string{
		"saved_posts.csv": "id,permalink\npost-1,https://www.reddit.com/r/example/comments/post-1/saved/\n",
	})

	_, err := parseRedditSavedArchive(bytes.NewReader(archive), redditArchiveLimits{maxArchiveBytes: int64(len(archive) - 1), maxSavedCSVBytes: 1_024, maxSavedEntries: 10})
	require.ErrorContains(t, err, "archive exceeds")

	_, err = parseRedditSavedArchive(bytes.NewReader(archive), redditArchiveLimits{maxArchiveBytes: 1_024, maxSavedCSVBytes: 8, maxSavedEntries: 10})
	require.ErrorContains(t, err, "saved CSV exceeds")
}

func TestRedditSavedArchiveBoundsCaptureCount(t *testing.T) {
	archive := redditArchive(t, map[string]string{
		"saved_posts.csv": "id,permalink\npost-1,https://www.reddit.com/r/example/comments/post-1/saved/\npost-2,https://www.reddit.com/r/example/comments/post-2/saved/\n",
	})
	_, err := parseRedditSavedArchive(bytes.NewReader(archive), redditArchiveLimits{maxArchiveBytes: 1_024, maxSavedCSVBytes: 1_024, maxSavedEntries: 1})
	require.ErrorContains(t, err, "saved entry count exceeds")
}

func TestXArchiveSeparatesAuthoredAndLikeStreams(t *testing.T) {
	t.Log("[REQ:SIG-P0-008]")
	export := redditArchive(t, map[string]string{
		"data/tweets.js":          `window.YTD.tweets.part0 = [{"tweet":{"id_str":"100","full_text":"authored post"}}]`,
		"data/like.js":            `window.YTD.like.part0 = [{"like":{"tweetId":"200","fullText":"liked post","expandedUrl":"https://x.com/example/status/200"}}]`,
		"data/direct-messages.js": `window.YTD.direct_messages.part0 = [{"dm":"must not import"}]`,
	})
	authored, err := (XAuthoredArchiveAdapter{}).Parse(context.Background(), bytes.NewReader(export))
	require.NoError(t, err)
	require.Len(t, authored, 1)
	require.Equal(t, "https://x.com/i/web/status/100", authored[0].URL)
	require.Equal(t, "authored post", authored[0].Text)
	require.Equal(t, []string{"x", "authored"}, authored[0].Tags)

	likes, err := (XLikesArchiveAdapter{}).Parse(context.Background(), bytes.NewReader(export))
	require.NoError(t, err)
	require.Len(t, likes, 1)
	require.Equal(t, "liked post", likes[0].Text)
	require.Equal(t, []string{"x", "liked"}, likes[0].Tags)
}

func TestXArchiveRejectsMissingDeclaredStream(t *testing.T) {
	_, err := (XAuthoredArchiveAdapter{}).Parse(context.Background(), bytes.NewReader(redditArchive(t, map[string]string{"data/like.js": `window.YTD.like.part0 = []`})))
	require.ErrorContains(t, err, "data/tweets.js is absent")
}

func redditArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range files {
		file, err := writer.Create(name)
		require.NoError(t, err)
		_, err = file.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}
