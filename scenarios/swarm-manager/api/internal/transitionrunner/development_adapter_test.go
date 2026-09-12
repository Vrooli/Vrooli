package transitionrunner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/development"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitions"
	"swarm-manager/internal/workflowcontract"
)

type developmentMemoryRepository struct {
	state    development.Engagement
	snapshot development.Snapshot
}

func (r *developmentMemoryRepository) Get(context.Context, string) (development.Engagement, error) {
	if r.state.WorkItem == "" {
		return development.Engagement{}, development.ErrNotFound
	}
	return r.state, nil
}

func (r *developmentMemoryRepository) Snapshot(context.Context, string) (development.Snapshot, error) {
	if r.snapshot.Digest == "" {
		return development.Snapshot{}, development.ErrNotFound
	}
	return r.snapshot, nil
}

func (r *developmentMemoryRepository) Commit(_ context.Context, expected int64, state development.Engagement, snapshot *development.Snapshot) error {
	if r.state.WorkItem != "" && expected != r.state.Version {
		return development.ErrConflict
	}
	r.state = state
	if snapshot != nil {
		r.snapshot = *snapshot
	}
	return nil
}

type developmentOwnerStub struct{ starts int }

func (s *developmentOwnerStub) Start(_ context.Context, invocation workflowcontract.Invocation) (workflowcontract.Start, error) {
	s.starts++
	return workflowcontract.Start{
		ExecutionID: "owner-development-1", WorkflowDigest: "sha256:workflow-development",
		ApprovalDigest: invocation.ApprovalDigest, GrantDigest: invocation.GrantDigest,
	}, nil
}

func (*developmentOwnerStub) Collect(context.Context, string) (workflowcontract.Completion, error) {
	return workflowcontract.Completion{}, nil
}

type developmentAgentManagerStub struct{}

func (*developmentAgentManagerStub) StartWorkflow(context.Context, agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	panic("development transition must use its registered starter")
}

func (*developmentAgentManagerStub) CollectWorkflow(context.Context, string) (agentmanager.InvocationCompletion, error) {
	return agentmanager.InvocationCompletion{}, nil
}

func developmentRunnerFixture(t *testing.T, scenario string) (*Runner, *developmentMemoryRepository, *developmentOwnerStub, string) {
	t.Helper()
	item := "execute/" + scenario + "-development"
	digest := "approval-" + scenario
	repository := &developmentMemoryRepository{
		state:    development.Engagement{WorkItem: item, WorkShape: ContractDevelopmentTransition, Version: 1, Digest: digest, Status: "approved"},
		snapshot: development.Snapshot{Digest: digest, Proposal: development.Proposal{WorkItem: item, Scenario: scenario, MaxTokens: 1000, MaxWallSeconds: 60}},
	}
	service := development.NewService(repository, development.Reviewer{}, nil, nil)
	owner := &developmentOwnerStub{}
	adapter := NewDevelopmentAdapter(service, development.NewCoordinator(service, owner))

	registryDir := t.TempDir()
	definition := transitions.Definition{
		SchemaVersion: transitions.SchemaVersion, Key: ContractDevelopmentTransition, Subject: "backlog-item", Kind: transitions.KindWorkflow,
		Workflow:      &transitions.Locator{Owner: "swarm-manager", Key: "swarm-manager/contract-development"},
		InputContract: "contract-development-input/v1", TerminalOutcomes: []string{"repaired"}, ApplyAction: "apply_development_outcome",
	}
	encoded, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(registryDir, "registry.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := transitions.LoadDir(registryDir)
	if err != nil {
		t.Fatal(err)
	}
	runner := New(registry, &developmentAgentManagerStub{}, transitionrun.NewFileStore(t.TempDir()), nil)
	adapter.Register(runner)
	runner.RegisterApply("apply_development_outcome", func(context.Context, string, Outcome) error { return nil })
	return runner, repository, owner, item
}

func TestDevelopmentAdapterRoutesRepeatedStartThroughOneCoordinatorReservation(t *testing.T) {
	runner, repository, owner, item := developmentRunnerFixture(t, "example")
	first, err := runner.Start(context.Background(), ContractDevelopmentTransition, item)
	if err != nil {
		t.Fatal(err)
	}
	second, err := runner.Start(context.Background(), ContractDevelopmentTransition, item)
	if err != nil {
		t.Fatal(err)
	}
	if first.ExecutionID != "owner-development-1" || second.ExecutionID != first.ExecutionID || owner.starts != 1 {
		t.Fatalf("starts first=%+v second=%+v owner_starts=%d", first, second, owner.starts)
	}
	if len(repository.state.Attempts) != 1 || repository.state.Attempts[0].ExecutionID != first.ExecutionID || repository.state.Reserved.Tokens != 1000 {
		t.Fatalf("development state=%+v", repository.state)
	}
	if repository.state.Campaign.ApprovalDigest != "approval-example" || repository.state.Campaign.AttemptKey == "" || repository.state.Campaign.OwnerExecutionID != first.ExecutionID || !repository.state.Campaign.Pending {
		t.Fatalf("campaign continuation state=%+v", repository.state.Campaign)
	}
	if first.WorkflowKey != "swarm-manager/contract-development" || first.DefinitionDigest != "sha256:workflow-development" {
		t.Fatalf("correlation=%+v", first)
	}
}

func TestDevelopmentAdapterKeepsScenarioTargetNeutral(t *testing.T) {
	runner, repository, owner, item := developmentRunnerFixture(t, "audio-tools")
	start, err := runner.Start(context.Background(), ContractDevelopmentTransition, item)
	if err != nil {
		t.Fatal(err)
	}
	if owner.starts != 1 || len(repository.state.Attempts) != 1 || repository.state.Campaign.ApprovalDigest != "approval-audio-tools" {
		t.Fatalf("scenario-neutral fixture did not use the same control plane: start=%+v state=%+v owner_starts=%d", start, repository.state, owner.starts)
	}
}
