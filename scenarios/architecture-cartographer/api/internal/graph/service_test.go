package graph_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/mocks"

	"github.com/vrooli/api-core/schedule"
)

func newClock() schedule.Clock { return schedule.System() }

func newAdapter(lang graph.Language, raw graph.RawGraph) *mocks.FakeCodeGraphAdapter {
	return &mocks.FakeCodeGraphAdapter{
		NameValue:      string(lang),
		LanguagesValue: []graph.Language{lang},
		Raw:            raw,
	}
}

func TestService_ExtractGraph_RejectsEmptyScenario(t *testing.T) {
	svc := graph.NewService(&mocks.FakeRepository{}, newClock(), &mocks.FakeCodeGraphAdapter{})
	_, _, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{})
	var typed graph.ErrInvalidExtractRequest
	if !errors.As(err, &typed) {
		t.Fatalf("want ErrInvalidExtractRequest, got %v", err)
	}
}

func TestService_ExtractGraph_NoAdapter(t *testing.T) {
	svc := graph.NewService(&mocks.FakeRepository{}, newClock())
	_, _, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "demo"})
	var typed graph.IntegrationError
	if !errors.As(err, &typed) {
		t.Fatalf("want IntegrationError, got %v", err)
	}
}

func TestService_ExtractGraph_SkipsUnreachableAdapter(t *testing.T) {
	// One adapter's backing producer is down (scenario_unreachable); the
	// other succeeds. The extract must skip the unreachable one and still
	// produce a snapshot from the reachable one rather than aborting.
	repo := &mocks.FakeRepository{}
	down := &mocks.FakeCodeGraphAdapter{
		NameValue:      "typescript",
		LanguagesValue: []graph.Language{graph.LanguageTypeScript},
		ExtractErr:     graph.IntegrationError{Kind: "scenario_unreachable", Scenario: "typescript-code-graph"},
	}
	up := newAdapter(graph.LanguageGo, graph.RawGraph{
		Languages: []graph.Language{graph.LanguageGo},
		Packages:  []graph.PackageNode{{ID: "pkg:demo", ImportPath: "demo", Language: graph.LanguageGo}},
	})
	svc := graph.NewService(repo, newClock(), down, up)

	snap, fromCache, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if fromCache {
		t.Fatal("first extract should not be from cache")
	}
	if snap.ContentHash == "" {
		t.Fatal("expected a snapshot from the reachable adapter")
	}
	if len(repo.Snapshots) != 1 {
		t.Fatalf("want 1 persisted snapshot, got %d", len(repo.Snapshots))
	}
}

func TestService_ExtractGraph_PropagatesNonUnreachableError(t *testing.T) {
	// A non-unreachable adapter error (e.g. invalid_argument) must still
	// abort the whole extract — graceful-skip is scoped to producers that
	// are simply not running.
	bad := &mocks.FakeCodeGraphAdapter{
		NameValue:      "typescript",
		LanguagesValue: []graph.Language{graph.LanguageTypeScript},
		ExtractErr:     graph.IntegrationError{Kind: "invalid_argument", Scenario: "typescript-code-graph"},
	}
	svc := graph.NewService(&mocks.FakeRepository{}, newClock(), bad)
	_, _, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "demo"})
	var ie graph.IntegrationError
	if !errors.As(err, &ie) || ie.Kind != "invalid_argument" {
		t.Fatalf("want invalid_argument IntegrationError to propagate, got %v", err)
	}
}

func TestService_ExtractGraph_PersistsAndCaches(t *testing.T) {
	repo := &mocks.FakeRepository{}
	raw := graph.RawGraph{
		Languages: []graph.Language{graph.LanguageGo},
		Files: []graph.FileNode{
			{ID: "file:a.go", Path: "a.go", PackageID: "pkg:demo", Language: graph.LanguageGo},
		},
		Packages: []graph.PackageNode{
			{ID: "pkg:demo", ImportPath: "demo", Language: graph.LanguageGo},
		},
	}
	svc := graph.NewService(repo, newClock(), newAdapter(graph.LanguageGo, raw))

	first, fromCache, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if fromCache {
		t.Fatal("first extract should not be from cache")
	}
	if first.ContentHash == "" {
		t.Fatal("expected content_hash populated")
	}
	if len(repo.Snapshots) != 1 {
		t.Fatalf("want 1 persisted snapshot, got %d", len(repo.Snapshots))
	}

	second, fromCache, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Extract second: %v", err)
	}
	if !fromCache {
		t.Fatal("second extract should be from cache")
	}
	if second.ContentHash != first.ContentHash {
		t.Fatalf("content_hash mismatch %q vs %q", second.ContentHash, first.ContentHash)
	}
}

