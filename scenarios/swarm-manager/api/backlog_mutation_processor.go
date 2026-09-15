package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/proposals"
)

// compositeMutationProcessor keeps agent-session proposal decisions on one
// durable path while dispatching to the domain that owns the target.
type compositeMutationProcessor struct {
	goal    *goalMutationProcessor
	backlog *backlogMutationProcessor
	capture *captureMutationProcessor
}

func newCompositeMutationProcessor(goalService *goals.Service, store backlog.Store, assigner *goals.BacklogMilestoneAssigner, lifecycle *backlog.Service) (*compositeMutationProcessor, error) {
	applier, err := proposals.NewApplier(proposals.Config{
		Store:         store,
		Assigner:      assigner,
		Creator:       lifecycle,
		ItemLifecycle: lifecycle,
		GoalCreator:   goalProposalCreator{service: goalService},
	})
	if err != nil {
		return nil, err
	}
	return &compositeMutationProcessor{
		goal:    newGoalMutationProcessor(goalService, lifecycle),
		backlog: &backlogMutationProcessor{store: store, goals: goalService, applier: applier},
		capture: &captureMutationProcessor{store: store, applier: applier},
	}, nil
}

func (p *compositeMutationProcessor) processor(target agentsessions.ProposalTarget) (agentsessions.MutationProposalProcessor, error) {
	switch target.Type {
	case agentsessions.ContextGoal:
		return p.goal, nil
	case agentsessions.ContextBacklogItem:
		return p.backlog, nil
	case agentsessions.ContextCapture:
		return p.capture, nil
	default:
		return nil, fmt.Errorf("mutation proposal target %q is not supported", target.Type)
	}
}

func (p *compositeMutationProcessor) Ingest(ctx context.Context, target agentsessions.ProposalTarget, reply string) (agentsessions.MutationProposalIngestion, error) {
	processor, err := p.processor(target)
	if err != nil {
		return agentsessions.MutationProposalIngestion{}, err
	}
	return processor.Ingest(ctx, target, reply)
}

func (p *compositeMutationProcessor) Apply(ctx context.Context, target agentsessions.ProposalTarget, payload string, accepted []string, source agentsessions.MutationProposalSource) (agentsessions.MutationProposalApplication, error) {
	processor, err := p.processor(target)
	if err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	return processor.Apply(ctx, target, payload, accepted, source)
}

func (p *compositeMutationProcessor) AcceptNoChange(ctx context.Context, target agentsessions.ProposalTarget, payload string, source agentsessions.MutationProposalSource) error {
	processor, err := p.processor(target)
	if err != nil {
		return err
	}
	return processor.AcceptNoChange(ctx, target, payload, source)
}

// backlogMutationProcessor is a small adapter around the canonical proposal
// applier. It owns only hydration of the current item/milestone state.
type backlogMutationProcessor struct {
	store   backlog.Store
	goals   *goals.Service
	applier *proposals.Applier
}

func (p *backlogMutationProcessor) Ingest(ctx context.Context, target agentsessions.ProposalTarget, reply string) (agentsessions.MutationProposalIngestion, error) {
	payload, warnings := extractGoalMutationEnvelope(reply)
	if payload == "" {
		return agentsessions.MutationProposalIngestion{ParseWarnings: warnings}, nil
	}
	if err := p.validate(ctx, target, payload); err != nil {
		return agentsessions.MutationProposalIngestion{PayloadJSON: payload, ValidationErrors: []string{err.Error()}}, nil
	}
	return agentsessions.MutationProposalIngestion{PayloadJSON: payload, ParseWarnings: warnings}, nil
}

func (p *backlogMutationProcessor) Apply(ctx context.Context, target agentsessions.ProposalTarget, payload string, accepted []string, source agentsessions.MutationProposalSource) (agentsessions.MutationProposalApplication, error) {
	var proposal proposals.Proposal
	if err := json.Unmarshal([]byte(payload), &proposal); err != nil {
		return agentsessions.MutationProposalApplication{}, fmt.Errorf("decode backlog proposal: %w", err)
	}
	state, err := p.state(target)
	if err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	result, err := p.applier.Apply(ctx, proposal, state, accepted, proposals.Source{
		MilestoneName:    state.MilestoneName,
		SessionID:        source.SessionID,
		RunID:            source.RunID,
		DecidedAtRFC3339: source.DecidedAt,
		Entrypoint:       "session.proposal",
	})
	if err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	outcomes := make([]agentsessions.MutationOutcome, 0, len(result.Outcomes))
	for _, outcome := range result.Outcomes {
		outcomes = append(outcomes, agentsessions.MutationOutcome{MutationID: outcome.MutationID, Op: string(outcome.Op), Target: outcome.Target, Applied: outcome.Applied, Skipped: outcome.Skipped, Error: outcome.Error})
	}
	return agentsessions.MutationProposalApplication{Outcomes: outcomes}, nil
}

