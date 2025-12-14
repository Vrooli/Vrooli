package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleAdminResetDemoData_Success(t *testing.T) {
	db := setupTestDB(t)
	server := &Server{db: db}

	var customVariantID int64
	err := db.QueryRow(`
		INSERT INTO variants (slug, name, description, weight, status)
		VALUES ('custom-reset', 'Custom Reset', 'temp', 10, 'active')
		RETURNING id
	`).Scan(&customVariantID)
	if err != nil {
		t.Fatalf("failed to insert custom variant: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO content_sections (variant_id, section_type, content, "order", enabled)
		VALUES ($1, 'hero', '{"headline":"Temp"}'::jsonb, 1, TRUE)
	`, customVariantID); err != nil {
		t.Fatalf("failed to seed custom section: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/reset-demo-data", nil)
	resp := httptest.NewRecorder()
	server.handleAdminResetDemoData(resp, req)

	if resp.Code != http.StatusOK {
		body := resp.Body.String()
		t.Fatalf("expected status 200, got %d: %s", resp.Code, body)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid reset response json: %v", err)
	}
	if payload["reset"] != true {
		t.Fatalf("expected reset flag true, got %v", payload["reset"])
	}

	var customCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM variants WHERE slug = 'custom-reset'`).Scan(&customCount); err != nil {
		t.Fatalf("failed to count custom variants: %v", err)
	}
	if customCount != 0 {
		t.Fatalf("custom variant should be removed after reset, still %d rows", customCount)
	}

	var controlCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM variants WHERE slug = 'control'`).Scan(&controlCount); err != nil {
		t.Fatalf("failed to count control variant: %v", err)
	}
	if controlCount == 0 {
		t.Fatalf("control variant missing after reset")
	}

	var downloadApps int
	if err := db.QueryRow(`SELECT COUNT(*) FROM download_apps`).Scan(&downloadApps); err != nil {
		t.Fatalf("failed to count download apps: %v", err)
	}
	if downloadApps == 0 {
		t.Fatalf("expected download apps seeded after reset")
	}
}

func TestHandleAdminResetDemoData_ResetErrorReturns500(t *testing.T) {
	db := setupTestDB(t)
	server := &Server{db: db}

	// Force a reset failure by closing the DB before invoking the handler.
	db.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/reset-demo-data", nil)
	resp := httptest.NewRecorder()

	server.handleAdminResetDemoData(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 when reset fails, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "failed to reset") {
		t.Fatalf("expected failure body, got %q", resp.Body.String())
	}
}
