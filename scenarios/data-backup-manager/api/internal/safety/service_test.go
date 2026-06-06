package safety

import (
	"context"
	"errors"
	"testing"

	"data-backup-manager/internal/sources"
)

// --- fakes for the four composed seams ------------------------------------

type fakeDestinations struct {
	list      []DestinationRef
	created   []DestinationRef
	createErr error
	listErr   error
}

func (f *fakeDestinations) List(context.Context) ([]DestinationRef, error) {
	return f.list, f.listErr
}

func (f *fakeDestinations) CreateSafety(_ context.Context, name, location string, _ int64) (DestinationRef, error) {
	if f.createErr != nil {
		return DestinationRef{}, f.createErr
	}
	d := DestinationRef{ID: "dst-safety", Name: name, Location: location, RepositoryLocation: location + "/repositories/" + name + ".kopia"}
	f.created = append(f.created, d)
	// Mirror real persistence so a subsequent List sees the new destination.
	f.list = append(f.list, d)
	return d, nil
}

type fakeTargets struct {
	byOwner     map[string][]TargetRef
	err         error
	registered  []registeredCall
	registerErr error
}

type registeredCall struct {
	owner   string
	name    string
	kind    sources.SourceKind
	locator string
}

func (f *fakeTargets) ListByOwner(_ context.Context, owner string) ([]TargetRef, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byOwner[owner], nil
}

func (f *fakeTargets) Register(_ context.Context, owner, name string, kind sources.SourceKind, locator string) error {
	if f.registerErr != nil {
		return f.registerErr
	}
	f.registered = append(f.registered, registeredCall{owner: owner, name: name, kind: kind, locator: locator})
	return nil
}

type fakeInspector struct {
	facts ScenarioFacts
	err   error
}

func (f *fakeInspector) Inspect(_ context.Context, _ string) (ScenarioFacts, error) {
	return f.facts, f.err
}

type fakePlans struct {
	plans             []PlanRef
	creates           int
	updates           int
	lastCreateTargets []string
	lastUpdateTargets []string
	lastDestIDs       []string
}

func (f *fakePlans) List(context.Context) ([]PlanRef, error) { return f.plans, nil }

func (f *fakePlans) Create(_ context.Context, name string, targetIDs, destinationIDs []string, _ int32) (PlanRef, error) {
	f.creates++
	f.lastCreateTargets = targetIDs
	f.lastDestIDs = destinationIDs
	p := PlanRef{ID: "plan-" + name, Name: name}
	f.plans = append(f.plans, p)
	return p, nil
}

func (f *fakePlans) Update(_ context.Context, id, name string, targetIDs, destinationIDs []string, _ int32) (PlanRef, error) {
	f.updates++
	f.lastUpdateTargets = targetIDs
	f.lastDestIDs = destinationIDs
	return PlanRef{ID: id, Name: name}, nil
}

type fakeRuns struct {
	calls      int
	lastPlanID string
}

func (f *fakeRuns) TriggerManual(_ context.Context, planID string) (RunRef, error) {
	f.calls++
	f.lastPlanID = planID
	return RunRef{ID: "run-1", PlanID: planID, Status: "pending"}, nil
}

func fixedRoot(p string) RuntimeRootFunc { return func() string { return p } }

// --- EnsureSafetyDestination ----------------------------------------------

func TestEnsureSafetyDestination_CreatesThenReuses(t *testing.T) {
	dests := &fakeDestinations{}
	svc := NewService(Deps{
		Destinations: dests,
		RuntimeRoot:  fixedRoot("/home/u/.vrooli"),
	})

	d1, created1, err := svc.EnsureSafetyDestination(context.Background(), 0)
	if err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	if !created1 {
		t.Fatalf("first ensure: want created=true")
	}
	if d1.Name != SafetyDestinationName {
		t.Fatalf("name = %q, want %q", d1.Name, SafetyDestinationName)
	}
	if want := "/home/u/.vrooli/baseline-safety"; d1.Location != want {
		t.Fatalf("location = %q, want %q", d1.Location, want)
	}

	d2, created2, err := svc.EnsureSafetyDestination(context.Background(), 0)
	if err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if created2 {
		t.Fatalf("second ensure: want created=false (idempotent reuse)")
	}
	if d2.ID != d1.ID {
		t.Fatalf("second ensure returned a different destination: %q vs %q", d2.ID, d1.ID)
	}
	if len(dests.created) != 1 {
		t.Fatalf("CreateSafety called %d times, want exactly 1", len(dests.created))
	}
}

