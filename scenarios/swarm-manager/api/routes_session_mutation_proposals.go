package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/proposals"
)

// wireSessionMutationProposals composes session lifecycle with the existing
// graph-backed proposal validation and ApplyFlow recipe. Sessions deliberately
// do not own a generation lock: fresh-state ApplyFlow is the stale-work guard.
func (s *Server) wireSessionMutationProposals(materializer *graph.Materializer) {
	if s.agentSessionSvc == nil || s.backlogHandler == nil || s.initStore == nil {
		return
	}
	applier, err := s.buildProposalApplier(materializer)
	if err != nil {
		slog.Warn("session proposals: build applier", "error", err)
		return
	}
	s.agentSessionSvc.SetMutationProposalProcessor(&sessionMutationProposalProcessor{
		applier: applier, stateBuilder: newSessionProposalStateBuilder(materializer, s.initStore, s.backlogHandler.Store()), resolveTarget: s.resolveSessionProposalTarget,
	})
}

type sessionMutationProposalProcessor struct {
	applier       *proposals.Applier
	stateBuilder  proposals.StateBuilder
	resolveTarget func(agentsessions.ProposalTarget) (string, error)
}

func (p *sessionMutationProposalProcessor) Ingest(_ context.Context, target agentsessions.ProposalTarget, assistantReply string) (agentsessions.MutationProposalIngestion, error) {
	proposal, raw, warnings := proposals.Extract(assistantReply)
	if proposal == nil {
		if len(warnings) == 0 {
			warnings = []string{"agent output did not contain a parseable proposal JSON block"}
		}
		return agentsessions.MutationProposalIngestion{PayloadJSON: `{}`, ParseWarnings: warnings}, nil
	}
	initiative, err := p.resolveTarget(target)
	if err != nil {
		return agentsessions.MutationProposalIngestion{PayloadJSON: raw, ParseWarnings: warnings, ValidationErrors: []string{err.Error()}}, nil
	}
	groups, err := p.scopedProposals(*proposal, initiative)
	if err != nil {
		return agentsessions.MutationProposalIngestion{PayloadJSON: raw, ParseWarnings: warnings, ValidationErrors: []string{err.Error()}}, nil
	}
	for scope, scoped := range groups {
		state, err := p.stateBuilder(scope)
		if err != nil {
			return agentsessions.MutationProposalIngestion{PayloadJSON: raw, ParseWarnings: warnings, ValidationErrors: []string{fmt.Sprintf("build proposal state for validation: %v", err)}}, nil
		}
		normalized, err := proposals.Normalize(scoped, state)
		if err == nil {
			err = proposals.Validate(normalized, state)
		}
		if err != nil {
			return agentsessions.MutationProposalIngestion{PayloadJSON: raw, ParseWarnings: warnings, ValidationErrors: []string{err.Error()}}, nil
		}
	}
	return agentsessions.MutationProposalIngestion{PayloadJSON: raw, ParseWarnings: warnings}, nil
}

func (p *sessionMutationProposalProcessor) Apply(ctx context.Context, target agentsessions.ProposalTarget, payloadJSON string, accepted []string, source agentsessions.MutationProposalSource) (agentsessions.MutationProposalApplication, error) {
	var proposal proposals.Proposal
	if err := json.Unmarshal([]byte(payloadJSON), &proposal); err != nil {
		return agentsessions.MutationProposalApplication{}, fmt.Errorf("parse stored mutation proposal: %w", err)
	}
	initiative, err := p.resolveTarget(target)
	if err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	// A mutation list can legitimately touch several independently-attached
	// items. Revalidate and apply each ownership scope separately so stale
	// work on one item cannot invalidate an unrelated decision.
	groups, err := p.scopedProposals(proposal, initiative)
	if err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	byID := make(map[string]proposals.Outcome, len(proposal.Mutations))
	for scope, scoped := range groups {
		result, err := p.applier.ApplyFlow(ctx, scoped, p.stateBuilder, accepted, proposals.Source{InitiativeName: scope, RunID: source.RunID, SessionID: source.SessionID, Entrypoint: "session.proposal", DecidedAtRFC3339: source.DecidedAt})
		if err != nil {
			return agentsessions.MutationProposalApplication{}, err
		}
		for _, outcome := range result.Outcomes {
			byID[outcome.MutationID] = outcome
		}
	}
	out := make([]agentsessions.MutationOutcome, 0, len(proposal.Mutations))
	for _, mutation := range proposal.Mutations {
		outcome, ok := byID[mutation.ID]
		if !ok {
			continue
		}
		out = append(out, agentsessions.MutationOutcome{MutationID: outcome.MutationID, Op: string(outcome.Op), Target: outcome.Target, Applied: outcome.Applied, Skipped: outcome.Skipped, Error: outcome.Error})
	}
	return agentsessions.MutationProposalApplication{Outcomes: out}, nil
}

