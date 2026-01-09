package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type workspaceValidationError struct {
	message string
}

func (e workspaceValidationError) Error() string {
	return e.message
}

func newValidationError(format string, args ...any) error {
	return workspaceValidationError{message: fmt.Sprintf(format, args...)}
}

var errTabNotFound = errors.New("tab not found")

// workspace represents the persisted layout and tab configuration
type workspace struct {
	mu                  sync.RWMutex
	storagePath         string
	ActiveTabID         string    `json:"activeTabId"`
	Version             int64     `json:"version"`
	Tabs                []tabMeta `json:"tabs"`
	KeyboardToolbarMode string    `json:"keyboardToolbarMode,omitempty"` // "disabled", "floating", or "top"
	IdleTimeoutSeconds  int64     `json:"idleTimeoutSeconds,omitempty"`
	broadcasters        map[chan workspaceEvent]struct{}
	broadcasterMu       sync.RWMutex
	tabIndex            map[string]int
	sessionIndex        map[string]int
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
		tabIndex:            make(map[string]int),
		sessionIndex:        make(map[string]int),
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
	data, err := os.ReadFile(ws.storagePath)
	if err != nil {
		return err
	}

	var loaded struct {
		ActiveTabID         string    `json:"activeTabId"`
		Version             int64     `json:"version"`
		Tabs                []tabMeta `json:"tabs"`
		KeyboardToolbarMode string    `json:"keyboardToolbarMode"`
		IdleTimeoutSeconds  *int64    `json:"idleTimeoutSeconds"`
	}

	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("parse workspace: %w", err)
	}

	ws.mu.Lock()
	ws.ActiveTabID = loaded.ActiveTabID
	ws.Version = loaded.Version
	ws.Tabs = loaded.Tabs
	if loaded.KeyboardToolbarMode != "" {
		ws.KeyboardToolbarMode = loaded.KeyboardToolbarMode
	}
	if loaded.IdleTimeoutSeconds != nil && *loaded.IdleTimeoutSeconds >= 0 {
		ws.IdleTimeoutSeconds = *loaded.IdleTimeoutSeconds
	}
	ws.rebuildIndexesLocked()
	ws.mu.Unlock()

	updatedActive, sanitizedTabs, changed := sanitizeLoadedWorkspace(loaded.ActiveTabID, loaded.Tabs)
	if changed {
		ws.mu.Lock()
		ws.ActiveTabID = updatedActive
		ws.Tabs = sanitizedTabs
		ws.rebuildIndexesLocked()
		ws.mu.Unlock()
		if err := ws.save(); err != nil {
			return fmt.Errorf("repair workspace: %w", err)
		}
		fmt.Printf("[workspace] repaired persisted layout (deduped %d tabs)\n", len(loaded.Tabs)-len(sanitizedTabs))
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
		IdleTimeoutSeconds  int64     `json:"idleTimeoutSeconds,omitempty"`
	}{
		ActiveTabID:         ws.ActiveTabID,
		Version:             ws.Version,
		Tabs:                append([]tabMeta{}, ws.Tabs...),
		KeyboardToolbarMode: ws.KeyboardToolbarMode,
		IdleTimeoutSeconds:  ws.IdleTimeoutSeconds,
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
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("write workspace: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write workspace: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("sync workspace: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close workspace: %w", err)
	}

	if err := os.Rename(tmpPath, ws.storagePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit workspace: %w", err)
	}

	// Ensure rename is durable by syncing the directory entry
	dirHandle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open workspace dir: %w", err)
	}
	if err := dirHandle.Sync(); err != nil {
		_ = dirHandle.Close()
		return fmt.Errorf("sync workspace dir: %w", err)
	}
	return dirHandle.Close()
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
		IdleTimeoutSeconds  int64     `json:"idleTimeoutSeconds,omitempty"`
	}{
		ActiveTabID:         ws.ActiveTabID,
		Version:             ws.Version,
		Tabs:                append([]tabMeta{}, ws.Tabs...),
		KeyboardToolbarMode: ws.KeyboardToolbarMode,
		IdleTimeoutSeconds:  ws.IdleTimeoutSeconds,
	}

	return json.Marshal(snapshot)
}

