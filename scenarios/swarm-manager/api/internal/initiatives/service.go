package initiatives

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/dispatch"
)

// ErrValidation wraps user-correctable validation failures from Create/Update
// so the HTTP handler can return 400 instead of 500. Store/cascade failures
// do not wrap this sentinel and remain 500s.
var ErrValidation = errors.New("initiative validation error")

func validationErr(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrValidation}, args...)...)
}

// BacklogLoader loads individual backlog items for rollup computation and
// performs cascade writes on an item's initiative field when initiative
// membership changes through the AddItems/RemoveItems paths or initiative
// deletion.
type BacklogLoader interface {
	LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error)
	// SetItemInitiative sets the item's initiative field, returning the
	// previous value. Errors if the item does not exist.
	SetItemInitiative(kind backlog.BacklogKind, name, initiative string) (string, error)
	// ClearItemInitiative clears the item's initiative field only if it
	// currently equals expected. Returns (prevValue, changed, error). If the
	// item does not exist or the field does not match, changed=false and err=nil.
	ClearItemInitiative(kind backlog.BacklogKind, name, expected string) (string, bool, error)
}

// EventLogger records initiative state-change events for analytics.
type EventLogger interface {
	EmitInitiativeCreated(name string)
	EmitInitiativeItemAdded(name, item string)
	EmitInitiativeItemRemoved(name, item string)
	EmitInitiativeStatusChanged(name, from, to string)
	EmitInitiativeModeChanged(name, from, to string)
	EmitInitiativeArchived(name, previousStatus, archivedAt string)
	EmitInitiativeUnarchived(name, archivedAt string)
	EmitInitiativeViewed(name string)
}

