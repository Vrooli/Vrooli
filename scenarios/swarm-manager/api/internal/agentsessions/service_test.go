package agentsessions

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/identity"
)

func TestServiceCreateSpawnsSessionWithEnvironmentAndActivitySpec(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindMetaOrchestration,
		Title:          "Plan quality gates",
		InitialMessage: "Plan this work.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if session.Status != StatusRunning || session.RunID != "run-1" || session.TaskID != "task-1" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if spawner.spawnReq.Environment[EnvSessionID] != session.ID {
		t.Fatalf("session env id = %q, want %q", spawner.spawnReq.Environment[EnvSessionID], session.ID)
	}
	if spawner.spawnReq.Environment[EnvSpawnSource] != "session/"+session.ID {
		t.Fatalf("spawn source = %q", spawner.spawnReq.Environment[EnvSpawnSource])
	}
	if spawner.spawnSpec.OwnerType != agentactivity.OwnerSession {
		t.Fatalf("owner type = %q, want session", spawner.spawnSpec.OwnerType)
	}
	if spawner.spawnSpec.Metadata["skill_id"] != SkillMetaOrchestrator {
		t.Fatalf("skill metadata = %q", spawner.spawnSpec.Metadata["skill_id"])
	}
	if len(session.Messages) != 1 || session.Messages[0].Content != "Plan this work." {
		t.Fatalf("messages = %+v", session.Messages)
	}
}

func TestServiceContinueAppendsMessageAndUsesTrackedContinuation(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{}
	svc := newTestService(t, spawner)
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindOperatingModeAuthoring,
		Title:          "Author mode",
		InitialMessage: "Draft a mode.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	continued, err := svc.Continue(context.Background(), ContinueRequest{
		SessionID: session.ID,
		Message:   "Continue.",
	})
	if err != nil {
		t.Fatalf("Continue() error = %v", err)
	}
	if spawner.continueRunID != "run-1" || spawner.continueMessage != "Continue." {
		t.Fatalf("continue call = run:%q message:%q", spawner.continueRunID, spawner.continueMessage)
	}
	if spawner.continueSpec.OwnerType != agentactivity.OwnerSession || spawner.continueSpec.Purpose != agentactivity.PurposeOperatingModeAuthoring {
		t.Fatalf("continue spec = %+v", spawner.continueSpec)
	}
	if len(continued.Messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(continued.Messages))
	}
}

func TestServiceResolveSessionForRun(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	svc := newTestService(t, &fakeSessionSpawner{})
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindMetaOrchestration,
		Title:          "Plan quality gates",
		InitialMessage: "Plan this work.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	ref, ok, err := svc.ResolveSessionForRun(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("ResolveSessionForRun() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveSessionForRun() ok = false, want true")
	}
	if ref.SessionID != session.ID || ref.SessionKind != string(KindMetaOrchestration) || ref.Source != "session/"+session.ID {
		t.Fatalf("reference = %+v", ref)
	}

	if _, ok, err := svc.ResolveSessionForRun(context.Background(), "run-missing"); err != nil || ok {
		t.Fatalf("missing resolve = ok:%v err:%v", ok, err)
	}
}

func TestServiceRefreshAndCancelUpdateLifecycle(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "complete", Summary: "Final handoff."}}
	svc := newTestService(t, spawner)
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindMetaOrchestration,
		Title:          "Plan",
		InitialMessage: "Plan.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	refreshed, err := svc.Refresh(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.Status != StatusComplete {
		t.Fatalf("status after refresh = %q, want complete", refreshed.Status)
	}
	if len(refreshed.Messages) != 2 || refreshed.Messages[1].Role != MessageRoleAssistant || refreshed.Messages[1].Content != "Final handoff." {
		t.Fatalf("messages after refresh = %+v", refreshed.Messages)
	}

	refreshedAgain, err := svc.Refresh(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Refresh() second call error = %v", err)
	}
	if len(refreshedAgain.Messages) != 2 {
		t.Fatalf("message count after duplicate summary refresh = %d, want 2", len(refreshedAgain.Messages))
	}

	canceled, err := svc.Cancel(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if canceled.Status != StatusCanceled {
		t.Fatalf("status after cancel = %q, want canceled", canceled.Status)
	}
}

