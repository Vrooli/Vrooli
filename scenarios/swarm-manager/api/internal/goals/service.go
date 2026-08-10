package goals

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/eta"
	"swarm-manager/internal/eventlog"
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
	EmitMilestoneCreated(goal, milestone string, payload eventlog.MilestonePayload)
	EmitMilestoneUpdated(goal, milestone string, payload eventlog.MilestonePayload)
	EmitMilestoneItemsAssigned(goal, milestone string, payload eventlog.MilestonePayload)
	EmitMilestoneItemsUnassigned(goal, milestone string, payload eventlog.MilestonePayload)
	EmitMilestoneArchived(goal, milestone string, payload eventlog.MilestonePayload)
}

// AIIndexer keeps a goal's semantic-search vector in sync without coupling
// this domain package to the aisearch implementation.
type AIIndexer interface {
	IndexGoal(context.Context, Goal) error
	DeleteGoal(context.Context, string) error
}

// Service provides business logic for goals.
type Service struct {
	store            *Store
	backlog          BacklogReader
	eventLogger      EventLogger
	eventDispatcher  dispatch.Invalidator
	estimatorFactory EstimatorFactory
	aiIndexer        AIIndexer
}

// NewService creates a goals Service.
func NewService(store *Store, backlogReader BacklogReader) *Service {
	return &Service{store: store, backlog: backlogReader}
}

// SetEventLogger injects an optional event logger.
func (s *Service) SetEventLogger(l EventLogger) { s.eventLogger = l }

// SetEventDispatcher injects an optional graph invalidation dispatcher.
func (s *Service) SetEventDispatcher(d dispatch.Invalidator) { s.eventDispatcher = d }

// SetEstimatorFactory injects an optional ETA estimator factory. When set, Get
// and List attach a p50/p80 completion band to each goal.
func (s *Service) SetEstimatorFactory(f EstimatorFactory) { s.estimatorFactory = f }

// SetAIIndexer configures optional best-effort semantic-search synchronization.
// Goal persistence is never blocked by an unavailable embedding service.
func (s *Service) SetAIIndexer(indexer AIIndexer) { s.aiIndexer = indexer }

func (s *Service) indexGoalAsync(goal Goal) {
	if s.aiIndexer == nil {
		return
	}
	go func() {
		if err := s.aiIndexer.IndexGoal(context.Background(), goal); err != nil {
			slog.Debug("[goals] semantic index upsert failed", "goal", goal.Name, "err", err)
		}
	}()
}

func (s *Service) deleteGoalAsync(name string) {
	if s.aiIndexer == nil {
		return
	}
	go func() {
		if err := s.aiIndexer.DeleteGoal(context.Background(), name); err != nil {
			slog.Debug("[goals] semantic index delete failed", "goal", name, "err", err)
		}
	}()
}

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

// ListFiles returns the files held by a goal. Only paths relative to the goal
// directory are exposed.
func (s *Service) ListFiles(name string) ([]File, error) { return s.store.ListFiles(name) }

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
		SpawnedFrom: req.SpawnedFrom,
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
	s.indexGoalAsync(*g)
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
	in, items, err := s.buildScopeData()
	if err != nil {
		return nil, err
	}
	in.Targets = g.Targets
	in.Milestones = g.Milestones
	scope := ComputeScope(in)
	s.recordDrift(g, scope)
	gws := &GoalWithScope{Goal: *g, Scope: scope}
	// Hydrate the rendered refs from the data the scope walk already loaded —
	// a map join, not extra I/O.
	gws.ScopeEntities = buildScopeEntities(g.Targets, scope, g.Milestones, items)
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
	in.Milestones = g.Milestones
	return ComputeScope(in).Closure, nil
}

// ItemGoalPriorities maps each item ref in an active goal's closure to the
// lowest numeric priority among the goals containing it. It backs the execution drain
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
		gin.Milestones = g.Milestones
		for _, ref := range ComputeScope(gin).Closure {
			if p, ok := out[ref]; !ok || g.Priority < p {
				out[ref] = g.Priority
			}
		}
	}
	return out, nil
}

