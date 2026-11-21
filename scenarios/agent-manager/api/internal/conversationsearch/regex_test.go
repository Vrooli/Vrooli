package conversationsearch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestTextAndRegexSearchContract(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	for index := 1; index <= 3; index++ {
		insertSearchDocument(t, repository, string(rune('a'+index-1)), "phase capacity correction", ContentClassProse, fixtureTimeValue(index))
	}

	first, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `phase\s+capacity`, PageSize: 1})
	require.NoError(t, err)
	require.Len(t, first.Hits, 1)
	require.Equal(t, SearchLegRegex, first.Hits[0].Leg)
	require.NotEmpty(t, first.NextCursor)
	require.NotEmpty(t, first.Hits[0].Highlights)

	second, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `phase\s+capacity`, PageSize: 1, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Hits, 1)
	require.NotEqual(t, first.Hits[0].Document.DocumentID, second.Hits[0].Document.DocumentID)

	_, err = service.SearchText(context.Background(), TextSearchRequest{Query: `phase\s+capacity`, PageSize: 1, Cursor: first.NextCursor})
	require.ErrorIs(t, err, ErrInvalidRequest)
	_, err = service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `(`})
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestRegexGoldenDateBoundedPattern(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	inside := searchDocument("inside", "test-genie repaired the system-monitor path", ContentClassProse, time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC))
	outside := searchDocument("outside", "test-genie repaired the system-monitor path", ContentClassProse, time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC))
	require.NoError(t, repository.UpsertDocument(context.Background(), inside))
	require.NoError(t, repository.UpsertDocument(context.Background(), outside))
	after := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	response, err := service.SearchRegex(context.Background(), RegexSearchRequest{
		Pattern: `(?i)test.genie.*system.monitor`,
		Filters: SearchFilters{OccurredAfter: &after, OccurredBefore: &before},
	})
	require.NoError(t, err)
	require.Len(t, response.Hits, 1)
	require.Equal(t, "inside", response.Hits[0].Document.DocumentID)
	require.Equal(t, RegexLimitNone, response.PartialReason)
}

func TestRegexAdversarialUnicodeMultilineZeroWidthAndInvalidUTF8(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	insertSearchDocument(t, repository, "unicode", "πρώτο 🚀\nsecond line\nlast λ", ContentClassProse, fixtureTimeValue(1))
	insertSearchDocument(t, repository, "invalid-utf8", "needle "+string([]byte{0xff})+" suffix", ContentClassProse, fixtureTimeValue(2))
	insertSearchDocument(t, repository, "catastrophic-looking", strings.Repeat("a", 64<<10)+"!", ContentClassProse, fixtureTimeValue(3))

	multiline, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `(?s)πρώτο.*last\s+λ`})
	require.NoError(t, err)
	require.Len(t, multiline.Hits, 1)
	require.True(t, utf8.ValidString(multiline.Hits[0].Snippet))

	zeroWidth, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `^`, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, zeroWidth.Hits, 3)
	require.Equal(t, zeroWidth.Hits[0].Highlights[0].StartRune, zeroWidth.Hits[0].Highlights[0].EndRune)

	invalidUTF8, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `needle.*suffix`})
	require.NoError(t, err)
	require.Len(t, invalidUTF8.Hits, 1)
	require.True(t, utf8.ValidString(invalidUTF8.Hits[0].Snippet))

	catastrophic, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `^(a+)+$`})
	require.NoError(t, err)
	require.Empty(t, catastrophic.Hits)
}