func TestServiceDeleteStopsActiveRunBeforeRemovingSession(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{}
	svc := newTestService(t, spawner)
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindMetaOrchestration,
		Title:          "Plan",
		InitialMessage: "Plan.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(context.Background(), session.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if spawner.stoppedRunID != session.RunID {
		t.Fatalf("stopped run = %q, want %q", spawner.stoppedRunID, session.RunID)
	}
	if _, err := svc.Get(context.Background(), session.ID); err == nil {
		t.Fatal("Get() after delete error = nil, want not found")
	}
}

func TestServiceDeleteStopFailureLeavesSessionStored(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{stopErr: errors.New("stop failed")}
	svc := newTestService(t, spawner)
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindMetaOrchestration,
		Title:          "Plan",
		InitialMessage: "Plan.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(context.Background(), session.ID); err == nil {
		t.Fatal("Delete() error = nil, want stop failure")
	}
	if _, err := svc.Get(context.Background(), session.ID); err != nil {
		t.Fatalf("Get() after failed delete error = %v", err)
	}
}

func TestServiceDeleteTerminalSessionDoesNotStopRun(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{}
	svc := newTestService(t, spawner)
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindMetaOrchestration,
		Title:          "Plan",
		InitialMessage: "Plan.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	session.Status = StatusComplete
	if err := svc.store.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	if err := svc.Delete(context.Background(), session.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if spawner.stoppedRunID != "" {
		t.Fatalf("stopped run = %q, want empty", spawner.stoppedRunID)
	}
}

func TestServiceApplyBacklogBatchImportProposalUsesSessionAttribution(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	applier := &fakeBacklogBatchApplier{}
	svc := newTestService(t, &fakeSessionSpawner{})
	svc.SetBacklogBatchApplier(applier)
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindMetaOrchestration,
		Title:          "Plan",
		InitialMessage: "Plan.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	proposal, err := svc.RecordProposal(context.Background(), session.ID, Proposal{
		ID:          "prop-1",
		Kind:        ProposalBacklogBatchImport,
		Status:      ProposalStatusReady,
		Summary:     "Create implementation work.",
		PayloadJSON: `{"items":[{"name":"session-item","title":"Session Item","kind":"idea"}]}`,
		CreatedAt:   testTimestamp,
		UpdatedAt:   testTimestamp,
	})
	if err != nil {
		t.Fatalf("RecordProposal() error = %v", err)
	}

	applied, artifacts, err := svc.ApplyProposal(context.Background(), session.ID, proposal.ID)
	if err != nil {
		t.Fatalf("ApplyProposal() error = %v", err)
	}
	if applied.Status != StatusWaitingForUser {
		t.Fatalf("session status = %q, want waiting_for_user", applied.Status)
	}
	if len(applied.Proposals) != 1 || applied.Proposals[0].Status != ProposalStatusApplied {
		t.Fatalf("proposal after apply = %+v", applied.Proposals)
	}
	if applier.payloadJSON != proposal.PayloadJSON {
		t.Fatalf("payload = %q, want %q", applier.payloadJSON, proposal.PayloadJSON)
	}
	if applier.provenance.SessionID != session.ID || applier.provenance.SessionKind != string(KindMetaOrchestration) {
		t.Fatalf("provenance = %+v", applier.provenance)
	}
	if applier.provenance.RunID != session.RunID || applier.provenance.Type != identity.TypeAgent {
		t.Fatalf("agent provenance = %+v, session run = %q", applier.provenance, session.RunID)
	}
	if len(artifacts) != 1 || artifacts[0].EntityRef != "idea/session-item" {
		t.Fatalf("artifacts = %+v", artifacts)
	}
}

