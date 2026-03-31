package initiatives

import (
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
)

// BacklogLoader loads individual backlog items for rollup computation.
type BacklogLoader interface {
	LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error)
}

// EventDispatcher emits graph invalidation events for graph projections.
type EventDispatcher interface {
	DispatchInvalidate(lenses ...string)
}

// EventLogger records initiative state-change events for analytics.
type EventLogger interface {
	EmitInitiativeCreated(name string)
	EmitInitiativeItemAdded(name, item string)
	EmitInitiativeItemRemoved(name, item string)
	EmitInitiativeStatusChanged(name, from, to string)
	EmitInitiativeArchived(name string)
}

// Service provides business logic for initiative operations.
type Service struct {
	store           *Store
	backlogLoader   BacklogLoader
	eventDispatcher EventDispatcher
	eventLogger     EventLogger
}

// NewService creates a Service with the given store and backlog loader.
func NewService(store *Store, backlogLoader BacklogLoader) *Service {
	return &Service{
		store:         store,
		backlogLoader: backlogLoader,
	}
}

// SetEventDispatcher injects an optional graph invalidation dispatcher.
func (s *Service) SetEventDispatcher(d EventDispatcher) {
	s.eventDispatcher = d
}

// SetEventLogger injects an optional event logger for analytics tracking.
func (s *Service) SetEventLogger(l EventLogger) {
	s.eventLogger = l
}

// InitDir returns the absolute path for an initiative's folder, delegating to
// the store. This is used by file management handlers.
func (s *Service) InitDir(name string) string {
	return s.store.InitDir(name)
}

func (s *Service) invalidateTopologyGraph() {
	if s.eventDispatcher == nil {
		return
	}
	s.eventDispatcher.DispatchInvalidate("topology")
}

// Create creates a new initiative.
func (s *Service) Create(req CreateRequest) (*Initiative, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if s.store.Exists(name) {
		return nil, fmt.Errorf("initiative %q already exists", name)
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}
	if !ValidateStatus(status) {
		return nil, fmt.Errorf("invalid status %q: must be active, completed, or archived", req.Status)
	}

	items := req.Items
	if items == nil {
		items = []string{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	init := &Initiative{
		Name:        name,
		Title:       title,
		Description: strings.TrimSpace(req.Description),
		Status:      status,
		Items:       items,
		Created:     now,
		Updated:     now,
	}
	if err := s.store.Save(init); err != nil {
		return nil, fmt.Errorf("save initiative: %w", err)
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeCreated(name)
		for _, item := range items {
			s.eventLogger.EmitInitiativeItemAdded(name, item)
		}
	}
	s.invalidateTopologyGraph()
	return init, nil
}

// Get loads an initiative with its computed rollup status.
func (s *Service) Get(name string) (*InitiativeWithRollup, error) {
	init, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	rollup, err := s.ComputeRollup(init)
	if err != nil {
		return nil, fmt.Errorf("compute rollup for %q: %w", name, err)
	}
	return &InitiativeWithRollup{
		Initiative: *init,
		Rollup:     *rollup,
	}, nil
}

// List returns all initiatives with their rollup status.
func (s *Service) List() ([]InitiativeWithRollup, error) {
	initiatives, err := s.store.LoadAll()
	if err != nil {
		return nil, err
	}
	result := make([]InitiativeWithRollup, 0, len(initiatives))
	for i := range initiatives {
		rollup, rollupErr := s.ComputeRollup(&initiatives[i])
		if rollupErr != nil {
			// Use empty rollup on error rather than failing the list.
			rollup = &RollupStatus{}
		}
		result = append(result, InitiativeWithRollup{
			Initiative: initiatives[i],
			Rollup:     *rollup,
		})
	}
	return result, nil
}

// Update modifies an existing initiative.
func (s *Service) Update(name string, req UpdateRequest) (*Initiative, error) {
	init, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	if !req.HasChanges() {
		return nil, fmt.Errorf("at least one field must be provided")
	}

	oldStatus := init.Status

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, fmt.Errorf("title is required")
		}
		init.Title = title
	}
	if req.Description != nil {
		init.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if !ValidateStatus(status) {
			return nil, fmt.Errorf("invalid status %q: must be active, completed, or archived", *req.Status)
		}
		init.Status = status
	}
	if req.Items != nil {
		init.Items = *req.Items
	}
	init.Updated = time.Now().UTC().Format(time.RFC3339)

	if err := s.store.Save(init); err != nil {
		return nil, fmt.Errorf("save initiative: %w", err)
	}
	if s.eventLogger != nil && oldStatus != init.Status {
		s.eventLogger.EmitInitiativeStatusChanged(name, oldStatus, init.Status)
	}
	s.invalidateTopologyGraph()
	return init, nil
}

