package transitionrunner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/development"
	"swarm-manager/internal/workflowcontract"
)

const ContractDevelopmentTransition = "contract-development"

// DevelopmentAdapter is the composition adapter for the retained
// contract-development work shape. Its start function is deliberately the
// only path that can turn an approved engagement into an owner invocation.
type DevelopmentAdapter struct {
	service     *development.Service
	coordinator *development.Coordinator
}

func NewDevelopmentAdapter(service *development.Service, coordinator *development.Coordinator) *DevelopmentAdapter {
	return &DevelopmentAdapter{service: service, coordinator: coordinator}
}

func (a *DevelopmentAdapter) Register(runner *Runner) {
	if a == nil || runner == nil {
		return
	}
	runner.RegisterInput(ContractDevelopmentTransition, a.buildInput)
	runner.RegisterStart(ContractDevelopmentTransition, a.start)
	runner.RegisterApply("apply_development_outcome", a.apply)
}

func (a *DevelopmentAdapter) buildInput(ctx context.Context, subjectRef string) (Snapshot, error) {
	if a == nil || a.service == nil {
		return Snapshot{}, fmt.Errorf("development service is not configured")
	}
	state, err := a.service.Get(ctx, subjectRef)
	if err != nil {
		return Snapshot{}, err
	}
	if state.WorkShape != ContractDevelopmentTransition {
		return Snapshot{}, fmt.Errorf("work item %q has incompatible work shape %q: %w", subjectRef, state.WorkShape, development.ErrDenied)
	}
	retained, err := a.service.Snapshot(ctx, state.Digest)
	if err != nil {
		return Snapshot{}, err
	}
	input, err := developmentInputValue(state, retained)
	if err != nil {
		return Snapshot{}, err
	}
	// Reservation, binding and settlement update the engagement version. The
	// transition input is therefore pinned to the approved target digest and
	// retained proposal, which are the facts that must remain unchanged while
	// the owner works.
	entity := struct {
		WorkItem string
		Digest   string
		Proposal development.Proposal
	}{state.WorkItem, state.Digest, retained.Proposal}
	frontier := struct{ Digest string }{state.Digest}
	return SnapshotFromSubject(input, entity, frontier)
}

func (a *DevelopmentAdapter) start(ctx context.Context, in StartInvocation) (agentmanager.WorkflowStart, error) {
	if a == nil || a.service == nil || a.coordinator == nil {
		return agentmanager.WorkflowStart{}, fmt.Errorf("development coordinator is not configured")
	}
	state, err := a.service.Get(ctx, in.SubjectRef)
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	if state.WorkShape != ContractDevelopmentTransition {
		return agentmanager.WorkflowStart{}, fmt.Errorf("work item %q has incompatible work shape %q: %w", in.SubjectRef, state.WorkShape, development.ErrDenied)
	}
	retained, err := a.service.Snapshot(ctx, state.Digest)
	if err != nil {
		return agentmanager.WorkflowStart{}, err
	}
	if in.Snapshot.EntityVersion == "" {
		return agentmanager.WorkflowStart{}, errors.New("development transition has no pinned target identity")
	}
	upper := development.Usage{
		Tokens:      retained.Proposal.MaxTokens - state.Used.Tokens,
		WallSeconds: retained.Proposal.MaxWallSeconds - state.Used.WallSeconds,
	}
	if upper.Tokens <= 0 || upper.WallSeconds <= 0 {
		return agentmanager.WorkflowStart{}, fmt.Errorf("development aggregate allowance is exhausted: %w", development.ErrDenied)
	}
	effects := make([]development.Effect, 0, len(retained.Proposal.AllowedEffects))
	for _, raw := range retained.Proposal.AllowedEffects {
		effect, parseErr := development.ParseEffect(raw)
		if parseErr != nil {
			return agentmanager.WorkflowStart{}, parseErr
		}
		effects = append(effects, effect)
	}
	activity := in.Invocation.Activity
	var ownerActivity *workflowcontract.Activity
	if activity != nil {
		ownerActivity = &workflowcontract.Activity{OwnerType: activity.OwnerType, OwnerKind: activity.OwnerKind, OwnerName: activity.OwnerName, OwnerTitle: activity.OwnerTitle, Purpose: activity.Purpose}
	}
	invocation := workflowcontract.Invocation{
		Owner: in.Invocation.Owner, WorkflowKey: in.Invocation.WorkflowKey, WorkflowDigest: "", Input: in.Invocation.Input,
		IdempotencyKey: developmentAttemptKey(state, state.Digest), FirstRunNodeID: in.Invocation.FirstRunNodeID,
		Activity: ownerActivity,
	}
	started, err := a.coordinator.Start(ctx, development.StartRequest{
		WorkItem: state.WorkItem, Digest: state.Digest, AttemptKey: invocation.IdempotencyKey, Mode: "workflow-fallback", Upper: upper,
		Invocation: invocation, Effects: effects,
	})
	if err != nil {
		return agentmanager.WorkflowStart{}, fmt.Errorf("development coordinator start: %w", err)
	}
	if strings.TrimSpace(started.Execution.WorkflowDigest) == "" {
		return agentmanager.WorkflowStart{}, fmt.Errorf("development owner returned no workflow revision: %w", development.ErrConflict)
	}
	return agentmanager.WorkflowStart{
		ExecutionID: started.Execution.ExecutionID, RunID: started.Execution.RunID,
		WorkflowDigest: started.Execution.WorkflowDigest, DefinitionDigest: started.Execution.DefinitionDigest,
		ApprovalDigest: started.Execution.ApprovalDigest, GrantDigest: started.Execution.GrantDigest,
	}, nil
}

