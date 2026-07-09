package initiativereview

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/review"
)

// ErrNotReady signals that TriggerIfReady was called before the initiative
// satisfied the "all items terminal and initiative active" precondition.
// Returned only from TriggerIfReady's error path — the non-error path
// silently reports via TriggerResult.Started=false.
var ErrNotReady = errors.New("initiative is not ready for review")

// AgentSpawner is the narrow interface the service needs from agentmanager.
// Defined here so tests can inject an in-process spawner without pulling in
// HTTP infrastructure.
type AgentSpawner interface {
	IsEnabled() bool
	SpawnInitiative(ctx context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error)
}

// RunInspector retrieves the current state of an agent run. Optional — when
// provided, the service polls gathering rounds and flips them to terminal
// status as the agent completes.
type RunInspector interface {
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
}

// InitiativeStore is the narrow slice of initiatives.Store used here: load
// for context building, save for status transitions. Kept behind an
// interface so tests can stub.
type InitiativeStore interface {
	Load(name string) (*initiatives.Initiative, error)
	LoadAll() ([]initiatives.Initiative, error)
	Save(init *initiatives.Initiative) error
	InitDir(name string) string
}

// BacklogLoader loads member-item metadata for building review context.
// Only read methods — this package never mutates backlog items.
type BacklogLoader interface {
	LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error)
	ItemDir(kind backlog.BacklogKind, name string) string
}

// GraphReader returns the materialized graph.json for an initiative. Used
// to compose the agent's context; the read is best-effort — if the
// projection hasn't run yet we fall through with an empty graph.
type GraphReader interface {
	ReadGraph(initiativeName string) (*graph.MaterializedGraph, error)
}

// ItemFinalization is the finalization slice initiative review needs to
// discover which scenarios a completed item touched. Only AffectedScenarios
// is consumed today — initiative review runs a *fresh* GCT pass against
// each affected scenario at review start rather than reusing the item's
// historical verdict, so the snapshot reviews live on item-level review
// rounds, not here.
type ItemFinalization struct {
	AffectedScenarios []string
}

// ExecutionLookup returns the latest completed execution's finalization
// snapshot for a backlog item — initiative review consumes only the
// `AffectedScenarios` field to discover which scenarios to run a fresh
// GCT review against. Returns (nil, nil) when the item has no
// finalization data (e.g. research items, or items that completed before
// finalization was instrumented); callers treat the miss as "no scenarios
// in scope for this item", not as an error.
type ExecutionLookup interface {
	LatestFinalizationFor(kind backlog.BacklogKind, name string) (*ItemFinalization, error)
}

type PlanContentResolver func(ctx context.Context, item backlog.BacklogItem, itemDir string) (string, error)

// Config bundles the dependencies the service needs. Fields with defaults
// (PromptClient, Clock, GCTPollInterval, GCTPollTimeout) are optional;
// the rest are required and validated in NewService.
//
// Lock is the shared per-initiative mutex that also gates feedback rounds.
// Passing a nil Lock is legal (for tests / degraded modes) but leaves the
// initiative open to concurrent feedback+review spawns — production wiring
// must always set it.
//
// ExecutionLookup is optional. When provided, the service unions the
// affected-scenarios list across all member items at review start and
// hands them to GCTClient for a *fresh* git-control-tower pass; the
// verdicts land on the review agent as `affected-scenarios` +
// `gct-review-results` attachments (same keys the backlog review flow
// uses, so skill authors only learn one vocabulary).
//
// GCTClient is optional but strongly recommended in production — without
// it, the review still runs, but the gct-review-results attachment is
// omitted. The interface is narrow (scenario name → verdict) so that
// wiring `execution.HTTPReviewClient` through an adapter is a one-liner
// in main.go.
//
// GCTPollInterval / GCTPollTimeout bound the fresh-GCT polling loop
// per scenario. Defaults: 3s interval, 5m timeout. Tests inject small
// values to keep poll latency out of the test budget.
type Config struct {
	InitStore       InitiativeStore
	BacklogLoader   BacklogLoader
	GraphReader     GraphReader
	Spawner         AgentSpawner
	Lock            *initiativelock.Lock
	ExecutionLookup ExecutionLookup
	PlanContent     PlanContentResolver
	GCTClient       GCTClient
	GCTPollInterval time.Duration
	GCTPollTimeout  time.Duration
	PromptClient    promptmanager.Client
	Clock           func() time.Time
}

