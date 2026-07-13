package goals

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/eta"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/initiatives"
)

// ErrValidation wraps user-correctable validation failures so the handler can
// return 400 rather than 500.
var ErrValidation = errors.New("goal validation error")

func validationErr(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrValidation}, args...)...)
}

// BacklogReader loads backlog items for scope computation.
type BacklogReader interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// InitiativeReader loads initiatives for target expansion and gate semantics.
type InitiativeReader interface {
	LoadAll() ([]initiatives.Initiative, error)
}

// EventLogger records goal state-change events. *eventlog.Emitter satisfies it.
type EventLogger interface {
	EmitGoalCreated(name string, payload eventlog.GoalCreatedPayload)
	EmitGoalUpdated(name string)
	EmitGoalTargetAdded(name, target string)
	EmitGoalTargetRemoved(name, target string)
	EmitGoalPriorityChanged(name string, from, to int)
	EmitGoalArchived(name, previousStatus, archivedAt string)
	EmitGoalUnarchived(name, archivedAt string)
	EmitGoalScopeSnapshot(name string, payload eventlog.GoalScopeSnapshotPayload)
}

// Service provides business logic for goals.
type Service struct {
	store            *Store
	backlog          BacklogReader
	initiatives      InitiativeReader
	eventLogger      EventLogger
	eventDispatcher  dispatch.Invalidator
	estimatorFactory EstimatorFactory
}

// NewService creates a goals Service.
func NewService(store *Store, backlogReader BacklogReader, initiativeReader InitiativeReader) *Service {
	return &Service{store: store, backlog: backlogReader, initiatives: initiativeReader}
}

// SetEventLogger injects an optional event logger.
func (s *Service) SetEventLogger(l EventLogger) { s.eventLogger = l }

// SetEventDispatcher injects an optional graph invalidation dispatcher.
func (s *Service) SetEventDispatcher(d dispatch.Invalidator) { s.eventDispatcher = d }

// SetEstimatorFactory injects an optional ETA estimator factory. When set, Get
// and List attach a p50/p80 completion band to each goal.
func (s *Service) SetEstimatorFactory(f EstimatorFactory) { s.estimatorFactory = f }

// newEstimator builds a fresh estimator via the factory, tolerating a nil
// factory or a build error by returning nil (ETA is then simply omitted).
func (s *Service) newEstimator() *eta.Estimator {
	if s.estimatorFactory == nil {
		return nil
	}
	est, err := s.estimatorFactory()
	if err != nil {
		return nil
	}
	return est
}

// GoalDir exposes the store path for file handlers.
func (s *Service) GoalDir(name string) string { return s.store.GoalDir(name) }

// Create validates and persists a new goal, recording a baseline scope
// snapshot. Returns the goal with its computed scope.
func (s *Service) Create(req CreateRequest) (*GoalWithScope, error) {
	name := sanitizeName(req.Name)
	if name == "" {
		name = sanitizeName(req.Title)
	}
	if name == "" {
		return nil, validationErr("goal name or title is required")
	}
	if s.store.Exists(name) {
		return nil, validationErr("goal %q already exists", name)
	}
	if req.Priority < 0 || req.Priority > 10 {
		return nil, validationErr("priority must be between 0 and 10")
	}
	targets, err := normalizeTargets(req.Targets)
	if err != nil {
		return nil, err
	}

	now := nowRFC3339()
	g := &Goal{
		Name:        name,
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      StatusActive,
		Priority:    req.Priority,
		Targets:     targets,
		Seeded:      req.Seeded,
		Created:     now,
		Updated:     now,
	}
	if g.Title == "" {
		g.Title = name
	}

	scope, err := s.computeScope(g)
	if err != nil {
		return nil, err
	}
	g.ScopeHistory = []ScopeSnapshot{scopeSnapshot(now, scope)}

	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitGoalCreated(name, eventlog.GoalCreatedPayload{
			Title:    g.Title,
			Priority: g.Priority,
			Targets:  g.Targets,
			Seeded:   g.Seeded,
		})
		s.eventLogger.EmitGoalScopeSnapshot(name, scopeSnapshotPayload(scope))
	}
	s.invalidate()
	return &GoalWithScope{Goal: *g, Scope: scope}, nil
}

// Get returns a goal with its freshly computed scope, recording a scope
// snapshot if the closure has drifted since the last one (creep tracking), and
// attaches its ETA band when an estimator is wired.
func (s *Service) Get(name string) (*GoalWithScope, error) {
	g, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	in, items, inits, err := s.buildScopeData()
	if err != nil {
		return nil, err
	}
	in.Targets = g.Targets
	scope := ComputeScope(in)
	s.recordDrift(g, scope)
	gws := &GoalWithScope{Goal: *g, Scope: scope}
	// Hydrate the rendered refs from the data the scope walk already loaded —
	// a map join, not extra I/O.
	gws.ScopeEntities = buildScopeEntities(g.Targets, scope, items, inits)
	attachETA(gws, in, s.newEstimator())
	return gws, nil
}

