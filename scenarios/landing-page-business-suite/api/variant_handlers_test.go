package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	varianthttp "landing-page-business-suite-api/handlers/experimentation"
	"landing-page-business-suite-api/internal/testutil"
)

func TestHandleVariantReads(t *testing.T) {
	store := setupTestConfigStore(t)
	variants := store.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	slug := variants[0].Variant.Slug

	t.Run("returns a variant by slug", func(t *testing.T) {
		response := httptest.NewRecorder()
		varianthttp.AdminGet(variantReadDependencies(store, "/api/v1/variants/"))(response, httptest.NewRequest(http.MethodGet, "/api/v1/variants/"+slug, nil))
		testutil.RequireHTTPStatus(t, response, http.StatusOK)
		var variant VariantResponse
		if err := json.NewDecoder(response.Body).Decode(&variant); err != nil || variant.Slug != slug {
			t.Fatalf("variant = %#v, err = %v", variant, err)
		}
	})

	t.Run("rejects an unknown variant", func(t *testing.T) {
		response := httptest.NewRecorder()
		varianthttp.AdminGet(variantReadDependencies(store, "/api/v1/variants/"))(response, httptest.NewRequest(http.MethodGet, "/api/v1/variants/not-found", nil))
		testutil.RequireHTTPStatus(t, response, http.StatusNotFound)
	})

	t.Run("lists all variants", func(t *testing.T) {
		response := httptest.NewRecorder()
		varianthttp.List(variantReadDependencies(store, ""))(response, httptest.NewRequest(http.MethodGet, "/api/v1/variants", nil))
		testutil.RequireHTTPStatus(t, response, http.StatusOK)
		var result struct {
			Variants []VariantResponse `json:"variants"`
		}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil || len(result.Variants) != len(variants) {
			t.Fatalf("result = %#v, err = %v", result, err)
		}
	})
}
