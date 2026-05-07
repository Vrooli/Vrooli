package initiativereview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/pathredact"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/review"
	"swarm-manager/internal/storage"
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

// TriggerIfReady checks whether the initiative is ready for review and, if
// so, spawns the review agent and returns TriggerResult{Started: true}. If
// the initiative is already under review (in_review / review_pending) or
// has outstanding non-terminal items, returns Started=false with a reason.
// Idempotent: safe to call from multiple places (item transitions, manual
// triggers, recovery).
//
// Serialized under triggerGate so two items reaching terminal in the
// same tick both try TriggerForItem and only one wins the race to spawn
// a round — the second sees the freshly-saved in_review status and
// reports Started=false with reason "initiative status is \"in_review\"".
func (s *Service) TriggerIfReady(ctx context.Context, initiativeName string) (TriggerResult, error) {
	s.triggerGate.Lock()
	defer s.triggerGate.Unlock()
	init, err := s.initStore.Load(initiativeName)
	if err != nil {
		return TriggerResult{}, fmt.Errorf("load initiative: %w", err)
	}

	// Guard: only active initiatives can enter review. Anything else means
	// review has already started, the user has already decided, or the
	// initiative is archived — none of which should re-trigger.
	if init.Status != initiatives.InitiativeStatusActive {
		return TriggerResult{
			Started: false,
			Reason:  fmt.Sprintf("initiative status is %q; review triggers only from %q", init.Status, initiatives.InitiativeStatusActive),
		}, nil
	}

	// Guard: every member item must be in a terminal state. A fresh
	// initiative with zero items is also not "ready" — nothing to review.
	if len(init.Items) == 0 {
		return TriggerResult{Started: false, Reason: "initiative has no items"}, nil
	}
	nonTerminal, err := s.findNonTerminalItems(init)
	if err != nil {
		return TriggerResult{}, fmt.Errorf("scan items: %w", err)
	}
	if len(nonTerminal) > 0 {
		return TriggerResult{
			Started: false,
			Reason:  fmt.Sprintf("%d item(s) not yet terminal: %s", len(nonTerminal), strings.Join(nonTerminal, ", ")),
		}, nil
	}

	return s.startReview(ctx, init)
}

// TriggerForItem is the hook the backlog review-decide path calls after a
// member item flips to a terminal status. It resolves the item's initiative
// (if any) and calls TriggerIfReady. Errors are logged but never block the
// item decision — the review phase is a downstream consequence, not a
// precondition.
func (s *Service) TriggerForItem(ctx context.Context, kind, name string) {
	item, err := s.backlogLoader.LoadItem(backlog.BacklogKind(kind), name)
	if err != nil {
		return
	}
	initiativeName := strings.TrimSpace(item.Initiative)
	if initiativeName == "" {
		return
	}
	result, err := s.TriggerIfReady(ctx, initiativeName)
	if err != nil {
		// Lock conflicts are expected when a feedback round is in flight;
		// they're signal for the user, not an operator-facing alarm. Any
		// other error is unexpected (load/save failure, etc.) and stays WARN.
		var conflict *initiativelock.Conflict
		if errors.As(err, &conflict) {
			slog.Info("initiative review deferred: lock held",
				"initiative", initiativeName, "holder_purpose", conflict.Holder.Purpose)
			return
		}
		slog.Warn("initiative review trigger failed", "initiative", initiativeName, "kind", kind, "name", name, "err", err)
		return
	}
	if result.Started {
		slog.Info("initiative review started",
			"initiative", initiativeName,
			"round", result.Round,
			"run_id", result.RunID,
			"trigger", fmt.Sprintf("item:%s/%s", kind, name),
		)
	}
}

