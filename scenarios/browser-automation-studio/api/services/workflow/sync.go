package workflow

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
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

func (s *WorkflowService) syncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	lock := s.getProjectLock(project.ID)
	lock.Lock()
	defer lock.Unlock()

	// Phase 1: Discover and convert external workflows first.
	// This ensures any workflow JSON files outside workflows/ directory get imported.
	if err := s.importExternalWorkflows(ctx, project); err != nil {
		s.log.WithError(err).WithField("project_id", projectID).Warn("Failed to import external workflows")
		// Continue with native sync even if external import fails
	}

	// Phase 2: Sync native workflows from workflows/ directory (existing logic).
	return s.syncNativeWorkflows(ctx, project)
}

// importExternalWorkflows discovers and converts external workflow files to native format.
func (s *WorkflowService) importExternalWorkflows(ctx context.Context, project *database.ProjectIndex) error {
	_ = ctx // context reserved for future use

	// Discover external workflows (not in workflows/ directory)
	external, err := DiscoverExternalWorkflows(project, 4)
	if err != nil {
		return fmt.Errorf("discover external workflows: %w", err)
	}

	if len(external) == 0 {
		return nil
	}

	// Load import manifest to check which files have already been imported
	manifest, err := LoadImportManifest(project)
	if err != nil {
		s.log.WithError(err).Warn("Failed to load import manifest, starting fresh")
		manifest = &ImportManifest{Entries: make(map[string]ImportedEntry)}
	}

	manifestUpdated := false

	for _, d := range external {
		// Skip if already imported and unchanged
		content, err := os.ReadFile(d.AbsolutePath)
		if err != nil {
			s.log.WithError(err).WithField("path", d.RelativePath).Debug("Cannot read external workflow file")
			continue
		}

		if manifest.IsImported(d.RelativePath) && !manifest.HasChanged(d.RelativePath, content) {
			continue
		}

		// Convert external workflow to native format
		result, err := ConvertExternalWorkflow(project, content, d.RelativePath)
		if err != nil {
			s.log.WithError(err).WithField("path", d.RelativePath).Warn("Failed to convert external workflow")
			continue
		}

		// Check for name conflicts before writing
		conflicting, _ := s.repo.GetWorkflowByName(ctx, result.Workflow.Name, result.Workflow.FolderPath)
		if conflicting != nil {
			// If we previously imported this file to this workflow, update it
			if existingID, ok := manifest.GetWorkflowIDBySource(d.RelativePath); ok && existingID == conflicting.ID {
				result.Workflow.Id = conflicting.ID.String()
			} else {
				s.log.WithFields(logrus.Fields{
					"source_path":   d.RelativePath,
					"workflow_name": result.Workflow.Name,
					"conflict_id":   conflicting.ID,
				}).Warn("Skipping external workflow that conflicts with existing name/folder_path")
				continue
			}
		}

		// Write converted workflow to workflows/ directory
		absPath, relPath, err := WriteConvertedWorkflow(project, result)
		if err != nil {
			s.log.WithError(err).WithField("path", d.RelativePath).Warn("Failed to write converted workflow")
			continue
		}

		// Record in manifest
		workflowID, _ := uuid.Parse(result.Workflow.Id)
		manifest.RecordImport(d.RelativePath, workflowID, relPath, content)
		manifestUpdated = true

		s.log.WithFields(logrus.Fields{
			"source":      d.RelativePath,
			"destination": relPath,
			"workflow_id": workflowID,
		}).Info("Imported external workflow")

		s.cacheWorkflowPath(workflowID, absPath, relPath)
	}

	// Save manifest if updated
	if manifestUpdated {
		if err := SaveImportManifest(project, manifest); err != nil {
			s.log.WithError(err).Warn("Failed to save import manifest")
		}
	}

	return nil
}

