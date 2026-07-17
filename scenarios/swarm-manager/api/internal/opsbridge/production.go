package opsbridge

import (
	"context"
	"fmt"
	"log/slog"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"
)

// BacklogRunnerConfig is everything BuildBacklogRunner needs to assemble the
// production operation runner. Every operating-mode seam is an interface so the
// wiring is testable with fakes; production passes one *operatingmode.Service
// for all three (it satisfies each). The Registry is pre-populated by the caller
// (backlog.RegisterOpsHandlers) so opsbridge stays free of a backlog import — the
// dependency edge to the domain flows through the caller, not the bridge.
type BacklogRunnerConfig struct {
	Catalog     *opscatalog.Catalog
	ModeDefs    map[operatingmode.Mode]operatingmode.Definition
	PhaseEngine opsrunner.PhaseEngine // StartTargetPhase: the live round start
	SimEngine   SimulationEngine      // SimulateMode: the simulation driver
	Refresher   RunRefresher          // RefreshRunByID: the target-round poller
	Locator     opsrunner.FSLocator
	Registry    *opsrunner.ActionRegistry
	// AdvanceResolver, when non-nil, resolves an operation-carrying scheduled
	// intent (intent.Operation != "") into the concrete Invoke the scheduler
	// firer runs. It is the domain seam that decides which operation advances the
	// target and derives a replay-safe idempotency key from current state, so the
	// bridge stays domain-free. When nil, operation-carrying intents cannot fire.
	AdvanceResolver AdvanceResolver
	// InitiativeOfItem, when non-nil, resolves the initiative a backlog item
	// belongs to (empty when none) so a backlog-item target's resolution also
	// sees its initiative's binding overrides (the initiative-override layer is
	// inherited by member items). Nil means item targets see only their own
	// overrides.
	InitiativeOfItem func(itemRef string) (string, error)
	RequestedBy      string
	Logger           *slog.Logger
}

// AdvanceResolver turns an operation-carrying scheduled intent into the concrete
// Invoke request that advances its target. It loads the domain entity, decides
// which operation to run from current readiness, and sets a state-derived
// IdempotencyKey so a crash re-fire REPLAYS rather than double-starting. ok=false
// means the advance is no longer warranted (the intent is consumed without an
// Invoke).
type AdvanceResolver interface {
	ResolveAdvance(ctx context.Context, w agentops.WorkflowInstance, intent agentops.ScheduledIntent) (opsrunner.InvokeRequest, bool, error)
}

// BacklogRunner is the assembled production runtime: the runner plus the
// background collaborators the caller starts and the round observer it installs
// on the operating-mode engine.
type BacklogRunner struct {
	Runner        *opsrunner.Runner
	Repo          *opsrunner.WorkflowRepo
	Executions    *opsrunner.ExecutionStore
	Scheduler     *opsrunner.Scheduler
	RefreshDriver *RefreshDriver
	// Observer is installed via operatingmode.Service.SetRoundObserver. It routes
	// a runner-owned terminal round into Runner.CommitResult and ignores every
	// other round (legacy initiative rounds, target rounds no operation started).
	Observer operatingmode.RoundObserver
}

