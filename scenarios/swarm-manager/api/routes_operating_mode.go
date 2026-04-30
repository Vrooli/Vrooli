package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/proposals"
)

func (s *Server) registerOperatingModeRoutes(scenarioRoot string, materializer *graph.Materializer) {
	if s.initStore == nil || s.initiativeService == nil || s.backlogHandler == nil {
		return
	}
	applier, err := s.buildProposalApplier(materializer)
	if err != nil {
		slog.Warn("operating-mode: proposal applier unavailable; mutation reconciliation disabled", "err", err)
	}
	store := operatingmode.NewStore(s.initiativeService.InitDir)
	lock := &initiativelock.Lock{Dir: s.initiativeService.InitDir}
	svc, err := operatingmode.NewService(operatingmode.Config{
		Store:       store,
		Lock:        lock,
		Initiatives: operatingModeInitiativeReader{store: s.initStore},
		ModeUpdater: operatingModeUpdater{service: s.initiativeService},
		Backlog:     operatingModeBacklogReader{store: s.backlogHandler.Store()},
		BacklogMutator: operatingModeBacklogMutator{
			store:  s.backlogHandler.Store(),
			events: s.emitter,
		},
		Reconciler: operatingModeProposalReconciler{
			applier:      applier,
			stateBuilder: newFeedbackStateBuilder(materializer, s.initStore, s.backlogHandler.Store()),
		},
		ItemExecutions: operatingModeExecutionController{
			service: s.executionSvc,
		},
		Agent:        s.agentSvc,
		Activity:     s.agentActivitySvc,
		PromptClient: promptmanager.NewHTTPClient(),
		Events:       s.emitter,
		ScenarioRoot: scenarioRoot,
	})
	if err != nil {
		slog.Warn("operating-mode: failed to build Service", "err", err)
		return
	}
	operatingmode.NewHandler(svc).RegisterRoutes(s.router)
}

type operatingModeProposalReconciler struct {
	applier      *proposals.Applier
	stateBuilder func(string) (proposals.CurrentState, error)
}

func (r operatingModeProposalReconciler) ApplyBacklogSyncProposal(ctx context.Context, req operatingmode.ProposalReconcileRequest) (*operatingmode.ProposalApplyResult, error) {
	if r.applier == nil {
		return nil, fmt.Errorf("proposal applier is not configured")
	}
	if r.stateBuilder == nil {
		return nil, fmt.Errorf("proposal state builder is not configured")
	}
	var proposal proposals.Proposal
	if err := json.Unmarshal(req.Proposal, &proposal); err != nil {
		return nil, fmt.Errorf("parse backlog_sync proposal: %w", err)
	}
	state, err := r.stateBuilder(req.InitiativeName)
	if err != nil {
		return nil, fmt.Errorf("build proposal state: %w", err)
	}
	normalized, err := proposals.Normalize(proposal, state)
	if err != nil {
		return nil, fmt.Errorf("normalize proposal: %w", err)
	}
	result, err := r.applier.Apply(ctx, normalized, state, req.AcceptedMutationIDs, proposals.Source{
		InitiativeName:   req.InitiativeName,
		RoundNumber:      req.Round,
		RoundSlug:        req.Phase,
		Entrypoint:       "initiative.operating_mode.backlog_sync",
		DecidedBy:        req.DecidedBy,
		DecidedAtRFC3339: req.DecidedAtRFC3339,
	})
	if err != nil {
		return nil, err
	}
	return operatingModeApplyResult(result, normalized), nil
}

func operatingModeApplyResult(result *proposals.ApplyResult, proposal proposals.Proposal) *operatingmode.ProposalApplyResult {
	if result == nil {
		return nil
	}
	out := &operatingmode.ProposalApplyResult{
		Outcomes: make([]operatingmode.ProposalOutcome, 0, len(result.Outcomes)),
		Applied:  result.Applied,
		Failed:   result.Failed,
		Skipped:  result.Skipped,
	}
	applied := make(map[string]bool, len(result.Outcomes))
	for _, outcome := range result.Outcomes {
		if outcome.Applied {
			applied[outcome.MutationID] = true
		}
		out.Outcomes = append(out.Outcomes, operatingmode.ProposalOutcome{
			MutationID: outcome.MutationID,
			Op:         string(outcome.Op),
			Target:     outcome.Target,
			Applied:    outcome.Applied,
			Skipped:    outcome.Skipped,
			Error:      outcome.Error,
		})
	}
	for _, mutation := range proposal.Mutations {
		if !applied[mutation.ID] {
			continue
		}
		switch mutation.Op {
		case proposals.OpAddItem, proposals.OpSplitItem, proposals.OpMergeItems:
			out.Created++
		default:
			out.Updated++
		}
	}
	return out
}

type operatingModeBacklogReader struct {
	store backlog.Store
}

