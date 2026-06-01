package dependencies

import (
	"context"
	"errors"
	"testing"
	"time"

	"security-health/internal/clock"
	"security-health/internal/dependencies/aisearch"
)

// fakeIndex is a test SemanticIndex: Query returns a scripted ranking, and an
// optional error forces the fallback-to-TEXT path.
type fakeIndex struct {
	available bool
	queryErr  error
	ranking   []aisearch.KeyScore
	points    int // reported by CountPoints; drives the readiness gate
	queries   int
	syncs     int
	lastSync  []aisearch.Item
}

func (f *fakeIndex) EnsureCollection(context.Context) error { return nil }
func (f *fakeIndex) Sync(_ context.Context, items []aisearch.Item) (int, int, error) {
	f.syncs++
	f.lastSync = items
	f.points = len(items)
	return len(items), 0, nil
}

func (f *fakeIndex) Query(context.Context, string, int) ([]aisearch.KeyScore, error) {
	f.queries++
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return f.ranking, nil
}
func (f *fakeIndex) CountPoints(context.Context) (int, error) { return f.points, nil }
func (f *fakeIndex) Available(context.Context) (bool, bool)   { return f.available, f.available }

type rankPair struct {
	rec   DependencyRecord
	score float64
}

// pkgRanking maps records to a package-keyed ranking (the index is keyed by
// package identity, not dep_key), pairing each with the given score.
func pkgRanking(pairs ...rankPair) []aisearch.KeyScore {
	out := make([]aisearch.KeyScore, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, aisearch.KeyScore{Key: packageKey(p.rec), Score: p.score})
	}
	return out
}

