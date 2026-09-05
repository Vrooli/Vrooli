// Findings capture-policy and store-ownership tests (OT-P1-004 / OT-P0-005).
//
// These live in package main because they cut across internal packages: the
// capture policy is a property of the WHOLE pipeline (research orchestration
// wired to the real findings store), and the L0/L1 statelessness + ownership
// invariants are properties of the import graph itself. Unlike
// main_e2e_test.go they need no build tag — everything here is hermetic.
package main

import (
	"context"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	localdb "web-search/internal/database"
	"web-search/internal/findings"
	"web-search/internal/research"
	"web-search/internal/research/agentmanager"

	testdb "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/schedule"
)

// --- static import analysis -------------------------------------------------

// packageImports parses every non-test .go file in dir (relative to the api
// module root, the test working directory) and returns the union of imported
// package paths.
func packageImports(t *testing.T, dir string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			require.NoError(t, err)
			out[path] = true
		}
	}
	return out
}

// TestL0AndL1PathsHaveNoFindingWritePath statically proves the L0 (live web
// search) and L1 (snippet synthesis) code paths cannot persist findings: the
// packages implementing them — internal/livesearch (Search service = L0,
// synthesis.go = L1) and their HTTP surface handlers/livesearch — do not
// import the findings store at all, so no code path can write a finding.
// This is the OT-P1-004 "L0/L1 never persist" half of the capture policy and
// the reason those tiers stay stateless.
func TestL0AndL1PathsHaveNoFindingWritePath(t *testing.T) {
	for _, dir := range []string{
		filepath.Join("internal", "livesearch"),
		filepath.Join("handlers", "livesearch"),
	} {
		for path := range packageImports(t, dir) {
			require.NotContains(t, path, "internal/findings",
				"%s must not depend on the findings store (L0/L1 are stateless)", dir)
		}
	}
}

// TestFindingsStoreDoesNotTouchKnowledgeObservatory proves the findings data
// is OWNED by web-search: no package anywhere in this API imports a
// knowledge-observatory client (and go.mod carries no such dependency), so a
// write into the knowledge-observatory store is impossible by construction.
// The companion half — findings rows landing in the web-search SQLite file —
// is pinned by the internal/findings repository tests.
func TestFindingsStoreDoesNotTouchKnowledgeObservatory(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if perr != nil {
			return perr
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			require.NotContains(t, strings.ToLower(p), "knowledge-observatory",
				"%s must not import a knowledge-observatory package", path)
		}
		return nil
	})
	require.NoError(t, err)

	gomod, err := os.ReadFile("go.mod")
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(string(gomod)), "knowledge-observatory")
}

// --- capture-policy workflow fixtures ----------------------------------------

type stubSearcher struct{ cands []research.Candidate }

func (s stubSearcher) Candidates(context.Context, string, int) (research.CandidateSet, error) {
	return research.CandidateSet{Candidates: s.cands}, nil
}

type stubFetcher struct{ text string }

func (s stubFetcher) Fetch(context.Context, string) (string, error) { return s.text, nil }

type stubSynth struct{ out research.Synthesis }

func (s stubSynth) Synthesize(context.Context, string, []research.Document) (research.Synthesis, error) {
	return s.out, nil
}

type stubAgentManager struct{ spawned agentmanager.SpawnRequest }

func (s *stubAgentManager) Spawn(_ context.Context, req agentmanager.SpawnRequest) (agentmanager.RunResult, error) {
	s.spawned = req
	return agentmanager.RunResult{RunID: "run-1", TaskID: "task-1", Status: "pending"}, nil
}

func (s *stubAgentManager) GetRunState(context.Context, string) (agentmanager.RunState, error) {
	return agentmanager.RunState{RunID: "run-1", Status: "complete"}, nil
}

// newCaptureFindingsService builds a real findings service (agent actor, as
// production wires for research capture) over a fresh SQLite database.
func newCaptureFindingsService(t *testing.T) findings.Service {
	t.Helper()
	d := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(findings.Schema),
	))
	return findings.NewServiceWithActor(findings.NewSQLiteRepository(d, schedule.System()), "agent")
}

// newL2Research wires the research service over happy-path stubs and the real
// findings store, so capture-policy behavior is observed on actual rows.
func newL2Research(fsvc findings.Service) *research.Service {
	return research.NewService(research.Deps{
		Searcher: stubSearcher{cands: []research.Candidate{{URL: "https://page.example", Title: "Page"}}},
		Fetcher:  stubFetcher{text: "the page says the answer is 42"},
		Synthesizer: stubSynth{out: research.Synthesis{
			Text:      "the answer is 42",
			Citations: []research.Citation{{ResultIndex: 0, URL: "https://page.example", Title: "Page"}},
		}},
		Findings: fsvc,
	})
}

// TestL2WithoutCaptureWritesNoFindings runs the full L2 pipeline against a
// real findings store WITHOUT the capture flag and proves the distillation /
// capture step never fires: the findings table row count stays at zero.
func TestL2WithoutCaptureWritesNoFindings(t *testing.T) {
	ctx := context.Background()
	fsvc := newCaptureFindingsService(t)
	svc := newL2Research(fsvc)

	out, err := svc.RunL2(ctx, "what is the answer", 3, false)
	require.NoError(t, err)
	require.False(t, out.Abstained)
	require.Empty(t, out.CapturedFindingIDs, "no capture flag, no captured ids")

	rows, err := fsvc.List(ctx, findings.ListFilter{IncludeArchived: true})
	require.NoError(t, err)
	require.Empty(t, rows, "L2 without --capture must write nothing to the findings store")
}

// TestL2CaptureEmitsSchemaValidFinding runs L2 WITH capture and validates the
// distilled finding's output schema: a non-empty claim, at least one
// citation, a confidence inside [0,1], and provenance linking it to the L2
// run (source=l2 + the originating query).
func TestL2CaptureEmitsSchemaValidFinding(t *testing.T) {
	ctx := context.Background()
	fsvc := newCaptureFindingsService(t)
	svc := newL2Research(fsvc)

	out, err := svc.RunL2(ctx, "what is the answer", 3, true)
	require.NoError(t, err)
	require.Len(t, out.CapturedFindingIDs, 1, "--capture distills exactly one finding from the synthesis")

	got, err := fsvc.Get(ctx, out.CapturedFindingIDs[0])
	require.NoError(t, err)
	require.NotEmpty(t, strings.TrimSpace(got.Claim), "a distilled finding carries a non-empty claim")
	require.GreaterOrEqual(t, len(got.Citations), 1, "a distilled finding must cite at least one source")
	require.GreaterOrEqual(t, got.Confidence, 0.0)
	require.LessOrEqual(t, got.Confidence, 1.0)
	require.Equal(t, findings.SourceL2, got.Source, "provenance links the finding to the L2 run")
	require.Equal(t, "what is the answer", got.Query)
}

// TestL3PromptEncodesAutoCaptureByDefault pins the L3 auto-capture policy at
// its deterministic seam: spawning an L3 run requires NO capture opt-in, and
// the task prompt the agent executes instructs it to distill citation-backed
// findings into the store (`findings add --source l3`, with confidence and
// citations) as part of the run-end RECONCILE step. This is how an L3 run
// builds the knowledge base automatically.
func TestL3PromptEncodesAutoCaptureByDefault(t *testing.T) {
	am := &stubAgentManager{}
	svc := research.NewService(research.Deps{AgentManager: am})

	_, err := svc.RunL3(context.Background(), "what changed about X")
	require.NoError(t, err)

	prompt := am.spawned.Prompt
	require.Contains(t, prompt, "findings add", "the L3 prompt encodes the distill-and-capture step")
	require.Contains(t, prompt, "--source l3", "captured findings carry L3 provenance")
	require.Contains(t, prompt, "--confidence", "distilled findings carry a confidence score")
	require.Contains(t, prompt, "--citations", "distilled findings carry citations")
	require.Contains(t, prompt, "RECONCILE", "capture happens in the run-end reconcile step")
}
