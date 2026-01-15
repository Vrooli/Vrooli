package workflow

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

const (
	// MaxSyncFiles is the maximum number of files to process during sync to prevent runaway operations.
	MaxSyncFiles = 1000
	// MaxSyncDepth is the maximum directory depth to traverse during sync.
	MaxSyncDepth = 4
)

// syncCacheEntry tracks metadata we compute during synchronization so subsequent API calls can resolve workflow file paths quickly.
type syncCacheEntry struct {
	RelativePath string
	AbsolutePath string
	SyncedAt     time.Time
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

// SyncProjectWorkflows synchronizes the workflow and asset DB indexes for a project from the filesystem.
// The filesystem is the source of truth; the DB is an index.
func (s *WorkflowService) SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error {
	return s.syncProjectWorkflows(ctx, projectID)
}

func (s *WorkflowService) syncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	lock := s.getProjectLock(project.ID)
	lock.Lock()
	defer lock.Unlock()

	// Get existing DB records for reconciliation
	dbWorkflows, err := s.listAllProjectWorkflows(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("failed to list existing workflows: %w", err)
	}
	dbAssets, err := s.repo.ListAssetsByProject(ctx, project.ID, 10000, 0)
	if err != nil {
		s.log.WithError(err).Warn("Failed to list existing assets, continuing with empty list")
		dbAssets = nil
	}

	dbWorkflowsByID := make(map[uuid.UUID]*database.WorkflowIndex, len(dbWorkflows))
	for _, wf := range dbWorkflows {
		dbWorkflowsByID[wf.ID] = wf
	}
	dbAssetsByPath := make(map[string]*database.AssetIndex, len(dbAssets))
	for _, asset := range dbAssets {
		dbAssetsByPath[asset.FilePath] = asset
	}

	// Track what we find during the walk
	seenWorkflowIDs := make(map[uuid.UUID]bool)
	seenAssetPaths := make(map[string]bool)
	fileCount := 0

	projectRoot := strings.TrimSpace(project.FolderPath)
	if projectRoot == "" {
		return fmt.Errorf("project has empty folder path")
	}

	// Walk the entire project tree
	walkErr := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Check file limit
		if fileCount >= MaxSyncFiles {
			s.log.WithFields(logrus.Fields{
				"project_id": project.ID,
				"limit":      MaxSyncFiles,
			}).Warn("Max file limit reached during sync, stopping early")
			return filepath.SkipAll
		}

		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		// Calculate relative path and depth
		relPath, relErr := filepath.Rel(projectRoot, path)
		if relErr != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		// Enforce depth limit
		depth := strings.Count(relPath, "/")
		if d.IsDir() && depth >= MaxSyncDepth {
			return filepath.SkipDir
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		fileCount++

		// Get file info
		info, statErr := d.Info()
		if statErr != nil {
			s.log.WithError(statErr).WithField("path", relPath).Debug("Cannot stat file, skipping")
			return nil
		}

		// Check if it's a workflow (JSON file with "nodes" array)
		isWorkflow := false
		if strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			content, readErr := os.ReadFile(path)
			if readErr == nil && isWorkflowContent(content) {
				isWorkflow = true
				if err := s.syncWorkflowFile(ctx, project, path, relPath, content, dbWorkflowsByID, seenWorkflowIDs); err != nil {
					s.log.WithError(err).WithField("path", relPath).Warn("Failed to sync workflow file")
				}
			}
		}

		// If not a workflow, index as an asset
		if !isWorkflow {
			if err := s.syncAssetFile(ctx, project, relPath, info, dbAssetsByPath, seenAssetPaths); err != nil {
				s.log.WithError(err).WithField("path", relPath).Debug("Failed to sync asset file")
			}
		}

		return nil
	})

	if walkErr != nil && walkErr != filepath.SkipAll {
		return fmt.Errorf("failed to walk project directory: %w", walkErr)
	}

	// Reconcile: remove workflows that no longer have backing files
	for id := range dbWorkflowsByID {
		if seenWorkflowIDs[id] {
			continue
		}
		if err := s.repo.DeleteWorkflow(ctx, id); err != nil {
			s.log.WithError(err).WithField("workflow_id", id).Warn("Failed to delete stale workflow")
		} else {
			s.removeWorkflowPath(id)
		}
	}

	// Reconcile: remove assets that no longer have backing files
	for filePath, asset := range dbAssetsByPath {
		if seenAssetPaths[filePath] {
			continue
		}
		if err := s.repo.DeleteAsset(ctx, asset.ID); err != nil {
			s.log.WithError(err).WithField("asset_path", filePath).Debug("Failed to delete stale asset")
		}
	}

	return nil
}