func TestEnsureSafetyDestination_NoRuntimeRoot(t *testing.T) {
	svc := NewService(Deps{
		Destinations: &fakeDestinations{},
		RuntimeRoot:  fixedRoot("  "),
	})
	if _, _, err := svc.EnsureSafetyDestination(context.Background(), 0); err == nil {
		t.Fatalf("want error when runtime root is unresolved")
	}
}

// --- BackupScenarioNow -----------------------------------------------------

func TestBackupScenarioNow_NoTargets(t *testing.T) {
	svc := NewService(Deps{
		Destinations: &fakeDestinations{},
		Targets:      &fakeTargets{byOwner: map[string][]TargetRef{}},
		Plans:        &fakePlans{},
		Runs:         &fakeRuns{},
		RuntimeRoot:  fixedRoot("/home/u/.vrooli"),
	})
	_, err := svc.BackupScenarioNow(context.Background(), "foo", 0)
	if !errors.Is(err, ErrNoTargets) {
		t.Fatalf("err = %v, want ErrNoTargets", err)
	}
}

func TestBackupScenarioNow_CreatesEphemeralPlanAndTriggers(t *testing.T) {
	dests := &fakeDestinations{}
	plans := &fakePlans{}
	runs := &fakeRuns{}
	svc := NewService(Deps{
		Destinations: dests,
		Targets: &fakeTargets{byOwner: map[string][]TargetRef{
			"foo": {{ID: "t1", Owner: "foo"}, {ID: "t2", Owner: "foo"}},
		}},
		Plans:       plans,
		Runs:        runs,
		RuntimeRoot: fixedRoot("/home/u/.vrooli"),
	})

	res, err := svc.BackupScenarioNow(context.Background(), "foo", 5)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if res.TargetCount != 2 {
		t.Fatalf("target count = %d, want 2", res.TargetCount)
	}
	if res.DestinationID != "dst-safety" {
		t.Fatalf("destination = %q, want dst-safety", res.DestinationID)
	}
	if plans.creates != 1 || plans.updates != 0 {
		t.Fatalf("plan creates=%d updates=%d, want creates=1 updates=0", plans.creates, plans.updates)
	}
	if got, want := plans.lastCreateTargets, []string{"t1", "t2"}; !equalStrings(got, want) {
		t.Fatalf("ephemeral plan targets = %v, want %v", got, want)
	}
	if got, want := plans.lastDestIDs, []string{"dst-safety"}; !equalStrings(got, want) {
		t.Fatalf("ephemeral plan destinations = %v, want %v", got, want)
	}
	if runs.calls != 1 {
		t.Fatalf("TriggerManual called %d times, want 1", runs.calls)
	}
	if runs.lastPlanID != "plan-baseline-safety-foo" {
		t.Fatalf("triggered plan = %q, want plan-baseline-safety-foo", runs.lastPlanID)
	}
	if dests.created == nil {
		t.Fatalf("backup-now should have ensured the safety destination")
	}
}

func TestBackupScenarioNow_ReusesExistingEphemeralPlan(t *testing.T) {
	plans := &fakePlans{plans: []PlanRef{{ID: "plan-x", Name: "baseline-safety-foo"}}}
	runs := &fakeRuns{}
	svc := NewService(Deps{
		Destinations: &fakeDestinations{list: []DestinationRef{{ID: "dst-safety", Name: SafetyDestinationName}}},
		Targets: &fakeTargets{byOwner: map[string][]TargetRef{
			"foo": {{ID: "t1", Owner: "foo"}, {ID: "t3", Owner: "foo"}},
		}},
		Plans:       plans,
		Runs:        runs,
		RuntimeRoot: fixedRoot("/home/u/.vrooli"),
	})

	if _, err := svc.BackupScenarioNow(context.Background(), "foo", 0); err != nil {
		t.Fatalf("backup: %v", err)
	}
	if plans.creates != 0 || plans.updates != 1 {
		t.Fatalf("plan creates=%d updates=%d, want creates=0 updates=1 (reuse)", plans.creates, plans.updates)
	}
	if got, want := plans.lastUpdateTargets, []string{"t1", "t3"}; !equalStrings(got, want) {
		t.Fatalf("refreshed plan targets = %v, want %v (membership must refresh)", got, want)
	}
	if runs.lastPlanID != "plan-x" {
		t.Fatalf("triggered plan = %q, want plan-x", runs.lastPlanID)
	}
}

// --- RegisterScenarioTargets ----------------------------------------------