// ReadyGoalItems returns the ready-to-run items across all active goals, lowest
// numeric goal priority first (then ref, for determinism). It backs the continuous
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
		gin.Milestones = g.Milestones
		for _, ref := range ComputeScope(gin).Ready {
			if p, ok := best[ref]; !ok || g.Priority < p {
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
			return out[i].GoalPriority < out[j].GoalPriority
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// LoadAllRaw returns every goal record without computing scope. Scope is a
// transitive closure over the whole backlog, so callers that only need names
// and versions — the workflow sweeper does — must not pay for it.
func (s *Service) LoadAllRaw() ([]Goal, error) { return s.store.LoadAll() }

// List returns all goals with computed scope and ETA bands. The scope input and
// estimator are built once and reused across goals.
func (s *Service) List() ([]GoalWithScope, error) {
	items, err := s.backlog.LoadAll(nil)
	if err != nil {
		return nil, fmt.Errorf("load backlog: %w", err)
	}
	listed, _, err := s.ListWithItemPriorities(items)
	return listed, err
}

// ListWithItemPriorities returns the scoped goal list and the item-to-priority
// map together, computed from backlog items the caller already loaded. They are
// two readings of one scope computation over one item set, so a caller that
// needs both — the operator inbox does — reads each store exactly once.
func (s *Service) ListWithItemPriorities(items []backlog.BacklogItem) ([]GoalWithScope, map[string]int, error) {
	goalsList, err := s.store.LoadAll()
	if err != nil {
		return nil, nil, err
	}
	in := scopeInputFrom(items)
	est := s.newEstimator()
	out := make([]GoalWithScope, 0, len(goalsList))
	priorities := make(map[string]int)
	for i := range goalsList {
		g := goalsList[i]
		goalIn := in
		goalIn.Targets = g.Targets
		goalIn.Milestones = g.Milestones
		scope := ComputeScope(goalIn)
		// Priority is an active-goal concept: a paused or completed goal must
		// not keep ranking items in the inbox.
		if g.Status == StatusActive {
			for _, ref := range scope.Closure {
				if p, ok := priorities[ref]; !ok || g.Priority > p {
					priorities[ref] = g.Priority
				}
			}
		}
		s.recordDrift(&g, scope)
		gws := GoalWithScope{Goal: g, Scope: scope}
		attachETA(&gws, goalIn, est)
		out = append(out, gws)
	}
	return out, priorities, nil
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
	s.indexGoalAsync(*g)
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
	s.indexGoalAsync(*g)
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
	s.indexGoalAsync(*g)
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
	s.indexGoalAsync(*g)
	s.invalidate()
	return g, nil
}

// CloseOut marks a goal achieved after independently verified milestone
// delivery. It intentionally does not infer delivery from member item status:
// the milestone review is the evidence authority for a goal-level outcome.
func (s *Service) CloseOut(name string) (*Goal, error) {
	g, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	if g.Status == StatusAchieved {
		return g, nil
	}
	if g.Status != StatusActive {
		return nil, validationErr("only active goals can be closed out")
	}
	if milestone := firstUnverifiedMilestone(g.Milestones); milestone != "" {
		return nil, validationErr("milestone %q is not verified delivered; review its criterion evidence before close-out", milestone)
	}
	g.Status = StatusAchieved
	g.Updated = nowRFC3339()
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	s.indexGoalAsync(*g)
	s.invalidate()
	return g, nil
}

// MarkMilestoneDeliveredWithVerdicts is the workflow-only delivery gate. A
// milestone cannot be independently verified unless every declared criterion
// is explicitly delivered with at least one evidence reference.
func (s *Service) MarkMilestoneDeliveredWithVerdicts(goalName, milestoneName string, verdicts []CriterionVerdict) (*GoalWithScope, error) {
	return s.markMilestoneDelivered(goalName, milestoneName, verdicts)
}

func (s *Service) markMilestoneDelivered(goalName, milestoneName string, verdicts []CriterionVerdict) (*GoalWithScope, error) {
	g, err := s.store.Load(goalName)
	if err != nil {
		return nil, err
	}
	index := milestoneIndex(g.Milestones, milestoneName)
	if index < 0 {
		return nil, validationErr("milestone %q not found", milestoneName)
	}
	if !coversMilestoneCriteria(g.Milestones[index].AcceptanceCriteria, verdicts) {
		return nil, validationErr("milestone review must deliver every acceptance criterion with evidence")
	}
	now := nowRFC3339()
	g.Milestones[index].VerifiedDeliveredAt = &now
	g.Milestones[index].CriterionVerdicts = append([]CriterionVerdict(nil), verdicts...)
	g.Updated = now
	scope, err := s.computeScope(g)
	if err != nil {
		return nil, err
	}
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	s.indexGoalAsync(*g)
	s.invalidate()
	return &GoalWithScope{Goal: *g, Scope: scope}, nil
}

func coversMilestoneCriteria(criteria []string, verdicts []CriterionVerdict) bool {
	if len(criteria) == 0 || len(verdicts) != len(criteria) {
		return false
	}
	seen := make(map[string]struct{}, len(verdicts))
	for _, verdict := range verdicts {
		if strings.TrimSpace(verdict.Verdict) != "delivered" || len(verdict.Evidence) == 0 {
			return false
		}
		seen[strings.TrimSpace(verdict.Criterion)] = struct{}{}
	}
	for _, criterion := range criteria {
		if _, ok := seen[strings.TrimSpace(criterion)]; !ok {
			return false
		}
	}
	return true
}

func allMilestonesVerifiedDelivered(milestones []Milestone) bool {
	return firstUnverifiedMilestone(milestones) == ""
}

func firstUnverifiedMilestone(milestones []Milestone) string {
	count := 0
	for _, milestone := range milestones {
		if milestone.ArchivedAt != nil {
			continue
		}
		count++
		if milestone.VerifiedDeliveredAt == nil || strings.TrimSpace(*milestone.VerifiedDeliveredAt) == "" {
			return milestone.Name
		}
	}
	if count == 0 {
		return "__none__"
	}
	return ""
}

// IsCloseOutReady reports whether the goal has evidence for the operator-only
// close-out action. It is intentionally read-only so projections never need to
// attempt a mutation merely to discover eligibility.
func IsCloseOutReady(goal Goal) bool {
	return goal.Status == StatusActive && allMilestonesVerifiedDelivered(goal.Milestones)
}

// Unarchive restores a goal to active status without discarding its history.
func (s *Service) Unarchive(name string) (*Goal, error) {
	g, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	if g.Status != StatusArchived {
		return g, nil
	}
	archivedAt := ""
	if g.ArchivedAt != nil {
		archivedAt = *g.ArchivedAt
	}
	g.Status = StatusActive
	g.ArchivedAt = nil
	g.Updated = nowRFC3339()
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitGoalUnarchived(name, archivedAt)
	}
	s.indexGoalAsync(*g)
	s.invalidate()
	return g, nil
}

// Delete removes a goal.
func (s *Service) Delete(name string) error {
	if err := s.store.Delete(name); err != nil {
		return err
	}
	s.deleteGoalAsync(name)
	s.invalidate()
	return nil
}

// CreateMilestone adds an owned milestone without changing derived scope.
func (s *Service) CreateMilestone(goalName string, milestone Milestone) (*GoalWithScope, error) {
	g, err := s.store.Load(goalName)
	if err != nil {
		return nil, err
	}
	if err := validateMilestone(milestone, g.Milestones, ""); err != nil {
		return nil, err
	}
	milestone = normalizeMilestone(milestone)
	g.Milestones = append(g.Milestones, milestone)
	g.Updated = nowRFC3339()
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitMilestoneCreated(goalName, milestone.Name, milestonePayload(goalName, milestone))
	}
	s.indexGoalAsync(*g)
	s.invalidate()
	return s.Get(goalName)
}

