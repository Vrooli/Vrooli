package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless"
	"github.com/vrooli/browser-automation-studio/browserless/events"
	"github.com/vrooli/browser-automation-studio/database"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

const (
	workflowJSONStartMarker = "<WORKFLOW_JSON>"
	workflowJSONEndMarker   = "</WORKFLOW_JSON>"
)

var ErrWorkflowVersionConflict = errors.New("workflow version conflict")
var ErrWorkflowVersionNotFound = errors.New("workflow version not found")
var ErrWorkflowRestoreProjectMismatch = errors.New("workflow does not belong to a project")

// WorkflowVersionSummary captures version metadata alongside high-level definition statistics so
// the UI can render history timelines without rehydrating full workflow payloads on every row.
type WorkflowVersionSummary struct {
	Version           int              `json:"version"`
	WorkflowID        uuid.UUID        `json:"workflow_id"`
	CreatedAt         time.Time        `json:"created_at"`
	CreatedBy         string           `json:"created_by"`
	ChangeDescription string           `json:"change_description"`
	DefinitionHash    string           `json:"definition_hash"`
	NodeCount         int              `json:"node_count"`
	EdgeCount         int              `json:"edge_count"`
	FlowDefinition    database.JSONMap `json:"flow_definition"`
}

// WorkflowService handles workflow business logic
type WorkflowService struct {
	repo          database.Repository
	browserless   *browserless.Client
	wsHub         *wsHub.Hub
	log           *logrus.Logger
	aiClient      *OpenRouterClient
	syncLocks     sync.Map
	filePathCache sync.Map
}

// AIWorkflowError represents a structured error returned by the AI generator when
// it cannot produce a valid workflow definition for the given prompt.
type AIWorkflowError struct {
	Reason string
}

// Error implements the error interface.
func (e *AIWorkflowError) Error() string {
	return e.Reason
}

// WorkflowUpdateInput describes the mutable fields for a workflow save operation. The UI and CLI send both the
// JSON graph definition and an explicit nodes/edges payload; we keep both so agents can hand-edit the file without
// worrying about schema drift. ExpectedVersion enables optimistic locking so we do not clobber filesystem edits that
// were synchronized after the client loaded the workflow.
type WorkflowUpdateInput struct {
	Name              string
	Description       string
	FolderPath        string
	Tags              []string
	FlowDefinition    map[string]any
	Nodes             []any
	Edges             []any
	ChangeDescription string
	Source            string
	ExpectedVersion   *int
}

// ExecutionExportPreview summarises the export readiness state for an execution.
type ExecutionExportPreview struct {
	ExecutionID uuid.UUID            `json:"execution_id"`
	Status      string               `json:"status"`
	Message     string               `json:"message"`
	Package     *ReplayExportPackage `json:"package,omitempty"`
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(repo database.Repository, browserless *browserless.Client, wsHub *wsHub.Hub, log *logrus.Logger) *WorkflowService {
	return &WorkflowService{
		repo:        repo,
		browserless: browserless,
		wsHub:       wsHub,
		log:         log,
		aiClient:    NewOpenRouterClient(log),
	}
}

func cloneJSONMap(source database.JSONMap) database.JSONMap {
	if source == nil {
		return nil
	}
	clone := make(database.JSONMap, len(source))
	for k, v := range source {
		clone[k] = deepCloneInterface(v)
	}
	return clone
}

func deepCloneInterface(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for k, v := range typed {
			cloned[k] = deepCloneInterface(v)
		}
		return cloned
	case database.JSONMap:
		return cloneJSONMap(typed)
	case []any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = deepCloneInterface(typed[i])
		}
		return result
	case pq.StringArray:
		return append(pq.StringArray{}, typed...)
	default:
		return typed
	}
}

func workflowDefinitionStats(def database.JSONMap) (nodeCount, edgeCount int) {
	if def == nil {
		return 0, 0
	}
	nodes := toInterfaceSlice(def["nodes"])
	edges := toInterfaceSlice(def["edges"])
	return len(nodes), len(edges)
}

var previewDataKeys = map[string]struct{}{
	"previewScreenshot":          {},
	"previewScreenshotCapturedAt": {},
	"previewScreenshotSourceUrl":  {},
}

func stripPreviewData(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}
	modified := false
	cleaned := make(map[string]any, len(data))
	for key, value := range data {
		if _, skip := previewDataKeys[key]; skip {
			modified = true
			continue
		}
		cleaned[key] = value
	}
	if !modified {
		return data
	}
	return cleaned
}

func newWorkflowVersionSummary(version *database.WorkflowVersion) *WorkflowVersionSummary {
	if version == nil {
		return nil
	}
	definition := cloneJSONMap(version.FlowDefinition)
	hash := hashWorkflowDefinition(definition)
	nodes, edges := workflowDefinitionStats(definition)
	return &WorkflowVersionSummary{
		Version:           version.Version,
		WorkflowID:        version.WorkflowID,
		CreatedAt:         version.CreatedAt,
		CreatedBy:         version.CreatedBy,
		ChangeDescription: version.ChangeDescription,
		DefinitionHash:    hash,
		NodeCount:         nodes,
		EdgeCount:         edges,
		FlowDefinition:    definition,
	}
}