// scopedProposals partitions a mutation list by the object it mutates. Full
// graph proposals retain their single initiative target: a graph is by
// definition one initiative's complete state.
func (p *sessionMutationProposalProcessor) scopedProposals(proposal proposals.Proposal, fallback string) (map[string]proposals.Proposal, error) {
	if proposal.Form != proposals.FormMutationList {
		return map[string]proposals.Proposal{fallback: proposal}, nil
	}
	groups := make(map[string]proposals.Proposal)
	for _, mutation := range proposal.Mutations {
		scope := fallback
		if ref := mutationScopeRef(mutation); ref != "" {
			resolved, err := p.resolveTarget(agentsessions.ProposalTarget{Type: agentsessions.ContextBacklogItem, Ref: ref, Name: ref})
			if err != nil {
				return nil, fmt.Errorf("resolve mutation %q target %q: %w", mutation.ID, ref, err)
			}
			scope = resolved
		}
		group := groups[scope]
		if group.Form == "" {
			group = proposals.Proposal{Form: proposals.FormMutationList, Rationale: proposal.Rationale}
		}
		group.Mutations = append(group.Mutations, mutation)
		groups[scope] = group
	}
	return groups, nil
}

func mutationScopeRef(m proposals.Mutation) string {
	// Initiative recreation is scoped by the proposal session's initiative.
	// Its Target intentionally contains an initiative name, not a backlog ref.
	if m.Op == proposals.OpRecreateInitiative {
		return ""
	}
	if strings.TrimSpace(m.Target) != "" {
		return strings.TrimSpace(m.Target)
	}
	if strings.TrimSpace(m.From) != "" {
		return strings.TrimSpace(m.From)
	}
	if len(m.Sources) > 0 {
		return strings.TrimSpace(m.Sources[0])
	}
	return ""
}

func (s *Server) resolveSessionProposalTarget(target agentsessions.ProposalTarget) (string, error) {
	if target.Type == agentsessions.ContextInitiative {
		if _, err := s.initStore.Load(strings.TrimSpace(target.Ref)); err != nil {
			return "", fmt.Errorf("proposal initiative target %q is unavailable: %w", target.Ref, err)
		}
		return strings.TrimSpace(target.Ref), nil
	}
	if target.Type != agentsessions.ContextBacklogItem {
		return "", fmt.Errorf("proposal target type %q is not supported", target.Type)
	}
	initiatives, err := s.initStore.LoadAll()
	if err != nil {
		return "", fmt.Errorf("load initiatives for backlog proposal target: %w", err)
	}
	for _, initiative := range initiatives {
		for _, ref := range initiative.Items {
			if strings.TrimSpace(ref) == strings.TrimSpace(target.Ref) {
				return initiative.Name, nil
			}
		}
	}
	ref := strings.TrimSpace(target.Ref)
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("invalid standalone backlog proposal target %q", target.Ref)
	}
	kind, err := backlog.ParseBacklogKind(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid standalone backlog proposal target %q: %w", ref, err)
	}
	if _, err := s.backlogHandler.Store().LoadItem(kind, parts[1]); err != nil {
		return "", fmt.Errorf("proposal backlog target %q is unavailable: %w", ref, err)
	}
	return standaloneProposalScopePrefix + ref, nil
}
