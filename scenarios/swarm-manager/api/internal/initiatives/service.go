package initiatives

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/pathutil"
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
	mode := NormalizeMode(req.Mode)
	if !ValidateMode(mode) {
		return nil, validationErr("invalid operating mode %q: must be one of %s", req.Mode, OperatingModeList())
	}

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
	}
	if err := s.store.Save(init); err != nil {
		return nil, fmt.Errorf("save initiative: %w", err)
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeCreated(name)
		defaultMode := NormalizeMode("")
		if mode != defaultMode {
			s.eventLogger.EmitInitiativeModeChanged(name, defaultMode, mode)
		}
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
		status := strings.TrimSpace(*req.Status)
		if !ValidateStatus(status) {
			return nil, validationErr("invalid status %q: must be %s", *req.Status, UserSettableInitiativeStatusList())
		}
		// Same-status no-ops are allowed so callers (e.g. the backlog
		// batch adapter's full-field replace) don't fail when an
		// existing initiative is already in a review/terminal status.
		if status != init.Status {
			if !IsUserSettableInitiativeStatus(status) {
				return nil, validationErr("status %q is owned by the review pipeline; use the initiatives review-decide endpoint so the decision is audited", status)
			}
			if IsReviewInitiativeStatus(init.Status) {
				return nil, validationErr("initiative is in status %q; use the review-decide endpoint to change status", init.Status)
			}
			if IsTerminalInitiativeStatus(init.Status) {
				return nil, validationErr("initiative is in terminal status %q; cannot revert via PATCH", init.Status)
			}
		}
		init.Status = status
	}
	if req.Mode != nil {
		mode := NormalizeMode(*req.Mode)
		if !ValidateMode(mode) {
			return nil, validationErr("invalid operating mode %q: must be one of %s", *req.Mode, OperatingModeList())
		}
		init.Mode = mode
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
	if s.eventLogger != nil && oldMode != init.Mode {
		s.eventLogger.EmitInitiativeModeChanged(name, oldMode, init.Mode)
	}
	s.invalidateTopologyGraph()
	return init, nil
}

// Delete removes an initiative and cascades referential integrity:
//   - Every member item has its `initiative` field cleared (orphaned, not deleted).
//   - Every other initiative that referenced this one via `depends_on` has the
//     reference removed.
//
// The cascade is best-effort atomic: side effects are captured up front so a
// failure mid-cascade can be rolled back. If the final store.Delete fails,
// prior cascades are reverted.
func (s *Service) Delete(name string) error {
	if !s.store.Exists(name) {
		return nil // idempotent
	}
	init, err := s.store.Load(name)
	if err != nil {
		return fmt.Errorf("load initiative before delete: %w", err)
	}

	type itemRef struct {
		kind      backlog.BacklogKind
		localName string
		ref       string
	}
	refs := make([]itemRef, 0, len(init.Items))
	for _, raw := range init.Items {
		parts := strings.SplitN(raw, "/", 2)
		if len(parts) != 2 {
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			continue
		}
		refs = append(refs, itemRef{kind: kind, localName: parts[1], ref: raw})
	}

	cleared := make([]itemRef, 0, len(refs))
	if s.backlogLoader != nil {
		for _, r := range refs {
			_, changed, err := s.backlogLoader.ClearItemInitiative(r.kind, r.localName, name)
			if err != nil {
				for _, done := range cleared {
					if _, setErr := s.backlogLoader.SetItemInitiative(done.kind, done.localName, name); setErr != nil {
						// Log via slog through service's nil-safe logger; keep rollback best-effort.
						_ = setErr
					}
				}
				return fmt.Errorf("cascade: clear initiative on %s: %w", r.ref, err)
			}
			if changed {
				cleared = append(cleared, r)
			}
		}
	}

	all, err := s.store.LoadAll()
	if err != nil {
		for _, done := range cleared {
			if _, setErr := s.backlogLoader.SetItemInitiative(done.kind, done.localName, name); setErr != nil {
				_ = setErr
			}
		}
		return fmt.Errorf("cascade: load initiatives for depends_on scrub: %w", err)
	}
	type depScrub struct {
		initName string
		oldDeps  []string
	}
	scrubbed := make([]depScrub, 0)
	for i := range all {
		other := &all[i]
		if other.Name == name {
			continue
		}
		if !stringSliceContains(other.DependsOn, name) {
			continue
		}
		oldDeps := append([]string(nil), other.DependsOn...)
		filtered := make([]string, 0, len(other.DependsOn))
		for _, d := range other.DependsOn {
			if d != name {
				filtered = append(filtered, d)
			}
		}
		other.DependsOn = filtered
		other.Updated = time.Now().UTC().Format(time.RFC3339)
		if saveErr := s.store.Save(other); saveErr != nil {
			for _, sc := range scrubbed {
				if rolled, rErr := s.store.Load(sc.initName); rErr == nil {
					rolled.DependsOn = sc.oldDeps
					_ = s.store.Save(rolled)
				}
			}
			for _, done := range cleared {
				if _, setErr := s.backlogLoader.SetItemInitiative(done.kind, done.localName, name); setErr != nil {
					_ = setErr
				}
			}
			return fmt.Errorf("cascade: scrub depends_on from %q: %w", other.Name, saveErr)
		}
		scrubbed = append(scrubbed, depScrub{initName: other.Name, oldDeps: oldDeps})
	}

	if err := s.store.Delete(name); err != nil {
		for _, sc := range scrubbed {
			if rolled, rErr := s.store.Load(sc.initName); rErr == nil {
				rolled.DependsOn = sc.oldDeps
				_ = s.store.Save(rolled)
			}
		}
		for _, done := range cleared {
			if _, setErr := s.backlogLoader.SetItemInitiative(done.kind, done.localName, name); setErr != nil {
				_ = setErr
			}
		}
		return err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeArchived(name, init.Status, "")
	}
	s.invalidateTopologyGraph()
	return nil
}

