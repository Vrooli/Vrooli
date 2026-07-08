package stats

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eta"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/initiatives"
)

// indexByteFast returns the index of the first occurrence of c in s,
// or -1 if absent. Used to parse kind from entity IDs of the form
// "<kind>/<name>" without dragging in the strings package for one call.
func indexByteFast(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

const refreshBatchSize = 5000

// ErrGoalScope wraps failures to resolve goal-scoped stats requests (goal
// scoping unavailable or the named goal not found).
var ErrGoalScope = errors.New("stats: goal scope")

// Engine incrementally aggregates events into metrics using a watermark pattern.
type Engine struct {
	mu        sync.RWMutex
	watermark int64
	repo      eventlog.Repository
	state     *aggregateState
	cfg       Config
}

// ETAEstimatorFactory builds a fresh estimator for a stats read.
type ETAEstimatorFactory func() (*eta.Estimator, error)

// BacklogLister loads backlog items for ETA closure construction.
type BacklogLister interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// InitiativeLister loads initiatives for ETA initiative-gate parity with planview.
type InitiativeLister interface {
	LoadAll() ([]initiatives.Initiative, error)
}

// GoalScoper resolves a goal name to the item refs ("<kind>/<name>") in its
// transitive prerequisite closure.
type GoalScoper interface {
	ClosureRefs(name string) ([]string, error)
}

// Config wires optional read models used for ETA. Metrics still degrade to the
// event-log aggregate when these are absent.
type Config struct {
	Backlog     BacklogLister
	Initiatives InitiativeLister
	Goals       GoalScoper
	ETA         ETAEstimatorFactory
}

// Params controls optional stats scoping.
type Params struct {
	Goal string
}

// NewEngine creates a stats engine backed by the given event repository.
func NewEngine(repo eventlog.Repository, configs ...Config) *Engine {
	var cfg Config
	if len(configs) > 0 {
		cfg = configs[0]
	}
	return &Engine{
		repo:  repo,
		state: newAggregateState(),
		cfg:   cfg,
	}
}

// Configure installs optional read-model dependencies after route setup has
// built them. It is safe to call once during startup.
func (e *Engine) Configure(cfg Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg = cfg
}

// Rebuild replays all events from scratch. Called once at startup.
func (e *Engine) Rebuild(ctx context.Context) error {
	events, err := e.repo.All(ctx)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	e.state = newAggregateState()
	for i := range events {
		e.state.processEvent(&events[i])
	}

	maxID, err := e.repo.MaxID(ctx)
	if err != nil {
		return err
	}
	e.watermark = maxID
	return nil
}

// Refresh incrementally processes events appended since the last watermark.
func (e *Engine) Refresh(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for {
		events, err := e.repo.Since(ctx, e.watermark, refreshBatchSize)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		for i := range events {
			e.state.processEvent(&events[i])
		}
		e.watermark = events[len(events)-1].ID
		if len(events) < refreshBatchSize {
			return nil
		}
	}
}

// GetStats returns the current computed metrics. Callers should call Refresh first.
func (e *Engine) GetStats() StatsResponse {
	return e.GetStatsContext(context.Background())
}

// GetStatsContext returns the current metrics and computes optional ETA data
// with the caller context.
func (e *Engine) GetStatsContext(ctx context.Context) StatsResponse {
	resp, _ := e.GetStatsForParams(ctx, Params{})
	return resp
}

// GetStatsForParams returns metrics for the requested scope. Empty params use
// the hot aggregate; goal-scoped requests replay a filtered event aggregate so
// every section is scoped uniformly.
func (e *Engine) GetStatsForParams(ctx context.Context, params Params) (StatsResponse, error) {
	e.mu.RLock()
	state := e.state
	cfg := e.cfg
	var resp StatsResponse
	if params.Goal == "" {
		resp = state.buildResponse()
	}
	e.mu.RUnlock()

	var inScope map[string]bool
	if params.Goal != "" {
		var err error
		inScope, err = resolveGoalScope(cfg, params.Goal)
		if err != nil {
			return StatsResponse{}, err
		}
		if e.repo == nil {
			return StatsResponse{}, fmt.Errorf("stats: goal scope requires event repository")
		}
		events, err := e.repo.All(ctx)
		if err != nil {
			return StatsResponse{}, err
		}
		scoped := newAggregateState()
		for _, ev := range filterEventsToScope(events, inScope) {
			scoped.processEvent(&ev)
		}
		resp = scoped.buildResponse()
	}

	resp.Dashboard.EstimatedRemaining = estimateRemaining(ctx, cfg, inScope)
	return resp, nil
}

func resolveGoalScope(cfg Config, goal string) (map[string]bool, error) {
	if cfg.Goals == nil {
		return nil, fmt.Errorf("%w: goal scoping unavailable", ErrGoalScope)
	}
	refs, err := cfg.Goals.ClosureRefs(goal)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGoalScope, err)
	}
	inScope := make(map[string]bool, len(refs))
	for _, ref := range refs {
		inScope[ref] = true
	}
	return inScope, nil
}

