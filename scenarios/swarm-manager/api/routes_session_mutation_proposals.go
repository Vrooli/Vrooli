package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/agentsessions"
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
		applier: applier, stateBuilder: newProposalStateBuilder(materializer, s.initStore, s.backlogHandler.Store()), resolveTarget: s.resolveSessionProposalTarget,
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
	state, err := p.stateBuilder(initiative)
	if err != nil {
		return agentsessions.MutationProposalIngestion{PayloadJSON: raw, ParseWarnings: warnings, ValidationErrors: []string{fmt.Sprintf("build initiative state for proposal validation: %v", err)}}, nil
	}
	normalized, err := proposals.Normalize(*proposal, state)
	if err == nil {
		err = proposals.Validate(normalized, state)
	}
	if err != nil {
		return agentsessions.MutationProposalIngestion{PayloadJSON: raw, ParseWarnings: warnings, ValidationErrors: []string{err.Error()}}, nil
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
	result, err := p.applier.ApplyFlow(ctx, proposal, p.stateBuilder, accepted, proposals.Source{InitiativeName: initiative, RunID: source.RunID, SessionID: source.SessionID, Entrypoint: "session.proposal", DecidedAtRFC3339: source.DecidedAt})
	if err != nil {
		return agentsessions.MutationProposalApplication{}, err
	}
	out := make([]agentsessions.MutationOutcome, 0, len(result.Outcomes))
	for _, outcome := range result.Outcomes {
		out = append(out, agentsessions.MutationOutcome{MutationID: outcome.MutationID, Op: string(outcome.Op), Target: outcome.Target, Applied: outcome.Applied, Skipped: outcome.Skipped, Error: outcome.Error})
	}
	return agentsessions.MutationProposalApplication{Outcomes: out}, nil
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
	return "", fmt.Errorf("backlog item %q has no owning initiative; attach it to an initiative before requesting a proposal", target.Ref)
}