// updateState replaces the workspace state
func (ws *workspace) updateState(activeTabID string, tabs []tabMeta) error {
	normalizedActiveID := strings.TrimSpace(activeTabID)
	normalizedTabs, err := validateWorkspaceTabs(normalizedActiveID, tabs)
	if err != nil {
		return err
	}

	ws.mu.Lock()
	ws.ActiveTabID = normalizedActiveID
	ws.Tabs = normalizedTabs
	ws.Version++
	ws.rebuildIndexesLocked()
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "workspace-full-update",
		Timestamp: time.Now().UTC(),
		Payload: mustJSON(struct {
			ActiveTabID string    `json:"activeTabId"`
			Version     int64     `json:"version"`
			Tabs        []tabMeta `json:"tabs"`
		}{
			ActiveTabID: normalizedActiveID,
			Version:     ws.Version,
			Tabs:        normalizedTabs,
		}),
	})

	return nil
}

// addTab adds a new tab to the workspace
func (ws *workspace) addTab(tab tabMeta) error {
	tab.ID = strings.TrimSpace(tab.ID)
	if tab.ID == "" {
		return newValidationError("tab id is required")
	}
	ws.mu.Lock()
	if _, exists := ws.tabIndex[tab.ID]; exists {
		ws.mu.Unlock()
		return fmt.Errorf("tab with ID %s already exists", tab.ID)
	}
	tab.Order = len(ws.Tabs)
	ws.Tabs = append(ws.Tabs, tab)
	ws.Version++
	ws.tabIndex[tab.ID] = len(ws.Tabs) - 1
	if tab.SessionID != nil && *tab.SessionID != "" {
		ws.sessionIndex[*tab.SessionID] = len(ws.Tabs) - 1
	}
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

func validateWorkspaceTabs(activeTabID string, tabs []tabMeta) ([]tabMeta, error) {
	seen := make(map[string]struct{}, len(tabs))
	normalized := make([]tabMeta, 0, len(tabs))

	for _, tab := range tabs {
		trimmedID := strings.TrimSpace(tab.ID)
		if trimmedID == "" {
			return nil, newValidationError("tab id is required")
		}
		if _, exists := seen[trimmedID]; exists {
			return nil, newValidationError("duplicate tab id: %s", trimmedID)
		}

		seen[trimmedID] = struct{}{}
		tab.ID = trimmedID
		tab.Order = len(normalized)
		normalized = append(normalized, tab)
	}

	if activeTabID != "" {
		if _, ok := seen[activeTabID]; !ok {
			return nil, newValidationError("active tab %s not present in workspace", activeTabID)
		}
	}

	return normalized, nil
}

func sanitizeLoadedWorkspace(activeTabID string, tabs []tabMeta) (string, []tabMeta, bool) {
	seen := make(map[string]struct{}, len(tabs))
	sanitized := make([]tabMeta, 0, len(tabs))
	changed := false

	trimmedActive := strings.TrimSpace(activeTabID)

	for _, tab := range tabs {
		trimmedID := strings.TrimSpace(tab.ID)
		if trimmedID == "" {
			changed = true
			continue
		}
		if _, exists := seen[trimmedID]; exists {
			changed = true
			continue
		}
		seen[trimmedID] = struct{}{}
		if tab.ID != trimmedID {
			changed = true
		}
		tab.ID = trimmedID
		if tab.Order != len(sanitized) {
			changed = true
		}
		tab.Order = len(sanitized)
		sanitized = append(sanitized, tab)
	}

	if trimmedActive != "" {
		if _, ok := seen[trimmedActive]; !ok {
			trimmedActive = ""
			changed = true
		}
	}

	if trimmedActive == "" && len(sanitized) > 0 {
		trimmedActive = sanitized[0].ID
		changed = true
	}

	if trimmedActive != activeTabID {
		changed = true
	}

	return trimmedActive, sanitized, changed
}

func (ws *workspace) rebuildIndexesLocked() {
	ws.tabIndex = make(map[string]int, len(ws.Tabs))
	ws.sessionIndex = make(map[string]int)
	for i := range ws.Tabs {
		tab := &ws.Tabs[i]
		tab.Order = i
		ws.tabIndex[tab.ID] = i
		if tab.SessionID != nil && *tab.SessionID != "" {
			ws.sessionIndex[*tab.SessionID] = i
		}
	}
}

func (ws *workspace) idleTimeoutDuration() time.Duration {
	ws.mu.RLock()
	seconds := ws.IdleTimeoutSeconds
	ws.mu.RUnlock()
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func (ws *workspace) setIdleTimeoutSeconds(seconds int64) error {
	if seconds < 0 {
		return newValidationError("idle timeout must be >= 0 seconds")
	}
	if seconds > 0 && seconds > int64(24*time.Hour/time.Second) {
		return newValidationError("idle timeout must be <= 86400 seconds")
	}
	ws.mu.Lock()
	if ws.IdleTimeoutSeconds == seconds {
		ws.mu.Unlock()
		return nil
	}
	ws.IdleTimeoutSeconds = seconds
	ws.Version++
	ws.mu.Unlock()

	if err := ws.save(); err != nil {
		return err
	}

	ws.broadcast(workspaceEvent{
		Type:      "idle-timeout-changed",
		Timestamp: time.Now().UTC(),
		Payload: mustJSON(struct {
			IdleTimeoutSeconds int64 `json:"idleTimeoutSeconds"`
		}{IdleTimeoutSeconds: seconds}),
	})

	return nil
}

// updateTab modifies an existing tab
func (ws *workspace) updateTab(tabID string, label, colorID string) error {
	ws.mu.Lock()
	idx, ok := ws.tabIndex[tabID]
	if !ok {
		ws.mu.Unlock()
		return fmt.Errorf("%w: %s", errTabNotFound, tabID)
	}
	ws.Tabs[idx].Label = label
	ws.Tabs[idx].ColorID = colorID
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
	idx, ok := ws.tabIndex[tabID]
	if !ok {
		ws.mu.Unlock()
		return fmt.Errorf("%w: %s", errTabNotFound, tabID)
	}
	ws.Tabs = append(ws.Tabs[:idx], ws.Tabs[idx+1:]...)
	if ws.ActiveTabID == tabID {
		if len(ws.Tabs) > 0 {
			ws.ActiveTabID = ws.Tabs[0].ID
		} else {
			ws.ActiveTabID = ""
		}
	}
	ws.Version++
	ws.rebuildIndexesLocked()
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
	if tabID != "" {
		if _, ok := ws.tabIndex[tabID]; !ok {
			ws.mu.Unlock()
			return fmt.Errorf("%w: %s", errTabNotFound, tabID)
		}
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
	idx, ok := ws.tabIndex[tabID]
	if !ok {
		ws.mu.Unlock()
		return fmt.Errorf("%w: %s", errTabNotFound, tabID)
	}
	tab := &ws.Tabs[idx]
	if tab.SessionID != nil {
		delete(ws.sessionIndex, *tab.SessionID)
	}
	if existingIdx, exists := ws.sessionIndex[sessionID]; exists && existingIdx != idx {
		ws.Tabs[existingIdx].SessionID = nil
		delete(ws.sessionIndex, sessionID)
	}
	sessionCopy := sessionID
	tab.SessionID = &sessionCopy
	ws.sessionIndex[sessionID] = idx
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
	idx, ok := ws.tabIndex[tabID]
	if !ok {
		ws.mu.Unlock()
		return fmt.Errorf("%w: %s", errTabNotFound, tabID)
	}
	tab := &ws.Tabs[idx]
	if tab.SessionID != nil {
		delete(ws.sessionIndex, *tab.SessionID)
	}
	tab.SessionID = nil
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

func (ws *workspace) detachSessionBySessionID(sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	ws.mu.RLock()
	idx, ok := ws.sessionIndex[sessionID]
	var tabID string
	if ok && idx >= 0 && idx < len(ws.Tabs) {
		tabID = ws.Tabs[idx].ID
	}
	ws.mu.RUnlock()
	if tabID == "" {
		return nil
	}
	return ws.detachSession(tabID)
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
