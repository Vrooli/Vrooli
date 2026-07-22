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

// GoalMutationProcessor adapts goal-scoped session mutation proposals to the
// existing goals.Service mutation methods. It keeps the session proposal store
// as the single operator decision inbox.
type goalMutationProcessor struct{ service *goals.Service }

func newGoalMutationProcessor(service *goals.Service) *goalMutationProcessor {
	return &goalMutationProcessor{service: service}
}

func (p *goalMutationProcessor) Ingest(ctx context.Context, target agentsessions.ProposalTarget, assistantReply string) (agentsessions.MutationProposalIngestion, error) {
	if target.Type != agentsessions.ContextGoal {
		return agentsessions.MutationProposalIngestion{}, fmt.Errorf("goal mutation processor does not support target %q", target.Type)
	}
	payload, warnings := extractGoalMutationEnvelope(assistantReply)
	if payload == "" {
		return agentsessions.MutationProposalIngestion{ParseWarnings: warnings}, nil
	}
	if err := p.validatePayload(ctx, target.Ref, payload); err != nil {
		return agentsessions.MutationProposalIngestion{PayloadJSON: payload, ValidationErrors: []string{err.Error()}}, nil
	}
	return agentsessions.MutationProposalIngestion{PayloadJSON: payload, ParseWarnings: warnings}, nil
}

func (p *goalMutationProcessor) Apply(ctx context.Context, target agentsessions.ProposalTarget, payloadJSON string, accepted []string, _ agentsessions.MutationProposalSource) (agentsessions.MutationProposalApplication, error) {
	if target.Type != agentsessions.ContextGoal {
		return agentsessions.MutationProposalApplication{}, fmt.Errorf("goal mutation processor does not support target %q", target.Type)
	}
	var proposal proposals.Proposal
	if err := json.Unmarshal([]byte(payloadJSON), &proposal); err != nil {
		return agentsessions.MutationProposalApplication{}, fmt.Errorf("decode goal proposal: %w", err)
	}
	state, err := p.state(ctx, target.Ref)
	if err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	if err := proposals.ValidateGoal(proposal, state); err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	acceptedSet := map[string]struct{}{}
	if accepted != nil {
		for _, id := range accepted {
			acceptedSet[strings.TrimSpace(id)] = struct{}{}
		}
	}
	out := make([]agentsessions.MutationOutcome, 0, len(proposal.Mutations))
	for _, mutation := range proposal.Mutations {
		result := agentsessions.MutationOutcome{MutationID: mutation.ID, Op: string(mutation.Op), Target: target.Ref}
		if accepted != nil {
			if _, ok := acceptedSet[mutation.ID]; !ok {
				result.Skipped = true
				out = append(out, result)
				continue
			}
		}
		if err := p.applyOne(ctx, target.Ref, mutation); err != nil {
			result.Error = err.Error()
		} else {
			result.Applied = true
		}
		out = append(out, result)
	}
	return agentsessions.MutationProposalApplication{Outcomes: out}, nil
}

func (p *goalMutationProcessor) AcceptNoChange(ctx context.Context, target agentsessions.ProposalTarget, payloadJSON string, _ agentsessions.MutationProposalSource) error {
	if target.Type != agentsessions.ContextGoal {
		return fmt.Errorf("goal mutation processor does not support target %q", target.Type)
	}
	return p.validatePayload(ctx, target.Ref, payloadJSON)
}

func (p *goalMutationProcessor) validatePayload(ctx context.Context, goalName, payload string) error {
	var proposal proposals.Proposal
	if err := json.Unmarshal([]byte(payload), &proposal); err != nil {
		return fmt.Errorf("decode goal proposal: %w", err)
	}
	state, err := p.state(ctx, goalName)
	if err != nil {
		return err
	}
	return proposals.ValidateGoal(proposal, state)
}