func stringSliceContains(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
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
		return fmt.Errorf("invalid status %q: must be active or completed", init.Status)
	}
	if !ValidatePriority(init.Priority) {
		return fmt.Errorf("invalid priority %d: must be 0 (unset) or 1-10", init.Priority)
	}
	init.Name = strings.TrimSpace(init.Name)
	init.Title = strings.TrimSpace(init.Title)
	init.Description = strings.TrimSpace(init.Description)
	init.Status = strings.TrimSpace(init.Status)
	init.Mode = NormalizeMode(init.Mode)
	if !ValidateMode(init.Mode) {
		return fmt.Errorf("invalid operating mode %q: must be one of %s", init.Mode, OperatingModeList())
	}
	init.DependsOn = normalizeDependsOn(init.DependsOn)
	init.AcceptanceCriteria = normalizeStringList(init.AcceptanceCriteria)
	if err := s.validateDependsOn(init.Name, init.DependsOn); err != nil {
		return err
	}
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
	rollup, _ := s.aggregateInitiativeData(init)
	return rollup, nil
}

// aggregateInitiativeData loads each referenced backlog item once and returns
// both the rollup and the deduped list of scenarios targeted by the item's
// acceptance_allow globs. Items that fail to load are counted as pending and
// contribute no scenarios.
func (s *Service) aggregateInitiativeData(init *Initiative) (*RollupStatus, []string) {
	rollup := &RollupStatus{
		Total: len(init.Items),
	}
	seen := make(map[string]struct{})
	var scenarios []string
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
		if backlog.IsArchived(item) {
			rollup.Archived++
			if item.Status == backlog.StatusCompleted {
				rollup.Completed++
			}
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
		for _, name := range pathutil.ScenariosFromGlobs(item.AcceptanceAllow) {
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				scenarios = append(scenarios, name)
			}
		}
	}
	return rollup, scenarios
}

