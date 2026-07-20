package backlog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/testutil"
)

// fakeAttacher records RememberItem calls so tests can assert
// initiative attachment.
type fakeAttacher struct {
	mu    sync.Mutex
	calls []string
	err   error
}

func (a *fakeAttacher) RememberItem(name, ref string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls = append(a.calls, name+"|"+ref)
	return a.err
}

// fakeEvents captures EmitBacklogCreatedFromSource so tests assert
// attribution actor + payload.
type fakeEvents struct {
	mu       sync.Mutex
	calls    []emittedCreate
	archives []emittedArchive
}

type emittedCreate struct {
	entityID, kind, status, initiative, effort string
	priority                                   int
	actorType, actorID                         string
}

type emittedArchive struct {
	entityID, previousStatus, archivedAt string
}

func (e *fakeEvents) EmitBacklogCreatedFromSource(entityID, kind, status string, priority int, initiative, effort, actorType, actorID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls = append(e.calls, emittedCreate{
		entityID:   entityID,
		kind:       kind,
		status:     status,
		priority:   priority,
		initiative: initiative,
		effort:     effort,
		actorType:  actorType,
		actorID:    actorID,
	})
}

// EmitBacklogCreated is here only to satisfy callers that reach for the
// older signature (none in this test, but it keeps the fake interchangeable
// with *eventlog.Emitter).
func (e *fakeEvents) EmitBacklogCreated(entityID, kind, status string, priority int, initiative, effort string) {
	e.EmitBacklogCreatedFromSource(entityID, kind, status, priority, initiative, effort, "user", "")
}

func (e *fakeEvents) EmitBacklogArchived(entityID, previousStatus, archivedAt string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.archives = append(e.archives, emittedArchive{entityID: entityID, previousStatus: previousStatus, archivedAt: archivedAt})
}

type fakeWorkshop struct {
	mu    sync.Mutex
	calls []string
}

func (w *fakeWorkshop) MaybeStartWorkshop(item BacklogItem) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = append(w.calls, string(item.Kind)+"/"+item.Name)
}

type fakeSessionArtifacts struct {
	mu        sync.Mutex
	artifacts []agentsessions.Artifact
	err       error
}

func (r *fakeSessionArtifacts) AttachArtifact(_ context.Context, artifact agentsessions.Artifact) (agentsessions.Artifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return agentsessions.Artifact{}, r.err
	}
	if artifact.ID == "" {
		artifact.ID = "art_test"
	}
	if artifact.CreatedAt == "" {
		artifact.CreatedAt = "2026-04-23T00:00:00Z"
	}
	r.artifacts = append(r.artifacts, artifact)
	return artifact, nil
}

func (r *fakeSessionArtifacts) AttachArtifacts(_ context.Context, artifacts []agentsessions.Artifact) ([]agentsessions.Artifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	for i := range artifacts {
		if artifacts[i].ID == "" {
			artifacts[i].ID = "art_test"
		}
		if artifacts[i].CreatedAt == "" {
			artifacts[i].CreatedAt = "2026-04-23T00:00:00Z"
		}
		r.artifacts = append(r.artifacts, artifacts[i])
	}
	return artifacts, nil
}

// serviceTestEnv bundles a temp-dir-backed FileStore and the four
// optional collaborators so each test can reach into them and assert
// the side-effect set.
type serviceTestEnv struct {
	root      string
	store     *FileStore
	att       *fakeAttacher
	events    *fakeEvents
	workshop  *fakeWorkshop
	inv       *testutil.RecordingScheduler
	artifacts *fakeSessionArtifacts
	svc       *Service
}

