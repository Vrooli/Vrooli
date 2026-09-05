package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	bundlehttp "landing-page-business-suite-api/handlers/bundles"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/testutil"
)

// ============================================================================
// Bundle catalog transport tests
// ============================================================================

func TestHandleAdminBundleCatalog_Success(t *testing.T) {
	db := setupTestDB(t)

	bundleKey := configureTestBundleEnv(t, "catalog_env")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Catalog Bundle", "prod_catalog", "catalog_env", 1000000, 0.001, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_catalog_1", "Catalog Plan", "pro", "month", "usd", 4999, false, "", 0, 0, "", 5000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	planService := requireTestPlanService(t)
	handler := bundlehttp.Catalog(bundleHandlerDependencies(planService, nil))

	req := httptest.NewRequest(http.MethodGet, "/admin/bundle-catalog", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	// Use generic JSON parsing to avoid protobuf unmarshaling issues
	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	bundles, ok := resp["bundles"].([]interface{})
	if !ok {
		t.Fatal("expected 'bundles' to be an array")
	}

	if len(bundles) == 0 {
		t.Error("expected at least one bundle in catalog")
	}

	entry, ok := bundles[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected bundle entry to be an object")
	}

	bundle, ok := entry["bundle"].(map[string]interface{})
	if !ok {
		t.Fatal("expected bundle to be an object")
	}
	if bundle["bundle_key"] != bundleKey {
		t.Errorf("expected bundle_key %s, got %v", bundleKey, bundle["bundle_key"])
	}
	if bundle["stripe_product_id"] != "prod_catalog" {
		t.Errorf("expected stripe_product_id prod_catalog, got %v", bundle["stripe_product_id"])
	}

	prices, ok := entry["prices"].([]interface{})
	if !ok {
		t.Fatal("expected prices to be an array")
	}
	if len(prices) == 0 {
		t.Fatal("expected at least one price in catalog")
	}

	price, ok := prices[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected price to be an object")
	}
	if price["billing_interval"] != "month" {
		t.Errorf("expected billing_interval month, got %v", price["billing_interval"])
	}
	if price["intro_enabled"] != false {
		t.Errorf("expected intro_enabled false, got %v", price["intro_enabled"])
	}
	if price["display_enabled"] != true {
		t.Errorf("expected display_enabled true, got %v", price["display_enabled"])
	}
	if price["kind"] != "subscription" {
		t.Errorf("expected kind subscription, got %v", price["kind"])
	}
}

func TestHandleAdminBundleCatalog_EmptyCatalog(t *testing.T) {
	db := setupTestDB(t)

	// Use a non-matching environment to get empty results
	os.Setenv("STRIPE_ENVIRONMENT", "nonexistent_env_12345")
	defer os.Unsetenv("STRIPE_ENVIRONMENT")

	planService := NewPlanService(db)
	handler := bundlehttp.Catalog(bundleHandlerDependencies(planService, nil))

	req := httptest.NewRequest(http.MethodGet, "/admin/bundle-catalog", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	// Just verify the response contains valid JSON with a bundles field
	// Don't try to unmarshal into protobuf types
	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	// Should return a response with bundles key
	if _, ok := resp["bundles"]; !ok {
		t.Error("expected 'bundles' key in response")
	}
}

// ============================================================================
// Bundle price update transport tests
// ============================================================================

func TestHandleAdminUpdateBundlePrice_Success(t *testing.T) {
	db := setupTestDB(t)

	bundleKey := configureTestBundleEnv(t, "update_env")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Update Bundle", "prod_update", "update_env", 1000000, 0.001, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_to_update", "Original Plan", "pro", "month", "usd", 4999, false, "", 0, 0, "", 5000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	planService := requireTestPlanService(t)
	handler := bundlehttp.UpdatePrice(bundleHandlerDependencies(planService, nil))

	body := bundlehttp.UpdatePriceRequest{}
	newName := "Updated Plan Name"
	body.PlanName = &newName
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/bundles/%s/prices/price_to_update", bundleKey), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"bundle_key": bundleKey,
		"price_id":   "price_to_update",
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)
	if resp["billing_interval"] != "month" {
		t.Errorf("expected billing_interval month, got %v", resp["billing_interval"])
	}
	if resp["display_enabled"] == nil {
		t.Errorf("expected display_enabled to be present")
	}
}

func TestHandleAdminUpdateBundlePrice_MissingBundleKey(t *testing.T) {
	db := setupTestDB(t)

	planService := NewPlanService(db)
	handler := bundlehttp.UpdatePrice(bundleHandlerDependencies(planService, nil))

	body := bundlehttp.UpdatePriceRequest{}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/bundles//prices/price_123", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"bundle_key": "",
		"price_id":   "price_123",
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleAdminUpdateBundlePrice_MissingPriceID(t *testing.T) {
	db := setupTestDB(t)

	planService := NewPlanService(db)
	handler := bundlehttp.UpdatePrice(bundleHandlerDependencies(planService, nil))

	body := bundlehttp.UpdatePriceRequest{}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/admin/bundles/bundle_key/prices/", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"bundle_key": "bundle_key",
		"price_id":   "",
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleAdminUpdateBundlePrice_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)

	planService := NewPlanService(db)
	handler := bundlehttp.UpdatePrice(bundleHandlerDependencies(planService, nil))

	req := httptest.NewRequest(http.MethodPut, "/admin/bundles/bundle_key/prices/price_id", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"bundle_key": "bundle_key",
		"price_id":   "price_id",
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleAdminUpdateBundlePrice_NotFound(t *testing.T) {
	db := setupTestDB(t)

	bundleKey := configureTestBundleEnv(t, "notfound_env")
	productID := upsertTestBundleProduct(t, db, bundleKey, "NotFound Bundle", "prod_notfound", "notfound_env", 1000000, 0.001, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	planService := requireTestPlanService(t)
	handler := bundlehttp.UpdatePrice(bundleHandlerDependencies(planService, nil))

	body := bundlehttp.UpdatePriceRequest{}
	newName := "New Name"
	body.PlanName = &newName
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/bundles/%s/prices/nonexistent_price", bundleKey), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"bundle_key": bundleKey,
		"price_id":   "nonexistent_price",
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

// ============================================================================
// handleAdminVerifyStripePrice Tests
// ============================================================================

func TestHandleAdminVerifyStripePrice_Success(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_verify", "test", 100, 1, "credits")

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/prices/price_verify" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"id": "price_verify",
				"currency": "usd",
				"unit_amount": 4900,
				"active": true,
				"recurring": {"interval": "month"}
			}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	stripeService := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := bundlehttp.VerifyStripePrice(bundleStripeHandlerDependencies(stripeService))

	req := httptest.NewRequest(http.MethodGet, "/admin/verify-stripe-price?key=price_verify", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)
}

func TestHandleAdminVerifyStripePrice_MissingKey(t *testing.T) {
	db := setupTestDB(t)

	stripeService := NewStripeService(db)
	handler := bundlehttp.VerifyStripePrice(bundleStripeHandlerDependencies(stripeService))

	req := httptest.NewRequest(http.MethodGet, "/admin/verify-stripe-price", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleAdminVerifyStripePrice_EmptyKey(t *testing.T) {
	db := setupTestDB(t)

	stripeService := NewStripeService(db)
	handler := bundlehttp.VerifyStripePrice(bundleStripeHandlerDependencies(stripeService))

	req := httptest.NewRequest(http.MethodGet, "/admin/verify-stripe-price?key=", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleAdminVerifyStripePrice_WhitespaceKey(t *testing.T) {
	db := setupTestDB(t)

	stripeService := NewStripeService(db)
	handler := bundlehttp.VerifyStripePrice(bundleStripeHandlerDependencies(stripeService))

	req := httptest.NewRequest(http.MethodGet, "/admin/verify-stripe-price?key=+++", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleAdminVerifyStripePrice_InvalidKey(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_verify", "test", 100, 1, "credits")

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty list for lookup keys
		if r.URL.RawQuery != "" && r.URL.Path == "/v1/prices" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data": []}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": {"message": "No such price"}}`)
	}))
	defer stripeServer.Close()

	stripeService := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	handler := bundlehttp.VerifyStripePrice(bundleStripeHandlerDependencies(stripeService))

	req := httptest.NewRequest(http.MethodGet, "/admin/verify-stripe-price?key=invalid_price", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

// ============================================================================
// Stripe import preview / import tests
// ============================================================================

func TestHandleAdminStripeImportPreview_Success(t *testing.T) {
	db := setupTestDB(t)

	bundleKey := configureTestBundleEnv(t, "import_preview_env")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Import Preview Bundle", "prod_bundle", "import_preview_env", 1000000, 0.001, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_existing", "Existing Plan", "pro", "month", "usd", 4999, false, "", 0, 0, "", 0, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/products":
			fmt.Fprint(w, `{"data":[{"id":"prod_bundle","name":"Import Bundle","active":true}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/prices":
			fmt.Fprint(w, `{"data":[{"id":"price_existing","lookup_key":"pro_monthly","currency":"usd","unit_amount":4999,"active":true,"recurring":{"interval":"month"}},{"id":"price_new","lookup_key":"pro_yearly","currency":"usd","unit_amount":49900,"active":true,"recurring":{"interval":"year"}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer stripeServer.Close()

	stripe := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	planService := requireTestPlanService(t)
	handler := bundlehttp.PreviewStripeImport(bundleImportHandlerDependencies(stripe, planService))

	req := httptest.NewRequest(http.MethodGet, "/admin/stripe/import-preview", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatusFatal(t, w, http.StatusOK)

	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	if resp["total_prices"] == nil {
		t.Fatal("expected total_prices in response")
	}
}

func TestHandleAdminStripeImport_Success(t *testing.T) {
	db := setupTestDB(t)

	bundleKey := configureTestBundleEnv(t, "import_env")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Import Bundle", "prod_bundle", "import_env", 1000000, 0.001, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/prices/price_new" {
			fmt.Fprint(w, `{"id":"price_new","lookup_key":"pro_monthly","currency":"usd","unit_amount":2900,"active":true,"recurring":{"interval":"month"},"product":{"id":"prod_bundle","name":"Import Bundle"}}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	stripe := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	planService := requireTestPlanService(t)
	handler := bundlehttp.ImportStripePrices(bundleStripeImportDependencies(stripe, planService))

	body := bundlehttp.StripeImportRequest{
		BundleProductID: "prod_bundle",
		Mode:            commerce.StripeImportModeReplace,
		Selections: []commerce.ImportPlanSelection{
			{PriceID: "price_new", Action: "import"},
		},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/admin/stripe/import", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatusFatal(t, w, http.StatusOK)

	var resp commerce.StripeImportResult
	decodeJSONResponse(t, w.Body.Bytes(), &resp)
	if resp.Imported != 1 {
		t.Fatalf("expected 1 imported, got %d", resp.Imported)
	}
}

// ============================================================================
// handleAdminCreateBundlePrice Tests
// ============================================================================

func TestHandleAdminCreateBundlePrice_Success(t *testing.T) {
	db := setupTestDB(t)

	bundleKey := configureTestBundleEnv(t, "create_env")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Create Bundle", "prod_create", "create_env", 1000000, 0.001, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	planService := requireTestPlanService(t)
	handler := bundlehttp.CreatePrice(bundleCreateHandlerDependencies(planService, nil))

	amount := int64(2900)
	currency := "usd"
	body := bundlehttp.CreatePriceRequest{
		StripePriceID:   "price_create",
		PlanName:        "Create Plan",
		PlanTier:        "pro",
		BillingInterval: "month",
		AmountCents:     &amount,
		Currency:        &currency,
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/bundles/%s/prices", bundleKey), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"bundle_key": bundleKey,
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatusFatal(t, w, http.StatusOK)

	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)
	if resp["billing_interval"] != "month" {
		t.Errorf("expected billing_interval month, got %v", resp["billing_interval"])
	}
	if resp["stripe_price_id"] != "price_create" {
		t.Errorf("expected stripe_price_id price_create, got %v", resp["stripe_price_id"])
	}

	if _, err := planService.GetPlanByPriceID("price_create"); err != nil {
		t.Fatalf("expected created plan to exist: %v", err)
	}
}

// ============================================================================
// updateBundlePriceRequest Additional Tests
// ============================================================================

func TestHandleAdminUpdateBundlePrice_UpdateDisplayWeight(t *testing.T) {
	db := setupTestDB(t)

	bundleKey := configureTestBundleEnv(t, "weight_env")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Weight Bundle", "prod_weight", "weight_env", 1000000, 0.001, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_weight", "Weight Plan", "pro", "month", "usd", 4999, false, "", 0, 0, "", 5000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	planService := requireTestPlanService(t)
	handler := bundlehttp.UpdatePrice(bundleHandlerDependencies(planService, nil))

	body := bundlehttp.UpdatePriceRequest{}
	newWeight := 99
	body.DisplayWeight = &newWeight
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/bundles/%s/prices/price_weight", bundleKey), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"bundle_key": bundleKey,
		"price_id":   "price_weight",
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)
}

func TestHandleAdminUpdateBundlePrice_ToggleDisplayEnabled(t *testing.T) {
	db := setupTestDB(t)

	bundleKey := configureTestBundleEnv(t, "toggle_env")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Toggle Bundle", "prod_toggle", "toggle_env", 1000000, 0.001, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_toggle", "Toggle Plan", "pro", "month", "usd", 4999, false, "", 0, 0, "", 5000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	planService := requireTestPlanService(t)
	handler := bundlehttp.UpdatePrice(bundleHandlerDependencies(planService, nil))

	body := bundlehttp.UpdatePriceRequest{}
	enabled := false
	body.DisplayEnabled = &enabled
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/admin/bundles/%s/prices/price_toggle", bundleKey), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{
		"bundle_key": bundleKey,
		"price_id":   "price_toggle",
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)
}