func TestServiceAttachArtifactsPersistsBatchAtomically(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	svc := newTestService(t, &fakeSessionSpawner{})
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindMetaOrchestration,
		Title:          "Plan",
		InitialMessage: "Plan.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	artifacts, err := svc.AttachArtifacts(context.Background(), []Artifact{
		{
			SessionID:    session.ID,
			ArtifactType: ArtifactBacklogItem,
			Action:       ArtifactActionCreated,
			EntityRef:    "idea/a",
		},
		{
			SessionID:    session.ID,
			ArtifactType: ArtifactInitiative,
			Action:       ArtifactActionCreated,
			EntityRef:    "initiative-a",
		},
	})
	if err != nil {
		t.Fatalf("AttachArtifacts() error = %v", err)
	}
	if len(artifacts) != 2 || artifacts[0].ID == "" || artifacts[1].CreatedAt == "" {
		t.Fatalf("artifacts = %+v", artifacts)
	}
	loaded, err := svc.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(loaded.Artifacts) != 2 {
		t.Fatalf("stored artifact count = %d, want 2", len(loaded.Artifacts))
	}
}

func TestServiceApplyOperatingModeDraftRecordsProposalArtifact(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	svc := newTestService(t, &fakeSessionSpawner{})
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindOperatingModeAuthoring,
		Title:          "Author mode",
		InitialMessage: "Draft a mode.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	proposal, err := svc.RecordProposal(context.Background(), session.ID, Proposal{
		ID:          "prop-draft",
		Kind:        ProposalOperatingModeDraft,
		Status:      ProposalStatusReady,
		Summary:     "Draft phased refactor mode.",
		PayloadJSON: `{"mode_id":"phased-refactor","phases":["plan","execute","review"]}`,
		CreatedAt:   testTimestamp,
		UpdatedAt:   testTimestamp,
	})
	if err != nil {
		t.Fatalf("RecordProposal() error = %v", err)
	}

	applied, artifacts, err := svc.ApplyProposal(context.Background(), session.ID, proposal.ID)
	if err != nil {
		t.Fatalf("ApplyProposal() error = %v", err)
	}
	if applied.Proposals[0].Status != ProposalStatusApplied {
		t.Fatalf("proposal status = %q, want applied", applied.Proposals[0].Status)
	}
	if len(artifacts) != 1 {
		t.Fatalf("artifact count = %d, want 1", len(artifacts))
	}
	artifact := artifacts[0]
	if artifact.ArtifactType != ArtifactOperatingModeProposal || artifact.Action != ArtifactActionProposed {
		t.Fatalf("artifact kind/action = %q/%q", artifact.ArtifactType, artifact.Action)
	}
	if artifact.EntityRef != "phased-refactor" || artifact.ProposalID != proposal.ID {
		t.Fatalf("artifact ref/proposal = %q/%q", artifact.EntityRef, artifact.ProposalID)
	}
	if artifact.Attribution == nil || artifact.Attribution.SessionID != session.ID || artifact.Attribution.SessionKind != KindOperatingModeAuthoring {
		t.Fatalf("artifact attribution = %+v", artifact.Attribution)
	}
}

