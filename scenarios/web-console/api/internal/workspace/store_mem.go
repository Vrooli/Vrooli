package workspace

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemStore is an in-memory Store implementation for unit tests.
type MemStore struct {
	mu     sync.RWMutex
	panes  map[string]*Pane
	groups map[string]*Group
	roles  map[string]*Role
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{
		panes:  make(map[string]*Pane),
		groups: make(map[string]*Group),
		roles:  make(map[string]*Role),
	}
}

func (m *MemStore) GetLayout(_ context.Context) (Layout, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	panes := make([]Pane, 0, len(m.panes))
	var activePaneID string
	for _, p := range m.panes {
		panes = append(panes, *p)
		if p.IsActive {
			activePaneID = p.SessionID
		}
	}
	sort.Slice(panes, func(i, j int) bool {
		return panes[i].SortOrder < panes[j].SortOrder
	})

	groups := make([]Group, 0, len(m.groups))
	for _, g := range m.groups {
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].SortOrder < groups[j].SortOrder
	})

	return Layout{
		ActivePane: activePaneID,
		Panes:      panes,
		Groups:     groups,
		Roles:      m.rolesLocked(""),
	}, nil
}

func (m *MemStore) SavePaneOrder(_ context.Context, activePaneID string, paneOrder []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.panes {
		p.IsActive = false
	}
	now := FormatTime(time.Now())
	for i, sid := range paneOrder {
		if p, ok := m.panes[sid]; ok {
			p.SortOrder = i
			p.IsActive = sid == activePaneID
			p.UpdatedAt = now
		}
	}
	return nil
}

func (m *MemStore) UpsertPane(_ context.Context, pane Pane) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := FormatTime(time.Now())
	if existing, ok := m.panes[pane.SessionID]; ok {
		existing.Name = pane.Name
		existing.HeaderColor = pane.HeaderColor
		existing.ThemeID = pane.ThemeID
		existing.FontSize = pane.FontSize
		existing.GroupID = pane.GroupID
		existing.SortOrder = pane.SortOrder
		existing.SupportsMessagesView = pane.SupportsMessagesView
		existing.ManuallyUnread = pane.ManuallyUnread
		existing.UpdatedAt = now
		return nil
	}

	cp := pane
	if cp.CreatedAt == "" {
		cp.CreatedAt = now
	}
	cp.UpdatedAt = now
	if cp.Name == "" {
		cp.Name = DefaultPaneName
	}
	if cp.HeaderColor == "" {
		cp.HeaderColor = DefaultPaneHeaderColor
	}
	if cp.ThemeID == "" {
		cp.ThemeID = DefaultPaneThemeID
	}
	if cp.FontSize == 0 {
		cp.FontSize = DefaultPaneFontSize
	}
	m.panes[cp.SessionID] = &cp
	return nil
}

func (m *MemStore) DeletePane(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.panes, sessionID)
	return nil
}

func (m *MemStore) ReassignPane(_ context.Context, oldSessionID, newSessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	old, ok := m.panes[oldSessionID]
	if !ok {
		return nil
	}
	delete(m.panes, newSessionID)
	copy := *old
	copy.SessionID = newSessionID
	copy.UpdatedAt = FormatTime(time.Now())
	m.panes[newSessionID] = &copy
	delete(m.panes, oldSessionID)
	return nil
}

