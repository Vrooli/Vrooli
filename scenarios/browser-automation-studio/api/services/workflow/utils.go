package workflow

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
)

const (
	workflowDirectoryName   = "workflows"
	workflowFileExt         = ".workflow.json"
	defaultWorkflowFolder   = "/"
	fileSourceManual        = "manual"
	fileSourceAutosave      = "autosave"
	fileSourceFileSync      = "file-sync"
	fileSyncChangeDesc      = "Synchronized from workflow file"
	defaultVersionIncrement = 1
	projectWorkflowPageSize = 500
)

var slugSeparators = regexp.MustCompile(`[^a-z0-9]+`)

// sanitizeWorkflowSlug converts user-provided workflow names into filesystem-safe filename components.
func sanitizeWorkflowSlug(name string) string {
	lowered := strings.ToLower(strings.TrimSpace(name))
	if lowered == "" {
		return "workflow"
	}
	cleaned := slugSeparators.ReplaceAllString(lowered, "-")
	cleaned = strings.Trim(cleaned, "-")
	if cleaned == "" {
		return "workflow"
	}
	// Collapse duplicate separators that may remain after trimming
	cleaned = strings.ReplaceAll(cleaned, "--", "-")
	return cleaned
}

// shortID returns an 8-character identifier for stable filename disambiguation.
func shortID(id uuid.UUID) string {
	return strings.ToLower(id.String()[:8])
}

// ProjectWorkflowsDir returns the absolute path to the workflows directory for a project.
func ProjectWorkflowsDir(project *database.ProjectIndex) string {
	if project == nil {
		return workflowDirectoryName
	}
	root := strings.TrimSpace(project.FolderPath)
	if root == "" {
		return workflowDirectoryName
	}
	return filepath.Join(root, workflowDirectoryName)
}

func normalizeFolderPath(folder string) string {
	trimmed := strings.TrimSpace(folder)
	if trimmed == "" || trimmed == "." {
		return defaultWorkflowFolder
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	// Collapse duplicate slashes and ensure trailing slash is removed unless root
	trimmed = strings.ReplaceAll(trimmed, "//", "/")
	if len(trimmed) > 1 && strings.HasSuffix(trimmed, "/") {
		trimmed = strings.TrimSuffix(trimmed, "/")
	}
	return trimmed
}

// validateFolderPath checks if a folder path looks like a valid logical workflow category.
// Returns an error if the path appears to be an absolute filesystem path.
func validateFolderPath(folder string) error {
	if folder == "" {
		return nil
	}
	// Check for patterns that indicate an absolute filesystem path rather than a logical category
	invalidPrefixes := []string{
		"/home/",
		"/Users/",
		"/var/",
		"/tmp/",
		"/opt/",
		"/usr/",
		"/etc/",
	}
	for _, prefix := range invalidPrefixes {
		if strings.HasPrefix(folder, prefix) {
			return fmt.Errorf("folder_path appears to be an absolute filesystem path; expected a logical category like '/', '/actions', or '/cases'")
		}
	}
	// Also check if path has more than 4 segments - likely a filesystem path
	segments := strings.Split(strings.Trim(folder, "/"), "/")
	if len(segments) > 4 {
		return fmt.Errorf("folder_path has too many segments (%d); expected a logical category like '/', '/actions', or '/cases/foundation'", len(segments))
	}
	return nil
}

func workflowsSubdir(folderPath string) string {
	normalized := normalizeFolderPath(folderPath)
	if normalized == defaultWorkflowFolder {
		return ""
	}
	trimmed := strings.TrimPrefix(normalized, "/")
	return filepath.FromSlash(trimmed)
}

func stringSliceFromAny(value any) []string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		clone := make([]string, len(typed))
		copy(clone, typed)
		return clone
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			switch v := item.(type) {
			case string:
				out = append(out, v)
			default:
				out = append(out, fmt.Sprint(v))
			}
		}
		return out
	default:
		return []string{fmt.Sprint(typed)}
	}
}

func anyToString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func parseFlexibleInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int64:
		return int(typed)
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return int(i)
		}
	case string:
		if typed == "" {
			return 0
		}
		if parsed, err := strconv.Atoi(typed); err == nil {
			return parsed
		}
	}
	return 0
}

// ToInterfaceSlice delegates to typeconv.ToInterfaceSlice for consistency.
// This wrapper handles database.JSONMap as a special case since the typeconv
// package doesn't depend on the database package.
func ToInterfaceSlice(value any) []any {
	// Handle database.JSONMap specially since it's a domain-specific type
	if jsonMap, ok := value.(database.JSONMap); ok {
		result := make([]any, 0, len(jsonMap))
		for _, v := range jsonMap {
			result = append(result, v)
		}
		return result
	}
	// Delegate to the general-purpose implementation
	return typeconv.ToInterfaceSlice(value)
}
