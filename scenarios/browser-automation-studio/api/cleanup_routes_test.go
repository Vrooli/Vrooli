package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type exactRouteStore struct{ exec *database.ExecutionIndex }

func (s *exactRouteStore) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if s.exec == nil || s.exec.ID != id {
		return nil, database.ErrNotFound
	}
	return s.exec, nil
}
func (s *exactRouteStore) ListExecutions(context.Context, database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	if s.exec == nil {
		return nil, 0, nil
	}
	return []*database.ExecutionIndex{s.exec}, 1, nil
}
func (s *exactRouteStore) DeleteExecution(_ context.Context, id uuid.UUID) error {
	if s.exec != nil && s.exec.ID == id {
		s.exec = nil
	}
	return nil
}

func TestExactPurgeRoutesAreMetadataOnlyAndUUIDBound(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "bas-cleanup-route-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	dir := filepath.Join(root, id.String())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	secret := []byte("private fixture page text")
	if err := os.WriteFile(filepath.Join(dir, "result.json"), secret, 0o600); err != nil {
		t.Fatal(err)
	}
	wf := uuid.New()
	manifest := `{"id":"` + uuid.NewString() + `","schema_version":"bas-replay/v1","execution_id":"` + id.String() + `","workflow_id":"` + wf.String() + `","evidence":{"id":"` + uuid.NewString() + `","schema_version":"bas-evidence/v1","execution_id":"` + id.String() + `","artifacts":[]}}`
	if err := os.WriteFile(filepath.Join(dir, "evidence.proto.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	writeRoutePurgeInventory(t, dir, id, wf)
	store := &exactRouteStore{exec: &database.ExecutionIndex{ID: id, WorkflowID: wf, Status: database.ExecutionStatusCompleted, ResultPath: filepath.Join(dir, "result.json")}}
	planner := retention.NewService(store, retention.OSFileSystem{}, root, nil)
	router := chi.NewRouter()
	registerOwnerCleanupRoutes(router, planner)
	path := "/api/v1/executions/" + id.String()
	previewResponse := httptest.NewRecorder()
	router.ServeHTTP(previewResponse, httptest.NewRequest(http.MethodPost, path+"/purge-preview", nil))
	if previewResponse.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", previewResponse.Code, previewResponse.Body.String())
	}
	if bytes.Contains(previewResponse.Body.Bytes(), secret) {
		t.Fatal("preview leaked artifact content")
	}
	var preview retention.ExactPurgePreview
	if err := json.Unmarshal(previewResponse.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	wrongBody, _ := json.Marshal(exactPurgeApplyRequest{Confirmation: uuid.NewString(), Preview: preview})
	wrong := httptest.NewRecorder()
	router.ServeHTTP(wrong, httptest.NewRequest(http.MethodPost, path+"/purge", bytes.NewReader(wrongBody)))
	if wrong.Code != http.StatusPreconditionFailed {
		t.Fatalf("wrong confirmation status=%d", wrong.Code)
	}
	applyBody, _ := json.Marshal(exactPurgeApplyRequest{Confirmation: id.String(), Preview: preview})
	applied := httptest.NewRecorder()
	router.ServeHTTP(applied, httptest.NewRequest(http.MethodPost, path+"/purge", bytes.NewReader(applyBody)))
	if applied.Code != http.StatusOK {
		t.Fatalf("apply status=%d body=%s", applied.Code, applied.Body.String())
	}
	if bytes.Contains(applied.Body.Bytes(), secret) {
		t.Fatal("apply leaked artifact content")
	}
	var response struct {
		retention.ExactPurgePreview
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(applied.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Deleted || response.ExecutionID != id.String() {
		t.Fatalf("response=%+v", response)
	}
}

func writeRoutePurgeInventory(t *testing.T, dir string, id, workflowID uuid.UUID) {
	t.Helper()
	files := make([]map[string]any, 0, 2)
	for _, name := range []string{"result.json", "evidence.proto.json"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		files = append(files, map[string]any{"path": name, "bytes": len(data), "sha256": hex.EncodeToString(sum[:])})
	}
	raw, err := json.Marshal(map[string]any{"schema_version": "bas-execution-directory/v1", "execution_id": id.String(), "workflow_id": workflowID.String(), "files": files})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "purge-inventory.v1.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

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
