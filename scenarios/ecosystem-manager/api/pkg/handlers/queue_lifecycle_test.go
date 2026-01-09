package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestQueueHandlersStartStop(t *testing.T) {
	settings.ResetSettings()
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_, queueHandlers, _, _, _ := createTestHandlers(t, tempDir)

	startReq := httptest.NewRequest(http.MethodPost, "/queue/start", nil)
	startResp := httptest.NewRecorder()
	queueHandlers.StartQueueProcessorHandler(startResp, startReq)

	if startResp.Code != http.StatusOK {
		t.Fatalf("start handler status = %d, want %d", startResp.Code, http.StatusOK)
	}
	if !settings.IsActive() {
		t.Fatalf("settings should be marked active after start")
	}

	stopReq := httptest.NewRequest(http.MethodPost, "/queue/stop", nil)
	stopResp := httptest.NewRecorder()
	queueHandlers.StopQueueProcessorHandler(stopResp, stopReq)

	if stopResp.Code != http.StatusOK {
		t.Fatalf("stop handler status = %d, want %d", stopResp.Code, http.StatusOK)
	}
	if settings.IsActive() {
		t.Fatalf("settings should be inactive after stop")
	}

	status := queueHandlers.processor.GetQueueStatus()
	if active, _ := status["settings_active"].(bool); active {
		t.Fatalf("processor status should reflect inactive settings")
	}
}

func TestQueueHandlersMaintenanceState(t *testing.T) {
	settings.ResetSettings()
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_, queueHandlers, _, _, _ := createTestHandlers(t, tempDir)

	payload := []byte(`{"state":"inactive"}`)
	req := httptest.NewRequest(http.MethodPost, "/queue/maintenance/state", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	queueHandlers.SetMaintenanceStateHandler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("maintenance handler status = %d, want %d", resp.Code, http.StatusOK)
	}

	status := queueHandlers.processor.GetQueueStatus()
	if state, _ := status["maintenance_state"].(string); state != "inactive" {
		t.Fatalf("expected maintenance_state inactive, got %s", state)
	}

	// Resume to active
	payload = []byte(`{"state":"active"}`)
	req = httptest.NewRequest(http.MethodPost, "/queue/maintenance/state", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	queueHandlers.SetMaintenanceStateHandler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("maintenance resume status = %d, want %d", resp.Code, http.StatusOK)
	}

	status = queueHandlers.processor.GetQueueStatus()
	if state, _ := status["maintenance_state"].(string); state != "active" {
		t.Fatalf("expected maintenance_state active after resume, got %s", state)
	}
}

func TestTerminateProcessHandlerNotFound(t *testing.T) {
	settings.ResetSettings()
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_, queueHandlers, _, _, _ := createTestHandlers(t, tempDir)

	body, _ := json.Marshal(map[string]string{"task_id": "non-existent"})
	req := httptest.NewRequest(http.MethodPost, "/queue/processes/terminate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	queueHandlers.TerminateProcessHandler(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("terminate handler status = %d, want %d", resp.Code, http.StatusNotFound)
	}
}

func TestTerminateProcessHandlerRespectsAutoRequeueLock(t *testing.T) {
	settings.ResetSettings()
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_, queueHandlers, _, _, _ := createTestHandlers(t, tempDir)

	task := tasks.TaskItem{
		ID:                   "locked-task",
		Status:               tasks.StatusInProgress,
		ProcessorAutoRequeue: false, // lock pending moves
	}
	mustSaveTask(t, queueHandlers.storage, tasks.StatusInProgress, task)

	body, _ := json.Marshal(map[string]string{"task_id": task.ID})
	req := httptest.NewRequest(http.MethodPost, "/queue/processes/terminate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	queueHandlers.TerminateProcessHandler(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("terminate handler status = %d, want %d (auto-requeue lock respected)", resp.Code, http.StatusConflict)
	}
}

func TestTerminateProcessHandlerAllowsPendingWhenAutoRequeueEnabled(t *testing.T) {
	settings.ResetSettings()
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_, queueHandlers, _, _, _ := createTestHandlers(t, tempDir)

	task := tasks.TaskItem{
		ID:                   "unlocked-task",
		Status:               tasks.StatusInProgress,
		ProcessorAutoRequeue: true,
	}
	mustSaveTask(t, queueHandlers.storage, tasks.StatusInProgress, task)

	body, _ := json.Marshal(map[string]string{"task_id": task.ID})
	req := httptest.NewRequest(http.MethodPost, "/queue/processes/terminate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	queueHandlers.TerminateProcessHandler(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("terminate handler status = %d, want %d", resp.Code, http.StatusOK)
	}

	updated, status, err := queueHandlers.storage.GetTaskByID(task.ID)
	if err != nil {
		t.Fatalf("load task after terminate: %v", err)
	}
	if status != tasks.StatusPending || updated.Status != tasks.StatusPending {
		t.Fatalf("expected task moved to pending, got status %s", status)
	}
}
