package workspace

import (
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
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{
		panes:  make(map[string]*Pane),
		groups: make(map[string]*Group),
	}
}

func (m *MemStore) GetLayout() (Layout, error) {
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
	}, nil
}

func (m *MemStore) SavePaneOrder(activePaneID string, paneOrder []string) error {
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

func (m *MemStore) UpsertPane(pane Pane) error {
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

func (m *MemStore) DeletePane(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.panes, sessionID)
	return nil
}

func (m *MemStore) ReassignPane(oldSessionID, newSessionID string) error {
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

func (m *MemStore) CreateGroup(name, color string) (Group, error) {
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

func (m *MemStore) UpdateGroup(id string, name *string, color *string, collapsed *bool) (Group, error) {
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

func (m *MemStore) DeleteGroup(id string) (bool, error) {
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
	return true, nil
}
