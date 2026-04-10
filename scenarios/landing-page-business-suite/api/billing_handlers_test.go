package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestHandleBillingCreateCheckoutSession_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Setup test bundle data
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_test_success", "Pro Plan", "pro", "month", "usd", 4900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	// Create mock Stripe server
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"cs_test_123","url":"https://checkout.stripe.test/cs_test_123","status":"open","customer_email":"test@example.com","subscription":"sub_123","amount_total":4900,"mode":"subscription","currency":"usd"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingCreateCheckoutSession(service)

	reqBody := `{"price_id":"price_test_success","success_url":"/success","cancel_url":"/cancel","customer_email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp landing_page_react_vite_v1.CreateCheckoutSessionResponse
	if err := protojson.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Session == nil {
		t.Fatal("Expected session in response")
	}
	if resp.Session.Url != "https://checkout.stripe.test/cs_test_123" {
		t.Errorf("Expected session URL 'https://checkout.stripe.test/cs_test_123', got '%s'", resp.Session.Url)
	}
}

func TestHandleBillingCreateCheckoutSession_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)
	handler := handleBillingCreateCheckoutSession(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	var errResp ApiErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}
	if errResp.ErrorType != ApiErrorTypeValidation {
		t.Errorf("Expected error type 'validation', got '%s'", errResp.ErrorType)
	}
}

func TestHandleBillingCreateCheckoutSession_StripeError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Setup test bundle data with missing price to trigger error
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_error_test", "Error Plan", "pro", "month", "usd", 4900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	// Create mock Stripe server that returns error
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"message":"No such price","type":"invalid_request_error"}}`)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingCreateCheckoutSession(service)

	reqBody := `{"price_id":"price_error_test","success_url":"/success","cancel_url":"/cancel","customer_email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	var errResp ApiErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}
	if !strings.Contains(errResp.Error, "No such price") {
		t.Errorf("Expected Stripe error message, got '%s'", errResp.Error)
	}
}

func TestHandleBillingCreateCreditsSession_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Setup test bundle data for credits
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_credits", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_credits_test", "Credits Pack", "credits", "one_time", "usd", 1000, false, "", 0, 0, "", 0, 10000, 1, 10, "none", sessionTypeCreditsTopup, map[string]interface{}{})

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"cs_credits_123","url":"https://checkout.stripe.test/cs_credits_123","status":"open","customer_email":"credits@example.com","amount_total":1000,"mode":"payment","currency":"usd"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingCreateCreditsSession(service)

	reqBody := `{"price_id":"price_credits_test","success_url":"/success","cancel_url":"/cancel","customer_email":"credits@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/credits", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleBillingPortalURL_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Insert test customer
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ('sub_portal_test', 'cus_portal_test', 'portal@example.com', 'active', 'pro', 'business_suite')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test subscription: %v", err)
	}

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bps_123","url":"https://billing.stripe.test/session/bps_123"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingPortalURL(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal?return_url=https://example.com/account", nil)
	// Inject authenticated user context
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Email: "portal@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	url, ok := resp["url"].(string)
	if !ok || !strings.Contains(url, "billing.stripe.test") {
		t.Errorf("Expected portal URL, got %v", resp)
	}
}

func TestHandleBillingPortalURL_Unauthenticated(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)
	handler := handleBillingPortalURL(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal?return_url=https://example.com/account", nil)
	// No user context injected
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}

	var errResp ApiErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}
	if errResp.ErrorType != ApiErrorTypeUnauthorized {
		t.Errorf("Expected error type 'unauthorized', got '%s'", errResp.ErrorType)
	}
}

func TestHandleBillingPortalURL_StripeError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Insert test customer without Stripe customer ID
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ('sub_error_test', 'cus_error_test', 'error@example.com', 'active', 'pro', 'business_suite')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test subscription: %v", err)
	}

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":{"message":"Customer not found","type":"invalid_request_error"}}`)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingPortalURL(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal?return_url=https://example.com/account", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Email: "error@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// TestCreateCheckoutSessionHandler_ConsolidatedLogic verifies the consolidated handler works correctly
func TestCreateCheckoutSessionHandler_ConsolidatedLogic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_consolidated", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_consolidated_test", "Consolidated Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"cs_consolidated","url":"https://checkout.stripe.test/cs_consolidated","status":"open","customer_email":"consolidated@example.com","amount_total":2900,"mode":"subscription","currency":"usd"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	// Test with custom log key and error message
	handler := createCheckoutSessionHandler(service, "custom_log_key", "Custom error message", false)

	reqBody, _ := json.Marshal(map[string]string{
		"price_id":       "price_consolidated_test",
		"success_url":    "/success",
		"cancel_url":     "/cancel",
		"customer_email": "consolidated@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/test", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// Additional Billing Portal Tests
// ============================================================================

// TestHandleBillingPortalURL_NoCustomerFound verifies behavior when no subscription
// exists for the authenticated user.
func TestHandleBillingPortalURL_NoCustomerFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// No subscription inserted - user has no Stripe customer
	// Mock server returns empty customer list
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/customers/search") {
			// Return empty result - no customer found
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)
	handler := handleBillingPortalURL(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal?return_url=https://example.com/account", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Email: "nocustomer@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for no customer, got %d: %s", rr.Code, rr.Body.String())
	}

	var errResp ApiErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}
	// The handler returns a generic error message to the client (for security reasons)
	// The detailed "no Stripe customer found" error is only logged
	if !strings.Contains(strings.ToLower(errResp.Error), "billing portal") {
		t.Errorf("Expected billing portal related error, got '%s'", errResp.Error)
	}
}

