// Package budgets owns per-scenario performance budgets (build time, bundle
// size, LCP, p95-where-available, startup, and component-commit budgets — avg
// and max) plus a ratchet (tighten-only). Budgets are declarative in scenario
// config (.vrooli/perf-budgets.json); CheckBudget evaluates the latest measured
// sample against the budget and reports violations — the signal a baseline-diff
// turns into an exit-1 regression through the maturity finding pipeline.
package budgets

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"

	mga "github.com/vrooli/maturity-go/assessment"
)

// Budget is one scenario's declared performance thresholds (0 = unset).
type Budget struct {
	Scenario       string
	GoBuildMaxMs   int64
	UIBuildMaxMs   int64
	BundleMaxBytes int64
	LCPMaxMs       int64
	P95MaxMs       int64
	StartupMaxMs   int64
	// ComponentCommitAvgMaxMs caps the slowest component's AVERAGE commit time.
	ComponentCommitAvgMaxMs float64
	// ComponentCommitMaxMs caps the slowest component's MAX commit time.
	ComponentCommitMaxMs float64
	// Ratchet, when true, makes SetBudget tighten-only: a write that loosens any
	// declared axis is rejected.
	Ratchet bool
}

// IsSet reports whether the budget declares at least one positive threshold.
func (b Budget) IsSet() bool {
	return b.GoBuildMaxMs > 0 || b.UIBuildMaxMs > 0 || b.BundleMaxBytes > 0 ||
		b.LCPMaxMs > 0 || b.P95MaxMs > 0 || b.StartupMaxMs > 0 ||
		b.ComponentCommitAvgMaxMs > 0 || b.ComponentCommitMaxMs > 0
}

// Measurement is the latest measured value per axis for one scenario. Integer
// axes use the int64 fields; component-commit axes (doubles) use the float
// fields. A zero value means "not measured this run" and is never a violation.
type Measurement struct {
	GoBuildMs            int64
	UIBuildMs            int64
	BundleBytes          int64
	LCPMs                int64
	P95Ms                int64
	StartupMs            int64
	ComponentCommitAvgMs float64
	ComponentCommitMaxMs float64
	// SlowestComponent names the component the avg/max readings came from (for
	// the violation message); optional.
	SlowestComponent string
}

// Violation is one axis exceeding its budget.
type Violation struct {
	Axis     string
	Measured int64
	Budget   int64
	Unit     string
	Detail   string
}

// MeasurementSource supplies the latest measured sample for a scenario. The
// production wiring reads the newest persisted trend sample; tests drive a fake.
// found=false means there is no measured sample yet (Check passes vacuously —
// there is nothing to be over-budget against).
type MeasurementSource interface {
	Latest(ctx context.Context, scenario string) (Measurement, bool, error)
}

// BudgetStore reads and writes declared budgets. The in-memory Store (tests) and
// the ConfigStore (.vrooli/perf-budgets.json) both satisfy it.
type BudgetStore interface {
	Get(ctx context.Context, scenario string) (Budget, bool, error)
	Set(ctx context.Context, b Budget, dryRun bool) (Budget, error)
}

// Store is an in-memory BudgetStore used by tests and as a default seam.
type Store struct {
	mu      sync.RWMutex
	budgets map[string]Budget
}

// NewStore builds an empty in-memory budget store.
func NewStore() *Store { return &Store{budgets: map[string]Budget{}} }

var _ BudgetStore = (*Store)(nil)

