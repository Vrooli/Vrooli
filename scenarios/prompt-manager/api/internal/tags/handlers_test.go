package tags

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeTagRepository struct {
	tags    []Tag
	created *Tag
}

func (r *fakeTagRepository) GetAll() ([]Tag, error) {
	return r.tags, nil
}

func (r *fakeTagRepository) Create(tag *Tag) error {
	r.created = tag
	return nil
}

func TestListReturnsTags(t *testing.T) {
	repo := &fakeTagRepository{tags: []Tag{{ID: "tag-1", Name: "Core"}}}
	handler := NewHandlers(repo)
	rec := httptest.NewRecorder()

	handler.List(rec, httptest.NewRequest(http.MethodGet, "/tags", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var tags []Tag
	if err := json.NewDecoder(rec.Body).Decode(&tags); err != nil {
		t.Fatalf("decode tags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "Core" {
		t.Fatalf("unexpected tags: %+v", tags)
	}
}

func TestCreateRejectsMissingName(t *testing.T) {
	handler := NewHandlers(&fakeTagRepository{})
	rec := httptest.NewRecorder()

	handler.Create(rec, httptest.NewRequest(http.MethodPost, "/tags", strings.NewReader(`{"name":""}`)))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateRejectsDuplicateNameWithoutMutation(t *testing.T) {
	repo := &fakeTagRepository{tags: []Tag{{ID: "tag-1", Name: "Testing"}}}
	handler := NewHandlers(repo)
	rec := httptest.NewRecorder()

	handler.Create(rec, httptest.NewRequest(http.MethodPost, "/tags", strings.NewReader(`{"name":" testing "}`)))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.created != nil {
		t.Fatalf("duplicate tag was persisted: %+v", repo.created)
	}
}
