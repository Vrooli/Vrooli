// Package repository tests verify repository interfaces and implementations.
// Full integration tests require testcontainers - these are unit tests for interface contracts.
// [REQ:MOD-P0-004] Storage layer - repository interface verification tests
package repository

import (
	"context"
	"errors"
	"testing"

	"reference-react-vite/api/domain/notes"
	"reference-react-vite/api/domain/projects"
	"reference-react-vite/api/domain/tasks"
)

// Compile-time interface verification
var (
	_ TaskRepository    = (*mockTaskRepo)(nil)
	_ ProjectRepository = (*mockProjectRepo)(nil)
	_ NoteRepository    = (*mockNoteRepo)(nil)
)

// mockTaskRepo is a minimal implementation for interface verification.
type mockTaskRepo struct {
	tasks map[string]*tasks.Task
}

func (m *mockTaskRepo) Create(_ context.Context, task *tasks.Task) error {
	if task == nil || task.ID == "" {
		return errors.New("invalid task")
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) FindByID(_ context.Context, id string) (*tasks.Task, error) {
	if task, ok := m.tasks[id]; ok {
		return task, nil
	}
	return nil, errors.New("not found")
}

func (m *mockTaskRepo) List(_ context.Context, _ tasks.ListFilter) ([]*tasks.Task, int, error) {
	result := make([]*tasks.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		result = append(result, t)
	}
	return result, len(result), nil
}

func (m *mockTaskRepo) Update(_ context.Context, task *tasks.Task) error {
	if _, ok := m.tasks[task.ID]; !ok {
		return errors.New("not found")
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) Delete(_ context.Context, id string) error {
	delete(m.tasks, id)
	return nil
}

// mockProjectRepo is a minimal implementation for interface verification.
type mockProjectRepo struct {
	projects map[string]*projects.Project
}

func (m *mockProjectRepo) Create(_ context.Context, project *projects.Project) error {
	if project == nil || project.ID == "" {
		return errors.New("invalid project")
	}
	m.projects[project.ID] = project
	return nil
}

func (m *mockProjectRepo) FindByID(_ context.Context, id string) (*projects.Project, error) {
	if project, ok := m.projects[id]; ok {
		return project, nil
	}
	return nil, errors.New("not found")
}

func (m *mockProjectRepo) List(_ context.Context, _ projects.ListFilter) ([]*projects.Project, int, error) {
	result := make([]*projects.Project, 0, len(m.projects))
	for _, p := range m.projects {
		result = append(result, p)
	}
	return result, len(result), nil
}

func (m *mockProjectRepo) Update(_ context.Context, project *projects.Project) error {
	if _, ok := m.projects[project.ID]; !ok {
		return errors.New("not found")
	}
	m.projects[project.ID] = project
	return nil
}

func (m *mockProjectRepo) Delete(_ context.Context, id string) error {
	delete(m.projects, id)
	return nil
}

// mockNoteRepo is a minimal implementation for interface verification.
type mockNoteRepo struct {
	notes map[string]*notes.Note
}

func (m *mockNoteRepo) Create(_ context.Context, note *notes.Note) error {
	if note == nil || note.ID == "" {
		return errors.New("invalid note")
	}
	m.notes[note.ID] = note
	return nil
}

func (m *mockNoteRepo) FindByID(_ context.Context, id string) (*notes.Note, error) {
	if note, ok := m.notes[id]; ok {
		return note, nil
	}
	return nil, errors.New("not found")
}

func (m *mockNoteRepo) ListByTask(_ context.Context, filter notes.ListFilter) ([]*notes.Note, int, error) {
	result := make([]*notes.Note, 0)
	for _, n := range m.notes {
		if n.TaskID == filter.TaskID {
			result = append(result, n)
		}
	}
	return result, len(result), nil
}

func (m *mockNoteRepo) Update(_ context.Context, note *notes.Note) error {
	if _, ok := m.notes[note.ID]; !ok {
		return errors.New("not found")
	}
	m.notes[note.ID] = note
	return nil
}

func (m *mockNoteRepo) Delete(_ context.Context, id string) error {
	delete(m.notes, id)
	return nil
}

func TestTaskRepositoryInterface(t *testing.T) {
	repo := &mockTaskRepo{tasks: make(map[string]*tasks.Task)}
	ctx := context.Background()

	t.Run("Create_stores_task", func(t *testing.T) {
		task := &tasks.Task{ID: "task-1", Title: "Test Task", Status: tasks.StatusPending}
		err := repo.Create(ctx, task)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if _, ok := repo.tasks["task-1"]; !ok {
			t.Error("task was not stored")
		}
	})

	t.Run("FindByID_returns_stored_task", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "task-1")
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found.Title != "Test Task" {
			t.Errorf("expected title 'Test Task', got '%s'", found.Title)
		}
	})

	t.Run("List_returns_all_tasks", func(t *testing.T) {
		list, count, err := repo.List(ctx, tasks.ListFilter{})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
		if len(list) != 1 {
			t.Errorf("expected list length 1, got %d", len(list))
		}
	})

	t.Run("Update_modifies_task", func(t *testing.T) {
		task := &tasks.Task{ID: "task-1", Title: "Updated Task", Status: tasks.StatusInProgress}
		err := repo.Update(ctx, task)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		found, _ := repo.FindByID(ctx, "task-1")
		if found.Title != "Updated Task" {
			t.Errorf("expected title 'Updated Task', got '%s'", found.Title)
		}
	})

	t.Run("Delete_removes_task", func(t *testing.T) {
		err := repo.Delete(ctx, "task-1")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, ok := repo.tasks["task-1"]; ok {
			t.Error("task was not deleted")
		}
	})
}

