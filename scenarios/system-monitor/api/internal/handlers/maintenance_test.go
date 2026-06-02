package handlers

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"system-monitor-api/internal/repository/sqlite"
	"system-monitor-api/internal/services"
	"system-monitor-api/internal/testutil"
)

func newMaintenanceHandler(t *testing.T) *MaintenanceHandler {
	t.Helper()
	repo, err := sqlite.NewInMemoryRepository()
	if err != nil {
		t.Fatalf("NewInMemoryRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	svc := services.NewMetricsMaintenanceService(repo)
	return NewMaintenanceHandler(svc, slog.Default())
}

func TestMaintenance_RetentionPreview_ReadOnly(t *testing.T) {
	h := newMaintenanceHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/maintenance/metrics/retention/preview?days=30", nil)
	w := httptest.NewRecorder()
	h.RetentionPreview(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if body["success"] != true {
		t.Errorf("expected success=true, got %v", body["success"])
	}
}

func TestMaintenance_RetentionPreview_RejectsBadDays(t *testing.T) {
	h := newMaintenanceHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/maintenance/metrics/retention/preview?days=0", nil)
	w := httptest.NewRecorder()
	h.RetentionPreview(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusBadRequest)
}

func TestMaintenance_RetentionApply_RequiresConfirm(t *testing.T) {
	h := newMaintenanceHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/metrics/retention/apply",
		strings.NewReader(`{"retentionDays":30,"confirm":false}`))
	w := httptest.NewRecorder()
	h.RetentionApply(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusBadRequest)
}

func TestMaintenance_RetentionApply_WithConfirm(t *testing.T) {
	h := newMaintenanceHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/metrics/retention/apply",
		strings.NewReader(`{"retentionDays":30,"confirm":true}`))
	w := httptest.NewRecorder()
	h.RetentionApply(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if body["success"] != true {
		t.Errorf("expected success=true, got %v", body["success"])
	}
}

func TestMaintenance_CompactionPreview(t *testing.T) {
	h := newMaintenanceHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/maintenance/metrics/compaction/preview", nil)
	w := httptest.NewRecorder()
	h.CompactionPreview(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}

func TestMaintenance_CompactionApply_RequiresConfirm(t *testing.T) {
	h := newMaintenanceHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/metrics/compaction/apply",
		strings.NewReader(`{"confirm":false}`))
	w := httptest.NewRecorder()
	h.CompactionApply(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusBadRequest)
}

func TestMaintenance_CompactionApply_WithConfirm(t *testing.T) {
	h := newMaintenanceHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/metrics/compaction/apply",
		strings.NewReader(`{"confirm":true}`))
	w := httptest.NewRecorder()
	h.CompactionApply(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}
