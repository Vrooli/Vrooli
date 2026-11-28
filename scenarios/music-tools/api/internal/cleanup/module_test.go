package cleanup

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

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"music-tools/internal/storage"
)

func TestOwnerCleanupEstimatesAppliesAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	store := storage.NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	if err := store.PutOwned(context.Background(), "out/old.wav", "job-old", bytes.NewReader([]byte("audio")), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	if err := store.PutOwned(context.Background(), "out/keep.wav", "job-keep", bytes.NewReader([]byte("keep")), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetProtected("out/keep.wav", true)
	store.SetProtected("out/old.wav", false)
	now := time.Now().UTC()

	router := mux.NewRouter()
	Module(Deps{Store: store, Now: func() time.Time { return now.Add(8 * 24 * time.Hour) }}).Mount(router)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate?min_age_seconds=1", nil)
	req.Header.Set("X-Vrooli-Recovery-Only", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("estimate status=%d body=%s", rec.Code, rec.Body.String())
	}
	var estimate EstimateResponse
	if err := json.NewDecoder(rec.Body).Decode(&estimate); err != nil {
		t.Fatal(err)
	}
	if estimate.ItemCount != 1 || estimate.EstimatedBytes != 5 {
		t.Fatalf("estimate=%+v", estimate)
	}
	body, _ := json.Marshal(struct {
		Estimate EstimateResponse `json:"estimate"`
	}{estimate})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/preview", bytes.NewReader(body))
	req.Header.Set("X-Vrooli-Recovery-Only", "true")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var preview PreviewResponse
	if err := json.NewDecoder(rec.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	applyBody, _ := json.Marshal(ApplyRequest{ProviderID: ProviderID, Preview: preview, IdempotencyKey: "once", ApprovalMode: "operator"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/apply", bytes.NewReader(applyBody))
	req.Header.Set("X-Vrooli-Recovery-Only", "true")
	req.Header.Set("X-Vrooli-Recovery-Lock", "held-by-storage-manager")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var applied ApplyResponse
	if err := json.NewDecoder(rec.Body).Decode(&applied); err != nil {
		t.Fatal(err)
	}
	if applied.ReclaimedBytes != 5 || len(applied.RemovedItemIDs) != 1 {
		t.Fatalf("applied=%+v", applied)
	}
	if _, _, err := store.Get(context.Background(), "out/old.wav"); err == nil {
		t.Fatal("old blob remains")
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/cleanup/apply", bytes.NewReader(applyBody))
	req.Header.Set("X-Vrooli-Recovery-Only", "true")
	req.Header.Set("X-Vrooli-Recovery-Lock", "held-by-storage-manager")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var repeated ApplyResponse
	if err := json.NewDecoder(rec.Body).Decode(&repeated); err != nil {
		t.Fatal(err)
	}
	if !repeated.AlreadyDone || repeated.ReclaimedBytes != 5 {
		t.Fatalf("repeated=%+v", repeated)
	}
}

func TestReviewedCleanupSkipsAReservedTakeAndItsOwnershipReceipt(t *testing.T) {
	root := t.TempDir()
	store := storage.NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root)
	if err := store.PutOwned(t.Context(), "out/take.wav", "take", bytes.NewBufferString("audio"), "audio/wav", true); err != nil {
		t.Fatal(err)
	}
	store.SetProtected("out/take.wav", false)
	owner := &owner{store: store, now: func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }, done: map[string]ApplyResponse{}}
	reviewed := owner.candidates(time.Second)
	if len(reviewed) != 1 {
		t.Fatal("candidate missing")
	}
	if _, err := store.ProtectAndCommit("out/take.wav", func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(ApplyRequest{ProviderID: ProviderID, Preview: PreviewResponse{Items: reviewed, MinAgeSeconds: 1}, IdempotencyKey: "reserved", ApprovalMode: "owner"})
	request := httptest.NewRequest(http.MethodPost, "/cleanup/apply", bytes.NewReader(body))
	request.Header.Set("X-Vrooli-Recovery-Only", "true")
	request.Header.Set("X-Vrooli-Recovery-Lock", "held-by-storage-manager")
	response := httptest.NewRecorder()
	owner.apply(response, request)
	var result ApplyResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.RemovedItemIDs) != 0 || len(result.SkippedItemIDs) != 1 {
		t.Fatalf("cleanup result %+v", result)
	}
	if _, _, err := store.Get(t.Context(), "out/take.wav"); err != nil {
		t.Fatal("reserved bytes lost", err)
	}
	receipts, err := os.ReadDir(filepath.Join(root, "ownership"))
	if err != nil || len(receipts) != 2 {
		t.Fatalf("ownership receipt or MIME sidecar lost: %v %v", receipts, err)
	}
}
