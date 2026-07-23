package eventlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type handlerRepository struct {
	entityType EntityType
	entityID   string
	after      int64
	limit      int
}

func (r *handlerRepository) Append(context.Context, Event) (int64, error)       { return 0, nil }
func (r *handlerRepository) Since(context.Context, int64, int) ([]Event, error) { return nil, nil }
func (r *handlerRepository) All(context.Context) ([]Event, error)               { return nil, nil }
func (r *handlerRepository) MaxID(context.Context) (int64, error)               { return 0, nil }
func (r *handlerRepository) QueryByEntity(_ context.Context, entityType EntityType, entityID string, after int64, limit int) ([]Event, error) {
	r.entityType, r.entityID, r.after, r.limit = entityType, entityID, after, limit
	return []Event{{ID: 9, EntityType: entityType, EntityID: entityID, EventType: EventBacklogStatusChanged}}, nil
}

func TestQueryByEntityParsesBacklogLocatorAndCursor(t *testing.T) {
	repo := &handlerRepository{}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/events?entity=backlog/execute/item-a&since_id=8&limit=25", nil)
	response := httptest.NewRecorder()
	NewHandler(repo).QueryByEntity(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if repo.entityType != EntityBacklogItem || repo.entityID != "execute/item-a" || repo.after != 8 || repo.limit != 25 {
		t.Fatalf("query=(%q,%q,%d,%d)", repo.entityType, repo.entityID, repo.after, repo.limit)
	}
}

func TestQueryByEntityRejectsInvalidEntity(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(&handlerRepository{}).QueryByEntity(response, httptest.NewRequest(http.MethodGet, "/api/v1/events?entity=execution/abc", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