// CheckHealth checks the health of all dependencies
func (s *WorkflowService) CheckHealth() string {
	status := "healthy"

	// Check browserless health
	if err := s.browserless.CheckBrowserlessHealth(); err != nil {
		s.log.WithError(err).Warn("Browserless health check failed")
		status = "degraded"
	}

	return status
}

// Project methods

// CreateProject creates a new project
func (s *WorkflowService) CreateProject(ctx context.Context, project *database.Project) error {
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	return s.repo.CreateProject(ctx, project)
}

// GetProject gets a project by ID
func (s *WorkflowService) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return s.repo.GetProject(ctx, id)
}

// GetProjectByName gets a project by name
func (s *WorkflowService) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return s.repo.GetProjectByName(ctx, name)
}

// GetProjectByFolderPath gets a project by folder path
func (s *WorkflowService) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return s.repo.GetProjectByFolderPath(ctx, folderPath)
}

// UpdateProject updates a project
func (s *WorkflowService) UpdateProject(ctx context.Context, project *database.Project) error {
	project.UpdatedAt = time.Now()
	return s.repo.UpdateProject(ctx, project)
}

// DeleteProject deletes a project and all its workflows
func (s *WorkflowService) DeleteProject(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteProject(ctx, id)
}

// ListProjects lists all projects
func (s *WorkflowService) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return s.repo.ListProjects(ctx, limit, offset)
}

// GetProjectStats gets statistics for a project
func (s *WorkflowService) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return s.repo.GetProjectStats(ctx, projectID)
}

// ListWorkflowsByProject lists workflows for a specific project
func (s *WorkflowService) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	if err := s.syncProjectWorkflows(ctx, projectID); err != nil {
		return nil, err
	}
	workflows, err := s.repo.ListWorkflowsByProject(ctx, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	for _, wf := range workflows {
		if wf == nil {
			continue
		}
		if err := s.ensureWorkflowChangeMetadata(ctx, wf); err != nil {
			s.log.WithError(err).WithField("workflow_id", wf.ID).Warn("Failed to normalize workflow change metadata")
		}
	}
	return workflows, nil
}

// ListWorkflowVersions returns version metadata for a workflow ordered by newest first.
func (s *WorkflowService) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*WorkflowVersionSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows), errors.Is(err, database.ErrNotFound):
			return nil, database.ErrNotFound
		default:
			return nil, err
		}
	}

	versions, err := s.repo.ListWorkflowVersions(ctx, workflowID, limit, offset)
	if err != nil {
		return nil, err
	}

	results := make([]*WorkflowVersionSummary, 0, len(versions))
	for _, version := range versions {
		if version == nil {
			continue
		}
		summary := newWorkflowVersionSummary(version)
		if summary == nil {
			continue
		}
		// Carry forward latest metadata in case CreatedBy is blank in older rows.
		if strings.TrimSpace(summary.CreatedBy) == "" {
			summary.CreatedBy = firstNonEmpty(workflow.LastChangeSource, fileSourceManual)
		}
		results = append(results, summary)
	}

	return results, nil
}

// GetWorkflowVersion fetches a specific workflow revision with metadata suitable for diffing.
func (s *WorkflowService) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*WorkflowVersionSummary, error) {
	if version <= 0 {
		return nil, fmt.Errorf("invalid workflow version: %d", version)
	}

	if _, err := s.repo.GetWorkflow(ctx, workflowID); err != nil {
		return nil, err
	}

	workflowVersion, err := s.repo.GetWorkflowVersion(ctx, workflowID, version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows), errors.Is(err, database.ErrNotFound):
			return nil, ErrWorkflowVersionNotFound
		default:
			return nil, fmt.Errorf("%w: %v", ErrWorkflowVersionNotFound, err)
		}
	}

	return newWorkflowVersionSummary(workflowVersion), nil
}

// RestoreWorkflowVersion replays the historical definition into the active workflow and
// records a new version row while retaining current metadata (name, folder, tags).
func (s *WorkflowService) RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, description string) (*database.Workflow, error) {
	if version <= 0 {
		return nil, fmt.Errorf("invalid workflow version: %d", version)
	}

	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows), errors.Is(err, database.ErrNotFound):
			return nil, database.ErrNotFound
		default:
			return nil, err
		}
	}

	if workflow.ProjectID == nil {
		return nil, ErrWorkflowRestoreProjectMismatch
	}

	versionRow, err := s.repo.GetWorkflowVersion(ctx, workflowID, version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows), errors.Is(err, database.ErrNotFound):
			return nil, ErrWorkflowVersionNotFound
		default:
			return nil, fmt.Errorf("%w: %v", ErrWorkflowVersionNotFound, err)
		}
	}

	restoreDefinition := cloneJSONMap(versionRow.FlowDefinition)
	definitionMap := map[string]any{}
	for k, v := range restoreDefinition {
		definitionMap[k] = v
	}

	currentVersion := workflow.Version
	if strings.TrimSpace(description) == "" {
		description = fmt.Sprintf("Restored from version %d", version)
	}

	input := WorkflowUpdateInput{
		Name:              workflow.Name,
		Description:       workflow.Description,
		FolderPath:        workflow.FolderPath,
		Tags:              append([]string(nil), []string(workflow.Tags)...),
		FlowDefinition:    definitionMap,
		ChangeDescription: description,
		Source:            "version-restore",
		ExpectedVersion:   &currentVersion,
	}

	updated, err := s.UpdateWorkflow(ctx, workflowID, input)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// DeleteProjectWorkflows deletes a set of workflows within a project boundary