// Service orchestrates initiative review rounds.
type Service struct {
	initStore       InitiativeStore
	backlogLoader   BacklogLoader
	graphReader     GraphReader
	spawner         AgentSpawner
	inspector       RunInspector
	lock            *initiativelock.Lock
	executionLookup ExecutionLookup
	planContent     PlanContentResolver
	gctClient       GCTClient
	gctPollInterval time.Duration
	gctPollTimeout  time.Duration
	promptClient    promptmanager.Client
	clock           func() time.Time

	mu           sync.Mutex
	activeRounds map[string]activeRound // keyed by RunID
	// triggerGate serializes TriggerIfReady per initiative so two items
	// reaching terminal in the same tick can't both spawn a review round.
	// Without this, the acquire/save race would let both callers pass the
	// "status == active" check before either flips the status to in_review.
	triggerGate sync.Mutex
}

// activeRound tracks one gathering round for the background poller.
type activeRound struct {
	InitiativeName string
	RoundNum       int
	RunID          string
}

// NewService validates Config and returns a ready-to-use Service.
func NewService(cfg Config) (*Service, error) {
	if cfg.InitStore == nil {
		return nil, errors.New("initiativereview: InitStore is required")
	}
	if cfg.BacklogLoader == nil {
		return nil, errors.New("initiativereview: BacklogLoader is required")
	}
	if cfg.GraphReader == nil {
		return nil, errors.New("initiativereview: GraphReader is required")
	}
	clk := cfg.Clock
	if clk == nil {
		clk = time.Now
	}
	pc := cfg.PromptClient
	if pc == nil {
		pc = promptmanager.NewHTTPClient()
	}
	svc := &Service{
		initStore:       cfg.InitStore,
		backlogLoader:   cfg.BacklogLoader,
		graphReader:     cfg.GraphReader,
		spawner:         cfg.Spawner,
		lock:            cfg.Lock,
		executionLookup: cfg.ExecutionLookup,
		planContent:     cfg.PlanContent,
		gctClient:       cfg.GCTClient,
		gctPollInterval: cfg.GCTPollInterval,
		gctPollTimeout:  cfg.GCTPollTimeout,
		promptClient:    pc,
		clock:           clk,
		activeRounds:    make(map[string]activeRound),
	}
	// Type-assert for RunInspector capability (mirrors review.Service).
	if inspector, ok := cfg.Spawner.(RunInspector); ok {
		svc.inspector = inspector
	}
	return svc, nil
}

// ListRounds returns all review rounds for the initiative, oldest-first.
// Rounds still in gathering state are probed for run-completion (if an
// inspector is wired) so the first List after an agent finishes picks up
// the new terminal state even before the background poller runs.
func (s *Service) ListRounds(initiativeName string) ([]review.Round, error) {
	itemDir := s.initStore.InitDir(initiativeName)
	rounds, err := review.LoadRounds(itemDir)
	if err != nil {
		return nil, err
	}
	if s.inspector == nil {
		return rounds, nil
	}
	for i := range rounds {
		r := &rounds[i]
		if r.Status != review.RoundStatusGathering || r.RunID == "" {
			continue
		}
		state, stateErr := s.inspector.GetRunState(context.Background(), r.RunID)
		if stateErr != nil {
			continue
		}
		if finalized, changed := finalizeRoundIfTerminal(*r, state); changed {
			*r = finalized
			_ = review.SaveRound(itemDir, *r)
			s.handleTerminalRound(context.Background(), initiativeName, *r)
			s.mu.Lock()
			delete(s.activeRounds, r.RunID)
			s.mu.Unlock()
		}
	}
	return rounds, nil
}

// GetRound returns a specific review round by number, or nil if not found.
func (s *Service) GetRound(initiativeName string, roundNum int) (*review.Round, error) {
	return review.LoadRound(s.initStore.InitDir(initiativeName), roundNum)
}

func (s *Service) setInitiativeStatus(init *initiatives.Initiative, status, when string) error {
	init.Status = status
	init.Updated = when
	if err := s.initStore.Save(init); err != nil {
		return fmt.Errorf("save initiative status %q: %w", status, err)
	}
	return nil
}

// renderInstructions builds the per-round review skill prompt.
func (s *Service) renderInstructions(ctx context.Context, init *initiatives.Initiative, roundNum int) (string, error) {
	vars := map[string]string{
		"INITIATIVE_NAME": init.Name,
		"ROUND_NUMBER":    fmt.Sprintf("%03d", roundNum),
	}
	return s.promptClient.ReadSkill(ctx, "swarm-manager-initiative-review", vars, true)
}

// --- Attachment assembly lives in context.go so this file stays focused on
// lifecycle.