// ClosureRefs returns the item refs ("<kind>/<name>") in a goal's transitive
// prerequisite closure (known backlog items only), without recording drift or
// attaching an ETA. It is the lightweight read planview uses to scope the board
// to a goal, so it must stay side-effect free (board polls call it often).
func (s *Service) ClosureRefs(name string) ([]string, error) {
	g, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	in, err := s.buildScopeInput()
	if err != nil {
		return nil, err
	}
	in.Targets = g.Targets
	return ComputeScope(in).Closure, nil
}

// ItemGoalPriorities maps each item ref in an active goal's closure to the
// highest priority among the goals containing it. It backs the execution drain
// comparator (goal-priority-first, FIFO fallback). Side-effect free.
func (s *Service) ItemGoalPriorities() (map[string]int, error) {
	goalsList, err := s.store.LoadAll()
	if err != nil {
		return nil, err
	}
	in, err := s.buildScopeInput()
	if err != nil {
		return nil, err
	}
	out := make(map[string]int)
	for i := range goalsList {
		g := goalsList[i]
		if g.Status != StatusActive {
			continue
		}
		gin := in
		gin.Targets = g.Targets
		for _, ref := range ComputeScope(gin).Closure {
			if p, ok := out[ref]; !ok || g.Priority > p {
				out[ref] = g.Priority
			}
		}
	}
	return out, nil
}

// ReadyGoalItems returns the ready-to-run items across all active goals, highest
// goal priority first (then ref, for determinism). It backs the continuous
// auto-enqueue drain. Side-effect free.
func (s *Service) ReadyGoalItems() ([]ReadyGoalItem, error) {
	goalsList, err := s.store.LoadAll()
	if err != nil {
		return nil, err
	}
	in, err := s.buildScopeInput()
	if err != nil {
		return nil, err
	}
	best := make(map[string]int)
	for i := range goalsList {
		g := goalsList[i]
		if g.Status != StatusActive {
			continue
		}
		gin := in
		gin.Targets = g.Targets
		for _, ref := range ComputeScope(gin).Ready {
			if p, ok := best[ref]; !ok || g.Priority > p {
				best[ref] = g.Priority
			}
		}
	}
	out := make([]ReadyGoalItem, 0, len(best))
	for ref, prio := range best {
		kind, name, ok := strings.Cut(ref, "/")
		if !ok || kind == "" || name == "" {
			continue
		}
		out = append(out, ReadyGoalItem{Kind: kind, Name: name, GoalPriority: prio})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GoalPriority != out[j].GoalPriority {
			return out[i].GoalPriority > out[j].GoalPriority
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// List returns all goals with computed scope and ETA bands. The scope input and
// estimator are built once and reused across goals.
func (s *Service) List() ([]GoalWithScope, error) {
	goalsList, err := s.store.LoadAll()
	if err != nil {
		return nil, err
	}
	in, err := s.buildScopeInput()
	if err != nil {
		return nil, err
	}
	est := s.newEstimator()
	out := make([]GoalWithScope, 0, len(goalsList))
	for i := range goalsList {
		g := goalsList[i]
		goalIn := in
		goalIn.Targets = g.Targets
		scope := ComputeScope(goalIn)
		s.recordDrift(&g, scope)
		gws := GoalWithScope{Goal: g, Scope: scope}
		attachETA(&gws, goalIn, est)
		out = append(out, gws)
	}
	return out, nil
}

// attachETA computes and attaches the ETA band for a goal, tolerating a nil
// estimator (band omitted) or an empty closure (no items to estimate).
func attachETA(gws *GoalWithScope, in ScopeInput, est *eta.Estimator) {
	if est == nil {
		return
	}
	if band, ok := est.EstimateGoal(closureInput(gws.Scope, in)); ok {
		gws.ETA = &band
	}
}

// Update mutates goal fields. Target changes re-snapshot scope.
func (s *Service) Update(name string, req UpdateRequest) (*GoalWithScope, error) {
	g, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	if !req.HasChanges() {
		return nil, validationErr("no fields to update")
	}
	if req.Title != nil {
		g.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		g.Description = strings.TrimSpace(*req.Description)
	}
	if req.Priority != nil {
		if *req.Priority < 0 || *req.Priority > 10 {
			return nil, validationErr("priority must be between 0 and 10")
		}
		if *req.Priority != g.Priority && s.eventLogger != nil {
			s.eventLogger.EmitGoalPriorityChanged(name, g.Priority, *req.Priority)
		}
		g.Priority = *req.Priority
	}
	if req.Targets != nil {
		targets, err := normalizeTargets(*req.Targets)
		if err != nil {
			return nil, err
		}
		g.Targets = targets
	}
	g.Updated = nowRFC3339()

	scope, err := s.computeScope(g)
	if err != nil {
		return nil, err
	}
	if req.Targets != nil {
		s.appendSnapshot(g, scope)
	}
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitGoalUpdated(name)
	}
	s.invalidate()
	return &GoalWithScope{Goal: *g, Scope: scope}, nil
}

// AddTargets adds one or more targets, ignoring duplicates.
func (s *Service) AddTargets(name string, targets []string) (*GoalWithScope, error) {
	g, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeTargets(targets)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]bool, len(g.Targets))
	for _, t := range g.Targets {
		existing[t] = true
	}
	added := false
	for _, t := range normalized {
		if existing[t] {
			continue
		}
		g.Targets = append(g.Targets, t)
		existing[t] = true
		added = true
		if s.eventLogger != nil {
			s.eventLogger.EmitGoalTargetAdded(name, t)
		}
	}
	if !added {
		return s.Get(name)
	}
	g.Updated = nowRFC3339()
	scope, err := s.computeScope(g)
	if err != nil {
		return nil, err
	}
	s.appendSnapshot(g, scope)
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	s.invalidate()
	return &GoalWithScope{Goal: *g, Scope: scope}, nil
}