func (s *WorkflowService) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	if len(workflowIDs) == 0 {
		return nil
	}

	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	if err := s.syncProjectWorkflows(ctx, projectID); err != nil {
		return err
	}

	workflows := make(map[uuid.UUID]*database.Workflow, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		if wf, err := s.repo.GetWorkflow(ctx, workflowID); err == nil {
			workflows[workflowID] = wf
		}
	}

	if err := s.repo.DeleteProjectWorkflows(ctx, projectID, workflowIDs); err != nil {
		return err
	}

	for _, workflowID := range workflowIDs {
		entry, hasEntry := s.lookupWorkflowPath(workflowID)
		if hasEntry {
			if removeErr := os.Remove(entry.AbsolutePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				s.log.WithError(removeErr).WithField("path", entry.AbsolutePath).Warn("Failed to remove workflow file during deletion")
			}
			s.removeWorkflowPath(workflowID)
			continue
		}

		if wf, ok := workflows[workflowID]; ok {
			desiredAbs, _ := s.desiredWorkflowFilePath(project, wf)
			if removeErr := os.Remove(desiredAbs); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				s.log.WithError(removeErr).WithField("path", desiredAbs).Warn("Failed to remove inferred workflow file during deletion")
			}
		}
	}

	return nil
}

// Workflow methods

// CreateWorkflow creates a new workflow without a project. This now delegates to CreateWorkflowWithProject and will
// return an error because workflows must belong to a project to ensure filesystem synchronization.
func (s *WorkflowService) CreateWorkflow(ctx context.Context, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
	return s.CreateWorkflowWithProject(ctx, nil, name, folderPath, flowDefinition, aiPrompt)
}

// CreateWorkflowWithProject creates a new workflow with optional project association
func (s *WorkflowService) CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
	if projectID == nil {
		return nil, fmt.Errorf("workflows must belong to a project so they can be synchronized with the filesystem")
	}

	project, err := s.repo.GetProject(ctx, *projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project for workflow creation: %w", err)
	}

	now := time.Now().UTC()
	workflow := &database.Workflow{
		ID:                    uuid.New(),
		ProjectID:             projectID,
		Name:                  name,
		FolderPath:            folderPath,
		CreatedAt:             now,
		UpdatedAt:             now,
		Tags:                  []string{},
		Version:               1,
		LastChangeSource:      fileSourceManual,
		LastChangeDescription: "Initial workflow creation",
	}

	if aiPrompt != "" {
		generated, genErr := s.generateWorkflowFromPrompt(ctx, aiPrompt)
		if genErr != nil {
			return nil, fmt.Errorf("ai workflow generation failed: %w", genErr)
		}
		workflow.FlowDefinition = generated
	} else if flowDefinition != nil {
		workflow.FlowDefinition = database.JSONMap(flowDefinition)
	} else {
		workflow.FlowDefinition = database.JSONMap{
			"nodes": []any{},
			"edges": []any{},
		}
	}

	if err := s.repo.CreateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}

	version := &database.WorkflowVersion{
		WorkflowID:        workflow.ID,
		Version:           workflow.Version,
		FlowDefinition:    workflow.FlowDefinition,
		ChangeDescription: "Initial workflow creation",
		CreatedBy:         fileSourceManual,
	}
	if err := s.repo.CreateWorkflowVersion(ctx, version); err != nil {
		return nil, err
	}

	nodes := toInterfaceSlice(workflow.FlowDefinition["nodes"])
	edges := toInterfaceSlice(workflow.FlowDefinition["edges"])
	absPath, relPath, err := s.writeWorkflowFile(project, workflow, nodes, edges, "")
	if err != nil {
		return nil, err
	}
	s.cacheWorkflowPath(workflow.ID, absPath, relPath)

	return workflow, nil
}

// ListWorkflows lists workflows with optional filtering
func (s *WorkflowService) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	// When no specific folder is requested we eagerly synchronize every project so filesystem edits are reflected.
	if strings.TrimSpace(folderPath) == "" {
		projects, err := s.repo.ListProjects(ctx, 1000, 0)
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			if err := s.syncProjectWorkflows(ctx, project.ID); err != nil {
				s.log.WithError(err).WithField("project_id", project.ID).Warn("Failed to synchronize workflows before listing")
			}
		}
	}
	workflows, err := s.repo.ListWorkflows(ctx, folderPath, limit, offset)
	if err != nil {
		return nil, err
	}
	for _, wf := range workflows {
		if wf == nil {
			continue
		}
		if err := s.ensureWorkflowChangeMetadata(ctx, wf); err != nil {
			s.log.WithError(err).WithField("workflow_id", wf.ID).Warn("Failed to normalize workflow change metadata")
		}
	}
	return workflows, nil
}

// GetWorkflow gets a workflow by ID
func (s *WorkflowService) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	workflow, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}

	if workflow.ProjectID != nil {
		if err := s.syncProjectWorkflows(ctx, *workflow.ProjectID); err != nil {
			return nil, err
		}
		// Re-fetch in case synchronization updated the row or removed it.
		workflow, err = s.repo.GetWorkflow(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	if err := s.ensureWorkflowChangeMetadata(ctx, workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// UpdateWorkflow persists workflow edits originating from the UI, CLI, or filesystem-triggered autosave. The
// filesystem is the source of truth, so we synchronise before applying updates and increment the workflow version so
// executions can record the lineage they ran against.
func (s *WorkflowService) UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input WorkflowUpdateInput) (*database.Workflow, error) {
	current, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if current.ProjectID == nil {
		return nil, fmt.Errorf("workflow %s is not associated with a project; cannot update file-backed workflows", workflowID)
	}

	if err := s.syncProjectWorkflows(ctx, *current.ProjectID); err != nil {
		return nil, err
	}

	// Reload after sync in case the filesystem introduced a new revision.
	current, err = s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if input.ExpectedVersion == nil {
		sourceLabel := strings.TrimSpace(input.Source)
		if strings.EqualFold(sourceLabel, fileSourceAutosave) {
			expected := current.Version
			input.ExpectedVersion = &expected
			if s.log != nil {
				s.log.WithFields(logrus.Fields{
					"workflow_id": workflowID,
					"version":     current.Version,
				}).Warn("Autosave update missing expected version; defaulting to current version")
			}
		}
	}

	if input.ExpectedVersion != nil && current.Version != *input.ExpectedVersion {
		return nil, fmt.Errorf("%w: expected %d, found %d", ErrWorkflowVersionConflict, *input.ExpectedVersion, current.Version)
	}

	originalName := current.Name
	originalFolder := current.FolderPath
	s.coalesceWorkflowChangeMetadata(current)
	existingFlowHash := hashWorkflowDefinition(current.FlowDefinition)

	metadataChanged := false

	if name := strings.TrimSpace(input.Name); name != "" && name != current.Name {
		current.Name = name
		metadataChanged = true
	}

	trimmedDescription := strings.TrimSpace(input.Description)
	if trimmedDescription != current.Description {
		current.Description = trimmedDescription
		metadataChanged = true
	}

	if folder := strings.TrimSpace(input.FolderPath); folder != "" {
		normalizedFolder := normalizeFolderPath(folder)
		if normalizedFolder != current.FolderPath {
			current.FolderPath = normalizedFolder
			metadataChanged = true
		}
	}

	if input.Tags != nil {
		tags := pq.StringArray(input.Tags)
		if !equalStringArrays(current.Tags, tags) {
			current.Tags = tags
			metadataChanged = true
		}
	}

	flowPayloadProvided := input.FlowDefinition != nil || len(input.Nodes) > 0 || len(input.Edges) > 0
	flowChanged := false
	updatedDefinition := current.FlowDefinition

	if flowPayloadProvided {
		definitionCandidate := input.FlowDefinition
		if definitionCandidate == nil {
			definitionCandidate = map[string]any{}
		}
		if _, ok := definitionCandidate["nodes"]; !ok && len(input.Nodes) > 0 {
			definitionCandidate["nodes"] = input.Nodes
		}
		if _, ok := definitionCandidate["edges"]; !ok && len(input.Edges) > 0 {
			definitionCandidate["edges"] = input.Edges
		}

		normalized, normErr := normalizeFlowDefinition(definitionCandidate)
		if normErr != nil {
			return nil, fmt.Errorf("invalid workflow definition: %w", normErr)
		}

		updatedDefinition = database.JSONMap(normalized)
		incomingHash := hashWorkflowDefinition(updatedDefinition)
		if incomingHash != existingFlowHash {
			flowChanged = true
		}
	}

	if !metadataChanged && !flowChanged {
		return current, nil
	}

	if flowPayloadProvided {
		current.FlowDefinition = updatedDefinition
	}

	source := firstNonEmpty(strings.TrimSpace(input.Source), fileSourceManual)

	changeDesc := strings.TrimSpace(input.ChangeDescription)
	if changeDesc == "" {
		switch {
		case flowChanged && metadataChanged:
			changeDesc = "Flow and metadata update"
		case flowChanged:
			if strings.EqualFold(source, fileSourceAutosave) {
				changeDesc = "Autosave"
			} else {
				changeDesc = "Workflow updated"
			}
		default:
			changeDesc = "Metadata update"
		}
	}

	current.LastChangeSource = source
	current.LastChangeDescription = changeDesc
	current.Version++
	current.UpdatedAt = time.Now().UTC()
	if current.CreatedAt.IsZero() {
		current.CreatedAt = current.UpdatedAt
	}

	if err := s.repo.UpdateWorkflow(ctx, current); err != nil {
		return nil, err
	}

	version := &database.WorkflowVersion{
		WorkflowID:        current.ID,
		Version:           current.Version,
		FlowDefinition:    current.FlowDefinition,
		ChangeDescription: changeDesc,
		CreatedBy:         source,
	}
	if err := s.repo.CreateWorkflowVersion(ctx, version); err != nil {
		return nil, err
	}

	project, err := s.repo.GetProject(ctx, *current.ProjectID)
	if err != nil {
		return nil, err
	}

	nodes := toInterfaceSlice(current.FlowDefinition["nodes"])
	edges := toInterfaceSlice(current.FlowDefinition["edges"])

	cacheEntry, hasEntry := s.lookupWorkflowPath(current.ID)
	preferredPath := ""
	if hasEntry && originalName == current.Name && originalFolder == current.FolderPath {
		preferredPath = cacheEntry.RelativePath
	}

	absPath, relPath, err := s.writeWorkflowFile(project, current, nodes, edges, preferredPath)
	if err != nil {
		return nil, err
	}
	s.cacheWorkflowPath(current.ID, absPath, relPath)

	if hasEntry && preferredPath == "" && cacheEntry.RelativePath != "" && cacheEntry.RelativePath != relPath {
		if removeErr := os.Remove(cacheEntry.AbsolutePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			s.log.WithError(removeErr).WithField("path", cacheEntry.AbsolutePath).Warn("Failed to remove stale workflow file after rename")
		}
	}

	return current, nil
}

func (s *WorkflowService) ensureWorkflowChangeMetadata(ctx context.Context, workflow *database.Workflow) error {
	if workflow == nil {
		return nil
	}
	if !s.coalesceWorkflowChangeMetadata(workflow) {
		return nil
	}
	return s.repo.UpdateWorkflow(ctx, workflow)
}

func (s *WorkflowService) coalesceWorkflowChangeMetadata(workflow *database.Workflow) bool {
	if workflow == nil {
		return false
	}
	changed := false
	if strings.TrimSpace(workflow.LastChangeSource) == "" {
		workflow.LastChangeSource = fileSourceManual
		changed = true
	}
	if strings.TrimSpace(workflow.LastChangeDescription) == "" {
		workflow.LastChangeDescription = "Manual save"
		changed = true
	}
	return changed
}

// ExecuteWorkflow executes a workflow
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	// Verify workflow exists
	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if workflow.ProjectID != nil {
		if err := s.syncProjectWorkflows(ctx, *workflow.ProjectID); err != nil {
			return nil, err
		}
		workflow, err = s.repo.GetWorkflow(ctx, workflowID)
		if err != nil {
			return nil, err
		}
	}

	if err := s.ensureWorkflowChangeMetadata(ctx, workflow); err != nil {
		return nil, err
	}

	execution := &database.Execution{
		ID:              uuid.New(),
		WorkflowID:      workflowID,
		WorkflowVersion: workflow.Version,
		Status:          "pending",
		TriggerType:     "manual",
		Parameters:      database.JSONMap(parameters),
		StartedAt:       time.Now(),
		Progress:        0,
		CurrentStep:     "Initializing",
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, err
	}

	// Start async execution
	go s.executeWorkflowAsync(execution, workflow)

	return execution, nil
}

// DescribeExecutionExport returns the current replay export status for an execution.
func (s *WorkflowService) DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*ExecutionExportPreview, error) {
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, err
	}

	var workflow *database.Workflow
	if wf, wfErr := s.repo.GetWorkflow(ctx, execution.WorkflowID); wfErr == nil {
		workflow = wf
	} else if !errors.Is(wfErr, database.ErrNotFound) {
		return nil, wfErr
	}

	timeline, err := s.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		return nil, err
	}

	exportPackage, err := BuildReplayExport(execution, workflow, timeline)
	if err != nil {
		return nil, err
	}

	frameCount := exportPackage.Summary.FrameCount
	if frameCount == 0 {
		frameCount = len(timeline.Frames)
	}
	message := fmt.Sprintf("Replay export ready (%d frames, %dms)", frameCount, exportPackage.Summary.TotalDurationMs)

	return &ExecutionExportPreview{
		ExecutionID: execution.ID,
		Status:      "ready",
		Message:     message,
		Package:     exportPackage,
	}, nil
}

