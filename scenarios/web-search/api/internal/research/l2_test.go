package research_test

import (
	"context"
	"errors"
	"testing"

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
		failErr:   errors.New("browserless down"),
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
// when the seams it needs are not wired (e.g. browserless down at boot).
func TestRunL2AbstainsWhenSeamsMissing(t *testing.T) {
	svc := research.NewService(research.Deps{})
	out, err := svc.RunL2(context.Background(), "q", 5, false)
	require.NoError(t, err)
	require.True(t, out.Abstained)
}