// RemoveTargets removes targets that are present.
func (s *Service) RemoveTargets(name string, targets []string) (*GoalWithScope, error) {
	g, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	remove := make(map[string]bool, len(targets))
	for _, t := range targets {
		remove[strings.TrimSpace(t)] = true
	}
	kept := g.Targets[:0]
	removed := false
	for _, t := range g.Targets {
		if remove[t] {
			removed = true
			if s.eventLogger != nil {
				s.eventLogger.EmitGoalTargetRemoved(name, t)
			}
			continue
		}
		kept = append(kept, t)
	}
	g.Targets = append([]string(nil), kept...)
	if !removed {
		return s.Get(name)
	}
	g.Updated = nowRFC3339()
	scope, err := s.computeScope(g)
	if err != nil {
		return nil, err
	}
	s.appendSnapshot(g, scope)
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	s.invalidate()
	return &GoalWithScope{Goal: *g, Scope: scope}, nil
}

// Archive marks a goal archived.
func (s *Service) Archive(name string) (*Goal, error) {
	g, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	if g.Status == StatusArchived {
		return g, nil
	}
	prev := g.Status
	now := nowRFC3339()
	g.Status = StatusArchived
	g.ArchivedAt = &now
	g.Updated = now
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitGoalArchived(name, prev, now)
	}
	s.invalidate()
	return g, nil
}

// Delete removes a goal.
func (s *Service) Delete(name string) error {
	if err := s.store.Delete(name); err != nil {
		return err
	}
	s.invalidate()
	return nil
}

// computeScope builds the scope input from live backlog + initiatives and runs
// the pure ComputeScope.
func (s *Service) computeScope(g *Goal) (Scope, error) {
	in, err := s.buildScopeInput()
	if err != nil {
		return Scope{}, err
	}
	in.Targets = g.Targets
	return ComputeScope(in), nil
}

func (s *Service) buildScopeInput() (ScopeInput, error) {
	in, _, _, err := s.buildScopeData()
	return in, err
}

// buildScopeData builds the scope input and also returns the raw items and
// initiatives it was built from, for read-time hydration (ScopeEntities).
func (s *Service) buildScopeData() (ScopeInput, []backlog.BacklogItem, []initiatives.Initiative, error) {
	items, err := s.backlog.LoadAll(nil)
	if err != nil {
		return ScopeInput{}, nil, nil, fmt.Errorf("load backlog: %w", err)
	}
	inits, err := s.initiatives.LoadAll()
	if err != nil {
		return ScopeInput{}, nil, nil, fmt.Errorf("load initiatives: %w", err)
	}
	in := ScopeInput{
		ItemDeps:        make(map[string][]string, len(items)),
		ItemStatus:      make(map[string]string, len(items)),
		ItemEffort:      make(map[string]string, len(items)),
		InitiativeItems: make(map[string][]string, len(inits)),
		InitiativeDeps:  make(map[string][]string, len(inits)),
	}
	for _, it := range items {
		ref := string(it.Kind) + "/" + it.Name
		in.ItemStatus[ref] = string(it.Status)
		in.ItemDeps[ref] = it.DependsOn
		in.ItemEffort[ref] = it.Effort
	}
	for _, ini := range inits {
		in.InitiativeItems[ini.Name] = ini.Items
		in.InitiativeDeps[ini.Name] = ini.DependsOn
	}
	return in, items, inits, nil
}

