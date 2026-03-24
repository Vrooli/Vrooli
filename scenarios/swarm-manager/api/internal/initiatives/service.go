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

// Service provides business logic for initiative operations.
type Service struct {
	store         *Store
	backlogLoader BacklogLoader
}

// NewService creates a Service with the given store and backlog loader.
func NewService(store *Store, backlogLoader BacklogLoader) *Service {
	return &Service{
		store:         store,
		backlogLoader: backlogLoader,
	}
}

// Create creates a new initiative.
func (s *Service) Create(req CreateRequest) (*Initiative, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if s.store.Exists(name) {
		return nil, fmt.Errorf("initiative %q already exists", name)
	}

	items := req.Items
	if items == nil {
		items = []string{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	init := &Initiative{
		Name:        name,
		Title:       req.Title,
		Description: strings.TrimSpace(req.Description),
		Status:      "active",
		Items:       items,
		Created:     now,
		Updated:     now,
	}
	if err := s.store.Save(init); err != nil {
		return nil, fmt.Errorf("save initiative: %w", err)
	}
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
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if !ValidateStatus(req.Status) {
		return nil, fmt.Errorf("invalid status %q: must be active, completed, or archived", req.Status)
	}

	init.Title = req.Title
	init.Description = strings.TrimSpace(req.Description)
	init.Status = req.Status
	if req.Items != nil {
		init.Items = req.Items
	}
	init.Updated = time.Now().UTC().Format(time.RFC3339)

	if err := s.store.Save(init); err != nil {
		return nil, fmt.Errorf("save initiative: %w", err)
	}
	return init, nil
}

// Delete removes an initiative.
func (s *Service) Delete(name string) error {
	if !s.store.Exists(name) {
		return nil // idempotent
	}
	return s.store.Delete(name)
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
	for _, item := range items {
		if !existing[item] {
			init.Items = append(init.Items, item)
			existing[item] = true
		}
	}
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	return s.store.Save(init)
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
	return s.store.Save(init)
}
