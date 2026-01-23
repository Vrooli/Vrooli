package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockConfigStore provides a configurable mock implementation of ConfigStorer for testing.
type mockConfigStore struct {
	variants    map[string]*VariantSnapshot
	branding    *SiteBranding
	variantsDir string

	// Error injection
	getVariantErr    error
	saveVariantErr   error
	deleteVariantErr error
	loadAllErr       error
}

func newMockConfigStore() *mockConfigStore {
	return &mockConfigStore{
		variants:    make(map[string]*VariantSnapshot),
		variantsDir: "/tmp/test-variants",
	}
}

func (m *mockConfigStore) GetVariant(slug string) (*VariantSnapshot, error) {
	if m.getVariantErr != nil {
		return nil, m.getVariantErr
	}
	v, ok := m.variants[slug]
	if !ok {
		return nil, fmt.Errorf("variant %q not found", slug)
	}
	return v, nil
}

func (m *mockConfigStore) ListVariants() []*VariantSnapshot {
	result := make([]*VariantSnapshot, 0, len(m.variants))
	for _, v := range m.variants {
		result = append(result, v)
	}
	return result
}

func (m *mockConfigStore) GetBranding() *SiteBranding {
	if m.branding == nil {
		return &SiteBranding{ID: 1, SiteName: "Test Site"}
	}
	return m.branding
}

func (m *mockConfigStore) VariantCount() int {
	return len(m.variants)
}

func (m *mockConfigStore) GetVariantsDir() string {
	return m.variantsDir
}

func (m *mockConfigStore) SaveVariant(slug string, snapshot *VariantSnapshot) error {
	if m.saveVariantErr != nil {
		return m.saveVariantErr
	}
	m.variants[slug] = snapshot
	return nil
}

func (m *mockConfigStore) DeleteVariant(slug string) error {
	if m.deleteVariantErr != nil {
		return m.deleteVariantErr
	}
	if _, ok := m.variants[slug]; !ok {
		return fmt.Errorf("variant %q not found", slug)
	}
	delete(m.variants, slug)
	return nil
}

func (m *mockConfigStore) SaveBranding(branding *SiteBranding) error {
	m.branding = branding
	return nil
}

func (m *mockConfigStore) UpdateBranding(req *BrandingUpdateRequest) (*SiteBranding, error) {
	if m.branding == nil {
		m.branding = &SiteBranding{ID: 1, SiteName: "Test Site"}
	}
	if req.SiteName != nil {
		m.branding.SiteName = *req.SiteName
	}
	return m.branding, nil
}

func (m *mockConfigStore) ClearBrandingField(field string) error {
	return nil
}

func (m *mockConfigStore) LoadAll() error {
	if m.loadAllErr != nil {
		return m.loadAllErr
	}
	return nil
}

// Helper to add test variant
func (m *mockConfigStore) addTestVariant(slug, name string, weight int) {
	m.variants[slug] = &VariantSnapshot{
		Variant: VariantSnapshotMeta{
			Slug:   slug,
			Name:   name,
			Weight: weight,
			Axes: map[string]string{
				"persona":         "ops_leader",
				"jtbd":            "launch_bundle",
				"conversionStyle": "demo_led",
			},
		},
		Sections: []VariantSection{},
	}
}

// Compile-time check that mockConfigStore implements ConfigStorer
var _ ConfigStorer = (*mockConfigStore)(nil)

// --- handleVariantSelect Tests ---

func TestHandleVariantSelect_Success(t *testing.T) {
	mock := newMockConfigStore()
	mock.addTestVariant("test-variant-a", "Test Variant A", 50)
	mock.addTestVariant("test-variant-b", "Test Variant B", 50)

	// Cast to *ConfigStore type expected by handler
	// NOTE: For this test, we use the real ConfigStore setup since handlers take *ConfigStore
	cs := setupTestConfigStore(t)

	handler := handleVariantSelect(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/select", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp VariantResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Slug == "" {
		t.Error("Expected non-empty slug in response")
	}
}

func TestHandleVariantSelect_NoVariants(t *testing.T) {
	// Create a temp directory for empty config store
	cs := NewConfigStore("", "", defaultVariantSpace)
	// Don't load any variants - empty store

	handler := handleVariantSelect(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/select", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d for no variants, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandleVariantSelect_MethodNotAllowed(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantSelect(cs)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/variants/select", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// --- handlePublicVariantBySlug Tests ---

func TestHandlePublicVariantBySlug_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Skip("No variants available for testing")
	}

	slug := variants[0].Variant.Slug
	handler := handlePublicVariantBySlug(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/variants/"+slug, nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp VariantResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Slug != slug {
		t.Errorf("Expected slug %s, got %s", slug, resp.Slug)
	}
}

