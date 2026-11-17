package files

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
	require.NoError(t, os.WriteFile(abs, []byte(body), 0o644))
}

func newService(t *testing.T, root string, embedder Embedder) *Service {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "files.db"))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	svc, err := NewService(context.Background(), db, map[string]string{DefaultCorpus: root}, embedder, nil)
	require.NoError(t, err)
	return svc
}

func search(t *testing.T, svc *Service, q Query) Result {
	t.Helper()
	res, err := svc.Search(context.Background(), q)
	require.NoError(t, err)
	return res
}

func degradation(t *testing.T, res Result, reason string) string {
	t.Helper()
	for _, d := range res.Degradations {
		if d.Reason == reason {
			return d.Detail
		}
	}
	t.Fatalf("no %s degradation in %+v", reason, res.Degradations)
	return ""
}

// Showcase (a): edit a topic file, query, get the new text back with its path
// and line span. Delete it and it is gone. No restart, no import step.
func TestSearchIsFreshAndCitedAfterEditsAndDeletes(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/ux/knowledge/MAP.md", "# Map\n\n## Scenarios\n\nThe planner owns week view.\n")
	svc := newService(t, root, nil)

	res := search(t, svc, Query{Text: "planner week"})
	require.Len(t, res.Hits, 1)
	require.Equal(t, "topics/ux/knowledge/MAP.md#L3-L5", res.Hits[0].Citation)
	require.Equal(t, "ux", res.Hits[0].Topic)
	require.Equal(t, "knowledge", res.Hits[0].Feed)
	require.Equal(t, "Map › Scenarios", res.Hits[0].Heading)
	require.Equal(t, "lexical", res.Mode)

	time.Sleep(10 * time.Millisecond) // distinct mtime on coarse filesystems
	write(t, root, "topics/ux/knowledge/MAP.md", "# Map\n\n## Scenarios\n\nThe planner owns week view.\n\n## Capture\n\nZebra capture pipeline replaces screenshots.\n")
	res = search(t, svc, Query{Text: "zebra capture"})
	require.Len(t, res.Hits, 1, "an edit is visible on the next read")
	require.Equal(t, "topics/ux/knowledge/MAP.md#L7-L9", res.Hits[0].Citation)
	require.Contains(t, res.Hits[0].Snippet, "Zebra capture pipeline")
	require.Equal(t, 1, res.Sync.Changed)

	require.NoError(t, os.Remove(filepath.Join(root, "topics/ux/knowledge/MAP.md")))
	res = search(t, svc, Query{Text: "zebra capture"})
	require.Empty(t, res.Hits, "a deleted file is never cited")
	require.Equal(t, 1, res.Sync.Removed)
}

func TestWalkPolicySummarisesHeavyFoldersAndHonoursFeedsJSON(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/a/README.md", "# A\n")
	write(t, root, "topics/a/runs/r1/README.md", "# run\n")
	write(t, root, "topics/a/runs/r1/notes/x.md", "kept: depth 1\n")
	write(t, root, "topics/a/runs/r1/notes/deep/README.md", "summary kept\n")
	write(t, root, "topics/a/runs/r1/notes/deep/raw.md", "dropped: depth 2\n")
	write(t, root, "topics/a/runs/r1/proto/package.json", "{}\n")
	write(t, root, "topics/a/runs/r1/proto/README.md", "code tree summary\n")
	write(t, root, "topics/a/runs/r1/proto/notes.md", "dropped: code tree\n")
	write(t, root, "topics/a/runs/r1/persist/db.md", "dropped: pruned dir\n")
	write(t, root, "topics/a/corpus/copy/doc.md", "dropped by topic feeds.json\n")
	write(t, root, "topics/a/feedback/2026-11.md", "operator said\n")
	write(t, root, "topics/a/feeds.json", `{"exclude":["corpus/**"],"include":["runs/*/notes/deep/raw.md"],"feeds":{"operator":["feedback/**"]}}`)
	for i := 0; i < 5; i++ {
		write(t, root, "topics/a/runs/r1/data/case"+string(rune('a'+i))+".md", "generated\n")
	}
	write(t, root, "feeds.json", `{"dataset_files": 4}`)

	walked := Walk(root, "")
	require.Empty(t, walked.Problems)
	got := map[string]string{}
	for _, e := range walked.Entries {
		got[e.Path] = e.Feed
	}
	require.Contains(t, got, "topics/a/runs/r1/notes/x.md")
	require.Contains(t, got, "topics/a/runs/r1/notes/deep/README.md")
	require.Contains(t, got, "topics/a/runs/r1/notes/deep/raw.md", "include re-admits a summarised path")
	require.Contains(t, got, "topics/a/runs/r1/proto/README.md")
	require.NotContains(t, got, "topics/a/runs/r1/proto/notes.md")
	require.NotContains(t, got, "topics/a/runs/r1/persist/db.md")
	require.NotContains(t, got, "topics/a/corpus/copy/doc.md")
	require.NotContains(t, got, "topics/a/runs/r1/data/casea.md", "a folder over dataset_files is summarised")
	require.Equal(t, "operator", got["topics/a/feedback/2026-11.md"])
	require.Equal(t, "(root)", got["topics/a/README.md"])
	require.Equal(t, "runs", got["topics/a/runs/r1/README.md"])
}

