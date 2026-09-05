package goals

import (
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eta"
)

type fakeBacklog struct{ items []backlog.BacklogItem }

type fakeDeliverableRanks map[string]int

func (f fakeDeliverableRanks) ReleaseRank(name string) (int, error) { return f[name], nil }

type fakeDeliverableProjection struct {
	ranks   map[string]int
	urgency map[string]int
}

func (f fakeDeliverableProjection) ReleaseRank(name string) (int, error) { return f.ranks[name], nil }
func (f fakeDeliverableProjection) DerivedUrgency(name string) (int, error) {
	return f.urgency[name], nil
}

func (f *fakeBacklog) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return f.items, nil
}

type mutableFakeBacklog struct{ items []backlog.BacklogItem }

func (f *mutableFakeBacklog) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return f.items, nil
}
func (f *mutableFakeBacklog) SaveItem(item backlog.BacklogItem) error {
	for i := range f.items {
		if f.items[i].Kind == item.Kind && f.items[i].Name == item.Name {
			f.items[i] = item
			return nil
		}
	}
	f.items = append(f.items, item)
	return nil
}

func newTestService(t *testing.T, items []backlog.BacklogItem) *Service {
	t.Helper()
	store := NewStore(t.TempDir())
	return NewService(store, &fakeBacklog{items: items})
}

func item(kind, name, status string, tags []string, deps ...string) backlog.BacklogItem {
	return backlog.BacklogItem{
		Name:      name,
		Kind:      backlog.BacklogKind(kind),
		Status:    backlog.BacklogStatus(status),
		Tags:      tags,
		DependsOn: deps,
	}
}