func TestHandlePublicVariantBySlug_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handlePublicVariantBySlug(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/variants/nonexistent-slug", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandlePublicVariantBySlug_EmptySlug(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handlePublicVariantBySlug(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/variants/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for empty slug, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleVariantBySlug Tests ---

func TestHandleVariantBySlug_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Skip("No variants available for testing")
	}

	slug := variants[0].Variant.Slug
	handler := handleVariantBySlug(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/"+slug, nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleVariantBySlug_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantBySlug(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/nonexistent", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// --- handleVariantsList Tests ---

func TestHandleVariantsList_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantsList(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	variants, ok := resp["variants"].([]interface{})
	if !ok {
		t.Fatal("Expected 'variants' array in response")
	}

	// Should have at least one variant from test setup
	if len(variants) == 0 {
		t.Skip("No variants available for testing")
	}
}

func TestHandleVariantsList_Empty(t *testing.T) {
	cs := NewConfigStore("", "", defaultVariantSpace)
	handler := handleVariantsList(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for empty list, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	variants, ok := resp["variants"].([]interface{})
	if !ok {
		t.Fatal("Expected 'variants' array in response")
	}

	if len(variants) != 0 {
		t.Errorf("Expected empty variants array, got %d items", len(variants))
	}
}

// --- handleVariantUpdate Tests ---

func TestHandleVariantUpdate_InvalidJSON(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Skip("No variants available for testing")
	}

	slug := variants[0].Variant.Slug
	handler := handleVariantUpdate(cs)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/"+slug, strings.NewReader("{invalid json"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleVariantUpdate_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantUpdate(cs)

	body := `{"name": "Updated Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/nonexistent", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleVariantUpdate_MethodNotAllowed(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantUpdate(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// --- handleVariantDelete Tests ---

func TestHandleVariantDelete_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantDelete(cs)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/variants/nonexistent", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for not found, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleVariantDelete_MethodNotAllowed(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantDelete(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// --- handleVariantExport Tests ---

func TestHandleVariantExport_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Skip("No variants available for testing")
	}

	slug := variants[0].Variant.Slug
	handler := handleVariantExport(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/variants/"+slug+"/export", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp VariantSnapshot
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Variant.Slug != slug {
		t.Errorf("Expected slug %s, got %s", slug, resp.Variant.Slug)
	}
}

func TestHandleVariantExport_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantExport(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/variants/nonexistent/export", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleVariantImport Tests ---

func TestHandleVariantImport_InvalidJSON(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantImport(cs)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants/test/import", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleVariantImport_SlugMismatch(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantImport(cs)

	payload := VariantSnapshotInput{
		Variant: VariantSnapshotMetaInput{
			Slug: "different-slug",
			Name: "Test",
			Axes: map[string]string{
				"persona":         "ops_leader",
				"jtbd":            "launch_bundle",
				"conversionStyle": "demo_led",
			},
		},
		Sections: []VariantSectionInput{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants/test-slug/import", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for slug mismatch, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "slug") {
		t.Errorf("Expected error about slug mismatch, got: %s", w.Body.String())
	}
}

// --- handleVariantSnapshotSync Tests ---

func TestHandleVariantSnapshotSync_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantSnapshotSync(cs)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", resp["status"])
	}
}

func TestHandleVariantSnapshotSync_MethodNotAllowed(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantSnapshotSync(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/variants/sync", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

// --- selectWeightedRandomVariant Tests ---

func TestSelectWeightedRandomVariant_SingleVariant(t *testing.T) {
	variants := []*VariantSnapshot{
		{Variant: VariantSnapshotMeta{Slug: "only-one", Weight: 100}},
	}

	selected := selectWeightedRandomVariant(variants)

	if selected == nil {
		t.Fatal("Expected a variant to be selected")
	}
	if selected.Variant.Slug != "only-one" {
		t.Errorf("Expected 'only-one', got %s", selected.Variant.Slug)
	}
}

func TestSelectWeightedRandomVariant_EmptyList(t *testing.T) {
	variants := []*VariantSnapshot{}

	selected := selectWeightedRandomVariant(variants)

	if selected != nil {
		t.Error("Expected nil for empty list")
	}
}

func TestSelectWeightedRandomVariant_ZeroWeights(t *testing.T) {
	variants := []*VariantSnapshot{
		{Variant: VariantSnapshotMeta{Slug: "first", Weight: 0}},
		{Variant: VariantSnapshotMeta{Slug: "second", Weight: 0}},
	}

	selected := selectWeightedRandomVariant(variants)

	if selected == nil {
		t.Fatal("Expected fallback to first variant")
	}
	if selected.Variant.Slug != "first" {
		t.Errorf("Expected 'first' as fallback, got %s", selected.Variant.Slug)
	}
}

func TestSelectWeightedRandomVariant_WeightDistribution(t *testing.T) {
	// Create variants with very skewed weights
	variants := []*VariantSnapshot{
		{Variant: VariantSnapshotMeta{Slug: "heavy", Weight: 1000}},
		{Variant: VariantSnapshotMeta{Slug: "light", Weight: 1}},
	}

	// Run multiple times and count
	heavyCount := 0
	iterations := 1000
	for i := 0; i < iterations; i++ {
		selected := selectWeightedRandomVariant(variants)
		if selected.Variant.Slug == "heavy" {
			heavyCount++
		}
	}

	// With 1000:1 ratio, heavy should be selected ~99.9% of time
	// Allow some variance, but it should be overwhelming
	if heavyCount < 900 {
		t.Errorf("Expected heavy variant to be selected mostly, but got %d/%d", heavyCount, iterations)
	}
}

// --- getVariantWeight Tests ---

func TestGetVariantWeight_WithWeight(t *testing.T) {
	snapshot := &VariantSnapshot{
		Variant: VariantSnapshotMeta{Weight: 75},
	}

	weight := getVariantWeight(snapshot)

	if weight != 75 {
		t.Errorf("Expected 75, got %d", weight)
	}
}

func TestGetVariantWeight_DefaultWeight(t *testing.T) {
	snapshot := &VariantSnapshot{
		Variant: VariantSnapshotMeta{Weight: 0},
	}

	weight := getVariantWeight(snapshot)

	if weight != 50 {
		t.Errorf("Expected default 50, got %d", weight)
	}
}