func TestLargeLogsKeepTheirNewestTailWithRealLineNumbers(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("when,event\n")
	for i := 1; i <= 3000; i++ {
		b.WriteString("2026-11-15,routine filler event\n")
	}
	b.WriteString("2026-11-15,quasar outage detected\n")
	write(t, root, "topics/ops/log.csv", b.String())
	write(t, root, "feeds.json", `{"max_file_kb": 8}`)
	svc := newService(t, root, nil)

	res := search(t, svc, Query{Text: "quasar outage", Topic: "ops"})
	require.Len(t, res.Hits, 1)
	require.Equal(t, 3002, res.Hits[0].LineEnd, "the tail keeps true line numbers")
}

type brokenEmbedder struct{}

func (brokenEmbedder) EmbedQueryFast(context.Context, string) ([]float64, error) {
	return nil, errors.New("ai-gateway unavailable")
}

func (brokenEmbedder) EmbedDocumentsWhenIdle(context.Context, []string) ([][]float64, error) {
	return nil, errors.New("ai-gateway unavailable")
}

// conceptEmbedder maps words to a few meaning axes, so texts that share no
// words can still be near each other — the property the semantic leg needs.
type conceptEmbedder struct{ calls atomic.Int64 }

var concepts = [][]string{
	{"anti-fragile", "antifragile", "unbreakable", "indestructible", "resilient", "robust", "disorder", "shocks"},
	{"invoice", "invoices", "quarterly", "totals", "billing", "payment"},
	{"planner", "week", "calendar", "schedule"},
}

func (e *conceptEmbedder) vec(text string) []float64 {
	v := make([]float64, len(concepts)+1)
	v[len(concepts)] = 0.1
	low := strings.ToLower(text)
	for i, words := range concepts {
		for _, w := range words {
			v[i] += float64(strings.Count(low, w))
		}
	}
	return v
}

func (e *conceptEmbedder) EmbedQueryFast(_ context.Context, text string) ([]float64, error) {
	return e.vec(text), nil
}

func (e *conceptEmbedder) EmbedDocumentsWhenIdle(_ context.Context, texts []string) ([][]float64, error) {
	e.calls.Add(1)
	out := make([][]float64, len(texts))
	for i, t := range texts {
		out[i] = e.vec(t)
	}
	return out, nil
}

func TestSemanticLegIsOptional(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/t/a.md", "# A\n\nrouting policy for the gateway fallback\n")
	broken := newService(t, root, brokenEmbedder{})
	broken.embedBackground(context.Background(), DefaultCorpus)
	res := search(t, broken, Query{Text: "routing policy"})
	require.Len(t, res.Hits, 1, "lexical search needs nothing but the files")
	require.Equal(t, "lexical", res.Mode)
	require.Equal(t, "semantic", res.Degradations[0].Leg)
}

