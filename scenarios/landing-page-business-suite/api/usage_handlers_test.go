package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	billinghttp "landing-page-business-suite-api/handlers/commerce"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/testutil"
)

// requestWithUserAuth creates a test request with user claims injected into the context.
// This simulates what the requireUserAuth middleware does.
func requestWithUserAuth(method, url string, body []byte, userEmail string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	// Inject user claims into context (simulating authenticated user)
	claims := &administration.UserClaims{
		Email:  userEmail,
		UserID: "test-user-id",
	}
	ctx := context.WithValue(req.Context(), userClaimsKey, claims)
	return req.WithContext(ctx)
}

func usageReportHandler(svc *commerce.UsageService) http.HandlerFunc {
	deps := usageHTTPDependencies()
	return billinghttp.RequireUsageServiceAuth(svc, deps, billinghttp.ReportUsage(svc, deps))
}

func usageCheckHandler(svc *commerce.UsageService) http.HandlerFunc {
	return billinghttp.CheckLimit(svc, usageHTTPDependencies())
}

func usageSummaryHandler(svc *commerce.UsageService) http.HandlerFunc {
	return billinghttp.GetUsageSummary(svc, nil, usageHTTPDependencies())
}

// ============================================================================
// POST /api/v1/usage/report Handler Tests
// ============================================================================

func TestHandleReportUsage_ValidRequest_Returns200(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
	defer db.Close()

	handler := usageReportHandler(svc)

	body := commerce.UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret-token")

	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	// Verify DB record was created
	var usageAmount int64
	err := db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "user@example.com", "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}
	if usageAmount != 100000 {
		t.Errorf("Expected usage_amount 100000, got %d", usageAmount)
	}
}

func TestHandleReportUsage_MissingAuthHeader_Returns401(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
	defer db.Close()

	handler := usageReportHandler(svc)

	body := commerce.UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusUnauthorized)
}

func TestHandleReportUsage_InvalidAuthToken_Returns401(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
	defer db.Close()

	handler := usageReportHandler(svc)

	body := commerce.UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")

	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusUnauthorized)
}

func TestHandleReportUsage_AuthWithoutBearerPrefix_Returns401(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
	defer db.Close()

	handler := usageReportHandler(svc)

	body := commerce.UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "test-secret-token") // Missing "Bearer " prefix

	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusUnauthorized)
}