// Decide flips the initiative from review_pending to a terminal status per
// the user's verdict, appending an audit record under review/decisions/.
// Only review_pending initiatives can be decided — any other state is an
// explicit error so double-decides and stale CLI calls can't mutate terminal
// state.
func (s *Service) Decide(ctx context.Context, initiativeName string, verdict Verdict, rationale, decidedBy string) (DecideResponse, error) {
	init, err := s.initStore.Load(initiativeName)
	if err != nil {
		return DecideResponse{}, err
	}
	if init.Status != initiatives.InitiativeStatusReviewPending {
		return DecideResponse{}, fmt.Errorf("initiative status is %q; decide requires %q", init.Status, initiatives.InitiativeStatusReviewPending)
	}
	target := verdict.TargetStatus()
	if target == "" {
		return DecideResponse{}, fmt.Errorf("invalid verdict %q", verdict)
	}

	priorStatus := init.Status
	decidedAt := s.clock().UTC().Format(time.RFC3339)

	init.Status = target
	init.Updated = decidedAt
	if err := s.initStore.Save(init); err != nil {
		return DecideResponse{}, fmt.Errorf("save initiative: %w", err)
	}

	latest, _, _ := review.LoadLatestRound(s.initStore.InitDir(initiativeName))
	latestRound := 0
	if latest != nil {
		latestRound = latest.RoundNum
	}

	// Decision record is supplementary audit; failure to persist it logs
	// but does not roll back the status flip (mirrors backlog review_decide).
	if writeErr := s.writeDecision(initiativeName, DecisionRecord{
		Verdict:     string(verdict),
		Status:      target,
		Rationale:   strings.TrimSpace(rationale),
		DecidedBy:   strings.TrimSpace(decidedBy),
		DecidedAt:   decidedAt,
		PriorStatus: priorStatus,
		Round:       latestRound,
	}); writeErr != nil {
		slog.Warn("initiative review: persist decision record", "initiative", initiativeName, "err", writeErr)
	}

	return DecideResponse{
		Initiative: initiativeName,
		Verdict:    string(verdict),
		Status:     target,
		Rationale:  strings.TrimSpace(rationale),
		DecidedAt:  decidedAt,
	}, nil
}

