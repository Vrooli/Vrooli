// Package workflow provides workflow management services including filesystem-database synchronization.
//
// # Reconciliation Overview
//
// This file implements the backend workflow sync system, one of three reconciliation
// systems in browser-automation-studio. See docs/architecture/reconciliation.md for
// the full architecture.
//
// # Problem
//
// Workflows are stored as JSON files on disk (source of truth) but we need a database
// index for fast queries. These can drift apart when files are modified externally,
// deleted, or when database operations fail.
//
// # Solution
//
// SyncProjectWorkflows performs bidirectional reconciliation:
//  1. Walk the filesystem to discover current state
//  2. Upsert database records to match filesystem
//  3. Delete stale database records that have no backing file
//
// # Design Decisions
//
// - Filesystem as source of truth: Users can version-control workflows with git
// - Per-project locking: Prevents concurrent syncs from corrupting state
// - File limits (1000 files, depth 4): Prevents runaway operations on large projects
// - In-place format conversion: External workflows are normalized to native format
//
// # Related Files
//
// - ui/src/domains/recording/utils/mergeActions.ts (frontend action merging)
// - ui/src/domains/recording/types/timeline-unified.ts (AI step reconciliation)
// - ui/src/domains/recording/RecordingSession.tsx (usage context)
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
	// MaxSyncFiles prevents runaway operations on unexpectedly large projects.
	// At 1000 files, sync completes in ~1-2 seconds on typical hardware.
	// Projects exceeding this likely have nested node_modules or similar that should be excluded.
	MaxSyncFiles = 1000

	// MaxSyncDepth prevents traversing into deeply nested structures (node_modules, .git objects).
	// Typical project structure: project/workflows/category/workflow.json (depth 3).
	// Depth 4 provides one extra level of flexibility.
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

// SyncProjectWorkflows reconciles the database index with the filesystem for a project.
//
// The filesystem is the source of truth. This function:
//  1. Walks the project directory to find all workflow and asset files
//  2. Creates or updates database records to match what's on disk
//  3. Deletes database records for files that no longer exist (garbage collection)
//
// Call this after file operations, on project load, or when external changes are suspected.
// The operation is idempotent - calling it multiple times with no filesystem changes is safe.
//
// Thread-safety: Uses per-project mutex to prevent concurrent syncs.
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

	projectRoot := strings.TrimSpace(project.FolderPath)
	if projectRoot == "" {
		return fmt.Errorf("project has empty folder path")
	}

	// Phase 1: Load DB state for O(1) lookups
	dbState, err := s.loadDBState(ctx, project.ID)
	if err != nil {
		return err
	}

	// Phase 2: Walk filesystem and reconcile
	seen, err := s.scanAndReconcile(ctx, project, projectRoot, dbState)
	if err != nil {
		return err
	}

	// Phase 3: Garbage collection
	return s.garbageCollect(ctx, dbState, seen)
}

// loadDBState loads existing database records into maps for O(1) lookups during sync.
// This enables efficient detection of unchanged files (skip writes) and stale records (delete).
func (s *WorkflowService) loadDBState(ctx context.Context, projectID uuid.UUID) (*DBState, error) {
	dbWorkflows, err := s.listAllProjectWorkflows(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing workflows: %w", err)
	}

	dbAssets, err := s.repo.ListAssetsByProject(ctx, projectID, 10000, 0)
	if err != nil {
		s.log.WithError(err).Warn("Failed to list existing assets, continuing with empty list")
		dbAssets = nil
	}

	state := NewDBState()
	for _, wf := range dbWorkflows {
		state.WorkflowsByID[wf.ID] = wf
	}
	for _, asset := range dbAssets {
		state.AssetsByPath[asset.FilePath] = asset
	}

	return state, nil
}

// scanAndReconcile walks the filesystem and syncs files with the database.
// Returns SeenState tracking which files were found on disk.
func (s *WorkflowService) scanAndReconcile(
	ctx context.Context,
	project *database.ProjectIndex,
	projectRoot string,
	dbState *DBState,
) (*SeenState, error) {
	seen := NewSeenState()
	fileCount := 0

	walkErr := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Enforce file limit
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
		if d.IsDir() && strings.Count(relPath, "/") >= MaxSyncDepth {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		fileCount++
		return s.syncSingleFile(ctx, project, path, relPath, d, dbState, seen)
	})

	if walkErr != nil && walkErr != filepath.SkipAll {
		return nil, fmt.Errorf("failed to walk project directory: %w", walkErr)
	}

	return seen, nil
}

// syncSingleFile processes one file during the filesystem walk.
func (s *WorkflowService) syncSingleFile(
	ctx context.Context,
	project *database.ProjectIndex,
	absPath, relPath string,
	d os.DirEntry,
	dbState *DBState,
	seen *SeenState,
) error {
	info, err := d.Info()
	if err != nil {
		s.log.WithError(err).WithField("path", relPath).Debug("Cannot stat file, skipping")
		return nil
	}

	// Check if it's a workflow (JSON with "nodes" array)
	if strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
		content, readErr := os.ReadFile(absPath)
		if readErr == nil && isWorkflowContent(content) {
			if err := s.syncWorkflowFile(ctx, project, absPath, relPath, content, dbState.WorkflowsByID, seen.WorkflowIDs); err != nil {
				s.log.WithError(err).WithField("path", relPath).Warn("Failed to sync workflow file")
			}
			return nil
		}
	}

	// Not a workflow, index as asset
	if err := s.syncAssetFile(ctx, project, relPath, info, dbState.AssetsByPath, seen.AssetPaths); err != nil {
		s.log.WithError(err).WithField("path", relPath).Debug("Failed to sync asset file")
	}
	return nil
}

// garbageCollect removes database records for files that no longer exist on disk.
// This handles: deleted files, moved files, renamed files.
func (s *WorkflowService) garbageCollect(ctx context.Context, dbState *DBState, seen *SeenState) error {
	// Remove stale workflows
	for id := range dbState.WorkflowsByID {
		if seen.WorkflowIDs[id] {
			continue
		}
		if err := s.repo.DeleteWorkflow(ctx, id); err != nil {
			s.log.WithError(err).WithField("workflow_id", id).Warn("Failed to delete stale workflow")
		} else {
			s.removeWorkflowPath(id)
		}
	}

	// Remove stale assets
	for filePath, asset := range dbState.AssetsByPath {
		if seen.AssetPaths[filePath] {
			continue
		}
		if err := s.repo.DeleteAsset(ctx, asset.ID); err != nil {
			s.log.WithError(err).WithField("asset_path", filePath).Debug("Failed to delete stale asset")
		}
	}

	return nil
}

// syncWorkflowFile processes a single workflow file during sync.
//
// Handles two workflow formats:
// - Native format: Has valid UUID in "id" field (our format)
// - External format: Other JSON structures (imported from other tools)
//
// External formats are converted IN-PLACE to native format. This is intentional:
// we want a single normalized format for consistency, and the conversion preserves
// the semantic content while enabling our tooling.
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