// The operator's ask: a query that shares no words with its answer still
// finds it, cited.
func TestSynonymQueryFindsTextThatSharesNoWords(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/eng/knowledge/stance.md", "# Stance\n\nBuild anti-fragile systems that gain from disorder and shocks.\n")
	write(t, root, "topics/eng/work/billing.md", "# Billing\n\nQuarterly invoice totals and payment schedule.\n")
	write(t, root, "topics/eng/work/plan.md", "# Plan\n\nThe planner owns the week calendar.\n")
	svc := newService(t, root, &conceptEmbedder{})

	res := search(t, svc, Query{Text: "unbreakable indestructible"})
	require.Empty(t, res.Hits, "nothing is embedded yet and no word matches")
	require.Equal(t, "SEMANTIC_WARMING", res.Degradations[0].Reason)

	svc.embedBackground(context.Background(), DefaultCorpus)
	res = search(t, svc, Query{Text: "unbreakable indestructible"})
	require.Equal(t, "hybrid", res.Mode)
	require.NotEmpty(t, res.Hits)
	require.Equal(t, "topics/eng/knowledge/stance.md#L1-L3", res.Hits[0].Citation)
	require.Contains(t, res.Hits[0].Snippet, "anti-fragile")

	res = search(t, svc, Query{Text: "unbreakable indestructible", Topic: "eng", Feed: "work"})
	for _, h := range res.Hits {
		require.Equal(t, "work", h.Feed, "filters hold for semantic candidates")
	}
}

// A paraphrase shares filler words with many chunks; that partial overlap
// must not outrank the chunk that means the same thing. A query whose every
// word appears in a chunk is still decided by those words.
func TestSharedFillerWordsNeverOutrankMeaningButExactTermsDo(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/eng/knowledge/stance.md", "# Stance\n\nBuild anti-fragile systems that gain from disorder.\n")
	write(t, root, "topics/eng/work/billing.md", "# Billing\n\nThe systems that send the quarterly invoice must be unbreakable and resilient for the auditors.\n")
	write(t, root, "topics/eng/work/plan.md", "# Plan\n\nThe planner owns the week calendar.\n")
	svc := newService(t, root, &conceptEmbedder{})
	search(t, svc, Query{Text: "systems"}) // the first read indexes the files
	svc.embedBackground(context.Background(), DefaultCorpus)

	res := search(t, svc, Query{Text: "make the platform indestructible against shocks"})
	require.Equal(t, "topics/eng/knowledge/stance.md", res.Hits[0].Path, "meaning beats the shared words 'the' and 'make'")

	res = search(t, svc, Query{Text: "quarterly invoice auditors"})
	require.Equal(t, "topics/eng/work/billing.md", res.Hits[0].Path, "every query word present is strong evidence")
}

func TestPartialCoverageStaysHybridAndQueuesWhatTheQueryNeeded(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/eng/knowledge/stance.md", "# Stance\n\nresilient systems gain from disorder\n")
	svc := newService(t, root, &conceptEmbedder{})
	search(t, svc, Query{Text: "resilient"})
	svc.embedBackground(context.Background(), DefaultCorpus)

	write(t, root, "topics/eng/work/later.md", "# Later\n\nresilient billing for the week\n")
	res := search(t, svc, Query{Text: "resilient"})
	require.Equal(t, "hybrid", res.Mode, "partial coverage never switches the semantic leg off")
	require.Equal(t, "SEMANTIC_PARTIAL", res.Degradations[0].Reason)
	require.Len(t, res.Hits, 2)
	require.Equal(t, 1, len(svc.priority), "the unembedded lexical hit is queued first")
	svc.embedBackground(context.Background(), DefaultCorpus)
	res = search(t, svc, Query{Text: "resilient"})
	require.Empty(t, res.Degradations)
}

func TestVectorIndexRanksByCosine(t *testing.T) {
	var x vectorIndex
	x.loaded, x.pos = true, map[string]int{}
	x.add("a", []float64{1, 0, 0})
	x.add("b", []float64{0.7, 0.7, 0})
	x.add("c", []float64{0, 0, 1})
	got := x.nearest([]float64{1, 0.1, 0}, nil, 2)
	require.Len(t, got, 2)
	require.Equal(t, "a", got[0].hash)
	require.Equal(t, "b", got[1].hash)
	require.InDelta(t, 0.995, got[0].cos, 0.02)
	require.Equal(t, "c", x.nearest([]float64{0, 0, 1}, map[string]bool{"c": true}, 5)[0].hash)
}