func TestService_ExtractGraph_SkipPersistenceAvoidsRepositoryCache(t *testing.T) {
	repo := &mocks.FakeRepository{Snapshots: []graph.GraphSnapshot{{Scenario: "control-plane", ContentHash: "same"}}}
	raw := graph.RawGraph{
		Languages: []graph.Language{graph.LanguageGo},
		Files:     []graph.FileNode{{ID: "file:root.go", Path: "root.go", PackageID: "pkg:root", Language: graph.LanguageGo}},
		Packages:  []graph.PackageNode{{ID: "pkg:root", ImportPath: "root", Language: graph.LanguageGo}},
	}
	svc := graph.NewServiceWithFingerprinter(repo, newClock(), fingerprinterFunc(func(context.Context, graph.ExtractGraphInput) (string, error) {
		t.Fatal("one-off extraction must not fingerprint the repository")
		return "", nil
	}), newAdapter(graph.LanguageGo, raw))

	snap, fromCache, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "control-plane", SkipPersistence: true})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if fromCache {
		t.Fatal("one-off extraction must not report a cache hit")
	}
	if snap.ID == "" {
		t.Fatal("one-off snapshot should still have a deterministic in-memory ID")
	}
	if got := repo.FindCalls.Load(); got != 0 {
		t.Fatalf("FindByHash calls=%d want 0", got)
	}
	if got := repo.SourceFindCalls.Load(); got != 0 {
		t.Fatalf("FindBySourceFingerprint calls=%d want 0", got)
	}
	if got := repo.SaveCalls.Load(); got != 0 {
		t.Fatalf("SaveSnapshot calls=%d want 0", got)
	}
}

func TestService_ExtractGraph_SourceFingerprintHitSkipsAdapters(t *testing.T) {
	repo := &mocks.FakeRepository{Snapshots: []graph.GraphSnapshot{{
		ID:                "snap:demo:src",
		Scenario:          "demo",
		ContentHash:       "graph-hash",
		SourceFingerprint: "src:demo",
		Files:             []graph.FileNode{{ID: "file:a.go", Path: "a.go"}},
	}}}
	adapter := newAdapter(graph.LanguageGo, graph.RawGraph{
		Files: []graph.FileNode{{ID: "file:should-not-run", Path: "slow.go"}},
	})
	svc := graph.NewServiceWithFingerprinter(repo, newClock(), fingerprinterFunc(func(context.Context, graph.ExtractGraphInput) (string, error) {
		return "src:demo", nil
	}), adapter)

	snap, fromCache, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if !fromCache {
		t.Fatal("source fingerprint hit should return from cache")
	}
	if snap.ID != "snap:demo:src" {
		t.Fatalf("got snapshot %q", snap.ID)
	}
	if got := adapter.ExtractCalls.Load(); got != 0 {
		t.Fatalf("adapter calls=%d want 0", got)
	}
}

func TestService_ExtractGraph_DegradedSourceFingerprintHitReExtracts(t *testing.T) {
	repo := &mocks.FakeRepository{Snapshots: []graph.GraphSnapshot{{
		ID:                "snap:demo:degraded",
		Scenario:          "demo",
		ContentHash:       "old-graph-hash",
		SourceFingerprint: "src:demo",
		SkippedAdapters:   []string{"typescript"},
	}}}
	adapter := newAdapter(graph.LanguageGo, graph.RawGraph{
		Files: []graph.FileNode{{ID: "file:a.go", Path: "a.go"}},
	})
	svc := graph.NewServiceWithFingerprinter(repo, newClock(), fingerprinterFunc(func(context.Context, graph.ExtractGraphInput) (string, error) {
		return "src:demo", nil
	}), adapter)

	_, fromCache, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if fromCache {
		t.Fatal("degraded source fingerprint hit must re-extract")
	}
	if got := adapter.ExtractCalls.Load(); got != 1 {
		t.Fatalf("adapter calls=%d want 1", got)
	}
}

func TestService_ClearSnapshots_DryRun(t *testing.T) {
	repo := &mocks.FakeRepository{}
	_, _ = repo.SaveSnapshot(context.Background(), graph.GraphSnapshot{Scenario: "demo", ContentHash: "abc"})
	svc := graph.NewService(repo, newClock())
	n, dry, err := svc.ClearSnapshots(context.Background(), "demo", true)
	if err != nil {
		t.Fatalf("Clear dry-run: %v", err)
	}
	if !dry || n != 1 {
		t.Fatalf("want n=1 dry=true, got n=%d dry=%v", n, dry)
	}
	if len(repo.Snapshots) != 1 {
		t.Fatal("dry-run should not delete")
	}
}

