// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#handler-tests
// DOC: docs/internal/SEAMS.md#http-handler-seam
// [REQ:RRV-API-003] Notes API - Unit tests for note HTTP handlers
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"reference-react-vite/api/domain/notes"
	"reference-react-vite/api/handlers"
	"reference-react-vite/api/internal/mocks"
	"reference-react-vite/api/internal/testutil"
)

// defaultNotePaginationCfg is used in tests.
var defaultNotePaginationCfg = handlers.PaginationConfig{
	DefaultLimit: 20,
	MaxLimit:     100,
}

// setupNoteRouter creates a test router with the note handler registered.
// Notes require both note and task repositories since notes are nested under tasks.
func setupNoteRouter(noteRepo *mocks.MockNoteRepository, taskRepo *mocks.MockTaskRepository) *mux.Router {
	r := mux.NewRouter()
	h := handlers.NewNoteHandler(noteRepo, taskRepo, defaultNotePaginationCfg)
	h.RegisterRoutes(r)
	return r
}

// =============================================================================
// NoteHandler.List Tests
// =============================================================================

func TestNoteHandler_List(t *testing.T) {
	tests := []struct {
		name         string
		taskID       string
		url          string
		setupTaskMock func(*mocks.MockTaskRepository)
		setupNoteMock func(*mocks.MockNoteRepository)
		wantStatus   int
		wantCount    int
		category     string
	}{
		{
			name:   "list_empty",
			taskID: "task-123",
			url:    "/api/v1/tasks/task-123/notes",
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {
				// No notes added
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
			category:   "happy_path",
		},
		{
			name:   "list_with_notes",
			taskID: "task-123",
			url:    "/api/v1/tasks/task-123/notes",
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {
				m.WithNote(testutil.NewNoteFactory().WithID("1").WithTaskID("task-123").Build())
				m.WithNote(testutil.NewNoteFactory().WithID("2").WithTaskID("task-123").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
			category:   "happy_path",
		},
		{
			name:   "list_filters_by_task",
			taskID: "task-123",
			url:    "/api/v1/tasks/task-123/notes",
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {
				m.WithNote(testutil.NewNoteFactory().WithID("1").WithTaskID("task-123").Build())
				m.WithNote(testutil.NewNoteFactory().WithID("2").WithTaskID("other-task").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name:   "list_with_pagination",
			taskID: "task-123",
			url:    "/api/v1/tasks/task-123/notes?limit=1&offset=1",
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {
				m.WithNote(testutil.NewNoteFactory().WithID("1").WithTaskID("task-123").Build())
				m.WithNote(testutil.NewNoteFactory().WithID("2").WithTaskID("task-123").Build())
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
			category:   "happy_path",
		},
		{
			name:   "list_task_not_found",
			taskID: "nonexistent",
			url:    "/api/v1/tasks/nonexistent/notes",
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				// Task does not exist
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusNotFound,
			category:      "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			noteRepo := mocks.NewMockNoteRepository()
			taskRepo := mocks.NewMockTaskRepository()
			tc.setupTaskMock(taskRepo)
			tc.setupNoteMock(noteRepo)
			router := setupNoteRouter(noteRepo, taskRepo)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusOK {
				var response ListResponse
				testutil.AssertJSON(t, rec, &response)

				if data, ok := response.Data.([]interface{}); ok {
					if len(data) != tc.wantCount {
						t.Errorf("expected %d notes, got %d", tc.wantCount, len(data))
					}
				}
			}
		})
	}
}

// =============================================================================
// NoteHandler.Create Tests
// =============================================================================

func TestNoteHandler_Create(t *testing.T) {
	tests := []struct {
		name          string
		taskID        string
		body          interface{}
		setupTaskMock func(*mocks.MockTaskRepository)
		setupNoteMock func(*mocks.MockNoteRepository)
		wantStatus    int
		category      string
	}{
		{
			name:   "create_valid_note",
			taskID: "task-123",
			body: map[string]interface{}{
				"content": "This is a note",
				"author":  "Test Author",
			},
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusCreated,
			category:      "happy_path",
		},
		{
			name:   "create_minimal_note",
			taskID: "task-123",
			body: map[string]interface{}{
				"content": "Just content",
			},
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusCreated,
			category:      "happy_path",
		},
		{
			name:   "create_task_not_found",
			taskID: "nonexistent",
			body: map[string]interface{}{
				"content": "Note content",
			},
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				// Task does not exist
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusNotFound,
			category:      "error",
		},
		{
			name:   "create_missing_content",
			taskID: "task-123",
			body:   map[string]interface{}{},
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusUnprocessableEntity,
			category:      "error",
		},
		{
			name:   "create_empty_content",
			taskID: "task-123",
			body: map[string]interface{}{
				"content": "",
			},
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusUnprocessableEntity,
			category:      "error",
		},
		{
			name:   "create_invalid_json",
			taskID: "task-123",
			body:   "not json",
			setupTaskMock: func(m *mocks.MockTaskRepository) {
				m.WithTask(testutil.NewTaskFactory().WithID("task-123").Build())
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusBadRequest,
			category:      "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			noteRepo := mocks.NewMockNoteRepository()
			taskRepo := mocks.NewMockTaskRepository()
			tc.setupTaskMock(taskRepo)
			tc.setupNoteMock(noteRepo)
			router := setupNoteRouter(noteRepo, taskRepo)

			req := testutil.MakeJSONRequest(t, http.MethodPost, "/api/v1/tasks/"+tc.taskID+"/notes", tc.body)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusCreated {
				var note notes.Note
				testutil.AssertJSON(t, rec, &note)
				if note.ID == "" {
					t.Error("expected non-empty note ID")
				}
				if note.TaskID != tc.taskID {
					t.Errorf("expected TaskID %q, got %q", tc.taskID, note.TaskID)
				}
			}
		})
	}
}

// =============================================================================
// NoteHandler.Get Tests
// =============================================================================

func TestNoteHandler_Get(t *testing.T) {
	tests := []struct {
		name          string
		noteID        string
		setupNoteMock func(*mocks.MockNoteRepository)
		wantStatus    int
		category      string
	}{
		{
			name:   "get_existing_note",
			noteID: "note-123",
			setupNoteMock: func(m *mocks.MockNoteRepository) {
				m.WithNote(testutil.NewNoteFactory().WithID("note-123").WithContent("Test content").Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:          "get_nonexistent_note",
			noteID:        "nonexistent",
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusNotFound,
			category:      "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			noteRepo := mocks.NewMockNoteRepository()
			taskRepo := mocks.NewMockTaskRepository()
			tc.setupNoteMock(noteRepo)
			router := setupNoteRouter(noteRepo, taskRepo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/"+tc.noteID, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			if tc.wantStatus == http.StatusOK {
				var note notes.Note
				testutil.AssertJSON(t, rec, &note)
				if note.ID != tc.noteID {
					t.Errorf("expected ID %q, got %q", tc.noteID, note.ID)
				}
			}
		})
	}
}

// =============================================================================
// NoteHandler.Update Tests
// =============================================================================

func TestNoteHandler_Update(t *testing.T) {
	tests := []struct {
		name          string
		noteID        string
		body          interface{}
		setupNoteMock func(*mocks.MockNoteRepository)
		wantStatus    int
		category      string
	}{
		{
			name:   "update_content",
			noteID: "note-123",
			body: map[string]interface{}{
				"content": "Updated content",
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {
				m.WithNote(testutil.NewNoteFactory().WithID("note-123").Build())
			},
			wantStatus: http.StatusOK,
			category:   "happy_path",
		},
		{
			name:   "update_nonexistent_note",
			noteID: "nonexistent",
			body: map[string]interface{}{
				"content": "Updated",
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusNotFound,
			category:      "error",
		},
		{
			name:   "update_with_empty_content",
			noteID: "note-123",
			body: map[string]interface{}{
				"content": "",
			},
			setupNoteMock: func(m *mocks.MockNoteRepository) {
				m.WithNote(testutil.NewNoteFactory().WithID("note-123").Build())
			},
			wantStatus: http.StatusUnprocessableEntity,
			category:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			noteRepo := mocks.NewMockNoteRepository()
			taskRepo := mocks.NewMockTaskRepository()
			tc.setupNoteMock(noteRepo)
			router := setupNoteRouter(noteRepo, taskRepo)

			req := testutil.MakeJSONRequest(t, http.MethodPatch, "/api/v1/notes/"+tc.noteID, tc.body)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)
		})
	}
}

// =============================================================================
// NoteHandler.Delete Tests
// =============================================================================

func TestNoteHandler_Delete(t *testing.T) {
	tests := []struct {
		name          string
		noteID        string
		setupNoteMock func(*mocks.MockNoteRepository)
		wantStatus    int
		category      string
	}{
		{
			name:   "delete_existing_note",
			noteID: "note-123",
			setupNoteMock: func(m *mocks.MockNoteRepository) {
				m.WithNote(testutil.NewNoteFactory().WithID("note-123").Build())
			},
			wantStatus: http.StatusNoContent,
			category:   "happy_path",
		},
		{
			name:          "delete_nonexistent_note",
			noteID:        "nonexistent",
			setupNoteMock: func(m *mocks.MockNoteRepository) {},
			wantStatus:    http.StatusNotFound,
			category:      "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ARRANGE
			noteRepo := mocks.NewMockNoteRepository()
			taskRepo := mocks.NewMockTaskRepository()
			tc.setupNoteMock(noteRepo)
			router := setupNoteRouter(noteRepo, taskRepo)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/notes/"+tc.noteID, nil)
			rec := httptest.NewRecorder()

			// ACT
			router.ServeHTTP(rec, req)

			// ASSERT
			testutil.AssertStatus(t, rec, tc.wantStatus)

			// Verify delete was called
			if tc.wantStatus == http.StatusNoContent {
				if noteRepo.DeleteCallCount() != 1 {
					t.Errorf("expected 1 delete call, got %d", noteRepo.DeleteCallCount())
				}
			}
		})
	}
}