// syncWorkflowFile processes a single workflow file during sync.
func (s *WorkflowService) syncWorkflowFile(
	ctx context.Context,
	project *database.ProjectIndex,
	absPath, relPath string,
	content []byte,
	dbWorkflowsByID map[uuid.UUID]*database.WorkflowIndex,
	seenWorkflowIDs map[uuid.UUID]bool,
) error {
	// Check if it's native format (has valid UUID in "id" field)
	isNative := isNativeWorkflowFormat(content)

	var workflowID uuid.UUID
	var workflowName string
	var workflowFolderPath string
	var workflowVersion int32

	if isNative {
		// Read the native workflow file
		snapshot, readErr := ReadWorkflowSummaryFile(ctx, project, absPath)
		if readErr != nil {
			return fmt.Errorf("read workflow file: %w", readErr)
		}
		if snapshot.Workflow == nil || strings.TrimSpace(snapshot.Workflow.Id) == "" {
			return fmt.Errorf("workflow file has no ID")
		}

		id, parseErr := uuid.Parse(snapshot.Workflow.Id)
		if parseErr != nil {
			return fmt.Errorf("invalid workflow ID: %w", parseErr)
		}

		workflowID = id
		workflowName = snapshot.Workflow.Name
		workflowFolderPath = snapshot.Workflow.FolderPath
		workflowVersion = snapshot.Workflow.Version

		// Cache the path
		s.cacheWorkflowPath(workflowID, absPath, relPath)

		// Write back if normalization was needed
		if snapshot.NeedsWrite {
			if _, _, err := WriteWorkflowSummaryFile(project, snapshot.Workflow, relPath); err != nil {
				s.log.WithError(err).WithField("path", relPath).Warn("Failed to write normalized workflow")
			}
		}
	} else {
		// External format - convert IN-PLACE
		result, convErr := ConvertExternalWorkflow(project, content, relPath)
		if convErr != nil {
			return fmt.Errorf("convert external workflow: %w", convErr)
		}

		id, parseErr := uuid.Parse(result.Workflow.Id)
		if parseErr != nil {
			return fmt.Errorf("invalid converted workflow ID: %w", parseErr)
		}

		workflowID = id
		workflowName = result.Workflow.Name
		workflowFolderPath = result.Workflow.FolderPath
		workflowVersion = result.Workflow.Version

		// Write converted workflow IN-PLACE (overwrite original file)
		if err := WriteWorkflowInPlace(absPath, result.Workflow); err != nil {
			return fmt.Errorf("write converted workflow in-place: %w", err)
		}

		s.log.WithFields(logrus.Fields{
			"path":        relPath,
			"workflow_id": workflowID,
		}).Info("Converted external workflow to native format in-place")

		// Cache the path
		s.cacheWorkflowPath(workflowID, absPath, relPath)
	}

	seenWorkflowIDs[workflowID] = true

	// Check if workflow exists in DB
	existing, exists := dbWorkflowsByID[workflowID]
	if !exists {
		// Create new workflow index
		index := &database.WorkflowIndex{
			ID:         workflowID,
			ProjectID:  &project.ID,
			Name:       workflowName,
			FolderPath: normalizeFolderPath(workflowFolderPath),
			FilePath:   relPath,
			Version:    int(workflowVersion),
		}
		if err := s.repo.CreateWorkflow(ctx, index); err != nil {
			return fmt.Errorf("create workflow index: %w", err)
		}
		return nil
	}

	// Update existing workflow if needed
	needsUpdate := false
	if existing.Name != workflowName {
		existing.Name = workflowName
		needsUpdate = true
	}
	if normalizeFolderPath(existing.FolderPath) != normalizeFolderPath(workflowFolderPath) {
		existing.FolderPath = normalizeFolderPath(workflowFolderPath)
		needsUpdate = true
	}
	if existing.FilePath != relPath {
		existing.FilePath = relPath
		needsUpdate = true
	}
	if existing.Version != int(workflowVersion) && workflowVersion > 0 {
		existing.Version = int(workflowVersion)
		needsUpdate = true
	}

	if needsUpdate {
		if err := s.repo.UpdateWorkflow(ctx, existing); err != nil {
			return fmt.Errorf("update workflow index: %w", err)
		}
	}

	return nil
}

// syncAssetFile processes a single asset file during sync.
func (s *WorkflowService) syncAssetFile(
	ctx context.Context,
	project *database.ProjectIndex,
	relPath string,
	info os.FileInfo,
	dbAssetsByPath map[string]*database.AssetIndex,
	seenAssetPaths map[string]bool,
) error {
	seenAssetPaths[relPath] = true

	// Determine MIME type from extension
	ext := filepath.Ext(relPath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Check if asset exists in DB
	existing, exists := dbAssetsByPath[relPath]
	if !exists {
		// Create new asset index
		asset := &database.AssetIndex{
			ID:        uuid.New(),
			ProjectID: project.ID,
			FilePath:  relPath,
			FileName:  filepath.Base(relPath),
			FileSize:  info.Size(),
			MimeType:  mimeType,
		}
		return s.repo.CreateAsset(ctx, asset)
	}

	// Update existing asset if needed
	needsUpdate := false
	if existing.FileName != filepath.Base(relPath) {
		existing.FileName = filepath.Base(relPath)
		needsUpdate = true
	}
	if existing.FileSize != info.Size() {
		existing.FileSize = info.Size()
		needsUpdate = true
	}
	if existing.MimeType != mimeType {
		existing.MimeType = mimeType
		needsUpdate = true
	}

	if needsUpdate {
		return s.repo.UpdateAsset(ctx, existing)
	}

	return nil
}
