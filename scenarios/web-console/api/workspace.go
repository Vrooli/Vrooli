package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// workspace represents the persisted layout and tab configuration
type workspace struct {
	mu                    sync.RWMutex
	storagePath           string
	ActiveTabID           string    `json:"activeTabId"`
	Version               int64     `json:"version"`
	Tabs                  []tabMeta `json:"tabs"`
	KeyboardToolbarMode   string    `json:"keyboardToolbarMode,omitempty"` // "disabled", "floating", or "top"
	broadcasters          map[chan workspaceEvent]struct{}
	broadcasterMu         sync.RWMutex
}

// tabMeta represents a single tab's persistent state
type tabMeta struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	ColorID   string  `json:"colorId"`
	Order     int     `json:"order"`
	SessionID *string `json:"sessionId,omitempty"` // nil if no active session
}

// workspaceEvent represents a change to the workspace that should be broadcast
type workspaceEvent struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// newWorkspace creates or loads a workspace from disk
func newWorkspace(storagePath string) (*workspace, error) {
	ws := &workspace{
		storagePath:         storagePath,
		Version:             0,
		Tabs:                []tabMeta{},
		KeyboardToolbarMode: "floating", // Default to floating mode
		broadcasters:        make(map[chan workspaceEvent]struct{}),
	}

	if err := ws.load(); err != nil {
		// If file doesn't exist, start fresh
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("load workspace: %w", err)
		}
	}

	return ws, nil
}

// load reads the workspace from disk
func (ws *workspace) load() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	data, err := os.ReadFile(ws.storagePath)
	if err != nil {
		return err
	}

	var loaded struct {
		ActiveTabID         string    `json:"activeTabId"`
		Version             int64     `json:"version"`
		Tabs                []tabMeta `json:"tabs"`
		KeyboardToolbarMode string    `json:"keyboardToolbarMode"`
	}

	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parse workspace: %w", err)
	}

	ws.ActiveTabID = loaded.ActiveTabID
	ws.Version = loaded.Version
	ws.Tabs = loaded.Tabs
	if loaded.KeyboardToolbarMode != "" {
		ws.KeyboardToolbarMode = loaded.KeyboardToolbarMode
	}

	return nil
}

// save writes the workspace to disk
func (ws *workspace) save() error {
	ws.mu.RLock()
	snapshot := struct {
		ActiveTabID         string    `json:"activeTabId"`
		Version             int64     `json:"version"`
		Tabs                []tabMeta `json:"tabs"`
		KeyboardToolbarMode string    `json:"keyboardToolbarMode,omitempty"`
	}{
		ActiveTabID:         ws.ActiveTabID,
		Version:             ws.Version,
		Tabs:                append([]tabMeta{}, ws.Tabs...),
		KeyboardToolbarMode: ws.KeyboardToolbarMode,
	}
	ws.mu.RUnlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workspace: %w", err)
	}

	dir := filepath.Dir(ws.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create workspace dir: %w", err)
	}

	tmpPath := ws.storagePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write workspace: %w", err)
	}

	if err := os.Rename(tmpPath, ws.storagePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit workspace: %w", err)
	}

	return nil
}

// getState returns the current workspace state as JSON
func (ws *workspace) getState() ([]byte, error) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	snapshot := struct {
		ActiveTabID         string    `json:"activeTabId"`
		Version             int64     `json:"version"`
		Tabs                []tabMeta `json:"tabs"`
		KeyboardToolbarMode string    `json:"keyboardToolbarMode,omitempty"`
	}{
		ActiveTabID:         ws.ActiveTabID,
		Version:             ws.Version,
		Tabs:                append([]tabMeta{}, ws.Tabs...),
		KeyboardToolbarMode: ws.KeyboardToolbarMode,
	}

	return json.Marshal(snapshot)
}

// updateState replaces the workspace state
func (ws *workspace) updateState(activeTabID string, tabs []tabMeta) error {
	ws.mu.Lock()
	ws.ActiveTabID = activeTabID
	ws.Tabs = tabs
	ws.Version++
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "workspace-full-update",
		Timestamp: time.Now().UTC(),
		Payload:   mustJSON(struct {
			ActiveTabID string    `json:"activeTabId"`
			Version     int64     `json:"version"`
			Tabs        []tabMeta `json:"tabs"`
		}{
			ActiveTabID: activeTabID,
			Version:     ws.Version,
			Tabs:        tabs,
		}),
	})

	return nil
}

// addTab adds a new tab to the workspace
func (ws *workspace) addTab(tab tabMeta) error {
	ws.mu.Lock()
	tab.Order = len(ws.Tabs)
	ws.Tabs = append(ws.Tabs, tab)
	ws.Version++
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "tab-added",
		Timestamp: time.Now().UTC(),
		Payload:   mustJSON(tab),
	})

	return nil
}

