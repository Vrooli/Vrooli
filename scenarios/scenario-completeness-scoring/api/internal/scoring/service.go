package scoring

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"

	"scenario-completeness-scoring/internal/freshness"
	"scenario-completeness-scoring/internal/signals"
)

// ErrUnknownScenario reports a scenario name that does not resolve to a
// directory under the scenarios root.
var ErrUnknownScenario = errors.New("unknown scenario")

// Service computes score payloads. The core path is filesystem-only: no
// network, no subprocesses (OT-P0-001).
type Service struct {
	scenariosRoot string
	signals       *signals.Service
	freshness     *freshness.Service
	now           func() time.Time
}

// Option configures a Service.
type Option func(*Service)

// WithScenariosRoot overrides scenarios-root resolution (tests).
func WithScenariosRoot(root string) Option {
	return func(s *Service) { s.scenariosRoot = root }
}

// WithClock overrides the timestamp source (tests).
func WithClock(now func() time.Time) Option {
	return func(s *Service) { s.now = now }
}

// New builds a Service. When no root override is given it resolves the
// Vrooli repo root the same way the legacy implementation did: VROOLI_ROOT
// env override, else repo-contract discovery from the working directory.
func New(opts ...Option) (*Service, error) {
	s := &Service{
		signals:   signals.NewService(),
		freshness: freshness.New(),
		now:       time.Now,
	}
	for _, o := range opts {
		o(s)
	}
	if s.scenariosRoot == "" {
		root, err := resolveVrooliRoot()
		if err != nil {
			return nil, fmt.Errorf("resolve vrooli root: %w", err)
		}
		s.scenariosRoot = filepath.Join(root, "scenarios")
	}
	return s, nil
}

func resolveVrooliRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return repocontract.FindRepoRootFromPath(root)
	}
	return repocontract.ResolveRepoRoot()
}

// GetScore assembles the full payload for one scenario.
func (s *Service) GetScore(scenario string) (Result, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" || strings.ContainsAny(scenario, `/\`) {
		return Result{}, fmt.Errorf("%w: %q", ErrUnknownScenario, scenario)
	}
	root := filepath.Join(s.scenariosRoot, scenario)
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return Result{}, fmt.Errorf("%w: %q", ErrUnknownScenario, scenario)
	}

	snap := s.signals.Collect(scenario, root)
	comp := computeComposite(snap)
	mat := deriveMaturity(snap)
	fresh := s.freshness.Check(scenario, root)
	recs := buildRecommendations(snap, comp, mat)

	return Result{
		Scenario:     scenario,
		Category:     snap.Category,
		Maturity:     mat,
		Composite:    comp,
		Freshness:    fresh,
		Recommends:   recs,
		ActionPlan:   buildActionPlan(comp, recs),
		Degradations: snap.Degradations,
		CalculatedAt: s.now(),
	}, nil
}