func (p *goalMutationProcessor) state(_ context.Context, goalName string) (proposals.GoalState, error) {
	goal, err := p.service.Get(goalName)
	if err != nil {
		return proposals.GoalState{}, err
	}
	state := proposals.GoalState{Name: goal.Goal.Name, Version: goal.Goal.Updated, Milestones: map[string]proposals.GoalMilestoneState{}, Closure: map[string]struct{}{}, Targets: map[string]struct{}{}}
	for _, ref := range goal.Scope.Closure {
		state.Closure[ref] = struct{}{}
	}
	for _, target := range goal.Goal.Targets {
		state.Targets[target] = struct{}{}
	}
	for _, milestone := range goal.Goal.Milestones {
		items := map[string]struct{}{}
		open := false
		for _, ref := range milestone.Items {
			items[ref] = struct{}{}
			if item, ok := goal.ScopeEntities.Items[ref]; ok && !backlog.IsTerminalStatus(item.Status) {
				open = true
			}
		}
		state.Milestones[milestone.Name] = proposals.GoalMilestoneState{Items: items, Open: open, Archive: milestone.ArchivedAt != nil}
	}
	return state, nil
}

func (p *goalMutationProcessor) applyOne(_ context.Context, goalName string, m proposals.Mutation) error {
	switch m.Op {
	case proposals.OpCreateMilestone:
		_, err := p.service.CreateMilestone(goalName, milestoneFromGoalProposal(*m.GoalMilestone))
		return err
	case proposals.OpUpdateMilestone:
		current, err := p.service.Get(goalName)
		if err != nil {
			return err
		}
		previous, found := findGoalMilestone(current.Goal.Milestones, m.GoalMilestone.Name)
		if !found {
			return fmt.Errorf("milestone %q not found", m.GoalMilestone.Name)
		}
		updated := milestoneFromGoalProposal(*m.GoalMilestone)
		updated.Items, updated.ArchivedAt = previous.Items, previous.ArchivedAt
		_, err = p.service.UpdateMilestone(goalName, updated)
		return err
	case proposals.OpArchiveMilestone:
		if m.DetachOpen {
			state, err := p.service.Get(goalName)
			if err != nil {
				return err
			}
			if previous, found := findGoalMilestone(state.Goal.Milestones, m.MilestoneName); found {
				if _, err = p.service.UnassignMilestoneItems(goalName, m.MilestoneName, previous.Items); err != nil {
					return err
				}
			}
		}
		_, err := p.service.ArchiveMilestone(goalName, m.MilestoneName)
		return err
	case proposals.OpAssignMilestoneItems:
		_, err := p.service.AssignMilestoneItems(goalName, m.MilestoneName, m.Items)
		return err
	case proposals.OpUnassignMilestoneItems:
		_, err := p.service.UnassignMilestoneItems(goalName, m.MilestoneName, m.Items)
		return err
	case proposals.OpAddGoalTarget:
		_, err := p.service.AddTargets(goalName, m.Targets)
		return err
	case proposals.OpRemoveGoalTarget:
		_, err := p.service.RemoveTargets(goalName, m.Targets)
		return err
	default:
		return fmt.Errorf("unsupported goal mutation %q", m.Op)
	}
}

func milestoneFromGoalProposal(m proposals.GoalMilestone) goals.Milestone {
	return goals.Milestone{Name: m.Name, Title: m.Title, Description: m.Description, AcceptanceCriteria: m.AcceptanceCriteria, DependsOn: m.DependsOn}
}

func findGoalMilestone(milestones []goals.Milestone, name string) (goals.Milestone, bool) {
	for _, milestone := range milestones {
		if milestone.Name == name {
			return milestone, true
		}
	}
	return goals.Milestone{}, false
}

func extractGoalMutationEnvelope(reply string) (string, []string) {
	start := strings.Index(reply, "```json")
	if start < 0 {
		return "", []string{"agent output did not contain a fenced JSON mutation_list envelope"}
	}
	body := reply[start+len("```json"):]
	end := strings.Index(body, "```")
	if end < 0 {
		return "", []string{"agent output contains an unterminated JSON fence"}
	}
	return strings.TrimSpace(body[:end]), nil
}