// BuildBacklogRunner constructs the production operation runner exactly as the
// server wires it: the live preparer as BOTH the runner's ModePreparer and the
// resolver's ModeChecker, the simulation driver as the synchronous execution
// driver, the engine run-starter as the live (non-blocking) start seam, and the
// dispatcher over the caller-populated action registry. It also builds the
// completion router (Observer), the target-round refresh driver, and the durable
// scheduler, so the whole async lifecycle is available from one call — which is
// what makes the production shape testable end to end.
func BuildBacklogRunner(cfg BacklogRunnerConfig) (*BacklogRunner, error) {
	if cfg.Catalog == nil {
		return nil, fmt.Errorf("opsbridge: BuildBacklogRunner requires a catalog")
	}
	if cfg.Registry == nil {
		return nil, fmt.Errorf("opsbridge: BuildBacklogRunner requires an action registry")
	}
	if cfg.PhaseEngine == nil || cfg.SimEngine == nil || cfg.Refresher == nil {
		return nil, fmt.Errorf("opsbridge: BuildBacklogRunner requires phase engine, simulation engine, and refresher")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	defsByID := make(map[string]operatingmode.Definition, len(cfg.ModeDefs))
	for mode, def := range cfg.ModeDefs {
		defsByID[string(mode)] = def
	}
	// The full mode set is also the delegated-mode pool so a composed mode's
	// executed_by (a wrapper mode delegating to a generic drain mode) resolves
	// when the preparer compiles its input contract.
	preparer := opsrunner.NewLivePreparer(cfg.Catalog, defsByID).WithDelegated(defsByID)

	repo := opsrunner.NewWorkflowRepo(cfg.Locator)
	execStore := opsrunner.NewExecutionStore(cfg.Locator)
	overrides := opsrunner.NewFSOverrideStore(cfg.Locator)
	// Item targets inherit their initiative's overrides (initiative-override
	// layer) when the caller provides the item→initiative mapping.
	overrides.InitiativeOfItem = cfg.InitiativeOfItem
	// The live preparer is also the resolver's ModeChecker, so a binding to a
	// deleted revision or an incompatible mode fails closed against the SAME
	// registry the runner prepares from.
	resolver := opsrunner.NewBindingResolver(cfg.Catalog, overrides, preparer)
	// The resolver needs the same item→initiative mapping the store uses, or a
	// fetched inherited override would be rejected as out of scope.
	resolver.InitiativeOfItem = cfg.InitiativeOfItem
	dispatcher := opsrunner.NewDispatcher(cfg.Registry, repo)
	driver := NewSimulationDriver(cfg.SimEngine, cfg.Catalog)
	starter := opsrunner.NewEngineRunStarter(cfg.PhaseEngine, cfg.RequestedBy)

	runner, err := opsrunner.New(opsrunner.Config{
		Catalog:    cfg.Catalog,
		Resolver:   resolver,
		Preparer:   preparer,
		Driver:     driver,
		Starter:    starter,
		Repo:       repo,
		Executions: execStore,
		Dispatcher: dispatcher,
	})
	if err != nil {
		return nil, fmt.Errorf("opsbridge: construct runner: %w", err)
	}

	scheduler := opsrunner.NewScheduler(repo, newIntentFirer(runner, dispatcher, cfg.AdvanceResolver, logger), nil)
	refresh := NewRefreshDriver(repo, cfg.Refresher, logger)
	observer := NewCompletionRouter(repo, runner, logger).Observe

	return &BacklogRunner{
		Runner:        runner,
		Repo:          repo,
		Executions:    execStore,
		Scheduler:     scheduler,
		RefreshDriver: refresh,
		Observer:      observer,
	}, nil
}

// newIntentFirer returns the scheduler's IntentFirer. It branches on the intent
// shape: an OPERATION-carrying intent (intent.Operation != "") starts the next
// operation through runner.Invoke — the honest model for "advance to the next
// round" — resolving the concrete Invoke (and its replay-safe idempotency key)
// through the domain AdvanceResolver; an ACTION-carrying intent dispatches the
// pre-decided registered action against the target's current workflow state (a
// coordination no-op that runs the domain handler). It is restart-safe because
// the scheduler reloads intents from durable workflow state each tick; a lost
// commit re-fires the same idempotency key, which both Invoke and the dispatcher
// treat as a replay.
func newIntentFirer(runner *opsrunner.Runner, dispatcher *opsrunner.Dispatcher, resolver AdvanceResolver, logger *slog.Logger) opsrunner.IntentFirer {
	return func(ctx context.Context, w agentops.WorkflowInstance, intent agentops.ScheduledIntent) error {
		if intent.Operation != "" {
			if resolver == nil {
				logger.Warn("opsbridge: operation intent cannot fire without an advance resolver", "intent", intent.Intent, "operation", intent.Operation)
				return nil // consume: no resolver means the advance can never run
			}
			req, ok, err := resolver.ResolveAdvance(ctx, w, intent)
			if err != nil {
				logger.Warn("opsbridge: resolve advance intent failed", "err", err, "intent", intent.Intent, "operation", intent.Operation)
				return err // retry on the next tick
			}
			if !ok {
				return nil // advance no longer warranted; consume the intent
			}
			if _, err := runner.Invoke(ctx, req); err != nil {
				logger.Warn("opsbridge: scheduled operation invoke failed", "err", err, "intent", intent.Intent, "operation", req.Operation)
				return err
			}
			return nil
		}
		sel := opsrunner.SelectedTransition{Action: intent.Action, ToState: w.State}
		target := opsrunner.TargetRef{Kind: agentops.TargetKind(w.Domain.Kind), ID: w.Domain.ID}
		_, err := dispatcher.Dispatch(ctx, target, w, sel, "", "", "intent-"+intent.Intent, opsrunner.DispatchDelivery{})
		if err != nil {
			logger.Warn("opsbridge: scheduled intent dispatch failed", "err", err, "intent", intent.Intent, "action", intent.Action)
		}
		return err
	}
}