// TestHandleBillingPortalURL_CustomerLookupFallback verifies that customer lookup
// falls back to Stripe API when not found locally.
func TestHandleBillingPortalURL_CustomerLookupFallback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// No local subscription, but customer exists in Stripe
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/customers/search") {
			// Return customer found by email lookup (search endpoint)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[{"id":"cus_fallback_123","email":"fallback@example.com"}]}`)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bps_fallback","url":"https://billing.stripe.test/session/bps_fallback"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingPortalURL(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal?return_url=https://example.com/account", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Email: "fallback@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	url, ok := resp["url"].(string)
	if !ok || !strings.Contains(url, "billing.stripe.test") {
		t.Errorf("Expected portal URL, got %v", resp)
	}
}

// TestHandleBillingPortalURL_ReturnURLValidation verifies return URL handling.
func TestHandleBillingPortalURL_ReturnURLValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Insert test customer
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ('sub_return_url', 'cus_return_url', 'returnurl@example.com', 'active', 'pro', 'business_suite')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test subscription: %v", err)
	}

	var capturedReturnURL string
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions" {
			if err := r.ParseForm(); err == nil {
				capturedReturnURL = r.FormValue("return_url")
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bps_return","url":"https://billing.stripe.test/session/bps_return"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingPortalURL(service)

	// Test with return_url
	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal?return_url=https://myapp.com/billing", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Email: "returnurl@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if capturedReturnURL != "https://myapp.com/billing" {
		t.Errorf("Expected return_url to be passed to Stripe, got '%s'", capturedReturnURL)
	}
}

// TestHandleBillingPortalURL_NoReturnURL verifies behavior when return URL is not provided.
func TestHandleBillingPortalURL_NoReturnURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Insert test customer
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ('sub_no_return', 'cus_no_return', 'noreturn@example.com', 'active', 'pro', 'business_suite')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test subscription: %v", err)
	}

	var capturedReturnURL string
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions" {
			if err := r.ParseForm(); err == nil {
				capturedReturnURL = r.FormValue("return_url")
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bps_noreturn","url":"https://billing.stripe.test/session/bps_noreturn"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingPortalURL(service)

	// Request without return_url
	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Email: "noreturn@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	// Return URL should be empty or not passed
	if capturedReturnURL != "" {
		t.Errorf("Expected no return_url when not provided, got '%s'", capturedReturnURL)
	}
}

// TestHandleBillingPortalURL_CustomerIDLookup verifies portal access using customer ID directly.
func TestHandleBillingPortalURL_CustomerIDLookup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Insert test customer
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ('sub_cusid_lookup', 'cus_cusid_lookup', 'cusidlookup@example.com', 'active', 'pro', 'business_suite')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test subscription: %v", err)
	}

	var capturedCustomerID string
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions" {
			if err := r.ParseForm(); err == nil {
				capturedCustomerID = r.FormValue("customer")
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bps_cusid","url":"https://billing.stripe.test/session/bps_cusid"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingPortalURL(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal?return_url=https://example.com", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Email: "cusidlookup@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if capturedCustomerID != "cus_cusid_lookup" {
		t.Errorf("Expected customer ID 'cus_cusid_lookup', got '%s'", capturedCustomerID)
	}
}

// TestHandleBillingPortalURL_CaseInsensitiveEmail verifies email lookup is case insensitive.
func TestHandleBillingPortalURL_CaseInsensitiveEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Insert subscription with lowercase email
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ('sub_case_insens', 'cus_case_insens', 'casetest@example.com', 'active', 'pro', 'business_suite')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test subscription: %v", err)
	}

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bps_case","url":"https://billing.stripe.test/session/bps_case"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingPortalURL(service)

	// Request with uppercase email
	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal?return_url=https://example.com", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Email: "CASETEST@EXAMPLE.COM"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 (case insensitive match), got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestHandleBillingPortalURL_PortalConfigurationError verifies error handling for
// portal configuration issues.
func TestHandleBillingPortalURL_PortalConfigurationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Insert test customer
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ('sub_config_err', 'cus_config_err', 'configerr@example.com', 'active', 'pro', 'business_suite')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test subscription: %v", err)
	}

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"message":"The customer portal is not configured","type":"invalid_request_error","code":"portal_not_configured"}}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := handleBillingPortalURL(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/portal?return_url=https://example.com", nil)
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Email: "configerr@example.com"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for portal config error, got %d", rr.Code)
	}

	var errResp ApiErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}
	if !strings.Contains(errResp.Error, "portal") {
		t.Errorf("Expected portal-related error, got '%s'", errResp.Error)
	}
}
