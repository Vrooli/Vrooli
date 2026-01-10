package handlers

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// MockRepository is an in-memory implementation of database.Repository for testing.
// All methods are thread-safe and maintain data consistency.
type MockRepository struct {
	mu sync.RWMutex

	projects   map[uuid.UUID]*database.ProjectIndex
	workflows  map[uuid.UUID]*database.WorkflowIndex
	executions map[uuid.UUID]*database.ExecutionIndex
	schedules  map[uuid.UUID]*database.ScheduleIndex
	exports    map[uuid.UUID]*database.ExportIndex
	settings   map[string]string

	// Error injection for testing error paths
	ListProjectsError error
}

// NewMockRepository creates a new in-memory mock repository.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		projects:   make(map[uuid.UUID]*database.ProjectIndex),
		workflows:  make(map[uuid.UUID]*database.WorkflowIndex),
		executions: make(map[uuid.UUID]*database.ExecutionIndex),
		schedules:  make(map[uuid.UUID]*database.ScheduleIndex),
		exports:    make(map[uuid.UUID]*database.ExportIndex),
		settings:   make(map[string]string),
	}
}

// Compile-time interface check
var _ database.Repository = (*MockRepository)(nil)

// ============================================================================
// Project Operations
// ============================================================================

func (r *MockRepository) CreateProject(_ context.Context, project *database.ProjectIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now

	// Make a copy to avoid external mutations
	copy := *project
	r.projects[project.ID] = &copy
	return nil
}

func (r *MockRepository) GetProject(_ context.Context, id uuid.UUID) (*database.ProjectIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, ok := r.projects[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	copy := *project
	return &copy, nil
}

func (r *MockRepository) GetProjectByName(_ context.Context, name string) (*database.ProjectIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, project := range r.projects {
		if project.Name == name {
			copy := *project
			return &copy, nil
		}
	}
	return nil, database.ErrNotFound
}

func (r *MockRepository) GetProjectByFolderPath(_ context.Context, folderPath string) (*database.ProjectIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, project := range r.projects {
		if project.FolderPath == folderPath {
			copy := *project
			return &copy, nil
		}
	}
	return nil, database.ErrNotFound
}

func (r *MockRepository) UpdateProject(_ context.Context, project *database.ProjectIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.projects[project.ID]; !ok {
		return database.ErrNotFound
	}
	project.UpdatedAt = time.Now()
	copy := *project
	r.projects[project.ID] = &copy
	return nil
}

func (r *MockRepository) DeleteProject(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.projects, id)
	return nil
}

func (r *MockRepository) ListProjects(_ context.Context, limit, offset int) ([]*database.ProjectIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.ListProjectsError != nil {
		return nil, r.ListProjectsError
	}

	// Collect all projects
	projects := make([]*database.ProjectIndex, 0, len(r.projects))
	for _, p := range r.projects {
		copy := *p
		projects = append(projects, &copy)
	}

	// Sort by updated_at DESC
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].UpdatedAt.After(projects[j].UpdatedAt)
	})

	// Apply pagination
	return applyPagination(projects, limit, offset), nil
}

func (r *MockRepository) GetProjectStats(_ context.Context, projectID uuid.UUID) (*database.ProjectStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &database.ProjectStats{ProjectID: projectID}

	// Count workflows
	for _, w := range r.workflows {
		if w.ProjectID != nil && *w.ProjectID == projectID {
			stats.WorkflowCount++

			// Count executions for this workflow
			for _, e := range r.executions {
				if e.WorkflowID == w.ID {
					stats.ExecutionCount++
					if stats.LastExecution == nil || e.StartedAt.After(*stats.LastExecution) {
						t := e.StartedAt
						stats.LastExecution = &t
					}
				}
			}
		}
	}

	return stats, nil
}