// startReview materializes a new round file in gathering state, flips the
// initiative to in_review, and spawns the review agent. Split out from
// TriggerIfReady so manual-trigger and auto-trigger share one code path.
//
// Lock contract: when a lock is wired, startReview acquires it with a
// provisional RunID before the spawn, then overrides it with the agent-
// manager RunID once SpawnInitiative succeeds. On spawn failure the
// provisional release clears the lock so the initiative isn't wedged.
// Release happens in handleTerminalRound (or on decide, if a future verdict
// path lands while the round is still alive). Returns a *initiativelock.
// Conflict error if feedback holds the lock; the caller can inspect the
// holder to render a user-facing conflict dialog.
func (s *Service) startReview(ctx context.Context, init *initiatives.Initiative) (TriggerResult, error) {
	itemDir := s.initStore.InitDir(init.Name)
	roundNum, err := review.NextRoundNumber(itemDir)
	if err != nil {
		return TriggerResult{}, fmt.Errorf("next round: %w", err)
	}

	instructions, err := s.renderInstructions(ctx, init, roundNum)
	if err != nil {
		return TriggerResult{}, fmt.Errorf("render skill: %w", err)
	}

	// Run a fresh GCT pass over the union of affected scenarios before
	// spawning the review agent. This is the "is the whole thing still
	// working together" integration check the initiative review is
	// designed around: per-item reviews landed earlier against earlier
	// scenario states, and something may have drifted since. The call
	// blocks (bounded by GCTPollTimeout per scenario) so results land
	// in the review agent's context as fresh evidence rather than
	// stale history.
	affectedScenarios := s.collectAffectedScenarios(init)
	freshGCT := s.runFreshGCT(ctx, affectedScenarios)

	attachments, err := s.buildContextAttachments(init, affectedScenarios, freshGCT)
	if err != nil {
		return TriggerResult{}, fmt.Errorf("build attachments: %w", err)
	}

	generatedAt := s.clock().UTC().Format(time.RFC3339)
	round := review.Round{
		RoundNum:    roundNum,
		GeneratedAt: generatedAt,
		Status:      review.RoundStatusGathering,
		Evidence:    []review.EvidenceItem{},
	}

	// Acquire the per-initiative lock before starting review work. Feedback
	// rounds use the same file (`.feedback-lock`), so a feedback holder here
	// is a real conflict even in degraded no-spawner mode.
	provisionalRunID := fmt.Sprintf("review-provisional-%d-%d", roundNum, s.clock().UnixNano())
	if s.lock != nil {
		if err := s.lock.Acquire(init.Name, initiativelock.Holder{
			RunID:       provisionalRunID,
			Purpose:     "review",
			RoundNumber: roundNum,
			AcquiredBy:  "swarm-manager:initiative-review",
		}); err != nil {
			return TriggerResult{}, err
		}
	}

	if s.spawner == nil || !s.spawner.IsEnabled() {
		// Degraded mode (no spawner): the round still lands on disk and
		// the initiative flips to in_review so the UI doesn't lie about
		// lifecycle, but no agent-manager work happens. The provisional
		// lock is released immediately because there is no live run to own
		// it after the status transition.
		if err := review.SaveRound(itemDir, round); err != nil {
			if s.lock != nil {
				_ = s.lock.Release(init.Name, provisionalRunID)
			}
			return TriggerResult{}, fmt.Errorf("save round: %w", err)
		}
		if err := s.setInitiativeStatus(init, initiatives.InitiativeStatusInReview, generatedAt); err != nil {
			if s.lock != nil {
				_ = s.lock.Release(init.Name, provisionalRunID)
			}
			return TriggerResult{}, err
		}
		if s.lock != nil {
			_ = s.lock.Release(init.Name, provisionalRunID)
		}
		return TriggerResult{Started: true, Round: roundNum}, nil
	}

	runResult, err := s.spawner.SpawnInitiative(ctx, agentmanager.InitiativeSpawnRequest{
		Name:               init.Name,
		Title:              "Review: " + fallbackInitiativeTitle(init),
		Description:        instructions,
		Prompt:             instructions,
		ScopePath:          ".",
		ProjectRoot:        ".",
		CreatedBy:          "swarm-manager:initiative-review",
		Purpose:            "review",
		RoundNumber:        roundNum,
		RoundSlug:          "review",
		ContextAttachments: attachments,
		Environment: map[string]string{
			"VROOLI_SPAWN_SOURCE": "swarm-manager-initiative-review",
		},
	})
	if err != nil {
		if s.lock != nil {
			_ = s.lock.Release(init.Name, provisionalRunID)
		}
		return TriggerResult{}, fmt.Errorf("spawn agent: %w", err)
	}

	// Swap the provisional holder for the agent-manager RunID so a later
	// Release(runResult.RunID) actually clears the lock. AcquireOverride is
	// a pure file write; on the unlikely failure path we release the
	// provisional to avoid a wedged lock.
	//
	// Safety: between the Acquire above and AcquireOverride here, triggerGate
	// is still held (TriggerIfReady serializes all startReview calls for
	// this initiative), so no other caller can observe or replace the
	// provisional holder. AcquireOverride itself doesn't validate that the
	// provisional is still present — it's an unconditional write — and that's
	// fine under the single-writer invariant this path guarantees.
	if s.lock != nil {
		if swapErr := s.lock.AcquireOverride(init.Name, initiativelock.Holder{
			RunID:       runResult.RunID,
			Purpose:     "review",
			RoundNumber: roundNum,
			AcquiredBy:  "swarm-manager:initiative-review",
		}); swapErr != nil {
			slog.Warn("initiative review: lock run-id swap failed; releasing provisional",
				"initiative", init.Name, "round", roundNum, "err", swapErr)
			_ = s.lock.Release(init.Name, provisionalRunID)
		}
	}

	round.RunID = runResult.RunID
	round.ExecutionID = "" // initiative reviews have no single execution owner
	if err := review.SaveRound(itemDir, round); err != nil {
		return TriggerResult{}, fmt.Errorf("save round: %w", err)
	}
	if err := s.setInitiativeStatus(init, initiatives.InitiativeStatusInReview, generatedAt); err != nil {
		return TriggerResult{}, err
	}
	s.trackActiveRound(init.Name, roundNum, runResult.RunID)

	slog.Info("initiative review started",
		"initiative", init.Name,
		"round", roundNum,
		"run_id", runResult.RunID,
	)

	return TriggerResult{Started: true, Round: roundNum, RunID: runResult.RunID}, nil
}

// findNonTerminalItems returns the "kind/name" refs of member items that
// are not yet in a terminal state. Missing-from-disk items are treated as
// non-terminal so a half-deleted initiative won't race into review.
func (s *Service) findNonTerminalItems(init *initiatives.Initiative) ([]string, error) {
	out := make([]string, 0)
	for _, ref := range init.Items {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			out = append(out, ref)
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			out = append(out, ref)
			continue
		}
		item, err := s.backlogLoader.LoadItem(kind, parts[1])
		if err != nil {
			out = append(out, ref)
			continue
		}
		if item.ArchivedAt != nil {
			continue // archived items are terminal for rollup purposes
		}
		if !backlog.IsTerminalStatus(item.Status) {
			out = append(out, ref)
		}
	}
	return out, nil
}