// buildScopeEntities hydrates the refs the goal detail view renders (targets ∪
// ready ∪ blocked) from the already-loaded items and initiatives. Returns nil
// when nothing resolves so the field is omitted from JSON.
func buildScopeEntities(targets []string, scope Scope, items []backlog.BacklogItem, inits []initiatives.Initiative) *ScopeEntities {
	wanted := make(map[string]bool, len(targets)+len(scope.Ready)+len(scope.Blocked))
	for _, refs := range [][]string{targets, scope.Ready, scope.Blocked} {
		for _, ref := range refs {
			wanted[ref] = true
		}
	}
	if len(wanted) == 0 {
		return nil
	}

	itemsByRef := make(map[string]backlog.BacklogItem, len(items))
	for _, it := range items {
		itemsByRef[string(it.Kind)+"/"+it.Name] = it
	}

	out := &ScopeEntities{}
	for ref := range wanted {
		if IsInitiativeTarget(ref) {
			continue
		}
		if it, ok := itemsByRef[ref]; ok {
			if out.Items == nil {
				out.Items = make(map[string]backlog.BacklogItem)
			}
			out.Items[ref] = it
		}
	}
	for i := range inits {
		ref := initiativeNode(inits[i].Name)
		if !wanted[ref] {
			continue
		}
		if out.Initiatives == nil {
			out.Initiatives = make(map[string]InitiativeSummary)
		}
		out.Initiatives[ref] = InitiativeSummary{
			Initiative: inits[i],
			Rollup:     rollupFromItems(inits[i], itemsByRef),
		}
	}
	if out.Items == nil && out.Initiatives == nil {
		return nil
	}
	return out
}

// rollupFromItems mirrors the initiatives domain's rollup semantics
// (aggregateInitiativeData) over the in-memory item set: unknown refs count as
// pending; archived items count as archived (and completed when they finished).
func rollupFromItems(ini initiatives.Initiative, itemsByRef map[string]backlog.BacklogItem) initiatives.RollupStatus {
	r := initiatives.RollupStatus{Total: len(ini.Items)}
	for _, ref := range ini.Items {
		it, ok := itemsByRef[ref]
		if !ok {
			r.Pending++
			continue
		}
		if backlog.IsArchived(it) {
			r.Archived++
			if it.Status == backlog.StatusCompleted {
				r.Completed++
			}
			continue
		}
		switch it.Status {
		case backlog.StatusCompleted:
			r.Completed++
		case backlog.StatusFailed:
			r.Failed++
		case backlog.StatusInProgress, backlog.StatusQueued, backlog.StatusResearching:
			r.InProgress++
		default:
			r.Pending++
		}
	}
	return r
}

// recordDrift appends a scope snapshot (and emits an event) only when the
// closure size has changed since the last recorded snapshot, then persists.
func (s *Service) recordDrift(g *Goal, scope Scope) {
	if n := len(g.ScopeHistory); n > 0 && g.ScopeHistory[n-1].ClosureSize == scope.Total {
		return
	}
	s.appendSnapshot(g, scope)
	// Persist the drift snapshot; a read that surfaces creep should durably
	// record it. Best-effort: a save failure must not fail the read.
	_ = s.store.Save(g)
	if s.eventLogger != nil {
		s.eventLogger.EmitGoalScopeSnapshot(g.Name, scopeSnapshotPayload(scope))
	}
}

// appendSnapshot appends a scope snapshot to the goal's history (in memory).
func (s *Service) appendSnapshot(g *Goal, scope Scope) {
	g.ScopeHistory = append(g.ScopeHistory, scopeSnapshot(nowRFC3339(), scope))
}

func (s *Service) invalidate() {
	if s.eventDispatcher != nil {
		s.eventDispatcher.DispatchInvalidate("topology", "plan")
	}
}

func scopeSnapshot(at string, scope Scope) ScopeSnapshot {
	return ScopeSnapshot{
		At:          at,
		TargetCount: len(scope.Targets),
		ClosureSize: scope.Total,
		Completed:   scope.CompletedCount,
	}
}

func scopeSnapshotPayload(scope Scope) eventlog.GoalScopeSnapshotPayload {
	return eventlog.GoalScopeSnapshotPayload{
		TargetCount:    len(scope.Targets),
		ClosureSize:    scope.Total,
		CompletedCount: scope.CompletedCount,
		BlockedCount:   scope.BlockedCount,
	}
}

// normalizeTargets trims, de-dupes, and validates target refs.
func normalizeTargets(raw []string) ([]string, error) {
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if IsInitiativeTarget(t) {
			if strings.TrimSpace(InitiativeName(t)) == "" {
				return nil, validationErr("invalid initiative target %q", t)
			}
		} else if !strings.Contains(t, "/") {
			return nil, validationErr("target %q must be '<kind>/<name>' or 'initiative/<name>'", t)
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out, nil
}

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }
