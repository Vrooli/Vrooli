package research_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"web-search/internal/clock"
	localdb "web-search/internal/database"
	"web-search/internal/findings"
	"web-search/internal/research"
	testdb "web-search/internal/testutil/db"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func newFindingsService(t *testing.T) findings.Service {
	t.Helper()
	d := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(findings.Schema),
	))
	repo := findings.NewSQLiteRepository(d, clock.System{})
	return findings.NewServiceWithActor(repo, "agent")
}

// TestRunL2CitedOutput asserts the happy path: candidates -> fetch each ->
// single-pass cited synthesis, with the brief carrying the citations the
// synthesizer grounded in.
func TestRunL2CitedOutput(t *testing.T) {
	ctx := context.Background()
	searcher := &fakeSearcher{candidates: []research.Candidate{
		{URL: "https://a.example", Title: "A"},
		{URL: "https://b.example", Title: "B"},
	}}
	fetcher := &fakeFetcher{textByURL: map[string]string{
		"https://a.example": "alpha body text",
		"https://b.example": "beta body text",
	}}
	syn := &fakeSynthesizer{result: research.Synthesis{
		Text: "alpha and beta agree",
		Citations: []research.Citation{
			{ResultIndex: 0, URL: "https://a.example", Title: "A"},
			{ResultIndex: 1, URL: "https://b.example", Title: "B"},
		},
	}}
	svc := research.NewService(research.Deps{Searcher: searcher, Fetcher: fetcher, Synthesizer: syn})

	out, err := svc.RunL2(ctx, "do alpha and beta agree", 5, false)
	require.NoError(t, err)
	require.False(t, out.Abstained)
	require.Equal(t, research.LevelL2, out.Brief.Level)
	require.Equal(t, "alpha and beta agree", out.Brief.Summary)
	require.Len(t, out.Brief.Citations, 2)
	require.Empty(t, out.CapturedFindingIDs)

	// Synthesizer saw both fetched documents.
	require.Len(t, syn.gotDocs, 2)
	require.Equal(t, []string{"https://a.example", "https://b.example"}, fetcher.fetched)
}

// TestRunL2AbstainOnConflict asserts the always-cited / abstain contract: when
// the synthesizer abstains (sources conflict), the brief is an explicit
// abstention and no finding is captured even with --capture.
func TestRunL2AbstainOnConflict(t *testing.T) {
	ctx := context.Background()
	searcher := &fakeSearcher{candidates: []research.Candidate{{URL: "https://a.example", Title: "A"}}}
	fetcher := &fakeFetcher{textByURL: map[string]string{"https://a.example": "contested body"}}
	syn := &fakeSynthesizer{result: research.Abstain()}
	fsvc := newFindingsService(t)
	svc := research.NewService(research.Deps{Searcher: searcher, Fetcher: fetcher, Synthesizer: syn, Findings: fsvc})

	out, err := svc.RunL2(ctx, "contested topic", 3, true) // capture on
	require.NoError(t, err)
	require.True(t, out.Abstained)
	require.Empty(t, out.CapturedFindingIDs, "abstention must never capture a finding")

	all, err := fsvc.List(ctx, findings.ListFilter{})
	require.NoError(t, err)
	require.Empty(t, all)
}

// TestRunL2CaptureWritesL2SourcedFinding asserts --capture writes the cited
// synthesis as a FINDING_SOURCE_L2 finding carrying every citation.
func TestRunL2CaptureWritesL2SourcedFinding(t *testing.T) {
	ctx := context.Background()
	searcher := &fakeSearcher{candidates: []research.Candidate{{URL: "https://a.example", Title: "A"}}}
	fetcher := &fakeFetcher{textByURL: map[string]string{"https://a.example": "the answer is 42"}}
	syn := &fakeSynthesizer{result: research.Synthesis{
		Text:      "the answer is 42",
		Citations: []research.Citation{{ResultIndex: 0, URL: "https://a.example", Title: "A"}},
	}}
	fsvc := newFindingsService(t)
	svc := research.NewService(research.Deps{Searcher: searcher, Fetcher: fetcher, Synthesizer: syn, Findings: fsvc})

	out, err := svc.RunL2(ctx, "what is the answer", 1, true)
	require.NoError(t, err)
	require.Len(t, out.CapturedFindingIDs, 1)

	got, err := fsvc.Get(ctx, out.CapturedFindingIDs[0])
	require.NoError(t, err)
	require.Equal(t, "the answer is 42", got.Claim)
	require.Equal(t, findings.SourceL2, got.Source)
	require.Len(t, got.Citations, 1)
	require.Equal(t, "https://a.example", got.Citations[0].URL)
}