func newServiceTestEnv(t *testing.T) *serviceTestEnv {
	t.Helper()
	root := t.TempDir()
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(root, dir))
	}
	store := NewFileStore(root)
	env := &serviceTestEnv{
		root:      root,
		store:     store,
		att:       &fakeAttacher{},
		events:    &fakeEvents{},
		workshop:  &fakeWorkshop{},
		inv:       &testutil.RecordingScheduler{},
		artifacts: &fakeSessionArtifacts{},
	}
	svc, err := NewService(ServiceConfig{
		Store:       store,
		Assigner:    env.att,
		Events:      env.events,
		Artifacts:   env.artifacts,
		Workshop:    env.workshop,
		Invalidator: env.inv,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	env.svc = svc
	return env
}

func TestService_Create_SessionProvenanceRecordsArtifactBeforeEvents(t *testing.T) {
	env := newServiceTestEnv(t)
	item := sampleItem("session-created")
	item.CreatedBy = &identity.Provenance{
		Type:        identity.TypeAgent,
		RunID:       "run-session-1",
		TaskID:      "task-session-1",
		ProfileKey:  "swarm-manager/default",
		SessionID:   "sess_123",
		SessionKind: "meta_orchestration",
		Source:      "session/sess_123",
	}

	if err := env.svc.Create(item, CreationContext{Source: SourceHumanHTTP, Entrypoint: "http.create"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if len(env.artifacts.artifacts) != 1 {
		t.Fatalf("session artifacts = %d, want 1", len(env.artifacts.artifacts))
	}
	got := env.artifacts.artifacts[0]
	if got.SessionID != "sess_123" || got.ArtifactType != agentsessions.ArtifactBacklogItem || got.EntityRef != "execute/session-created" {
		t.Fatalf("unexpected artifact: %+v", got)
	}
	if got.Action != agentsessions.ArtifactActionCreated || got.MutationSource != "http.create" || got.RunID != "run-session-1" {
		t.Fatalf("unexpected artifact attribution fields: %+v", got)
	}
}

func TestService_Create_ArtifactFailureRollsBackItem(t *testing.T) {
	env := newServiceTestEnv(t)
	env.artifacts.err = errors.New("artifact store failed")
	item := sampleItem("artifact-rollback")
	item.CreatedBy = &identity.Provenance{
		Type:      identity.TypeAgent,
		RunID:     "run-session-2",
		SessionID: "sess_rollback",
	}

	err := env.svc.Create(item, CreationContext{Source: SourceHumanHTTP})
	if err == nil || !contains(err.Error(), "record session artifact") {
		t.Fatalf("expected artifact failure, got %v", err)
	}
	if _, statErr := os.Stat(env.store.ItemDir(KindExecute, "artifact-rollback")); !os.IsNotExist(statErr) {
		t.Fatalf("item dir still exists after rollback: %v", statErr)
	}
	if len(env.events.calls) != 0 {
		t.Fatalf("events fired before artifact rollback: %+v", env.events.calls)
	}
}

func sampleItem(name string) BacklogItem {
	return BacklogItem{
		Name:       name,
		Title:      "Title for " + name,
		Kind:       KindExecute,
		Status:     StatusBacklog,
		Priority:   4,
		Effort:     "M",
		Initiative: "demo",
		Created:    "2026-04-23T00:00:00Z",
		Updated:    "2026-04-23T00:00:00Z",
	}
}

func TestService_Create_RejectsMissingSource(t *testing.T) {
	env := newServiceTestEnv(t)
	err := env.svc.Create(sampleItem("missing-source"), CreationContext{})
	if err == nil || !contains(err.Error(), "Source is required") {
		t.Fatalf("expected Source-required error, got %v", err)
	}
}

func TestService_Create_HumanHTTP_TriggersWorkshopAndEmitsUserActor(t *testing.T) {
	env := newServiceTestEnv(t)
	if err := env.svc.Create(sampleItem("alpha"), CreationContext{
		Source: SourceHumanHTTP, DecidedBy: "matt",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if got := len(env.workshop.calls); got != 1 {
		t.Errorf("workshop trigger calls = %d, want 1", got)
	}
	if got := len(env.events.calls); got != 1 {
		t.Fatalf("event emit calls = %d, want 1", got)
	}
	got := env.events.calls[0]
	if got.actorType != "user" || got.actorID != "matt" {
		t.Errorf("actor: got (%q,%q) want (user,matt)", got.actorType, got.actorID)
	}
	if got := len(env.att.calls); got != 1 {
		t.Errorf("RememberItem calls = %d, want 1", got)
	}
	if got := env.inv.Count(); got != 1 {
		t.Errorf("ScheduleAll calls = %d, want 1", got)
	}
}

func TestService_Create_Batch_TriggersWorkshopUnlessSkipped(t *testing.T) {
	env := newServiceTestEnv(t)
	cc := CreationContext{
		Source:                SourceBatch,
		SkipDuplicateCheck:    true,
		SkipCycleCheck:        true,
		SkipInitiativeAttach:  true,
		SkipWorkshopTrigger:   true,
		SkipGraphInvalidation: true,
	}
	if err := env.svc.Create(sampleItem("beta"), cc); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if len(env.workshop.calls) != 0 {
		t.Errorf("SkipWorkshopTrigger ignored: workshop fired %v", env.workshop.calls)
	}
	if got := env.inv.Count(); got != 0 {
		t.Errorf("SkipGraphInvalidation ignored: ScheduleAll fired %d times", got)
	}
	if len(env.att.calls) != 0 {
		t.Errorf("SkipInitiativeAttach ignored: RememberItem fired %v", env.att.calls)
	}
	// Event still fires — attribution must land regardless of side-effect skips.
	if len(env.events.calls) != 1 {
		t.Errorf("event emit calls = %d, want 1", len(env.events.calls))
	}
}

func TestService_Create_Proposal_AttributesToFeedbackRound(t *testing.T) {
	env := newServiceTestEnv(t)
	cc := CreationContext{
		Source:          SourceProposal,
		FeedbackRoundID: "demo/round-001",
		RoundNumber:     1,
		RoundSlug:       "kickoff",
		Entrypoint:      "initiative.feedback",
		DecidedBy:       "matt",
		SkipCycleCheck:  true,
	}
	if err := env.svc.Create(sampleItem("gamma"), cc); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if len(env.workshop.calls) != 0 {
		t.Errorf("proposal source must NOT trigger workshop, got %v", env.workshop.calls)
	}
	if len(env.events.calls) != 1 {
		t.Fatalf("event emit calls = %d", len(env.events.calls))
	}
	got := env.events.calls[0]
	if got.actorType != "feedback_round" || got.actorID != "demo/round-001" {
		t.Errorf("actor: got (%q,%q) want (feedback_round, demo/round-001)", got.actorType, got.actorID)
	}
}

func TestService_Create_Proposal_ReviewRoundDominatesFeedbackRound(t *testing.T) {
	env := newServiceTestEnv(t)
	cc := CreationContext{
		Source:          SourceProposal,
		FeedbackRoundID: "demo/round-001",
		ReviewRoundID:   "demo/review-002",
		SkipCycleCheck:  true,
	}
	if err := env.svc.Create(sampleItem("delta"), cc); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got := env.events.calls[0]
	if got.actorType != "initiative_review" || got.actorID != "demo/review-002" {
		t.Errorf("review must dominate feedback: got (%q,%q)", got.actorType, got.actorID)
	}
}

func TestService_Create_Duplicate_ReturnsErrItemExists(t *testing.T) {
	env := newServiceTestEnv(t)
	if err := env.svc.Create(sampleItem("dup"), CreationContext{Source: SourceHumanHTTP}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	err := env.svc.Create(sampleItem("dup"), CreationContext{Source: SourceHumanHTTP})
	if !errors.Is(err, ErrItemExists) {
		t.Fatalf("second Create should return ErrItemExists, got %v", err)
	}
}

func TestServiceArchiveItem_AppendsReasonAndEmitsArchiveOnce(t *testing.T) {
	env := newServiceTestEnv(t)
	item := sampleItem("resolved")
	item.Kind = KindFix
	item.Status = StatusSuggested
	if err := env.svc.Create(item, CreationContext{Source: SourceAutoFiler}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	archived, err := env.svc.ArchiveItem(context.Background(), KindFix, "resolved", "dismissed by operator")
	if err != nil {
		t.Fatalf("ArchiveItem: %v", err)
	}
	if archived.ArchivedAt == nil || *archived.ArchivedAt == "" {
		t.Fatalf("ArchivedAt was not set")
	}
	if archived.Note != "dismissed by operator" {
		t.Fatalf("note = %q, want reason", archived.Note)
	}
	if len(env.events.archives) != 1 {
		t.Fatalf("archive events = %+v, want one", env.events.archives)
	}

	again, err := env.svc.ArchiveItem(context.Background(), KindFix, "resolved", "dismissed by operator")
	if err != nil {
		t.Fatalf("ArchiveItem second: %v", err)
	}
	if again.Note != "dismissed by operator" {
		t.Fatalf("second note = %q, want unchanged", again.Note)
	}
	if len(env.events.archives) != 1 {
		t.Fatalf("archive events after idempotent call = %+v, want still one", env.events.archives)
	}
}

func TestServiceAnnotateItem_AppendsIdempotentNote(t *testing.T) {
	env := newServiceTestEnv(t)
	item := sampleItem("accepted")
	item.Kind = KindFix
	if err := env.svc.Create(item, CreationContext{Source: SourceAutoFiler}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	first, err := env.svc.AnnotateItem(context.Background(), KindFix, "accepted", "resolved upstream")
	if err != nil {
		t.Fatalf("AnnotateItem: %v", err)
	}
	if first.Note != "resolved upstream" {
		t.Fatalf("note = %q, want annotation", first.Note)
	}
	second, err := env.svc.AnnotateItem(context.Background(), KindFix, "accepted", "resolved upstream")
	if err != nil {
		t.Fatalf("AnnotateItem second: %v", err)
	}
	if second.Note != "resolved upstream" {
		t.Fatalf("second note = %q, want idempotent annotation", second.Note)
	}
}

func TestService_Create_RollbackOnAttachFailure(t *testing.T) {
	env := newServiceTestEnv(t)
	env.att.err = errors.New("boom")
	err := env.svc.Create(sampleItem("rollback"), CreationContext{Source: SourceHumanHTTP})
	if err == nil {
		t.Fatal("expected attach failure to surface")
	}
	dir := env.store.ItemDir(KindExecute, "rollback")
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Errorf("expected item dir %s to be cleaned up after attach failure", dir)
	}
}

func TestService_Create_CycleChecker_RejectsCycle(t *testing.T) {
	env := newServiceTestEnv(t)
	env.svc.cycleChecker = CycleCheckerFunc(func(_ BacklogItem) error {
		return errors.New("dependency cycle: alpha -> beta -> alpha")
	})
	item := sampleItem("cyc")
	item.DependsOn = []string{"execute/other"}
	// Pre-seed dependency target so ValidateDependencies passes.
	otherDir := env.store.ItemDir(KindExecute, "other")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("seed mkdir: %v", err)
	}
	if err := env.store.SaveItem(BacklogItem{
		Name: "other", Kind: KindExecute, Status: StatusBacklog,
		Created: "2026-04-23T00:00:00Z", Updated: "2026-04-23T00:00:00Z",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	err := env.svc.Create(item, CreationContext{Source: SourceHumanHTTP})
	if err == nil || !contains(err.Error(), "dependency cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestService_ResetArtifacts_PlanUnbindRevertsReadyOnly(t *testing.T) {
	env := newServiceTestEnv(t)
	item := sampleItem("reset-lifecycle")
	item.Status = StatusReady
	item.PlanRef = &PlanRef{Provider: "plan-manager", PlanID: "p1", Slug: "p1", Role: PlanRefRoleExecutionSpec}
	if err := env.svc.Create(item, CreationContext{Source: SourceHumanHTTP}); err != nil {
		t.Fatalf("create: %v", err)
	}
	result, err := env.svc.ResetArtifacts(context.Background(), KindExecute, item.Name, []ResetArtifactScope{ResetScopePlanUnbind})
	if err != nil {
		t.Fatalf("ResetArtifacts: %v", err)
	}
	if !result.StatusReverted || result.Item.Status != StatusBacklog || result.Item.PlanRef != nil {
		t.Fatalf("reset result = %#v, want backlog status and no plan ref", result)
	}
}

func TestService_ResetArtifacts_RemovesOnlySelectedArtifactScope(t *testing.T) {
	cases := []struct {
		name    string
		scope   ResetArtifactScope
		removed []string
	}{
		{name: "clarifications", scope: ResetScopeClarifications, removed: []string{"clarifications"}},
		{name: "review", scope: ResetScopeReview, removed: []string{"review"}},
		{name: "handoff and executions", scope: ResetScopeHandoffExecutions, removed: []string{"handoff", "executions"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newServiceTestEnv(t)
			item := sampleItem("scoped-reset")
			if err := env.svc.Create(item, CreationContext{Source: SourceHumanHTTP}); err != nil {
				t.Fatalf("create: %v", err)
			}
			itemDir := env.store.ItemDir(item.Kind, item.Name)
			for _, dir := range []string{"clarifications", "review", "handoff", "executions", "untouched"} {
				if err := os.MkdirAll(filepath.Join(itemDir, dir), 0o755); err != nil {
					t.Fatalf("create %s: %v", dir, err)
				}
			}
			if _, err := env.svc.ResetArtifacts(context.Background(), item.Kind, item.Name, []ResetArtifactScope{tc.scope}); err != nil {
				t.Fatalf("ResetArtifacts: %v", err)
			}
			for _, dir := range tc.removed {
				if _, err := os.Stat(filepath.Join(itemDir, dir)); !os.IsNotExist(err) {
					t.Fatalf("%s exists after %s reset: %v", dir, tc.scope, err)
				}
			}
			if _, err := os.Stat(filepath.Join(itemDir, "untouched")); err != nil {
				t.Fatalf("unrelated artifact removed by %s reset: %v", tc.scope, err)
			}
		})
	}
}

func TestService_RecreateItem_RetargetsDependentsAndArchivesSource(t *testing.T) {
	env := newServiceTestEnv(t)
	source := sampleItem("stale-source")
	source.Initiative = ""
	if err := env.svc.Create(source, CreationContext{Source: SourceHumanHTTP}); err != nil {
		t.Fatalf("create source: %v", err)
	}
	if stored, err := env.store.LoadItem(KindExecute, source.Name); err != nil || stored.Initiative != "" {
		t.Fatalf("source initiative = %q, %v; want empty", stored.Initiative, err)
	}
	dependent := sampleItem("dependent")
	dependent.DependsOn = []string{"execute/stale-source"}
	if err := env.svc.Create(dependent, CreationContext{Source: SourceHumanHTTP}); err != nil {
		t.Fatalf("create dependent: %v", err)
	}
	clone, err := env.svc.RecreateItem(context.Background(), KindExecute, source.Name)
	if err != nil {
		t.Fatalf("RecreateItem: %v", err)
	}
	if clone.SpawnedFrom != "execute/stale-source" || clone.Status != StatusBacklog {
		t.Fatalf("clone = %#v, want backlog clone with lineage", clone)
	}
	archived, err := env.store.LoadItem(KindExecute, source.Name)
	if err != nil || archived.ArchivedAt == nil {
		t.Fatalf("source after recreation = %#v, %v; want archived", archived, err)
	}
	updatedDependent, err := env.store.LoadItem(KindExecute, dependent.Name)
	if err != nil {
		t.Fatalf("load dependent: %v", err)
	}
	if len(updatedDependent.DependsOn) != 1 || updatedDependent.DependsOn[0] != "execute/"+clone.Name {
		t.Fatalf("dependent deps = %v, want clone ref", updatedDependent.DependsOn)
	}
}
