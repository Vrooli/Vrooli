// Package system tests for Claude Code cache health check
// [REQ:SYSTEM-CLAUDE-CACHE-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"vrooli-autoheal/internal/checks"
)

// =============================================================================
// ClaudeCacheCheck Unit Tests with Mock Interfaces
// =============================================================================

// TestClaudeCacheCheckRunWithMock_Healthy tests healthy cache state
// [REQ:SYSTEM-CLAUDE-CACHE-001] [REQ:TEST-SEAM-001]
func TestClaudeCacheCheckRunWithMock_Healthy(t *testing.T) {
	// Create temp directory structure for testing
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(filepath.Join(claudeDir, "todos"), 0755)
	os.WriteFile(filepath.Join(claudeDir, "todos", "test.json"), []byte("{}"), 0644)

	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return tmpDir, nil }),
		WithClaudeCacheThresholds(5000, 10000), // Warning at 5000, critical at 10000
	)
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v (Message: %s)", result.Status, checks.StatusOK, result.Message)
	}
	if result.CheckID != check.ID() {
		t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
	}
	if result.Details["exists"] != true {
		t.Errorf("Details[exists] = %v, want true", result.Details["exists"])
	}
}

// TestClaudeCacheCheckRunWithMock_DirNotExists tests when .claude doesn't exist
func TestClaudeCacheCheckRunWithMock_DirNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create .claude directory

	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return tmpDir, nil }),
	)
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v (dir doesn't exist)", result.Status, checks.StatusOK)
	}
	if result.Details["exists"] != false {
		t.Errorf("Details[exists] = %v, want false", result.Details["exists"])
	}
}

// TestClaudeCacheCheckRunWithMock_Warning tests warning threshold
func TestClaudeCacheCheckRunWithMock_Warning(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	todosDir := filepath.Join(claudeDir, "todos")
	os.MkdirAll(todosDir, 0755)

	// Create files to exceed warning threshold (15 files with low threshold of 10)
	for i := 0; i < 15; i++ {
		filename := filepath.Join(todosDir, "todo_"+string(rune('a'+i))+".json")
		os.WriteFile(filename, []byte("{}"), 0644)
	}

	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return tmpDir, nil }),
		WithClaudeCacheThresholds(10, 20), // Warning at 10, critical at 20
	)
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v (totalFiles=%v)", result.Status, checks.StatusWarning, result.Details["totalFiles"])
	}
}

// TestClaudeCacheCheckRunWithMock_Critical tests critical threshold
func TestClaudeCacheCheckRunWithMock_Critical(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	todosDir := filepath.Join(claudeDir, "todos")
	os.MkdirAll(todosDir, 0755)

	// Create files to exceed critical threshold (25 files with threshold of 20)
	for i := 0; i < 25; i++ {
		filename := filepath.Join(todosDir, "todo_"+string(rune('a'+i))+".json")
		os.WriteFile(filename, []byte("{}"), 0644)
	}

	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return tmpDir, nil }),
		WithClaudeCacheThresholds(10, 20), // Warning at 10, critical at 20
	)
	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want %v (totalFiles=%v)", result.Status, checks.StatusCritical, result.Details["totalFiles"])
	}
}

// TestClaudeCacheCheckRunWithMock_HomeDirError tests home directory error handling
func TestClaudeCacheCheckRunWithMock_HomeDirError(t *testing.T) {
	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return "", checks.ErrFileNotFound }),
	)
	result := check.Run(context.Background())

	if result.Status != checks.StatusWarning {
		t.Errorf("Status = %v, want %v (home dir error)", result.Status, checks.StatusWarning)
	}
	if result.Message != "Could not determine home directory" {
		t.Errorf("Message = %q, want %q", result.Message, "Could not determine home directory")
	}
}

// TestClaudeCacheCheckMetrics tests health metrics calculation
func TestClaudeCacheCheckMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(filepath.Join(claudeDir, "todos"), 0755)
	os.WriteFile(filepath.Join(claudeDir, "todos", "test.json"), []byte("{}"), 0644)

	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return tmpDir, nil }),
	)
	result := check.Run(context.Background())

	if result.Metrics == nil {
		t.Fatal("Metrics should not be nil")
	}
	if result.Metrics.Score == nil {
		t.Fatal("Score should not be nil")
	}
	if len(result.Metrics.SubChecks) < 2 {
		t.Errorf("SubChecks count = %d, want at least 2", len(result.Metrics.SubChecks))
	}
}

