package signals_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

type stubSnapshots struct {
	snap graph.GraphSnapshot
}

func (s stubSnapshots) GetLatestSnapshot(_ context.Context, _ string) (graph.GraphSnapshot, error) {
	return s.snap, nil
}

type stubDomainMaps struct{}

func (stubDomainMaps) GetDomainMap(_ context.Context, _ string) (domains.DerivedDomainMap, error) {
	return domains.DerivedDomainMap{Scenario: "demo"}, nil
}

func newSvc(snap graph.GraphSnapshot) signals.Service {
	reg := signals.NewRegistry()
	agg := signals.NewAggregator(reg, nil)
	return signals.NewService(reg, agg, stubSnapshots{snap: snap}, stubDomainMaps{})
}

func TestScoreChunk_ResolvesByRepoPath(t *testing.T) {
	snap := graph.GraphSnapshot{
		Scenario: "demo",
		Files: []graph.FileNode{
			{ID: "file:api/graph/service.go", Path: "api/graph/service.go", PackageID: "pkg:graph", Language: graph.LanguageGo},
		},
	}
	svc := newSvc(snap)
	v, err := svc.ScoreChunk(context.Background(), signals.ScoreInput{
		Scenario: "demo",
		RepoPath: "api/graph/service.go",
	})
	if err != nil {
		t.Fatalf("ScoreChunk: %v", err)
	}
	if v.ChunkPath != "api/graph/service.go" {
		t.Fatalf("verdict ChunkPath = %q, want %q", v.ChunkPath, "api/graph/service.go")
	}
}

func TestScoreChunk_RepoPathNotFoundErrorsClearly(t *testing.T) {
	snap := graph.GraphSnapshot{
		Scenario: "demo",
		Files:    []graph.FileNode{{ID: "file:a.go", Path: "a.go", Language: graph.LanguageGo}},
	}
	svc := newSvc(snap)
	_, err := svc.ScoreChunk(context.Background(), signals.ScoreInput{
		Scenario: "demo",
		RepoPath: "does/not/exist.go",
	})
	var bad signals.ErrInvalidScoreRequest
	if !errors.As(err, &bad) {
		t.Fatalf("err = %v, want ErrInvalidScoreRequest", err)
	}
	if bad.Field != "repo_path" {
		t.Fatalf("err.Field = %q, want %q", bad.Field, "repo_path")
	}
}

func TestScoreChunk_FileIDStillWorks(t *testing.T) {
	snap := graph.GraphSnapshot{
		Scenario: "demo",
		Files: []graph.FileNode{
			{ID: "file:a.go", Path: "a.go", Language: graph.LanguageGo},
		},
	}
	svc := newSvc(snap)
	if _, err := svc.ScoreChunk(context.Background(), signals.ScoreInput{
		Scenario: "demo",
		FileID:   "file:a.go",
	}); err != nil {
		t.Fatalf("ScoreChunk by file id: %v", err)
	}
}
