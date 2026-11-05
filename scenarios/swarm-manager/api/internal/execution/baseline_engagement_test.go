package execution

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
)

// fakeEngagementRunner is a test double for BaselineEngagementRunner. It records
// every Start/Promote/Abandon and lets a test inject per-scenario errors and
// decided modes.
type fakeEngagementRunner struct {
	mu sync.Mutex

	startMode  map[string]string // scenario -> mode returned by Start (default = requested)
	startErr   map[string]error  // scenario -> error returned by Start
	promoteErr map[string]error  // scenario -> error returned by Promote
	abandonErr map[string]error  // scenario -> error returned by Abandon

	started     []string
	promoted    []string
	abandoned   []string
	excludeRuns []string // ExcludeRun seen on each Promote
}

func newFakeEngagementRunner() *fakeEngagementRunner {
	return &fakeEngagementRunner{
		startMode:  map[string]string{},
		startErr:   map[string]error{},
		promoteErr: map[string]error{},
		abandonErr: map[string]error{},
	}
}

func (f *fakeEngagementRunner) Start(_ context.Context, scenario, mode string) (BaselineEngagement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, scenario)
	if err := f.startErr[scenario]; err != nil {
		return BaselineEngagement{}, err
	}
	m := f.startMode[scenario]
	if m == "" {
		m = mode
	}
	return BaselineEngagement{Scenario: scenario, Mode: m}, nil
}

func (f *fakeEngagementRunner) Promote(_ context.Context, p BaselinePromoteParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.promoted = append(f.promoted, p.Scenario)
	f.excludeRuns = append(f.excludeRuns, p.ExcludeRun)
	return f.promoteErr[p.Scenario]
}

func (f *fakeEngagementRunner) Abandon(_ context.Context, scenario string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.abandoned = append(f.abandoned, scenario)
	return f.abandonErr[scenario]
}

func (f *fakeEngagementRunner) snapshot(get func(*fakeEngagementRunner) []string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), get(f)...)
}

// fakeRunDiffer returns a configured diff for the pre-merge hold tests.
type fakeRunDiffer struct {
	diff agentmanager.RunDiff
	err  error
}

func (f *fakeRunDiffer) GetRunDiff(_ context.Context, _ string) (agentmanager.RunDiff, error) {
	return f.diff, f.err
}

// fakeRunApprover records approved run IDs and can inject an error.
type fakeRunApprover struct {
	mu       sync.Mutex
	approved []string
	err      error
}

func (f *fakeRunApprover) ApproveRun(_ context.Context, runID, _, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.approved = append(f.approved, runID)
	return nil
}

func (f *fakeRunApprover) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.approved)
}

