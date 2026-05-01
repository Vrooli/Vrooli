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
	mu    sync.Mutex
	calls []emittedCreate
}

type emittedCreate struct {
	entityID, kind, status, initiative, effort string
	priority                                   int
	actorType, actorID                         string
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

type fakeWorkshop struct {
	mu    sync.Mutex
	calls []string
}

func (w *fakeWorkshop) MaybeStartWorkshop(item BacklogItem) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = append(w.calls, string(item.Kind)+"/"+item.Name)
}

type fakeInvalidator struct {
	mu    sync.Mutex
	calls int
}

func (i *fakeInvalidator) ScheduleAll() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.calls++
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
	inv       *fakeInvalidator
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
		inv:       &fakeInvalidator{},
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
	if env.inv.calls != 1 {
		t.Errorf("ScheduleAll calls = %d, want 1", env.inv.calls)
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
	if env.inv.calls != 0 {
		t.Errorf("SkipGraphInvalidation ignored: ScheduleAll fired %d times", env.inv.calls)
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