func TestProjectRepositoryInterface(t *testing.T) {
	repo := &mockProjectRepo{projects: make(map[string]*projects.Project)}
	ctx := context.Background()

	t.Run("Create_stores_project", func(t *testing.T) {
		project := &projects.Project{ID: "proj-1", Name: "Test Project", Status: projects.StatusActive}
		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if _, ok := repo.projects["proj-1"]; !ok {
			t.Error("project was not stored")
		}
	})

	t.Run("FindByID_returns_stored_project", func(t *testing.T) {
		found, err := repo.FindByID(ctx, "proj-1")
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found.Name != "Test Project" {
			t.Errorf("expected name 'Test Project', got '%s'", found.Name)
		}
	})

	t.Run("Delete_removes_project", func(t *testing.T) {
		err := repo.Delete(ctx, "proj-1")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, ok := repo.projects["proj-1"]; ok {
			t.Error("project was not deleted")
		}
	})
}

func TestNoteRepositoryInterface(t *testing.T) {
	repo := &mockNoteRepo{notes: make(map[string]*notes.Note)}
	ctx := context.Background()

	t.Run("Create_stores_note", func(t *testing.T) {
		note := &notes.Note{ID: "note-1", TaskID: "task-1", Content: "Test content"}
		err := repo.Create(ctx, note)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if _, ok := repo.notes["note-1"]; !ok {
			t.Error("note was not stored")
		}
	})

	t.Run("ListByTask_filters_by_task", func(t *testing.T) {
		// Add another note for a different task
		note2 := &notes.Note{ID: "note-2", TaskID: "task-2", Content: "Other content"}
		_ = repo.Create(ctx, note2)

		list, count, err := repo.ListByTask(ctx, notes.ListFilter{TaskID: "task-1"})
		if err != nil {
			t.Fatalf("ListByTask failed: %v", err)
		}
		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
		if len(list) != 1 {
			t.Errorf("expected list length 1, got %d", len(list))
		}
		if list[0].TaskID != "task-1" {
			t.Error("returned note has wrong TaskID")
		}
	})
}

func TestRepositoriesStruct(t *testing.T) {
	t.Run("Repositories_struct_holds_all_repos", func(t *testing.T) {
		repos := &Repositories{
			Tasks:    &mockTaskRepo{tasks: make(map[string]*tasks.Task)},
			Projects: &mockProjectRepo{projects: make(map[string]*projects.Project)},
			Notes:    &mockNoteRepo{notes: make(map[string]*notes.Note)},
		}

		if repos.Tasks == nil {
			t.Error("Tasks repository should not be nil")
		}
		if repos.Projects == nil {
			t.Error("Projects repository should not be nil")
		}
		if repos.Notes == nil {
			t.Error("Notes repository should not be nil")
		}
	})
}
