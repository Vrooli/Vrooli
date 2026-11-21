package runannotations

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// ErrPackNotFound says the requested pack name is not the loaded pack.
var ErrPackNotFound = errors.New("annotation pack not found")

// ErrSelectionUnavailable says no run selector is wired; the caller must name
// runs explicitly or the orchestrator must provide one.
var ErrSelectionUnavailable = errors.New("annotation run selection unavailable")

// RunSelector supplies a bounded run set for a batch. The SQLite adapter is
// RunSelector in features_sqlite.go; tests use SliceSelector.
type RunSelector interface {
	CandidateRuns(ctx context.Context, since time.Time, limit int) ([]string, error)
}

// SliceSelector is a deterministic selector for tests.
type SliceSelector []string

// CandidateRuns implements RunSelector.
func (s SliceSelector) CandidateRuns(_ context.Context, _ time.Time, limit int) ([]string, error) {
	if limit > 0 && len(s) > limit {
		return append([]string(nil), s[:limit]...), nil
	}
	return append([]string(nil), s...), nil
}

// WindowSelector is an optional selector that also honors an upper bound.
type WindowSelector interface {
	CandidateRunsWindow(ctx context.Context, since, until time.Time, limit int) ([]string, error)
}

// HashedWindowSelector is an optional selector that also knows the pack's
// current definition hashes. A selector that implements it can prefer runs
// that this pack has not annotated yet (or annotated longest ago), which turns
// a bounded pass into forward progress instead of re-reading the same newest
// rows every interval.
type HashedWindowSelector interface {
	CandidateRunsWindowFor(ctx context.Context, since, until time.Time, limit int, definitionHashes []string) ([]string, error)
}

// RunCensus optionally supplies the candidate-run denominator for coverage.
// Without one, coverage's eligible_runs is the annotated population and the
// response says so explicitly.
type RunCensus interface {
	EligibleRuns(ctx context.Context, since, until time.Time) (int, error)
}

// SignalReader reads existing operator signals for one run. The interface is
// narrow so the fixture path never needs the interventions package.
type SignalReader interface {
	RunSignals(ctx context.Context, runID string) (map[string]SignalFact, error)
}

// Options compose a Service. A nil Decisions client makes decision labels
// state unavailable rather than failing the run.
type Options struct {
	Store          *Store
	Features       FeatureSource
	Signals        SignalReader
	Selector       RunSelector
	Decisions      DecisionClient
	Census         RunCensus
	Clock          func() time.Time
	MaxDigestChars int
	CEL            celEvaluator
	// Limits supplies each decision role's declared model limits. Without it the
	// pack's batch and digest sizes stand.
	Limits LimitsSource
	// StateRoot is the scenario's durable state directory (the one holding its
	// database). The gold root resolves against it. Empty leaves a relative
	// gold root unresolved, which the CLI reports instead of guessing a path.
	StateRoot string
}

// Service evaluates a pack and persists observations.
type Service struct {
	pack        *CompiledPack
	store       *Store
	features    FeatureSource
	signals     SignalReader
	selector    RunSelector
	decisions   DecisionClient
	census      RunCensus
	now         func() time.Time
	digestLimit int
	cel         celEvaluator
	limits      LimitsSource
	roles       *resolver
	stateRoot   string
}

// NewService validates its dependencies and returns a service.
func NewService(pack *CompiledPack, opts Options) (*Service, error) {
	if pack == nil {
		return nil, errors.New("annotation pack is required")
	}
	if opts.Store == nil {
		return nil, errors.New("annotation store is required")
	}
	if opts.Features == nil {
		return nil, errors.New("annotation feature source is required")
	}
	if opts.CEL.CostLimit != 0 && opts.CEL.CostLimit != defaultCostLimit {
		recompiled, err := pack.Pack.CompileWithCostLimit(opts.CEL.CostLimit)
		if err != nil {
			return nil, err
		}
		pack = recompiled
	}
	service := &Service{
		pack: pack, store: opts.Store, features: opts.Features, signals: opts.Signals,
		selector: opts.Selector, decisions: opts.Decisions, census: opts.Census, now: opts.Clock,
		digestLimit: opts.MaxDigestChars, cel: opts.CEL, limits: opts.Limits, roles: newResolver(), stateRoot: opts.StateRoot,
	}
	if service.now == nil {
		service.now = time.Now
	}
	if service.digestLimit <= 0 {
		service.digestLimit = DefaultDigestLimit
	}
	if service.cel.CostLimit == 0 {
		service.cel.CostLimit = defaultCostLimit
	}
	return service, nil
}

// OpenService loads a compiled pack and applies the annotation schema.
func OpenService(ctx context.Context, packPath string, store *Store, opts Options) (*Service, error) {
	pack, err := LoadCompiled(packPath)
	if err != nil {
		return nil, err
	}
	if err := store.EnsureSchema(ctx); err != nil {
		return nil, err
	}
	return NewService(pack, opts)
}

// Pack returns the compiled pack.
func (s *Service) Pack() *CompiledPack { return s.pack }

// DefinitionHashes returns the pack's current label definition hashes in a
// stable order. A selector uses them to recognize runs this pack has already
// answered, without knowing which label produced the row.
func (s *Service) DefinitionHashes() []string {
	if s == nil || s.pack == nil {
		return nil
	}
	out := make([]string, 0, len(s.pack.Pack.Labels))
	for _, declaration := range s.pack.Pack.Labels {
		if hash := s.pack.LabelHashes[declaration.ID]; hash != "" {
			out = append(out, hash)
		}
	}
	return out
}

// PackSummary returns the loaded pack identity.
func (s *Service) PackSummary() PackSummary {
	summary := s.pack.Summary()
	summary.GoldRoot = s.GoldRoot()
	return summary
}

// DefaultGoldDir is the directory under the state root gold lives in when the
// pack names none.
const DefaultGoldDir = "gold"

// GoldRoot is the absolute directory gold files are read from and written to:
// runtime.gold_root, or DefaultGoldDir, resolved against the durable state
// root. It is empty when the root is relative and no state root is known.
func (s *Service) GoldRoot() string {
	root := s.pack.Pack.Runtime.GoldRoot
	if root == "" {
		root = DefaultGoldDir
	}
	if !filepath.IsAbs(root) {
		if s.stateRoot == "" {
			return ""
		}
		root = filepath.Join(s.stateRoot, root)
	}
	return filepath.Clean(root)
}

// PackSummaryFor rejects a pack name that is not this service's pack.
func (s *Service) PackSummaryFor(name string) (PackSummary, error) {
	if name != "" && name != s.pack.Pack.Name {
		return PackSummary{}, fmt.Errorf("%w: %s", ErrPackNotFound, name)
	}
	return s.PackSummary(), nil
}