func (s *Service) setInitiativeStatus(init *initiatives.Initiative, status, when string) error {
	init.Status = status
	init.Updated = when
	if err := s.initStore.Save(init); err != nil {
		return fmt.Errorf("save initiative status %q: %w", status, err)
	}
	return nil
}

// handleTerminalRound flips the initiative from in_review to review_pending
// when the review round reaches a terminal status and releases the per-
// initiative lock so feedback submissions can proceed once the user is
// looking at the review verdict. Called from both the ListRounds inline-
// poll and the background worker.
func (s *Service) handleTerminalRound(_ context.Context, initiativeName string, round review.Round) {
	if round.Status != review.RoundStatusComplete && round.Status != review.RoundStatusFailed {
		return
	}

	// Release the lock first — even if the status flip below fails, we'd
	// rather leak the status machine (which the user can fix with decide)
	// than leak a lock that blocks every subsequent feedback submission.
	if s.lock != nil && round.RunID != "" {
		if err := s.lock.Release(initiativeName, round.RunID); err != nil {
			slog.Warn("initiative review: release lock", "initiative", initiativeName, "round", round.RoundNum, "err", err)
		}
	}

	init, err := s.initStore.Load(initiativeName)
	if err != nil {
		slog.Warn("initiative review: load after terminal round", "initiative", initiativeName, "err", err)
		return
	}
	// Only transition from in_review; if the user has already decided or
	// another flow moved us, don't overwrite.
	if init.Status != initiatives.InitiativeStatusInReview {
		return
	}
	if err := s.setInitiativeStatus(init, initiatives.InitiativeStatusReviewPending, s.clock().UTC().Format(time.RFC3339)); err != nil {
		slog.Warn("initiative review: flip to review_pending", "initiative", initiativeName, "err", err)
	}
}

// writeDecision persists a DecisionRecord under review/decisions/.
// Timestamp+verdict in the filename preserves chronological sort and lets
// operators grep the audit log without parsing JSON.
func (s *Service) writeDecision(initiativeName string, rec DecisionRecord) error {
	dir := filepath.Join(s.initStore.InitDir(initiativeName), "review", "decisions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create decisions dir: %w", err)
	}
	safeTS := strings.ReplaceAll(strings.ReplaceAll(rec.DecidedAt, ":", ""), "-", "")
	filename := fmt.Sprintf("%s-%s.json", safeTS, rec.Verdict)
	path := filepath.Join(dir, filename)
	value := any(rec)
	if redacted, changed, err := pathredact.NewForArtifactPath(path).RedactJSONValue(rec); err != nil {
		return fmt.Errorf("redact decision: %w", err)
	} else if changed {
		value = redacted
	}
	return storage.WriteJSONAtomic(path, value)
}

// ListDecisions returns all decision records for an initiative, newest-first,
// for audit/CLI consumers. Missing directory → empty slice, not an error.
func (s *Service) ListDecisions(initiativeName string) ([]DecisionRecord, error) {
	dir := filepath.Join(s.initStore.InitDir(initiativeName), "review", "decisions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read decisions dir: %w", err)
	}
	out := make([]DecisionRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var rec DecisionRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DecidedAt > out[j].DecidedAt })
	return out, nil
}

// --- Background polling ---------------------------------------------------

func (s *Service) trackActiveRound(initiativeName string, roundNum int, runID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeRounds[runID] = activeRound{
		InitiativeName: initiativeName,
		RoundNum:       roundNum,
		RunID:          runID,
	}
}

// RefreshGatheringRounds polls each tracked gathering round for terminal
// status and, on transition, flips the initiative to review_pending.
// Safe to call concurrently with Start/List; the internal map is guarded.
func (s *Service) RefreshGatheringRounds(ctx context.Context) {
	if s.inspector == nil {
		return
	}
	s.mu.Lock()
	snapshot := make(map[string]activeRound, len(s.activeRounds))
	for k, v := range s.activeRounds {
		snapshot[k] = v
	}
	s.mu.Unlock()

	for runID, ar := range snapshot {
		state, err := s.inspector.GetRunState(ctx, runID)
		if err != nil {
			continue
		}
		itemDir := s.initStore.InitDir(ar.InitiativeName)
		round, loadErr := review.LoadRound(itemDir, ar.RoundNum)
		if loadErr != nil || round == nil {
			s.mu.Lock()
			delete(s.activeRounds, runID)
			s.mu.Unlock()
			continue
		}
		finalized, changed := finalizeRoundIfTerminal(*round, state)
		if !changed {
			continue
		}
		if err := review.SaveRound(itemDir, finalized); err != nil {
			slog.Warn("initiative review: save round", "initiative", ar.InitiativeName, "round", ar.RoundNum, "err", err)
			continue
		}
		s.handleTerminalRound(ctx, ar.InitiativeName, finalized)
		s.mu.Lock()
		delete(s.activeRounds, runID)
		s.mu.Unlock()
	}
}

