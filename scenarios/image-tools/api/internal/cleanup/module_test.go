package cleanup

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
)

type fakeJobs struct{ jobs []internaljobs.Job }

func (f fakeJobs) ListRetentionCandidates(_ context.Context, _ time.Time, _ int) ([]internaljobs.Job, error) {
	return f.jobs, nil
}

func (f fakeJobs) List(_ context.Context, _ int) ([]internaljobs.Job, error) { return f.jobs, nil }

func TestOwnerCleanupOnlyDeletesReviewedTerminalOutput(t *testing.T) {
	root := t.TempDir()
	store := storage.NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	if err := store.Put(context.Background(), "out/old.png", bytes.NewReader([]byte("pixels")), "image/png"); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), "out/pinned.png", bytes.NewReader([]byte("keep")), "image/png"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	old := now.Add(-8 * 24 * time.Hour)
	jobs := fakeJobs{jobs: []internaljobs.Job{
		{ID: "old", ResultRef: "out/old.png", State: internaljobs.StateSucceeded, CreatedAt: old, FinishedAt: &old},
		{ID: "pinned", ResultRef: "out/pinned.png", State: internaljobs.StateSucceeded, Meta: map[string]string{"retention_pinned": "true"}, CreatedAt: old, FinishedAt: &old},
		{ID: "model", ResultRef: filepath.Join(root, "models", "weights.safetensors"), State: internaljobs.StateSucceeded, CreatedAt: old, FinishedAt: &old},
	}}
	m := Module(Deps{Jobs: jobs, Store: store, Now: func() time.Time { return now }})
	r := mux.NewRouter()
	m.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate?min_age_seconds=604800", nil)
	req.Header.Set("X-Vrooli-Recovery-Only", "true")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("estimate status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var estimate ownerEstimateResponse
	if err := json.NewDecoder(rec.Body).Decode(&estimate); err != nil {
		t.Fatal(err)
	}
	if estimate.ProviderID != ProviderID || estimate.ItemCount != 1 || estimate.EstimatedBytes != int64(len("pixels")) {
		t.Fatalf("estimate = %+v", estimate)
	}

	body, _ := json.Marshal(struct {
		Estimate ownerEstimateResponse `json:"estimate"`
	}{Estimate: estimate})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/preview", bytes.NewReader(body))
	req.Header.Set("X-Vrooli-Recovery-Only", "true")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var preview ownerPreviewResponse
	if err := json.NewDecoder(rec.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	applyBody, _ := json.Marshal(ownerApplyRequest{ProviderID: ProviderID, Preview: preview, IdempotencyKey: "cleanup-once", ApprovalMode: "operator"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/apply", bytes.NewReader(applyBody))
	req.Header.Set("X-Vrooli-Recovery-Only", "true")
	req.Header.Set("X-Vrooli-Recovery-Lock", "held-by-storage-manager")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("apply status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var applied ownerApplyResponse
	if err := json.NewDecoder(rec.Body).Decode(&applied); err != nil {
		t.Fatal(err)
	}
	if applied.ReclaimedBytes != int64(len("pixels")) || len(applied.RemovedItemIDs) != 1 {
		t.Fatalf("apply = %+v", applied)
	}
	if _, _, err := store.Get(context.Background(), "out/old.png"); err == nil {
		t.Fatal("managed output still exists after owner apply")
	}
	if _, _, err := store.Get(context.Background(), "out/pinned.png"); err != nil {
		t.Fatalf("pinned output was removed: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/apply", bytes.NewReader(applyBody))
	req.Header.Set("X-Vrooli-Recovery-Only", "true")
	req.Header.Set("X-Vrooli-Recovery-Lock", "held-by-storage-manager")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var repeated ownerApplyResponse
	if err := json.NewDecoder(rec.Body).Decode(&repeated); err != nil {
		t.Fatal(err)
	}
	if !repeated.AlreadyDone || repeated.ReclaimedBytes != applied.ReclaimedBytes {
		t.Fatalf("idempotent apply = %+v", repeated)
	}
}

func TestOwnerCleanupRejectsUnscopedRequests(t *testing.T) {
	root := t.TempDir()
	m := Module(Deps{Jobs: fakeJobs{}, Store: storage.NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)})
	r := mux.NewRouter()
	m.Mount(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestOwnerCleanupProtectsReferencedOutput(t *testing.T) {
	root := t.TempDir()
	store := storage.NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	if err := store.Put(context.Background(), "out/referenced.png", bytes.NewReader([]byte("keep")), "image/png"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	old := now.Add(-8 * 24 * time.Hour)
	m := Module(Deps{
		Jobs: fakeJobs{jobs: []internaljobs.Job{
			{ID: "old", ResultRef: "out/referenced.png", State: internaljobs.StateSucceeded, CreatedAt: old, FinishedAt: &old},
			{ID: "consumer", State: internaljobs.StateQueued, Payload: []byte(`{"input":"out/referenced.png"}`), CreatedAt: now},
		}},
		Store: store,
		Now:   func() time.Time { return now },
	})
	r := mux.NewRouter()
	m.Mount(r)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate?min_age_seconds=604800", nil)
	req.Header.Set("X-Vrooli-Recovery-Only", "true")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var estimate ownerEstimateResponse
	if err := json.NewDecoder(rec.Body).Decode(&estimate); err != nil {
		t.Fatal(err)
	}
	if estimate.ItemCount != 0 || estimate.EstimatedBytes != 0 {
		t.Fatalf("referenced output was eligible: %+v", estimate)
	}
}