func (a *DevelopmentAdapter) apply(ctx context.Context, subjectRef string, outcome Outcome) error {
	if a == nil || a.service == nil {
		return fmt.Errorf("development service is not configured")
	}
	state, err := a.service.Get(ctx, subjectRef)
	if err != nil {
		return err
	}
	var attempt *development.Attempt
	for i := range state.Attempts {
		if state.Attempts[i].ExecutionID == outcome.ExecutionID {
			attempt = &state.Attempts[i]
			break
		}
	}
	if attempt == nil {
		return fmt.Errorf("development execution %q is not bound to %q: %w", outcome.ExecutionID, subjectRef, development.ErrConflict)
	}
	if outcome.WorkflowDigest != "" && attempt.WorkflowDigest != "" && outcome.WorkflowDigest != attempt.WorkflowDigest {
		return fmt.Errorf("development completion workflow revision %q differs from bound %q: %w", outcome.WorkflowDigest, attempt.WorkflowDigest, development.ErrConflict)
	}
	if outcome.ApprovalDigest != "" && outcome.ApprovalDigest != attempt.Digest {
		return fmt.Errorf("development completion approval %q differs from bound %q: %w", outcome.ApprovalDigest, attempt.Digest, development.ErrConflict)
	}
	if outcome.GrantDigest != "" && attempt.GrantDigest != "" && outcome.GrantDigest != attempt.GrantDigest {
		return fmt.Errorf("development completion grant %q differs from bound %q: %w", outcome.GrantDigest, attempt.GrantDigest, development.ErrConflict)
	}
	if outcome.Usage == nil || !outcome.Usage.TokensKnown || outcome.Usage.WallSeconds <= 0 {
		if a.coordinator == nil {
			return fmt.Errorf("development terminal usage is unavailable: %w", development.ErrDenied)
		}
		_, err = a.coordinator.Collect(ctx, state.WorkItem, attempt.Key, outcome.ExecutionID)
		return err
	}
	_, err = a.service.Settle(ctx, state.WorkItem, attempt.Key, outcome.ExecutionID, &development.Usage{Tokens: outcome.Usage.Tokens, WallSeconds: outcome.Usage.WallSeconds}, firstNonEmpty(outcome.TerminalCode, outcome.Name))
	return err
}

func developmentInputValue(state development.Engagement, retained development.Snapshot) (*structpb.Value, error) {
	proposal, err := jsonValue(retained.Proposal)
	if err != nil {
		return nil, err
	}
	artifacts, err := jsonValue(retained.Artifacts)
	if err != nil {
		return nil, err
	}
	checkpoint, err := jsonValue(state.Campaign)
	if err != nil {
		return nil, err
	}
	value := map[string]any{
		"entity": map[string]any{"kind": "backlog-item", "name": state.WorkItem, "version": state.Digest},
		"development": map[string]any{
			"workItem": state.WorkItem, "approvalDigest": state.Digest, "goalMessage": retained.GoalMessage,
			"proposal": proposal, "artifacts": artifacts, "checkpoint": checkpoint,
		},
	}
	return structpb.NewValue(value)
}

func jsonValue(value any) (any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func developmentAttemptKey(state development.Engagement, digest string) string {
	for _, attempt := range state.Attempts {
		if attempt.Digest == digest && attempt.SettledAt == nil && attempt.Key != "" {
			return attempt.Key
		}
	}
	seed := fmt.Sprintf("%s\x00%s\x00%d", state.WorkItem, digest, len(state.Attempts))
	sum := sha256.Sum256([]byte(seed))
	return "dev-" + hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "terminal"
}
