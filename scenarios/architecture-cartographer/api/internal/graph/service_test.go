package graph_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/mocks"
)

func newClock() clock.Clock { return clock.System{} }

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