// updateTab modifies an existing tab
func (ws *workspace) updateTab(tabID string, label, colorID string) error {
	ws.mu.Lock()
	found := false
	for i := range ws.Tabs {
		if ws.Tabs[i].ID == tabID {
			ws.Tabs[i].Label = label
			ws.Tabs[i].ColorID = colorID
			found = true
			break
		}
	}
	if !found {
		ws.mu.Unlock()
		return fmt.Errorf("tab not found: %s", tabID)
	}
	ws.Version++
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "tab-updated",
		Timestamp: time.Now().UTC(),
		Payload: mustJSON(map[string]string{
			"id":      tabID,
			"label":   label,
			"colorId": colorID,
		}),
	})

	return nil
}

// removeTab deletes a tab from the workspace
func (ws *workspace) removeTab(tabID string) error {
	ws.mu.Lock()
	newTabs := []tabMeta{}
	found := false
	for _, tab := range ws.Tabs {
		if tab.ID != tabID {
			newTabs = append(newTabs, tab)
		} else {
			found = true
		}
	}
	if !found {
		ws.mu.Unlock()
		return fmt.Errorf("tab not found: %s", tabID)
	}

	// Reorder remaining tabs
	for i := range newTabs {
		newTabs[i].Order = i
	}

	ws.Tabs = newTabs
	if ws.ActiveTabID == tabID {
		if len(ws.Tabs) > 0 {
			ws.ActiveTabID = ws.Tabs[0].ID
		} else {
			ws.ActiveTabID = ""
		}
	}
	ws.Version++
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "tab-removed",
		Timestamp: time.Now().UTC(),
		Payload:   mustJSON(map[string]string{"id": tabID}),
	})

	return nil
}

// setActiveTab changes the active tab
func (ws *workspace) setActiveTab(tabID string) error {
	ws.mu.Lock()
	found := false
	for _, tab := range ws.Tabs {
		if tab.ID == tabID {
			found = true
			break
		}
	}
	if !found && tabID != "" {
		ws.mu.Unlock()
		return fmt.Errorf("tab not found: %s", tabID)
	}

	ws.ActiveTabID = tabID
	ws.Version++
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "active-tab-changed",
		Timestamp: time.Now().UTC(),
		Payload:   mustJSON(map[string]string{"id": tabID}),
	})

	return nil
}

// attachSession links a session to a tab
func (ws *workspace) attachSession(tabID, sessionID string) error {
	ws.mu.Lock()
	found := false
	for i := range ws.Tabs {
		if ws.Tabs[i].ID == tabID {
			ws.Tabs[i].SessionID = &sessionID
			found = true
			break
		}
	}
	if !found {
		ws.mu.Unlock()
		return fmt.Errorf("tab not found: %s", tabID)
	}
	ws.Version++
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "session-attached",
		Timestamp: time.Now().UTC(),
		Payload: mustJSON(map[string]string{
			"tabId":     tabID,
			"sessionId": sessionID,
		}),
	})

	return nil
}

// detachSession removes a session link from a tab
func (ws *workspace) detachSession(tabID string) error {
	ws.mu.Lock()
	found := false
	for i := range ws.Tabs {
		if ws.Tabs[i].ID == tabID {
			ws.Tabs[i].SessionID = nil
			found = true
			break
		}
	}
	if !found {
		ws.mu.Unlock()
		return fmt.Errorf("tab not found: %s", tabID)
	}
	ws.Version++
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "session-detached",
		Timestamp: time.Now().UTC(),
		Payload:   mustJSON(map[string]string{"tabId": tabID}),
	})

	return nil
}

// subscribe adds a broadcaster channel for workspace events
func (ws *workspace) subscribe() chan workspaceEvent {
	ws.broadcasterMu.Lock()
	defer ws.broadcasterMu.Unlock()

	ch := make(chan workspaceEvent, 32)
	ws.broadcasters[ch] = struct{}{}
	return ch
}

// unsubscribe removes a broadcaster channel
func (ws *workspace) unsubscribe(ch chan workspaceEvent) {
	ws.broadcasterMu.Lock()
	defer ws.broadcasterMu.Unlock()

	delete(ws.broadcasters, ch)
	close(ch)
}

// setKeyboardToolbarMode updates the keyboard toolbar mode setting
func (ws *workspace) setKeyboardToolbarMode(mode string) error {
	// Validate mode
	validModes := map[string]bool{
		"disabled": true,
		"floating": true,
		"top":      true,
	}
	if !validModes[mode] {
		return fmt.Errorf("invalid keyboard toolbar mode: %s (must be disabled, floating, or top)", mode)
	}

	ws.mu.Lock()
	ws.KeyboardToolbarMode = mode
	ws.Version++
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "keyboard-toolbar-mode-changed",
		Timestamp: time.Now().UTC(),
		Payload:   mustJSON(map[string]string{"mode": mode}),
	})

	return nil
}

// broadcast sends an event to all subscribers
func (ws *workspace) broadcast(event workspaceEvent) {
	ws.broadcasterMu.RLock()
	defer ws.broadcasterMu.RUnlock()

	for ch := range ws.broadcasters {
		select {
		case ch <- event:
		default:
			// Drop if channel is full (slow consumer)
		}
	}
}
