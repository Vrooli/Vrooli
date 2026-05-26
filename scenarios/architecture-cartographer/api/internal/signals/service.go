package signals

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"
)

// Service is the application-layer surface for signal scoring.
type Service interface {
	ScoreChunk(ctx context.Context, in ScoreInput) (Verdict, error)
	ExplainVerdict(ctx context.Context, in ScoreInput) (Verdict, error)
	ListSignals(ctx context.Context, scenario string) ([]SignalDescriptor, error)
}

// ScoreInput is the explicit input DTO for ScoreChunk / ExplainVerdict.
type ScoreInput struct {
	Scenario string
	Chunk    graph.Chunk
	FileID   string
}

// SnapshotProvider is the seam the service consults to resolve a
// scenario name to its latest graph snapshot. Production wires the
// graph.Service; tests pass a fake.
type SnapshotProvider interface {
	GetLatestSnapshot(ctx context.Context, scenario string) (graph.GraphSnapshot, error)
}

// ManifestProvider resolves a scenario name to its parsed manifest.
type ManifestProvider interface {
	GetManifest(ctx context.Context, scenario string) (manifest.ManifestDefinition, error)
}

type service struct {
	registry   *Registry
	aggregator *Aggregator
	snapshots  SnapshotProvider
	manifests  ManifestProvider
}

func NewService(reg *Registry, agg *Aggregator, snapshots SnapshotProvider, manifests ManifestProvider) Service {
	return &service{registry: reg, aggregator: agg, snapshots: snapshots, manifests: manifests}
}

var _ Service = (*service)(nil)

func (s *service) ScoreChunk(ctx context.Context, in ScoreInput) (Verdict, error) {
	return s.score(ctx, in)
}

func (s *service) ExplainVerdict(ctx context.Context, in ScoreInput) (Verdict, error) {
	return s.score(ctx, in)
}

func (s *service) score(ctx context.Context, in ScoreInput) (Verdict, error) {
	if strings.TrimSpace(in.Scenario) == "" {
		return Verdict{}, ErrInvalidScoreRequest{Field: "scenario", Reason: "required"}
	}
	if in.Chunk.ID == "" && strings.TrimSpace(in.FileID) == "" {
		return Verdict{}, ErrInvalidScoreRequest{Field: "chunk", Reason: "chunk or file_id required"}
	}

	snap, err := s.snapshots.GetLatestSnapshot(ctx, in.Scenario)
	if err != nil {
		return Verdict{}, err
	}
	man, err := s.manifests.GetManifest(ctx, in.Scenario)
	if err != nil {
		var notFound manifest.ErrManifestNotFound
		if !errors.As(err, &notFound) {
			return Verdict{}, err
		}
	}
	chunk := in.Chunk
	if chunk.ID == "" {
		for _, c := range snap.Chunks() {
			if c.FileID == in.FileID {
				chunk = c
				break
			}
		}
		if chunk.ID == "" {
			return Verdict{}, ErrInvalidScoreRequest{Field: "file_id", Reason: "not found in latest snapshot"}
		}
	}
	gctx := NewGraphContext(in.Scenario, snap, man)
	verdict := s.aggregator.Aggregate(ctx, gctx, chunk)
	return verdict, nil
}

func (s *service) ListSignals(ctx context.Context, _ string) ([]SignalDescriptor, error) {
	return s.registry.Describe(ctx), nil
}
