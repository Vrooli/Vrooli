package retention

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakePruner records the budget it was handed and returns a scripted result, so
// engine behavior is exercised without a filesystem or a database.
type fakePruner struct {
	before     Usage
	after      Usage
	result     Result
	pruneErr   error
	measureErr error

	gotBudget   Budget
	pruneCalls  int
	measureCall int
}

func (f *fakePruner) Measure(ctx context.Context) (Usage, error) {
	f.measureCall++
	if f.measureErr != nil {
		return Usage{}, f.measureErr
	}
	if f.pruneCalls == 0 {
		return f.before, nil
	}
	return f.after, nil
}

func (f *fakePruner) Prune(ctx context.Context, b Budget) (Result, error) {
	f.pruneCalls++
	f.gotBudget = b
	return f.result, f.pruneErr
}

func specFor(name string, b Budget, mode PrunerMode) Spec {
	b.Name = name
	return Spec{
		Budget: b,
		Target: Target{Kind: TargetDirectory, Path: name},
		Mode:   mode,
	}
}

func TestEngineReportsBoundAgeWhenOnlyAgeIsDeclared(t *testing.T) {
	pruner := &fakePruner{
		before: Usage{Bytes: 1000, Items: 100},
		after:  Usage{Bytes: 400, Items: 40},
		result: Result{Deleted: 60, BoundBy: BoundAge},
	}
	engine := newTestEngine(t, specFor("logs", Budget{MaxAge: 30 * 24 * time.Hour}, PrunerBuiltin), pruner)

	results, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].BoundBy != BoundAge {
		t.Fatalf("BoundBy = %v, want age", results[0].BoundBy)
	}
	if results[0].FreedBytes != 600 {
		t.Errorf("FreedBytes = %d, want 600", results[0].FreedBytes)
	}
}

func TestEngineReportsBoundBytesWhenSizeIsWhatBinds(t *testing.T) {
	// This is the autoheal case and the reason the package exists: the age bound
	// is configured, runs, and frees nothing useful, while the size ceiling is
	// what actually limits what is kept.
	pruner := &fakePruner{
		before: Usage{Bytes: 453 << 30, Items: 846_000_000},
		after:  Usage{Bytes: 2 << 30, Items: 3_000_000},
		result: Result{Deleted: 843_000_000, BoundBy: BoundBytes},
	}
	engine := newTestEngine(t, specFor("system_events", Budget{MaxAge: 30 * 24 * time.Hour, MaxBytes: 2 << 30}, PrunerBuiltin), pruner)

	results, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].BoundBy != BoundBytes {
		t.Fatalf("BoundBy = %v, want bytes; the size ceiling is what did the work", results[0].BoundBy)
	}
}

func TestEngineUpgradesToBoundBytesWhenStillOverCeiling(t *testing.T) {
	// A pruner that deleted everything it was permitted to and is still over the
	// ceiling has not succeeded quietly. The engine must say so, because that
	// result is a finding about the producer.
	pruner := &fakePruner{
		before: Usage{Bytes: 10 << 30},
		after:  Usage{Bytes: 6 << 30},
		result: Result{Deleted: 5, BoundBy: BoundNone, After: Usage{Bytes: 6 << 30}},
	}
	budget := Budget{MaxAge: 24 * time.Hour, MaxBytes: 5 << 30}
	engine := newTestEngine(t, specFor("snapshots", budget, PrunerBuiltin), pruner)

	results, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].BoundBy != BoundBytes {
		t.Fatalf("BoundBy = %v, want bytes when the target is still over its ceiling", results[0].BoundBy)
	}
	if !results[0].OverBudget(budget) {
		t.Fatal("OverBudget = false for a target measured above its ceiling")
	}
}

func TestEngineReportsBoundNoneWhenNothingBinds(t *testing.T) {
	pruner := &fakePruner{
		before: Usage{Bytes: 100},
		after:  Usage{Bytes: 100},
		result: Result{Deleted: 0, BoundBy: BoundNone},
	}
	engine := newTestEngine(t, specFor("small", Budget{MaxAge: 30 * 24 * time.Hour, MaxBytes: 1 << 30}, PrunerBuiltin), pruner)

	results, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].BoundBy != BoundNone {
		t.Fatalf("BoundBy = %v, want none", results[0].BoundBy)
	}
}