// UpdateMilestone replaces one milestone while preserving its ownership.
func (s *Service) UpdateMilestone(goalName string, milestone Milestone) (*GoalWithScope, error) {
	g, err := s.store.Load(goalName)
	if err != nil {
		return nil, err
	}
	index := milestoneIndex(g.Milestones, milestone.Name)
	if index < 0 {
		return nil, validationErr("milestone %q not found", milestone.Name)
	}
	if err := validateMilestone(milestone, g.Milestones, milestone.Name); err != nil {
		return nil, err
	}
	milestone = normalizeMilestone(milestone)
	milestone = carryServerOwnedMilestoneFields(g.Milestones[index], milestone)
	g.Milestones[index] = milestone
	g.Updated = nowRFC3339()
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitMilestoneUpdated(goalName, milestone.Name, milestonePayload(goalName, milestone))
	}
	s.indexGoalAsync(*g)
	s.invalidate()
	return s.Get(goalName)
}

// AssignMilestoneItems assigns closure items to exactly one owned milestone.
func (s *Service) AssignMilestoneItems(goalName, milestoneName string, items []string) (*GoalWithScope, error) {
	g, err := s.store.Load(goalName)
	if err != nil {
		return nil, err
	}
	index := milestoneIndex(g.Milestones, milestoneName)
	if index < 0 {
		return nil, validationErr("milestone %q not found", milestoneName)
	}
	scope, err := s.computeScope(g)
	if err != nil {
		return nil, err
	}
	valid := make(map[string]bool, len(scope.Closure))
	for _, ref := range scope.Closure {
		valid[ref] = true
	}
	for _, item := range items {
		if !valid[item] {
			return nil, validationErr("item %q is outside goal scope", item)
		}
	}
	for i := range g.Milestones {
		if i != index {
			g.Milestones[i].Items = withoutRefs(g.Milestones[i].Items, items)
		}
	}
	g.Milestones[index].Items = appendUnique(g.Milestones[index].Items, items)
	g.Updated = nowRFC3339()
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitMilestoneItemsAssigned(goalName, milestoneName, eventlog.MilestonePayload{GoalName: goalName, MilestoneName: milestoneName, Items: items})
	}
	s.indexGoalAsync(*g)
	s.invalidate()
	return s.Get(goalName)
}

