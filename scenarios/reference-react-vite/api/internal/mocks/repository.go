// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#mock-organization
// DOC: docs/internal/SEAMS.md#repository-seam
// Package mocks provides mock implementations of repository interfaces for testing.
// These mocks enable unit testing of handlers without database dependencies.
package mocks

import (
	"context"
	"errors"
	"sync"

	"reference-react-vite/api/domain/notes"
	"reference-react-vite/api/domain/projects"
	"reference-react-vite/api/domain/tasks"
	"reference-react-vite/api/repository"
)

// ErrNotFound is returned when an entity is not found.
var ErrNotFound = errors.New("entity not found")

// =============================================================================
// MockTaskRepository - In-memory task storage for testing
// =============================================================================

// MockTaskRepository implements repository.TaskRepository for testing.
// It stores tasks in memory and provides configurable error injection.
type MockTaskRepository struct {
	mu    sync.RWMutex
	tasks map[string]*tasks.Task

	// Error injection
	createErr  error
	findErr    error
	listErr    error
	updateErr  error
	deleteErr  error

	// Call tracking
	createCalls []createTaskCall
	findCalls   []string
	listCalls   []tasks.ListFilter
	updateCalls []updateTaskCall
	deleteCalls []string
}

type createTaskCall struct {
	Task *tasks.Task
}

type updateTaskCall struct {
	Task *tasks.Task
}

// NewMockTaskRepository creates a new mock task repository.
func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		tasks: make(map[string]*tasks.Task),
	}
}

// Create stores a new task.
func (m *MockTaskRepository) Create(_ context.Context, task *tasks.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createCalls = append(m.createCalls, createTaskCall{Task: task})

	if m.createErr != nil {
		return m.createErr
	}

	// Store a copy to avoid mutation issues
	stored := *task
	m.tasks[task.ID] = &stored
	return nil
}

// FindByID retrieves a task by ID.
func (m *MockTaskRepository) FindByID(_ context.Context, id string) (*tasks.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.findCalls = append(m.findCalls, id)

	if m.findErr != nil {
		return nil, m.findErr
	}

	task, ok := m.tasks[id]
	if !ok {
		return nil, nil // Return nil, nil for not found (matching real repo behavior)
	}

	// Return a copy to avoid mutation
	result := *task
	return &result, nil
}

// List returns tasks matching the filter.
func (m *MockTaskRepository) List(_ context.Context, filter tasks.ListFilter) ([]*tasks.Task, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.listCalls = append(m.listCalls, filter)

	if m.listErr != nil {
		return nil, 0, m.listErr
	}

	var result []*tasks.Task
	for _, t := range m.tasks {
		// Apply filters
		if filter.ProjectID != nil && t.ProjectID != *filter.ProjectID {
			continue
		}
		if filter.Status != nil && t.Status != *filter.Status {
			continue
		}
		if filter.Priority != nil && t.Priority != *filter.Priority {
			continue
		}
		taskCopy := *t
		result = append(result, &taskCopy)
	}

	total := len(result)

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		result = nil
	}
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result, total, nil
}

// Update modifies an existing task.
func (m *MockTaskRepository) Update(_ context.Context, task *tasks.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateCalls = append(m.updateCalls, updateTaskCall{Task: task})

	if m.updateErr != nil {
		return m.updateErr
	}

	if _, ok := m.tasks[task.ID]; !ok {
		return errors.New("task not found")
	}

	stored := *task
	m.tasks[task.ID] = &stored
	return nil
}

// Delete removes a task by ID.
func (m *MockTaskRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls = append(m.deleteCalls, id)

	if m.deleteErr != nil {
		return m.deleteErr
	}

	if _, ok := m.tasks[id]; !ok {
		return errors.New("task not found")
	}

	delete(m.tasks, id)
	return nil
}

// Builder methods for configuring the mock

// WithTask adds a task to the mock's internal storage.
func (m *MockTaskRepository) WithTask(task *tasks.Task) *MockTaskRepository {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := *task
	m.tasks[task.ID] = &stored
	return m
}

// WithCreateError configures Create to return an error.
func (m *MockTaskRepository) WithCreateError(err error) *MockTaskRepository {
	m.createErr = err
	return m
}

// WithFindError configures FindByID to return an error.
func (m *MockTaskRepository) WithFindError(err error) *MockTaskRepository {
	m.findErr = err
	return m
}

// WithListError configures List to return an error.
func (m *MockTaskRepository) WithListError(err error) *MockTaskRepository {
	m.listErr = err
	return m
}

// WithUpdateError configures Update to return an error.
func (m *MockTaskRepository) WithUpdateError(err error) *MockTaskRepository {
	m.updateErr = err
	return m
}

// WithDeleteError configures Delete to return an error.
func (m *MockTaskRepository) WithDeleteError(err error) *MockTaskRepository {
	m.deleteErr = err
	return m
}

// Assertion methods

// CreateCallCount returns the number of times Create was called.
func (m *MockTaskRepository) CreateCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.createCalls)
}

