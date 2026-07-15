package main

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"swarm-manager/internal/agentopsdiag"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"
)

// initiativeOfItemFunc returns the item→initiative mapping over the backlog
// store, so a backlog-item target inherits its initiative's binding overrides
// (the initiative-override layer). A missing item maps to "no initiative"
// rather than an error: reading overrides for a not-yet-created item is a
// legitimate diagnostics question and must not fail the whole resolution.
func initiativeOfItemFunc(store backlog.Store) func(string) (string, error) {
	return func(itemRef string) (string, error) {
		kind, name, ok := strings.Cut(itemRef, "/")
		if !ok || strings.TrimSpace(kind) == "" || strings.TrimSpace(name) == "" {
			return "", fmt.Errorf("backlog-item ref %q must be kind/name", itemRef)
		}
		item, err := store.LoadItem(backlog.BacklogKind(kind), name)
		if err != nil {
			if errors.Is(err, backlog.ErrNotFound) {
				return "", nil
			}
			return "", err
		}
		return item.Initiative, nil
	}
}

// registerAgentOperationsDiagnostics mounts the AgentOperationsService operator
// surface over the declarative agent-operations runtime: read-only projections
// (catalog, compatibility, resolved bindings, workflow, executions, migration
// status) plus the binding-override write path. It is best-effort: a scenario
// that has not yet authored an operation catalog logs a notice and skips
// registration rather than failing server startup, because this surface is
// observability + operator control, not a load-bearing dependency of any other
// subsystem.
func (s *Server) registerAgentOperationsDiagnostics(scenarioRoot string) {
	if s.initiativeService == nil || s.backlogHandler == nil {
		slog.Warn("agent-operations diagnostics: initiative/backlog services unavailable; skipping")
		return
	}
	catalog, err := opscatalog.Load(scenarioRoot)
	if err != nil {
		slog.Warn("agent-operations diagnostics: operation catalog not loadable; diagnostics disabled", "err", err)
		return
	}
	backlogStore := s.backlogHandler.Store()
	locator := opsrunner.FSLocator{
		BacklogItemDir: func(kind, name string) (string, error) {
			return backlogStore.ItemDir(backlog.BacklogKind(kind), name), nil
		},
		InitiativeDir: func(name string) (string, error) {
			return s.initiativeService.InitDir(name), nil
		},
	}
	repo := opsrunner.NewWorkflowRepo(locator)
	execStore := opsrunner.NewExecutionStore(locator)

	// The overrides read store inherits initiative overrides for item targets,
	// matching the live invocation path: an item without its own override is
	// governed by its initiative's binding.
	overrides := opsrunner.NewFSOverrideStore(locator)
	overrides.InitiativeOfItem = initiativeOfItemFunc(backlogStore)

	// The mode checker is the SAME LivePreparer construction the live runner
	// uses (opsbridge.BuildBacklogRunner), so revision-existence and
	// mode-compatibility answers here agree with what an Invoke would decide.
	// Mode loading is best-effort for the READ paths (a nil checker still
	// resolves precedence fail-closed); the WRITE path refuses to accept an
	// override it cannot validate against a mode registry (fail-closed in the
	// service).
	var defsByID map[string]operatingmode.Definition
	svc := agentopsdiag.NewService(catalog, nil, repo, execStore)
	if modeDefs, err := operatingmode.LoadModesFromDir(filepath.Join(scenarioRoot, "modes")); err != nil {
		slog.Warn("agent-operations diagnostics: modes not loadable; mode checks unavailable and override writes disabled", "err", err)
	} else {
		defsByID = make(map[string]operatingmode.Definition, len(modeDefs))
		for mode, def := range modeDefs {
			defsByID[string(mode)] = def
		}
		checker := opsrunner.NewLivePreparer(catalog, defsByID).WithDelegated(defsByID)
		svc = svc.WithModes(defsByID, checker)
	}
	resolver := opsrunner.NewBindingResolver(catalog, overrides, svc.ModeChecker())
	// Same mapping as the store: inherited overrides must also be in scope.
	resolver.InitiativeOfItem = overrides.InitiativeOfItem
	svc = svc.WithResolver(resolver).
		WithOverrideAdmin(overrides, opsrunner.NewOverrideWriter(locator)).
		WithMigrationStatusPath(filepath.Join(s.dataRoot, "agentops", "migration-status.json"))

	agentopsdiag.RegisterConnectService(s.router, svc)
	slog.Info("agent-operations diagnostics registered", "modes", len(defsByID))
}