func TestRegexReportsCandidateByteAndDeadlineBounds(t *testing.T) {
	t.Parallel()
	t.Run("candidate cap", func(t *testing.T) {
		service, repository := searchFixtureService(t)
		service.regexPolicy.NoLiteralMaxCandidates = 2
		for index := 0; index < 3; index++ {
			insertSearchDocument(t, repository, string(rune('a'+index)), "ordinary text", ContentClassProse, fixtureTimeValue(index+1))
		}
		response, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `.*`})
		require.NoError(t, err)
		require.Equal(t, RegexLimitCandidates, response.PartialReason)
		require.Equal(t, 2, response.ScannedCandidates)
	})

	t.Run("byte cap", func(t *testing.T) {
		service, repository := searchFixtureService(t)
		service.regexPolicy.MaxCandidateBytes = 32
		insertSearchDocument(t, repository, "large", "needle "+strings.Repeat("x", 128), ContentClassProse, fixtureTimeValue(1))
		response, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `needle`})
		require.NoError(t, err)
		require.Equal(t, RegexLimitBytes, response.PartialReason)
		require.Empty(t, response.Hits)
		require.LessOrEqual(t, response.ScannedBytes, 32)
	})

	t.Run("server deadline", func(t *testing.T) {
		service, repository := searchFixtureService(t)
		service.regexPolicy.MaxDuration = -1
		insertSearchDocument(t, repository, "deadline", "deadline candidate", ContentClassProse, fixtureTimeValue(1))
		response, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `deadline`})
		require.NoError(t, err)
		require.Equal(t, RegexLimitDeadline, response.PartialReason)
		require.Empty(t, response.Hits)
	})

	t.Run("result cap", func(t *testing.T) {
		service, repository := searchFixtureService(t)
		service.regexPolicy.MaxResults = 1
		insertSearchDocument(t, repository, "result-a", "result cap", ContentClassProse, fixtureTimeValue(1))
		insertSearchDocument(t, repository, "result-b", "result cap", ContentClassProse, fixtureTimeValue(2))
		response, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `result`})
		require.NoError(t, err)
		require.Equal(t, RegexLimitCandidates, response.PartialReason)
		require.Len(t, response.Hits, 1)
	})
}

func TestRegexHonorsContextCancellationAndDefaultContentPolicy(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	insertSearchDocument(t, repository, "prose", "search needle", ContentClassProse, fixtureTimeValue(1))
	insertSearchDocument(t, repository, "tool", "search needle", ContentClassToolResult, fixtureTimeValue(2))

	response, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `needle`})
	require.NoError(t, err)
	require.Len(t, response.Hits, 1)
	require.Equal(t, "prose", response.Hits[0].Document.DocumentID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.SearchRegex(ctx, RegexSearchRequest{Pattern: `needle`})
	require.True(t, errors.Is(err, context.Canceled))

	deadlineCtx, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer deadlineCancel()
	_, err = service.SearchRegex(deadlineCtx, RegexSearchRequest{Pattern: `needle`})
	require.True(t, errors.Is(err, context.DeadlineExceeded))
}

func TestRegexNewestCursorIgnoresConcurrentInsertAheadOfContinuation(t *testing.T) {
	t.Parallel()
	service, repository := searchFixtureService(t)
	insertSearchDocument(t, repository, "old", "cursor needle", ContentClassProse, fixtureTimeValue(1))
	insertSearchDocument(t, repository, "current", "cursor needle", ContentClassProse, fixtureTimeValue(2))
	first, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `needle`, Sort: SearchSortNewest, PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, "current", first.Hits[0].Document.DocumentID)

	insertSearchDocument(t, repository, "concurrent", "cursor needle", ContentClassProse, fixtureTimeValue(3))
	second, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `needle`, Sort: SearchSortNewest, PageSize: 1, Cursor: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Hits, 1)
	require.Equal(t, "old", second.Hits[0].Document.DocumentID)
}

func TestMandatoryRegexLiteralIsConservative(t *testing.T) {
	t.Parallel()
	require.Equal(t, "MONITOR", mandatoryRegexLiteral(`(?i)test.genie.*system.monitor`))
	require.Equal(t, "", mandatoryRegexLiteral(`foo|bar`))
	require.Equal(t, "", mandatoryRegexLiteral(`.*`))
	require.Equal(t, "", mandatoryRegexLiteral(`λiteral.*suffix`), "non-ASCII literals avoid tokenizer-dependent false negatives")
}

func BenchmarkRegexSearchWorstPermittedScan(b *testing.B) {
	service, repository := searchFixtureService(b)
	payload := strings.Repeat("a", 7<<10) + "!"
	for index := 0; index < service.regexPolicy.NoLiteralMaxCandidates; index++ {
		insertSearchDocument(b, repository, fmt.Sprintf("worst-scan-%04d", index), payload, ContentClassProse, fixtureTimeValue(index%59))
	}
	b.ResetTimer()
	for b.Loop() {
		response, err := service.SearchRegex(context.Background(), RegexSearchRequest{Pattern: `^(a+)+$`})
		require.NoError(b, err)
		require.Empty(b, response.Hits)
		require.LessOrEqual(b, response.ScannedBytes, service.regexPolicy.NoLiteralMaxBytes)
	}
}