func TestEngineBothBoundsAssertedInBothDirections(t *testing.T) {
	budget := Budget{MaxAge: 30 * 24 * time.Hour, MaxBytes: 2 << 30}

	// Age binds: the pruner stopped on the horizon and the result fits under the
	// ceiling.
	ageBound := &fakePruner{
		before: Usage{Bytes: 3 << 30},
		after:  Usage{Bytes: 1 << 30},
		result: Result{Deleted: 10, BoundBy: BoundAge, After: Usage{Bytes: 1 << 30}},
	}
	engine := newTestEngine(t, specFor("b", budget, PrunerBuiltin), ageBound)
	results, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].BoundBy != BoundAge {
		t.Errorf("age direction: BoundBy = %v, want age", results[0].BoundBy)
	}

	// Bytes bind: everything is inside the horizon, so only the ceiling can stop
	// it.
	byteBound := &fakePruner{
		before: Usage{Bytes: 9 << 30},
		after:  Usage{Bytes: 2 << 30},
		result: Result{Deleted: 900, BoundBy: BoundBytes, After: Usage{Bytes: 2 << 30}},
	}
	engine = newTestEngine(t, specFor("b", budget, PrunerBuiltin), byteBound)
	results, err = engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].BoundBy != BoundBytes {
		t.Errorf("byte direction: BoundBy = %v, want bytes", results[0].BoundBy)
	}
}

