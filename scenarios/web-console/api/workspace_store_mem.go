package main

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemWorkspaceStore is an in-memory WorkspaceStore for unit tests.
type MemWorkspaceStore struct {
	mu     sync.RWMutex
	panes  map[string]*WorkspacePane // keyed by session_id
	groups map[string]*TabGroup      // keyed by group id
}

// NewMemWorkspaceStore creates an empty in-memory workspace store.
func NewMemWorkspaceStore() *MemWorkspaceStore {
	return &MemWorkspaceStore{
		panes:  make(map[string]*WorkspacePane),
		groups: make(map[string]*TabGroup),
	}
}

func (m *MemWorkspaceStore) GetLayout() (*WorkspaceLayout, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	panes := make([]*WorkspacePane, 0, len(m.panes))
	var activePaneID string
	for _, p := range m.panes {
		cp := *p
		panes = append(panes, &cp)
		if p.IsActive {
			activePaneID = p.SessionID
		}
	}
	sort.Slice(panes, func(i, j int) bool {
		return panes[i].SortOrder < panes[j].SortOrder
	})

	groups := make([]*TabGroup, 0, len(m.groups))
	for _, g := range m.groups {
		cp := *g
		groups = append(groups, &cp)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].SortOrder < groups[j].SortOrder
	})

	return &WorkspaceLayout{
		ActivePane: activePaneID,
		Panes:      panes,
		Groups:     groups,
	}, nil
}

func (m *MemWorkspaceStore) SavePaneOrder(activePaneID string, paneOrder []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.panes {
		p.IsActive = false
	}
	for i, sid := range paneOrder {
		if p, ok := m.panes[sid]; ok {
			p.SortOrder = i
			p.IsActive = sid == activePaneID
			p.UpdatedAt = formatTime(time.Now())
		}
	}
	return nil
}

func (m *MemWorkspaceStore) UpsertPane(pane *WorkspacePane) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := formatTime(time.Now())
	if existing, ok := m.panes[pane.SessionID]; ok {
		// Update mutable fields
		existing.Name = pane.Name
		existing.HeaderColor = pane.HeaderColor
		existing.ThemeID = pane.ThemeID
		existing.FontSize = pane.FontSize
		existing.GroupID = pane.GroupID
		existing.SortOrder = pane.SortOrder
		existing.UpdatedAt = now
	} else {
		cp := *pane
		if cp.CreatedAt == "" {
			cp.CreatedAt = now
		}
		cp.UpdatedAt = now
		if cp.Name == "" {
			cp.Name = defaultPaneName
		}
		if cp.HeaderColor == "" {
			cp.HeaderColor = defaultPaneHeaderColor
		}
		if cp.ThemeID == "" {
			cp.ThemeID = defaultPaneThemeID
		}
		if cp.FontSize == 0 {
			cp.FontSize = defaultPaneFontSize
		}
		m.panes[cp.SessionID] = &cp
	}
	return nil
}

func (m *MemWorkspaceStore) DeletePane(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.panes, sessionID)
	return nil
}

func (m *MemWorkspaceStore) CreateGroup(name, color string) (*TabGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := formatTime(time.Now())
	// Determine next sort_order
	maxOrder := -1
	for _, g := range m.groups {
		if g.SortOrder > maxOrder {
			maxOrder = g.SortOrder
		}
	}

	g := &TabGroup{
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
	m.groups[g.ID] = g

	cp := *g
	return &cp, nil
}

func (m *MemWorkspaceStore) UpdateGroup(id string, name *string, color *string, collapsed *bool) (*TabGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	g, ok := m.groups[id]
	if !ok {
		return nil, fmt.Errorf("group not found")
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
	g.UpdatedAt = formatTime(time.Now())

	cp := *g
	return &cp, nil
}

func (m *MemWorkspaceStore) DeleteGroup(id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groups[id]; !ok {
		return false, nil
	}
	delete(m.groups, id)

	// Clear group_id on panes that referenced this group
	for _, p := range m.panes {
		if p.GroupID == id {
			p.GroupID = ""
			p.UpdatedAt = formatTime(time.Now())
		}
	}
	return true, nil
}