// TestRunL2NoCaptureByDefault asserts the L2 opt-in capture contract: with the
// capture flag absent (false), fetched page content and the synthesis are NOT
// persisted as findings even though a findings store is wired.
func TestRunL2NoCaptureByDefault(t *testing.T) {
	ctx := context.Background()
	searcher := &fakeSearcher{candidates: []research.Candidate{{URL: "https://a.example", Title: "A"}}}
	fetcher := &fakeFetcher{textByURL: map[string]string{"https://a.example": "full page body"}}
	syn := &fakeSynthesizer{result: research.Synthesis{
		Text:      "a grounded answer",
		Citations: []research.Citation{{ResultIndex: 0, URL: "https://a.example", Title: "A"}},
	}}
	fsvc := newFindingsService(t)
	svc := research.NewService(research.Deps{Searcher: searcher, Fetcher: fetcher, Synthesizer: syn, Findings: fsvc})

	out, err := svc.RunL2(ctx, "q", 3, false) // capture absent
	require.NoError(t, err)
	require.False(t, out.Abstained)
	require.Empty(t, out.CapturedFindingIDs)

	all, err := fsvc.List(ctx, findings.ListFilter{})
	require.NoError(t, err)
	require.Empty(t, all, "L2 must not persist findings unless capture is requested")
}

// TestRunL2TopFiveWithinLatencyBudget is the REQ-P1-001 performance gate over
// the package fakes: the top-5 fetch -> extract -> synthesize pipeline performs
// exactly the bounded work (5 fetches, one synthesis pass) and its orchestration
// adds no pathological overhead (sleeps, retry storms). The production 15s p95
// budget is dominated by network/LLM time, which fakes deliberately exclude, so
// the in-process bound asserted here is far stricter.
func TestRunL2TopFiveWithinLatencyBudget(t *testing.T) {
	ctx := context.Background()
	cands := make([]research.Candidate, 0, 5)
	texts := make(map[string]string, 5)
	for i := 0; i < 5; i++ {
		url := fmt.Sprintf("https://p%d.example", i)
		cands = append(cands, research.Candidate{URL: url, Title: fmt.Sprintf("P%d", i)})
		texts[url] = fmt.Sprintf("body of page %d", i)
	}
	searcher := &fakeSearcher{candidates: cands}
	fetcher := &fakeFetcher{textByURL: texts}
	syn := &fakeSynthesizer{result: research.Synthesis{
		Text:      "synthesized over five pages",
		Citations: []research.Citation{{ResultIndex: 0, URL: cands[0].URL, Title: cands[0].Title}},
	}}
	svc := research.NewService(research.Deps{Searcher: searcher, Fetcher: fetcher, Synthesizer: syn})

	start := time.Now()
	out, err := svc.RunL2(ctx, "q", 5, false)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.False(t, out.Abstained)
	require.Len(t, fetcher.fetched, 5, "top-5 request fetches exactly 5 pages")
	require.Len(t, syn.gotDocs, 5, "all five extracted documents reach the single synthesis pass")
	require.Less(t, elapsed, 2*time.Second, "L2 pipeline orchestration overhead must stay well inside the 15s p95 budget")
}

