package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ============================================================================
// isOriginAllowed Tests
// ============================================================================

func TestIsOriginAllowed_NilHandler(t *testing.T) {
	var h *Handler = nil
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://example.com")

	if h.isOriginAllowed(req) {
		t.Error("expected isOriginAllowed to return false for nil handler")
	}
}

func TestIsOriginAllowed_AllowAll(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()
	handler.wsAllowAll = true

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://any-origin.com")

	if !handler.isOriginAllowed(req) {
		t.Error("expected isOriginAllowed to return true when wsAllowAll is true")
	}
}

func TestIsOriginAllowed_EmptyOrigin(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()
	handler.wsAllowAll = false
	handler.wsAllowedOrigins = []string{"http://allowed.com"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	// No Origin header set

	if !handler.isOriginAllowed(req) {
		t.Error("expected isOriginAllowed to return true for requests without Origin header")
	}
}

func TestIsOriginAllowed_MatchingOrigin(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()
	handler.wsAllowAll = false
	handler.wsAllowedOrigins = []string{"http://allowed.com", "http://also-allowed.com"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://allowed.com")

	if !handler.isOriginAllowed(req) {
		t.Error("expected isOriginAllowed to return true for matching origin")
	}
}

func TestIsOriginAllowed_CaseInsensitiveMatch(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()
	handler.wsAllowAll = false
	handler.wsAllowedOrigins = []string{"http://ALLOWED.com"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://allowed.com")

	if !handler.isOriginAllowed(req) {
		t.Error("expected isOriginAllowed to match case-insensitively")
	}
}

func TestIsOriginAllowed_NonMatchingOrigin(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()
	handler.wsAllowAll = false
	handler.wsAllowedOrigins = []string{"http://allowed.com"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://not-allowed.com")

	if handler.isOriginAllowed(req) {
		t.Error("expected isOriginAllowed to return false for non-matching origin")
	}
}

func TestIsOriginAllowed_WhitespaceOrigin(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()
	handler.wsAllowAll = false
	handler.wsAllowedOrigins = []string{"http://allowed.com"}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "   ") // Only whitespace

	// Trimmed whitespace becomes empty string, should be allowed
	if !handler.isOriginAllowed(req) {
		t.Error("expected isOriginAllowed to return true for whitespace-only origin")
	}
}

// ============================================================================
// GetExecutionService Tests
// ============================================================================

func TestGetExecutionService_ReturnsService(t *testing.T) {
	handler, _, execSvc, _, _, _ := createTestHandler()

	result := handler.GetExecutionService()

	if result == nil {
		t.Fatal("expected GetExecutionService to return non-nil service")
	}
	if result != execSvc {
		t.Error("expected GetExecutionService to return the execution service")
	}
}

func TestGetExecutionService_NilService(t *testing.T) {
	handler := &Handler{
		executionService: nil,
	}

	result := handler.GetExecutionService()

	if result != nil {
		t.Error("expected GetExecutionService to return nil when service is nil")
	}
}

// ============================================================================
// GetPerfRegistry Tests
// ============================================================================

func TestGetPerfRegistry_ReturnsNilWhenNotSet(t *testing.T) {
	handler := &Handler{
		perfRegistry: nil,
	}

	result := handler.GetPerfRegistry()

	if result != nil {
		t.Error("expected GetPerfRegistry to return nil when not set")
	}
}

// ============================================================================
// Config Function Tests
// ============================================================================

func TestRecordingUploadLimitBytes_ReturnsConfiguredLimit(t *testing.T) {
	// This function reads from config, just verify it doesn't panic
	limit := recordingUploadLimitBytes()
	if limit <= 0 {
		t.Error("expected recordingUploadLimitBytes to return positive value")
	}
}

func TestRecordingImportTimeout_ReturnsConfiguredTimeout(t *testing.T) {
	// This function reads from config, just verify it doesn't panic
	timeout := recordingImportTimeout()
	if timeout <= 0 {
		t.Error("expected recordingImportTimeout to return positive duration")
	}
}

func TestEventBufferLimits_ReturnsValidLimits(t *testing.T) {
	// This function reads from config, just verify it doesn't panic
	limits := eventBufferLimits()
	if limits.PerExecution < 0 {
		t.Error("expected eventBufferLimits to return non-negative PerExecution")
	}
}

// ============================================================================
// AI Handler Delegation Tests
// ============================================================================

func TestTakePreviewScreenshot_DelegatesToHandler(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Initialize AI subhandlers (nil in basic test handler)
	// These will panic if called on nil, but we just want to verify delegation
	// In production, these are initialized in NewHandlerWithDeps

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/screenshot", nil)
	rr := httptest.NewRecorder()

	// This will panic because screenshotHandler is nil, but that's expected
	// We're testing that the method exists and attempts delegation
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when screenshotHandler is nil")
		}
	}()

	handler.TakePreviewScreenshot(rr, req)
}

func TestGetDOMTree_DelegatesToHandler(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/dom", nil)
	rr := httptest.NewRecorder()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when domHandler is nil")
		}
	}()

	handler.GetDOMTree(rr, req)
}

func TestAnalyzeElements_DelegatesToHandler(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/elements", nil)
	rr := httptest.NewRecorder()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when elementAnalysisHandler is nil")
		}
	}()

	handler.AnalyzeElements(rr, req)
}

func TestGetElementAtCoordinate_DelegatesToHandler(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/element-at", nil)
	rr := httptest.NewRecorder()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when elementAnalysisHandler is nil")
		}
	}()

	handler.GetElementAtCoordinate(rr, req)
}

func TestAIAnalyzeElements_DelegatesToHandler(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/analyze", nil)
	rr := httptest.NewRecorder()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when aiAnalysisHandler is nil")
		}
	}()

	handler.AIAnalyzeElements(rr, req)
}