// Service provides business logic for initiative operations.
type Service struct {
	store           *Store
	backlogLoader   BacklogLoader
	eventDispatcher dispatch.Invalidator
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
func (s *Service) SetEventDispatcher(d dispatch.Invalidator) {
	s.eventDispatcher = d
}

// SetEventLogger injects an optional event logger for analytics tracking.
func (s *Service) SetEventLogger(l EventLogger) {
	s.eventLogger = l
}

// SetAIIndexer wires an optional AI search indexer that receives fire-and-forget
// notifications from the underlying Store after every Save/Delete.
func (s *Service) SetAIIndexer(indexer AIIndexer) {
	s.store.SetAIIndexer(indexer)
}

// RecordView emits a view event for analytics.
func (s *Service) RecordView(name string) {
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeViewed(name)
	}
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
		status = InitiativeStatusActive
	}
	if !ValidateStatus(status) {
		return nil, validationErr("invalid status %q: must be %s", req.Status, UserSettableInitiativeStatusList())
	}
	if !IsUserSettableInitiativeStatus(status) {
		return nil, validationErr("status %q is owned by the review pipeline; initiatives are created as %q and transition via the review-decide endpoint", status, InitiativeStatusActive)
	}
	mode := NormalizeMode("")

	if !ValidatePriority(req.Priority) {
		return nil, fmt.Errorf("invalid priority %d: must be 0 (unset) or 1-10", req.Priority)
	}
	dependsOn := normalizeDependsOn(req.DependsOn)
	if err := s.validateDependsOn(name, dependsOn); err != nil {
		return nil, err
	}

	items := req.Items
	if items == nil {
		items = []string{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	init := &Initiative{
		Name:               name,
		Title:              title,
		Description:        strings.TrimSpace(req.Description),
		Status:             status,
		Mode:               mode,
		Priority:           req.Priority,
		DependsOn:          dependsOn,
		Items:              items,
		AcceptanceCriteria: normalizeStringList(req.AcceptanceCriteria),
		Created:            now,
		Updated:            now,
		CreatedBy:          req.CreatedBy,
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

// GetContext loads an initiative together with its immediate neighborhood:
// member items (compact view), direct upstream initiatives (targets of its
// depends_on), and direct downstream initiatives (those that depend_on it).
// Member items that reference missing files are skipped silently.
func (s *Service) GetContext(name string) (*InitiativeContext, error) {
	init, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	rollup, scenarios := s.aggregateInitiativeData(init)
	if rollup == nil {
		rollup = &RollupStatus{}
	}

	items := make([]ContextItem, 0, len(init.Items))
	if s.backlogLoader != nil {
		for _, raw := range init.Items {
			parts := strings.SplitN(raw, "/", 2)
			if len(parts) != 2 {
				continue
			}
			kind, err := backlog.ParseBacklogKind(parts[0])
			if err != nil {
				continue
			}
			item, err := s.backlogLoader.LoadItem(kind, parts[1])
			if err != nil {
				continue
			}
			items = append(items, ContextItem{
				Kind:       string(item.Kind),
				Name:       item.Name,
				Title:      item.Title,
				Status:     string(item.Status),
				Priority:   item.Priority,
				DependsOn:  append([]string(nil), item.DependsOn...),
				Initiative: item.Initiative,
				ArchivedAt: item.ArchivedAt,
			})
		}
	}

	all, err := s.store.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("load initiatives for context: %w", err)
	}
	byName := make(map[string]Initiative, len(all))
	for _, other := range all {
		byName[other.Name] = other
	}

	upstream := make([]Initiative, 0, len(init.DependsOn))
	for _, dep := range init.DependsOn {
		if other, ok := byName[dep]; ok {
			upstream = append(upstream, other)
		}
	}

	downstream := make([]Initiative, 0)
	for _, other := range all {
		if other.Name == name {
			continue
		}
		if stringSliceContains(other.DependsOn, name) {
			downstream = append(downstream, other)
		}
	}

	return &InitiativeContext{
		Initiative:            *init,
		Rollup:                *rollup,
		Items:                 items,
		UpstreamInitiatives:   upstream,
		DownstreamInitiatives: downstream,
		TargetScenarios:       scenarios,
	}, nil
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

// List returns all initiatives with their rollup status and the deduped
// scenarios targeted by each initiative's member items.
func (s *Service) List() ([]InitiativeWithRollup, error) {
	initiatives, err := s.store.LoadAll()
	if err != nil {
		return nil, err
	}
	result := make([]InitiativeWithRollup, 0, len(initiatives))
	for i := range initiatives {
		rollup, scenarios := s.aggregateInitiativeData(&initiatives[i])
		if rollup == nil {
			rollup = &RollupStatus{}
		}
		result = append(result, InitiativeWithRollup{
			Initiative:      initiatives[i],
			Rollup:          *rollup,
			TargetScenarios: scenarios,
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
	oldMode := NormalizeMode(init.Mode)
	init.Mode = oldMode

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
		status, err := validateStatusTransition(init.Status, *req.Status)
		if err != nil {
			return nil, err
		}
		init.Status = status
	}
	if req.Priority != nil {
		if !ValidatePriority(*req.Priority) {
			return nil, fmt.Errorf("invalid priority %d: must be 0 (unset) or 1-10", *req.Priority)
		}
		init.Priority = *req.Priority
	}
	if req.DependsOn != nil {
		deps := normalizeDependsOn(*req.DependsOn)
		if err := s.validateDependsOn(init.Name, deps); err != nil {
			return nil, err
		}
		init.DependsOn = deps
	}
	if req.Items != nil {
		init.Items = *req.Items
	}
	if req.AcceptanceCriteria != nil {
		init.AcceptanceCriteria = normalizeStringList(*req.AcceptanceCriteria)
	}
	if req.Note != nil {
		init.Note = strings.TrimSpace(*req.Note)
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

// validateStatusTransition validates a requested status change against the
// current status, returning the trimmed target status. Same-status no-ops are
// allowed so callers (e.g. the backlog batch adapter's full-field replace)
// don't fail when an existing initiative is already in a review/terminal status.
func validateStatusTransition(currentStatus, requestedStatus string) (string, error) {
	status := strings.TrimSpace(requestedStatus)
	if !ValidateStatus(status) {
		return "", validationErr("invalid status %q: must be %s", requestedStatus, UserSettableInitiativeStatusList())
	}
	if status != currentStatus {
		if !IsUserSettableInitiativeStatus(status) {
			return "", validationErr("status %q is owned by the review pipeline; use the initiatives review-decide endpoint so the decision is audited", status)
		}
		if IsReviewInitiativeStatus(currentStatus) {
			return "", validationErr("initiative is in status %q; use the review-decide endpoint to change status", currentStatus)
		}
		if IsTerminalInitiativeStatus(currentStatus) {
			return "", validationErr("initiative is in terminal status %q; cannot revert via PATCH", currentStatus)
		}
	}
	return status, nil
}

// SetModeLifecycle is the single initiative-mode mutation path. It is intended
// for the operating-mode lifecycle service only; public initiative create/update
// APIs always create item-level initiatives and reject mode mutation.
func (s *Service) SetModeLifecycle(name, mode string) (*Initiative, error) {
	init, err := s.store.Load(name)
	if err != nil {
		return nil, err
	}
	if init.ArchivedAt != nil {
		return nil, validationErr("archived initiative %q cannot change operating mode", name)
	}
	oldMode := NormalizeMode(init.Mode)
	nextMode := NormalizeMode(mode)
	if !ValidateMode(nextMode) {
		return nil, validationErr("invalid operating mode %q: must be one of %s", mode, OperatingModeList())
	}
	init.Mode = nextMode
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return nil, fmt.Errorf("save initiative: %w", err)
	}
	if s.eventLogger != nil && oldMode != init.Mode {
		s.eventLogger.EmitInitiativeModeChanged(name, oldMode, init.Mode)
	}
	s.invalidateTopologyGraph()
	return init, nil
}