func diffForScenarios(scenarios ...string) agentmanager.RunDiff {
	files := make([]agentmanager.RunDiffFile, 0, len(scenarios))
	for _, s := range scenarios {
		files = append(files, agentmanager.RunDiffFile{Path: "scenarios/" + s + "/api/main.go", ChangeType: "modified"})
	}
	return agentmanager.RunDiff{SandboxID: "sbox-1", Files: files}
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func TestParseEngagementStartJSON(t *testing.T) {
	t.Run("variant field wins", func(t *testing.T) {
		eng, err := parseEngagementStartJSON("alpha", `{"scenario":"alpha","variant":"shadow","ambientVar":"","decision":{"mode":"live","reflexive":true}}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eng.Mode != "shadow" {
			t.Errorf("mode = %q, want shadow (variant field is authoritative)", eng.Mode)
		}
		if !eng.Reflexive {
			t.Errorf("reflexive should pass through")
		}
	})

	t.Run("falls back to decision.mode when variant empty", func(t *testing.T) {
		eng, err := parseEngagementStartJSON("alpha", `{"scenario":"alpha","decision":{"mode":"live"}}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eng.Mode != "live" {
			t.Errorf("mode = %q, want live", eng.Mode)
		}
	})

	t.Run("malformed JSON errors", func(t *testing.T) {
		if _, err := parseEngagementStartJSON("alpha", "not json"); err == nil {
			t.Fatal("expected an error for malformed JSON")
		}
	})
}

func TestEngagementStore_AddHolderRemove(t *testing.T) {
	store := NewEngagementStore(filepath.Join(t.TempDir(), "eng.json"))

	if _, _, ok, _ := store.HolderOf("alpha"); ok {
		t.Fatal("nothing engaged yet")
	}

	if _, err := store.Add("execute/item-a", map[string]string{"alpha": "shadow"}, nowRFC3339()); err != nil {
		t.Fatalf("add: %v", err)
	}
	// Expand the same owner's set.
	set, err := store.Add("execute/item-a", map[string]string{"beta": "live"}, nowRFC3339())
	if err != nil {
		t.Fatalf("add 2: %v", err)
	}
	if len(set.Engagements) != 2 {
		t.Fatalf("expected 2 engagements, got %v", set.Engagements)
	}

	owner, mode, ok, _ := store.HolderOf("alpha")
	if !ok || owner != "execute/item-a" || mode != "shadow" {
		t.Errorf("HolderOf(alpha) = %q/%q/%v", owner, mode, ok)
	}

	popped, ok, err := store.Remove("execute/item-a")
	if err != nil || !ok {
		t.Fatalf("remove: ok=%v err=%v", ok, err)
	}
	if len(popped.Engagements) != 2 {
		t.Errorf("removed set should carry both engagements, got %v", popped.Engagements)
	}
	if _, _, ok, _ := store.HolderOf("beta"); ok {
		t.Error("engagements should be gone after remove")
	}
}

func TestOpenEngagementsForOwner_SkipsSelfAndRequestsShadow(t *testing.T) {
	runner := newFakeEngagementRunner()
	runner.startMode["legacy-writer"] = "live" // GCT downgrades a non-shadow-eligible scenario
	svc := &Service{selfScenarioName: "swarm-manager", baselineEngagementRunner: runner}

	got, err := svc.openEngagementsForOwner(context.Background(),
		[]string{"swarm-manager", "git-control-tower", "legacy-writer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["swarm-manager"]; ok {
		t.Errorf("self must be skipped, got %v", got)
	}
	if got["git-control-tower"] != "shadow" {
		t.Errorf("git-control-tower mode = %q, want shadow", got["git-control-tower"])
	}
	if got["legacy-writer"] != "live" {
		t.Errorf("legacy-writer should record GCT's live downgrade, got %q", got["legacy-writer"])
	}
}

// A start ERROR must abort (not silently proceed) so the caller never approves
// a merge without isolation.
func TestOpenEngagementsForOwner_StartErrorAborts(t *testing.T) {
	runner := newFakeEngagementRunner()
	runner.startErr["broken"] = errors.New("baseline start blew up")
	svc := &Service{baselineEngagementRunner: runner}

	if _, err := svc.openEngagementsForOwner(context.Background(), []string{"broken"}); err == nil {
		t.Fatal("expected a start error to propagate")
	}
}

func TestProcessEngagementHold_OpensShadowThenApproves(t *testing.T) {
	root := t.TempDir()
	runner := newFakeEngagementRunner()
	approver := &fakeRunApprover{}
	store := NewStore(filepath.Join(root, "exec.json"))
	if err := store.Save([]Record{{
		ExecutionID: "exec-1", BacklogKind: "execute", BacklogName: "item-a",
		RunID: "run-1", Status: StatusNeedsReview, CreatedAt: nowRFC3339(),
	}}); err != nil {
		t.Fatal(err)
	}
	fc := DefaultFinalizationConfig()
	fc.BaselineEngagementEnabled = true
	svc := &Service{
		selfScenarioName:         "swarm-manager",
		finalizationCfg:          fc,
		store:                    store,
		baselineEngagementRunner: runner,
		approver:                 approver,
		differ:                   &fakeRunDiffer{diff: diffForScenarios("git-control-tower", "audio-tools")},
		engagementStore:          NewEngagementStore(filepath.Join(root, "eng.json")),
		processingHolds:          map[string]struct{}{},
	}

	if err := svc.processEngagementHold(context.Background(), "exec-1"); err != nil {
		t.Fatalf("processEngagementHold: %v", err)
	}

	if got := runner.snapshot(func(f *fakeEngagementRunner) []string { return f.started }); len(got) != 2 {
		t.Fatalf("expected 2 shadow starts, got %v", got)
	}
	if approver.count() != 1 {
		t.Fatalf("expected the merge to be approved once, got %d", approver.count())
	}
	// Owner set persisted.
	set, ok, _ := svc.engagementStore.Get("execute/item-a")
	if !ok || len(set.Engagements) != 2 {
		t.Fatalf("expected 2 engagements under the owner, got %+v", set)
	}
	// Idempotency marker set.
	records, _ := store.Load()
	if records[0].EngagementHoldAt == "" {
		t.Error("EngagementHoldAt should be set after a processed hold")
	}

	// Re-running is a no-op (idempotent).
	if err := svc.processEngagementHold(context.Background(), "exec-1"); err != nil {
		t.Fatalf("second hold: %v", err)
	}
	if approver.count() != 1 {
		t.Errorf("approve must not fire twice, got %d", approver.count())
	}
}

// A merge must NOT be approved if opening a restore point fails.
func TestProcessEngagementHold_NoApproveWhenStartFails(t *testing.T) {
	root := t.TempDir()
	runner := newFakeEngagementRunner()
	runner.startErr["git-control-tower"] = errors.New("gct down")
	approver := &fakeRunApprover{}
	store := NewStore(filepath.Join(root, "exec.json"))
	_ = store.Save([]Record{{
		ExecutionID: "exec-1", BacklogKind: "execute", BacklogName: "item-a",
		RunID: "run-1", Status: StatusNeedsReview, CreatedAt: nowRFC3339(),
	}})
	fc := DefaultFinalizationConfig()
	fc.BaselineEngagementEnabled = true
	svc := &Service{
		finalizationCfg:          fc,
		store:                    store,
		baselineEngagementRunner: runner,
		approver:                 approver,
		differ:                   &fakeRunDiffer{diff: diffForScenarios("git-control-tower")},
		engagementStore:          NewEngagementStore(filepath.Join(root, "eng.json")),
		processingHolds:          map[string]struct{}{},
	}

	if err := svc.processEngagementHold(context.Background(), "exec-1"); err == nil {
		t.Fatal("expected an error when a restore point fails to open")
	}
	if approver.count() != 0 {
		t.Errorf("merge must NOT be approved without isolation, approved=%d", approver.count())
	}
	records, _ := store.Load()
	if records[0].EngagementHoldAt != "" {
		t.Error("hold must not be marked done when it failed")
	}
}

// A scenario already engaged under a DIFFERENT owner blocks the merge.
func TestProcessEngagementHold_DiffLevelExclusivityConflict(t *testing.T) {
	root := t.TempDir()
	runner := newFakeEngagementRunner()
	approver := &fakeRunApprover{}
	store := NewStore(filepath.Join(root, "exec.json"))
	_ = store.Save([]Record{{
		ExecutionID: "exec-1", BacklogKind: "execute", BacklogName: "item-a",
		RunID: "run-1", Status: StatusNeedsReview, CreatedAt: nowRFC3339(),
	}})
	engStore := NewEngagementStore(filepath.Join(root, "eng.json"))
	// Another owner already holds git-control-tower.
	_, _ = engStore.Add("execute/other-item", map[string]string{"git-control-tower": "shadow"}, nowRFC3339())
	fc := DefaultFinalizationConfig()
	fc.BaselineEngagementEnabled = true
	svc := &Service{
		finalizationCfg:          fc,
		store:                    store,
		baselineEngagementRunner: runner,
		approver:                 approver,
		differ:                   &fakeRunDiffer{diff: diffForScenarios("git-control-tower")},
		engagementStore:          engStore,
		processingHolds:          map[string]struct{}{},
	}

	if err := svc.processEngagementHold(context.Background(), "exec-1"); err == nil {
		t.Fatal("expected a conflict error")
	}
	if approver.count() != 0 {
		t.Error("a conflicting hold must not approve the merge")
	}
}

func TestCheckExclusivityAtStart(t *testing.T) {
	root := t.TempDir()
	engStore := NewEngagementStore(filepath.Join(root, "eng.json"))
	_, _ = engStore.Add("execute/owner-x", map[string]string{"git-control-tower": "shadow"}, nowRFC3339())
	fc := DefaultFinalizationConfig()
	fc.BaselineEngagementEnabled = true
	svc := &Service{
		finalizationCfg:          fc,
		baselineEngagementRunner: newFakeEngagementRunner(),
		approver:                 &fakeRunApprover{},
		differ:                   &fakeRunDiffer{},
		engagementStore:          engStore,
	}

	conflictItem := backlogItem{Kind: "execute", Name: "owner-y", AcceptanceAllow: []string{"scenarios/git-control-tower/**"}}
	if err := svc.checkExclusivityAtStart(conflictItem, "execute/owner-y"); err == nil {
		t.Fatal("expected a block-at-start conflict for a scenario held by another owner")
	}

	// Same owner re-starting (a fixup) is fine.
	sameOwner := backlogItem{Kind: "execute", Name: "owner-x", AcceptanceAllow: []string{"scenarios/git-control-tower/**"}}
	if err := svc.checkExclusivityAtStart(sameOwner, "execute/owner-x"); err != nil {
		t.Errorf("same owner must not conflict with itself: %v", err)
	}

	// A free scenario is fine.
	freeItem := backlogItem{Kind: "execute", Name: "owner-z", AcceptanceAllow: []string{"scenarios/audio-tools/**"}}
	if err := svc.checkExclusivityAtStart(freeItem, "execute/owner-z"); err != nil {
		t.Errorf("a free scenario must not conflict: %v", err)
	}
}

func TestCloseOwnerEngagements_PromoteAbandonLeave(t *testing.T) {
	root := t.TempDir()

	newSvc := func(engStore *EngagementStore, runner BaselineEngagementRunner) *Service {
		return &Service{
			store:                    NewStore(filepath.Join(root, "exec.json")),
			baselineEngagementRunner: runner,
			engagementStore:          engStore,
		}
	}

	t.Run("accept promotes the whole set", func(t *testing.T) {
		runner := newFakeEngagementRunner()
		engStore := NewEngagementStore(filepath.Join(t.TempDir(), "eng.json"))
		_, _ = engStore.Add("execute/item-a", map[string]string{"alpha": "shadow", "beta": "shadow"}, nowRFC3339())
		svc := newSvc(engStore, runner)

		svc.CloseOwnerEngagements(context.Background(), "execute", "item-a", EngagementPromote)
		waitUntil(t, func() bool {
			return len(runner.snapshot(func(f *fakeEngagementRunner) []string { return f.promoted })) == 2
		})
		if _, ok, _ := engStore.Get("execute/item-a"); ok {
			t.Error("set should be removed after promote")
		}
	})

	t.Run("reject abandons the whole set", func(t *testing.T) {
		runner := newFakeEngagementRunner()
		engStore := NewEngagementStore(filepath.Join(t.TempDir(), "eng.json"))
		_, _ = engStore.Add("execute/item-b", map[string]string{"alpha": "shadow"}, nowRFC3339())
		svc := newSvc(engStore, runner)

		svc.CloseOwnerEngagements(context.Background(), "execute", "item-b", EngagementAbandon)
		waitUntil(t, func() bool {
			return len(runner.snapshot(func(f *fakeEngagementRunner) []string { return f.abandoned })) == 1
		})
	})

	t.Run("followup leaves the set open", func(t *testing.T) {
		runner := newFakeEngagementRunner()
		engStore := NewEngagementStore(filepath.Join(t.TempDir(), "eng.json"))
		_, _ = engStore.Add("execute/item-c", map[string]string{"alpha": "shadow"}, nowRFC3339())
		svc := newSvc(engStore, runner)

		svc.CloseOwnerEngagements(context.Background(), "execute", "item-c", EngagementLeaveOpen)
		// Give any (incorrect) goroutine a chance to fire.
		time.Sleep(50 * time.Millisecond)
		if got := runner.snapshot(func(f *fakeEngagementRunner) []string { return f.promoted }); len(got) != 0 {
			t.Errorf("followup must not promote, got %v", got)
		}
		if _, ok, _ := engStore.Get("execute/item-c"); !ok {
			t.Error("followup must leave the set open")
		}
	})
}

func TestShadowTargetFor(t *testing.T) {
	engStore := NewEngagementStore(filepath.Join(t.TempDir(), "eng.json"))
	_, _ = engStore.Add("execute/item-a", map[string]string{
		"git-control-tower": "shadow", // shadow-engaged → @shadow
		"legacy-writer":     "live",   // live downgrade → bare name
	}, nowRFC3339())
	svc := &Service{engagementStore: engStore}
	rec := Record{BacklogKind: "execute", BacklogName: "item-a"}

	if got := svc.shadowTargetFor(rec, "git-control-tower"); got != "git-control-tower@shadow" {
		t.Errorf("shadow-engaged target = %q, want git-control-tower@shadow", got)
	}
	if got := svc.shadowTargetFor(rec, "legacy-writer"); got != "legacy-writer" {
		t.Errorf("live-downgrade target = %q, want bare name", got)
	}
	if got := svc.shadowTargetFor(rec, "unrelated"); got != "unrelated" {
		t.Errorf("unengaged target = %q, want bare name", got)
	}
	// No owner set at all → bare name.
	other := Record{BacklogKind: "execute", BacklogName: "item-z"}
	if got := svc.shadowTargetFor(other, "git-control-tower"); got != "git-control-tower" {
		t.Errorf("no-owner target = %q, want bare name", got)
	}
}

func TestScenariosFromRunDiff(t *testing.T) {
	diff := agentmanager.RunDiff{Files: []agentmanager.RunDiffFile{
		{Path: "scenarios/alpha/api/main.go"},
		{Path: "scenarios/alpha/api/other.go"},
		{Path: "scenarios/beta/cli/run.go"},
		{Path: "packages/shared/util.go"}, // shared — excluded
	}}
	got := scenariosFromRunDiff(diff)
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("expected [alpha beta] (shared paths excluded), got %v", got)
	}
}