func (p *backlogMutationProcessor) AcceptNoChange(ctx context.Context, target agentsessions.ProposalTarget, payload string, _ agentsessions.MutationProposalSource) error {
	return p.validate(ctx, target, payload)
}

func (p *backlogMutationProcessor) validate(_ context.Context, target agentsessions.ProposalTarget, payload string) error {
	var proposal proposals.Proposal
	if err := json.Unmarshal([]byte(payload), &proposal); err != nil {
		return fmt.Errorf("decode backlog proposal: %w", err)
	}
	state, err := p.state(target)
	if err != nil {
		return err
	}
	return proposals.Validate(proposal, state)
}

func (p *backlogMutationProcessor) state(target agentsessions.ProposalTarget) (proposals.CurrentState, error) {
	parts := strings.SplitN(strings.TrimSpace(target.Ref), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return proposals.CurrentState{}, fmt.Errorf("invalid backlog proposal target %q", target.Ref)
	}
	kind, err := backlog.ParseBacklogKind(parts[0])
	if err != nil {
		return proposals.CurrentState{}, err
	}
	item, err := p.store.LoadItem(kind, parts[1])
	if err != nil {
		return proposals.CurrentState{}, err
	}
	goalList, err := p.goals.List()
	if err != nil {
		return proposals.CurrentState{}, err
	}
	milestone, milestoneItems, err := resolveBacklogProposalMilestone(target.Ref, item.Milestone, goalList)
	if err != nil {
		return proposals.CurrentState{}, err
	}
	all, err := p.store.LoadAll(nil)
	if err != nil {
		return proposals.CurrentState{}, err
	}
	state := proposals.CurrentState{MilestoneName: milestone, Nodes: map[string]proposals.GraphNode{}, KnownMilestones: map[string]struct{}{}, InProgressRefs: map[string]struct{}{}}
	for _, candidate := range all {
		ref := string(candidate.Kind) + "/" + candidate.Name
		_, listedInMilestone := milestoneItems[ref]
		if candidate.Milestone != milestone && !listedInMilestone {
			continue
		}
		state.Nodes[ref] = proposals.GraphNode{ID: ref, Kind: string(candidate.Kind), Name: candidate.Name, Title: candidate.Title, Priority: candidate.Priority, Effort: candidate.Effort}
		if candidate.Status == backlog.StatusInProgress {
			state.InProgressRefs[ref] = struct{}{}
		}
		for _, dependency := range candidate.DependsOn {
			state.Edges = append(state.Edges, proposals.GraphEdge{From: ref, To: dependency})
		}
	}
	for _, listed := range goalList {
		for _, candidate := range listed.Goal.Milestones {
			state.KnownMilestones[listed.Goal.Name+"/"+candidate.Name] = struct{}{}
		}
	}
	return state, nil
}

func resolveBacklogProposalMilestone(targetRef, storedMilestone string, goalList []goals.GoalWithScope) (string, map[string]struct{}, error) {
	storedMilestone = strings.TrimSpace(storedMilestone)
	matches := make(map[string]map[string]struct{})
	for _, listed := range goalList {
		for _, milestone := range listed.Goal.Milestones {
			qualified := listed.Goal.Name + "/" + milestone.Name
			members := make(map[string]struct{}, len(milestone.Items))
			for _, ref := range milestone.Items {
				members[strings.TrimSpace(ref)] = struct{}{}
			}
			if storedMilestone == qualified {
				return qualified, members, nil
			}
			if _, ok := members[targetRef]; ok {
				matches[qualified] = members
			}
		}
	}
	if storedMilestone != "" {
		return storedMilestone, nil, nil
	}
	if len(matches) == 0 {
		return "", nil, fmt.Errorf("backlog proposal target %q is not assigned to a goal milestone", targetRef)
	}
	if len(matches) > 1 {
		return "", nil, fmt.Errorf("backlog proposal target %q is assigned to multiple goal milestones", targetRef)
	}
	for milestone, members := range matches {
		return milestone, members, nil
	}
	panic("unreachable")
}
