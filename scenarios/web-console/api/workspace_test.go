package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkspaceNewAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "workspace.json")

	// Create new workspace
	ws, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	if ws.Version != 0 {
		t.Errorf("Expected version 0, got %d", ws.Version)
	}
	if len(ws.Tabs) != 0 {
		t.Errorf("Expected 0 tabs, got %d", len(ws.Tabs))
	}

	// Add a tab
	tab := tabMeta{
		ID:      "tab-1",
		Label:   "Test Tab",
		ColorID: "sky",
		Order:   0,
	}
	if err := ws.addTab(tab); err != nil {
		t.Fatalf("Failed to add tab: %v", err)
	}

	// Verify version incremented
	if ws.Version != 1 {
		t.Errorf("Expected version 1, got %d", ws.Version)
	}

	// Load workspace from disk
	ws2, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to load workspace: %v", err)
	}

	if ws2.Version != 1 {
		t.Errorf("Expected loaded version 1, got %d", ws2.Version)
	}
	if len(ws2.Tabs) != 1 {
		t.Fatalf("Expected 1 tab, got %d", len(ws2.Tabs))
	}
	if ws2.Tabs[0].ID != "tab-1" {
		t.Errorf("Expected tab ID 'tab-1', got '%s'", ws2.Tabs[0].ID)
	}
}

func TestWorkspaceUpdateTab(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "workspace.json")

	ws, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Add a tab
	tab := tabMeta{
		ID:      "tab-1",
		Label:   "Original Label",
		ColorID: "sky",
	}
	if err := ws.addTab(tab); err != nil {
		t.Fatalf("Failed to add tab: %v", err)
	}

	// Update the tab
	if err := ws.updateTab("tab-1", "Updated Label", "emerald"); err != nil {
		t.Fatalf("Failed to update tab: %v", err)
	}

	// Verify changes
	ws.mu.RLock()
	if ws.Tabs[0].Label != "Updated Label" {
		t.Errorf("Expected label 'Updated Label', got '%s'", ws.Tabs[0].Label)
	}
	if ws.Tabs[0].ColorID != "emerald" {
		t.Errorf("Expected color 'emerald', got '%s'", ws.Tabs[0].ColorID)
	}
	ws.mu.RUnlock()

	// Update non-existent tab
	if err := ws.updateTab("non-existent", "Label", "color"); err == nil {
		t.Error("Expected error updating non-existent tab")
	}
}

func TestWorkspaceRemoveTab(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "workspace.json")

	ws, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Add two tabs
	tab1 := tabMeta{ID: "tab-1", Label: "Tab 1", ColorID: "sky"}
	tab2 := tabMeta{ID: "tab-2", Label: "Tab 2", ColorID: "emerald"}

	if err := ws.addTab(tab1); err != nil {
		t.Fatalf("Failed to add tab1: %v", err)
	}
	if err := ws.addTab(tab2); err != nil {
		t.Fatalf("Failed to add tab2: %v", err)
	}

	ws.ActiveTabID = "tab-1"

	// Remove first tab
	if err := ws.removeTab("tab-1"); err != nil {
		t.Fatalf("Failed to remove tab: %v", err)
	}

	ws.mu.RLock()
	if len(ws.Tabs) != 1 {
		t.Errorf("Expected 1 tab after removal, got %d", len(ws.Tabs))
	}
	if ws.Tabs[0].ID != "tab-2" {
		t.Errorf("Expected remaining tab 'tab-2', got '%s'", ws.Tabs[0].ID)
	}
	// Active tab should change to remaining tab
	if ws.ActiveTabID != "tab-2" {
		t.Errorf("Expected active tab 'tab-2', got '%s'", ws.ActiveTabID)
	}
	ws.mu.RUnlock()
}

func TestWorkspaceSessionAttachDetach(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "workspace.json")

	ws, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Add a tab
	tab := tabMeta{ID: "tab-1", Label: "Test Tab", ColorID: "sky"}
	if err := ws.addTab(tab); err != nil {
		t.Fatalf("Failed to add tab: %v", err)
	}

	// Attach session
	sessionID := "session-123"
	if err := ws.attachSession("tab-1", sessionID); err != nil {
		t.Fatalf("Failed to attach session: %v", err)
	}

	ws.mu.RLock()
	if ws.Tabs[0].SessionID == nil {
		t.Error("Expected session to be attached")
	} else if *ws.Tabs[0].SessionID != sessionID {
		t.Errorf("Expected session ID '%s', got '%s'", sessionID, *ws.Tabs[0].SessionID)
	}
	ws.mu.RUnlock()

	// Detach session
	if err := ws.detachSession("tab-1"); err != nil {
		t.Fatalf("Failed to detach session: %v", err)
	}

	ws.mu.RLock()
	if ws.Tabs[0].SessionID != nil {
		t.Error("Expected session to be detached")
	}
	ws.mu.RUnlock()
}

