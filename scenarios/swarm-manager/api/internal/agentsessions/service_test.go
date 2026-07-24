package agentsessions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/identity"

	agentdomainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestLegacyArtifactsRemainReadableFromSessionStorage(t *testing.T) {
	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	session, err := svc.Create(context.Background(), CreateRequest{Kind: KindMetaOrchestration, Title: "Evidence migration"})
	if err != nil {
		t.Fatal(err)
	}
	writeLegacyArtifacts(t, svc.store.(*FileStore), Artifact{ID: "art-legacy", SessionID: session.ID, ArtifactType: ArtifactMilestone, Action: ArtifactActionCreated, EntityRef: "milestone/evidence", Title: "Evidence", CreatedAt: testTimestamp})
	artifacts, err := svc.ListArtifacts(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].ID != "art-legacy" || artifacts[0].RunID != "" || artifacts[0].Title != "Evidence" {
		t.Fatalf("stored artifacts = %+v", artifacts)
	}
}

func TestServiceCreateMakesDraftWithoutSpawning(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	session, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindMetaOrchestration,
		Title: "Plan quality gates",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if session.Status != StatusDraft || session.RunID != "" || session.TaskID != "" || len(session.Messages) != 0 {
		t.Fatalf("unexpected session: %+v", session)
	}
	if spawner.spawnCalls != 0 {
		t.Fatalf("spawn calls = %d, want 0", spawner.spawnCalls)
	}
}

