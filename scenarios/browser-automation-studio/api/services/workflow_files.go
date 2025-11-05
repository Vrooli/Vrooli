package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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

// workflowFileSnapshot captures the parsed state of a workflow file prior to reconciling it with the database.
type workflowFileSnapshot struct {
	Workflow         *database.Workflow
	Nodes            []any
	Edges            []any
	RelativePath     string
	AbsolutePath     string
	RawJSON          []byte
	FileVersion      int
	ChangeDesc       string
	SourceLabel      string
	NeedsWriteBack   bool
	FileHash         string
	LastModifiedTime time.Time
}

// syncCacheEntry tracks metadata we compute during synchronization so subsequent API calls can resolve workflow file paths quickly.
type syncCacheEntry struct {
	RelativePath string
	AbsolutePath string
	SyncedAt     time.Time
}

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

func (s *WorkflowService) readWorkflowFile(ctx context.Context, project *database.Project, absPath string) (*workflowFileSnapshot, error) {
	rel, err := filepath.Rel(s.projectWorkflowsDir(project), absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine relative workflow path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat workflow file: %w", err)
	}
	rawBytes, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}
	// Allow empty files to be treated as a blank workflow shell.
	if len(rawBytes) == 0 {
		rawBytes = []byte("{}")
	}
	var payload map[string]any
	if err := json.Unmarshal(rawBytes, &payload); err != nil {
		return nil, fmt.Errorf("invalid workflow JSON in %s: %w", rel, err)
	}

	idValue := anyToString(payload["id"])
	var workflowID uuid.UUID
	var generatedID bool
	if idValue == "" {
		workflowID = uuid.New()
		generatedID = true
	} else {
		parsed, parseErr := uuid.Parse(idValue)
		if parseErr != nil {
			return nil, fmt.Errorf("workflow file %s has invalid id: %w", rel, parseErr)
		}
		workflowID = parsed
	}

	name := strings.TrimSpace(anyToString(payload["name"]))
	if name == "" {
		base := filepath.Base(absPath)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	folderPath := normalizeFolderPath(anyToString(payload["folder_path"]))
	if folderPath == defaultWorkflowFolder {
		// If the file lives in a nested directory and did not specify folder_path, derive it from the path.
		dir := filepath.Dir(rel)
		if dir != "." {
			folderPath = normalizeFolderPath(dir)
		}
	}

	description := strings.TrimSpace(anyToString(payload["description"]))
	tags := pq.StringArray(stringSliceFromAny(payload["tags"]))
	changeDesc := strings.TrimSpace(anyToString(payload["change_description"]))
	sourceLabel := strings.TrimSpace(anyToString(payload["source"]))
	if changeDesc == "" {
		changeDesc = fileSyncChangeDesc
	}
	if sourceLabel == "" {
		sourceLabel = fileSourceFileSync
	}

	versionHint := parseFlexibleInt(payload["version"])

	var flowCandidate map[string]any
	if rawFlow, ok := payload["flow_definition"].(map[string]any); ok {
		flowCandidate = rawFlow
	} else {
		flowCandidate = make(map[string]any)
		if nodes, ok := payload["nodes"]; ok {
			flowCandidate["nodes"] = nodes
		}
		if edges, ok := payload["edges"]; ok {
			flowCandidate["edges"] = edges
		}
	}

	// Ensure we always have a nodes/edges tuple even if empty.
	if _, hasNodes := flowCandidate["nodes"]; !hasNodes {
		flowCandidate["nodes"] = []any{}
	}
	if _, hasEdges := flowCandidate["edges"]; !hasEdges {
		flowCandidate["edges"] = []any{}
	}

	normalized, normErr := normalizeFlowDefinition(flowCandidate)
	if normErr != nil {
		return nil, fmt.Errorf("workflow file %s has invalid flow definition: %w", rel, normErr)
	}

	nodes := toInterfaceSlice(normalized["nodes"])
	edges := toInterfaceSlice(normalized["edges"])

	workflow := &database.Workflow{
		ID:                    workflowID,
		ProjectID:             &project.ID,
		Name:                  name,
		FolderPath:            folderPath,
		Description:           description,
		Tags:                  tags,
		FlowDefinition:        normalized,
		Version:               versionHint,
		CreatedBy:             strings.TrimSpace(anyToString(payload["created_by"])),
		LastChangeDescription: changeDesc,
		LastChangeSource:      sourceLabel,
	}

	snapshot := &workflowFileSnapshot{
		Workflow:         workflow,
		Nodes:            nodes,
		Edges:            edges,
		RelativePath:     filepath.ToSlash(rel),
		AbsolutePath:     absPath,
		RawJSON:          rawBytes,
		FileVersion:      versionHint,
		ChangeDesc:       changeDesc,
		SourceLabel:      sourceLabel,
		NeedsWriteBack:   generatedID,
		FileHash:         hashWorkflowDefinition(normalized),
		LastModifiedTime: info.ModTime(),
	}

	return snapshot, nil
}

