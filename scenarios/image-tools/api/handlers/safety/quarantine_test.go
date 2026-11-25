package safety

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	internalquarantine "image-tools/internal/quarantine"
	internalsafety "image-tools/internal/safety"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
)

func newQuarantine(t *testing.T, retention time.Duration) *internalquarantine.Store {
	t.Helper()
	db := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(internalquarantine.Schema)); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return internalquarantine.New(db, blobstore.NewMemoryBlobStore(), retention)
}

func quarantineHandler(t *testing.T, store *internalquarantine.Store, requireAdmin bool, token string) *connectHandler {
	t.Helper()
	return NewConnectHandlerWithDeps(Deps{
		Policy:       internalsafety.PolicyFor(internalsafety.TierLocal),
		Quarantine:   store,
		RequireAdmin: requireAdmin,
		AdminToken:   token,
		Jobs:         fakeJobs{},
	})
}

func quarantined(t *testing.T, store *internalquarantine.Store) int64 {
	t.Helper()
	if _, err := store.Quarantine(context.Background(), internalquarantine.Item{
		JobID: "job-1", OriginalRef: "out/a.png", ReasonCode: "sexual_unassessable",
		SafetyReview: "refused", Score: 0.95, Evidence: `{"profile_name":"local-balanced"}`,
		Finding: `{"refuse":true}`, MIME: "image/png",
	}, []byte("refused pixels")); err != nil {
		t.Fatal(err)
	}
	items, _, err := store.List(context.Background(), 10, 0)
	if err != nil || len(items) != 1 {
		t.Fatalf("list = %v/%v", items, err)
	}
	return items[0].ID
}

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

func TestQuarantineReviewIsOpenLocally(t *testing.T) {
	store := newQuarantine(t, time.Hour)
	id := quarantined(t, store)
	handler := quarantineHandler(t, store, false, "")

	list, err := handler.ListQuarantinedOutputs(context.Background(), connect.NewRequest(&safetyv1.ListQuarantinedOutputsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if list.Msg.GetTotal() != 1 || list.Msg.GetOutputs()[0].GetJobId() != "job-1" {
		t.Fatalf("list = %+v", list.Msg)
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/safety/quarantine/{id}", handler.quarantineImage)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/safety/quarantine/"+itoa(id)+"?download=1", nil))
	if w.Code != http.StatusOK || w.Body.String() != "refused pixels" {
		t.Fatalf("image = %d %q", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store, max-age=0" {
		t.Fatalf("cache = %q", got)
	}
	if got := w.Header().Get("Content-Disposition"); got == "" {
		t.Fatalf("download disposition = %q", got)
	}
}

func TestQuarantineReviewRequiresTokenOnPublic(t *testing.T) {
	store := newQuarantine(t, time.Hour)
	id := quarantined(t, store)
	handler := quarantineHandler(t, store, true, "")

	if _, err := handler.ListQuarantinedOutputs(context.Background(), connect.NewRequest(&safetyv1.ListQuarantinedOutputsRequest{})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("public without token = %v", err)
	}

	handler = quarantineHandler(t, store, true, "secret")
	req := connect.NewRequest(&safetyv1.ListQuarantinedOutputsRequest{})
	if _, err := handler.ListQuarantinedOutputs(context.Background(), req); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("public without header = %v", err)
	}
	req.Header().Set("Authorization", "Bearer secret")
	if _, err := handler.ListQuarantinedOutputs(context.Background(), req); err != nil {
		t.Fatalf("public with token = %v", err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/safety/quarantine/{id}", handler.quarantineImage)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/safety/quarantine/"+itoa(id), nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("image without token = %d", w.Code)
	}
	reqImage := httptest.NewRequest(http.MethodGet, "/api/v1/safety/quarantine/"+itoa(id), nil)
	reqImage.Header.Set("X-Image-Tools-Admin-Token", "secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, reqImage)
	if w.Code != http.StatusOK || w.Body.String() != "refused pixels" {
		t.Fatalf("image with token = %d %q", w.Code, w.Body.String())
	}
}

func TestQuarantineImageExpires(t *testing.T) {
	store := newQuarantine(t, 100*time.Millisecond)
	id := quarantined(t, store)
	handler := quarantineHandler(t, store, false, "")
	time.Sleep(150 * time.Millisecond)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/safety/quarantine/{id}", handler.quarantineImage)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/safety/quarantine/"+itoa(id), nil))
	if w.Code != http.StatusGone {
		t.Fatalf("expired image = %d", w.Code)
	}
}

func TestEmptyQuarantineListIsAnEmptyResponse(t *testing.T) {
	handler := NewConnectHandler(internalsafety.PolicyFor(internalsafety.TierLocal))
	resp, err := handler.ListQuarantinedOutputs(context.Background(), connect.NewRequest(&safetyv1.ListQuarantinedOutputsRequest{}))
	if err != nil || resp.Msg.GetTotal() != 0 {
		t.Fatalf("empty = %v/%v", resp, err)
	}
}