func TestServiceStartSpawnsSessionWithFirstMessageEnvironmentAndActivitySpec(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindMetaOrchestration,
		Title: "Plan quality gates",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	session, err := svc.Start(context.Background(), ContinueRequest{
		SessionID: draft.ID,
		Message:   "Plan this work.",
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
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

func TestServiceStartStoresResolvedContextAndAddsItToPrompt(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	svc.SetContextResolver(fakeContextResolver{})
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindMetaOrchestration,
		Title: "Plan quality gates",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	session, err := svc.Start(context.Background(), ContinueRequest{
		SessionID: draft.ID,
		Message:   "Plan this work.",
		ContextRefs: []ContextRef{
			{Type: ContextGoal, Ref: "quality-gates"},
			{Type: ContextGoal, Ref: "quality-gates"},
		},
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(session.Messages) != 1 || len(session.Messages[0].Context) != 1 {
		t.Fatalf("message context = %+v", session.Messages)
	}
	if session.Messages[0].Context[0].SelectedAt != testTimestamp {
		t.Fatalf("selected_at = %q, want %q", session.Messages[0].Context[0].SelectedAt, testTimestamp)
	}
	if !strings.Contains(spawner.spawnReq.Prompt, "Attached context:") ||
		!strings.Contains(spawner.spawnReq.Prompt, "[goal] Quality Gates (quality-gates)") ||
		!strings.Contains(spawner.spawnReq.Prompt, "Operator message:\nPlan this work.") {
		t.Fatalf("prompt did not include context before operator message:\n%s", spawner.spawnReq.Prompt)
	}
}

func TestServiceStartRejectsContextOverKindCaps(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	svc := newTestService(t, &fakeSessionSpawner{})
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindMetaOrchestration,
		Title: "Author mode",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	refs := make([]ContextRef, 0, 9)
	for i := 0; i < 9; i++ {
		refs = append(refs, ContextRef{Type: ContextBacklogItem, Ref: "idea/item-" + string(rune('a'+i))})
	}
	if _, err := svc.Start(context.Background(), ContinueRequest{
		SessionID:   draft.ID,
		Message:     "Plan.",
		ContextRefs: refs,
	}); err == nil {
		t.Fatal("Start() error = nil, want cap validation")
	}
}

// [REQ:SWM-P1-004] swarm sessions primed with the operator-loop skill
func TestServiceStartSwarmOperationsUsesOperationsSkillAndPurpose(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindSwarmOperations,
		Title: "Manage Swarm operations",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	session, err := svc.Start(context.Background(), ContinueRequest{
		SessionID: draft.ID,
		Message:   "What should I unblock next?",
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if session.Kind != KindSwarmOperations || session.SkillID != SkillSwarmOperations {
		t.Fatalf("session kind/skill = %q/%q", session.Kind, session.SkillID)
	}
	if spawner.spawnSpec.Purpose != agentactivity.PurposeSwarmOperations {
		t.Fatalf("spawn purpose = %q, want %q", spawner.spawnSpec.Purpose, agentactivity.PurposeSwarmOperations)
	}
	if spawner.spawnSpec.Metadata["skill_id"] != SkillSwarmOperations {
		t.Fatalf("skill metadata = %q", spawner.spawnSpec.Metadata["skill_id"])
	}
	if !strings.Contains(spawner.spawnReq.Prompt, "swarm-manager-operations-session") {
		t.Fatalf("initial prompt did not reference operations skill: %s", spawner.spawnReq.Prompt)
	}
}

func TestServiceStartWorkflowAuthoringUsesAuthoringSkillAndPurpose(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	draft, err := svc.Create(context.Background(), CreateRequest{Kind: KindWorkflowAuthoring, Title: "Author a workflow"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	session, err := svc.Start(context.Background(), ContinueRequest{SessionID: draft.ID, Message: "I want a reliable way to repair plans."})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if session.SkillID != SkillWorkflowAuthoring {
		t.Fatalf("skill = %q, want %q", session.SkillID, SkillWorkflowAuthoring)
	}
	if spawner.spawnSpec.Purpose != agentactivity.PurposeWorkflowAuthoring {
		t.Fatalf("spawn purpose = %q, want %q", spawner.spawnSpec.Purpose, agentactivity.PurposeWorkflowAuthoring)
	}
	if !strings.Contains(spawner.spawnReq.Prompt, "prompt-manager skill read "+SkillWorkflowAuthoring) {
		t.Fatalf("initial prompt did not reference workflow authoring skill: %s", spawner.spawnReq.Prompt)
	}
}

func TestProposalSessionRefreshRecordsNoChangeRecommendation(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()
	processor := &fakeMutationProposalProcessor{ingestion: MutationProposalIngestion{PayloadJSON: `{"form":"mutation_list","mutations":[]}`}}
	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "waiting_for_user", Summary: "```json\n{\"form\":\"mutation_list\",\"mutations\":[]}\n```"}}
	svc := newTestService(t, spawner)
	svc.SetContextResolver(fakeContextResolver{})
	svc.SetMutationProposalProcessor(processor)
	draft, err := svc.Create(context.Background(), CreateRequest{Kind: KindSwarmOperations, Title: "Proposal", ProposalTarget: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"}})
	if err != nil {
		t.Fatal(err)
	}
	if draft.SkillID != SkillProposals {
		t.Fatalf("skill = %q", draft.SkillID)
	}
	if _, err := svc.Start(context.Background(), ContinueRequest{SessionID: draft.ID, Message: "Find missing work."}); err != nil {
		t.Fatal(err)
	}
	refreshed, err := svc.Refresh(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if processor.ingestCalls != 1 || len(refreshed.Proposals) != 1 {
		t.Fatalf("ingest/proposals = %d/%+v", processor.ingestCalls, refreshed.Proposals)
	}
	proposal := refreshed.Proposals[0]
	if proposal.Kind != ProposalNoChangeRecommendation || proposal.Status != ProposalStatusReady || proposal.Target == nil || proposal.Target.Ref != "quality-gates" {
		t.Fatalf("proposal = %+v", proposal)
	}
}

func TestProposalSessionRefreshKeepsNonEmptyMutationListActionable(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()
	processor := &fakeMutationProposalProcessor{ingestion: MutationProposalIngestion{PayloadJSON: `{"form":"mutation_list","mutations":[{"id":"m1","op":"reset_artifacts"}]}`}}
	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "waiting_for_user", Summary: "proposal"}}
	svc := newTestService(t, spawner)
	svc.SetContextResolver(fakeContextResolver{})
	svc.SetMutationProposalProcessor(processor)
	draft, err := svc.Create(context.Background(), CreateRequest{Kind: KindSwarmOperations, Title: "Proposal", ProposalTarget: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Start(context.Background(), ContinueRequest{SessionID: draft.ID, Message: "Find missing work."}); err != nil {
		t.Fatal(err)
	}
	refreshed, err := svc.Refresh(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed.Proposals) != 1 || refreshed.Proposals[0].Kind != ProposalMutationList {
		t.Fatalf("proposal = %+v", refreshed.Proposals)
	}
}

func TestAcceptNoChangeRecommendationSupportsLegacyEmptyMutationLists(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()
	processor := &fakeMutationProposalProcessor{}
	svc := newTestService(t, &fakeSessionSpawner{})
	svc.SetMutationProposalProcessor(processor)
	session := createStartedSession(t, svc, KindSwarmOperations, "Proposal", "Review item.")
	proposal, err := svc.RecordProposal(context.Background(), session.ID, Proposal{Kind: ProposalMutationList, Status: ProposalStatusReady, Summary: "Keep", PayloadJSON: `{"form":"mutation_list","rationale":"Still valid.","mutations":[]}`, Target: &ProposalTarget{Type: ContextBacklogItem, Ref: "research/item", Name: "Item"}})
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := svc.AcceptNoChangeRecommendation(context.Background(), session.ID, proposal.ID, "confirmed")
	if err != nil {
		t.Fatal(err)
	}
	if !processor.acceptedKeep || accepted.Proposals[0].Status != ProposalStatusApplied || accepted.Proposals[0].Decisions[0].Kind != "accept_keep" {
		t.Fatalf("accepted = %+v processor=%+v", accepted.Proposals, processor)
	}
}

func TestProposalSessionRefreshPersistsRevisionStateForMalformedTurn(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()
	processor := &fakeMutationProposalProcessor{ingestion: MutationProposalIngestion{PayloadJSON: `{}`, ParseWarnings: []string{"invalid JSON"}}}
	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "waiting_for_user", Summary: "not valid proposal output"}}
	svc := newTestService(t, spawner)
	svc.SetContextResolver(fakeContextResolver{})
	svc.SetMutationProposalProcessor(processor)
	draft, err := svc.Create(context.Background(), CreateRequest{Kind: KindSwarmOperations, Title: "Proposal", ProposalTarget: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Start(context.Background(), ContinueRequest{SessionID: draft.ID, Message: "Find missing work."}); err != nil {
		t.Fatal(err)
	}
	refreshed, err := svc.Refresh(context.Background(), draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(refreshed.Proposals) != 1 || refreshed.Proposals[0].Status != ProposalStatusNeedsRevision || !refreshed.Proposals[0].NeedsRevision {
		t.Fatalf("proposal = %+v", refreshed.Proposals)
	}
}

func TestMutationProposalDecisionAndRevisionUseSameSessionRun(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()
	processor := &fakeMutationProposalProcessor{application: MutationProposalApplication{Outcomes: []MutationOutcome{{MutationID: "m1", Applied: true}, {MutationID: "m2", Skipped: true}}}}
	spawner := &fakeSessionSpawner{}
	svc := newTestService(t, spawner)
	svc.SetMutationProposalProcessor(processor)
	session := createStartedSession(t, svc, KindSwarmOperations, "Proposal", "Find missing work.")
	proposal, err := svc.RecordProposal(context.Background(), session.ID, Proposal{Kind: ProposalMutationList, Status: ProposalStatusReady, Summary: "Proposal", PayloadJSON: `{"form":"mutation_list","mutations":[]}`, Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"}})
	if err != nil {
		t.Fatal(err)
	}
	applied, err := svc.DecideMutationListProposal(context.Background(), session.ID, proposal.ID, []string{"m1"}, "apply one")
	if err != nil {
		t.Fatal(err)
	}
	if processor.accepted[0] != "m1" || applied.Proposals[0].Status != ProposalStatusApplied || len(applied.Proposals[0].Decisions) != 1 {
		t.Fatalf("applied = %+v processor=%+v", applied.Proposals, processor)
	}
	proposal, err = svc.RecordProposal(context.Background(), session.ID, Proposal{Kind: ProposalMutationList, Status: ProposalStatusNeedsRevision, NeedsRevision: true, Summary: "Needs revision", PayloadJSON: `{}`, Target: &ProposalTarget{Type: ContextGoal, Ref: "quality-gates", Name: "Quality Gates"}, ValidationErrors: []string{"unknown target"}})
	if err != nil {
		t.Fatal(err)
	}
	revised, err := svc.RequestMutationProposalRevision(context.Background(), session.ID, proposal.ID, "please narrow scope")
	if err != nil {
		t.Fatal(err)
	}
	if spawner.continueRunID != session.RunID || !strings.Contains(spawner.continueMessage, "unknown target") || !strings.Contains(spawner.continueMessage, "please narrow scope") {
		t.Fatalf("continue = %q / %q", spawner.continueRunID, spawner.continueMessage)
	}
	revisedProposal, found := findProposal(revised, proposal.ID)
	if !found || revisedProposal.Status != ProposalStatusSuperseded {
		t.Fatalf("revision proposal = %+v", revised.Proposals)
	}
}

// TestServiceStartInitialPromptDeliversFullSkill proves the spawned agent is
// directed to read its whole operating guide (the full skill methodology), not
// just the attached startup-brief snapshot — the Phase 7 guarantee that the
// authoring methodology reaches the agent verbatim.
func TestServiceStartInitialPromptDeliversFullSkill(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindMetaOrchestration,
		Title: "Author a mode",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := svc.Start(context.Background(), ContinueRequest{
		SessionID: draft.ID,
		Message:   "I keep running the same review loop by hand.",
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	prompt := spawner.spawnReq.Prompt
	if !strings.Contains(prompt, "prompt-manager skill read "+SkillMetaOrchestrator) {
		t.Fatalf("initial prompt does not direct the agent to read the full skill: %s", prompt)
	}
}

func TestServiceStartInjectsStartupBriefContextByDefault(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	for _, kind := range []Kind{KindSwarmOperations, KindMetaOrchestration, KindWorkflowAuthoring} {
		t.Run(string(kind), func(t *testing.T) {
			spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
			svc := newTestService(t, spawner)
			svc.SetContextResolver(fakeStartupContextResolver{})
			draft, err := svc.Create(context.Background(), CreateRequest{
				Kind:  kind,
				Title: "Manage session",
			})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			session, err := svc.Start(context.Background(), ContinueRequest{
				SessionID: draft.ID,
				Message:   "What is the current status?",
			})
			if err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			if len(session.Messages) != 1 || len(session.Messages[0].Context) != 1 {
				t.Fatalf("context = %+v", session.Messages)
			}
			item := session.Messages[0].Context[0]
			if item.Type != ContextStartupBrief || item.Ref != StartupBriefRefForKind(kind) {
				t.Fatalf("context item = %+v, want startup brief", item)
			}
			if !strings.Contains(spawner.spawnReq.Prompt, "Startup brief context is attached below") {
				t.Fatalf("prompt missing startup brief instruction:\n%s", spawner.spawnReq.Prompt)
			}
		})
	}
}

func TestServiceStartSkipsStartupBriefWhenAutoContextNone(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	svc.SetContextResolver(fakeStartupContextResolver{})
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindSwarmOperations,
		Title: "Manage Swarm operations",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	session, err := svc.Start(context.Background(), ContinueRequest{
		SessionID:         draft.ID,
		Message:           "What is the current status?",
		AutoContextPolicy: AutoContextNone,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(session.Messages) != 1 || len(session.Messages[0].Context) != 0 {
		t.Fatalf("context = %+v, want none", session.Messages)
	}
}

func TestServiceContinueAppendsMessageAndUsesTrackedContinuation(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{}
	svc := newTestService(t, spawner)
	session := createStartedSession(t, svc, KindMetaOrchestration, "Session", "Continue.")

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
	if spawner.continueSpec.OwnerType != agentactivity.OwnerSession || spawner.continueSpec.Purpose != agentactivity.PurposeMetaOrchestration {
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
	session := createStartedSession(t, svc, KindMetaOrchestration, "Plan quality gates", "Plan this work.")

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
	session := createStartedSession(t, svc, KindMetaOrchestration, "Plan", "Plan.")

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

func TestServiceRefreshMapsFailedRunAndIsIdempotent(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "failed", ErrorMsg: "sandbox process ended without exit info"}}
	svc := newTestService(t, spawner)
	session := createStartedSession(t, svc, KindMetaOrchestration, "Plan", "Plan.")

	refreshed, err := svc.Refresh(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.Status != StatusFailed {
		t.Fatalf("status after refresh = %q, want failed", refreshed.Status)
	}
	if refreshed.FailureReason != "sandbox process ended without exit info" {
		t.Fatalf("failure reason = %q", refreshed.FailureReason)
	}
	if len(refreshed.Messages) != 1 {
		t.Fatalf("failed run with no summary should not append messages: %+v", refreshed.Messages)
	}

	refreshedAgain, err := svc.Refresh(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Refresh() second call error = %v", err)
	}
	if refreshedAgain.Status != StatusFailed || len(refreshedAgain.Messages) != 1 {
		t.Fatalf("second refresh should be idempotent: %+v", refreshedAgain)
	}
}

func TestServiceListEventsReturnsEmptyForDraftAndMapsRunEvents(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{}
	svc := newTestService(t, spawner)
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindMetaOrchestration,
		Title: "Plan",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	empty, err := svc.ListEvents(context.Background(), ListEventsRequest{SessionID: draft.ID})
	if err != nil {
		t.Fatalf("ListEvents(draft) error = %v", err)
	}
	if len(empty.Events) != 0 {
		t.Fatalf("draft events = %+v, want empty", empty.Events)
	}

	spawner.events = []*agentdomainpb.RunEvent{
		{
			Id:        "evt-1",
			RunId:     "run-1",
			Sequence:  7,
			EventType: agentdomainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL,
			Timestamp: timestamppb.New(time.Date(2026, 5, 1, 12, 3, 0, 0, time.UTC)),
			Data: &agentdomainpb.RunEvent_ToolCall{ToolCall: &agentdomainpb.ToolCallEventData{
				ToolName:   "Read",
				ToolCallId: "call-1",
				Input:      mustStruct(t, map[string]any{"file": "AGENTS.md"}),
			}},
		},
	}
	started, err := svc.Start(context.Background(), ContinueRequest{SessionID: draft.ID, Message: "Plan."})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	result, err := svc.ListEvents(context.Background(), ListEventsRequest{SessionID: started.ID, AfterSequence: 3, Limit: 25})
	if err != nil {
		t.Fatalf("ListEvents(started) error = %v", err)
	}
	if spawner.eventsAfterSequence != 3 || spawner.eventsLimit != 25 {
		t.Fatalf("event cursor = %d/%d", spawner.eventsAfterSequence, spawner.eventsLimit)
	}
	if len(result.Events) != 1 || result.Events[0].EventType != "tool_call" || result.Events[0].ToolName != "Read" {
		t.Fatalf("events = %+v", result.Events)
	}
	if result.NextAfterSequence != 7 {
		t.Fatalf("next sequence = %d, want 7", result.NextAfterSequence)
	}
}

func TestServiceListEventsSanitizesHTMLErrorPages(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	const cloudflareHTML = `<!DOCTYPE html><html><head><title>itsagitime.com | 502: Bad gateway</title></head><body><div id="cf-wrapper">cloudflare</div></body></html>`

	spawner := &fakeSessionSpawner{
		events: []*agentdomainpb.RunEvent{
			{
				Id:        "evt-html",
				RunId:     "run-1",
				Sequence:  18,
				EventType: agentdomainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT,
				Timestamp: timestamppb.New(time.Date(2026, 5, 15, 22, 0, 0, 0, time.UTC)),
				Data: &agentdomainpb.RunEvent_ToolResult{ToolResult: &agentdomainpb.ToolResultEventData{
					ToolName: "Fetch",
					Output:   cloudflareHTML,
				}},
			},
		},
	}
	svc := newTestService(t, spawner)
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindSwarmOperations,
		Title: "Manage Swarm operations",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	started, err := svc.Start(context.Background(), ContinueRequest{SessionID: draft.ID, Message: "Check the tunnel."})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	result, err := svc.ListEvents(context.Background(), ListEventsRequest{SessionID: started.ID})
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events = %+v", result.Events)
	}
	if strings.Contains(result.Events[0].Output, "<!DOCTYPE html>") || strings.Contains(result.Events[0].Output, "cf-wrapper") {
		t.Fatalf("raw HTML leaked into event output: %q", result.Events[0].Output)
	}
	if result.Events[0].Output != "Upstream tunnel returned an HTML 502 Bad Gateway page. The target service may be unavailable or timed out." {
		t.Fatalf("unexpected sanitized output: %q", result.Events[0].Output)
	}
}

func TestServiceDeleteStopsActiveRunBeforeRemovingSession(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	spawner := &fakeSessionSpawner{}
	svc := newTestService(t, spawner)
	session := createStartedSession(t, svc, KindMetaOrchestration, "Plan", "Plan.")

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
	session := createStartedSession(t, svc, KindMetaOrchestration, "Plan", "Plan.")

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
	session := createStartedSession(t, svc, KindMetaOrchestration, "Plan", "Plan.")
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

func TestServiceUploadAttachmentsStoresSessionOwnedImages(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	svc := newTestService(t, &fakeSessionSpawner{})
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  KindMetaOrchestration,
		Title: "Plan",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	attachments, err := svc.UploadAttachments(context.Background(), draft.ID, []AttachmentUpload{{
		Filename:    "screenshot.png",
		ContentType: "image/png",
		SizeBytes:   4,
		Reader:      strings.NewReader("data"),
	}})
	if err != nil {
		t.Fatalf("UploadAttachments() error = %v", err)
	}
	if len(attachments) != 1 || attachments[0].ID == "" || attachments[0].ContentType != "image/png" {
		t.Fatalf("attachments = %+v", attachments)
	}
	loaded, err := svc.Get(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(loaded.Attachments) != 1 || loaded.Attachments[0].Filename != "screenshot.png" {
		t.Fatalf("stored attachments = %+v", loaded.Attachments)
	}
	path, _, err := svc.AttachmentPath(draft.ID, attachments[0].ID)
	if err != nil {
		t.Fatalf("AttachmentPath() error = %v", err)
	}
	if !strings.HasSuffix(path, "screenshot.png") {
		t.Fatalf("attachment path = %q", path)
	}
	if _, err := svc.UploadAttachments(context.Background(), draft.ID, []AttachmentUpload{{
		Filename:    "notes.txt",
		ContentType: "text/plain",
		SizeBytes:   4,
		Reader:      strings.NewReader("data"),
	}}); err == nil {
		t.Fatal("UploadAttachments() text/plain error = nil, want validation")
	}
}

func TestServiceApplyBacklogBatchImportProposalUsesSessionAttribution(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	applier := &fakeBacklogBatchApplier{}
	svc := newTestService(t, &fakeSessionSpawner{})
	svc.SetBacklogBatchApplier(applier)
	session := createStartedSession(t, svc, KindMetaOrchestration, "Plan", "Plan.")
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
	session := createStartedSession(t, svc, KindMetaOrchestration, "Plan", "Plan.")

	artifacts, err := svc.AttachArtifacts(context.Background(), []Artifact{
		{
			SessionID:    session.ID,
			ArtifactType: ArtifactBacklogItem,
			Action:       ArtifactActionCreated,
			EntityRef:    "idea/a",
		},
		{
			SessionID:    session.ID,
			ArtifactType: ArtifactMilestone,
			Action:       ArtifactActionCreated,
			EntityRef:    "milestone-a",
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
	store := svc.store.(*FileStore)
	if _, err := os.Stat(filepath.Join(store.sessionDir(session.ID), artifactsFileName)); err != nil {
		t.Fatalf("new artifact attachment must persist session artifact JSONL: %v", err)
	}
}

func TestServiceApplyLegacyOperatingModeDraftIsReadOnly(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	svc := newTestService(t, &fakeSessionSpawner{})
	session := createStartedSession(t, svc, KindMetaOrchestration, "Session", "Continue.")
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

	if _, _, err := svc.ApplyProposal(context.Background(), session.ID, proposal.ID); err == nil {
		t.Fatal("ApplyProposal() error = nil, want legacy proposal rejection")
	}
}

func TestServiceApplyLegacyOperatingModeImplementationPlanIsReadOnly(t *testing.T) {
	restoreClock := freezeAgentSessionClock(t)
	defer restoreClock()

	applier := &fakeBacklogBatchApplier{}
	svc := newTestService(t, &fakeSessionSpawner{})
	svc.SetBacklogBatchApplier(applier)
	session := createStartedSession(t, svc, KindMetaOrchestration, "Session", "Continue.")
	proposal, err := svc.RecordProposal(context.Background(), session.ID, Proposal{
		ID:      "prop-plan",
		Kind:    ProposalOperatingModeImplementationPlan,
		Status:  ProposalStatusReady,
		Summary: "Create implementation milestone and items.",
		PayloadJSON: `{
			"mode_id":"phased-refactor",
			"backlog_batch_import":{
				"milestones":[{"name":"phased-refactor-mode","title":"Phased Refactor Mode"}],
				"items":[{"name":"implement-mode","title":"Implement mode","kind":"feature","milestone":"phased-refactor-mode"}]
			}
		}`,
		CreatedAt: testTimestamp,
		UpdatedAt: testTimestamp,
	})
	if err != nil {
		t.Fatalf("RecordProposal() error = %v", err)
	}

	if _, _, err = svc.ApplyProposal(context.Background(), session.ID, proposal.ID); err == nil {
		t.Fatal("ApplyProposal() error = nil, want legacy proposal rejection")
	}
	if applier.payloadJSON != "" {
		t.Fatalf("legacy proposal invoked backlog batch applier: %s", applier.payloadJSON)
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

func createStartedSession(t *testing.T, svc *Service, kind Kind, title, message string) Session {
	t.Helper()
	draft, err := svc.Create(context.Background(), CreateRequest{
		Kind:  kind,
		Title: title,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	session, err := svc.Start(context.Background(), ContinueRequest{
		SessionID: draft.ID,
		Message:   message,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return session
}

func mustStruct(t *testing.T, fields map[string]any) *structpb.Struct {
	t.Helper()
	value, err := structpb.NewStruct(fields)
	if err != nil {
		t.Fatalf("NewStruct() error = %v", err)
	}
	return value
}

type fakeBacklogBatchApplier struct {
	payloadJSON string
	provenance  identity.Provenance
}

type fakeMutationProposalProcessor struct {
	ingestion    MutationProposalIngestion
	application  MutationProposalApplication
	ingestCalls  int
	accepted     []string
	acceptedKeep bool
}

func (f *fakeMutationProposalProcessor) Ingest(_ context.Context, _ ProposalTarget, _ string) (MutationProposalIngestion, error) {
	f.ingestCalls++
	return f.ingestion, nil
}

func (f *fakeMutationProposalProcessor) Apply(_ context.Context, _ ProposalTarget, _ string, accepted []string, _ MutationProposalSource) (MutationProposalApplication, error) {
	f.accepted = append([]string(nil), accepted...)
	return f.application, nil
}

func (f *fakeMutationProposalProcessor) AcceptNoChange(_ context.Context, _ ProposalTarget, _ string, _ MutationProposalSource) error {
	f.acceptedKeep = true
	return nil
}

type fakeContextResolver struct{}

func (fakeContextResolver) ResolveSessionMessageContext(_ context.Context, refs []ContextRef, _ ContextLimits) ([]ContextItem, error) {
	items := make([]ContextItem, 0, len(refs))
	for _, ref := range refs {
		items = append(items, ContextItem{
			Type:    ref.Type,
			Ref:     ref.Ref,
			Title:   "Quality Gates",
			Summary: "Tighten quality gates.",
		})
	}
	return items, nil
}

type fakeStartupContextResolver struct {
	fakeContextResolver
}

func (fakeStartupContextResolver) ResolveSessionStartupBrief(_ context.Context, kind Kind, _ ContextLimits) (ContextItem, error) {
	return ContextItem{
		Type:    ContextStartupBrief,
		Ref:     StartupBriefRefForKind(kind),
		Title:   "Startup Brief",
		Summary: "Use this brief first.",
	}, nil
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
	spawnCalls          int
	spawnReq            agentmanager.SessionSpawnRequest
	spawnSpec           agentactivity.Spec
	continueRunID       string
	continueMessage     string
	continueSpec        agentactivity.Spec
	stoppedRunID        string
	stopErr             error
	runState            agentmanager.RunState
	events              []*agentdomainpb.RunEvent
	eventsHasMore       bool
	eventsAfterSequence int64
	eventsLimit         int32
}

func (f *fakeSessionSpawner) SpawnSession(ctx context.Context, req agentmanager.SessionSpawnRequest) (agentmanager.RunResult, error) {
	f.spawnCalls++
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

func (f *fakeSessionSpawner) GetRunEvents(_ context.Context, _ string, opts agentmanager.RunEventsOptions) ([]*agentdomainpb.RunEvent, bool, error) {
	f.eventsAfterSequence = opts.AfterSequence
	f.eventsLimit = opts.Limit
	return f.events, f.eventsHasMore, nil
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