func (r *MockRepository) GetProjectsStats(_ context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[uuid.UUID]*database.ProjectStats, len(projectIDs))
	for _, id := range projectIDs {
		result[id] = &database.ProjectStats{ProjectID: id}
	}

	// Build workflow map per project
	for _, w := range r.workflows {
		if w.ProjectID == nil {
			continue
		}
		if stats, ok := result[*w.ProjectID]; ok {
			stats.WorkflowCount++
		}
	}

	// Count executions
	for _, e := range r.executions {
		w, ok := r.workflows[e.WorkflowID]
		if !ok || w.ProjectID == nil {
			continue
		}
		if stats, ok := result[*w.ProjectID]; ok {
			stats.ExecutionCount++
			if stats.LastExecution == nil || e.StartedAt.After(*stats.LastExecution) {
				t := e.StartedAt
				stats.LastExecution = &t
			}
		}
	}

	return result, nil
}

// ============================================================================
// Workflow Operations
// ============================================================================

func (r *MockRepository) CreateWorkflow(_ context.Context, workflow *database.WorkflowIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if workflow.ID == uuid.Nil {
		workflow.ID = uuid.New()
	}
	if workflow.Version == 0 {
		workflow.Version = 1
	}
	now := time.Now()
	workflow.CreatedAt = now
	workflow.UpdatedAt = now

	copy := *workflow
	r.workflows[workflow.ID] = &copy
	return nil
}

func (r *MockRepository) GetWorkflow(_ context.Context, id uuid.UUID) (*database.WorkflowIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflow, ok := r.workflows[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	copy := *workflow
	return &copy, nil
}

func (r *MockRepository) GetWorkflowByName(_ context.Context, name, folderPath string) (*database.WorkflowIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, w := range r.workflows {
		if w.Name == name && w.FolderPath == folderPath {
			copy := *w
			return &copy, nil
		}
	}
	return nil, database.ErrNotFound
}

func (r *MockRepository) UpdateWorkflow(_ context.Context, workflow *database.WorkflowIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.workflows[workflow.ID]; !ok {
		return database.ErrNotFound
	}
	workflow.UpdatedAt = time.Now()
	copy := *workflow
	r.workflows[workflow.ID] = &copy
	return nil
}

func (r *MockRepository) DeleteWorkflow(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.workflows, id)
	return nil
}

func (r *MockRepository) ListWorkflows(_ context.Context, folderPath string, limit, offset int) ([]*database.WorkflowIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflows := make([]*database.WorkflowIndex, 0)
	for _, w := range r.workflows {
		if folderPath == "" || w.FolderPath == folderPath {
			copy := *w
			workflows = append(workflows, &copy)
		}
	}

	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].UpdatedAt.After(workflows[j].UpdatedAt)
	})

	return applyPagination(workflows, limit, offset), nil
}

func (r *MockRepository) ListWorkflowsByProject(_ context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflows := make([]*database.WorkflowIndex, 0)
	for _, w := range r.workflows {
		if w.ProjectID != nil && *w.ProjectID == projectID {
			copy := *w
			workflows = append(workflows, &copy)
		}
	}

	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].UpdatedAt.After(workflows[j].UpdatedAt)
	})

	return applyPagination(workflows, limit, offset), nil
}

// ============================================================================
// Execution Operations
// ============================================================================

func (r *MockRepository) CreateExecution(_ context.Context, execution *database.ExecutionIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if execution.ID == uuid.Nil {
		execution.ID = uuid.New()
	}
	now := time.Now()
	execution.CreatedAt = now
	execution.UpdatedAt = now

	copy := *execution
	r.executions[execution.ID] = &copy
	return nil
}

func (r *MockRepository) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	execution, ok := r.executions[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	copy := *execution
	return &copy, nil
}

func (r *MockRepository) UpdateExecution(_ context.Context, execution *database.ExecutionIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.executions[execution.ID]; !ok {
		return database.ErrNotFound
	}
	execution.UpdatedAt = time.Now()
	copy := *execution
	r.executions[execution.ID] = &copy
	return nil
}