func (r operatingModeBacklogReader) LoadBacklogItem(kind, name string) (operatingmode.BacklogItemSnapshot, error) {
	item, err := r.store.LoadItem(backlog.BacklogKind(kind), name)
	if err != nil {
		return operatingmode.BacklogItemSnapshot{}, err
	}
	return operatingmode.BacklogItemSnapshot{
		Title:    item.Title,
		Status:   string(item.Status),
		Priority: item.Priority,
		Effort:   item.Effort,
	}, nil
}

type operatingModeBacklogMutator struct {
	store  backlog.Store
	events interface {
		EmitBacklogStatusChanged(entityID, from, to string)
	}
}

func (m operatingModeBacklogMutator) MarkBacklogItemCompleted(_ context.Context, kind, name, source string) (operatingmode.BacklogCompletionResult, error) {
	item, err := m.store.LoadItem(backlog.BacklogKind(kind), name)
	if err != nil {
		return operatingmode.BacklogCompletionResult{}, err
	}
	prior := item.Status
	if prior != backlog.StatusCompleted {
		item.Status = backlog.StatusCompleted
		if err := m.store.SaveItem(item); err != nil {
			return operatingmode.BacklogCompletionResult{}, err
		}
		if m.events != nil {
			m.events.EmitBacklogStatusChanged(kind+"/"+name, string(prior), string(item.Status))
		}
	}
	_ = source
	return operatingmode.BacklogCompletionResult{
		ItemRef:    kind + "/" + name,
		FromStatus: string(prior),
		ToStatus:   string(backlog.StatusCompleted),
	}, nil
}

type operatingModeInitiativeReader struct {
	store *initiatives.Store
}

func (r operatingModeInitiativeReader) LoadInitiative(name string) (operatingmode.InitiativeSnapshot, error) {
	init, err := r.store.Load(name)
	if err != nil {
		return operatingmode.InitiativeSnapshot{}, err
	}
	return operatingmode.InitiativeSnapshot{
		Name:               init.Name,
		Title:              init.Title,
		Description:        init.Description,
		Mode:               init.Mode,
		Items:              append([]string(nil), init.Items...),
		AcceptanceCriteria: append([]string(nil), init.AcceptanceCriteria...),
	}, nil
}

type operatingModeUpdater struct {
	service *initiatives.Service
}

func (u operatingModeUpdater) UpdateInitiativeMode(name, mode string) (operatingmode.InitiativeSnapshot, error) {
	if u.service == nil {
		return operatingmode.InitiativeSnapshot{}, nil
	}
	updated, err := u.service.Update(name, initiatives.UpdateRequest{Mode: &mode})
	if err != nil {
		return operatingmode.InitiativeSnapshot{}, err
	}
	return operatingmode.InitiativeSnapshot{
		Name:               updated.Name,
		Title:              updated.Title,
		Description:        updated.Description,
		Mode:               updated.Mode,
		Items:              append([]string(nil), updated.Items...),
		AcceptanceCriteria: append([]string(nil), updated.AcceptanceCriteria...),
	}, nil
}

type operatingModeExecutionController struct {
	service *execution.Service
}

func (c operatingModeExecutionController) ActiveExecutionsForInitiative(ctx context.Context, init operatingmode.InitiativeSnapshot) ([]operatingmode.ActiveItemExecution, error) {
	if c.service == nil {
		return nil, nil
	}
	var active []operatingmode.ActiveItemExecution
	for _, ref := range init.Items {
		parts := strings.SplitN(strings.TrimSpace(ref), "/", 2)
		if len(parts) != 2 {
			continue
		}
		records, err := c.service.List(ctx, execution.ListFilters{
			BacklogKind: parts[0],
			BacklogName: parts[1],
		})
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			if !isOperatingModeCancelableStatus(record.Status) {
				continue
			}
			active = append(active, operatingmode.ActiveItemExecution{
				ItemRef:     ref,
				ExecutionID: record.ExecutionID,
				RunID:       record.RunID,
				Status:      string(record.Status),
			})
		}
	}
	return active, nil
}

func (c operatingModeExecutionController) CancelActiveExecutionsForInitiative(ctx context.Context, init operatingmode.InitiativeSnapshot) ([]operatingmode.ActiveItemExecution, error) {
	active, err := c.ActiveExecutionsForInitiative(ctx, init)
	if err != nil {
		return nil, err
	}
	canceled := make([]operatingmode.ActiveItemExecution, 0, len(active))
	for _, item := range active {
		if strings.TrimSpace(item.ExecutionID) == "" {
			continue
		}
		record, err := c.service.Cancel(ctx, item.ExecutionID)
		if err != nil {
			return canceled, err
		}
		item.Status = string(record.Status)
		item.RunID = record.RunID
		canceled = append(canceled, item)
	}
	return canceled, nil
}

func isOperatingModeCancelableStatus(status execution.Status) bool {
	switch status {
	case execution.StatusPending, execution.StatusStarting, execution.StatusRunning,
		execution.StatusNeedsReview, execution.StatusValidating, execution.StatusNeedsFixup:
		return true
	default:
		return false
	}
}