// AddItems appends items to an initiative, deduplicating. Each item must be
// in "kind/name" format (e.g., "idea/my-feature"). Maintains symmetry with
// the item side: items already attached to a different initiative are
// rejected; orphan items (with an empty initiative field) have their
// initiative field set to this name so the two references stay in sync.
func (s *Service) AddItems(name string, items []string) error {
	type parsedItem struct {
		kind      backlog.BacklogKind
		localName string
		ref       string
	}
	parsed := make([]parsedItem, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, "/", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("invalid item reference %q: expected format kind/name", item)
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			return fmt.Errorf("invalid item reference %q: %w", item, err)
		}
		parsed = append(parsed, parsedItem{kind: kind, localName: parts[1], ref: item})
	}

	if s.backlogLoader != nil {
		for _, p := range parsed {
			item, err := s.backlogLoader.LoadItem(p.kind, p.localName)
			if err != nil {
				continue
			}
			current := strings.TrimSpace(item.Initiative)
			if current != "" && current != name {
				return fmt.Errorf("item %q already belongs to initiative %q; use PATCH on the item to move it", p.ref, current)
			}
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
	added := make([]parsedItem, 0, len(parsed))
	for _, p := range parsed {
		if existing[p.ref] {
			continue
		}
		init.Items = append(init.Items, p.ref)
		existing[p.ref] = true
		added = append(added, p)
	}
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return err
	}

	if s.backlogLoader != nil {
		for _, p := range added {
			if _, err := s.backlogLoader.SetItemInitiative(p.kind, p.localName, name); err != nil {
				// Item may not exist yet (e.g., batch create writes items
				// separately); not-found is not an error here.
				continue
			}
		}
	}

	if s.eventLogger != nil {
		for _, p := range added {
			s.eventLogger.EmitInitiativeItemAdded(name, p.ref)
		}
	}
	s.invalidateTopologyGraph()
	return nil
}

// RemoveItems removes items from an initiative and clears the item's
// initiative field if it currently equals this initiative, maintaining
// two-way referential integrity.
func (s *Service) RemoveItems(name string, items []string) error {
	type parsedItem struct {
		kind      backlog.BacklogKind
		localName string
		ref       string
	}
	parsed := make([]parsedItem, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, "/", 2)
		if len(parts) != 2 {
			parsed = append(parsed, parsedItem{ref: item})
			continue
		}
		kind, err := backlog.ParseBacklogKind(parts[0])
		if err != nil {
			parsed = append(parsed, parsedItem{ref: item})
			continue
		}
		parsed = append(parsed, parsedItem{kind: kind, localName: parts[1], ref: item})
	}

	init, err := s.store.Load(name)
	if err != nil {
		return err
	}
	remove := make(map[string]bool, len(parsed))
	for _, p := range parsed {
		remove[p.ref] = true
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

	if s.backlogLoader != nil {
		for _, p := range parsed {
			if p.localName == "" || p.kind == "" {
				continue
			}
			if _, _, err := s.backlogLoader.ClearItemInitiative(p.kind, p.localName, name); err != nil {
				continue
			}
		}
	}

	if s.eventLogger != nil {
		for _, p := range parsed {
			if remove[p.ref] {
				s.eventLogger.EmitInitiativeItemRemoved(name, p.ref)
			}
		}
	}
	s.invalidateTopologyGraph()
	return nil
}

// RememberItem appends a single ref to the initiative's items[] list if not
// already present. This is a one-way helper: it does not modify the item's
// initiative field. Used by single-item create/patch cascade, which writes
// the item's initiative field itself via SaveItem.
func (s *Service) RememberItem(initiativeName, ref string) error {
	init, err := s.store.Load(initiativeName)
	if err != nil {
		return err
	}
	for _, existing := range init.Items {
		if existing == ref {
			return nil
		}
	}
	init.Items = append(init.Items, ref)
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeItemAdded(initiativeName, ref)
	}
	s.invalidateTopologyGraph()
	return nil
}

// ForgetItem removes a single ref from the initiative's items[] list. This
// is a one-way helper: it does not modify the item's initiative field. Used
// by single-item delete/patch cascade, where the item file is already gone
// or its initiative field is written separately.
func (s *Service) ForgetItem(initiativeName, ref string) error {
	if !s.store.Exists(initiativeName) {
		return nil
	}
	init, err := s.store.Load(initiativeName)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(init.Items))
	removed := false
	for _, existing := range init.Items {
		if existing == ref {
			removed = true
			continue
		}
		filtered = append(filtered, existing)
	}
	if !removed {
		return nil
	}
	init.Items = filtered
	init.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := s.store.Save(init); err != nil {
		return err
	}
	if s.eventLogger != nil {
		s.eventLogger.EmitInitiativeItemRemoved(initiativeName, ref)
	}
	s.invalidateTopologyGraph()
	return nil
}