func TestHandleReportUsage_MalformedJSON_Returns400(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
	defer db.Close()

	handler := usageReportHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader([]byte("not valid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret-token")

	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleReportUsage_MissingRequiredFields_Returns400(t *testing.T) {
	testCases := []struct {
		name string
		body commerce.UsageReportRequest
	}{
		{
			name: "missing user_identity",
			body: commerce.UsageReportRequest{
				UserIdentity: "",
				LimitKey:     "ai_credits",
				Amount:       100000,
			},
		},
		{
			name: "missing limit_key",
			body: commerce.UsageReportRequest{
				UserIdentity: "user@example.com",
				LimitKey:     "",
				Amount:       100000,
			},
		},
		{
			name: "zero amount without BYOK",
			body: commerce.UsageReportRequest{
				UserIdentity: "user@example.com",
				LimitKey:     "ai_credits",
				Amount:       0,
				IsBYOK:       false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
			defer db.Close()

			handler := usageReportHandler(svc)

			bodyBytes, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-secret-token")

			w := httptest.NewRecorder()
			handler(w, req)

			testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
		})
	}
}

func TestHandleReportUsage_BYOK_RecordsZeroAmount(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
	defer db.Close()

	handler := usageReportHandler(svc)

	body := commerce.UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000, // Should be ignored
		AppBundleKey: "browser-automation-studio",
		IsBYOK:       true,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret-token")

	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	// Verify DB record was created with 0 amount
	var usageAmount int64
	err := db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "user@example.com", "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}
	if usageAmount != 0 {
		t.Errorf("Expected usage_amount 0 for BYOK, got %d", usageAmount)
	}
}

// ============================================================================
// GET /api/v1/usage/check Handler Tests
// ============================================================================

func TestHandleCheckLimit_ValidCheck_ReturnsCanProceedAndRemaining(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	// Seed some usage (below limit)
	currentPeriod := getCurrentBillingPeriodTest()
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount)
		VALUES (?, ?, ?, ?)
	`, "user@example.com", currentPeriod, "ai_credits", 100000000) // 100 million (below solo's $5 = 500 million)
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	handler := usageCheckHandler(svc)

	req := requestWithUserAuth(http.MethodGet, "/api/v1/usage/check?tier=solo&limit_key=ai_credits", nil, "user@example.com")
	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	if resp["can_proceed"] != true {
		t.Errorf("Expected can_proceed=true, got %v", resp["can_proceed"])
	}
	// remaining should be 500000000 - 100000000 = 400000000
	if remaining, ok := resp["remaining"].(float64); !ok || int64(remaining) != 400000000 {
		t.Errorf("Expected remaining=400000000, got %v", resp["remaining"])
	}
}

func TestHandleCheckLimit_UserAtLimit_ReturnsCanProceedFalse(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	// Seed usage at the limit
	currentPeriod := getCurrentBillingPeriodTest()
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount)
		VALUES (?, ?, ?, ?)
	`, "user@example.com", currentPeriod, "ai_credits", 500000000) // At solo limit
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	handler := usageCheckHandler(svc)

	req := requestWithUserAuth(http.MethodGet, "/api/v1/usage/check?tier=solo&limit_key=ai_credits", nil, "user@example.com")
	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	if resp["can_proceed"] != false {
		t.Errorf("Expected can_proceed=false, got %v", resp["can_proceed"])
	}
	if remaining, ok := resp["remaining"].(float64); !ok || int64(remaining) != 0 {
		t.Errorf("Expected remaining=0, got %v", resp["remaining"])
	}
}

func TestHandleCheckLimit_MissingParams_Returns400(t *testing.T) {
	t.Run("missing limit_key", func(t *testing.T) {
		svc, _, db := createTestUsageService(t)
		defer db.Close()

		handler := usageCheckHandler(svc)

		req := requestWithUserAuth(http.MethodGet, "/api/v1/usage/check?tier=solo", nil, "user@example.com")
		w := httptest.NewRecorder()
		handler(w, req)

		testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
	})
}

func TestHandleCheckLimit_Unauthenticated_Returns401(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	handler := usageCheckHandler(svc)

	// Request without auth context
	req := httptest.NewRequest(http.MethodGet, "/api/v1/usage/check?tier=solo&limit_key=ai_credits", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusUnauthorized)
}

func TestHandleCheckLimit_UnlimitedTier_ReturnsNegativeOneRemaining(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	handler := usageCheckHandler(svc)

	req := requestWithUserAuth(http.MethodGet, "/api/v1/usage/check?tier=business&limit_key=ai_credits", nil, "user@example.com")
	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	if resp["can_proceed"] != true {
		t.Errorf("Expected can_proceed=true for unlimited tier, got %v", resp["can_proceed"])
	}
	if remaining, ok := resp["remaining"].(float64); !ok || int64(remaining) != -1 {
		t.Errorf("Expected remaining=-1 (unlimited), got %v", resp["remaining"])
	}
}

// ============================================================================
// GET /api/v1/usage/summary Handler Tests
// ============================================================================