func TestEngineHandsCustomPrunerTheDeclaredBudget(t *testing.T) {
	pruner := &fakePruner{
		before: Usage{Bytes: 10 << 30},
		after:  Usage{Bytes: 1 << 30},
		result: Result{Deleted: 3, BoundBy: BoundAge, After: Usage{Bytes: 1 << 30}},
	}
	registry := NewRegistry()
	if err := registry.Register("graph_snapshots", pruner); err != nil {
		t.Fatalf("Register: %v", err)
	}
	spec := specFor("graph_snapshots", Budget{MaxBytes: 5 << 30}, PrunerCustom)

	engine, err := NewEngine(EngineConfig{
		Specs: []Spec{spec},
		// No builtin factory: a custom budget must never reach one.
		Registry: registry,
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if pruner.gotBudget.MaxBytes != 5<<30 || pruner.gotBudget.Name != "graph_snapshots" {
		t.Fatalf("custom pruner received %+v, want the declared budget", pruner.gotBudget)
	}
	if pruner.pruneCalls != 1 {
		t.Fatalf("custom pruner ran %d times, want 1", pruner.pruneCalls)
	}
}

func TestEngineFailsLoudlyOnUnregisteredCustomPruner(t *testing.T) {
	// A silent fallback to the builtin pruner would apply a generic age rule to
	// data that has a domain selection rule and delete the wrong items, so this
	// must fail at construction and the builtin factory must never be called.
	builtinCalled := false
	_, err := NewEngine(EngineConfig{
		Specs:    []Spec{specFor("graph_snapshots", Budget{MaxBytes: 5 << 30}, PrunerCustom)},
		Registry: NewRegistry(),
		Builtin: func(Spec) (Pruner, error) {
			builtinCalled = true
			return &fakePruner{}, nil
		},
	})
	if !errors.Is(err, ErrPrunerNotRegistered) {
		t.Fatalf("error = %v, want ErrPrunerNotRegistered", err)
	}
	if builtinCalled {
		t.Fatal("the builtin pruner was constructed for a custom budget; a silent fallback deletes the wrong rows")
	}
}

func TestEngineFailsWhenBuiltinFactoryIsMissing(t *testing.T) {
	_, err := NewEngine(EngineConfig{Specs: []Spec{specFor("b", Budget{MaxBytes: 1}, PrunerBuiltin)}})
	if !errors.Is(err, ErrNoBuiltinFactory) {
		t.Fatalf("error = %v, want ErrNoBuiltinFactory", err)
	}
}

func TestEngineWithNoBudgetsRunsCleanly(t *testing.T) {
	engine, err := NewEngine(EngineConfig{})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	results, err := engine.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 0 || len(engine.Budgets()) != 0 {
		t.Fatalf("expected an engine with no budgets, got %d results and %d budgets", len(results), len(engine.Budgets()))
	}
}

func TestEngineCancelledContextReportsIncomplete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	pruner := &fakePruner{before: Usage{Bytes: 10}}
	engine := newTestEngine(t, specFor("b", Budget{MaxBytes: 1}, PrunerBuiltin), pruner)

	results, err := engine.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(results) != 1 || !results[0].Incomplete {
		t.Fatalf("results = %+v, want one Incomplete result", results)
	}
	if pruner.pruneCalls != 0 {
		t.Fatal("pruned despite a cancelled context")
	}
}

func TestEngineCancelDuringPruneReportsIncomplete(t *testing.T) {
	pruner := &fakePruner{
		before:   Usage{Bytes: 10 << 30},
		result:   Result{Deleted: 42, After: Usage{Bytes: 9 << 30}},
		pruneErr: context.Canceled,
	}
	engine := newTestEngine(t, specFor("b", Budget{MaxBytes: 1 << 30}, PrunerBuiltin), pruner)

	results, err := engine.Run(context.Background())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if !results[0].Incomplete {
		t.Fatal("Incomplete = false after a cancelled prune")
	}
	// Partial progress must survive: a target needing hours to reach budget
	// would never finish if every cancelled cycle were discarded.
	if results[0].Deleted != 42 {
		t.Fatalf("Deleted = %d, want the 42 items the cancelled prune had already removed", results[0].Deleted)
	}
}

func TestEngineObservesEveryCycleResult(t *testing.T) {
	var seen []Result
	var seenSpecs []Spec
	spec := specFor("system_events", Budget{MaxBytes: 1 << 30}, PrunerBuiltin)
	spec.Rationale = "host-driven ingest"

	pruner := &fakePruner{
		before: Usage{Bytes: 5 << 30},
		after:  Usage{Bytes: 3 << 30},
		result: Result{Deleted: 7, BoundBy: BoundNone, After: Usage{Bytes: 3 << 30}},
	}
	engine, err := NewEngine(EngineConfig{
		Specs:   []Spec{spec},
		Builtin: func(Spec) (Pruner, error) { return pruner, nil },
		Observe: func(r Result, s Spec) {
			seen = append(seen, r)
			seenSpecs = append(seenSpecs, s)
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(seen) != 1 || seen[0].Budget != "system_events" {
		t.Fatalf("observer saw %+v, want one result naming the budget", seen)
	}
	// The observer is what raises a finding, and a finding must be able to name
	// the producer.
	if seen[0].BoundBy != BoundBytes {
		t.Errorf("BoundBy = %v, want bytes for a target still above its ceiling", seen[0].BoundBy)
	}
	if seenSpecs[0].Rationale != "host-driven ingest" {
		t.Errorf("observer got rationale %q, want the declared one", seenSpecs[0].Rationale)
	}
}

func TestEngineRunsBudgetsInDeclaredOrder(t *testing.T) {
	var order []string
	specs := []Spec{
		specFor("alpha", Budget{MaxBytes: 1}, PrunerBuiltin),
		specFor("beta", Budget{MaxBytes: 1}, PrunerBuiltin),
		specFor("gamma", Budget{MaxBytes: 1}, PrunerBuiltin),
	}
	engine, err := NewEngine(EngineConfig{
		Specs: specs,
		Builtin: func(s Spec) (Pruner, error) {
			name := s.Budget.Name
			return &recordingPruner{onPrune: func() { order = append(order, name) }}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(order) != 3 || order[0] != "alpha" || order[1] != "beta" || order[2] != "gamma" {
		t.Fatalf("cycle order = %v, want the declared order", order)
	}
}

type recordingPruner struct{ onPrune func() }

func (r *recordingPruner) Measure(context.Context) (Usage, error) { return Usage{}, nil }

func (r *recordingPruner) Prune(context.Context, Budget) (Result, error) {
	r.onPrune()
	return Result{}, nil
}

func TestRegistryRejectsDuplicateAndEmptyRegistration(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register("", &fakePruner{}); err == nil {
		t.Error("expected an empty name to be rejected")
	}
	if err := registry.Register("b", nil); err == nil {
		t.Error("expected a nil pruner to be rejected")
	}
	if err := registry.Register("b", &fakePruner{}); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	// Which of two selection rules enforces a budget must not depend on init
	// order.
	if err := registry.Register("b", &fakePruner{}); err == nil {
		t.Error("expected a duplicate registration to be rejected")
	}
	if names := registry.Names(); len(names) != 1 || names[0] != "b" {
		t.Errorf("Names() = %v, want [b]", names)
	}
}

func TestBoundString(t *testing.T) {
	for bound, want := range map[Bound]string{BoundNone: "none", BoundAge: "age", BoundBytes: "bytes"} {
		if got := bound.String(); got != want {
			t.Errorf("Bound(%d).String() = %q, want %q", int(bound), got, want)
		}
	}
}

func newTestEngine(t *testing.T, spec Spec, pruner Pruner) *Engine {
	t.Helper()
	engine, err := NewEngine(EngineConfig{
		Specs:   []Spec{spec},
		Builtin: func(Spec) (Pruner, error) { return pruner, nil },
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return engine
}