// Delete removes an initiative.
func (s *Service) Delete(name string) error {
	if !s.store.Exists(name) {
		return nil // idempotent
	}
	if err := s.store.Delete(name); err != nil {
		return err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeArchived(name)
	}
	s.invalidateTopologyGraph()
	return nil
}

// Replace writes a full initiative snapshot, used for internal rollback flows.
func (s *Service) Replace(init Initiative) error {
	if strings.TrimSpace(init.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(init.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if !ValidateStatus(strings.TrimSpace(init.Status)) {
		return fmt.Errorf("invalid status %q: must be active, completed, or archived", init.Status)
	}
	init.Name = strings.TrimSpace(init.Name)
	init.Title = strings.TrimSpace(init.Title)
	init.Description = strings.TrimSpace(init.Description)
	init.Status = strings.TrimSpace(init.Status)
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(&init); err != nil {
		return err
	}
	s.invalidateTopologyGraph()
	return nil
}

// ComputeRollup loads each referenced backlog item and aggregates status
// counts. Items that fail to load are counted as pending.
func (s *Service) ComputeRollup(init *Initiative) (*RollupStatus, error) {
	rollup := &RollupStatus{
		Total: len(init.Items),
	}
	for _, ref := range init.Items {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			rollup.Pending++
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			rollup.Pending++
			continue
		}
		item, loadErr := s.backlogLoader.LoadItem(kind, parts[1])
		if loadErr != nil {
			rollup.Pending++
			continue
		}
		switch item.Status {
		case backlog.StatusCompleted:
			rollup.Completed++
		case backlog.StatusFailed:
			rollup.Failed++
		case backlog.StatusInProgress, backlog.StatusQueued, backlog.StatusResearching:
			rollup.InProgress++
		default:
			rollup.Pending++
		}
	}
	return rollup, nil
}

// AddItems appends items to an initiative, deduplicating. Each item must be
// in "kind/name" format (e.g., "idea/my-feature").
func (s *Service) AddItems(name string, items []string) error {
	for _, item := range items {
		parts := strings.SplitN(item, "/", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("invalid item reference %q: expected format kind/name", item)
		}
	}
	init, err := s.store.Load(name)
	if err != nil {
		return err
	}
	existing := make(map[string]bool, len(init.Items))
	for _, item := range init.Items {
		existing[item] = true
	}
	var added []string
	for _, item := range items {
		if !existing[item] {
			init.Items = append(init.Items, item)
			existing[item] = true
			added = append(added, item)
		}
	}
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return err
	}
	if s.eventLogger != nil {
		for _, item := range added {
			s.eventLogger.EmitInitiativeItemAdded(name, item)
		}
	}
	s.invalidateTopologyGraph()
	return nil
}

// RemoveItems removes items from an initiative.
func (s *Service) RemoveItems(name string, items []string) error {
	init, err := s.store.Load(name)
	if err != nil {
		return err
	}
	remove := make(map[string]bool, len(items))
	for _, item := range items {
		remove[item] = true
	}
	filtered := make([]string, 0, len(init.Items))
	for _, item := range init.Items {
		if !remove[item] {
			filtered = append(filtered, item)
		}
	}
	init.Items = filtered
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return err
	}
	if s.eventLogger != nil {
		for _, item := range items {
			if remove[item] {
				s.eventLogger.EmitInitiativeItemRemoved(name, item)
			}
		}
	}
	s.invalidateTopologyGraph()
	return nil
}
