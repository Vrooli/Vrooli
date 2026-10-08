package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/proposals"
)

// captureMutationProcessor applies the deliberately small mutation vocabulary
// emitted by capture classification. It reuses the canonical proposal applier
// so capture intake gets the same validation, provenance, rollback, and graph
// invalidation semantics as every other proposal surface.
type captureMutationProcessor struct {
	store   backlog.Store
	applier *proposals.Applier
}

func (p *captureMutationProcessor) Ingest(ctx context.Context, target agentsessions.ProposalTarget, reply string) (agentsessions.MutationProposalIngestion, error) {
	if target.Type != agentsessions.ContextCapture {
		return agentsessions.MutationProposalIngestion{}, fmt.Errorf("capture mutation processor does not support target %q", target.Type)
	}
	// Capture proposals are already structured by the classification workflow.
	// This method exists only to satisfy the shared session processor contract.
	var proposal proposals.Proposal
	if err := json.Unmarshal([]byte(reply), &proposal); err != nil {
		return agentsessions.MutationProposalIngestion{ValidationErrors: []string{err.Error()}}, nil
	}
	if err := p.validate(target, proposal); err != nil {
		return agentsessions.MutationProposalIngestion{PayloadJSON: reply, ValidationErrors: []string{err.Error()}}, nil
	}
	return agentsessions.MutationProposalIngestion{PayloadJSON: reply}, nil
}

func (p *captureMutationProcessor) Apply(ctx context.Context, target agentsessions.ProposalTarget, payload string, accepted []string, source agentsessions.MutationProposalSource) (agentsessions.MutationProposalApplication, error) {
	if target.Type != agentsessions.ContextCapture {
		return agentsessions.MutationProposalApplication{}, fmt.Errorf("capture mutation processor does not support target %q", target.Type)
	}
	var proposal proposals.Proposal
	if err := json.Unmarshal([]byte(payload), &proposal); err != nil {
		return agentsessions.MutationProposalApplication{}, fmt.Errorf("decode capture proposal: %w", err)
	}
	if err := p.validate(target, proposal); err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	// Create items before a create_goal mutation. A classifier may reasonably
	// list the goal first, but goals.Service computes target closure at create
	// time and therefore needs the proposed items on disk already.
	proposal.Mutations = intakeOrder(proposal.Mutations)
	state, err := p.state(target.Ref)
	if err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	result, err := p.applier.Apply(ctx, proposal, state, accepted, proposals.Source{
		MilestoneName:    "capture/" + strings.TrimSpace(target.Ref),
		SessionID:        source.SessionID,
		RunID:            source.RunID,
		DecidedAtRFC3339: source.DecidedAt,
		Entrypoint:       "capture.intake",
	})
	if err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	out := make([]agentsessions.MutationOutcome, 0, len(result.Outcomes))
	for _, outcome := range result.Outcomes {
		out = append(out, agentsessions.MutationOutcome{MutationID: outcome.MutationID, Op: string(outcome.Op), Target: outcome.Target, Applied: outcome.Applied, Skipped: outcome.Skipped, Error: outcome.Error})
	}
	return agentsessions.MutationProposalApplication{Outcomes: out}, nil
}

func (p *captureMutationProcessor) AcceptNoChange(_ context.Context, target agentsessions.ProposalTarget, payload string, _ agentsessions.MutationProposalSource) error {
	if target.Type != agentsessions.ContextCapture {
		return fmt.Errorf("capture mutation processor does not support target %q", target.Type)
	}
	var proposal proposals.Proposal
	if err := json.Unmarshal([]byte(payload), &proposal); err != nil {
		return fmt.Errorf("decode capture proposal: %w", err)
	}
	return p.validate(target, proposal)
}

func (p *captureMutationProcessor) validate(_ agentsessions.ProposalTarget, proposal proposals.Proposal) error {
	for _, mutation := range proposal.Mutations {
		if mutation.Op != proposals.OpAddItem && mutation.Op != proposals.OpCreateGoal {
			return fmt.Errorf("capture proposal operation %q is not allowed", mutation.Op)
		}
	}
	state, err := captureState(p.store, "capture")
	if err != nil {
		return err
	}
	return proposals.Validate(proposal, state)
}

func (p *captureMutationProcessor) state(ref string) (proposals.CurrentState, error) {
	return captureState(p.store, "capture/"+strings.TrimSpace(ref))
}

func captureState(store backlog.Store, milestoneName string) (proposals.CurrentState, error) {
	if store == nil {
		return proposals.CurrentState{}, fmt.Errorf("capture proposal store is not configured")
	}
	all, err := store.LoadAll(nil)
	if err != nil {
		return proposals.CurrentState{}, err
	}
	state := proposals.CurrentState{MilestoneName: milestoneName, Nodes: map[string]proposals.GraphNode{}, KnownMilestones: map[string]struct{}{}, InProgressRefs: map[string]struct{}{}}
	for _, item := range all {
		ref := backlog.ItemRef(item)
		state.Nodes[ref] = proposals.GraphNode{ID: ref, Kind: string(item.Kind), Name: item.Name, Title: item.Title, Priority: item.Priority, Effort: item.Effort}
		if item.Status == backlog.StatusInProgress {
			state.InProgressRefs[ref] = struct{}{}
		}
		for _, dependency := range item.DependsOn {
			state.Edges = append(state.Edges, proposals.GraphEdge{From: ref, To: dependency})
		}
	}
	return state, nil
}

func intakeOrder(mutations []proposals.Mutation) []proposals.Mutation {
	ordered := make([]proposals.Mutation, 0, len(mutations))
	for _, mutation := range mutations {
		if mutation.Op == proposals.OpAddItem {
			ordered = append(ordered, mutation)
		}
	}
	for _, mutation := range mutations {
		if mutation.Op == proposals.OpCreateGoal {
			ordered = append(ordered, mutation)
		}
	}
	return ordered
}