func TestRegisterScenarioTargets_PostgresAndDataDir(t *testing.T) {
	targets := &fakeTargets{}
	svc := NewService(Deps{
		Targets:   targets,
		Inspector: &fakeInspector{facts: ScenarioFacts{UsesPostgres: true, DataDir: "/home/u/.vrooli/data/vrooli/alpha", DataDirPresent: true}},
	})

	res, err := svc.RegisterScenarioTargets(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(res.Registered) != 2 {
		t.Fatalf("registered %d targets, want 2: %+v", len(res.Registered), res.Registered)
	}
	if len(targets.registered) != 2 {
		t.Fatalf("Register called %d times, want 2", len(targets.registered))
	}
	// Postgres locator is the universal lifecycle convention.
	pg := targets.registered[0]
	if pg.owner != "alpha" || pg.name != postgresTargetName || pg.kind != sources.KindPostgres || pg.locator != "vrooli_alpha" {
		t.Fatalf("postgres registration = %+v, want owner=alpha name=postgres kind=postgres locator=vrooli_alpha", pg)
	}
	// Filesystem target uses the resolved absolute data dir.
	fsT := targets.registered[1]
	if fsT.kind != sources.KindFilesystem || fsT.locator != "/home/u/.vrooli/data/vrooli/alpha" {
		t.Fatalf("filesystem registration = %+v, want kind=filesystem locator=<data dir>", fsT)
	}
	// Non-derivable kinds are always reported as skipped.
	if !hasSkip(res.Skipped, nonDerivableKinds) {
		t.Fatalf("missing non-derivable skip note: %+v", res.Skipped)
	}
}

func TestRegisterScenarioTargets_NoPostgresNoDataDir(t *testing.T) {
	targets := &fakeTargets{}
	svc := NewService(Deps{
		Targets:   targets,
		Inspector: &fakeInspector{facts: ScenarioFacts{UsesPostgres: false, DataDirPresent: false}},
	})

	res, err := svc.RegisterScenarioTargets(context.Background(), "beta")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(res.Registered) != 0 {
		t.Fatalf("registered %d targets, want 0", len(res.Registered))
	}
	if len(targets.registered) != 0 {
		t.Fatalf("Register called %d times, want 0", len(targets.registered))
	}
	// Both convention kinds + the non-derivable note are reported as skipped.
	if !hasSkip(res.Skipped, string(sources.KindPostgres)) || !hasSkip(res.Skipped, string(sources.KindFilesystem)) {
		t.Fatalf("expected postgres + filesystem skip notes, got %+v", res.Skipped)
	}
}

func TestRegisterScenarioTargets_PostgresOnly(t *testing.T) {
	targets := &fakeTargets{}
	svc := NewService(Deps{
		Targets:   targets,
		Inspector: &fakeInspector{facts: ScenarioFacts{UsesPostgres: true, DataDirPresent: false}},
	})

	res, err := svc.RegisterScenarioTargets(context.Background(), "gamma")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(res.Registered) != 1 || res.Registered[0].Kind != sources.KindPostgres {
		t.Fatalf("registered = %+v, want exactly the postgres target", res.Registered)
	}
	if !hasSkip(res.Skipped, string(sources.KindFilesystem)) {
		t.Fatalf("expected filesystem skip note, got %+v", res.Skipped)
	}
}

func TestRegisterScenarioTargets_BlankScenario(t *testing.T) {
	svc := NewService(Deps{Targets: &fakeTargets{}, Inspector: &fakeInspector{}})
	if _, err := svc.RegisterScenarioTargets(context.Background(), "  "); err == nil {
		t.Fatalf("want error for blank scenario")
	}
}

func TestRegisterScenarioTargets_InspectError(t *testing.T) {
	svc := NewService(Deps{Targets: &fakeTargets{}, Inspector: &fakeInspector{err: errors.New("no repo root")}})
	if _, err := svc.RegisterScenarioTargets(context.Background(), "alpha"); err == nil {
		t.Fatalf("want error when inspection fails")
	}
}

func TestRegisterScenarioTargets_RegisterErrorPropagates(t *testing.T) {
	targets := &fakeTargets{registerErr: errors.New("db down")}
	svc := NewService(Deps{
		Targets:   targets,
		Inspector: &fakeInspector{facts: ScenarioFacts{UsesPostgres: true}},
	})
	if _, err := svc.RegisterScenarioTargets(context.Background(), "alpha"); err == nil {
		t.Fatalf("want error when target registration fails")
	}
}

func hasSkip(notes []SkippedNote, kind string) bool {
	for _, n := range notes {
		if n.Kind == kind {
			return true
		}
	}
	return false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
