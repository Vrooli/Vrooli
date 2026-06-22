// Package budgets owns per-scenario performance budgets (build time, bundle
// size, LCP, startup, and component-commit budgets — avg and max) plus a
// ratchet (tighten-only). Budgets are declarative in scenario config under the
// `performance.budgets` block of .vrooli/testing.json — the single source of
// truth for every perf threshold; CheckBudget evaluates the latest measured
// sample against the budget and reports violations — the signal that, as an
// ERROR finding, fails the test-genie Performance phase (and therefore the
// suite run, `vrooli scenario test`) through the maturity finding pipeline.
//
// NB: a breach fails the SUITE RUN, not `git-control-tower baseline diff` — the
// baseline-diff verdict buckets phase results into surfaces (structure/rules/
// tests/workflows/visuals) and the `performance` phase maps to none of them, so
// a perf regression is invisible there. The suite run is the enforcement gate.
package budgets

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
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
	StartupMaxMs   int64
	// ComponentCommitAvgMaxMs caps the slowest component's AVERAGE commit time.
	ComponentCommitAvgMaxMs float64
	// ComponentCommitMaxMs caps the slowest component's MAX commit time.
	ComponentCommitMaxMs float64
	// Ratchet, when true, makes SetBudget tighten-only: a write that loosens any
	// declared axis is rejected.
	Ratchet bool
	// Flows holds per-interaction-flow budgets keyed by flow slug. These gate a
	// specific targeted journey (driven by `audit run --workflow <slug>`) on the
	// continuous cadence, independently of the scenario aggregate. Only the
	// Tier-1/web-vitals axes apply per-flow (build/bundle/startup stay scenario-
	// level — there is no per-flow build).
	Flows map[string]FlowBudget
}

// FlowBudget is one interaction-flow's declared thresholds (0 = unset). It
// carries only the axes a targeted capture can measure: LCP and the slowest
// component's avg/max commit time.
type FlowBudget struct {
	LCPMaxMs                int64
	ComponentCommitAvgMaxMs float64
	ComponentCommitMaxMs    float64
}

// IsSet reports whether the flow budget declares at least one positive threshold.
func (f FlowBudget) IsSet() bool {
	return f.LCPMaxMs > 0 || f.ComponentCommitAvgMaxMs > 0 || f.ComponentCommitMaxMs > 0
}

// IsSet reports whether the budget declares at least one positive threshold
// (scenario-level or any per-flow).
func (b Budget) IsSet() bool {
	if b.GoBuildMaxMs > 0 || b.UIBuildMaxMs > 0 || b.BundleMaxBytes > 0 ||
		b.LCPMaxMs > 0 || b.StartupMaxMs > 0 ||
		b.ComponentCommitAvgMaxMs > 0 || b.ComponentCommitMaxMs > 0 {
		return true
	}
	for _, fb := range b.Flows {
		if fb.IsSet() {
			return true
		}
	}
	return false
}