func seedCorpus(t *testing.T) (*Store, []DependencyRecord) {
	t.Helper()
	s := newTestStore(t)
	recs := []DependencyRecord{
		{Scenario: "alpha", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", SourceFile: "api/go.mod", VulnIDs: []string{"GO-2024-1"}, MaxSeverity: "high"},
		{Scenario: "beta", Ecosystem: EcosystemNPM, Name: "react", Version: "18.0.0", SourceFile: "ui/pnpm-lock.yaml"},
		{Scenario: "gamma", Ecosystem: EcosystemNPM, Name: "esbuild", Version: "0.21.5", SourceFile: "ui/pnpm-lock.yaml", VulnIDs: []string{"GHSA-x"}, MaxSeverity: "moderate"},
	}
	if err := s.Apply(context.Background(), "", recs, "2026-06-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	return s, recs
}

func newSvc(t *testing.T, s *Store, idx SemanticIndex) *Service {
	t.Helper()
	return NewService(Deps{
		RepoRoot:  t.TempDir(),
		Store:     s,
		Annotator: NewAnnotator("", noopCommander{}),
		Clock:     clock.System{},
		Index:     idx,
	})
}

func TestSearch_AIRankingPreservedAndHydrated(t *testing.T) {
	ctx := context.Background()
	s, recs := seedCorpus(t)
	idx := &fakeIndex{
		available: true,
		points:    3, // ≥ ceil(3*0.95) ⇒ index ready, AI served
		ranking: pkgRanking(
			rankPair{recs[2], 0.95}, // esbuild first
			rankPair{recs[0], 0.80}, // x/net second
		),
	}
	svc := newSvc(t, s, idx)

	resp, err := svc.Search(ctx, SearchRequest{Query: "frontend bundler vuln", Mode: ModeAI})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ModeUsed != ModeAI {
		t.Fatalf("ModeUsed = %q, want ai", resp.ModeUsed)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("got %d results, want 2", len(resp.Results))
	}
	if resp.Results[0].Record.Name != "esbuild" || resp.Results[1].Record.Name != "golang.org/x/net" {
		t.Errorf("AI ranking not preserved: %+v", resp.Results)
	}
	if resp.Results[0].Score != 0.95 {
		t.Errorf("vector score not carried through: %v", resp.Results[0].Score)
	}
}

func TestSearch_AIRespectsStructuredFilters(t *testing.T) {
	ctx := context.Background()
	s, recs := seedCorpus(t)
	idx := &fakeIndex{
		available: true,
		points:    3,
		ranking: pkgRanking(
			rankPair{recs[0], 0.9}, // x/net (go, vulnerable)
			rankPair{recs[1], 0.8}, // react (npm, clean)
			rankPair{recs[2], 0.7}, // esbuild (npm, vulnerable)
		),
	}
	svc := newSvc(t, s, idx)

	// vulnerable-only + npm should keep only esbuild out of the AI ranking.
	resp, err := svc.Search(ctx, SearchRequest{Query: "anything", Mode: ModeAI, Ecosystem: EcosystemNPM, VulnerableOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ModeUsed != ModeAI {
		t.Fatalf("ModeUsed = %q, want ai", resp.ModeUsed)
	}
	if len(resp.Results) != 1 || resp.Results[0].Record.Name != "esbuild" {
		t.Fatalf("AI filters wrong: %+v", resp.Results)
	}
}

func TestSearch_AIFallsBackToTextOnError(t *testing.T) {
	ctx := context.Background()
	s, _ := seedCorpus(t)
	idx := &fakeIndex{available: true, points: 3, queryErr: errors.New("qdrant down")}
	svc := newSvc(t, s, idx)

	resp, err := svc.Search(ctx, SearchRequest{Query: "react", Mode: ModeAI})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ModeUsed != ModeText {
		t.Fatalf("expected TEXT fallback on AI error, got %q", resp.ModeUsed)
	}
	if len(resp.Results) != 1 || resp.Results[0].Record.Name != "react" {
		t.Fatalf("TEXT fallback result wrong: %+v", resp.Results)
	}
}

func TestSearch_TextModeNeverConsultsIndex(t *testing.T) {
	ctx := context.Background()
	s, _ := seedCorpus(t)
	idx := &fakeIndex{available: true}
	svc := newSvc(t, s, idx)

	if _, err := svc.Search(ctx, SearchRequest{Query: "react", Mode: ModeText}); err != nil {
		t.Fatal(err)
	}
	if idx.queries != 0 {
		t.Errorf("TEXT mode must not query the semantic index, got %d queries", idx.queries)
	}
}

func TestSearch_AIEmptyIndexFallsBackToText(t *testing.T) {
	ctx := context.Background()
	s, _ := seedCorpus(t)
	// Index reports ready (points high) but Query returns an empty ranking —
	// the defense-in-depth len(scored)==0 fallback should still serve TEXT
	// rather than hand back zero hits the corpus could answer.
	idx := &fakeIndex{available: true, points: 3, ranking: nil}
	svc := newSvc(t, s, idx)

	resp, err := svc.Search(ctx, SearchRequest{Query: "react", Mode: ModeAI})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ModeUsed != ModeText {
		t.Fatalf("empty AI index should fall back to TEXT, got %q", resp.ModeUsed)
	}
	if len(resp.Results) != 1 || resp.Results[0].Record.Name != "react" {
		t.Fatalf("TEXT fallback result wrong: %+v", resp.Results)
	}
}

func TestSyncIndex_FullCorpusSyncsScopedSkips(t *testing.T) {
	ctx := context.Background()
	s, recs := seedCorpus(t)
	idx := &fakeIndex{available: true}
	svc := newSvc(t, s, idx)

	// Full-corpus reconcile (scenario=="") embeds the deduped package universe.
	// seedCorpus has 3 distinct packages, one per scenario, so the count happens
	// to equal the record count here.
	distinct, err := s.DistinctPackageCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	svc.syncIndex(ctx, "")
	if idx.syncs != 1 || len(idx.lastSync) != distinct {
		t.Fatalf("full sync = %d calls, %d items; want 1 call, %d items", idx.syncs, len(idx.lastSync), distinct)
	}
	_ = recs

	// A scoped reindex must NOT sync — Sync would treat the whole collection as
	// the universe and delete every other scenario's vectors.
	svc.syncIndex(ctx, "alpha")
	if idx.syncs != 1 {
		t.Errorf("scoped reindex must not touch the index, got %d total syncs", idx.syncs)
	}
}

func TestSyncIndex_DedupsPackagesAcrossScenarios(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	// Same package (go|golang.org/x/net|v0.17.0) used by three scenarios → one
	// embedded item, not three.
	recs := []DependencyRecord{
		{Scenario: "alpha", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-2024-1"}, MaxSeverity: "high"},
		{Scenario: "beta", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-2024-1"}, MaxSeverity: "high"},
		{Scenario: "gamma", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-2024-1"}, MaxSeverity: "high"},
		{Scenario: "alpha", Ecosystem: EcosystemNPM, Name: "react", Version: "18.0.0"},
	}
	if err := s.Apply(ctx, "", recs, "2026-06-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	idx := &fakeIndex{available: true}
	svc := newSvc(t, s, idx)
	svc.syncIndex(ctx, "")
	if len(idx.lastSync) != 2 {
		t.Fatalf("expected 2 deduped package items, got %d: %+v", len(idx.lastSync), idx.lastSync)
	}
}

func TestSearch_AIFanoutAcrossScenarios(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	// One package exposed by two scenarios + a distinct second package.
	recs := []DependencyRecord{
		{Scenario: "alpha", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-2024-1"}, MaxSeverity: "high"},
		{Scenario: "beta", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-2024-1"}, MaxSeverity: "high"},
		{Scenario: "gamma", Ecosystem: EcosystemNPM, Name: "react", Version: "18.0.0"},
	}
	if err := s.Apply(ctx, "", recs, "2026-06-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	idx := &fakeIndex{
		available: true,
		points:    2,
		ranking: pkgRanking(
			rankPair{recs[0], 0.95}, // x/net package (fans out to alpha + beta)
			rankPair{recs[2], 0.50}, // react
		),
	}
	svc := newSvc(t, s, idx)
	resp, err := svc.Search(ctx, SearchRequest{Query: "serialization CVE", Mode: ModeAI})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ModeUsed != ModeAI {
		t.Fatalf("ModeUsed = %q, want ai", resp.ModeUsed)
	}
	// The matched package fans out to BOTH exposed scenarios, ahead of react.
	if len(resp.Results) != 3 {
		t.Fatalf("got %d results, want 3 (x/net×2 + react)", len(resp.Results))
	}
	if resp.Results[0].Record.Name != "golang.org/x/net" || resp.Results[1].Record.Name != "golang.org/x/net" {
		t.Errorf("x/net should fan out first across scenarios: %+v", resp.Results)
	}
	// Secondary order within the package is by dep_key (alpha before beta).
	if resp.Results[0].Record.Scenario != "alpha" || resp.Results[1].Record.Scenario != "beta" {
		t.Errorf("fan-out not dep_key-ordered: %+v", resp.Results)
	}
	if resp.Results[0].Score != 0.95 || resp.Results[1].Score != 0.95 {
		t.Errorf("fanned-out records should inherit the package score: %+v", resp.Results)
	}
}

func TestSearch_AIGatedUntilReady(t *testing.T) {
	ctx := context.Background()
	s, recs := seedCorpus(t)
	// Backends up, but coverage below threshold (1/3 < 0.95) ⇒ not ready ⇒ TEXT.
	idx := &fakeIndex{
		available: true,
		points:    1,
		ranking:   pkgRanking(rankPair{recs[1], 0.9}), // react
	}
	svc := newSvc(t, s, idx)

	resp, err := svc.Search(ctx, SearchRequest{Query: "react", Mode: ModeAI})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ModeUsed != ModeText {
		t.Fatalf("AI must be gated until ready: got mode %q", resp.ModeUsed)
	}
	if idx.queries != 0 {
		t.Errorf("a not-ready index must not be queried, got %d queries", idx.queries)
	}

	// Coverage reaches threshold ⇒ AI served. Force a cache refresh by zeroing
	// the check time (TTL would otherwise keep the stale not-ready verdict).
	idx.points = 3
	svc.readyCheckedAt = time.Time{}
	resp, err = svc.Search(ctx, SearchRequest{Query: "react", Mode: ModeAI})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ModeUsed != ModeAI {
		t.Fatalf("AI should be served once ready: got mode %q", resp.ModeUsed)
	}
}

func TestRefreshReadiness_Threshold(t *testing.T) {
	ctx := context.Background()
	s, _ := seedCorpus(t) // 3 distinct packages ⇒ ceil(3*0.95)=3 needed
	cases := []struct {
		points int
		want   bool
	}{
		{0, false},
		{2, false}, // 2/3 ≈ 67% < 95%
		{3, true},  // 100% ≥ 95%
	}
	for _, c := range cases {
		idx := &fakeIndex{available: true, points: c.points}
		svc := newSvc(t, s, idx)
		svc.refreshReadiness(ctx)
		svc.readyMu.Lock()
		got := svc.indexReady
		exp := svc.expectedVectors
		svc.readyMu.Unlock()
		if exp != 3 {
			t.Fatalf("expectedVectors = %d, want 3", exp)
		}
		if got != c.want {
			t.Errorf("points=%d: ready=%v, want %v", c.points, got, c.want)
		}
	}
}

func TestStatus_ReportsCoverage(t *testing.T) {
	ctx := context.Background()
	s, _ := seedCorpus(t)
	idx := &fakeIndex{available: true, points: 3}
	svc := newSvc(t, s, idx)
	svc.refreshReadiness(ctx)
	st, err := svc.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st.IndexedVectors != 3 || st.ExpectedVectors != 3 || !st.IndexReady {
		t.Errorf("coverage not reported: %+v", st)
	}
}

func TestStatus_ReportsIndexAvailability(t *testing.T) {
	ctx := context.Background()
	s, _ := seedCorpus(t)
	svc := newSvc(t, s, &fakeIndex{available: true})
	st, err := svc.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Ollama || !st.Qdrant {
		t.Errorf("status should reflect index availability: %+v", st)
	}
}
