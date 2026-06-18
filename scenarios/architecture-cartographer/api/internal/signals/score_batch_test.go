package signals_test

import (
	"context"
	"reflect"
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/importcluster"
	"architecture-cartographer/internal/signals/pathtoken"
)

// TestScoreBatch_ParityWithScoreChunk asserts that ScoreBatch and
// ScoreChunk produce byte-identical verdicts for the same chunks —
// the reproducibility invariant the batch path must preserve. This
// guards against worker-pool / cache races introducing nondeterminism.
func TestScoreBatch_ParityWithScoreChunk(t *testing.T) {
	snap := graph.GraphSnapshot{
		Scenario: "demo",
		Files: []graph.FileNode{
			{ID: "file:a", Path: "internal/conflicts/a.go", PackageID: "pkg:c", Language: graph.LanguageGo},
			{ID: "file:b", Path: "internal/conflicts/b.go", PackageID: "pkg:c", Language: graph.LanguageGo},
			{ID: "file:c", Path: "internal/graph/c.go", PackageID: "pkg:g", Language: graph.LanguageGo},
			{ID: "file:d", Path: "internal/graph/d.go", PackageID: "pkg:g", Language: graph.LanguageGo},
		},
		Packages: []graph.PackageNode{
			{ID: "pkg:c", RepoPath: "internal/conflicts", Language: graph.LanguageGo},
			{ID: "pkg:g", RepoPath: "internal/graph", Language: graph.LanguageGo},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:c", ToPackageID: "pkg:g"},
		},
	}
	dmap := domains.DerivedDomainMap{
		Scenario: "demo",
		Domains: []domains.DerivedDomain{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
			{Name: "graph", Paths: []string{"internal/graph/**"}},
		},
	}
	snaps := batchStubSnap{snap: snap}
	dmaps := batchStubDmap{dmap: dmap}
	reg := signals.NewRegistry(pathtoken.New(), importcluster.New())
	agg := signals.NewAggregator(reg, nil)
	svc := signals.NewService(reg, agg, snaps, dmaps)

	chunks := snap.Chunks()
	batch, err := svc.ScoreBatch(context.Background(), signals.ScoreBatchInput{Scenario: "demo", Chunks: chunks})
	if err != nil {
		t.Fatalf("ScoreBatch: %v", err)
	}
	if len(batch) != len(chunks) {
		t.Fatalf("want %d verdicts, got %d", len(chunks), len(batch))
	}
	for i, c := range chunks {
		single, err := svc.ScoreChunk(context.Background(), signals.ScoreInput{Scenario: "demo", Chunk: c})
		if err != nil {
			t.Fatalf("ScoreChunk[%d]: %v", i, err)
		}
		if !reflect.DeepEqual(single, batch[i]) {
			t.Fatalf("verdict[%d] diverges between single and batch:\nsingle=%+v\nbatch =%+v", i, single, batch[i])
		}
	}
}

func TestScoreBatch_EmptyInputReturnsNil(t *testing.T) {
	reg := signals.NewRegistry(pathtoken.New())
	svc := signals.NewService(reg, signals.NewAggregator(reg, nil), batchStubSnap{}, batchStubDmap{})
	got, err := svc.ScoreBatch(context.Background(), signals.ScoreBatchInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Fatalf("empty input must return nil verdicts, got %+v", got)
	}
}

func TestContentScoreBatch_DisablesPathTokenAuthority(t *testing.T) {
	snap := graph.GraphSnapshot{
		Scenario: "demo",
		Files: []graph.FileNode{
			{ID: "file:a", Path: "internal/graph/service.go", PackageID: "pkg:g", Language: graph.LanguageGo},
		},
		Packages: []graph.PackageNode{{ID: "pkg:g", RepoPath: "internal/graph", Language: graph.LanguageGo}},
	}
	dmap := domains.DerivedDomainMap{
		Scenario: "demo",
		Domains:  []domains.DerivedDomain{{Name: "graph", Paths: []string{"internal/graph/**"}}},
	}
	reg := signals.NewRegistry(pathtoken.New())
	svc := signals.NewService(reg, signals.NewAggregator(reg, nil), batchStubSnap{snap: snap}, batchStubDmap{dmap: dmap})
	chunks := snap.Chunks()

	normal, err := svc.ScoreBatch(context.Background(), signals.ScoreBatchInput{Scenario: "demo", Chunks: chunks})
	if err != nil {
		t.Fatalf("ScoreBatch: %v", err)
	}
	content, err := svc.ContentScoreBatch(context.Background(), signals.ScoreBatchInput{Scenario: "demo", Chunks: chunks})
	if err != nil {
		t.Fatalf("ContentScoreBatch: %v", err)
	}
	if normal[0].TopDomain != "graph" {
		t.Fatalf("normal verdict should use path-token, got %+v", normal[0])
	}
	if content[0].TopDomain != "" || content[0].DirectionValue != 0 || content[0].Confidence != 0 {
		t.Fatalf("content verdict must not count path-token authority, got %+v", content[0])
	}
	if len(content[0].Scores) == 0 {
		t.Fatalf("content verdict should retain signal evidence for explainability")
	}
}

type batchStubSnap struct {
	snap graph.GraphSnapshot
}

func (s batchStubSnap) GetLatestSnapshot(_ context.Context, _ string) (graph.GraphSnapshot, error) {
	return s.snap, nil
}

type batchStubDmap struct {
	dmap domains.DerivedDomainMap
}

func (d batchStubDmap) GetDomainMap(_ context.Context, _ string) (domains.DerivedDomainMap, error) {
	return d.dmap, nil
}
