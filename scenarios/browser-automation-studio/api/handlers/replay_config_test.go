package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	exportservices "github.com/vrooli/browser-automation-studio/services/export"
)

type testRepository struct {
	settings map[string]string
}

func newTestRepository() *testRepository {
	return &testRepository{settings: make(map[string]string)}
}

func (r *testRepository) CreateProject(context.Context, *database.ProjectIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) GetProject(context.Context, uuid.UUID) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) GetProjectByName(context.Context, string) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) GetProjectByFolderPath(context.Context, string) (*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) UpdateProject(context.Context, *database.ProjectIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) DeleteProject(context.Context, uuid.UUID) error {
	return errors.New("not implemented")
}

func (r *testRepository) ListProjects(context.Context, int, int) ([]*database.ProjectIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) GetProjectStats(context.Context, uuid.UUID) (*database.ProjectStats, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) GetProjectsStats(context.Context, []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) CreateWorkflow(context.Context, *database.WorkflowIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) GetWorkflow(context.Context, uuid.UUID) (*database.WorkflowIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) GetWorkflowByName(context.Context, string, string) (*database.WorkflowIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) UpdateWorkflow(context.Context, *database.WorkflowIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) DeleteWorkflow(context.Context, uuid.UUID) error {
	return errors.New("not implemented")
}

func (r *testRepository) ListWorkflows(context.Context, string, int, int) ([]*database.WorkflowIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) ListWorkflowsByProject(context.Context, uuid.UUID, int, int) ([]*database.WorkflowIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) CreateExecution(context.Context, *database.ExecutionIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) GetExecution(context.Context, uuid.UUID) (*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) UpdateExecution(context.Context, *database.ExecutionIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) DeleteExecution(context.Context, uuid.UUID) error {
	return errors.New("not implemented")
}

func (r *testRepository) ListExecutions(context.Context, *uuid.UUID, int, int) ([]*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) ListExecutionsByStatus(context.Context, string, int, int) ([]*database.ExecutionIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) CreateSchedule(context.Context, *database.ScheduleIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) GetSchedule(context.Context, uuid.UUID) (*database.ScheduleIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) UpdateSchedule(context.Context, *database.ScheduleIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) DeleteSchedule(context.Context, uuid.UUID) error {
	return errors.New("not implemented")
}

func (r *testRepository) ListSchedules(context.Context, *uuid.UUID, bool, int, int) ([]*database.ScheduleIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) GetActiveSchedulesDue(context.Context, time.Time) ([]*database.ScheduleIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) UpdateScheduleNextRun(context.Context, uuid.UUID, time.Time) error {
	return errors.New("not implemented")
}

func (r *testRepository) UpdateScheduleLastRun(context.Context, uuid.UUID, time.Time) error {
	return errors.New("not implemented")
}

func (r *testRepository) GetSetting(_ context.Context, key string) (string, error) {
	value, ok := r.settings[key]
	if !ok {
		return "", database.ErrNotFound
	}
	return value, nil
}

func (r *testRepository) SetSetting(_ context.Context, key, value string) error {
	r.settings[key] = value
	return nil
}

func (r *testRepository) DeleteSetting(_ context.Context, key string) error {
	delete(r.settings, key)
	return nil
}

func (r *testRepository) CreateExport(context.Context, *database.ExportIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) GetExport(context.Context, uuid.UUID) (*database.ExportIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) UpdateExport(context.Context, *database.ExportIndex) error {
	return errors.New("not implemented")
}

func (r *testRepository) DeleteExport(context.Context, uuid.UUID) error {
	return errors.New("not implemented")
}

func (r *testRepository) ListExports(context.Context, int, int) ([]*database.ExportIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) ListExportsByExecution(context.Context, uuid.UUID) ([]*database.ExportIndex, error) {
	return nil, errors.New("not implemented")
}

func (r *testRepository) ListExportsByWorkflow(context.Context, uuid.UUID, int, int) ([]*database.ExportIndex, error) {
	return nil, errors.New("not implemented")
}

var _ database.Repository = (*testRepository)(nil)

func TestReplayConfigHandlers(t *testing.T) {
	repo := newTestRepository()
	handler := &Handler{repo: repo, log: logrus.New()}

	t.Run("get empty config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/replay-config", nil)
		rr := httptest.NewRecorder()
		handler.GetReplayConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		var payload ReplayConfigResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if len(payload.Config) != 0 {
			t.Fatalf("expected empty config, got %+v", payload.Config)
		}
	})

	t.Run("put and get config", func(t *testing.T) {
		body := `{"config":{"chromeTheme":"aurora","cursorSpeedProfile":"easeInOut"}}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/replay-config", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.PutReplayConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/replay-config", nil)
		rr = httptest.NewRecorder()
		handler.GetReplayConfig(rr, req)

		var payload ReplayConfigResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if payload.Config["chromeTheme"] != "aurora" {
			t.Fatalf("expected chromeTheme aurora, got %v", payload.Config["chromeTheme"])
		}
	})

	t.Run("delete config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/replay-config", nil)
		rr := httptest.NewRecorder()
		handler.DeleteReplayConfig(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/v1/replay-config", nil)
		rr = httptest.NewRecorder()
		handler.GetReplayConfig(rr, req)

		var payload ReplayConfigResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if len(payload.Config) != 0 {
			t.Fatalf("expected empty config after delete, got %+v", payload.Config)
		}
	})
}

func TestReplayConfigOverridePrecedence(t *testing.T) {
	spec := &exportservices.ReplayMovieSpec{
		Decor: exportservices.ExportDecor{
			ChromeTheme:     "aurora",
			BackgroundTheme: "aurora",
		},
	}
	config := map[string]any{
		"chromeTheme":     "midnight",
		"backgroundTheme": "nebula",
	}
	applyExportOverrides(spec, replayConfigToOverrides(config))

	requestOverrides := &executionExportOverrides{
		ThemePreset: &themePresetOverride{
			ChromeTheme:     "solar",
			BackgroundTheme: "dawn",
		},
	}
	applyExportOverrides(spec, requestOverrides)

	if spec.Decor.ChromeTheme != "solar" {
		t.Fatalf("expected chrome theme solar, got %s", spec.Decor.ChromeTheme)
	}
	if spec.Decor.BackgroundTheme != "dawn" {
		t.Fatalf("expected background theme dawn, got %s", spec.Decor.BackgroundTheme)
	}
}
