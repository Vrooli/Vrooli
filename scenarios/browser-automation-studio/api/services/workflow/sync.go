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