func toInterfaceSlice(value any) []any {
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

func (s *WorkflowService) getProjectLock(projectID uuid.UUID) *sync.Mutex {
	value, _ := s.syncLocks.LoadOrStore(projectID, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func (s *WorkflowService) cacheWorkflowPath(workflowID uuid.UUID, absPath, relPath string) {
	s.filePathCache.Store(workflowID, syncCacheEntry{
		RelativePath: filepath.ToSlash(relPath),
		AbsolutePath: absPath,
		SyncedAt:     time.Now(),
	})
}

func (s *WorkflowService) removeWorkflowPath(workflowID uuid.UUID) {
	s.filePathCache.Delete(workflowID)
}

func (s *WorkflowService) lookupWorkflowPath(workflowID uuid.UUID) (syncCacheEntry, bool) {
	value, ok := s.filePathCache.Load(workflowID)
	if !ok {
		return syncCacheEntry{}, false
	}
	entry, entryOK := value.(syncCacheEntry)
	return entry, entryOK
}

func (s *WorkflowService) listAllProjectWorkflows(ctx context.Context, projectID uuid.UUID) ([]*database.Workflow, error) {
	var all []*database.Workflow
	offset := 0
	for {
		batch, err := s.repo.ListWorkflowsByProject(ctx, projectID, projectWorkflowPageSize, offset)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < projectWorkflowPageSize {
			break
		}
		offset += len(batch)
	}
	return all, nil
}

func (s *WorkflowService) writeWorkflowFile(project *database.Project, workflow *database.Workflow, nodes, edges []any, preferredPath string) (string, string, error) {
	workflow.FlowDefinition = sanitizeWorkflowDefinition(workflow.FlowDefinition)
	nodes = toInterfaceSlice(workflow.FlowDefinition["nodes"])
	edges = toInterfaceSlice(workflow.FlowDefinition["edges"])
	desiredAbs, desiredRel := s.desiredWorkflowFilePath(project, workflow)
	targetAbs := desiredAbs
	targetRel := desiredRel

	if preferredPath != "" {
		targetAbs = filepath.Join(s.projectWorkflowsDir(project), filepath.FromSlash(preferredPath))
		targetRel = preferredPath
	}

	// Ensure directory exists before attempting to write.
	dir := filepath.Dir(targetAbs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("failed to create workflow directory: %w", err)
	}

	s.coalesceWorkflowChangeMetadata(workflow)

	payload := map[string]any{
		"id":          workflow.ID.String(),
		"name":        workflow.Name,
		"folder_path": workflow.FolderPath,
		"description": workflow.Description,
		"tags":        []string(workflow.Tags),
		"version":     workflow.Version,
		"flow_definition": map[string]any{
			"nodes": nodes,
			"edges": edges,
		},
		"nodes":      nodes,
		"edges":      edges,
		"updated_at": workflow.UpdatedAt.UTC().Format(time.RFC3339),
		"created_at": workflow.CreatedAt.UTC().Format(time.RFC3339),
	}
	if workflow.CreatedBy != "" {
		payload["created_by"] = workflow.CreatedBy
	}
	if strings.TrimSpace(workflow.LastChangeDescription) != "" {
		payload["change_description"] = workflow.LastChangeDescription
	}
	if strings.TrimSpace(workflow.LastChangeSource) != "" {
		payload["source"] = workflow.LastChangeSource
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal workflow file: %w", err)
	}

	tmpFile := targetAbs + ".tmp"
	if err := os.WriteFile(tmpFile, bytes, 0o644); err != nil {
		return "", "", fmt.Errorf("failed to write workflow temp file: %w", err)
	}

	if err := os.Rename(tmpFile, targetAbs); err != nil {
		return "", "", fmt.Errorf("failed to finalize workflow file write: %w", err)
	}

	return targetAbs, targetRel, nil
}

func (s *WorkflowService) syncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	lock := s.getProjectLock(project.ID)
	lock.Lock()
	defer lock.Unlock()

	workflowsRoot := s.projectWorkflowsDir(project)
	if err := os.MkdirAll(workflowsRoot, 0o755); err != nil {
		return fmt.Errorf("failed to ensure workflows directory: %w", err)
	}

	// Discover file snapshots.
	snapshots := make(map[uuid.UUID]*workflowFileSnapshot)
	var discoveryErr error
	err = filepath.WalkDir(workflowsRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), workflowFileExt) {
			return nil
		}
		snapshot, readErr := s.readWorkflowFile(ctx, project, path)
		if readErr != nil {
			discoveryErr = readErr
			return readErr
		}
		snapshots[snapshot.Workflow.ID] = snapshot
		s.cacheWorkflowPath(snapshot.Workflow.ID, snapshot.AbsolutePath, snapshot.RelativePath)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan workflow directory: %w", err)
	}
	if discoveryErr != nil {
		return discoveryErr
	}

	dbWorkflows, err := s.listAllProjectWorkflows(ctx, project.ID)
	if err != nil {
		return err
	}

	dbByID := make(map[uuid.UUID]*database.Workflow, len(dbWorkflows))
	for _, wf := range dbWorkflows {
		dbByID[wf.ID] = wf
	}

	processed := make(map[uuid.UUID]bool)

	for id, snapshot := range snapshots {
		processed[id] = true
		fileWF := snapshot.Workflow
		existing, exists := dbByID[id]
		if !exists {
			// Create new workflow from file.
			now := time.Now().UTC()
			fileWF.Version = max(fileWF.Version, defaultVersionIncrement)
			fileWF.CreatedAt = now
			fileWF.UpdatedAt = now
			fileWF.LastChangeSource = firstNonEmpty(snapshot.SourceLabel, fileSourceFileSync)
			fileWF.LastChangeDescription = firstNonEmpty(snapshot.ChangeDesc, fileSyncChangeDesc)
			if err := s.repo.CreateWorkflow(ctx, fileWF); err != nil {
				return fmt.Errorf("failed to create workflow from file %s: %w", snapshot.RelativePath, err)
			}
			version := &database.WorkflowVersion{
				WorkflowID:        fileWF.ID,
				Version:           fileWF.Version,
				FlowDefinition:    fileWF.FlowDefinition,
				ChangeDescription: firstNonEmpty(snapshot.ChangeDesc, fileSyncChangeDesc),
				CreatedBy:         firstNonEmpty(snapshot.SourceLabel, fileSourceFileSync),
			}
			if err := s.repo.CreateWorkflowVersion(ctx, version); err != nil {
				return fmt.Errorf("failed to record initial workflow version: %w", err)
			}
			if _, _, err := s.writeWorkflowFile(project, fileWF, snapshot.Nodes, snapshot.Edges, snapshot.RelativePath); err != nil {
				return err
			}
			continue
		}

		needsUpdate := false
		updateReason := fileSyncChangeDesc

		// Align metadata
		if existing.Name != fileWF.Name {
			existing.Name = fileWF.Name
			needsUpdate = true
		}
		if existing.Description != fileWF.Description {
			existing.Description = fileWF.Description
			needsUpdate = true
		}
		if existing.FolderPath != fileWF.FolderPath {
			existing.FolderPath = fileWF.FolderPath
			needsUpdate = true
		}
		if !equalStringArrays(existing.Tags, fileWF.Tags) {
			existing.Tags = fileWF.Tags
			needsUpdate = true
		}

		fileHash := hashWorkflowDefinition(fileWF.FlowDefinition)
		dbHash := hashWorkflowDefinition(existing.FlowDefinition)
		if fileHash != dbHash {
			existing.FlowDefinition = fileWF.FlowDefinition
			needsUpdate = true
		}

		if needsUpdate {
			existing.Version = max(existing.Version+1, max(fileWF.Version, defaultVersionIncrement))
			existing.UpdatedAt = time.Now().UTC()
			if existing.CreatedAt.IsZero() {
				existing.CreatedAt = existing.UpdatedAt
			}
			existing.LastChangeSource = firstNonEmpty(snapshot.SourceLabel, fileSourceFileSync)
			existing.LastChangeDescription = firstNonEmpty(snapshot.ChangeDesc, updateReason)
			if err := s.repo.UpdateWorkflow(ctx, existing); err != nil {
				return fmt.Errorf("failed to update workflow %s from file: %w", existing.ID, err)
			}
			version := &database.WorkflowVersion{
				WorkflowID:        existing.ID,
				Version:           existing.Version,
				FlowDefinition:    existing.FlowDefinition,
				ChangeDescription: firstNonEmpty(snapshot.ChangeDesc, updateReason),
				CreatedBy:         firstNonEmpty(snapshot.SourceLabel, fileSourceFileSync),
			}
			if err := s.repo.CreateWorkflowVersion(ctx, version); err != nil {
				return fmt.Errorf("failed to record workflow version: %w", err)
			}
			if _, _, err := s.writeWorkflowFile(project, existing, snapshot.Nodes, snapshot.Edges, snapshot.RelativePath); err != nil {
				return err
			}
			continue
		}

		// No changes but ensure file has canonical metadata when ID was missing, etc.
		if snapshot.NeedsWriteBack {
			existing.UpdatedAt = time.Now().UTC()
			if existing.CreatedAt.IsZero() {
				existing.CreatedAt = existing.UpdatedAt
			}
			nodes := snapshot.Nodes
			edges := snapshot.Edges
			if _, _, err := s.writeWorkflowFile(project, existing, nodes, edges, snapshot.RelativePath); err != nil {
				return err
			}
		}
	}

	// Remove workflows that no longer have backing files.
	for id := range dbByID {
		if processed[id] {
			continue
		}
		if err := s.repo.DeleteWorkflow(ctx, id); err != nil {
			return fmt.Errorf("failed to delete workflow %s removed from filesystem: %w", id, err)
		}
		s.removeWorkflowPath(id)
	}

	return nil
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
