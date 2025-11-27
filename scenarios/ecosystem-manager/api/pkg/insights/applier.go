package insights

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/systemlog"
)

// SuggestionApplier handles applying suggestions to the codebase
type SuggestionApplier struct {
	scenarioRoot string
	backupDir    string
}

// NewSuggestionApplier creates a new suggestion applier
func NewSuggestionApplier(scenarioRoot string) *SuggestionApplier {
	backupDir := filepath.Join(scenarioRoot, ".vrooli", "suggestion-backups")
	return &SuggestionApplier{
		scenarioRoot: scenarioRoot,
		backupDir:    backupDir,
	}
}

// ApplySuggestionResult represents the result of applying a suggestion
type ApplySuggestionResult struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	FilesChanged []string `json:"files_changed,omitempty"`
	BackupPath   string   `json:"backup_path,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// ApplySuggestion applies a suggestion to the codebase
func (sa *SuggestionApplier) ApplySuggestion(suggestion queue.Suggestion, dryRun bool) (*ApplySuggestionResult, error) {
	result := &ApplySuggestionResult{
		Success:      false,
		FilesChanged: []string{},
	}

	if len(suggestion.Changes) == 0 {
		result.Message = "No changes to apply"
		result.Success = true
		return result, nil
	}

	// Create backup directory
	if !dryRun {
		timestamp := time.Now().Format("20060102-150405")
		backupPath := filepath.Join(sa.backupDir, fmt.Sprintf("%s-%s", suggestion.ID, timestamp))
		if err := os.MkdirAll(backupPath, 0755); err != nil {
			return nil, fmt.Errorf("create backup directory: %w", err)
		}
		result.BackupPath = backupPath
	}

	// Apply each change
	for i, change := range suggestion.Changes {
		if err := sa.applyChange(change, result, dryRun); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to apply change %d: %v", i+1, err)
			systemlog.Errorf("Failed to apply change %d for suggestion %s: %v", i+1, suggestion.ID, err)
			return result, err
		}
	}

	result.Success = true
	if dryRun {
		result.Message = fmt.Sprintf("Dry run successful - would apply %d changes to %d files",
			len(suggestion.Changes), len(result.FilesChanged))
	} else {
		result.Message = fmt.Sprintf("Successfully applied %d changes to %d files",
			len(suggestion.Changes), len(result.FilesChanged))
		systemlog.Infof("Applied suggestion %s: %s (%d files changed)",
			suggestion.ID, suggestion.Title, len(result.FilesChanged))
	}

	return result, nil
}

// applyChange applies a single change
func (sa *SuggestionApplier) applyChange(change queue.ProposedChange, result *ApplySuggestionResult, dryRun bool) error {
	// Validate file path (prevent directory traversal)
	filePath := filepath.Join(sa.scenarioRoot, change.File)
	if !strings.HasPrefix(filePath, sa.scenarioRoot) {
		return fmt.Errorf("invalid file path: %s (escapes scenario root)", change.File)
	}

	switch change.Type {
	case "edit":
		return sa.applyFileEdit(filePath, change, result, dryRun)
	case "create":
		return sa.createFile(filePath, change, result, dryRun)
	case "config_update":
		return sa.applyConfigUpdate(filePath, change, result, dryRun)
	default:
		return fmt.Errorf("unknown change type: %s", change.Type)
	}
}

// applyFileEdit applies an edit to an existing file
func (sa *SuggestionApplier) applyFileEdit(filePath string, change queue.ProposedChange, result *ApplySuggestionResult, dryRun bool) error {
	// Read current content
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}
		return fmt.Errorf("read file: %w", err)
	}

	// Create backup
	if !dryRun {
		if err := sa.createBackup(filePath, result.BackupPath); err != nil {
			return fmt.Errorf("create backup: %w", err)
		}
	}

	// Apply edit (simple string replacement)
	currentContent := string(content)
	if !strings.Contains(currentContent, change.Before) {
		return fmt.Errorf("before string not found in file: %s", change.Before)
	}

	newContent := strings.Replace(currentContent, change.Before, change.After, 1)

	// Write updated content
	if !dryRun {
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	result.FilesChanged = append(result.FilesChanged, change.File)
	systemlog.Infof("Applied edit to %s", change.File)
	return nil
}

// createFile creates a new file with the specified content
func (sa *SuggestionApplier) createFile(filePath string, change queue.ProposedChange, result *ApplySuggestionResult, dryRun bool) error {
	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("file already exists: %s", filePath)
	}

	// Create parent directories
	parentDir := filepath.Dir(filePath)
	if !dryRun {
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("create parent directory: %w", err)
		}
	}

	// Write file
	if !dryRun {
		if err := os.WriteFile(filePath, []byte(change.Content), 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	result.FilesChanged = append(result.FilesChanged, change.File)
	systemlog.Infof("Created file %s", change.File)
	return nil
}

// applyConfigUpdate updates a configuration file (JSON)
func (sa *SuggestionApplier) applyConfigUpdate(filePath string, change queue.ProposedChange, result *ApplySuggestionResult, dryRun bool) error {
	// Read current config
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	// Create backup
	if !dryRun {
		if err := sa.createBackup(filePath, result.BackupPath); err != nil {
			return fmt.Errorf("create backup: %w", err)
		}
	}

	// Parse JSON
	var config map[string]any
	if err := json.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("parse JSON config: %w", err)
	}

	// Update config using path
	// Simple implementation: only supports top-level keys
	// For nested paths like "phases[0].timeout", would need more sophisticated path parsing
	if err := updateConfigValue(config, change.ConfigPath, change.ConfigValue); err != nil {
		return fmt.Errorf("update config value: %w", err)
	}

	// Marshal updated config
	updatedContent, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal updated config: %w", err)
	}

	// Validate JSON syntax
	var validateTest map[string]any
	if err := json.Unmarshal(updatedContent, &validateTest); err != nil {
		return fmt.Errorf("validation failed - invalid JSON after update: %w", err)
	}

	// Write updated config
	if !dryRun {
		if err := os.WriteFile(filePath, updatedContent, 0644); err != nil {
			return fmt.Errorf("write config file: %w", err)
		}
	}

	result.FilesChanged = append(result.FilesChanged, change.File)
	systemlog.Infof("Updated config in %s (path: %s)", change.File, change.ConfigPath)
	return nil
}

// createBackup creates a backup of a file
func (sa *SuggestionApplier) createBackup(filePath, backupPath string) error {
	if backupPath == "" {
		return nil // No backup needed (dry run)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file for backup: %w", err)
	}

	// Create backup file with same relative path structure
	relPath, err := filepath.Rel(sa.scenarioRoot, filePath)
	if err != nil {
		return fmt.Errorf("get relative path: %w", err)
	}

	backupFile := filepath.Join(backupPath, relPath)
	backupDir := filepath.Dir(backupFile)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	if err := os.WriteFile(backupFile, content, 0644); err != nil {
		return fmt.Errorf("write backup file: %w", err)
	}

	return nil
}

// updateConfigValue updates a value in a config map using a path like "key" or "nested.key"
func updateConfigValue(config map[string]any, path string, value any) error {
	// Simple implementation for top-level keys
	// For nested paths, would need to parse and traverse the map

	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		// Top-level key
		config[path] = value
		return nil
	}

	// Nested path (basic implementation)
	current := config
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		next, ok := current[key].(map[string]any)
		if !ok {
			// Create intermediate map if it doesn't exist
			next = make(map[string]any)
			current[key] = next
		}
		current = next
	}

	// Set the final value
	current[parts[len(parts)-1]] = value
	return nil
}