func TestService_CreateComputesScopeAndBaseline(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{
		item("execute", "a", "ready", nil, "execute/b"),
		item("execute", "b", "completed", nil),
	})

	res, err := svc.Create(CreateRequest{Name: "My Goal", Targets: []string{"execute/a"}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Goal.Name != "my-goal" {
		t.Fatalf("name = %q, want my-goal", res.Goal.Name)
	}
	if res.Scope.Total != 2 || res.Scope.CompletedCount != 1 || res.Scope.ProgressPct != 50 {
		t.Fatalf("scope = %+v, want total 2 completed 1 progress 50", res.Scope)
	}
	if len(res.Goal.ScopeHistory) != 1 || res.Goal.ScopeHistory[0].ClosureSize != 2 {
		t.Fatalf("expected baseline snapshot with closure 2, got %+v", res.Goal.ScopeHistory)
	}
	if !svc.store.Exists("my-goal") {
		t.Fatal("goal not persisted")
	}
}

func TestServiceArchiveRequiresDispositionAndForceNamesDroppedItems(t *testing.T) {
	items := &mutableFakeBacklog{items: []backlog.BacklogItem{item("execute", "orphan", "backlog", nil)}}
	svc := NewService(NewStore(t.TempDir()), items)
	if _, err := svc.Create(CreateRequest{Name: "archive-guard", Targets: []string{"execute/orphan"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Archive("archive-guard"); err == nil || !strings.Contains(err.Error(), "execute/orphan") {
		t.Fatalf("archive without disposition = %v", err)
	}
	archived, err := svc.ArchiveWithForce("archive-guard", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(archived.DroppedItems) != 1 || archived.DroppedItems[0] != "execute/orphan" {
		t.Fatalf("dropped items = %v", archived.DroppedItems)
	}
	if items.items[0].Status != backlog.StatusDropped {
		t.Fatalf("item status = %q, want dropped", items.items[0].Status)
	}
}

// A milestone is the only carrier of a goal's definition of done, so the
// domain refuses to persist one without acceptance criteria on either write
// path. This is the rule that also covers the CLI and the legacy batch seam,
// not just typed proposals.
func TestService_MilestoneRequiresAcceptanceCriteria(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "backlog", nil)})
	if _, err := svc.Create(CreateRequest{Name: "release", Targets: []string{"execute/a"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMilestone("release", Milestone{Name: "ship", Title: "Ship"}); err == nil {
		t.Fatal("CreateMilestone accepted a milestone with no acceptance criteria")
	}
	criteria := []string{"Given the release work, when it lands, then the artifact ships."}
	if _, err := svc.CreateMilestone("release", Milestone{Name: "ship", Title: "Ship", AcceptanceCriteria: criteria}); err != nil {
		t.Fatalf("CreateMilestone with criteria: %v", err)
	}
	// UpdateMilestone replaces the definition wholesale, so an update that
	// omits criteria would erase the definition of done.
	if _, err := svc.UpdateMilestone("release", Milestone{Name: "ship", Title: "Ship it"}); err == nil {
		t.Fatal("UpdateMilestone erased the definition of done")
	}
}

// [REQ:SWM-P0-011] Only evidence-verified milestone delivery can close out a goal.
func TestService_CloseOutRequiresVerifiedMilestones(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "completed", nil)})
	if _, err := svc.Create(CreateRequest{Name: "release", Targets: []string{"execute/a"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMilestone("release", Milestone{Name: "ship", Title: "Ship", Items: []string{"execute/a"}, AcceptanceCriteria: []string{"Given the release work, when it lands, then the artifact ships."}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CloseOut("release"); err == nil || !strings.Contains(err.Error(), "ship") {
		t.Fatalf("CloseOut error = %v, want unresolved milestone name", err)
	}
	if _, err := svc.MarkMilestoneDeliveredWithVerdicts("release", "ship", []CriterionVerdict{{Criterion: "Given the release work, when it lands, then the artifact ships.", Verdict: "delivered", Evidence: []string{"evidence-1"}}}); err != nil {
		t.Fatalf("MarkMilestoneDeliveredWithVerdicts: %v", err)
	}
	goal, err := svc.CloseOut("release")
	if err != nil {
		t.Fatalf("CloseOut: %v", err)
	}
	if goal.Status != StatusAchieved {
		t.Fatalf("status = %q, want %q", goal.Status, StatusAchieved)
	}
}

func TestService_MilestoneDeliveryRequiresCriterionCoverage(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.Create(CreateRequest{Name: "criteria", Title: "Criteria"}); err != nil {
		t.Fatal(err)
	}
	criteria := []string{"Given one When reviewed Then proven", "Given two When reviewed Then proven"}
	if _, err := svc.CreateMilestone("criteria", Milestone{Name: "proof", Title: "Proof", AcceptanceCriteria: criteria}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkMilestoneDeliveredWithVerdicts("criteria", "proof", []CriterionVerdict{{Criterion: criteria[0], Verdict: "delivered", Evidence: []string{"e1"}}}); err == nil {
		t.Fatal("expected missing criterion coverage rejection")
	}
	if _, err := svc.MarkMilestoneDeliveredWithVerdicts("criteria", "proof", []CriterionVerdict{{Criterion: criteria[0], Verdict: "delivered", Evidence: []string{"e1"}}, {Criterion: criteria[1], Verdict: "delivered", Evidence: []string{"e2"}}}); err != nil {
		t.Fatalf("expected full coverage: %v", err)
	}
}

func TestBacklogMilestoneAssignerAddsScopeBeforeMembership(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{
		item("execute", "a", "ready", nil),
	})
	if _, err := svc.Create(CreateRequest{Name: "release", Title: "Release"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.CreateMilestone("release", Milestone{Name: "build", Title: "Build", AcceptanceCriteria: []string{"Given the sources, when the build runs, then it produces an artifact."}}); err != nil {
		t.Fatalf("CreateMilestone: %v", err)
	}
	assigner := NewBacklogMilestoneAssigner(svc)
	if err := assigner.RememberItem("release/build", "execute/a"); err != nil {
		t.Fatalf("RememberItem: %v", err)
	}
	got, err := assigner.Get("release/build")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0] != "execute/a" {
		t.Fatalf("milestone items = %#v, want execute/a", got.Items)
	}
	goal, err := svc.Get("release")
	if err != nil {
		t.Fatalf("Get goal: %v", err)
	}
	if len(goal.Goal.Targets) != 1 || goal.Goal.Targets[0] != "execute/a" {
		t.Fatalf("goal targets = %#v, want execute/a", goal.Goal.Targets)
	}
	if err := assigner.RememberItem("build", "execute/a"); err == nil {
		t.Fatal("unqualified milestone reference was accepted")
	}
}

func TestService_ScopeCreepRecordedOnDrift(t *testing.T) {
	fb := &fakeBacklog{items: []backlog.BacklogItem{
		item("execute", "a", "ready", nil),
	}}
	svc := NewService(NewStore(t.TempDir()), fb)

	if _, err := svc.Create(CreateRequest{Name: "g", Targets: []string{"execute/a"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Grow the closure: a now depends on a new item c.
	fb.items = []backlog.BacklogItem{
		item("execute", "a", "ready", nil, "execute/c"),
		item("execute", "c", "backlog", nil),
	}
	res, err := svc.Get("g")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if res.Scope.Total != 2 {
		t.Fatalf("closure should have grown to 2, got %d", res.Scope.Total)
	}
	if n := len(res.Goal.ScopeHistory); n < 2 {
		t.Fatalf("expected a drift snapshot recorded, history len = %d", n)
	}
	last := res.Goal.ScopeHistory[len(res.Goal.ScopeHistory)-1]
	if last.ClosureSize != 2 {
		t.Fatalf("latest snapshot closure = %d, want 2", last.ClosureSize)
	}
}

func TestService_SeedFromTags(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{
		item("execute", "x", "backlog", []string{"monetization-v1"}),
		item("fix", "y", "completed", []string{"monetization-v1"}),
		item("execute", "z", "backlog", []string{"unrelated"}),
	})

	created, err := svc.SeedFromTags([]SeedSpec{
		{Tag: "monetization-v1", Name: "monetization-v1", Title: "Monetization v1", Description: "Monetization capability", Priority: seedPriority(5)},
		{Tag: "no-such-tag", Name: "no-such-tag", Title: "Nope", Description: "No matching tag", Priority: seedPriority(5)},
	})
	if err != nil {
		t.Fatalf("SeedFromTags: %v", err)
	}
	if created != 1 {
		t.Fatalf("created = %d, want 1 (empty tag skipped)", created)
	}
	g, err := svc.Get("monetization-v1")
	if err != nil {
		t.Fatalf("Get seeded: %v", err)
	}
	if !g.Goal.Seeded {
		t.Fatal("seeded goal should carry Seeded=true")
	}
	want := []string{"execute/x", "fix/y"}
	if len(g.Goal.Targets) != 2 || g.Goal.Targets[0] != want[0] || g.Goal.Targets[1] != want[1] {
		t.Fatalf("targets = %v, want %v", g.Goal.Targets, want)
	}

	// Idempotent: re-seeding creates nothing.
	created, err = svc.SeedFromTags([]SeedSpec{{Tag: "monetization-v1", Name: "monetization-v1", Title: "Monetization v1", Description: "Monetization capability", Priority: seedPriority(5)}})
	if err != nil || created != 0 {
		t.Fatalf("re-seed created = %d err = %v, want 0/nil", created, err)
	}
}

func TestService_SeedFromTags_RequiresDescriptionAndExplicitPriority(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "x", "backlog", []string{"seed"})})
	if _, err := svc.SeedFromTags([]SeedSpec{{Tag: "seed", Name: "missing-description", Title: "Missing description", Priority: seedPriority(5)}}); err == nil {
		t.Fatal("expected missing seed description to be rejected")
	}
	if _, err := svc.SeedFromTags([]SeedSpec{{Tag: "seed", Name: "missing-priority", Title: "Missing priority", Description: "Has a description"}}); err == nil {
		t.Fatal("expected missing seed priority to be rejected")
	}
}

// [REQ:SWM-P1-002] goal progress: per-goal ETA band attached
func TestService_AttachesETABand(t *testing.T) {
	items := []backlog.BacklogItem{
		{Name: "a", Kind: "execute", Status: "ready", Effort: "M"},
		{Name: "b", Kind: "execute", Status: "backlog", Effort: "M", DependsOn: []string{"execute/a"}},
	}
	svc := newTestService(t, items)

	// Cold start: no samples → the band rests on priors.
	est := eta.NewEstimator(nil, nil, 2, eta.DefaultTrials, eta.DefaultSeed)
	svc.SetEstimatorFactory(func() (*eta.Estimator, error) { return est, nil })

	if _, err := svc.Create(CreateRequest{Name: "g", Targets: []string{"execute/b"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	res, err := svc.Get("g")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if res.ETA == nil {
		t.Fatal("expected an ETA band attached")
	}
	if res.ETA.RemainingItems != 2 {
		t.Errorf("remaining = %d, want 2 (both pending)", res.ETA.RemainingItems)
	}
	if res.ETA.BasisLabel != "priors only" {
		t.Errorf("basis label = %q, want %q", res.ETA.BasisLabel, "priors only")
	}
	if res.ETA.P50Hours > res.ETA.P80Hours {
		t.Errorf("p50 %v must be <= p80 %v", res.ETA.P50Hours, res.ETA.P80Hours)
	}
	if res.ETA.LaneCapacity != 2 {
		t.Errorf("lane capacity = %d, want 2", res.ETA.LaneCapacity)
	}

	// A confident set of live M samples flips the basis label to a count.
	var samples []eta.Sample
	for _, h := range []float64{44, 46, 48, 50, 52, 48} {
		samples = append(samples, eta.Sample{EffortClass: "M", DurationHours: h, Origin: eta.OriginLive})
	}
	sampled := eta.NewEstimator(samples, nil, 2, eta.DefaultTrials, eta.DefaultSeed)
	svc.SetEstimatorFactory(func() (*eta.Estimator, error) { return sampled, nil })
	res2, err := svc.Get("g")
	if err != nil {
		t.Fatalf("Get (sampled): %v", err)
	}
	if res2.ETA == nil || res2.ETA.BasisLabel == "priors only" {
		t.Fatalf("expected a sample-backed basis label, got %+v", res2.ETA)
	}

	// No factory → no band.
	svc.SetEstimatorFactory(nil)
	res3, err := svc.Get("g")
	if err != nil {
		t.Fatalf("Get (no factory): %v", err)
	}
	if res3.ETA != nil {
		t.Error("expected no ETA band when the factory is unset")
	}
}

func TestService_AddRemoveTargets(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{
		item("execute", "a", "ready", nil),
		item("execute", "b", "ready", nil),
	})
	if _, err := svc.Create(CreateRequest{Name: "g", Targets: []string{"execute/a"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	res, err := svc.AddTargets("g", []string{"execute/b", "execute/a"}) // a is dup
	if err != nil {
		t.Fatalf("AddTargets: %v", err)
	}
	if len(res.Goal.Targets) != 2 {
		t.Fatalf("targets = %v, want 2", res.Goal.Targets)
	}
	res, err = svc.RemoveTargets("g", []string{"execute/a"})
	if err != nil {
		t.Fatalf("RemoveTargets: %v", err)
	}
	if len(res.Goal.Targets) != 1 || res.Goal.Targets[0] != "execute/b" {
		t.Fatalf("targets after remove = %v, want [execute/b]", res.Goal.Targets)
	}
}

func TestService_RejectsLegacyInitiativeTargets(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.Create(CreateRequest{Name: "legacy", Targets: []string{"initiative/old-work"}}); err == nil {
		t.Fatal("expected legacy initiative target to be rejected")
	}
}

// [REQ:SWM-P0-011] goals own scoped milestones over member backlog items
func TestService_MilestonesAreOwnedScopedAndRoundTrip(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{
		item("execute", "a", "ready", nil, "execute/b"),
		item("execute", "b", "completed", nil),
		item("execute", "outside", "ready", nil),
	})
	if _, err := svc.Create(CreateRequest{Name: "g", Targets: []string{"execute/a"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMilestone("g", Milestone{Name: "build", Title: "Build", AcceptanceCriteria: []string{"Given the sources, when the build runs, then it produces an artifact."}}); err != nil {
		t.Fatalf("CreateMilestone: %v", err)
	}
	if _, err := svc.CreateMilestone("g", Milestone{Name: "verify", Title: "Verify", DependsOn: []string{"build"}, AcceptanceCriteria: []string{"Given a built artifact, when it is verified, then the checks pass."}}); err != nil {
		t.Fatalf("CreateMilestone dependent: %v", err)
	}
	if _, err := svc.AssignMilestoneItems("g", "build", []string{"execute/a"}); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if _, err := svc.AssignMilestoneItems("g", "verify", []string{"execute/b"}); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if _, err := svc.AssignMilestoneItems("g", "build", []string{"execute/outside"}); err == nil {
		t.Fatal("expected out-of-scope assignment to fail")
	}
	got, err := svc.Get("g")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Goal.Milestones) != 2 || len(got.Scope.Milestones) != 2 {
		t.Fatalf("milestones = %+v scope = %+v", got.Goal.Milestones, got.Scope.Milestones)
	}
	if len(got.Scope.Unassigned) != 0 {
		t.Fatalf("unassigned = %v, want none", got.Scope.Unassigned)
	}
	if _, err := svc.ArchiveMilestone("g", "verify"); err != nil {
		t.Fatalf("ArchiveMilestone: %v", err)
	}
	archived, err := svc.Get("g")
	if err != nil || archived.Goal.Milestones[1].ArchivedAt == nil {
		t.Fatalf("archived milestone = %+v err=%v", archived.Goal.Milestones[1], err)
	}
}

func TestService_ClosureRefsReturnsClosureWithoutDrift(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{
		item("execute", "a", "ready", nil, "execute/b"),
		item("execute", "b", "ready", nil),
		item("execute", "c", "ready", nil), // unrelated, must not appear
	})
	if _, err := svc.Create(CreateRequest{Name: "goal-x", Targets: []string{"execute/a"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Baseline snapshot count after Create; ClosureRefs must not append more.
	g0, _ := svc.store.Load("goal-x")
	before := len(g0.ScopeHistory)

	refs, err := svc.ClosureRefs("goal-x")
	if err != nil {
		t.Fatalf("ClosureRefs: %v", err)
	}
	got := map[string]bool{}
	for _, r := range refs {
		got[r] = true
	}
	if !got["execute/a"] || !got["execute/b"] || got["execute/c"] {
		t.Fatalf("closure = %v, want {execute/a, execute/b}", refs)
	}
	g1, _ := svc.store.Load("goal-x")
	if len(g1.ScopeHistory) != before {
		t.Fatalf("ClosureRefs recorded drift: history %d -> %d", before, len(g1.ScopeHistory))
	}
	if _, err := svc.ClosureRefs("missing"); err == nil {
		t.Fatal("expected error for unknown goal")
	}
}

func TestService_ItemGoalPrioritiesAndReadyItems(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{
		item("execute", "a", "ready", nil),              // ready, in high-priority goal
		item("execute", "b", "ready", nil, "execute/a"), // blocked by a -> not ready
		item("execute", "c", "ready", nil),              // ready, in low-priority goal
	})
	if _, err := svc.Create(CreateRequest{Name: "high", Priority: 1, Targets: []string{"execute/b"}}); err != nil {
		t.Fatalf("create high: %v", err)
	}
	if _, err := svc.Create(CreateRequest{Name: "low", Priority: 8, Targets: []string{"execute/c"}}); err != nil {
		t.Fatalf("create low: %v", err)
	}

	prio, err := svc.ItemGoalPriorities()
	if err != nil {
		t.Fatalf("ItemGoalPriorities: %v", err)
	}
	// a and b are in the "high" goal's closure (b depends on a); c in "low".
	if prio["execute/a"] != 1 || prio["execute/b"] != 1 || prio["execute/c"] != 8 {
		t.Fatalf("priorities = %v, want a=1 b=1 c=8", prio)
	}

	ready, err := svc.ReadyGoalItems()
	if err != nil {
		t.Fatalf("ReadyGoalItems: %v", err)
	}
	// Only a (high) and c (low) are ready; lowest number sorts first.
	if len(ready) != 2 {
		t.Fatalf("ready = %v, want 2 items", ready)
	}
	if ready[0].Name != "a" || ready[0].GoalPriority != 1 {
		t.Fatalf("first ready = %+v, want a@1", ready[0])
	}
	if ready[1].Name != "c" || ready[1].GoalPriority != 8 {
		t.Fatalf("second ready = %+v, want c@8", ready[1])
	}
}

func TestService_ReleaseRankDerivesPriorityWithoutMutatingAuthoredPriority(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "ready", nil)})
	svc.SetDeliverableRankReader(fakeDeliverableRanks{"web-console": 2})
	created, err := svc.Create(CreateRequest{Name: "release-goal", Priority: 9, ServesDeliverable: "web-console", Targets: []string{"execute/a"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	priorities, err := svc.ItemGoalPriorities()
	if err != nil {
		t.Fatalf("priorities: %v", err)
	}
	if priorities["execute/a"] != 2 {
		t.Fatalf("derived priority = %d, want 2", priorities["execute/a"])
	}
	loaded, err := svc.store.Load(created.Goal.Name)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Priority != 9 || loaded.ServesDeliverable != "web-console" {
		t.Fatalf("stored goal = %+v; authored priority must remain 9", loaded)
	}
}

func TestService_EnablingUrgencyDerivesPriorityWithoutMutatingAuthoredPriority(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "ready", nil)})
	svc.SetDeliverableRankReader(fakeDeliverableProjection{ranks: map[string]int{"scenario-to-desktop": 0}, urgency: map[string]int{"scenario-to-desktop": 1}})
	created, err := svc.Create(CreateRequest{Name: "desktop-goal", Priority: 9, ServesDeliverable: "scenario-to-desktop", Targets: []string{"execute/a"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	priorities, err := svc.ItemGoalPriorities()
	if err != nil {
		t.Fatalf("priorities: %v", err)
	}
	if priorities["execute/a"] != 1 {
		t.Fatalf("derived enabling priority = %d, want 1", priorities["execute/a"])
	}
	loaded, err := svc.store.Load(created.Goal.Name)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Priority != 9 || loaded.ServesDeliverable != "scenario-to-desktop" {
		t.Fatalf("stored goal = %+v; authored priority must remain 9", loaded)
	}
}

func TestService_ZeroDerivedUrgencyIsResolvedWithoutWarningFallback(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "ready", nil)})
	svc.SetDeliverableRankReader(fakeDeliverableProjection{ranks: map[string]int{"money-ledger": 0}, urgency: map[string]int{"money-ledger": 0}})
	created, err := svc.Create(CreateRequest{Name: "ledger-goal", Priority: 9, ServesDeliverable: "money-ledger", Targets: []string{"execute/a"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	priorities, err := svc.ItemGoalPriorities()
	if err != nil {
		t.Fatalf("priorities: %v", err)
	}
	if priorities["execute/a"] != 0 {
		t.Fatalf("derived priority = %d, want 0", priorities["execute/a"])
	}
	loaded, err := svc.store.Load(created.Goal.Name)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Priority != 9 {
		t.Fatalf("stored priority = %d, want authored 9", loaded.Priority)
	}
}
