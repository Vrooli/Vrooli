package main

import (
	"context"
	"log/slog"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/promptmanager"
)

func (s *Server) registerOperatingModeRoutes(scenarioRoot string) {
	if s.initStore == nil || s.initiativeService == nil {
		return
	}
	store := operatingmode.NewStore(s.initiativeService.InitDir)
	lock := &initiativelock.Lock{Dir: s.initiativeService.InitDir}
	svc, err := operatingmode.NewService(operatingmode.Config{
		Store:       store,
		Lock:        lock,
		Initiatives: operatingModeInitiativeReader{store: s.initStore},
		ModeUpdater: operatingModeUpdater{service: s.initiativeService},
		Backlog:     operatingModeBacklogReader{store: s.backlogHandler.Store()},
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
