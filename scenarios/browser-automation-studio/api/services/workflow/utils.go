package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vrooli/browser-automation-studio/database"
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

func (s *WorkflowService) projectWorkflowsDir(project *database.Project) string {
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

func workflowsSubdir(folderPath string) string {
	normalized := normalizeFolderPath(folderPath)
	if normalized == defaultWorkflowFolder {
		return ""
	}
	trimmed := strings.TrimPrefix(normalized, "/")
	return filepath.FromSlash(trimmed)
}

func (s *WorkflowService) desiredWorkflowFilePath(project *database.Project, workflow *database.Workflow) (string, string) {
	subdir := workflowsSubdir(workflow.FolderPath)
	slug := sanitizeWorkflowSlug(workflow.Name)
	fileName := fmt.Sprintf("%s--%s%s", slug, shortID(workflow.ID), workflowFileExt)
	baseDir := s.projectWorkflowsDir(project)
	if subdir != "" {
		return filepath.Join(baseDir, subdir, fileName), filepath.Join(subdir, fileName)
	}
	return filepath.Join(baseDir, fileName), fileName
}

func hashWorkflowDefinition(def database.JSONMap) string {
	// We need deterministic hashing; marshal using sorted keys to avoid random map ordering.
	canonical := canonicalizeJSONMap(def)
	hasher := sha256.New()
	hasher.Write(canonical)
	return hex.EncodeToString(hasher.Sum(nil))
}

func canonicalizeJSONMap(m database.JSONMap) []byte {
	if m == nil {
		return []byte("null")
	}
	// We need stable key ordering for nested maps.
	normalized := normalizeInterfaces(m)
	bytes, err := json.Marshal(normalized)
	if err != nil {
		return []byte("null")
	}
	return bytes
}

func normalizeInterfaces(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for k := range typed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ordered := make(map[string]any, len(typed))
		for _, k := range keys {
			ordered[k] = normalizeInterfaces(typed[k])
		}
		return ordered
	case database.JSONMap:
		keys := make([]string, 0, len(typed))
		for k := range typed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ordered := make(map[string]any, len(typed))
		for _, k := range keys {
			ordered[k] = normalizeInterfaces(typed[k])
		}
		return ordered
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = normalizeInterfaces(item)
		}
		return out
	default:
		return value
	}
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

func ToInterfaceSlice(value any) []any {
	switch typed := value.(type) {
	case nil:
		return []any{}
	case []any:
		return typed
	case []map[string]any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = typed[i]
		}
		return result
	case database.JSONMap:
		result := make([]any, 0, len(typed))
		for _, v := range typed {
			result = append(result, v)
		}
		return result
	default:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return []any{}
		}
		var arr []any
		if err := json.Unmarshal(bytes, &arr); err != nil {
			return []any{}
		}
		return arr
	}
}

func equalStringArrays(a, b pq.StringArray) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func max(values ...int) int {
	result := values[0]
	for _, v := range values[1:] {
		if v > result {
			result = v
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