// GetExecutionScreenshots gets screenshots for an execution
func (s *WorkflowService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return s.repo.GetExecutionScreenshots(ctx, executionID)
}

// GetExecution gets an execution by ID
func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return s.repo.GetExecution(ctx, id)
}

// ListExecutions lists executions with optional workflow filtering
func (s *WorkflowService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return s.repo.ListExecutions(ctx, workflowID, limit, offset)
}

// StopExecution stops a running execution
func (s *WorkflowService) StopExecution(ctx context.Context, executionID uuid.UUID) error {
	// Get current execution
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return err
	}

	// Only stop if currently running
	if execution.Status != "running" && execution.Status != "pending" {
		return nil // Already stopped/completed
	}

	// Update execution status
	execution.Status = "cancelled"
	now := time.Now()
	execution.CompletedAt = &now

	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		return err
	}

	// Log the cancellation
	logEntry := &database.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       "info",
		StepName:    "execution_cancelled",
		Message:     "Execution cancelled by user request",
	}
	s.repo.CreateExecutionLog(ctx, logEntry)

	// Broadcast cancellation
	s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
		Type:        "cancelled",
		ExecutionID: execution.ID,
		Status:      "cancelled",
		Progress:    execution.Progress,
		CurrentStep: execution.CurrentStep,
		Message:     "Execution cancelled by user",
	})

	s.log.WithField("execution_id", executionID).Info("Execution stopped by user request")
	return nil
}