func TestTopicAndFeedFiltersAndUnknownCorpus(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/one/knowledge/x.md", "shared phrase alpha\n")
	write(t, root, "topics/two/work/y.md", "shared phrase beta\n")
	svc := newService(t, root, nil)

	require.Len(t, search(t, svc, Query{Text: "shared phrase"}).Hits, 2)
	res := search(t, svc, Query{Text: "shared phrase", Topic: "two"})
	require.Len(t, res.Hits, 1)
	require.Equal(t, "two", res.Hits[0].Topic)
	require.Len(t, search(t, svc, Query{Text: "shared phrase", Feed: "knowledge"}).Hits, 1)

	_, err := svc.Search(context.Background(), Query{Text: "x y", Corpus: "nope"})
	require.Error(t, err)
	_, err = svc.Search(context.Background(), Query{Text: "x y", Topic: "../etc"})
	require.Error(t, err)

	feeds, _, err := svc.ListFeeds(context.Background(), "", "one", 2)
	require.NoError(t, err)
	require.Len(t, feeds, 1)
	require.Equal(t, "knowledge", feeds[0].Feed)
	require.Equal(t, "topics/one/knowledge/x.md", feeds[0].Recent[0].Path)
}

func TestEntryFilesOutrankTheEvidenceAboutThem(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/ux/skills/present-decisions/SKILL.md", "# Usage\n\nTurn a redesign into decisions the operator answers on a phone; present each one with today beside it.\n")
	write(t, root, "topics/ux/skills/present-decisions/VALIDATION.md", "# Validation\n\nChecked the decisions page on a phone. Present and readable.\n")
	write(t, root, "topics/ux/runs/r1/decisions.md", "# Run\n\npresent decisions present decisions present decisions phone\n")
	svc := newService(t, root, nil)

	res := search(t, svc, Query{Text: "present decisions"})
	require.Equal(t, "topics/ux/skills/present-decisions/SKILL.md", res.Hits[0].Path)
	require.Equal(t, 1.0, res.Hits[0].Score)
}

// Raw material (run output, support copies, preimages, history) ranks below
// the curated page that says the same thing, unless the caller asks for it.
func TestRawMaterialRanksBelowCuratedPagesUnlessAsked(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/ux/knowledge/latency.md", "# Latency\n\nThe latency budget for the phone page is two seconds.\n")
	for _, raw := range []string{"topics/ux/runs/r1/notes.md", "topics/ux/support/copy/notes.md", "topics/ux/history/old.md"} {
		write(t, root, raw, "# Notes\n\nlatency budget latency budget latency budget phone page\n")
	}
	svc := newService(t, root, nil)

	res := search(t, svc, Query{Text: "latency budget"})
	require.Equal(t, "topics/ux/knowledge/latency.md", res.Hits[0].Path, "the finding before the evidence and copies behind it")
	res = search(t, svc, Query{Text: "latency budget", IncludeRaw: true})
	require.NotEqual(t, "topics/ux/knowledge/latency.md", res.Hits[0].Path, "asked for, raw material ranks on its words alone")
	require.True(t, isRaw("topics/a/b/c.pyc"))
	require.True(t, isRaw("reviews/r/replay/homes/old/topics/a/MISSION.md"))
	require.False(t, isRaw("topics/a/knowledge/runs.md"), "a file named like a raw folder is not raw")
}

// The first results name as many different files as they can: a file's
// second and third chunks come after every other file's best chunk.
func TestTopResultsNameDifferentFilesFirst(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/a/knowledge/big.md", "# One\n\nquota quota quota\n\n# Two\n\nquota quota quota\n\n# Three\n\nquota quota quota\n")
	write(t, root, "topics/a/knowledge/small.md", "# Other\n\nthe quota is mentioned once among many other words in a much longer passage of text\n")
	svc := newService(t, root, nil)

	res := search(t, svc, Query{Text: "quota", Limit: 2})
	require.Len(t, res.Hits, 2)
	require.NotEqual(t, res.Hits[0].Path, res.Hits[1].Path)
	res = search(t, svc, Query{Text: "quota", Limit: 4})
	require.Equal(t, "topics/a/knowledge/big.md", res.Hits[2].Path, "further chunks follow, in rank order")
}

