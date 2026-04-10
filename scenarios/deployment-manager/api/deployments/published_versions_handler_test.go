package deployments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// mockPublishedVersionsRepo implements PublishedVersionsRepository for testing.
type mockPublishedVersionsRepo struct {
	records       []PublishedVersion
	recordErr     error
	latestResult  []PublishedVersion
	latestErr     error
	historyResult []PublishedVersion
	historyErr    error
}

func (m *mockPublishedVersionsRepo) RecordPublish(_ context.Context, record *PublishedVersion) error {
	if m.recordErr != nil {
		return m.recordErr
	}
	record.ID = len(m.records) + 1
	m.records = append(m.records, *record)
	return nil
}

func (m *mockPublishedVersionsRepo) GetLatestByProfile(_ context.Context, _ string) ([]PublishedVersion, error) {
	return m.latestResult, m.latestErr
}

func (m *mockPublishedVersionsRepo) GetHistory(_ context.Context, _, _ string, _ int) ([]PublishedVersion, error) {
	return m.historyResult, m.historyErr
}

func TestGetPublishedVersions_Latest(t *testing.T) {
	now := time.Now()
	repo := &mockPublishedVersionsRepo{
		latestResult: []PublishedVersion{
			{ID: 1, ProfileID: "prof-1", Platform: "windows", Version: "1.2.3", PublishedAt: now},
			{ID: 2, ProfileID: "prof-1", Platform: "linux", Version: "1.2.3", PublishedAt: now},
		},
	}
	handler := NewPublishedVersionsHandler(repo, func(string, map[string]interface{}) {})

	req := httptest.NewRequest("GET", "/api/v1/profiles/prof-1/published-versions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "prof-1"})
	w := httptest.NewRecorder()

	handler.GetPublishedVersions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	versions := resp["versions"].([]interface{})
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if resp["profile_id"] != "prof-1" {
		t.Errorf("expected profile_id prof-1, got %v", resp["profile_id"])
	}
}

func TestGetPublishedVersions_History(t *testing.T) {
	now := time.Now()
	repo := &mockPublishedVersionsRepo{
		historyResult: []PublishedVersion{
			{ID: 3, ProfileID: "prof-1", Platform: "windows", Version: "1.2.3", PublishedAt: now},
			{ID: 1, ProfileID: "prof-1", Platform: "windows", Version: "1.2.2", PublishedAt: now.Add(-time.Hour)},
		},
	}
	handler := NewPublishedVersionsHandler(repo, func(string, map[string]interface{}) {})

	req := httptest.NewRequest("GET", "/api/v1/profiles/prof-1/published-versions?history=true&platform=windows&limit=10", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "prof-1"})
	w := httptest.NewRecorder()

	handler.GetPublishedVersions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	versions := resp["versions"].([]interface{})
	if len(versions) != 2 {
		t.Fatalf("expected 2 history versions, got %d", len(versions))
	}
}

func TestGetPublishedVersions_EmptyResult(t *testing.T) {
	repo := &mockPublishedVersionsRepo{
		latestResult: nil,
	}
	handler := NewPublishedVersionsHandler(repo, func(string, map[string]interface{}) {})

	req := httptest.NewRequest("GET", "/api/v1/profiles/prof-1/published-versions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "prof-1"})
	w := httptest.NewRecorder()

	handler.GetPublishedVersions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	versions := resp["versions"].([]interface{})
	if len(versions) != 0 {
		t.Fatalf("expected empty versions, got %d", len(versions))
	}
}