func (r *MockRepository) UpdateExecutionStatus(_ context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	execution, ok := r.executions[id]
	if !ok {
		return database.ErrNotFound
	}
	execution.Status = status
	if errorMessage != nil {
		execution.ErrorMessage = *errorMessage
	}
	execution.CompletedAt = completedAt
	execution.UpdatedAt = updatedAt
	return nil
}

func (r *MockRepository) UpdateExecutionResultPath(_ context.Context, id uuid.UUID, resultPath string, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	execution, ok := r.executions[id]
	if !ok {
		return database.ErrNotFound
	}
	execution.ResultPath = resultPath
	execution.UpdatedAt = updatedAt
	return nil
}

func (r *MockRepository) DeleteExecution(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.executions, id)
	return nil
}

func (r *MockRepository) ListExecutions(_ context.Context, workflowID *uuid.UUID, projectID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executions := make([]*database.ExecutionIndex, 0)
	for _, e := range r.executions {
		// Note: projectID filtering would require workflow lookup, skip for mock
		if workflowID == nil || e.WorkflowID == *workflowID {
			copy := *e
			executions = append(executions, &copy)
		}
	}

	sort.Slice(executions, func(i, j int) bool {
		return executions[i].StartedAt.After(executions[j].StartedAt)
	})

	return applyPagination(executions, limit, offset), nil
}

func (r *MockRepository) ListExecutionsByStatus(_ context.Context, status string, limit, offset int) ([]*database.ExecutionIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executions := make([]*database.ExecutionIndex, 0)
	for _, e := range r.executions {
		if e.Status == status {
			copy := *e
			executions = append(executions, &copy)
		}
	}

	sort.Slice(executions, func(i, j int) bool {
		return executions[i].StartedAt.After(executions[j].StartedAt)
	})

	return applyPagination(executions, limit, offset), nil
}

// ============================================================================
// Schedule Operations
// ============================================================================

func (r *MockRepository) CreateSchedule(_ context.Context, schedule *database.ScheduleIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	if schedule.Timezone == "" {
		schedule.Timezone = "UTC"
	}
	now := time.Now()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	copy := *schedule
	r.schedules[schedule.ID] = &copy
	return nil
}

func (r *MockRepository) GetSchedule(_ context.Context, id uuid.UUID) (*database.ScheduleIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schedule, ok := r.schedules[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	copy := *schedule
	return &copy, nil
}

func (r *MockRepository) UpdateSchedule(_ context.Context, schedule *database.ScheduleIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.schedules[schedule.ID]; !ok {
		return database.ErrNotFound
	}
	schedule.UpdatedAt = time.Now()
	copy := *schedule
	r.schedules[schedule.ID] = &copy
	return nil
}

func (r *MockRepository) DeleteSchedule(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.schedules, id)
	return nil
}

func (r *MockRepository) ListSchedules(_ context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*database.ScheduleIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schedules := make([]*database.ScheduleIndex, 0)
	for _, s := range r.schedules {
		if workflowID != nil && s.WorkflowID != *workflowID {
			continue
		}
		if activeOnly && !s.IsActive {
			continue
		}
		copy := *s
		schedules = append(schedules, &copy)
	}

	sort.Slice(schedules, func(i, j int) bool {
		// Sort by next_run_at ASC, with nil values last
		if schedules[i].NextRunAt == nil && schedules[j].NextRunAt == nil {
			return false
		}
		if schedules[i].NextRunAt == nil {
			return false
		}
		if schedules[j].NextRunAt == nil {
			return true
		}
		return schedules[i].NextRunAt.Before(*schedules[j].NextRunAt)
	})

	return applyPagination(schedules, limit, offset), nil
}