// generateWorkflowFromPrompt generates a workflow definition via OpenRouter.
func (s *WorkflowService) generateWorkflowFromPrompt(ctx context.Context, prompt string) (database.JSONMap, error) {
	if s.aiClient == nil {
		return nil, errors.New("openrouter client not configured")
	}

	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return nil, errors.New("empty AI prompt")
	}

	instruction := fmt.Sprintf(`You are a strict JSON generator. Produce exactly one JSON object with the following structure:
{
  "nodes": [
    {
      "id": "node-1",
      "type": "navigate" | "click" | "type" | "wait" | "screenshot" | "extract" | "workflowCall",
      "position": {"x": <number>, "y": <number>},
      "data": { ... } // include all parameters needed for the step (url, selector, text, waitMs, etc.)
    }
  ],
  "edges": [
    {"id": "edge-1", "source": "node-1", "target": "node-2"}
  ]
}

Rules:
1. Tailor every node and selector to the user's request. Never use placeholders such as "https://example.com" or generic selectors.
2. Provide only the fields needed to execute the step (e.g., url, selector, text, waitMs, timeoutMs, screenshot name). Keep the response concise.
3. Arrange nodes with sensible coordinates (e.g., x increments by ~180 horizontally, y by ~120 vertically for branches).
4. Include necessary wait/ensure steps before interactions to make the automation reliable. Use the "wait" type for waits/ensure conditions.
5. Valid node types are limited to: navigate, click, type, wait, screenshot, extract, workflowCall. Do not invent new types.
6. Wrap the JSON in markers exactly like this: <WORKFLOW_JSON>{...}</WORKFLOW_JSON>.
7. The response MUST start with '<WORKFLOW_JSON>{' and end with '}</WORKFLOW_JSON>'. Output minified JSON on a single line (no spaces or newlines) and keep it under 1200 characters in total.
8. If you cannot produce a valid workflow, respond with <WORKFLOW_JSON>{"error":"reason"}</WORKFLOW_JSON>.

User prompt:
%s`, trimmed)

	start := time.Now()
	response, err := s.aiClient.ExecutePrompt(ctx, instruction)
	if err != nil {
		return nil, fmt.Errorf("openrouter execution error: %w", err)
	}
	s.log.WithFields(logrus.Fields{
		"model":            s.aiClient.model,
		"duration_ms":      time.Since(start).Milliseconds(),
		"response_preview": truncateForLog(response, 400),
	}).Info("AI workflow generated via OpenRouter")

	cleaned := extractJSONObject(stripJSONCodeFence(response))

	var payload map[string]any
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"raw_response": truncateForLog(response, 2000),
			"cleaned":      truncateForLog(cleaned, 2000),
		}).Error("Failed to parse workflow JSON returned by OpenRouter")
		return nil, fmt.Errorf("failed to parse OpenRouter JSON: %w", err)
	}

	if err := detectAIWorkflowError(payload); err != nil {
		return nil, err
	}

	definition, err := normalizeFlowDefinition(payload)
	if err != nil {
		return nil, err
	}

	if len(toInterfaceSlice(definition["nodes"])) == 0 {
		return nil, &AIWorkflowError{Reason: "AI workflow generation did not return any steps. Specify real URLs, selectors, and actions, then try again."}
	}

	return definition, nil
}

func stripJSONCodeFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```")
		if idx := strings.Index(trimmed, "\n"); idx != -1 {
			trimmed = trimmed[idx+1:]
		}
		if idx := strings.LastIndex(trimmed, "```"); idx != -1 {
			trimmed = trimmed[:idx]
		}
	}
	return strings.TrimSpace(trimmed)
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end < start {
		return strings.TrimSpace(raw)
	}
	return strings.TrimSpace(raw[start : end+1])
}

