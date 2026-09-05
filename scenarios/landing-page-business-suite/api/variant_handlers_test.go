package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	varianthttp "landing-page-business-suite-api/handlers/experimentation"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/testutil"
)

func TestHandleVariantSelect(t *testing.T) {
	t.Run("selects a configured variant", func(t *testing.T) {
		store := setupTestConfigStore(t)
		response := httptest.NewRecorder()
		varianthttp.Select(variantReadDependencies(store, "/api/v1/variants/"))(response, httptest.NewRequest(http.MethodGet, "/api/v1/variants/select", nil))
		testutil.RequireHTTPStatus(t, response, http.StatusOK)
		var variant VariantResponse
		if err := json.NewDecoder(response.Body).Decode(&variant); err != nil || variant.Slug == "" {
			t.Fatalf("selection = %#v, err = %v", variant, err)
		}
	})

	t.Run("reports no configured variants", func(t *testing.T) {
		store := experimentation.NewConfigStore("", "", nil)
		response := httptest.NewRecorder()
		varianthttp.Select(variantReadDependencies(store, "/api/v1/variants/"))(response, httptest.NewRequest(http.MethodGet, "/api/v1/variants/select", nil))
		testutil.RequireHTTPStatus(t, response, http.StatusInternalServerError)
		if !strings.Contains(response.Body.String(), "No variants available.") {
			t.Fatalf("unexpected response: %s", response.Body.String())
		}
	})

	t.Run("rejects unsupported methods", func(t *testing.T) {
		response := httptest.NewRecorder()
		varianthttp.Select(variantReadDependencies(setupTestConfigStore(t), "/api/v1/variants/"))(response, httptest.NewRequest(http.MethodPost, "/api/v1/variants/select", nil))
		testutil.RequireHTTPStatus(t, response, http.StatusMethodNotAllowed)
	})
}

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