func estimateRemaining(ctx context.Context, cfg Config, inScope map[string]bool) *eta.Band {
	if cfg.Backlog == nil || cfg.ETA == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return nil
	}
	items, err := cfg.Backlog.LoadAll(nil)
	if err != nil {
		return nil
	}
	if inScope != nil {
		items = filterBacklogItemsToScope(items, inScope)
	}
	var inits []initiatives.Initiative
	if cfg.Initiatives != nil {
		inits, err = cfg.Initiatives.LoadAll()
		if err != nil {
			return nil
		}
	}
	if err := ctx.Err(); err != nil {
		return nil
	}
	est, err := cfg.ETA()
	if err != nil || est == nil {
		return nil
	}
	in := eta.BuildClosureInput(items, inits)
	band, ok := est.EstimateGoal(in)
	if !ok {
		return nil
	}
	return &band
}

func filterBacklogItemsToScope(items []backlog.BacklogItem, inScope map[string]bool) []backlog.BacklogItem {
	out := make([]backlog.BacklogItem, 0, len(inScope))
	for _, item := range items {
		if inScope[string(item.Kind)+"/"+item.Name] {
			out = append(out, item)
		}
	}
	return out
}

func filterEventsToScope(events []eventlog.Event, inScope map[string]bool) []eventlog.Event {
	scopedExecutions := make(map[string]bool)
	scopedInitiatives := make(map[string]bool)
	scopedRecords := make(map[string]bool)

	for i := range events {
		e := &events[i]
		switch e.EventType {
		case eventlog.EventExecutionCreated:
			var p eventlog.ExecutionCreatedPayload
			if unmarshalMeta(e.Metadata, &p) && inScope[p.BacklogKind+"/"+p.BacklogName] {
				scopedExecutions[e.EntityID] = true
			}
		case eventlog.EventInitiativeItemAdded:
			var p eventlog.InitiativeItemPayload
			if unmarshalMeta(e.Metadata, &p) && inScope[p.Item] {
				scopedInitiatives[e.EntityID] = true
			}
		case eventlog.EventBacklogCreated:
			var p eventlog.BacklogCreatedPayload
			if unmarshalMeta(e.Metadata, &p) && inScope[e.EntityID] && p.Initiative != "" {
				scopedInitiatives[p.Initiative] = true
			}
		case eventlog.EventBacklogInitiativeChanged:
			var p eventlog.InitiativeChangePayload
			if unmarshalMeta(e.Metadata, &p) && inScope[e.EntityID] {
				if p.From != "" {
					scopedInitiatives[p.From] = true
				}
				if p.To != "" {
					scopedInitiatives[p.To] = true
				}
			}
		case eventlog.EventRecordCreated:
			var p eventlog.RecordCreatedPayload
			if unmarshalMeta(e.Metadata, &p) && inScope[p.BacklogRef] {
				scopedRecords[e.EntityID] = true
			}
		case eventlog.EventOperatingModeBacklogSynced:
			var p eventlog.OperatingModeBacklogSyncPayload
			if unmarshalMeta(e.Metadata, &p) {
				for _, ref := range p.ItemRefs {
					if inScope[ref] {
						if p.InitiativeName != "" {
							scopedInitiatives[p.InitiativeName] = true
						}
						if p.ScopeID != "" {
							scopedInitiatives[p.ScopeID] = true
						}
						break
					}
				}
			}
		}
	}

	out := make([]eventlog.Event, 0, len(events))
	for _, e := range events {
		switch e.EntityType {
		case eventlog.EntityBacklogItem, eventlog.EntityQueue:
			if inScope[e.EntityID] {
				out = append(out, e)
			}
		case eventlog.EntityExecution:
			if scopedExecutions[e.EntityID] {
				out = append(out, e)
			}
		case eventlog.EntityInitiative:
			if scopedInitiatives[e.EntityID] {
				out = append(out, e)
			}
		case eventlog.EntityRecord:
			if scopedRecords[e.EntityID] {
				out = append(out, e)
			}
		}
	}
	return out
}