// UnassignMilestoneItems returns items to the goal's unassigned bucket.
func (s *Service) UnassignMilestoneItems(goalName, milestoneName string, items []string) (*GoalWithScope, error) {
	g, err := s.store.Load(goalName)
	if err != nil {
		return nil, err
	}
	index := milestoneIndex(g.Milestones, milestoneName)
	if index < 0 {
		return nil, validationErr("milestone %q not found", milestoneName)
	}
	g.Milestones[index].Items = withoutRefs(g.Milestones[index].Items, items)
	g.Updated = nowRFC3339()
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitMilestoneItemsUnassigned(goalName, milestoneName, eventlog.MilestonePayload{GoalName: goalName, MilestoneName: milestoneName, Items: items})
	}
	s.indexGoalAsync(*g)
	s.invalidate()
	return s.Get(goalName)
}

// ArchiveMilestone preserves the milestone and its provenance while removing
// it from active work management.
func (s *Service) ArchiveMilestone(goalName, milestoneName string) (*GoalWithScope, error) {
	g, err := s.store.Load(goalName)
	if err != nil {
		return nil, err
	}
	index := milestoneIndex(g.Milestones, milestoneName)
	if index < 0 {
		return nil, validationErr("milestone %q not found", milestoneName)
	}
	if g.Milestones[index].ArchivedAt == nil {
		now := nowRFC3339()
		g.Milestones[index].ArchivedAt = &now
		g.Updated = now
	}
	if err := s.store.Save(g); err != nil {
		return nil, err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitMilestoneArchived(goalName, milestoneName, milestonePayload(goalName, g.Milestones[index]))
	}
	s.indexGoalAsync(*g)
	s.invalidate()
	return s.Get(goalName)
}

func milestoneIndex(milestones []Milestone, name string) int {
	for i := range milestones {
		if milestones[i].Name == name {
			return i
		}
	}
	return -1
}

// carryServerOwnedMilestoneFields merges the fields a milestone update may not
// set into the incoming definition. UpdateMilestone replaces the milestone
// wholesale, so any field the caller cannot express is otherwise erased:
// membership vanishes, an archived milestone reappears, and a verified-delivered
// stamp is lost — silently, because the payload looked complete.
//
// Membership and lifecycle are server-owned. Membership changes only through
// AssignMilestoneItems, which validates goal closure; archival only through
// ArchiveMilestone; delivery only through MarkMilestoneDelivered, whose stamp is
// the sole evidence CloseOut accepts.
//
// The one field deliberately NOT carried forward unconditionally is the
// verified-delivered stamp: a verdict is a judgement about a specific definition
// of done. When the acceptance criteria change, the old verdict no longer
// describes what the milestone now claims, so the stamp is cleared and the
// milestone must be reviewed again.
func carryServerOwnedMilestoneFields(previous, next Milestone) Milestone {
	next.Items = append([]string(nil), previous.Items...)
	next.ArchivedAt = previous.ArchivedAt
	if sameCriteria(previous.AcceptanceCriteria, next.AcceptanceCriteria) {
		next.VerifiedDeliveredAt = previous.VerifiedDeliveredAt
	} else {
		next.VerifiedDeliveredAt = nil
	}
	return next
}

func sameCriteria(previous, next []string) bool {
	if len(previous) != len(next) {
		return false
	}
	for i := range previous {
		if strings.TrimSpace(previous[i]) != strings.TrimSpace(next[i]) {
			return false
		}
	}
	return true
}

func normalizeMilestone(m Milestone) Milestone {
	m.Name = sanitizeName(m.Name)
	m.Title = strings.TrimSpace(m.Title)
	m.Description = strings.TrimSpace(m.Description)
	m.Items = appendUnique(nil, m.Items)
	m.DependsOn = appendUnique(nil, m.DependsOn)
	return m
}