func normalizeFlowDefinition(payload map[string]any) (database.JSONMap, error) {
	candidate := payload
	if workflow, ok := payload["workflow"].(map[string]any); ok {
		candidate = workflow
	}

	if err := detectAIWorkflowError(candidate); err != nil {
		return nil, err
	}

	rawNodes, ok := candidate["nodes"]
	if !ok {
		if steps, ok := candidate["steps"].([]any); ok {
			candidate["nodes"] = steps
			rawNodes = steps
		}
	}

	nodes, ok := rawNodes.([]any)
	if !ok || nodes == nil {
		nodes = []any{}
	}

	for i, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok {
			return nil, errors.New("AI node payload is not an object")
		}
		if _, ok := node["id"].(string); !ok {
			node["id"] = fmt.Sprintf("node-%d", i+1)
		}
		if _, ok := node["position"].(map[string]any); !ok {
			node["position"] = map[string]any{
				"x": float64(100 + i*200),
				"y": float64(100 + i*120),
			}
		}
		if dataMap, ok := node["data"].(map[string]any); ok {
			node["data"] = stripPreviewData(dataMap)
		}
		nodes[i] = node
	}
	candidate["nodes"] = nodes

	edgesRaw, hasEdges := candidate["edges"]
	edges, _ := edgesRaw.([]any)
	if !hasEdges || edges == nil {
		edges = []any{}
	}

	if len(edges) == 0 && len(nodes) > 1 {
		edges = make([]any, 0, len(nodes)-1)
		for i := 0; i < len(nodes)-1; i++ {
			source := nodes[i].(map[string]any)["id"].(string)
			target := nodes[i+1].(map[string]any)["id"].(string)
			edges = append(edges, map[string]any{
				"id":     fmt.Sprintf("edge-%d", i+1),
				"source": source,
				"target": target,
			})
		}
	}
	candidate["edges"] = edges

	return database.JSONMap(candidate), nil
}

func detectAIWorkflowError(payload map[string]any) error {
	if reason := extractAIErrorMessage(payload); reason != "" {
		return &AIWorkflowError{Reason: reason}
	}
	return nil
}

func extractAIErrorMessage(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if msg, ok := typed["error"].(string); ok {
			trimmed := strings.TrimSpace(msg)
			if trimmed != "" {
				return trimmed
			}
		}
		if msg, ok := typed["message"].(string); ok {
			trimmed := strings.TrimSpace(msg)
			if trimmed != "" {
				return trimmed
			}
		}
		if workflow, ok := typed["workflow"].(map[string]any); ok {
			if nested := extractAIErrorMessage(workflow); nested != "" {
				return nested
			}
		}
		for _, nested := range typed {
			if nestedMsg := extractAIErrorMessage(nested); nestedMsg != "" {
				return nestedMsg
			}
		}
	}
	return ""
}

func defaultWorkflowDefinition() database.JSONMap {
	return database.JSONMap{
		"nodes": []any{
			map[string]any{
				"id":   "node-1",
				"type": "navigate",
				"position": map[string]any{
					"x": float64(100),
					"y": float64(100),
				},
				"data": map[string]any{
					"url": "https://example.com",
				},
			},
			map[string]any{
				"id":   "node-2",
				"type": "screenshot",
				"position": map[string]any{
					"x": float64(350),
					"y": float64(220),
				},
				"data": map[string]any{
					"name": "homepage",
				},
			},
		},
		"edges": []any{
			map[string]any{
				"id":     "edge-1",
				"source": "node-1",
				"target": "node-2",
			},
		},
	}
}

// ModifyWorkflow applies an AI-driven modification to an existing workflow.
func (s *WorkflowService) ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, modificationPrompt string, overrideFlow map[string]any) (*database.Workflow, error) {
	if s.aiClient == nil {
		return nil, errors.New("openrouter client not configured")
	}
	if strings.TrimSpace(modificationPrompt) == "" {
		return nil, errors.New("modification prompt is required")
	}

	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	baseFlow := map[string]any{}
	if overrideFlow != nil {
		baseFlow = overrideFlow
	} else if workflow.FlowDefinition != nil {
		bytes, err := json.Marshal(workflow.FlowDefinition)
		if err == nil {
			_ = json.Unmarshal(bytes, &baseFlow)
		}
	}

	baseFlowJSON, err := json.MarshalIndent(baseFlow, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal base workflow: %w", err)
	}

	instruction := fmt.Sprintf(`You are a strict JSON generator. Update the existing workflow so it satisfies the user's request.

Rules:
1. Respond with a single JSON object that uses the same schema as the original workflow ("nodes" array + "edges" array).
2. Preserve existing node IDs when the step remains applicable. Modify node types/data/positions only where necessary, and keep data concise (only the parameters required to execute the step).
3. Keep the graph valid: edges must describe a reachable execution path.
4. Fill in realistic selectors, URLs, filenames, waits, etc.—no placeholders. Only use the allowed node types: navigate, click, type, wait, screenshot, extract, workflowCall.
5. Wrap the JSON in markers exactly like this: <WORKFLOW_JSON>{...}</WORKFLOW_JSON>.
6. The response MUST start with '<WORKFLOW_JSON>{' and end with '}</WORKFLOW_JSON>'. Output minified JSON on a single line (no spaces or newlines) and keep it shorter than 1200 characters.
7. If the request cannot be satisfied, respond with <WORKFLOW_JSON>{"error":"reason"}</WORKFLOW_JSON>.

Original workflow JSON:
%s

User requested modifications:
%s`, string(baseFlowJSON), strings.TrimSpace(modificationPrompt))

	start := time.Now()
	response, err := s.aiClient.ExecutePrompt(ctx, instruction)
	if err != nil {
		return nil, fmt.Errorf("openrouter execution error: %w", err)
	}
	s.log.WithFields(logrus.Fields{
		"model":       s.aiClient.model,
		"duration_ms": time.Since(start).Milliseconds(),
		"workflow_id": workflowID,
	}).Info("AI workflow modification generated via OpenRouter")

	cleaned := extractJSONObject(stripJSONCodeFence(response))

	var payload map[string]any
	if err := json.Unmarshal([]byte(cleaned), &payload); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"raw_response": truncateForLog(response, 2000),
			"cleaned":      truncateForLog(cleaned, 2000),
			"workflow_id":  workflowID,
		}).Error("Failed to parse modified workflow JSON returned by OpenRouter")
		return nil, fmt.Errorf("failed to parse OpenRouter JSON: %w", err)
	}

	definition, err := normalizeFlowDefinition(payload)
	if err != nil {
		return nil, err
	}

	if len(toInterfaceSlice(definition["nodes"])) == 0 {
		return nil, &AIWorkflowError{Reason: "AI workflow generation did not return any steps. Specify real URLs, selectors, and actions, then try again."}
	}

	workflow.FlowDefinition = definition
	workflow.Version++
	workflow.UpdatedAt = time.Now()

	if err := s.repo.UpdateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// executeWorkflowAsync executes a workflow asynchronously