// FindCallCount returns the number of times FindByID was called.
func (m *MockTaskRepository) FindCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.findCalls)
}

// DeleteCallCount returns the number of times Delete was called.
func (m *MockTaskRepository) DeleteCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.deleteCalls)
}

// Reset clears all state and call tracking.
func (m *MockTaskRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = make(map[string]*tasks.Task)
	m.createCalls = nil
	m.findCalls = nil
	m.listCalls = nil
	m.updateCalls = nil
	m.deleteCalls = nil
	m.createErr = nil
	m.findErr = nil
	m.listErr = nil
	m.updateErr = nil
	m.deleteErr = nil
}

// Ensure MockTaskRepository implements TaskRepository at compile time.
var _ repository.TaskRepository = (*MockTaskRepository)(nil)

// =============================================================================
// MockProjectRepository - In-memory project storage for testing
// =============================================================================

// MockProjectRepository implements repository.ProjectRepository for testing.
type MockProjectRepository struct {
	mu       sync.RWMutex
	projects map[string]*projects.Project

	// Error injection
	createErr  error
	findErr    error
	listErr    error
	updateErr  error
	deleteErr  error

	// Call tracking
	createCalls []createProjectCall
	findCalls   []string
	listCalls   []projects.ListFilter
	updateCalls []updateProjectCall
	deleteCalls []string
}

type createProjectCall struct {
	Project *projects.Project
}

type updateProjectCall struct {
	Project *projects.Project
}

// NewMockProjectRepository creates a new mock project repository.
func NewMockProjectRepository() *MockProjectRepository {
	return &MockProjectRepository{
		projects: make(map[string]*projects.Project),
	}
}

// Create stores a new project.
func (m *MockProjectRepository) Create(_ context.Context, project *projects.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createCalls = append(m.createCalls, createProjectCall{Project: project})

	if m.createErr != nil {
		return m.createErr
	}

	stored := *project
	m.projects[project.ID] = &stored
	return nil
}

// FindByID retrieves a project by ID.
func (m *MockProjectRepository) FindByID(_ context.Context, id string) (*projects.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.findCalls = append(m.findCalls, id)

	if m.findErr != nil {
		return nil, m.findErr
	}

	project, ok := m.projects[id]
	if !ok {
		return nil, nil
	}

	result := *project
	return &result, nil
}

// List returns projects matching the filter.
func (m *MockProjectRepository) List(_ context.Context, filter projects.ListFilter) ([]*projects.Project, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.listCalls = append(m.listCalls, filter)

	if m.listErr != nil {
		return nil, 0, m.listErr
	}

	var result []*projects.Project
	for _, p := range m.projects {
		if filter.Status != nil && p.Status != *filter.Status {
			continue
		}
		projectCopy := *p
		result = append(result, &projectCopy)
	}

	total := len(result)

	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		result = nil
	}
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result, total, nil
}

// Update modifies an existing project.
func (m *MockProjectRepository) Update(_ context.Context, project *projects.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateCalls = append(m.updateCalls, updateProjectCall{Project: project})

	if m.updateErr != nil {
		return m.updateErr
	}

	if _, ok := m.projects[project.ID]; !ok {
		return errors.New("project not found")
	}

	stored := *project
	m.projects[project.ID] = &stored
	return nil
}

// Delete removes a project by ID.
func (m *MockProjectRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls = append(m.deleteCalls, id)

	if m.deleteErr != nil {
		return m.deleteErr
	}

	if _, ok := m.projects[id]; !ok {
		return errors.New("project not found")
	}

	delete(m.projects, id)
	return nil
}

// Builder methods

// WithProject adds a project to the mock's internal storage.
func (m *MockProjectRepository) WithProject(project *projects.Project) *MockProjectRepository {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := *project
	m.projects[project.ID] = &stored
	return m
}

// WithCreateError configures Create to return an error.
func (m *MockProjectRepository) WithCreateError(err error) *MockProjectRepository {
	m.createErr = err
	return m
}

// WithFindError configures FindByID to return an error.
func (m *MockProjectRepository) WithFindError(err error) *MockProjectRepository {
	m.findErr = err
	return m
}

// WithListError configures List to return an error.
func (m *MockProjectRepository) WithListError(err error) *MockProjectRepository {
	m.listErr = err
	return m
}

// WithUpdateError configures Update to return an error.
func (m *MockProjectRepository) WithUpdateError(err error) *MockProjectRepository {
	m.updateErr = err
	return m
}

// WithDeleteError configures Delete to return an error.
func (m *MockProjectRepository) WithDeleteError(err error) *MockProjectRepository {
	m.deleteErr = err
	return m
}

// Assertion methods

// CreateCallCount returns the number of times Create was called.
func (m *MockProjectRepository) CreateCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.createCalls)
}

// DeleteCallCount returns the number of times Delete was called.
func (m *MockProjectRepository) DeleteCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.deleteCalls)
}

// Reset clears all state.
func (m *MockProjectRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects = make(map[string]*projects.Project)
	m.createCalls = nil
	m.findCalls = nil
	m.listCalls = nil
	m.updateCalls = nil
	m.deleteCalls = nil
	m.createErr = nil
	m.findErr = nil
	m.listErr = nil
	m.updateErr = nil
	m.deleteErr = nil
}

