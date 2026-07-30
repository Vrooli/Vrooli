package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	downloadhttp "landing-page-business-suite-api/handlers/delivery"
	"landing-page-business-suite-api/internal/administration"
)

func TestHandleDownloads_Success(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_dl", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_dl_test", "Pro Plan", "pro", "month", "usd", 4900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	// Create subscription for the user
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key)
		VALUES ('sub_dl_test', 'cus_dl', 'downloads@example.com', 'active', 'pro', 'price_dl_test', 'business_suite')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test subscription: %v", err)
	}

	// Create downloadable app configuration
	_, err = db.Exec(`
		INSERT INTO download_apps (bundle_key, app_key, name)
		VALUES ('business_suite', 'test_app', 'Test App')
	`)
	if err != nil {
		t.Fatalf("Failed to insert download app: %v", err)
	}

	// Create download asset
	_, err = db.Exec(`
		INSERT INTO download_assets (bundle_key, app_key, platform, artifact_url, release_version, release_notes, checksum)
		VALUES ('business_suite', 'test_app', 'windows', 'https://example.com/download.exe', '1.0.0', 'Initial release', 'abc123')
	`)
	if err != nil {
		t.Fatalf("Failed to insert download asset: %v", err)
	}

	planService := NewPlanService(db)
	accountService := NewAccountService(db, planService)
	downloadService := NewDownloadService(db)
	authorizer := NewDownloadAuthorizer(downloadService, accountService, "business_suite")
	hostingService := NewDownloadHostingService(db)

	handler := downloadhttp.Authorize(downloadAuthorizationDependencies(authorizer, hostingService, planService))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/downloads?app=test_app&platform=windows", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &administration.UserClaims{Email: "downloads@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDownloads_MissingApp(t *testing.T) {
	db := setupTestDB(t)

	planService := NewPlanService(db)
	accountService := NewAccountService(db, planService)
	downloadService := NewDownloadService(db)
	authorizer := NewDownloadAuthorizer(downloadService, accountService, "business_suite")
	hostingService := NewDownloadHostingService(db)

	handler := downloadhttp.Authorize(downloadAuthorizationDependencies(authorizer, hostingService, planService))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/downloads?platform=windows", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &administration.UserClaims{Email: "test@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	var errResp ApiErrorResponse
	decodeJSONResponse(t, rr.Body.Bytes(), &errResp)
	if errResp.ErrorType != ApiErrorTypeValidation {
		t.Errorf("Expected error type 'validation', got '%s'", errResp.ErrorType)
	}
}

func TestHandleDownloads_MissingPlatform(t *testing.T) {
	db := setupTestDB(t)

	planService := NewPlanService(db)
	accountService := NewAccountService(db, planService)
	downloadService := NewDownloadService(db)
	authorizer := NewDownloadAuthorizer(downloadService, accountService, "business_suite")
	hostingService := NewDownloadHostingService(db)

	handler := downloadhttp.Authorize(downloadAuthorizationDependencies(authorizer, hostingService, planService))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/downloads?app=test_app", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &administration.UserClaims{Email: "test@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestHandleDownloads_Unauthenticated(t *testing.T) {
	db := setupTestDB(t)

	planService := NewPlanService(db)
	accountService := NewAccountService(db, planService)
	downloadService := NewDownloadService(db)
	authorizer := NewDownloadAuthorizer(downloadService, accountService, "business_suite")
	hostingService := NewDownloadHostingService(db)

	handler := downloadhttp.Authorize(downloadAuthorizationDependencies(authorizer, hostingService, planService))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/downloads?app=test_app&platform=windows", nil)
	// No user context
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestHandleDownloads_AppNotFound(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	planService := NewPlanService(db)
	accountService := NewAccountService(db, planService)
	downloadService := NewDownloadService(db)
	authorizer := NewDownloadAuthorizer(downloadService, accountService, "business_suite")
	hostingService := NewDownloadHostingService(db)

	handler := downloadhttp.Authorize(downloadAuthorizationDependencies(authorizer, hostingService, planService))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/downloads?app=nonexistent_app&platform=windows", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &administration.UserClaims{Email: "test@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWriteJSON_Protobuf(t *testing.T) {
	rr := httptest.NewRecorder()

	msg := &shared.CreditsBalance{
		BalanceCredits: 1000000,
	}

	writeJSON(rr, msg)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	var resp map[string]interface{}
	decodeJSONResponse(t, rr.Body.Bytes(), &resp)
}

func TestWriteJSON_Map(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]interface{}{
		"key": "value",
		"num": 42,
	}

	writeJSON(rr, data)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	decodeJSONResponse(t, rr.Body.Bytes(), &resp)
	if resp["key"] != "value" {
		t.Errorf("Expected key 'value', got '%v'", resp["key"])
	}
}

func TestWriteJSONError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		errorType  string
		wantStatus int
		wantType   string
	}{
		{"bad request", http.StatusBadRequest, "Invalid input", ApiErrorTypeValidation, 400, ApiErrorTypeValidation},
		{"unauthorized", http.StatusUnauthorized, "Not authenticated", ApiErrorTypeUnauthorized, 401, ApiErrorTypeUnauthorized},
		{"forbidden", http.StatusForbidden, "Access denied", ApiErrorTypeForbidden, 403, ApiErrorTypeForbidden},
		{"not found", http.StatusNotFound, "Resource not found", ApiErrorTypeNotFound, 404, ApiErrorTypeNotFound},
		{"server error", http.StatusInternalServerError, "Something went wrong", ApiErrorTypeServerError, 500, ApiErrorTypeServerError},
		{"infer type", http.StatusTeapot, "I'm a teapot", "", 418, ApiErrorTypeValidation}, // inferErrorType fallback
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeJSONError(rr, tt.status, tt.message, tt.errorType)

			if rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			var errResp ApiErrorResponse
			decodeJSONResponse(t, rr.Body.Bytes(), &errResp)
			if errResp.Error != tt.message {
				t.Errorf("Expected error message '%s', got '%s'", tt.message, errResp.Error)
			}
			if errResp.ErrorType != tt.wantType {
				t.Errorf("Expected error type '%s', got '%s'", tt.wantType, errResp.ErrorType)
			}
		})
	}
}

func TestInferErrorType(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{http.StatusBadRequest, ApiErrorTypeValidation},
		{http.StatusUnauthorized, ApiErrorTypeUnauthorized},
		{http.StatusForbidden, ApiErrorTypeForbidden},
		{http.StatusNotFound, ApiErrorTypeNotFound},
		{http.StatusTooManyRequests, ApiErrorTypeRateLimited},
		{http.StatusServiceUnavailable, ApiErrorTypeServerError},
		{http.StatusGatewayTimeout, ApiErrorTypeServerError},
		{http.StatusInternalServerError, ApiErrorTypeServerError},
		{http.StatusTeapot, ApiErrorTypeValidation}, // Default for non-5xx
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			result := inferErrorType(tt.status)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIsRetryableErrorType(t *testing.T) {
	tests := []struct {
		errorType string
		retryable bool
	}{
		{ApiErrorTypeNetwork, true},
		{ApiErrorTypeTimeout, true},
		{ApiErrorTypeServerError, true},
		{ApiErrorTypeRateLimited, true},
		{ApiErrorTypeValidation, false},
		{ApiErrorTypeUnauthorized, false},
		{ApiErrorTypeForbidden, false},
		{ApiErrorTypeNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.errorType, func(t *testing.T) {
			result := isRetryableErrorType(tt.errorType)
			if result != tt.retryable {
				t.Errorf("Expected %v, got %v", tt.retryable, result)
			}
		})
	}
}