func (s *WorkflowService) executeWorkflowAsync(execution *database.Execution, workflow *database.Workflow) {
	ctx := context.Background()
	emitter := events.NewEmitter(s.wsHub, s.log)
	s.log.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"workflow_id":  execution.WorkflowID,
	}).Info("Starting async workflow execution")

	// Update status to running and broadcast
	execution.Status = "running"
	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		s.log.WithError(err).Error("Failed to update execution status to running")
		return
	}

	// Broadcast execution started
	s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
		Type:        "progress",
		ExecutionID: execution.ID,
		Status:      "running",
		Progress:    0,
		CurrentStep: "Initializing",
		Message:     "Workflow execution started",
	})

	if emitter != nil {
		emitter.Emit(events.NewEvent(
			events.EventExecutionStarted,
			execution.ID,
			execution.WorkflowID,
			events.WithStatus("running"),
			events.WithMessage("Workflow execution started"),
			events.WithProgress(0),
		))
	}

	// Use browserless client to execute the workflow
	if err := s.browserless.ExecuteWorkflow(ctx, execution, workflow, emitter); err != nil {
		s.log.WithError(err).Error("Workflow execution failed")

		// Mark execution as failed
		execution.Status = "failed"
		execution.Error = err.Error()
		now := time.Now()
		execution.CompletedAt = &now

		// Log the error
		logEntry := &database.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       "error",
			StepName:    "execution_failed",
			Message:     "Workflow execution failed: " + err.Error(),
		}
		s.repo.CreateExecutionLog(ctx, logEntry)

		// Broadcast failure
		s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
			Type:        "failed",
			ExecutionID: execution.ID,
			Status:      "failed",
			Progress:    execution.Progress,
			CurrentStep: execution.CurrentStep,
			Message:     "Workflow execution failed: " + err.Error(),
		})

		if emitter != nil {
			emitter.Emit(events.NewEvent(
				events.EventExecutionFailed,
				execution.ID,
				execution.WorkflowID,
				events.WithStatus("failed"),
				events.WithMessage(err.Error()),
				events.WithProgress(execution.Progress),
				events.WithPayload(map[string]any{
					"current_step": execution.CurrentStep,
					"error":        err.Error(),
				}),
			))
		}
	} else {
		// Mark execution as completed
		execution.Status = "completed"
		execution.Progress = 100
		execution.CurrentStep = "Completed"
		now := time.Now()
		execution.CompletedAt = &now
		execution.Result = database.JSONMap{
			"success": true,
			"message": "Workflow completed successfully",
		}

		s.log.WithField("execution_id", execution.ID).Info("Workflow execution completed successfully")

		// Broadcast completion
		s.wsHub.BroadcastUpdate(wsHub.ExecutionUpdate{
			Type:        "completed",
			ExecutionID: execution.ID,
			Status:      "completed",
			Progress:    100,
			CurrentStep: "Completed",
			Message:     "Workflow completed successfully",
			Data: map[string]any{
				"success": true,
				"result":  execution.Result,
			},
		})

		if emitter != nil {
			emitter.Emit(events.NewEvent(
				events.EventExecutionCompleted,
				execution.ID,
				execution.WorkflowID,
				events.WithStatus("completed"),
				events.WithProgress(100),
				events.WithMessage("Workflow completed successfully"),
				events.WithPayload(map[string]any{
					"result": execution.Result,
				}),
			))
		}
	}

	// Final update
	if err := s.repo.UpdateExecution(ctx, execution); err != nil {
		s.log.WithError(err).Error("Failed to update final execution status")
	}
}
