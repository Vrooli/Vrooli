package signals

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals/boundaries"
)

// Service is the application-layer surface for signal scoring.
type Service interface {
	ScoreChunk(ctx context.Context, in ScoreInput) (Verdict, error)
	ExplainVerdict(ctx context.Context, in ScoreInput) (Verdict, error)
	// ScoreBatch scores every input chunk in one pass: snapshot,
	// domain map, and GraphContext are fetched/built once and the
	// aggregator runs concurrently across the input slice. Returns
	// one Verdict per input Chunk, aligned by index, so a detector
	// can correlate results positionally.
	ScoreBatch(ctx context.Context, in ScoreBatchInput) ([]Verdict, error)
	// ContentScoreBatch scores chunks with path-token authority disabled.
	// This is the content-verdict seam used by placement detectors so a
	// file's current path cannot vote for itself.
	ContentScoreBatch(ctx context.Context, in ScoreBatchInput) ([]Verdict, error)
	ListSignals(ctx context.Context, scenario string) ([]SignalDescriptor, error)
	// BoundaryHealth computes domain-level coupling/boundary-health over the
	// latest snapshot + derived domain map.
	BoundaryHealth(ctx context.Context, scenario string) (boundaries.Report, error)
}

// ScoreBatchInput is the explicit input DTO for ScoreBatch. Chunks must
// be already-resolved graph.Chunk values (e.g., obtained from
// snap.Chunks()); the batch path does not do FileID/RepoPath lookup.
type ScoreBatchInput struct {
	Scenario string
	Chunks   []graph.Chunk
}

// MaxBatchWorkers caps the worker-pool concurrency used by ScoreBatch.
// Picked to match the GraphContext-shared-state shape: deterministic,
// modest, and large enough to saturate the per-chunk aggregator cost on
// typical dev hardware. Not configurable in v1 (per plan §8).
const MaxBatchWorkers = 8

// ScoreInput is the explicit input DTO for ScoreChunk / ExplainVerdict.
// Resolution precedence: Chunk → FileID → RepoPath. The service resolves
// FileID or RepoPath against the latest snapshot to build a Chunk.
type ScoreInput struct {
	Scenario string
	Chunk    graph.Chunk
	FileID   string
	RepoPath string
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
	maxWorkers  int
}

// Option customizes the signals service.
type Option func(*service)

// WithBoundaryConfig sets the coupling/boundary-health thresholds used by
// BoundaryHealth. Defaults to boundaries.DefaultConfig.
func WithBoundaryConfig(cfg boundaries.Config) Option {
	return func(s *service) { s.boundaryCfg = cfg }
}

// WithMaxBatchWorkers caps per-request ScoreBatch worker fan-out.
func WithMaxBatchWorkers(workers int) Option {
	return func(s *service) {
		if workers < 1 {
			workers = 1
		}
		if workers > MaxBatchWorkers {
			workers = MaxBatchWorkers
		}
		s.maxWorkers = workers
	}
}

func NewService(reg *Registry, agg *Aggregator, snapshots SnapshotProvider, domainMaps DomainMapProvider, opts ...Option) Service {
	s := &service{registry: reg, aggregator: agg, snapshots: snapshots, domainMaps: domainMaps, boundaryCfg: boundaries.DefaultConfig(), maxWorkers: defaultBatchWorkers()}
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
	if in.Chunk.ID == "" && strings.TrimSpace(in.FileID) == "" && strings.TrimSpace(in.RepoPath) == "" {
		return Verdict{}, ErrInvalidScoreRequest{Field: "chunk", Reason: "chunk, file_id, or repo_path required"}
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
		repoPath := strings.TrimSpace(in.RepoPath)
		fileID := strings.TrimSpace(in.FileID)
		for _, c := range snap.Chunks() {
			if fileID != "" && c.FileID == fileID {
				chunk = c
				break
			}
			if repoPath != "" && c.Path == repoPath {
				chunk = c
				break
			}
		}
		if chunk.ID == "" {
			field, reason := "file_id", "not found in latest snapshot"
			if fileID == "" && repoPath != "" {
				field, reason = "repo_path", "no file with that path in latest snapshot"
			}
			return Verdict{}, ErrInvalidScoreRequest{Field: field, Reason: reason}
		}
	}
	gctx := NewGraphContext(in.Scenario, snap, dmap)
	verdict := s.aggregator.Aggregate(ctx, gctx, chunk)
	return verdict, nil
}

func (s *service) ListSignals(ctx context.Context, _ string) ([]SignalDescriptor, error) {
	return s.registry.Describe(ctx), nil
}

func (s *service) ScoreBatch(ctx context.Context, in ScoreBatchInput) ([]Verdict, error) {
	return s.scoreBatch(ctx, in, s.aggregator)
}

func (s *service) ContentScoreBatch(ctx context.Context, in ScoreBatchInput) ([]Verdict, error) {
	contentAggregator := s.aggregator.WithWeightOverrides(map[string]float64{"path-token": 0})
	return s.scoreBatch(ctx, in, contentAggregator)
}

func (s *service) scoreBatch(ctx context.Context, in ScoreBatchInput, aggregator *Aggregator) ([]Verdict, error) {
	if strings.TrimSpace(in.Scenario) == "" {
		return nil, ErrInvalidScoreRequest{Field: "scenario", Reason: "required"}
	}
	if len(in.Chunks) == 0 {
		return nil, nil
	}
	snap, err := s.snapshots.GetLatestSnapshot(ctx, in.Scenario)
	if err != nil {
		return nil, err
	}
	dmap, err := s.domainMaps.GetDomainMap(ctx, in.Scenario)
	if err != nil {
		// Same tolerance as ScoreChunk: a scenario with no derivable
		// domain map still scores via signals that don't consult the
		// map (import-cluster, symbol-glossary partial).
		var (
			noAuthority domains.ErrNoAuthority
			notFound    domains.ErrScenarioNotFound
		)
		if !errors.As(err, &noAuthority) && !errors.As(err, &notFound) {
			return nil, err
		}
	}
	gctx := NewGraphContext(in.Scenario, snap, dmap)

	out := make([]Verdict, len(in.Chunks))
	workers := s.maxWorkers
	if workers > len(in.Chunks) {
		workers = len(in.Chunks)
	}
	if workers < 1 {
		workers = 1
	}
	idxCh := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idxCh {
				if ctx.Err() != nil {
					return
				}
				out[i] = aggregator.Aggregate(ctx, gctx, in.Chunks[i])
			}
		}()
	}
	for i := range in.Chunks {
		select {
		case <-ctx.Done():
			close(idxCh)
			wg.Wait()
			return nil, ctx.Err()
		case idxCh <- i:
		}
	}
	close(idxCh)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func defaultBatchWorkers() int {
	workers := runtime.NumCPU()
	if workers > MaxBatchWorkers {
		workers = MaxBatchWorkers
	}
	if workers < 1 {
		workers = 1
	}
	return workers
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