// StartBackgroundWorker polls gathering rounds on a 5-second tick until
// the stop channel is closed. Runs once per process, started from main.go.
func (s *Service) StartBackgroundWorker(stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.RefreshGatheringRounds(context.Background())
		}
	}
}

// RecoverActiveRounds scans every initiative for rounds in gathering state
// and re-populates the in-memory tracking map. Call this at startup so
// rounds spawned before a restart resume polling.
//
// Discovers initiatives itself via the injected InitiativeStore — callers
// must not pre-filter the list, otherwise initiatives created immediately
// before a crash (and thus absent from a cached name list) leak their
// gathering rounds until the next manual trigger.
func (s *Service) RecoverActiveRounds() {
	inits, err := s.initStore.LoadAll()
	if err != nil {
		slog.Warn("initiative review: list initiatives for recovery failed", "err", err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	recovered := 0
	for _, init := range inits {
		rounds, err := review.LoadRounds(s.initStore.InitDir(init.Name))
		if err != nil {
			continue
		}
		for _, r := range rounds {
			if r.Status == review.RoundStatusGathering && r.RunID != "" {
				s.activeRounds[r.RunID] = activeRound{
					InitiativeName: init.Name,
					RoundNum:       r.RoundNum,
					RunID:          r.RunID,
				}
				recovered++
			}
		}
	}
	if recovered > 0 {
		slog.Info("recovered active initiative review rounds", "count", recovered)
	}
}

// --- Helpers --------------------------------------------------------------

// finalizeRoundIfTerminal returns (newRound, true) when the agent state
// indicates the run has reached a terminal status; otherwise (round, false).
// Failure reasons are recorded on the round for UI rendering.
func finalizeRoundIfTerminal(round review.Round, state agentmanager.RunState) (review.Round, bool) {
	switch strings.ToLower(strings.TrimSpace(state.Status)) {
	case "complete":
		if err := validateCompletedRound(round); err != nil {
			round.Status = review.RoundStatusFailed
			round.FailureReason = err.Error()
			return round, true
		}
		round.Status = review.RoundStatusComplete
		round.FailureReason = ""
		return round, true
	case "failed":
		round.Status = review.RoundStatusFailed
		round.FailureReason = firstNonEmpty(state.ErrorMsg, "initiative review agent run failed")
		return round, true
	case "cancelled":
		round.Status = review.RoundStatusFailed
		round.FailureReason = firstNonEmpty(state.ErrorMsg, "initiative review agent run was cancelled")
		return round, true
	}
	return round, false
}

func validateCompletedRound(round review.Round) error {
	if strings.TrimSpace(round.AgentAssessment) == "" {
		return fmt.Errorf("review run completed without agent_assessment")
	}
	classification := strings.TrimSpace(round.Classification)
	if classification == "" {
		return fmt.Errorf("review run completed without classification")
	}
	switch classification {
	case "delivered", "partial", "failed":
		return nil
	}
	return fmt.Errorf("review run completed with invalid classification %q (want delivered|partial|failed)", classification)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func fallbackInitiativeTitle(init *initiatives.Initiative) string {
	if title := strings.TrimSpace(init.Title); title != "" {
		return title
	}
	return init.Name
}

// --- Instruction rendering -------------------------------------------------

func (s *Service) renderInstructions(ctx context.Context, init *initiatives.Initiative, roundNum int) (string, error) {
	vars := map[string]string{
		"INITIATIVE_NAME": init.Name,
		"ROUND_NUMBER":    fmt.Sprintf("%03d", roundNum),
	}
	return s.promptClient.ReadSkill(ctx, "swarm-manager-initiative-review", vars, true)
}

// --- Attachment assembly lives in context.go so this file stays focused on
// lifecycle.