// TestClaudeCacheCheckRecoveryActions tests recovery action availability
func TestClaudeCacheCheckRecoveryActions(t *testing.T) {
	tests := []struct {
		name            string
		lastResult      *checks.Result
		expectAvailable map[string]bool
	}{
		{
			name:       "nil result",
			lastResult: nil,
			expectAvailable: map[string]bool{
				"cleanup-stale": false, // No stale files
				"cleanup-all":   true,
				"analyze":       true,
			},
		},
		{
			name: "has stale files",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"staleFiles": 100,
				},
			},
			expectAvailable: map[string]bool{
				"cleanup-stale": true, // Has stale files
				"cleanup-all":   true,
				"analyze":       true,
			},
		},
		{
			name: "no stale files",
			lastResult: &checks.Result{
				Details: map[string]interface{}{
					"staleFiles": 0,
				},
			},
			expectAvailable: map[string]bool{
				"cleanup-stale": false,
				"cleanup-all":   true,
				"analyze":       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewClaudeCacheCheck()
			actions := check.RecoveryActions(tt.lastResult)

			actionMap := make(map[string]checks.RecoveryAction)
			for _, a := range actions {
				actionMap[a.ID] = a
			}

			for id, expectAvail := range tt.expectAvailable {
				action, exists := actionMap[id]
				if !exists {
					t.Errorf("Action %q not found", id)
					continue
				}
				if action.Available != expectAvail {
					t.Errorf("Action %q.Available = %v, want %v", id, action.Available, expectAvail)
				}
			}
		})
	}
}

// TestClaudeCacheCheckRecoveryActionsDangerous tests dangerous action marking
func TestClaudeCacheCheckRecoveryActionsDangerous(t *testing.T) {
	check := NewClaudeCacheCheck()
	actions := check.RecoveryActions(nil)

	actionMap := make(map[string]checks.RecoveryAction)
	for _, a := range actions {
		actionMap[a.ID] = a
	}

	// cleanup-all should be dangerous
	if action, ok := actionMap["cleanup-all"]; ok {
		if !action.Dangerous {
			t.Error("cleanup-all action should be dangerous")
		}
	} else {
		t.Error("cleanup-all action not found")
	}

	// cleanup-stale should be safe
	if action, ok := actionMap["cleanup-stale"]; ok {
		if action.Dangerous {
			t.Error("cleanup-stale action should not be dangerous")
		}
	}

	// analyze should be safe
	if action, ok := actionMap["analyze"]; ok {
		if action.Dangerous {
			t.Error("analyze action should not be dangerous")
		}
	}
}

// TestClaudeCacheCheckExecuteAction_Analyze tests analyze action
func TestClaudeCacheCheckExecuteAction_Analyze(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(filepath.Join(claudeDir, "todos"), 0755)
	os.MkdirAll(filepath.Join(claudeDir, "file-history"), 0755)
	os.WriteFile(filepath.Join(claudeDir, "todos", "test.json"), []byte("{}"), 0644)

	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return tmpDir, nil }),
	)
	result := check.ExecuteAction(context.Background(), "analyze")

	if !result.Success {
		t.Errorf("Success = %v, want true (Error: %s)", result.Success, result.Error)
	}
	if result.ActionID != "analyze" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "analyze")
	}
}

// TestClaudeCacheCheckExecuteAction_CleanupStale tests cleanup-stale action
func TestClaudeCacheCheckExecuteAction_CleanupStale(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	todosDir := filepath.Join(claudeDir, "todos")
	os.MkdirAll(todosDir, 0755)
	os.WriteFile(filepath.Join(todosDir, "test.json"), []byte("{}"), 0644)

	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return tmpDir, nil }),
	)
	result := check.ExecuteAction(context.Background(), "cleanup-stale")

	if !result.Success {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.ActionID != "cleanup-stale" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "cleanup-stale")
	}
}