type fingerprinterFunc func(context.Context, graph.ExtractGraphInput) (string, error)

func (f fingerprinterFunc) Fingerprint(ctx context.Context, in graph.ExtractGraphInput) (string, error) {
	return f(ctx, in)
}

func TestNormalize_DedupesAndHashesStably(t *testing.T) {
	raw := graph.RawGraph{
		Files: []graph.FileNode{
			{ID: "file:a", Path: "a.go"},
			{ID: "file:a", Path: "a.go"}, // duplicate
			{ID: "file:b", Path: "b.go"},
		},
		Imports: []graph.ImportEdge{
			{From: "file:a", ToPackageID: "pkg:x"},
			{From: "file:a", ToPackageID: "pkg:x"}, // duplicate
		},
	}
	snap1 := graph.Normalize("demo", raw)
	snap2 := graph.Normalize("demo", raw)
	if snap1.ContentHash != snap2.ContentHash {
		t.Fatalf("hash not stable: %q vs %q", snap1.ContentHash, snap2.ContentHash)
	}
	if len(snap1.Files) != 2 {
		t.Fatalf("dedupe files: got %d", len(snap1.Files))
	}
	if len(snap1.Imports) != 1 {
		t.Fatalf("dedupe imports: got %d", len(snap1.Imports))
	}
}

func TestSnapshotClone_DoesNotAlias(t *testing.T) {
	orig := graph.GraphSnapshot{
		Files: []graph.FileNode{{ID: "file:a"}},
	}
	c := orig.Clone()
	c.Files[0].ID = "file:mutated"
	if orig.Files[0].ID != "file:a" {
		t.Fatal("Clone aliased the underlying slice")
	}
}

// TestService_ExtractGraph_SkipTSRecordsSkippedAdapter asserts the
// --skip-ts seam: when ExtractGraphInput.SkipTS is true, the TS adapter
// is dropped before extraction and its name appears in
// snapshot.SkippedAdapters so the audit layer can mark partial.
func TestService_ExtractGraph_SkipTSRecordsSkippedAdapter(t *testing.T) {
	repo := &mocks.FakeRepository{}
	ts := &mocks.FakeCodeGraphAdapter{
		NameValue:      "typescript",
		LanguagesValue: []graph.Language{graph.LanguageTypeScript},
		// Should NOT be called when SkipTS=true; an error here would
		// indicate the skip didn't happen.
		ExtractErr: graph.IntegrationError{Kind: "invalid_argument", Scenario: "typescript-code-graph"},
	}
	goSide := newAdapter(graph.LanguageGo, graph.RawGraph{
		Languages: []graph.Language{graph.LanguageGo},
		Packages:  []graph.PackageNode{{ID: "pkg:demo", ImportPath: "demo", Language: graph.LanguageGo}},
	})
	svc := graph.NewService(repo, newClock(), ts, goSide)

	snap, _, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "demo", SkipTS: true})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(snap.SkippedAdapters) != 1 || snap.SkippedAdapters[0] != "typescript" {
		t.Fatalf("want SkippedAdapters=[typescript], got %v", snap.SkippedAdapters)
	}
}

// TestService_ExtractGraph_UnimplementedDegradesGracefully proves an
// adapter returning IntegrationError{Kind:"unimplemented"} (the tscg
// workspace_unsupported signal) is dropped silently and recorded in
// SkippedAdapters — the audit then maps to outcome=partial instead of
// tool_error.
func TestService_ExtractGraph_UnimplementedDegradesGracefully(t *testing.T) {
	repo := &mocks.FakeRepository{}
	ts := &mocks.FakeCodeGraphAdapter{
		NameValue:      "typescript",
		LanguagesValue: []graph.Language{graph.LanguageTypeScript},
		ExtractErr:     graph.IntegrationError{Kind: "unimplemented", Scenario: "typescript-code-graph"},
	}
	goSide := newAdapter(graph.LanguageGo, graph.RawGraph{
		Languages: []graph.Language{graph.LanguageGo},
		Packages:  []graph.PackageNode{{ID: "pkg:demo", ImportPath: "demo", Language: graph.LanguageGo}},
	})
	svc := graph.NewService(repo, newClock(), ts, goSide)

	snap, _, err := svc.ExtractGraph(context.Background(), graph.ExtractGraphInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(snap.SkippedAdapters) != 1 || snap.SkippedAdapters[0] != "typescript" {
		t.Fatalf("want SkippedAdapters=[typescript], got %v", snap.SkippedAdapters)
	}
}