func TestCoverageIsCountedAndSummarisedSubtreesKeepTheirSummaries(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/a/runs/r1/notes/deep/raw.md", "dropped: depth 2\n")
	write(t, root, "topics/a/runs/r1/notes/deep/deeper/README.md", "summary three levels down\n")
	write(t, root, "topics/a/runs/r1/big.md", strings.Repeat("long line of evidence\n", 100))
	write(t, root, "topics/a/runs/r1/huge.json", `{"x":"`+strings.Repeat("y", 2048)+`"}`)
	write(t, root, "topics/a/cohort/OPERATOR.md", "operator conversation\n")
	write(t, root, "topics/a/excluded/x.md", "excluded\n")
	write(t, root, "topics/a/feeds.json", `{"exclude":["excluded/*.md"]}`)
	write(t, root, "feeds.json", `{"max_file_kb": 1, "max_json_kb": 1}`)

	walked := Walk(root, "")
	paths := map[string]bool{}
	for _, e := range walked.Entries {
		paths[e.Path] = true
	}
	require.True(t, paths["topics/a/runs/r1/notes/deep/deeper/README.md"], "a summarised folder's subfolders still contribute their summaries")
	require.False(t, paths["topics/a/runs/r1/notes/deep/raw.md"])
	require.Equal(t, WalkStats{Private: 1, Excluded: 1, Summarised: 1, OversizeJSON: 1, Truncated: 1}, walked.Stats)

	svc := newService(t, root, nil)
	res := search(t, svc, Query{Text: "long line evidence"})
	require.True(t, res.Hits[0].Truncated, "a hit from a cut-short file says so")
	require.Equal(t, 1, res.Sync.Stats.Private)
}

func TestBrokenPolicyIsReportedAndNeverWidensTheIndex(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/a/README.md", "# A\n\nshared marker word\n")
	write(t, root, "topics/a/corpus/copy.md", "shared marker word in an excluded corpus\n")
	write(t, root, "topics/a/feeds.json", `{"exclude": ["corpus/**"],`) // malformed
	svc := newService(t, root, nil)

	res := search(t, svc, Query{Text: "shared marker"})
	require.Len(t, res.Hits, 1, "a malformed topic policy indexes only the topic's summaries")
	require.Equal(t, "topics/a/README.md", res.Hits[0].Path)
	require.Contains(t, degradation(t, res, "CORPUS_PROBLEMS"), "feeds.json")

	write(t, root, "feeds.json", `{nope`)
	res = search(t, svc, Query{Text: "shared marker"})
	require.NotEmpty(t, res.Hits, "a malformed root policy falls back to defaults instead of failing every search")
	require.Contains(t, degradation(t, res, "CORPUS_PROBLEMS"), "using defaults")
}

func TestPrivateFilesAreNeverServedUnlessATopicOptsIn(t *testing.T) {
	root := t.TempDir()
	write(t, root, "cohorts/c1/STANCE-MESSAGE.txt", "zircon stance words\n")
	write(t, root, "topics/jobs/runs/r/resume-facts.md", "zircon resume facts\n")
	write(t, root, "topics/open/OPERATOR.md", "zircon operator notes\n")
	write(t, root, "topics/open/feeds.json", `{"private": []}`)
	svc := newService(t, root, nil)

	res := search(t, svc, Query{Text: "zircon"})
	require.Len(t, res.Hits, 1)
	require.Equal(t, "topics/open/OPERATOR.md", res.Hits[0].Path, "only the topic that set private: [] serves its operator file")
	require.Equal(t, 2, res.Sync.Stats.Private)
}

// An edit that keeps size and restores mtime (cp -p, rsync -t) still moves
// the inode change time, so the next read serves the new text.
func TestSameSizeEditWithRestoredMtimeIsStillSeen(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "topics/a/notes.md")
	write(t, root, "topics/a/notes.md", "# N\n\nalpha bravo\n")
	info, err := os.Stat(abs)
	require.NoError(t, err)
	svc := newService(t, root, nil)
	require.Len(t, search(t, svc, Query{Text: "alpha"}).Hits, 1)

	time.Sleep(10 * time.Millisecond)
	write(t, root, "topics/a/notes.md", "# N\n\ngamma delta\n")
	require.NoError(t, os.Chtimes(abs, info.ModTime(), info.ModTime()))
	require.Empty(t, search(t, svc, Query{Text: "alpha"}).Hits)
	require.Len(t, search(t, svc, Query{Text: "gamma"}).Hits, 1)
}

// poisonEmbedder refuses any text holding "POISON", failing the whole batch
// it arrives in, like a chunk longer than the model's context.
type poisonEmbedder struct {
	conceptEmbedder
	refused atomic.Int64
}