func TestHandleGetUsageSummary_ReturnsCorrectValues(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	// Seed usage from multiple apps
	currentPeriod := getCurrentBillingPeriodTest()
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)
	`,
		"user@example.com", currentPeriod, "ai_credits", 100000000, "browser-automation-studio",
		"user@example.com", currentPeriod, "ai_credits", 50000000, "other-app",
	)
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	handler := usageSummaryHandler(svc)

	req := requestWithUserAuth(http.MethodGet, "/api/v1/usage/summary?tier=solo", nil, "user@example.com")
	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp commerce.UsageSummary
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	// Total usage should be 100000000 + 50000000 = 150000000
	if resp.Usage["ai_credits"] != 150000000 {
		t.Errorf("Expected usage 150000000, got %d", resp.Usage["ai_credits"])
	}

	// Remaining should be 500000000 - 150000000 = 350000000
	if resp.Remaining["ai_credits"] != 350000000 {
		t.Errorf("Expected remaining 350000000, got %d", resp.Remaining["ai_credits"])
	}

	// By-app breakdown
	if resp.ByApp["browser-automation-studio"] != 100000000 {
		t.Errorf("Expected browser-automation-studio usage 100000000, got %d", resp.ByApp["browser-automation-studio"])
	}
	if resp.ByApp["other-app"] != 50000000 {
		t.Errorf("Expected other-app usage 50000000, got %d", resp.ByApp["other-app"])
	}
}

func TestHandleGetUsageSummary_NewUser_ReturnsEmptyUsage(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	handler := usageSummaryHandler(svc)

	req := requestWithUserAuth(http.MethodGet, "/api/v1/usage/summary?tier=solo", nil, "newuser@example.com")
	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp commerce.UsageSummary
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	// New user should have 0 usage
	if resp.Usage["ai_credits"] != 0 {
		t.Errorf("Expected usage 0 for new user, got %d", resp.Usage["ai_credits"])
	}

	// Full remaining balance (solo tier = 500000000)
	if resp.Remaining["ai_credits"] != 500000000 {
		t.Errorf("Expected remaining 500000000, got %d", resp.Remaining["ai_credits"])
	}
}

// ============================================================================
// Service Token Edge Cases
// ============================================================================

func TestHandleReportUsage_EmptyConfiguredToken_RejectsAll(t *testing.T) {
	// When no token is configured, all requests should be rejected
	svc, _, db := createTestUsageServiceWithToken(t, "") // Empty token
	defer db.Close()

	handler := usageReportHandler(svc)

	body := commerce.UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer any-token")

	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusUnauthorized)
}

func TestHandleReportUsage_WhitespaceOnlyToken_Rejected(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
	defer db.Close()

	handler := usageReportHandler(svc)

	body := commerce.UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer    ") // Whitespace only

	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusUnauthorized)
}

// ============================================================================
// UsageReportRequest Metadata Tests
// ============================================================================

func TestHandleReportUsage_WithMetadata(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
	defer db.Close()

	handler := usageReportHandler(svc)

	body := commerce.UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
		Operation:    "ai.workflow_generate",
		Metadata: map[string]string{
			"model":         "gpt-4",
			"prompt_tokens": "1500",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret-token")

	w := httptest.NewRecorder()
	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	// Metadata is logged but not stored - just verify the request succeeded
	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)
	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
}

// ============================================================================
// Response Format Verification
// ============================================================================

func TestHandleReportUsage_ResponseFormat(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "test-secret-token")
	defer db.Close()

	handler := usageReportHandler(svc)

	body := commerce.UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/usage/report", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret-token")

	w := httptest.NewRecorder()
	handler(w, req)

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify response is valid JSON
	var resp map[string]interface{}
	assertJSONResponse(t, w.Body.Bytes(), &resp)
}

func TestHandleCheckLimit_ResponseFormat(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	handler := usageCheckHandler(svc)

	req := requestWithUserAuth(http.MethodGet, "/api/v1/usage/check?tier=solo&limit_key=ai_credits", nil, "user@example.com")
	w := httptest.NewRecorder()
	handler(w, req)

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify response structure
	var resp map[string]interface{}
	assertJSONResponse(t, w.Body.Bytes(), &resp)

	// Required fields
	if _, ok := resp["can_proceed"]; !ok {
		t.Error("Response missing 'can_proceed' field")
	}
	if _, ok := resp["remaining"]; !ok {
		t.Error("Response missing 'remaining' field")
	}
	if _, ok := resp["limit_key"]; !ok {
		t.Error("Response missing 'limit_key' field")
	}
}