// Measurement is the latest measured value per axis for one scenario. Integer
// axes use the int64 fields; component-commit axes (doubles) use the float
// fields. A zero value means "not measured this run" and is never a violation.
type Measurement struct {
	GoBuildMs            int64
	UIBuildMs            int64
	BundleBytes          int64
	LCPMs                int64
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
// the ConfigStore (.vrooli/testing.json performance.budgets) both satisfy it.
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
		loosened("startup", existing.StartupMaxMs, incoming.StartupMaxMs),
		loosenedF("component_commit_avg", existing.ComponentCommitAvgMaxMs, incoming.ComponentCommitAvgMaxMs),
		loosenedF("component_commit_max", existing.ComponentCommitMaxMs, incoming.ComponentCommitMaxMs),
	} {
		if err != nil {
			return err
		}
	}
	// Per-flow axes ratchet independently: a flow present in both must only
	// tighten. A newly-added flow has no prior value to loosen against.
	for slug, was := range existing.Flows {
		now, ok := incoming.Flows[slug]
		if !ok {
			continue
		}
		for _, err := range []error{
			loosened("flow:"+slug+".lcp", was.LCPMaxMs, now.LCPMaxMs),
			loosenedF("flow:"+slug+".component_commit_avg", was.ComponentCommitAvgMaxMs, now.ComponentCommitAvgMaxMs),
			loosenedF("flow:"+slug+".component_commit_max", was.ComponentCommitMaxMs, now.ComponentCommitMaxMs),
		} {
			if err != nil {
				return err
			}
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

// Advisories returns the INFO findings describing a scenario's declared-but-
// ungated budget axes (none when no budget is declared or all declared axes are
// synchronously gated). It never fails — advisories only inform.
func (s *Service) Advisories(ctx context.Context, scenario string) ([]mga.Finding, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("budgets: service not wired")
	}
	budget, declared, err := s.store.Get(ctx, scenario)
	if err != nil {
		return nil, err
	}
	if !declared {
		return nil, nil
	}
	return AdvisoryFindings(budget), nil
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

// FlowMeasurementSource supplies the latest flow-tagged measured sample. The
// production sampleMeasurementSource satisfies it; tests drive a fake. found=
// false means the flow has no measured sample yet (CheckFlow passes vacuously).
type FlowMeasurementSource interface {
	LatestFlow(ctx context.Context, scenario, flow string) (Measurement, bool, error)
}

// CheckFlow evaluates one interaction flow's latest flow-tagged sample against
// its per-flow budget. With no per-flow budget declared, no flow-aware source,
// or no flow-tagged sample yet, it passes (nothing to be over-budget against).
func (s *Service) CheckFlow(ctx context.Context, scenario, flow string) (bool, []Violation, error) {
	if s == nil || s.store == nil {
		return false, nil, errors.New("budgets: service not wired")
	}
	if strings.TrimSpace(flow) == "" {
		return false, nil, errors.New("budgets: flow is required")
	}
	budget, declared, err := s.store.Get(ctx, scenario)
	if err != nil {
		return false, nil, err
	}
	if !declared {
		return true, nil, nil
	}
	fb, ok := budget.Flows[flow]
	if !ok || !fb.IsSet() {
		return true, nil, nil
	}
	fs, ok := s.source.(FlowMeasurementSource)
	if s.source == nil || !ok {
		return true, nil, nil
	}
	m, found, err := fs.LatestFlow(ctx, scenario, flow)
	if err != nil {
		return false, nil, err
	}
	if !found {
		return true, nil, nil
	}
	return len(EvaluateFlow(fb, m)) == 0, EvaluateFlow(fb, m), nil
}

// FlowFindings checks every declared per-flow budget against its latest
// flow-tagged sample and returns ERROR findings for breaches (empty when all
// pass or none declared). It is the per-flow analogue of the scenario-level
// budget gate: the validation handler folds these in so a regression on a
// budgeted journey fails the test-genie Performance phase (and therefore the
// suite run, `vrooli scenario test`), evaluated against the flow samples the
// continuous capture-sweep persists. The CHECK is synchronous — it only reads
// the latest persisted flow sample; the browser CAPTURE that produced it runs
// out-of-band in the sweep, never inside the gated run.
func (s *Service) FlowFindings(ctx context.Context, scenario string) ([]mga.Finding, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("budgets: service not wired")
	}
	budget, declared, err := s.store.Get(ctx, scenario)
	if err != nil {
		return nil, err
	}
	if !declared || len(budget.Flows) == 0 {
		return nil, nil
	}
	slugs := make([]string, 0, len(budget.Flows))
	for slug := range budget.Flows {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	var out []mga.Finding
	for _, slug := range slugs {
		passed, violations, err := s.CheckFlow(ctx, scenario, slug)
		if err != nil {
			return nil, err
		}
		if passed || len(violations) == 0 {
			continue
		}
		out = append(out, FindingsForFlow(slug, violations)...)
	}
	return out, nil
}

// EvaluateFlow compares a flow-tagged measured sample against a per-flow budget.
// It reuses Evaluate by projecting the flow budget onto the scenario axes it
// shares (lcp + component-commit); build/bundle/startup stay zero (unset) so
// they never trip per-flow.
func EvaluateFlow(fb FlowBudget, m Measurement) []Violation {
	return Evaluate(Budget{
		LCPMaxMs:                fb.LCPMaxMs,
		ComponentCommitAvgMaxMs: fb.ComponentCommitAvgMaxMs,
		ComponentCommitMaxMs:    fb.ComponentCommitMaxMs,
	}, m)
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
// assessment.DeriveValidationStatus, which is how a perf regression fails the
// test-genie Performance phase — and therefore the suite run (`vrooli scenario
// test` exit 1) — exactly like any other health regression. The code is stable
// so the finding is dedupable across runs.
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

// FindingsForFlow projects per-flow budget violations into ERROR findings,
// tagged by flow slug so each budgeted journey fails the Performance phase
// independently. The code keeps the PERF_BUDGET_BREACH_ prefix (so the existing
// budget-finding handling applies) but namespaces the flow + axis for dedup.
func FindingsForFlow(flow string, violations []Violation) []mga.Finding {
	out := make([]mga.Finding, 0, len(violations))
	for _, v := range violations {
		msg := fmt.Sprintf("performance budget breach on flow %q: %s measured %d%s exceeds budget %d%s",
			flow, v.Axis, v.Measured, unitSuffix(v.Unit), v.Budget, unitSuffix(v.Unit))
		if v.Detail != "" {
			msg += " (" + v.Detail + ")"
		}
		out = append(out, mga.Finding{
			Code:             "PERF_BUDGET_BREACH_" + upper(flow) + "_" + upper(v.Axis),
			Severity:         "error",
			Title:            fmt.Sprintf("Performance budget breach: flow %s / %s", flow, v.Axis),
			Message:          msg,
			AutofixAvailable: false,
		})
	}
	return out
}

// UngatedDeclaredAxes returns the declared budget axes the SYNCHRONOUS
// performance gate cannot MEASURE hermetically. The gate freshly measures build +
// bundle (go_build, ui_build, bundle); lcp, startup, and the component-commit
// axes are measured continuously out-of-band (capture/sweep → trend). A budget
// declared on an ungated axis is still real protection: the breach CHECK runs
// synchronously (it reads the latest persisted sample and fails the Performance
// phase / suite run), only the MEASUREMENT is out-of-band — so the value gated is
// the last out-of-band capture, not this run's code.
func UngatedDeclaredAxes(b Budget) []string {
	var out []string
	if b.LCPMaxMs > 0 {
		out = append(out, "lcp")
	}
	if b.StartupMaxMs > 0 {
		out = append(out, "startup")
	}
	if b.ComponentCommitAvgMaxMs > 0 {
		out = append(out, "component_commit_avg")
	}
	if b.ComponentCommitMaxMs > 0 {
		out = append(out, "component_commit_max")
	}
	return out
}

// AdvisoryFindings projects a budget's declared-but-ungated axes into INFO
// findings so a declared budget can't silently masquerade as freshly-measured
// synchronous protection. INFO severity never fails the gate; it makes the
// freshly-measured-vs-continuous split visible in the assessment.
func AdvisoryFindings(b Budget) []mga.Finding {
	axes := UngatedDeclaredAxes(b)
	if len(axes) == 0 {
		return nil
	}
	return []mga.Finding{{
		Code:     "PERF_BUDGET_AXIS_UNGATED",
		Severity: "info",
		Title:    "Performance budget axis measured continuously, not freshly this run",
		Message: fmt.Sprintf("declared budget axes [%s] are measured continuously out-of-band (capture/sweep → trend), "+
			"not freshly by this synchronous performance gate; a breach is checked against the latest persisted "+
			"sample and fails the Performance phase (suite run), but reflects the last out-of-band capture, not this run",
			strings.Join(axes, ", ")),
		AutofixAvailable: false,
	}}
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