func (r *MockRepository) GetActiveSchedulesDue(_ context.Context, before time.Time) ([]*database.ScheduleIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schedules := make([]*database.ScheduleIndex, 0)
	for _, s := range r.schedules {
		if s.IsActive && s.NextRunAt != nil && !s.NextRunAt.After(before) {
			copy := *s
			schedules = append(schedules, &copy)
		}
	}

	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].NextRunAt.Before(*schedules[j].NextRunAt)
	})

	return schedules, nil
}

func (r *MockRepository) UpdateScheduleNextRun(_ context.Context, id uuid.UUID, nextRun time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	schedule, ok := r.schedules[id]
	if !ok {
		return database.ErrNotFound
	}
	schedule.NextRunAt = &nextRun
	return nil
}

func (r *MockRepository) UpdateScheduleLastRun(_ context.Context, id uuid.UUID, lastRun time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	schedule, ok := r.schedules[id]
	if !ok {
		return database.ErrNotFound
	}
	schedule.LastRunAt = &lastRun
	return nil
}

// ============================================================================
// Settings Operations
// ============================================================================

func (r *MockRepository) GetSetting(_ context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.settings[key]
	if !ok {
		return "", database.ErrNotFound
	}
	return value, nil
}

func (r *MockRepository) SetSetting(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.settings[key] = value
	return nil
}

func (r *MockRepository) DeleteSetting(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.settings, key)
	return nil
}

// ============================================================================
// Export Operations
// ============================================================================

func (r *MockRepository) CreateExport(_ context.Context, export *database.ExportIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if export.ID == uuid.Nil {
		export.ID = uuid.New()
	}
	if export.Status == "" {
		export.Status = "pending"
	}
	now := time.Now()
	export.CreatedAt = now
	export.UpdatedAt = now

	copy := *export
	r.exports[export.ID] = &copy
	return nil
}

func (r *MockRepository) GetExport(_ context.Context, id uuid.UUID) (*database.ExportIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	export, ok := r.exports[id]
	if !ok {
		return nil, database.ErrNotFound
	}

	// Join workflow name and execution date if available
	copy := *export
	if copy.WorkflowID != nil {
		if w, ok := r.workflows[*copy.WorkflowID]; ok {
			copy.WorkflowName = w.Name
		}
	}
	if e, ok := r.executions[copy.ExecutionID]; ok {
		copy.ExecutionDate = &e.StartedAt
	}

	return &copy, nil
}

func (r *MockRepository) UpdateExport(_ context.Context, export *database.ExportIndex) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.exports[export.ID]; !ok {
		return database.ErrNotFound
	}
	export.UpdatedAt = time.Now()
	copy := *export
	r.exports[export.ID] = &copy
	return nil
}

func (r *MockRepository) DeleteExport(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.exports, id)
	return nil
}

func (r *MockRepository) UpdateExportStatus(_ context.Context, id uuid.UUID, status string, errorMessage string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	export, ok := r.exports[id]
	if !ok {
		return database.ErrNotFound
	}
	export.Status = status
	export.UpdatedAt = time.Now()
	if errorMessage != "" {
		export.Error = errorMessage
	}
	return nil
}

func (r *MockRepository) UpdateExportComplete(_ context.Context, id uuid.UUID, storageURL string, fileSizeBytes int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	export, ok := r.exports[id]
	if !ok {
		return database.ErrNotFound
	}
	export.Status = "completed"
	export.StorageURL = storageURL
	export.FileSizeBytes = &fileSizeBytes
	export.UpdatedAt = time.Now()
	return nil
}

func (r *MockRepository) ListExports(_ context.Context, limit, offset int) ([]*database.ExportIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exports := make([]*database.ExportIndex, 0, len(r.exports))
	for _, e := range r.exports {
		copy := *e
		// Join workflow name
		if copy.WorkflowID != nil {
			if w, ok := r.workflows[*copy.WorkflowID]; ok {
				copy.WorkflowName = w.Name
			}
		}
		// Join execution date
		if ex, ok := r.executions[copy.ExecutionID]; ok {
			copy.ExecutionDate = &ex.StartedAt
		}
		exports = append(exports, &copy)
	}

	sort.Slice(exports, func(i, j int) bool {
		return exports[i].CreatedAt.After(exports[j].CreatedAt)
	})

	if limit <= 0 {
		limit = 100
	}
	return applyPagination(exports, limit, offset), nil
}

