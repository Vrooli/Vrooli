package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

func setupTaskBoardTestHandlers(t *testing.T) (*Handlers, *store.FileTeamStore) {
	t.Helper()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	// Create a test team
	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Test Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}
	return handlers, teamStore
}

func TestGetTaskBoard_Empty(t *testing.T) {
	handlers, _ := setupTaskBoardTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/tasks", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTaskBoard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TaskBoardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(resp.Tasks))
	}
}

func TestGetTaskBoard_WithTasks(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{
		Tasks: []store.TeamTask{
			{ID: "task-1", Title: "First", Status: "todo", Priority: "P2"},
			{ID: "task-2", Title: "Second", Status: "done", Priority: "P1"},
		},
	}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatalf("save board: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/tasks", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTaskBoard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TaskBoardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resp.Tasks))
	}
}

func TestAddTask_Success(t *testing.T) {
	handlers, _ := setupTaskBoardTestHandlers(t)

	body, _ := json.Marshal(AddTaskRequest{
		Title:    "New task",
		Assignee: "agent-1",
		Priority: "P2",
		From:     "ui-user",
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddTask(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var task store.TeamTask
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if task.Title != "New task" {
		t.Errorf("expected 'New task', got: %s", task.Title)
	}
	if task.ID == "" {
		t.Error("expected non-empty ID")
	}
	if task.Status != "todo" {
		t.Errorf("expected default status 'todo', got: %s", task.Status)
	}
}

func TestAddTask_MissingTitle(t *testing.T) {
	handlers, _ := setupTaskBoardTestHandlers(t)

	body, _ := json.Marshal(AddTaskRequest{
		Title: "",
		From:  "ui-user",
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddTask(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddTask_InvalidJSON(t *testing.T) {
	handlers, _ := setupTaskBoardTestHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/tasks", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddTask(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddTask_DefaultPriority(t *testing.T) {
	handlers, _ := setupTaskBoardTestHandlers(t)

	body, _ := json.Marshal(AddTaskRequest{
		Title: "No priority set",
		From:  "ui-user",
	})

	req := httptest.NewRequest(http.MethodPost, "/teams/team-1/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.AddTask(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var task store.TeamTask
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if task.Priority != "P3" {
		t.Errorf("expected default priority 'P3', got: %s", task.Priority)
	}
}

func TestUpdateTask_Status(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{
		Tasks: []store.TeamTask{
			{ID: "task-1", Title: "Test", Status: "todo", Priority: "P2", UpdatedAt: "2025-01-01T00:00:00Z"},
		},
	}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	status := "done"
	body, _ := json.Marshal(UpdateTaskRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/tasks/task-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "taskId": "task-1"})
	w := httptest.NewRecorder()

	handlers.UpdateTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var task store.TeamTask
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if task.Status != "done" {
		t.Errorf("expected 'done', got: %s", task.Status)
	}
}

func TestUpdateTask_Title(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{
		Tasks: []store.TeamTask{
			{ID: "task-1", Title: "Original", Status: "todo", Priority: "P2", UpdatedAt: "2025-01-01T00:00:00Z"},
		},
	}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	title := "Updated title"
	body, _ := json.Marshal(UpdateTaskRequest{Title: &title})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/tasks/task-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "taskId": "task-1"})
	w := httptest.NewRecorder()

	handlers.UpdateTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var task store.TeamTask
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if task.Title != "Updated title" {
		t.Errorf("expected 'Updated title', got: %s", task.Title)
	}
}

func TestUpdateTask_Note(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{
		Tasks: []store.TeamTask{
			{ID: "task-1", Title: "Test", Status: "todo", Priority: "P2", UpdatedAt: "2025-01-01T00:00:00Z"},
		},
	}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	note := "Progress update"
	body, _ := json.Marshal(UpdateTaskRequest{Note: &note})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/tasks/task-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "taskId": "task-1"})
	w := httptest.NewRecorder()

	handlers.UpdateTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var task store.TeamTask
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(task.Notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(task.Notes))
	}
	if task.Notes[0].Text != "Progress update" {
		t.Errorf("expected 'Progress update', got: %s", task.Notes[0].Text)
	}
}

func TestUpdateTask_NotFound(t *testing.T) {
	handlers, _ := setupTaskBoardTestHandlers(t)

	status := "done"
	body, _ := json.Marshal(UpdateTaskRequest{Status: &status})

	req := httptest.NewRequest(http.MethodPatch, "/teams/team-1/tasks/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "taskId": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.UpdateTaskHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTask_Success(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{
		Tasks: []store.TeamTask{
			{ID: "task-1", Title: "Delete me", Status: "todo", Priority: "P2"},
		},
	}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/tasks/task-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "taskId": "task-1"})
	w := httptest.NewRecorder()

	handlers.DeleteTaskHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deletion
	got, err := teamStore.GetTaskBoard(ctx, "team-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tasks) != 0 {
		t.Errorf("expected 0 tasks after delete, got %d", len(got.Tasks))
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	handlers, _ := setupTaskBoardTestHandlers(t)

	req := httptest.NewRequest(http.MethodDelete, "/teams/team-1/tasks/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1", "taskId": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.DeleteTaskHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// --- Pagination tests ---

func makeTasks(n int) []store.TeamTask {
	tasks := make([]store.TeamTask, n)
	for i := range tasks {
		tasks[i] = store.TeamTask{
			ID:       fmt.Sprintf("task-%d", i+1),
			Title:    fmt.Sprintf("Task %d", i+1),
			Status:   "todo",
			Priority: "P3",
			Assignee: fmt.Sprintf("agent-%d", (i%3)+1),
		}
	}
	return tasks
}

func TestGetTaskBoard_Pagination(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{Tasks: makeTasks(30)}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/tasks?limit=10&offset=0", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTaskBoard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TaskBoardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tasks) != 10 {
		t.Fatalf("expected 10 tasks, got %d", len(resp.Tasks))
	}
	if resp.Total != 30 {
		t.Errorf("expected total=30, got %d", resp.Total)
	}
	if resp.Limit != 10 {
		t.Errorf("expected limit=10, got %d", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("expected offset=0, got %d", resp.Offset)
	}
}

func TestGetTaskBoard_Offset(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{Tasks: makeTasks(30)}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/tasks?limit=25&offset=20", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTaskBoard(w, req)

	var resp TaskBoardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tasks) != 10 {
		t.Fatalf("expected 10 tasks (30-20), got %d", len(resp.Tasks))
	}
	if resp.Total != 30 {
		t.Errorf("expected total=30, got %d", resp.Total)
	}
}

func TestGetTaskBoard_StatusFilter(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	tasks := makeTasks(10)
	tasks[0].Status = "done"
	tasks[3].Status = "done"
	tasks[7].Status = "done"
	board := &store.TeamTaskBoard{Tasks: tasks}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/tasks?status=done", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTaskBoard(w, req)

	var resp TaskBoardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tasks) != 3 {
		t.Fatalf("expected 3 done tasks, got %d", len(resp.Tasks))
	}
	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	for _, task := range resp.Tasks {
		if task.Status != "done" {
			t.Errorf("expected status 'done', got: %s", task.Status)
		}
	}
}

func TestGetTaskBoard_AssigneeFilter(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{Tasks: makeTasks(9)}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/tasks?assignee=agent-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTaskBoard(w, req)

	var resp TaskBoardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// makeTasks assigns agent-1 to indices 0, 3, 6 (i%3==0)
	if len(resp.Tasks) != 3 {
		t.Fatalf("expected 3 tasks for agent-1, got %d", len(resp.Tasks))
	}
	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	for _, task := range resp.Tasks {
		if task.Assignee != "agent-1" {
			t.Errorf("expected assignee 'agent-1', got: %s", task.Assignee)
		}
	}
}

func TestGetTaskBoard_OffsetBeyondTotal(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{Tasks: makeTasks(5)}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/tasks?offset=100", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTaskBoard(w, req)

	var resp TaskBoardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Tasks) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(resp.Tasks))
	}
	if resp.Total != 5 {
		t.Errorf("expected total=5, got %d", resp.Total)
	}
}

func TestGetTaskBoard_DefaultPagination(t *testing.T) {
	handlers, teamStore := setupTaskBoardTestHandlers(t)
	ctx := context.Background()

	board := &store.TeamTaskBoard{Tasks: makeTasks(5)}
	if err := teamStore.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/tasks", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()

	handlers.GetTaskBoard(w, req)

	var resp TaskBoardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Limit != 25 {
		t.Errorf("expected default limit=25, got %d", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("expected default offset=0, got %d", resp.Offset)
	}
	if len(resp.Tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(resp.Tasks))
	}
	if resp.Total != 5 {
		t.Errorf("expected total=5, got %d", resp.Total)
	}
}