// TestRunL2ToleratesFetchFailures asserts a per-page fetch failure is skipped
// and synthesis runs over whatever was retrieved.
func TestRunL2ToleratesFetchFailures(t *testing.T) {
	ctx := context.Background()
	searcher := &fakeSearcher{candidates: []research.Candidate{
		{URL: "https://ok.example", Title: "OK"},
		{URL: "https://bad.example", Title: "BAD"},
	}}
	fetcher := &fakeFetcher{
		textByURL: map[string]string{"https://ok.example": "good body"},
		failErr:   errors.New("fetch substrate down"),
	}
	syn := &fakeSynthesizer{result: research.Synthesis{
		Text:      "grounded in the one good page",
		Citations: []research.Citation{{ResultIndex: 0, URL: "https://ok.example", Title: "OK"}},
	}}
	svc := research.NewService(research.Deps{Searcher: searcher, Fetcher: fetcher, Synthesizer: syn})

	out, err := svc.RunL2(ctx, "q", 5, false)
	require.NoError(t, err)
	require.False(t, out.Abstained)
	require.Len(t, syn.gotDocs, 1, "only the successfully fetched page reaches synthesis")
}

// TestRunL2AbstainsWhenSeamsMissing asserts L2 degrades gracefully (abstains)
// when the seams it needs are not wired (e.g. the fetch stack down at boot).
func TestRunL2AbstainsWhenSeamsMissing(t *testing.T) {
	svc := research.NewService(research.Deps{})
	out, err := svc.RunL2(context.Background(), "q", 5, false)
	require.NoError(t, err)
	require.True(t, out.Abstained)
}

// TestRunL2AbstainReasonsDistinguishCollapses asserts the four formerly
// indistinguishable abstain causes each surface their own machine-readable
// reason (OT-P1-001 observability).
func TestRunL2AbstainReasonsDistinguishCollapses(t *testing.T) {
	ctx := context.Background()

	t.Run("no candidates", func(t *testing.T) {
		svc := research.NewService(research.Deps{
			Searcher:    &fakeSearcher{},
			Fetcher:     &fakeFetcher{},
			Synthesizer: &fakeSynthesizer{},
		})
		out, err := svc.RunL2(ctx, "q", 5, false)
		require.NoError(t, err)
		require.True(t, out.Abstained)
		require.Equal(t, research.ReasonNoCandidates, out.AbstainReason)
	})

	t.Run("all fetches empty", func(t *testing.T) {
		searcher := &fakeSearcher{candidates: []research.Candidate{{URL: "https://a.example", Title: "A"}}}
		fetcher := &fakeFetcher{failErr: errors.New("fetch substrate down")}
		svc := research.NewService(research.Deps{
			Searcher:    searcher,
			Fetcher:     fetcher,
			Synthesizer: &fakeSynthesizer{},
		})
		out, err := svc.RunL2(ctx, "q", 5, false)
		require.NoError(t, err)
		require.True(t, out.Abstained)
		require.Equal(t, research.ReasonAllFetchesEmpty, out.AbstainReason)
	})

	t.Run("model abstained", func(t *testing.T) {
		searcher := &fakeSearcher{candidates: []research.Candidate{{URL: "https://a.example", Title: "A"}}}
		fetcher := &fakeFetcher{textByURL: map[string]string{"https://a.example": "body"}}
		syn := &fakeSynthesizer{result: research.AbstainWith(research.ReasonModelAbstained)}
		svc := research.NewService(research.Deps{Searcher: searcher, Fetcher: fetcher, Synthesizer: syn})
		out, err := svc.RunL2(ctx, "q", 5, false)
		require.NoError(t, err)
		require.True(t, out.Abstained)
		require.Equal(t, research.ReasonModelAbstained, out.AbstainReason)
		require.NotEmpty(t, out.Excerpts, "excerpts are observable even on a model abstention")
	})

	t.Run("parse-layer reasons", func(t *testing.T) {
		docs := []research.Document{{URL: "https://a.example", Title: "A", Text: "body"}}
		require.Equal(t, research.ReasonReplyUnparseable, research.ParseSynthesisReply("no json here", docs).AbstainReason)
		require.Equal(t, research.ReasonModelAbstained, research.ParseSynthesisReply(`{"abstained":true,"text":"","citations":[]}`, docs).AbstainReason)
		require.Equal(t, research.ReasonCitationsInvalid, research.ParseSynthesisReply(`{"abstained":false,"text":"claim","citations":[9]}`, docs).AbstainReason)
	})
}
