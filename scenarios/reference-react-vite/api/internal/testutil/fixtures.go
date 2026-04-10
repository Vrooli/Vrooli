// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// Package testutil provides test fixtures and factories for creating test data.
package testutil

import (
	"time"

	"reference-react-vite/api/domain/notes"
	"reference-react-vite/api/domain/projects"
	"reference-react-vite/api/domain/tasks"
)

// TaskFactory creates Task test fixtures with builder pattern.
type TaskFactory struct {
	task tasks.Task
}

// NewTaskFactory creates a new factory with default valid values.
func NewTaskFactory() *TaskFactory {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &TaskFactory{
		task: tasks.Task{
			ID:          "00000000-0000-0000-0000-000000000001",
			Title:       "Test Task",
			Description: "A test task description",
			Status:      tasks.StatusPending,
			Priority:    tasks.PriorityMedium,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

// WithID sets a custom ID.
func (f *TaskFactory) WithID(id string) *TaskFactory {
	f.task.ID = id
	return f
}

// WithProjectID sets the project ID.
func (f *TaskFactory) WithProjectID(id string) *TaskFactory {
	f.task.ProjectID = id
	return f
}

// WithTitle sets a custom title.
func (f *TaskFactory) WithTitle(title string) *TaskFactory {
	f.task.Title = title
	return f
}

// WithDescription sets a custom description.
func (f *TaskFactory) WithDescription(desc string) *TaskFactory {
	f.task.Description = desc
	return f
}

// WithStatus sets a custom status.
func (f *TaskFactory) WithStatus(status tasks.Status) *TaskFactory {
	f.task.Status = status
	return f
}

// WithPriority sets a custom priority.
func (f *TaskFactory) WithPriority(priority tasks.Priority) *TaskFactory {
	f.task.Priority = priority
	return f
}

// WithDueDate sets a custom due date.
func (f *TaskFactory) WithDueDate(dueDate time.Time) *TaskFactory {
	f.task.DueDate = &dueDate
	return f
}

// Build returns the configured Task.
func (f *TaskFactory) Build() *tasks.Task {
	// Return a copy to prevent accidental mutation
	result := f.task
	return &result
}

// ProjectFactory creates Project test fixtures with builder pattern.
type ProjectFactory struct {
	project projects.Project
}

// NewProjectFactory creates a new factory with default valid values.
func NewProjectFactory() *ProjectFactory {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &ProjectFactory{
		project: projects.Project{
			ID:          "00000000-0000-0000-0000-000000000001",
			Name:        "Test Project",
			Description: "A test project description",
			Status:      projects.StatusActive,
			Color:       "#3498db",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}

// WithID sets a custom ID.
func (f *ProjectFactory) WithID(id string) *ProjectFactory {
	f.project.ID = id
	return f
}

// WithName sets a custom name.
func (f *ProjectFactory) WithName(name string) *ProjectFactory {
	f.project.Name = name
	return f
}

// WithDescription sets a custom description.
func (f *ProjectFactory) WithDescription(desc string) *ProjectFactory {
	f.project.Description = desc
	return f
}

// WithStatus sets a custom status.
func (f *ProjectFactory) WithStatus(status projects.Status) *ProjectFactory {
	f.project.Status = status
	return f
}

// WithColor sets a custom color.
func (f *ProjectFactory) WithColor(color string) *ProjectFactory {
	f.project.Color = color
	return f
}

// Build returns the configured Project.
func (f *ProjectFactory) Build() *projects.Project {
	// Return a copy to prevent accidental mutation
	result := f.project
	return &result
}

// NoteFactory creates Note test fixtures with builder pattern.
type NoteFactory struct {
	note notes.Note
}

// NewNoteFactory creates a new factory with default valid values.
func NewNoteFactory() *NoteFactory {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &NoteFactory{
		note: notes.Note{
			ID:        "00000000-0000-0000-0000-000000000001",
			TaskID:    "00000000-0000-0000-0000-000000000002",
			Content:   "Test note content",
			Author:    "Test Author",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// WithID sets a custom ID.
func (f *NoteFactory) WithID(id string) *NoteFactory {
	f.note.ID = id
	return f
}

// WithTaskID sets the task ID.
func (f *NoteFactory) WithTaskID(taskID string) *NoteFactory {
	f.note.TaskID = taskID
	return f
}

// WithContent sets a custom content.
func (f *NoteFactory) WithContent(content string) *NoteFactory {
	f.note.Content = content
	return f
}

// WithAuthor sets a custom author.
func (f *NoteFactory) WithAuthor(author string) *NoteFactory {
	f.note.Author = author
	return f
}

// Build returns the configured Note.
func (f *NoteFactory) Build() *notes.Note {
	// Return a copy to prevent accidental mutation
	result := f.note
	return &result
}
