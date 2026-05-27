package signals

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals/boundaries"
)

// Service is the application-layer surface for signal scoring.
type Service interface {
	ScoreChunk(ctx context.Context, in ScoreInput) (Verdict, error)
	ExplainVerdict(ctx context.Context, in ScoreInput) (Verdict, error)
	ListSignals(ctx context.Context, scenario string) ([]SignalDescriptor, error)
	// BoundaryHealth computes domain-level coupling/boundary-health over the
	// latest snapshot + derived domain map.
	BoundaryHealth(ctx context.Context, scenario string) (boundaries.Report, error)
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

// DomainMapProvider resolves a scenario name to its derived domain map.
// domains.Service satisfies this directly.
type DomainMapProvider interface {
	GetDomainMap(ctx context.Context, scenario string) (domains.DerivedDomainMap, error)
}

type service struct {
	registry    *Registry
	aggregator  *Aggregator
	snapshots   SnapshotProvider
	domainMaps  DomainMapProvider
	boundaryCfg boundaries.Config
}

// Option customizes the signals service.
type Option func(*service)

// WithBoundaryConfig sets the coupling/boundary-health thresholds used by
// BoundaryHealth. Defaults to boundaries.DefaultConfig.
func WithBoundaryConfig(cfg boundaries.Config) Option {
	return func(s *service) { s.boundaryCfg = cfg }
}

func NewService(reg *Registry, agg *Aggregator, snapshots SnapshotProvider, domainMaps DomainMapProvider, opts ...Option) Service {
	s := &service{registry: reg, aggregator: agg, snapshots: snapshots, domainMaps: domainMaps, boundaryCfg: boundaries.DefaultConfig()}
	for _, o := range opts {
		o(s)
	}
	return s
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
	dmap, err := s.domainMaps.GetDomainMap(ctx, in.Scenario)
	if err != nil {
		// A scenario with no derivable domain map (no DOMAINS.md, no
		// folders) still scores: signals that don't consult the map (e.g.
		// import-cluster community structure) remain meaningful. Other
		// errors are real failures.
		var (
			noAuthority domains.ErrNoAuthority
			notFound    domains.ErrScenarioNotFound
		)
		if !errors.As(err, &noAuthority) && !errors.As(err, &notFound) {
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
	gctx := NewGraphContext(in.Scenario, snap, dmap)
	verdict := s.aggregator.Aggregate(ctx, gctx, chunk)
	return verdict, nil
}

func (s *service) ListSignals(ctx context.Context, _ string) ([]SignalDescriptor, error) {
	return s.registry.Describe(ctx), nil
}

func (s *service) BoundaryHealth(ctx context.Context, scenario string) (boundaries.Report, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return boundaries.Report{}, ErrInvalidScoreRequest{Field: "scenario", Reason: "required"}
	}
	snap, err := s.snapshots.GetLatestSnapshot(ctx, scenario)
	if err != nil {
		return boundaries.Report{}, err
	}
	dmap, err := s.domainMaps.GetDomainMap(ctx, scenario)
	if err != nil {
		// A scenario with no derivable domain map yields an empty report
		// (nothing to score) rather than an error.
		var (
			noAuthority domains.ErrNoAuthority
			notFound    domains.ErrScenarioNotFound
		)
		if !errors.As(err, &noAuthority) && !errors.As(err, &notFound) {
			return boundaries.Report{}, err
		}
	}
	return boundaries.Analyze(scenario, snap, dmap, s.boundaryCfg), nil
}