func TestServiceApplyOperatingModeImplementationPlanUsesBatchApplier(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	applier := &fakeBacklogBatchApplier{}
	svc := newTestService(t, &fakeSessionSpawner{})
	svc.SetBacklogBatchApplier(applier)
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:           KindOperatingModeAuthoring,
		Title:          "Author mode",
		InitialMessage: "Draft a mode.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	proposal, err := svc.RecordProposal(context.Background(), session.ID, Proposal{
		ID:      "prop-plan",
		Kind:    ProposalOperatingModeImplementationPlan,
		Status:  ProposalStatusReady,
		Summary: "Create implementation initiative and items.",
		PayloadJSON: `{
			"mode_id":"phased-refactor",
			"backlog_batch_import":{
				"initiatives":[{"name":"phased-refactor-mode","title":"Phased Refactor Mode"}],
				"items":[{"name":"implement-mode","title":"Implement mode","kind":"feature","initiative":"phased-refactor-mode"}]
			}
		}`,
		CreatedAt: testTimestamp,
		UpdatedAt: testTimestamp,
	})
	if err != nil {
		t.Fatalf("RecordProposal() error = %v", err)
	}

	_, _, err = svc.ApplyProposal(context.Background(), session.ID, proposal.ID)
	if err != nil {
		t.Fatalf("ApplyProposal() error = %v", err)
	}
	var appliedPayload struct {
		Initiatives []struct {
			Name string `json:"name"`
		} `json:"initiatives"`
		Items []struct {
			Name       string `json:"name"`
			Initiative string `json:"initiative"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(applier.payloadJSON), &appliedPayload); err != nil {
		t.Fatalf("applier payload is invalid JSON: %v", err)
	}
	if len(appliedPayload.Initiatives) != 1 || appliedPayload.Initiatives[0].Name != "phased-refactor-mode" {
		t.Fatalf("initiatives payload = %+v", appliedPayload.Initiatives)
	}
	if len(appliedPayload.Items) != 1 || appliedPayload.Items[0].Name != "implement-mode" || appliedPayload.Items[0].Initiative != "phased-refactor-mode" {
		t.Fatalf("items payload = %+v", appliedPayload.Items)
	}
	if applier.provenance.SessionID != session.ID || applier.provenance.SessionKind != string(KindOperatingModeAuthoring) {
		t.Fatalf("provenance = %+v", applier.provenance)
	}
}

func newTestService(t *testing.T, spawner *fakeSessionSpawner) *Service {
	t.Helper()
	svc, err := NewService(ServiceConfig{
		Store:       NewFileStore(t.TempDir()),
		Spawner:     spawner,
		ProjectRoot: "/repo",
		ProfileKey:  "swarm-manager/default",
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

type fakeBacklogBatchApplier struct {
	payloadJSON string
	provenance  identity.Provenance
}

func (f *fakeBacklogBatchApplier) ApplyAgentSessionBacklogBatchImport(_ context.Context, payloadJSON string, prov identity.Provenance) ([]Artifact, error) {
	f.payloadJSON = payloadJSON
	f.provenance = prov
	return []Artifact{{
		ID:           "art-1",
		SessionID:    prov.SessionID,
		ArtifactType: ArtifactBacklogItem,
		Action:       ArtifactActionCreated,
		EntityRef:    "idea/session-item",
		CreatedAt:    testTimestamp,
	}}, nil
}

type fakeSessionSpawner struct {
	spawnReq        agentmanager.SessionSpawnRequest
	spawnSpec       agentactivity.Spec
	continueRunID   string
	continueMessage string
	continueSpec    agentactivity.Spec
	stoppedRunID    string
	stopErr         error
	runState        agentmanager.RunState
}

func (f *fakeSessionSpawner) SpawnSession(ctx context.Context, req agentmanager.SessionSpawnRequest) (agentmanager.RunResult, error) {
	f.spawnReq = req
	f.spawnSpec = mustSpecFromContext(ctx)
	return agentmanager.RunResult{TaskID: "task-1", RunID: "run-1"}, nil
}

func (f *fakeSessionSpawner) ContinueRun(ctx context.Context, runID string, message string) error {
	f.continueRunID = runID
	f.continueMessage = message
	f.continueSpec = mustSpecFromContext(ctx)
	return nil
}

func (f *fakeSessionSpawner) GetRunState(context.Context, string) (agentmanager.RunState, error) {
	if f.runState.Status == "" {
		return agentmanager.RunState{Status: "running"}, nil
	}
	return f.runState, nil
}

func (f *fakeSessionSpawner) StopRun(_ context.Context, runID string) error {
	f.stoppedRunID = runID
	return f.stopErr
}

func mustSpecFromContext(ctx context.Context) agentactivity.Spec {
	spec, err := agentactivity.SpecFromContext(ctx)
	if err != nil {
		panic(err)
	}
	return spec
}

func freezeAgentSessionClock(t *testing.T) func() {
	t.Helper()
	original := nowUTC
	nowUTC = func() time.Time {
		return time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	}
	return func() { nowUTC = original }
}