func TestWorkspaceBroadcast(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "workspace.json")

	ws, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Subscribe to events
	ch1 := ws.subscribe()
	ch2 := ws.subscribe()

	// Add a tab (should broadcast)
	tab := tabMeta{ID: "tab-1", Label: "Test Tab", ColorID: "sky"}
	if err := ws.addTab(tab); err != nil {
		t.Fatalf("Failed to add tab: %v", err)
	}

	// Both channels should receive the event
	timeout := time.After(1 * time.Second)
	select {
	case event := <-ch1:
		if event.Type != "tab-added" {
			t.Errorf("Expected 'tab-added' event, got '%s'", event.Type)
		}
	case <-timeout:
		t.Error("Timeout waiting for event on channel 1")
	}

	select {
	case event := <-ch2:
		if event.Type != "tab-added" {
			t.Errorf("Expected 'tab-added' event, got '%s'", event.Type)
		}
	case <-timeout:
		t.Error("Timeout waiting for event on channel 2")
	}

	// Unsubscribe
	ws.unsubscribe(ch1)
	ws.unsubscribe(ch2)
}

func TestWorkspaceUpdateStateValidation(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "workspace.json")

	ws, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Valid update should trim IDs and normalise order
	validTabs := []tabMeta{
		{ID: " tab-1 ", Label: "One", ColorID: "sky", Order: 42},
		{ID: "tab-2", Label: "Two", ColorID: "emerald", Order: 7},
	}
	if err := ws.updateState("tab-1", validTabs); err != nil {
		t.Fatalf("Expected valid update to succeed, got error: %v", err)
	}
	ws.mu.RLock()
	if ws.ActiveTabID != "tab-1" {
		t.Fatalf("Expected active tab 'tab-1', got '%s'", ws.ActiveTabID)
	}
	if len(ws.Tabs) != 2 {
		t.Fatalf("Expected 2 tabs, got %d", len(ws.Tabs))
	}
	if ws.Tabs[0].ID != "tab-1" || ws.Tabs[0].Order != 0 {
		t.Errorf("Expected first tab to be order 0 with ID 'tab-1', got ID '%s' order %d", ws.Tabs[0].ID, ws.Tabs[0].Order)
	}
	if ws.Tabs[1].Order != 1 {
		t.Errorf("Expected second tab order to be 1, got %d", ws.Tabs[1].Order)
	}
	ws.mu.RUnlock()

	// Duplicate IDs should fail validation
	duplicateTabs := []tabMeta{
		{ID: "tab-dup", Label: "Dup", ColorID: "sky"},
		{ID: "tab-dup", Label: "Dup2", ColorID: "emerald"},
	}
	err = ws.updateState("tab-dup", duplicateTabs)
	if err == nil {
		t.Fatal("Expected duplicate tab IDs to return validation error")
	}
	if _, ok := err.(workspaceValidationError); !ok {
		t.Fatalf("Expected workspaceValidationError, got %T", err)
	}

	// Active tab must exist within payload
	missingActiveTabs := []tabMeta{{ID: "tab-a"}}
	err = ws.updateState("tab-missing", missingActiveTabs)
	if err == nil {
		t.Fatal("Expected missing active tab to return validation error")
	}
	if _, ok := err.(workspaceValidationError); !ok {
		t.Fatalf("Expected workspaceValidationError for missing active tab, got %T", err)
	}
}

func TestWorkspacePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	storagePath := filepath.Join(tmpDir, "workspace.json")

	// Create workspace and add data
	ws1, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	tab := tabMeta{ID: "tab-1", Label: "Persistent Tab", ColorID: "violet"}
	if err := ws1.addTab(tab); err != nil {
		t.Fatalf("Failed to add tab: %v", err)
	}
	ws1.setActiveTab("tab-1")

	sessionID := "session-xyz"
	if err := ws1.attachSession("tab-1", sessionID); err != nil {
		t.Fatalf("Failed to attach session: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		t.Error("Workspace file was not created")
	}

	// Load workspace from disk
	ws2, err := newWorkspace(storagePath)
	if err != nil {
		t.Fatalf("Failed to load workspace: %v", err)
	}

	// Verify all data persisted
	ws2.mu.RLock()
	if len(ws2.Tabs) != 1 {
		t.Fatalf("Expected 1 tab, got %d", len(ws2.Tabs))
	}
	if ws2.Tabs[0].Label != "Persistent Tab" {
		t.Errorf("Expected label 'Persistent Tab', got '%s'", ws2.Tabs[0].Label)
	}
	if ws2.Tabs[0].SessionID == nil || *ws2.Tabs[0].SessionID != sessionID {
		t.Error("Session ID not persisted correctly")
	}
	if ws2.ActiveTabID != "tab-1" {
		t.Errorf("Expected active tab 'tab-1', got '%s'", ws2.ActiveTabID)
	}
	ws2.mu.RUnlock()
}