// TestClaudeCacheCheckExecuteAction_CleanupAll tests cleanup-all action
func TestClaudeCacheCheckExecuteAction_CleanupAll(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	todosDir := filepath.Join(claudeDir, "todos")
	os.MkdirAll(todosDir, 0755)
	os.WriteFile(filepath.Join(todosDir, "test.json"), []byte("{}"), 0644)

	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return tmpDir, nil }),
	)
	result := check.ExecuteAction(context.Background(), "cleanup-all")

	if !result.Success {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.ActionID != "cleanup-all" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "cleanup-all")
	}
}

// TestClaudeCacheCheckExecuteAction_UnknownAction tests unknown action handling
func TestClaudeCacheCheckExecuteAction_UnknownAction(t *testing.T) {
	check := NewClaudeCacheCheck()
	result := check.ExecuteAction(context.Background(), "unknown-action")

	if result.Success {
		t.Error("Success should be false for unknown action")
	}
	if result.Error != "unknown action: unknown-action" {
		t.Errorf("Error = %q, want %q", result.Error, "unknown action: unknown-action")
	}
}

// TestClaudeCacheCheckInterface verifies Check interface implementation
func TestClaudeCacheCheckInterface(t *testing.T) {
	var _ checks.Check = (*ClaudeCacheCheck)(nil)
	var _ checks.HealableCheck = (*ClaudeCacheCheck)(nil)

	check := NewClaudeCacheCheck()
	if check.ID() != "system-claude-cache" {
		t.Errorf("ID() = %q, want %q", check.ID(), "system-claude-cache")
	}
	if check.Title() == "" {
		t.Error("Title() is empty")
	}
	if check.Description() == "" {
		t.Error("Description() is empty")
	}
	if check.Importance() == "" {
		t.Error("Importance() is empty")
	}
	if check.IntervalSeconds() <= 0 {
		t.Error("IntervalSeconds() should be positive")
	}
	if check.Category() != checks.CategorySystem {
		t.Errorf("Category() = %v, want %v", check.Category(), checks.CategorySystem)
	}
	// Claude cache check runs on all platforms
	if check.Platforms() != nil {
		t.Error("Platforms() should be nil (all platforms)")
	}
}

// TestClaudeCacheCheckHomeDirFuncInjection verifies home dir func is properly injected
func TestClaudeCacheCheckHomeDirFuncInjection(t *testing.T) {
	customFunc := func() (string, error) { return "/custom/path", nil }
	check := NewClaudeCacheCheck(WithHomeDirFunc(customFunc))

	dir, _ := check.homeDirFunc()
	if dir != "/custom/path" {
		t.Error("Home dir func was not properly injected")
	}
}

// TestFormatCacheBytes tests byte formatting function
func TestFormatCacheBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatCacheBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatCacheBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

// TestClaudeCacheCheckCategorization tests file categorization
func TestClaudeCacheCheckCategorization(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")

	// Create different directory types
	os.MkdirAll(filepath.Join(claudeDir, "todos"), 0755)
	os.MkdirAll(filepath.Join(claudeDir, "file-history"), 0755)
	os.MkdirAll(filepath.Join(claudeDir, "shell-snapshots"), 0755)

	// Create files in each
	os.WriteFile(filepath.Join(claudeDir, "todos", "t1.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(claudeDir, "todos", "t2.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(claudeDir, "file-history", "h1.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(claudeDir, "shell-snapshots", "s1.json"), []byte("{}"), 0644)

	check := NewClaudeCacheCheck(
		WithHomeDirFunc(func() (string, error) { return tmpDir, nil }),
	)
	result := check.Run(context.Background())

	if result.Details["todoFiles"] != 2 {
		t.Errorf("Details[todoFiles] = %v, want 2", result.Details["todoFiles"])
	}
	if result.Details["historyFiles"] != 1 {
		t.Errorf("Details[historyFiles] = %v, want 1", result.Details["historyFiles"])
	}
	if result.Details["shellFiles"] != 1 {
		t.Errorf("Details[shellFiles] = %v, want 1", result.Details["shellFiles"])
	}
	if result.Details["totalFiles"] != 4 {
		t.Errorf("Details[totalFiles] = %v, want 4", result.Details["totalFiles"])
	}
}