// Ensure MockProjectRepository implements ProjectRepository at compile time.
var _ repository.ProjectRepository = (*MockProjectRepository)(nil)

// =============================================================================
// MockNoteRepository - In-memory note storage for testing
// =============================================================================

// MockNoteRepository implements repository.NoteRepository for testing.
type MockNoteRepository struct {
	mu    sync.RWMutex
	notes map[string]*notes.Note

	// Error injection
	createErr  error
	findErr    error
	listErr    error
	updateErr  error
	deleteErr  error

	// Call tracking
	createCalls []createNoteCall
	findCalls   []string
	listCalls   []notes.ListFilter
	updateCalls []updateNoteCall
	deleteCalls []string
}

type createNoteCall struct {
	Note *notes.Note
}

type updateNoteCall struct {
	Note *notes.Note
}

// NewMockNoteRepository creates a new mock note repository.
func NewMockNoteRepository() *MockNoteRepository {
	return &MockNoteRepository{
		notes: make(map[string]*notes.Note),
	}
}

// Create stores a new note.
func (m *MockNoteRepository) Create(_ context.Context, note *notes.Note) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createCalls = append(m.createCalls, createNoteCall{Note: note})

	if m.createErr != nil {
		return m.createErr
	}

	stored := *note
	m.notes[note.ID] = &stored
	return nil
}

// FindByID retrieves a note by ID.
func (m *MockNoteRepository) FindByID(_ context.Context, id string) (*notes.Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.findCalls = append(m.findCalls, id)

	if m.findErr != nil {
		return nil, m.findErr
	}

	note, ok := m.notes[id]
	if !ok {
		return nil, nil
	}

	result := *note
	return &result, nil
}

// ListByTask returns notes for a task matching the filter.
func (m *MockNoteRepository) ListByTask(_ context.Context, filter notes.ListFilter) ([]*notes.Note, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.listCalls = append(m.listCalls, filter)

	if m.listErr != nil {
		return nil, 0, m.listErr
	}

	var result []*notes.Note
	for _, n := range m.notes {
		if filter.TaskID != "" && n.TaskID != filter.TaskID {
			continue
		}
		noteCopy := *n
		result = append(result, &noteCopy)
	}

	total := len(result)

	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		result = nil
	}
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}

	return result, total, nil
}

// Update modifies an existing note.
func (m *MockNoteRepository) Update(_ context.Context, note *notes.Note) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateCalls = append(m.updateCalls, updateNoteCall{Note: note})

	if m.updateErr != nil {
		return m.updateErr
	}

	if _, ok := m.notes[note.ID]; !ok {
		return errors.New("note not found")
	}

	stored := *note
	m.notes[note.ID] = &stored
	return nil
}

// Delete removes a note by ID.
func (m *MockNoteRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteCalls = append(m.deleteCalls, id)

	if m.deleteErr != nil {
		return m.deleteErr
	}

	if _, ok := m.notes[id]; !ok {
		return errors.New("note not found")
	}

	delete(m.notes, id)
	return nil
}

// Builder methods

// WithNote adds a note to the mock's internal storage.
func (m *MockNoteRepository) WithNote(note *notes.Note) *MockNoteRepository {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := *note
	m.notes[note.ID] = &stored
	return m
}

// WithCreateError configures Create to return an error.
func (m *MockNoteRepository) WithCreateError(err error) *MockNoteRepository {
	m.createErr = err
	return m
}

// WithFindError configures FindByID to return an error.
func (m *MockNoteRepository) WithFindError(err error) *MockNoteRepository {
	m.findErr = err
	return m
}

// WithListError configures ListByTask to return an error.
func (m *MockNoteRepository) WithListError(err error) *MockNoteRepository {
	m.listErr = err
	return m
}

// WithUpdateError configures Update to return an error.
func (m *MockNoteRepository) WithUpdateError(err error) *MockNoteRepository {
	m.updateErr = err
	return m
}

// WithDeleteError configures Delete to return an error.
func (m *MockNoteRepository) WithDeleteError(err error) *MockNoteRepository {
	m.deleteErr = err
	return m
}

// Assertion methods

// CreateCallCount returns the number of times Create was called.
func (m *MockNoteRepository) CreateCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.createCalls)
}

// DeleteCallCount returns the number of times Delete was called.
func (m *MockNoteRepository) DeleteCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.deleteCalls)
}

// Reset clears all state.
func (m *MockNoteRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notes = make(map[string]*notes.Note)
	m.createCalls = nil
	m.findCalls = nil
	m.listCalls = nil
	m.updateCalls = nil
	m.deleteCalls = nil
	m.createErr = nil
	m.findErr = nil
	m.listErr = nil
	m.updateErr = nil
	m.deleteErr = nil
}

// Ensure MockNoteRepository implements NoteRepository at compile time.
var _ repository.NoteRepository = (*MockNoteRepository)(nil)
