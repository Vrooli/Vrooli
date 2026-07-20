package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"

	"github.com/google/uuid"
)

func TestEnforceWorkflowTriggerRejectsDisallowedInitiator(t *testing.T) {
	o := &Orchestrator{}
	revision := &domain.WorkflowRevision{Key: "owner/guarded", Definition: domain.WorkflowDefinition{Trigger: domain.WorkflowTriggerPolicy{Initiators: []domain.WorkflowInitiator{domain.WorkflowInitiatorHuman}}}}
	err := o.enforceWorkflowTrigger(context.Background(), revision, StartWorkflowExecutionRequest{Initiator: domain.WorkflowInitiatorProgrammatic})
	var denied *TriggerPolicyError
	if !errors.As(err, &denied) {
		t.Fatalf("error = %v, want TriggerPolicyError", err)
	}
	if denied.Decision != "initiator_not_allowed" || denied.WorkflowKey != revision.Key {
		t.Fatalf("denial = %#v", denied)
	}
}

func TestStartWorkflowExecutionDeniesPersistedSelfCallerChain(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	secret := []byte("workflow-trigger-policy-test-secret-0123456789")
	o.identitySecret = secret

	revision := relayDefinition()
	revision.Key = "owner/guarded"
	revision.Definition.Key = revision.Key
	revision.Definition.Trigger = domain.WorkflowTriggerPolicy{SelfTrigger: domain.WorkflowSelfTriggerPolicy{Mode: domain.WorkflowSelfTriggerDeny}}
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatalf("activate workflow revision: %v", err)
	}
	parent, err := o.workflowEngine.Start(ctx, revision, json.RawMessage(`{}`), "trigger-parent")
	if err != nil {
		t.Fatalf("start parent: %v", err)
	}
	if _, err = o.workflowEngine.Advance(ctx, parent.ID); err != nil {
		t.Fatalf("advance parent: %v", err)
	}
	if _, err = o.workflowEngine.Advance(ctx, parent.ID); err != nil {
		t.Fatalf("dispatch parent: %v", err)
	}
	runID := runIDForNode(t, repos.WorkflowExecutions, parent.ID, "a")
	if runID == uuid.Nil {
		t.Fatal("workflow did not persist parent run attempt")
	}
	profile, err := o.CreateProfile(ctx, &domain.AgentProfile{Name: "trigger profile", ProfileKey: "owner/trigger-profile", RoleRef: "code.default"})
	if err != nil {
		t.Fatalf("create token profile: %v", err)
	}
	task, err := o.CreateTask(ctx, &domain.Task{Title: "trigger task", Description: "workflow trigger test", ScopePath: ".", ProjectRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("create token task: %v", err)
	}
	run := &domain.Run{ID: runID, TaskID: task.ID, AgentProfileID: &profile.ID, Status: domain.RunStatusRunning, RunMode: domain.RunModeInPlace, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("persist token-owning run: %v", err)
	}
	now := time.Now()
	token, err := identity.GenerateToken(&identity.Claims{RunID: runID, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix()}, secret)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	run.IdentityTokenHash = identity.HashToken(token)
	if err := repos.Runs.Update(ctx, run); err != nil {
		t.Fatalf("activate token: %v", err)
	}

	_, err = o.StartWorkflowExecution(ctx, StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: revision.Key, Input: json.RawMessage(`{}`), IdempotencyKey: "trigger-child", Initiator: domain.WorkflowInitiatorAgent, IdentityToken: token})
	var denied *TriggerPolicyError
	if !errors.As(err, &denied) || denied.Decision != "self_trigger_denied" {
		t.Fatalf("self start error = %v, want self_trigger_denied", err)
	}
	if got, err := repos.WorkflowExecutions.GetByIdempotencyKey(ctx, "trigger-child"); err != nil || got != nil {
		t.Fatalf("denied execution was persisted: execution=%+v err=%v", got, err)
	}

	depthRevision := *revision
	depthRevision.ID = uuid.New()
	depthRevision.Digest = "sha256:guarded-depth-one"
	depthRevision.Definition = revision.Definition
	depthRevision.Definition.Trigger = domain.WorkflowTriggerPolicy{SelfTrigger: domain.WorkflowSelfTriggerPolicy{Mode: domain.WorkflowSelfTriggerAllow, MaxDepth: 1}}
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{&depthRevision}); err != nil {
		t.Fatalf("activate depth-limited revision: %v", err)
	}
	_, err = o.StartWorkflowExecution(ctx, StartWorkflowExecutionRequest{Owner: "owner", WorkflowKey: revision.Key, DefinitionDigest: depthRevision.Digest, Input: json.RawMessage(`{}`), IdempotencyKey: "trigger-depth-child", Initiator: domain.WorkflowInitiatorAgent, IdentityToken: token})
	if !errors.As(err, &denied) || denied.Decision != "self_trigger_depth_reached" {
		t.Fatalf("depth-limited self start error = %v, want self_trigger_depth_reached", err)
	}
}

func TestEnforceWorkflowTriggerAllowsProgrammaticDefault(t *testing.T) {
	o := &Orchestrator{}
	revision := &domain.WorkflowRevision{Key: "owner/default", Definition: domain.WorkflowDefinition{}}
	if err := o.enforceWorkflowTrigger(context.Background(), revision, StartWorkflowExecutionRequest{}); err != nil {
		t.Fatalf("default policy rejected programmatic start: %v", err)
	}
}

func TestEnforceWorkflowTriggerDoesNotTreatUnverifiedTokenAsAgent(t *testing.T) {
	o := &Orchestrator{}
	revision := &domain.WorkflowRevision{Key: "owner/agent-only", Definition: domain.WorkflowDefinition{Trigger: domain.WorkflowTriggerPolicy{Initiators: []domain.WorkflowInitiator{domain.WorkflowInitiatorAgent}}}}
	err := o.enforceWorkflowTrigger(context.Background(), revision, StartWorkflowExecutionRequest{Initiator: domain.WorkflowInitiatorAgent, IdentityToken: "not-verifiable"})
	var denied *TriggerPolicyError
	if !errors.As(err, &denied) || denied.Initiator != domain.WorkflowInitiatorProgrammatic {
		t.Fatalf("unverified identity must be governed as programmatic, got %#v (%v)", denied, err)
	}
}