func appendUnique(existing, values []string) []string {
	seen := make(map[string]bool, len(existing)+len(values))
	out := make([]string, 0, len(existing)+len(values))
	for _, value := range append(existing, values...) {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func withoutRefs(existing, removed []string) []string {
	drop := make(map[string]bool, len(removed))
	for _, ref := range removed {
		drop[strings.TrimSpace(ref)] = true
	}
	out := make([]string, 0, len(existing))
	for _, ref := range existing {
		if !drop[ref] {
			out = append(out, ref)
		}
	}
	return out
}

func milestonePayload(goal string, m Milestone) eventlog.MilestonePayload {
	return eventlog.MilestonePayload{GoalName: goal, MilestoneName: m.Name, Items: m.Items}
}

// hasAcceptanceCriteria reports whether the milestone carries at least one
// non-blank criterion.
func hasAcceptanceCriteria(m Milestone) bool {
	for _, criterion := range m.AcceptanceCriteria {
		if strings.TrimSpace(criterion) != "" {
			return true
		}
	}
	return false
}

func validateMilestone(m Milestone, existing []Milestone, replacing string) error {
	name := sanitizeName(m.Name)
	if name == "" {
		return validationErr("milestone name is required")
	}
	if strings.TrimSpace(m.Title) == "" {
		return validationErr("milestone title is required")
	}
	// A milestone is the only carrier of a goal's definition of done:
	// milestone review reads the criteria, and CloseOut is gated on that
	// review's verdict. Creating or replacing one without criteria produces a
	// milestone that can never be reviewed, so the goal can never be achieved.
	// Both write paths replace the definition wholesale, so both must state it.
	if !hasAcceptanceCriteria(m) {
		return validationErr("milestone acceptance criteria are required: without them the milestone can never be reviewed and its goal can never be closed out")
	}
	names := make(map[string]bool, len(existing)+1)
	for _, candidate := range existing {
		if candidate.Name != replacing {
			names[candidate.Name] = true
		}
	}
	if names[name] {
		return validationErr("milestone %q already exists", name)
	}
	names[name] = true
	for _, dependency := range m.DependsOn {
		dependency = strings.TrimSpace(dependency)
		if dependency == "" || dependency == name || !names[dependency] {
			return validationErr("milestone dependency %q must name a sibling milestone", dependency)
		}
	}
	return nil
}

// computeScope builds the scope input from live backlog and runs the pure
// ComputeScope.
func (s *Service) computeScope(g *Goal) (Scope, error) {
	in, err := s.buildScopeInput()
	if err != nil {
		return Scope{}, err
	}
	in.Targets = g.Targets
	in.Milestones = g.Milestones
	return ComputeScope(in), nil
}

func (s *Service) buildScopeInput() (ScopeInput, error) {
	in, _, err := s.buildScopeData()
	return in, err
}

// buildScopeData builds the scope input and also returns the raw items for
// read-time hydration (ScopeEntities).
func (s *Service) buildScopeData() (ScopeInput, []backlog.BacklogItem, error) {
	items, err := s.backlog.LoadAll(nil)
	if err != nil {
		return ScopeInput{}, nil, fmt.Errorf("load backlog: %w", err)
	}
	return scopeInputFrom(items), items, nil
}

// scopeInputFrom builds the scope index from items the caller already holds.
func scopeInputFrom(items []backlog.BacklogItem) ScopeInput {
	in := ScopeInput{
		ItemDeps:   make(map[string][]string, len(items)),
		ItemStatus: make(map[string]string, len(items)),
		ItemEffort: make(map[string]string, len(items)),
	}
	for _, it := range items {
		ref := string(it.Kind) + "/" + it.Name
		in.ItemStatus[ref] = string(it.Status)
		in.ItemDeps[ref] = it.DependsOn
		in.ItemEffort[ref] = it.Effort
	}
	return in
}

// buildScopeEntities hydrates the refs the goal detail view renders (targets ∪
// ready ∪ blocked ∪ milestone assignments) from the already-loaded items. Returns nil
// when nothing resolves so the field is omitted from JSON.
func buildScopeEntities(targets []string, scope Scope, milestones []Milestone, items []backlog.BacklogItem) *ScopeEntities {
	wanted := make(map[string]bool, len(targets)+len(scope.Ready)+len(scope.Blocked))
	for _, refs := range [][]string{targets, scope.Ready, scope.Blocked} {
		for _, ref := range refs {
			wanted[ref] = true
		}
	}
	for _, milestone := range milestones {
		for _, ref := range milestone.Items {
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
		if it, ok := itemsByRef[ref]; ok {
			if out.Items == nil {
				out.Items = make(map[string]backlog.BacklogItem)
			}
			out.Items[ref] = it
		}
	}
	if out.Items == nil {
		return nil
	}
	return out
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
		if strings.HasPrefix(t, "initiative/") {
			return nil, validationErr("legacy initiative target %q is no longer supported; use its item roots", t)
		}
		if !strings.Contains(t, "/") {
			return nil, validationErr("target %q must be '<kind>/<name>'", t)
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
