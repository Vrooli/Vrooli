package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	adminhttp "landing-page-business-suite-api/handlers/admin"
)

// NOTE: TestHandleAdminResetDemoData_Success has been removed - depends on variants table which was removed
// Variant configuration is now stored in JSON files and managed via ConfigStore

func TestHandleAdminResetDemoData_ResetErrorReturns500(t *testing.T) {
	db := setupTestDB(t)
	server := &Server{db: db}

	// Force a reset failure by closing the DB before invoking the handler.
	db.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/reset-demo-data", nil)
	resp := httptest.NewRecorder()

	adminhttp.ResetDemoData(adminhttp.ResetDependencies{Reset: server.resetDemoData, Now: time.Now, LogError: logStructuredError}).ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 when reset fails, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "failed to reset") {
		t.Fatalf("expected failure body, got %q", resp.Body.String())
	}
}
