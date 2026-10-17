package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/retention"
)

type cleanupRouteStore struct{}

func (cleanupRouteStore) GetExecution(context.Context, uuid.UUID) (*database.ExecutionIndex, error) {
	return nil, database.ErrNotFound
}

func (cleanupRouteStore) ListExecutions(context.Context, database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	return nil, 0, nil
}

func (cleanupRouteStore) DeleteExecution(context.Context, uuid.UUID) error { return nil }

func TestCleanupRoutesPreserveEstimatePreviewApplyAndIdempotency(t *testing.T) {
	recordingsRoot := t.TempDir()
	capturesRoot := t.TempDir()
	capture := filepath.Join(capturesRoot, "old-bundle")
	if err := os.Mkdir(capture, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(capture, "frame.png"), []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldAt := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(capture, oldAt, oldAt); err != nil {
		t.Fatal(err)
	}
	planner := retention.NewService(cleanupRouteStore{}, retention.OSFileSystem{}, recordingsRoot, nil).
		WithCaptureRoot(capturesRoot).
		WithRecoveryLockPath(filepath.Join(t.TempDir(), "recovery.lock"))
	router := chi.NewRouter()
	registerOwnerCleanupRoutes(router, planner)

	estimateRequest := httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate?min_age_seconds=3600", nil)
	estimateResponse := httptest.NewRecorder()
	router.ServeHTTP(estimateResponse, estimateRequest)
	if estimateResponse.Code != http.StatusOK {
		t.Fatalf("estimate status=%d body=%s", estimateResponse.Code, estimateResponse.Body.String())
	}
	var estimate ownerCleanupEstimate
	if err := json.Unmarshal(estimateResponse.Body.Bytes(), &estimate); err != nil {
		t.Fatal(err)
	}
	if estimate.ProviderID != "browser-automation-studio" || estimate.ItemCount != 1 || estimate.EstimatedBytes != 5 {
		t.Fatalf("estimate = %+v", estimate)
	}
	if _, err := os.Stat(capture); err != nil {
		t.Fatalf("estimate mutated evidence: %v", err)
	}

	previewBody, err := json.Marshal(map[string]any{"estimate": estimate})
	if err != nil {
		t.Fatal(err)
	}
	previewRequest := httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/preview", bytes.NewReader(previewBody))
	previewResponse := httptest.NewRecorder()
	router.ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", previewResponse.Code, previewResponse.Body.String())
	}
	var preview ownerCleanupPreview
	if err := json.Unmarshal(previewResponse.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 || preview.Items[0].ID != "capture:old-bundle" {
		t.Fatalf("preview = %+v", preview)
	}
	if _, err := os.Stat(capture); err != nil {
		t.Fatalf("preview mutated evidence: %v", err)
	}

	applyBody, err := json.Marshal(ownerCleanupApply{ProviderID: estimate.ProviderID, Preview: preview, IdempotencyKey: "cleanup-once", ApprovalMode: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	applyRequest := httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/apply", bytes.NewReader(applyBody))
	applyResponse := httptest.NewRecorder()
	router.ServeHTTP(applyResponse, applyRequest)
	if applyResponse.Code != http.StatusOK {
		t.Fatalf("apply status=%d body=%s", applyResponse.Code, applyResponse.Body.String())
	}
	var applied ownerCleanupApplyResponse
	if err := json.Unmarshal(applyResponse.Body.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	if applied.ReclaimedBytes != 5 || len(applied.RemovedItemIDs) != 1 || applied.RemovedItemIDs[0] != "capture:old-bundle" {
		t.Fatalf("apply response = %+v", applied)
	}
	if _, err := os.Stat(capture); !os.IsNotExist(err) {
		t.Fatalf("apply did not remove the selected evidence: %v", err)
	}

	applyAgain := httptest.NewRecorder()
	router.ServeHTTP(applyAgain, httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/apply", bytes.NewReader(applyBody)))
	if applyAgain.Code != http.StatusOK || applyAgain.Body.String() != applyResponse.Body.String() {
		t.Fatalf("idempotent replay status=%d body=%s, first=%s", applyAgain.Code, applyAgain.Body.String(), applyResponse.Body.String())
	}
}

func TestCleanupRoutesRejectApplyWithoutApproval(t *testing.T) {
	planner := retention.NewService(cleanupRouteStore{}, retention.OSFileSystem{}, t.TempDir(), nil).WithRecoveryLockPath(filepath.Join(t.TempDir(), "recovery.lock"))
	router := chi.NewRouter()
	registerOwnerCleanupRoutes(router, planner)
	body, err := json.Marshal(ownerCleanupApply{IdempotencyKey: "cleanup", ApprovalMode: "none"})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/apply", bytes.NewReader(body)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("apply status=%d body=%s", response.Code, response.Body.String())
	}

	delegatedBody, err := json.Marshal(ownerCleanupApply{IdempotencyKey: "recovery-apply", ApprovalMode: "none"})
	if err != nil {
		t.Fatal(err)
	}
	delegatedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/apply", bytes.NewReader(delegatedBody))
	delegatedRequest.Header.Set("X-Vrooli-Recovery-Lock", "held-by-storage-manager")
	delegatedRequest.Header.Set("X-Vrooli-Recovery-Only", "true")
	delegatedResponse := httptest.NewRecorder()
	router.ServeHTTP(delegatedResponse, delegatedRequest)
	if delegatedResponse.Code != http.StatusOK {
		t.Fatalf("delegated recovery apply status=%d body=%s", delegatedResponse.Code, delegatedResponse.Body.String())
	}
}

func TestCleanupEstimateExcludesActiveEvidence(t *testing.T) {
	recordingsRoot := t.TempDir()
	capturesRoot := t.TempDir()
	capture := filepath.Join(capturesRoot, "active-bundle")
	if err := os.Mkdir(capture, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(capture, "frame.png"), []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldAt := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(capture, oldAt, oldAt); err != nil {
		t.Fatal(err)
	}
	release := retention.BeginEvidenceActivity(capture)
	defer release()
	planner := retention.NewService(cleanupRouteStore{}, retention.OSFileSystem{}, recordingsRoot, nil).WithCaptureRoot(capturesRoot)
	router := chi.NewRouter()
	registerOwnerCleanupRoutes(router, planner)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate?min_age_seconds=3600", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("estimate status=%d body=%s", response.Code, response.Body.String())
	}
	var estimate ownerCleanupEstimate
	if err := json.Unmarshal(response.Body.Bytes(), &estimate); err != nil {
		t.Fatal(err)
	}
	if estimate.ItemCount != 0 || estimate.EstimatedBytes != 0 {
		t.Fatalf("active evidence was offered for cleanup: %+v", estimate)
	}
}
