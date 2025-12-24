package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// withWildcardParam adds a chi wildcard URL parameter to the request context
func withWildcardParam(r *http.Request, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("*", value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ============================================================================
// ServeScreenshot Tests
// ============================================================================

func TestServeScreenshot_Success(t *testing.T) {
	handler, _, _, _, _, storageMock := createTestHandler()

	// Add a screenshot to the mock storage
	executionID := uuid.New()
	objectName := executionID.String() + "/step-0.png"
	storageMock.screenshots[objectName] = []byte("fake-png-data")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/screenshots/"+objectName, nil)
	req = withWildcardParam(req, objectName)
	rr := httptest.NewRecorder()

	handler.ServeScreenshot(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify content was served
	if rr.Body.Len() == 0 {
		t.Fatal("expected response body to contain screenshot data")
	}
}

func TestServeScreenshot_MissingPath(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/screenshots/", nil)
	req = withWildcardParam(req, "")
	rr := httptest.NewRecorder()

	handler.ServeScreenshot(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing path, got %d", rr.Code)
	}
}

func TestServeScreenshot_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/screenshots/nonexistent/screenshot.png", nil)
	req = withWildcardParam(req, "nonexistent/screenshot.png")
	rr := httptest.NewRecorder()

	handler.ServeScreenshot(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestServeScreenshot_StorageUnavailable(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Set storage to nil to simulate unavailable storage
	handler.storage = nil

	req := httptest.NewRequest(http.MethodGet, "/api/v1/screenshots/test/screenshot.png", nil)
	req = withWildcardParam(req, "test/screenshot.png")
	rr := httptest.NewRecorder()

	handler.ServeScreenshot(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 for unavailable storage, got %d", rr.Code)
	}
}

func TestServeScreenshot_LeadingSlashRemoved(t *testing.T) {
	handler, _, _, _, _, storageMock := createTestHandler()

	// Add screenshot without leading slash
	objectName := "execution-123/step-0.png"
	storageMock.screenshots[objectName] = []byte("fake-png-data")

	// Request with leading slash
	req := httptest.NewRequest(http.MethodGet, "/api/v1/screenshots//"+objectName, nil)
	req = withWildcardParam(req, "/"+objectName) // With leading slash
	rr := httptest.NewRecorder()

	handler.ServeScreenshot(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 (leading slash should be trimmed), got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// ServeThumbnail Tests
// ============================================================================

func TestServeThumbnail_Success(t *testing.T) {
	handler, _, _, _, _, storageMock := createTestHandler()

	// Add a screenshot for thumbnail
	objectName := "test-execution/thumbnail.png"
	storageMock.screenshots[objectName] = []byte("fake-thumbnail-data")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/screenshots/thumbnail/"+objectName, nil)
	req = withWildcardParam(req, objectName)
	rr := httptest.NewRecorder()

	handler.ServeThumbnail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServeThumbnail_NotFound(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/screenshots/thumbnail/nonexistent.png", nil)
	req = withWildcardParam(req, "nonexistent.png")
	rr := httptest.NewRecorder()

	handler.ServeThumbnail(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

// ============================================================================
// Cache Control Tests
// ============================================================================

func TestServeScreenshot_CacheHeaders(t *testing.T) {
	handler, _, _, _, _, storageMock := createTestHandler()

	objectName := "cached-execution/step.png"
	storageMock.screenshots[objectName] = []byte("cached-data")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/screenshots/"+objectName, nil)
	req = withWildcardParam(req, objectName)
	rr := httptest.NewRecorder()

	handler.ServeScreenshot(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	cacheControl := rr.Header().Get("Cache-Control")
	if cacheControl == "" {
		t.Fatal("expected Cache-Control header to be set")
	}
	if cacheControl != "public, max-age=3600" {
		t.Fatalf("expected 'public, max-age=3600', got %s", cacheControl)
	}
}
