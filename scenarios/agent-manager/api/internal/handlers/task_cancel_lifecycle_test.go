package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func taskCancellationAsVerifiedHuman(req *http.Request) *http.Request {
	principal := coreidentity.Principal{
		Kind: coreidentity.ActorHuman, Subject: "fixture-owner", Verified: true,
		Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour),
	}
	return req.WithContext(coreidentity.WithPrincipal(req.Context(), principal))
}

// TestCancelTask_NotFound reaches the task lookup only after supplying the
// verified operator authority required by the endpoint.
func TestCancelTask_NotFound(t *testing.T) {
	_, router := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/cancel", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, taskCancellationAsVerifiedHuman(req))
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestTaskCancellationAndDeletionLifecycle(t *testing.T) {
	_, router := setupTestHandler(t)
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: &domain.Task{Title: "cleanup", ScopePath: "src/cleanup"}})
	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created apipb.CreateTaskResponse
	decodeProtoJSON(t, create.Body.Bytes(), &created)
	id := created.Task.GetId()

	deleteBeforeCancel := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+id, nil)
	router.ServeHTTP(deleteBeforeCancel, taskCancellationAsVerifiedHuman(deleteReq))
	if deleteBeforeCancel.Code != http.StatusConflict {
		t.Fatalf("uncancelled delete status=%d body=%s", deleteBeforeCancel.Code, deleteBeforeCancel.Body.String())
	}
	cancel := httptest.NewRecorder()
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/cancel", nil)
	router.ServeHTTP(cancel, taskCancellationAsVerifiedHuman(cancelReq))
	if cancel.Code != http.StatusOK {
		t.Fatalf("cancel status=%d body=%s", cancel.Code, cancel.Body.String())
	}
	remove := httptest.NewRecorder()
	removeReq := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+id, nil)
	router.ServeHTTP(remove, taskCancellationAsVerifiedHuman(removeReq))
	if remove.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", remove.Code, remove.Body.String())
	}
	var deleted apipb.DeleteTaskResponse
	decodeProtoJSON(t, remove.Body.Bytes(), &deleted)
	if !deleted.Success {
		t.Fatalf("delete response=%+v", &deleted)
	}
}

// TestCancelTask_Success keeps the legacy cancellation behavior under the
// explicit verified-human route used by the owner contract.
func TestCancelTask_Success(t *testing.T) {
	_, router := setupTestHandler(t)
	body := encodeProtoJSON(t, &apipb.CreateTaskRequest{Task: &domain.Task{Title: "To cancel", ScopePath: "src/"}})
	create := httptest.NewRecorder()
	router.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created apipb.CreateTaskResponse
	decodeProtoJSON(t, create.Body.Bytes(), &created)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+created.Task.GetId()+"/cancel", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, taskCancellationAsVerifiedHuman(req))
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
}
