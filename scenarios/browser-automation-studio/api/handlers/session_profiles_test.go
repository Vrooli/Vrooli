package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
)

// createTestHandlerWithSessionProfiles creates a handler with a real SessionProfileStore in a temp directory.
func createTestHandlerWithSessionProfiles(t *testing.T) (*Handler, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "session-profiles-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	store := archiveingestion.NewSessionProfileStore(tempDir, log)

	handler := &Handler{
		sessionProfiles: store,
		log:             log,
	}

	return handler, tempDir
}

// ============================================================================
// ListRecordingSessionProfiles Tests
// ============================================================================

func TestListRecordingSessionProfiles_Success_Empty(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/session-profiles", nil)
	rr := httptest.NewRecorder()

	handler.ListRecordingSessionProfiles(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string][]sessionProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	profiles := response["profiles"]
	if len(profiles) != 0 {
		t.Fatalf("expected 0 profiles, got %d", len(profiles))
	}
}

func TestListRecordingSessionProfiles_Success_WithProfiles(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	// Create some profiles
	_, err := handler.sessionProfiles.Create("Profile 1")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}
	_, err = handler.sessionProfiles.Create("Profile 2")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/session-profiles", nil)
	rr := httptest.NewRecorder()

	handler.ListRecordingSessionProfiles(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string][]sessionProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	profiles := response["profiles"]
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
}

func TestListRecordingSessionProfiles_ServiceUnavailable(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := &Handler{
		sessionProfiles: nil, // No store configured
		log:             log,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/session-profiles", nil)
	rr := httptest.NewRecorder()

	handler.ListRecordingSessionProfiles(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

// ============================================================================
// CreateRecordingSessionProfile Tests
// ============================================================================

func TestCreateRecordingSessionProfile_Success(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	body := `{"name": "My Test Profile"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session-profiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var profile sessionProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &profile); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if profile.Name != "My Test Profile" {
		t.Fatalf("expected name 'My Test Profile', got %q", profile.Name)
	}
	if profile.ID == "" {
		t.Fatal("expected profile ID to be set")
	}
}

func TestCreateRecordingSessionProfile_EmptyName(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session-profiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var profile sessionProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &profile); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should get an auto-generated name
	if profile.Name == "" {
		t.Fatal("expected auto-generated name")
	}
}

func TestCreateRecordingSessionProfile_InvalidJSON(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session-profiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateRecordingSessionProfile_ServiceUnavailable(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := &Handler{
		sessionProfiles: nil,
		log:             log,
	}

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session-profiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.CreateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

// ============================================================================
// UpdateRecordingSessionProfile Tests
// ============================================================================

func TestUpdateRecordingSessionProfile_Success(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	// Create a profile first
	profile, err := handler.sessionProfiles.Create("Original Name")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	body := `{"name": "Updated Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/session-profiles/"+profile.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", profile.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.UpdateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var updated sessionProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Fatalf("expected name 'Updated Name', got %q", updated.Name)
	}
}

func TestUpdateRecordingSessionProfile_MissingProfileID(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	body := `{"name": "Updated Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/session-profiles/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.UpdateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestUpdateRecordingSessionProfile_MissingName(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	profile, err := handler.sessionProfiles.Create("Original")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	body := `{"name": ""}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/session-profiles/"+profile.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", profile.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.UpdateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestUpdateRecordingSessionProfile_NotFound(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	body := `{"name": "Updated Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/session-profiles/nonexistent-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", "nonexistent-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.UpdateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestUpdateRecordingSessionProfile_InvalidJSON(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	profile, err := handler.sessionProfiles.Create("Original")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/session-profiles/"+profile.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", profile.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.UpdateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestUpdateRecordingSessionProfile_ServiceUnavailable(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := &Handler{
		sessionProfiles: nil,
		log:             log,
	}

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/session-profiles/some-id", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", "some-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.UpdateRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

// ============================================================================
// DeleteRecordingSessionProfile Tests
// ============================================================================

func TestDeleteRecordingSessionProfile_Success(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	// Create a profile first
	profile, err := handler.sessionProfiles.Create("To Delete")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/session-profiles/"+profile.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", profile.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.DeleteRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify it's gone
	_, err = handler.sessionProfiles.Get(profile.ID)
	if err == nil {
		t.Fatal("expected profile to be deleted")
	}
}

func TestDeleteRecordingSessionProfile_MissingProfileID(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/session-profiles/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.DeleteRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestDeleteRecordingSessionProfile_NotFound(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/session-profiles/nonexistent-id", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", "nonexistent-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.DeleteRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestDeleteRecordingSessionProfile_ServiceUnavailable(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := &Handler{
		sessionProfiles: nil,
		log:             log,
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/session-profiles/some-id", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", "some-id")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.DeleteRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

func TestDeleteRecordingSessionProfile_ClearsActiveSessions(t *testing.T) {
	handler, tempDir := createTestHandlerWithSessionProfiles(t)
	defer os.RemoveAll(tempDir)

	// Create a profile and associate a session with it
	profile, err := handler.sessionProfiles.Create("To Delete")
	if err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	// Associate a browser session with this profile
	handler.sessionProfiles.SetActiveSession("browser-session-123", profile.ID)

	// Verify the association exists
	if got := handler.sessionProfiles.GetActiveSession("browser-session-123"); got != profile.ID {
		t.Fatalf("expected active session to be associated with profile, got %q", got)
	}

	// Delete the profile
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/session-profiles/"+profile.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileId", profile.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	handler.DeleteRecordingSessionProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify the session association was cleared
	if got := handler.sessionProfiles.GetActiveSession("browser-session-123"); got != "" {
		t.Fatalf("expected active session to be cleared, got %q", got)
	}
}