func (r *MockRepository) ListExportsByExecution(_ context.Context, executionID uuid.UUID) ([]*database.ExportIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exports := make([]*database.ExportIndex, 0)
	for _, e := range r.exports {
		if e.ExecutionID == executionID {
			copy := *e
			// Join workflow name
			if copy.WorkflowID != nil {
				if w, ok := r.workflows[*copy.WorkflowID]; ok {
					copy.WorkflowName = w.Name
				}
			}
			// Join execution date
			if ex, ok := r.executions[copy.ExecutionID]; ok {
				copy.ExecutionDate = &ex.StartedAt
			}
			exports = append(exports, &copy)
		}
	}

	sort.Slice(exports, func(i, j int) bool {
		return exports[i].CreatedAt.After(exports[j].CreatedAt)
	})

	return exports, nil
}

func (r *MockRepository) ListExportsByWorkflow(_ context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.ExportIndex, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exports := make([]*database.ExportIndex, 0)
	for _, e := range r.exports {
		if e.WorkflowID != nil && *e.WorkflowID == workflowID {
			copy := *e
			// Join workflow name
			if w, ok := r.workflows[workflowID]; ok {
				copy.WorkflowName = w.Name
			}
			// Join execution date
			if ex, ok := r.executions[copy.ExecutionID]; ok {
				copy.ExecutionDate = &ex.StartedAt
			}
			exports = append(exports, &copy)
		}
	}

	sort.Slice(exports, func(i, j int) bool {
		return exports[i].CreatedAt.After(exports[j].CreatedAt)
	})

	if limit <= 0 {
		limit = 100
	}
	return applyPagination(exports, limit, offset), nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// applyPagination applies limit and offset to a slice
func applyPagination[T any](items []*T, limit, offset int) []*T {
	if offset >= len(items) {
		return []*T{}
	}

	items = items[offset:]

	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}

	return items
}

// ============================================================================
// Test Helper Methods (for setting up test data)
// ============================================================================

// AddProject adds a project directly for test setup
func (r *MockRepository) AddProject(project *database.ProjectIndex) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	copy := *project
	r.projects[project.ID] = &copy
}

// AddWorkflow adds a workflow directly for test setup
func (r *MockRepository) AddWorkflow(workflow *database.WorkflowIndex) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if workflow.ID == uuid.Nil {
		workflow.ID = uuid.New()
	}
	copy := *workflow
	r.workflows[workflow.ID] = &copy
}

// AddExecution adds an execution directly for test setup
func (r *MockRepository) AddExecution(execution *database.ExecutionIndex) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if execution.ID == uuid.Nil {
		execution.ID = uuid.New()
	}
	copy := *execution
	r.executions[execution.ID] = &copy
}

// AddSchedule adds a schedule directly for test setup
func (r *MockRepository) AddSchedule(schedule *database.ScheduleIndex) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	copy := *schedule
	r.schedules[schedule.ID] = &copy
}

// AddExport adds an export directly for test setup
func (r *MockRepository) AddExport(export *database.ExportIndex) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if export.ID == uuid.Nil {
		export.ID = uuid.New()
	}
	copy := *export
	r.exports[export.ID] = &copy
}

// Reset clears all data in the repository
func (r *MockRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.projects = make(map[uuid.UUID]*database.ProjectIndex)
	r.workflows = make(map[uuid.UUID]*database.WorkflowIndex)
	r.executions = make(map[uuid.UUID]*database.ExecutionIndex)
	r.schedules = make(map[uuid.UUID]*database.ScheduleIndex)
	r.exports = make(map[uuid.UUID]*database.ExportIndex)
	r.settings = make(map[string]string)
}