func (e *poisonEmbedder) EmbedDocumentsWhenIdle(ctx context.Context, texts []string) ([][]float64, error) {
	for _, t := range texts {
		if strings.Contains(t, "POISON") {
			e.refused.Add(1)
			return nil, errors.New("embed: HTTP 500: the input length exceeds the context length")
		}
	}
	return e.conceptEmbedder.EmbedDocumentsWhenIdle(ctx, texts)
}

// One chunk the model refuses must not stall the queue behind it, and a
// batch made only of refused chunks is not mistaken for an outage.
func TestChunksTheModelRefusesNeverStallTheQueue(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/eng/knowledge/stance.md", "# Stance\n\nresilient systems gain from disorder\n")
	write(t, root, "topics/eng/MISSION.md", "# Mission\n\n## Data\n\nPOISON row one\n")
	var dense strings.Builder
	for i := range 40 {
		dense.WriteString("## Part " + string(rune('A'+i%26)) + "\n\nPOISON dense table\n\n")
	}
	write(t, root, "topics/eng/README.md", dense.String())
	write(t, root, "topics/eng/work/plan.md", "# Plan\n\nThe planner owns the week calendar.\n")
	emb := &poisonEmbedder{}
	svc := newService(t, root, emb)
	search(t, svc, Query{Text: "planner"})

	svc.embedBackground(context.Background(), DefaultCorpus)
	res := search(t, svc, Query{Text: "planner"})
	for _, d := range res.Degradations {
		require.NotEqual(t, "SEMANTIC_PARTIAL", d.Reason, "every chunk the model accepts is embedded in one pass; refused ones are not counted as missing")
	}
	feeds, _, err := svc.ListFeeds(context.Background(), DefaultCorpus, "eng", 0)
	require.NoError(t, err)
	for _, f := range feeds {
		require.Equal(t, f.Chunks, f.Embedded, f.Feed)
	}

	before := emb.refused.Load()
	svc.embedBackground(context.Background(), DefaultCorpus)
	require.Equal(t, before, emb.refused.Load(), "a refused chunk is never sent again")
}

func TestAnOutageMarksNothing(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/eng/knowledge/stance.md", "# Stance\n\nresilient systems\n")
	svc := newService(t, root, brokenEmbedder{})
	search(t, svc, Query{Text: "resilient"})
	svc.embedBackground(context.Background(), DefaultCorpus)
	var rows int
	require.NoError(t, svc.store.db.QueryRow(`SELECT COUNT(*) FROM vectors`).Scan(&rows))
	require.Zero(t, rows, "a down embedder is retried later, never recorded as refusing the text")
}

func TestFilesInOtherEncodingsAreIndexedAsValidText(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/eng/work/export.csv", "name,city\nBENNETT CONSTRUCTION,BIG PINEY \xe9\xff\n")
	svc := newService(t, root, nil)
	res := search(t, svc, Query{Text: "bennett piney"})
	require.Len(t, res.Hits, 1)
	require.True(t, utf8.ValidString(res.Hits[0].Snippet), "a Latin-1 byte must not reach protobuf as invalid UTF-8")
}

// Data files are searched by their words, never by meaning: they are not
// embedded, not counted as missing coverage, and still found lexically.
func TestDataFilesAreLexicalOnly(t *testing.T) {
	root := t.TempDir()
	write(t, root, "topics/eng/knowledge/stance.md", "# Stance\n\nresilient systems gain from disorder\n")
	write(t, root, "topics/eng/runs/r1/result.json", `{"settlement": "EIP-3009", "resilient": true}`+"\n")
	emb := &conceptEmbedder{}
	svc := newService(t, root, emb)
	search(t, svc, Query{Text: "resilient"})
	svc.embedBackground(context.Background(), DefaultCorpus)

	var jsonVectors int
	require.NoError(t, svc.store.db.QueryRow(`SELECT COUNT(*) FROM vectors v JOIN chunks c ON c.hash = v.hash WHERE c.path LIKE '%.json'`).Scan(&jsonVectors))
	require.Zero(t, jsonVectors, "data chunks are never embedded")

	res := search(t, svc, Query{Text: "EIP-3009 settlement"})
	require.Equal(t, "topics/eng/runs/r1/result.json", res.Hits[0].Path, "an identifier in a data file is still found")
	for _, d := range res.Degradations {
		require.NotEqual(t, "SEMANTIC_PARTIAL", d.Reason, "unembedded data files are not missing coverage")
	}
	require.Empty(t, svc.priority, "a data hit is not queued for embedding")
}