func (m *MemStore) CreateGroup(_ context.Context, name, color string) (Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := FormatTime(time.Now())
	maxOrder := -1
	for _, g := range m.groups {
		if g.SortOrder > maxOrder {
			maxOrder = g.SortOrder
		}
	}

	g := Group{
		ID:        uuid.New().String(),
		Name:      name,
		Color:     color,
		SortOrder: maxOrder + 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if g.Name == "" {
		g.Name = "Group"
	}
	if g.Color == "" {
		g.Color = "#3b82f6"
	}
	stored := g
	m.groups[g.ID] = &stored
	return g, nil
}

func (m *MemStore) UpdateGroup(_ context.Context, id string, name *string, color *string, collapsed *bool) (Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	g, ok := m.groups[id]
	if !ok {
		return Group{}, ErrGroupNotFound
	}

	if name != nil {
		g.Name = *name
	}
	if color != nil {
		g.Color = *color
	}
	if collapsed != nil {
		g.IsCollapsed = *collapsed
	}
	g.UpdatedAt = FormatTime(time.Now())

	return *g, nil
}

func (m *MemStore) DeleteGroup(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groups[id]; !ok {
		return false, nil
	}
	delete(m.groups, id)

	now := FormatTime(time.Now())
	for _, p := range m.panes {
		if p.GroupID == id {
			p.GroupID = ""
			p.UpdatedAt = now
		}
	}
	// Roles go with the group. The SQL store gets this from ON DELETE
	// CASCADE; the memory store must produce the same observable result.
	for roleID, r := range m.roles {
		if r.GroupID == id {
			delete(m.roles, roleID)
		}
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

func (m *MemStore) ListRoles(_ context.Context, groupID string) ([]Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rolesLocked(groupID), nil
}

// rolesLocked returns a sorted copy of the roles matching groupID (all roles
// when it is empty). The caller must hold at least a read lock.
func (m *MemStore) rolesLocked(groupID string) []Role {
	out := make([]Role, 0, len(m.roles))
	for _, r := range m.roles {
		if groupID != "" && r.GroupID != groupID {
			continue
		}
		out = append(out, *r)
	}
	sortRoles(out)
	return out
}

func (m *MemStore) CreateRole(_ context.Context, req CreateRoleRequest) (Role, error) {
	if req.GroupID == "" {
		return Role{}, ErrInvalidRole
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	// A running session backs at most one role. The SQL store enforces this
	// with a partial unique index; the memory store must agree or a test that
	// passes here would fail in production.
	if req.SessionID != "" {
		for _, r := range m.roles {
			if r.SessionID == req.SessionID {
				return Role{}, ErrInvalidRole
			}
		}
	}

	sortOrder := req.SortOrder
	if !req.HasSortOrder {
		sortOrder = 0
		for _, r := range m.roles {
			if r.GroupID == req.GroupID && r.SortOrder >= sortOrder {
				sortOrder = r.SortOrder + 1
			}
		}
	}

	now := FormatTime(time.Now())
	role := Role{
		ID:             uuid.New().String(),
		GroupID:        req.GroupID,
		Label:          req.Label,
		Command:        req.Command,
		WorkingDir:     req.WorkingDir,
		IncomingPrompt: req.IncomingPrompt,
		Backend:        req.Backend,
		TargetID:       req.TargetID,
		SessionID:      req.SessionID,
		SortOrder:      sortOrder,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if role.Label == "" {
		role.Label = DefaultRoleLabel
	}
	stored := role
	m.roles[role.ID] = &stored
	return role, nil
}

func (m *MemStore) UpdateRole(_ context.Context, req UpdateRoleRequest) (Role, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, ok := m.roles[req.ID]
	if !ok {
		return Role{}, ErrRoleNotFound
	}
	if req.HasSessionID && req.SessionID != "" {
		for id, other := range m.roles {
			if id != req.ID && other.SessionID == req.SessionID {
				return Role{}, ErrInvalidRole
			}
		}
	}

	if req.HasLabel {
		r.Label = req.Label
	}
	if req.HasCommand {
		r.Command = req.Command
	}
	if req.HasWorkingDir {
		r.WorkingDir = req.WorkingDir
	}
	if req.HasIncomingPrompt {
		r.IncomingPrompt = req.IncomingPrompt
	}
	if req.HasSessionID {
		r.SessionID = req.SessionID
	}
	if req.HasSortOrder {
		r.SortOrder = req.SortOrder
	}
	if req.HasBackend {
		r.Backend = req.Backend
	}
	if req.HasTargetID {
		r.TargetID = req.TargetID
	}
	if req.HasGroupID {
		if req.GroupID == "" {
			return Role{}, ErrInvalidRole
		}
		r.GroupID = req.GroupID
	}
	r.UpdatedAt = FormatTime(time.Now())
	return *r, nil
}

func (m *MemStore) DeleteRole(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.roles[id]; !ok {
		return false, nil
	}
	delete(m.roles, id)
	return true, nil
}

func (m *MemStore) ReassignRoleSession(_ context.Context, oldSessionID, newSessionID string) error {
	if oldSessionID == "" || newSessionID == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	// Drop any role already claiming the replacement id before re-pointing, so
	// the one-session-one-role invariant survives the move.
	for _, r := range m.roles {
		if r.SessionID == newSessionID {
			r.SessionID = ""
		}
	}
	now := FormatTime(time.Now())
	for _, r := range m.roles {
		if r.SessionID == oldSessionID {
			r.SessionID = newSessionID
			r.UpdatedAt = now
		}
	}
	return nil
}

// sortRoles orders roles by group, then position, then id so the sequence is
// total and stable — two roles sharing a sort_order must not swap between
// reads, or the sidebar reorders itself on every refresh.
func sortRoles(roles []Role) {
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].GroupID != roles[j].GroupID {
			return roles[i].GroupID < roles[j].GroupID
		}
		if roles[i].SortOrder != roles[j].SortOrder {
			return roles[i].SortOrder < roles[j].SortOrder
		}
		return roles[i].ID < roles[j].ID
	})
}