// syncNativeWorkflows synchronizes native workflows from the workflows/ directory.
func (s *WorkflowService) syncNativeWorkflows(ctx context.Context, project *database.ProjectIndex) error {
	workflowsRoot := ProjectWorkflowsDir(project)

	// Discover file snapshots (only if workflows directory exists).
	snapshots := make(map[uuid.UUID]*workflowProtoSnapshot)
	var discoveryErr error
	if info, statErr := os.Stat(workflowsRoot); statErr == nil && info.IsDir() {
		err := filepath.WalkDir(workflowsRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), workflowFileExt) {
				return nil
			}
			snapshot, readErr := ReadWorkflowSummaryFile(ctx, project, path)
			if readErr != nil {
				discoveryErr = readErr
				return readErr
			}
			if snapshot.Workflow == nil || strings.TrimSpace(snapshot.Workflow.Id) == "" {
				return nil
			}
			id, parseErr := uuid.Parse(snapshot.Workflow.Id)
			if parseErr != nil {
				return fmt.Errorf("workflow file %s has invalid id: %w", snapshot.RelativePath, parseErr)
			}
			snapshots[id] = snapshot
			s.cacheWorkflowPath(id, snapshot.AbsolutePath, snapshot.RelativePath)
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to scan workflow directory: %w", err)
		}
		if discoveryErr != nil {
			return discoveryErr
		}
	}

	dbWorkflows, err := s.listAllProjectWorkflows(ctx, project.ID)
	if err != nil {
		return err
	}

	dbByID := make(map[uuid.UUID]*database.WorkflowIndex, len(dbWorkflows))
	for _, wf := range dbWorkflows {
		dbByID[wf.ID] = wf
	}

	processed := make(map[uuid.UUID]bool)

	for id, snapshot := range snapshots {
		processed[id] = true
		fileWF := snapshot.Workflow
		existing, exists := dbByID[id]
		if !exists {
			// Check if a workflow with the same name/folder_path already exists
			// This can happen if the ID in the file doesn't match the DB record
			conflictingWF, conflictErr := s.repo.GetWorkflowByName(ctx, fileWF.Name, fileWF.FolderPath)
			if conflictErr == nil && conflictingWF != nil {
				// Skip this file - there's already a workflow with this name/folder
				s.log.WithFields(logrus.Fields{
					"file_id":       id,
					"file_name":     fileWF.Name,
					"file_path":     snapshot.RelativePath,
					"existing_id":   conflictingWF.ID,
					"existing_name": conflictingWF.Name,
				}).Warn("Skipping workflow file that conflicts with existing workflow name/folder_path")
				continue
			}

			// Create new workflow index from file.
			index := &database.WorkflowIndex{
				ID:         id,
				ProjectID:  &project.ID,
				Name:       fileWF.Name,
				FolderPath: normalizeFolderPath(fileWF.FolderPath),
				FilePath:   snapshot.RelativePath,
				Version:    int(fileWF.Version),
			}
			if err := s.repo.CreateWorkflow(ctx, index); err != nil {
				return fmt.Errorf("failed to create workflow from file %s: %w", snapshot.RelativePath, err)
			}
			if snapshot.NeedsWrite {
				if _, _, err := WriteWorkflowSummaryFile(project, fileWF, snapshot.RelativePath); err != nil {
					return err
				}
			}
			continue
		}

		needsUpdate := false
		if existing.Name != fileWF.Name {
			existing.Name = fileWF.Name
			needsUpdate = true
		}
		if normalizeFolderPath(existing.FolderPath) != normalizeFolderPath(fileWF.FolderPath) {
			existing.FolderPath = normalizeFolderPath(fileWF.FolderPath)
			needsUpdate = true
		}
		if filepath.ToSlash(strings.TrimSpace(existing.FilePath)) != filepath.ToSlash(strings.TrimSpace(snapshot.RelativePath)) {
			existing.FilePath = snapshot.RelativePath
			needsUpdate = true
		}
		if existing.Version != int(fileWF.Version) && fileWF.Version > 0 {
			existing.Version = int(fileWF.Version)
			needsUpdate = true
		}

		if needsUpdate {
			if err := s.repo.UpdateWorkflow(ctx, existing); err != nil {
				return fmt.Errorf("failed to update workflow %s from file: %w", existing.ID, err)
			}
			if snapshot.NeedsWrite {
				if _, _, err := WriteWorkflowSummaryFile(project, fileWF, snapshot.RelativePath); err != nil {
					return err
				}
			}
			continue
		}

		if snapshot.NeedsWrite {
			if _, _, err := WriteWorkflowSummaryFile(project, fileWF, snapshot.RelativePath); err != nil {
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

// SyncProjectWorkflows synchronizes the workflow DB index for a project from the filesystem.
// The filesystem is the source of truth; the DB is an index.
func (s *WorkflowService) SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error {
	return s.syncProjectWorkflows(ctx, projectID)
}
