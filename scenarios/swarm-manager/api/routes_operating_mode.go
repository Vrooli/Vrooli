package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"path/filepath"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/proposals"
)

func (s *Server) registerOperatingModeRoutes(scenarioRoot string, materializer *graph.Materializer) {
	if s.initStore == nil || s.initiativeService == nil || s.backlogHandler == nil {
		log.Fatalf("operating-mode: required initiative/backlog services are not registered")
	}
	applier, err := s.buildProposalApplier(materializer)
	if err != nil {
		slog.Warn("operating-mode: proposal applier unavailable; mutation reconciliation disabled", "err", err)
	}
	store := operatingmode.NewStore(s.initiativeService.InitDir)
	lock := &initiativelock.Lock{Dir: s.initiativeService.InitDir}
	overlayPath := filepath.Join(scenarioRoot, ".vrooli", "operating-modes", "overrides.json")
	overlay := operatingmode.NewOverlayStore(overlayPath)
	svc, err := operatingmode.NewService(operatingmode.Config{
		Store:            store,
		Overlay:          overlay,
		Lock:             lock,
		Initiatives:      operatingModeInitiativeReader{store: s.initStore},
		InitiativeLister: operatingModeInitiativeLister{store: s.initStore},
		ModeUpdater:      operatingModeUpdater{service: s.initiativeService},
		Backlog:          operatingModeBacklogReader{store: s.backlogHandler.Store()},
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
		PromptCatalog: func(mode, phase string) (operatingmode.PromptCatalogEntry, bool) {
			entry, ok := promptcatalog.ResolveInitiativeModeSkill(mode, phase)
			if !ok {
				return operatingmode.PromptCatalogEntry{}, false
			}
			return operatingmode.PromptCatalogEntry{
				CatalogID:   entry.ID,
				SkillID:     entry.SkillID,
				Mode:        firstCatalogValue(entry.Modes),
				Phase:       firstCatalogValue(entry.Operations),
				OutputPaths: append([]string{}, entry.OutputPaths...),
			}, true
		},
		Events:       s.emitter,
		ScenarioRoot: scenarioRoot,
	})
	if err != nil {
		log.Fatalf("operating-mode: failed to build Service: %v", err)
	}
	s.operatingModeSvc = svc
	// The operating-mode subsystem serves the UI and CLI exclusively over the
	// typed OperatingModeService Connect contract — the bespoke gorilla/mux JSON
	// surface was retired once both consumers migrated (no untagged REST holdout).
	operatingmode.RegisterConnectService(s.router, svc)

	// Wire the active-rounds reader into the graph projection so initiative
	// nodes can render an operating-mode chip + pulse without N+1 fetches.
	// The graph projection was constructed earlier (registerGraphRoutes runs
	// before this); a setter is the seam that keeps both registrations
	// independently testable while letting the projection learn about the
	// operating-mode service late.
	if s.graphProjection != nil {
		s.graphProjection.SetOperatingModeReader(operatingModeActiveRoundReader{svc: svc})
	}
}

// operatingModeActiveRoundReader adapts operatingmode.Service into the
// graph.OperatingModeReader interface. Translation between the two
// active-round shapes lives here so neither package imports the other.
type operatingModeActiveRoundReader struct {
	svc *operatingmode.Service
}

func (r operatingModeActiveRoundReader) ActiveRoundsByInitiative(ctx context.Context) (map[string]graph.OperatingModeActiveRound, error) {
	if r.svc == nil {
		return map[string]graph.OperatingModeActiveRound{}, nil
	}
	rounds, err := r.svc.ActiveRoundsByInitiative(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]graph.OperatingModeActiveRound, len(rounds))
	for name, round := range rounds {
		out[name] = graph.OperatingModeActiveRound{
			Mode:   round.Mode,
			Phase:  round.Phase,
			Round:  round.Round,
			Status: round.Status,
		}
	}
	return out, nil
}

func firstCatalogValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
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
	source := proposals.Source{
		InitiativeName:   req.InitiativeName,
		Mode:             req.Mode,
		Phase:            req.Phase,
		RoundNumber:      req.Round,
		RoundSlug:        req.Phase,
		RunID:            req.RunID,
		Entrypoint:       "initiative.operating_mode.backlog_sync",
		DecidedBy:        req.DecidedBy,
		DecidedAtRFC3339: req.DecidedAtRFC3339,
	}
	result, err := r.applier.ApplyFlow(ctx, proposal, proposals.StateBuilder(r.stateBuilder), req.AcceptedMutationIDs, source)
	if err != nil {
		return nil, err
	}
	return operatingModeApplyResult(result), nil
}

// operatingModeApplyResult lifts the proposals.ApplyResult into the
// operating-mode wire shape and tallies per-op create/update counts off
// outcome metadata directly. Outcome carries both MutationID and Op, so
// we don't need the normalized proposal as a second input.
func operatingModeApplyResult(result *proposals.ApplyResult) *operatingmode.ProposalApplyResult {
	if result == nil {
		return nil
	}
	out := &operatingmode.ProposalApplyResult{
		Outcomes: make([]operatingmode.ProposalOutcome, 0, len(result.Outcomes)),
		Applied:  result.Applied,
		Failed:   result.Failed,
		Skipped:  result.Skipped,
	}
	for _, outcome := range result.Outcomes {
		out.Outcomes = append(out.Outcomes, operatingmode.ProposalOutcome{
			MutationID: outcome.MutationID,
			Op:         string(outcome.Op),
			Target:     outcome.Target,
			Applied:    outcome.Applied,
			Skipped:    outcome.Skipped,
			Error:      outcome.Error,
		})
		if !outcome.Applied {
			continue
		}
		switch outcome.Op {
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
		EmitBacklogStatusChangedFromSource(entityID, from, to string, source eventlog.BacklogMutationSourcePayload, itemRefs []string)
	}
}

func (m operatingModeBacklogMutator) MarkBacklogItemCompleted(_ context.Context, kind, name string, source operatingmode.BacklogMutationSource) (operatingmode.BacklogCompletionResult, error) {
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
			ref := kind + "/" + name
			m.events.EmitBacklogStatusChangedFromSource(ref, string(prior), string(item.Status), eventlog.BacklogMutationSourcePayload{
				Entrypoint:     source.Entrypoint,
				InitiativeName: source.InitiativeName,
				Mode:           source.Mode,
				Phase:          source.Phase,
				Round:          source.Round,
				RunID:          source.RunID,
				RequestedBy:    source.RequestedBy,
			}, []string{ref})
		}
	}
	return operatingmode.BacklogCompletionResult{
		ItemRef:    kind + "/" + name,
		FromStatus: string(prior),
		ToStatus:   string(backlog.StatusCompleted),
	}, nil
}

type operatingModeInitiativeLister struct {
	store *initiatives.Store
}

func (l operatingModeInitiativeLister) ListInitiatives() ([]operatingmode.InitiativeSummary, error) {
	if l.store == nil {
		return nil, nil
	}
	all, err := l.store.LoadAll()
	if err != nil {
		return nil, err
	}
	out := make([]operatingmode.InitiativeSummary, 0, len(all))
	for _, init := range all {
		if init.ArchivedAt != nil {
			continue
		}
		out = append(out, operatingmode.InitiativeSummary{
			Name:    init.Name,
			Title:   init.Title,
			Mode:    init.Mode,
			Status:  init.Status,
			Updated: init.Updated,
		})
	}
	return out, nil
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
		return operatingmode.InitiativeSnapshot{}, fmt.Errorf("operating-mode initiative updater is not configured")
	}
	updated, err := u.service.SetModeLifecycle(name, mode)
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