// Get returns the declared budget for a scenario and whether one was declared.
func (s *Store) Get(_ context.Context, scenario string) (Budget, bool, error) {
	if s == nil {
		return Budget{}, false, errors.New("budgets: nil store")
	}
	if scenario == "" {
		return Budget{}, false, errors.New("budgets: scenario is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.budgets[scenario]
	if !ok {
		return Budget{Scenario: scenario}, false, nil
	}
	return b, true, nil
}

// Set writes/updates a scenario's budget. When dryRun is true it validates
// (including the ratchet) but does not persist.
func (s *Store) Set(_ context.Context, b Budget, dryRun bool) (Budget, error) {
	if s == nil {
		return Budget{}, errors.New("budgets: nil store")
	}
	if b.Scenario == "" {
		return Budget{}, errors.New("budgets: scenario is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.budgets[b.Scenario]; ok {
		if err := enforceRatchet(existing, b); err != nil {
			return Budget{}, err
		}
	}
	if dryRun {
		return b, nil
	}
	s.budgets[b.Scenario] = b
	return b, nil
}

// enforceRatchet rejects a write that loosens any declared axis when the
// EXISTING budget (or the incoming one) opts into the ratchet. Tightening
// (lowering a max / raising bundle? no — every axis here is a max, so lower is
// always stricter) and leaving an axis unchanged are always allowed.
func enforceRatchet(existing, incoming Budget) error {
	if !existing.Ratchet && !incoming.Ratchet {
		return nil
	}
	loosened := func(axis string, was, now int64) error {
		// now == 0 means "unset/clear" — allowed (you can stop enforcing an axis).
		if was > 0 && now > was {
			return fmt.Errorf("budgets: ratchet violation: %s budget may only tighten (was %d, requested %d)", axis, was, now)
		}
		return nil
	}
	loosenedF := func(axis string, was, now float64) error {
		if was > 0 && now > was {
			return fmt.Errorf("budgets: ratchet violation: %s budget may only tighten (was %.1f, requested %.1f)", axis, was, now)
		}
		return nil
	}
	for _, err := range []error{
		loosened("go_build", existing.GoBuildMaxMs, incoming.GoBuildMaxMs),
		loosened("ui_build", existing.UIBuildMaxMs, incoming.UIBuildMaxMs),
		loosened("bundle", existing.BundleMaxBytes, incoming.BundleMaxBytes),
		loosened("lcp", existing.LCPMaxMs, incoming.LCPMaxMs),
		loosened("p95", existing.P95MaxMs, incoming.P95MaxMs),
		loosened("startup", existing.StartupMaxMs, incoming.StartupMaxMs),
		loosenedF("component_commit_avg", existing.ComponentCommitAvgMaxMs, incoming.ComponentCommitAvgMaxMs),
		loosenedF("component_commit_max", existing.ComponentCommitMaxMs, incoming.ComponentCommitMaxMs),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

// Service is the engine behind BudgetService.
type Service struct {
	store  BudgetStore
	source MeasurementSource
}

// NewService wires a budgets Service over a store; the measurement source is
// optional (nil => Check has no sample to evaluate and reports passed).
func NewService(store BudgetStore, opts ...Option) *Service {
	s := &Service{store: store}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Option customizes a Service.
type Option func(*Service)

// WithMeasurementSource wires the source CheckBudget evaluates against.
func WithMeasurementSource(src MeasurementSource) Option {
	return func(s *Service) { s.source = src }
}

// Get returns a scenario's declared budget.
func (s *Service) Get(ctx context.Context, scenario string) (Budget, bool, error) {
	if s == nil || s.store == nil {
		return Budget{}, false, errors.New("budgets: service not wired")
	}
	return s.store.Get(ctx, scenario)
}

// Set writes a scenario's budget (ratchet enforced by the store).
func (s *Service) Set(ctx context.Context, b Budget, dryRun bool) (Budget, error) {
	if s == nil || s.store == nil {
		return Budget{}, errors.New("budgets: service not wired")
	}
	return s.store.Set(ctx, b, dryRun)
}

// Check evaluates a scenario's latest measured sample against its declared
// budget. With no budget declared, or no measured sample, it passes (there is
// nothing to be over-budget against). Otherwise it returns the sorted
// violations; an empty set means within budget.
func (s *Service) Check(ctx context.Context, scenario string) (bool, []Violation, error) {
	if s == nil || s.store == nil {
		return false, nil, errors.New("budgets: service not wired")
	}
	budget, declared, err := s.store.Get(ctx, scenario)
	if err != nil {
		return false, nil, err
	}
	if !declared || !budget.IsSet() {
		return true, nil, nil
	}
	if s.source == nil {
		return true, nil, nil
	}
	m, found, err := s.source.Latest(ctx, scenario)
	if err != nil {
		return false, nil, err
	}
	if !found {
		return true, nil, nil
	}
	violations := Evaluate(budget, m)
	return len(violations) == 0, violations, nil
}

// Evaluate compares a measured sample against a budget and returns sorted
// violations. It is the pure core CheckBudget uses; a zero measured value for an
// axis is treated as "not measured" and never violates.
func Evaluate(b Budget, m Measurement) []Violation {
	var out []Violation
	checkInt := func(axis string, measured, budget int64, unit string) {
		if budget > 0 && measured > budget {
			out = append(out, Violation{Axis: axis, Measured: measured, Budget: budget, Unit: unit})
		}
	}
	checkInt("go_build", m.GoBuildMs, b.GoBuildMaxMs, "ms")
	checkInt("ui_build", m.UIBuildMs, b.UIBuildMaxMs, "ms")
	checkInt("bundle", m.BundleBytes, b.BundleMaxBytes, "bytes")
	checkInt("lcp", m.LCPMs, b.LCPMaxMs, "ms")
	checkInt("p95", m.P95Ms, b.P95MaxMs, "ms")
	checkInt("startup", m.StartupMs, b.StartupMaxMs, "ms")

	checkFloat := func(axis string, measured, budget float64, detail string) {
		if budget > 0 && measured > budget {
			out = append(out, Violation{
				Axis:     axis,
				Measured: int64(math.Round(measured)),
				Budget:   int64(math.Round(budget)),
				Unit:     "ms",
				Detail:   detail,
			})
		}
	}
	checkFloat("component_commit_avg", m.ComponentCommitAvgMs, b.ComponentCommitAvgMaxMs, m.SlowestComponent)
	checkFloat("component_commit_max", m.ComponentCommitMaxMs, b.ComponentCommitMaxMs, m.SlowestComponent)

	sort.Slice(out, func(i, j int) bool { return out[i].Axis < out[j].Axis })
	return out
}

// Findings projects budget violations into shared maturity findings at ERROR
// severity. An ERROR finding drives the scenario-validation status to FAILED via
// assessment.DeriveValidationStatus, which is how a perf regression fails
// `git-control-tower baseline diff` (exit 1) exactly like any other health
// regression. The code is stable so the finding is dedupable across runs.
func Findings(scenario string, violations []Violation) []mga.Finding {
	out := make([]mga.Finding, 0, len(violations))
	for _, v := range violations {
		title := v.Axis
		msg := fmt.Sprintf("performance budget breach: %s measured %d%s exceeds budget %d%s",
			v.Axis, v.Measured, unitSuffix(v.Unit), v.Budget, unitSuffix(v.Unit))
		if v.Detail != "" {
			msg += " (" + v.Detail + ")"
		}
		out = append(out, mga.Finding{
			Code:             "PERF_BUDGET_BREACH_" + upper(v.Axis),
			Severity:         "error",
			Title:            "Performance budget breach: " + title,
			Message:          msg,
			AutofixAvailable: false,
		})
	}
	return out
}

func unitSuffix(unit string) string {
	switch unit {
	case "ms":
		return "ms"
	case "bytes":
		return "B"
	default:
		return ""
	}
}

func upper(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 'a' - 'A'
		}
	}
	return string(b)
}
