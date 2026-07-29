package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/testutil"
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

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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

	testutil.RequireHTTPStatus(t, w, http.StatusInternalServerError)
	if !strings.Contains(w.Body.String(), "No variants available.") {
		t.Errorf("Expected no-variants error, got: %s", w.Body.String())
	}
}

func TestHandleVariantSelect_MethodNotAllowed(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantSelect(cs)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/variants/select", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusMethodNotAllowed)
}

// --- handlePublicVariantBySlug Tests ---

func TestHandlePublicVariantBySlug_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}

	slug := variants[0].Variant.Slug
	handler := handlePublicVariantBySlug(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/variants/"+slug, nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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

	testutil.RequireHTTPStatus(t, w, http.StatusNotFound)
}

func TestHandlePublicVariantBySlug_EmptySlug(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handlePublicVariantBySlug(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/variants/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

// --- handleVariantBySlug Tests ---

func TestHandleVariantBySlug_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}

	slug := variants[0].Variant.Slug
	handler := handleVariantBySlug(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/"+slug, nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)
}

func TestHandleVariantBySlug_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantBySlug(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/nonexistent", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusNotFound)
}

// --- handleVariantsList Tests ---

func TestHandleVariantsList_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantsList(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	variants, ok := resp["variants"].([]interface{})
	if !ok {
		t.Fatal("Expected 'variants' array in response")
	}

	if len(variants) == 0 {
		t.Fatal("tracked test configuration must produce at least one variant")
	}
}

func TestHandleVariantsList_Empty(t *testing.T) {
	cs := NewConfigStore("", "", defaultVariantSpace)
	handler := handleVariantsList(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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
		t.Fatal("tracked test configuration must contain at least one variant")
	}

	slug := variants[0].Variant.Slug
	handler := handleVariantUpdate(cs)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/"+slug, strings.NewReader("{invalid json"))
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleVariantUpdate_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantUpdate(cs)

	body := `{"name": "Updated Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/nonexistent", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusNotFound)
}

func TestHandleVariantUpdate_MethodNotAllowed(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantUpdate(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusMethodNotAllowed)
}

// --- handleVariantDelete Tests ---

func TestHandleVariantDelete_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantDelete(cs)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/variants/nonexistent", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleVariantDelete_MethodNotAllowed(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantDelete(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/variants/test", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusMethodNotAllowed)
}

// --- handleVariantExport Tests ---

func TestHandleVariantExport_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}

	slug := variants[0].Variant.Slug
	handler := handleVariantExport(cs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/variants/"+slug+"/export", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

// --- handleVariantImport Tests ---

func TestHandleVariantImport_InvalidJSON(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleVariantImport(cs)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants/test/import", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
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

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)

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

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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

	testutil.RequireHTTPStatus(t, w, http.StatusMethodNotAllowed)
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

func TestGetVariantWeight_ZeroWeight(t *testing.T) {
	snapshot := &VariantSnapshot{
		Variant: VariantSnapshotMeta{Weight: 0},
	}

	weight := getVariantWeight(snapshot)

	if weight != 0 {
		t.Errorf("Expected 0 (disabled), got %d", weight)
	}
}

func TestGetVariantWeight_NegativeWeight(t *testing.T) {
	snapshot := &VariantSnapshot{
		Variant: VariantSnapshotMeta{Weight: -5},
	}

	weight := getVariantWeight(snapshot)

	if weight != 0 {
		t.Errorf("Expected 0 (disabled), got %d", weight)
	}
}

// --- handleVariantUpdate Success Tests (using mock) ---

func TestHandleVariantUpdate_Success(t *testing.T) {
	mock := newMockConfigStore()
	mock.addTestVariant("test-variant", "Original Name", 50)

	handler := handleVariantUpdate(mock)

	body := `{"name": "Updated Name", "weight": 75, "description": "New description"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/test-variant", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp VariantResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", resp.Name)
	}
	if resp.Weight != 75 {
		t.Errorf("Expected weight 75, got %d", resp.Weight)
	}
}

func TestHandleVariantUpdate_PartialUpdate(t *testing.T) {
	mock := newMockConfigStore()
	mock.addTestVariant("partial-update", "Original Name", 50)

	handler := handleVariantUpdate(mock)

	// Only update weight, leave name unchanged
	body := `{"weight": 25}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/partial-update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp VariantResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Name should remain unchanged
	if resp.Name != "Original Name" {
		t.Errorf("Expected name 'Original Name', got %s", resp.Name)
	}
	if resp.Weight != 25 {
		t.Errorf("Expected weight 25, got %d", resp.Weight)
	}
}

func TestHandleVariantUpdate_SaveError(t *testing.T) {
	mock := newMockConfigStore()
	mock.addTestVariant("save-error", "Test", 50)
	mock.saveVariantErr = fmt.Errorf("disk full")

	handler := handleVariantUpdate(mock)

	body := `{"name": "New Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/save-error", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)

	if !strings.Contains(w.Body.String(), "disk full") {
		t.Errorf("Expected error message to contain 'disk full', got: %s", w.Body.String())
	}
}

func TestHandleVariantUpdate_NormalizesHeaderConfig(t *testing.T) {
	mock := newMockConfigStore()
	mock.addTestVariant("header-test", "Test Variant", 50)

	handler := handleVariantUpdate(mock)

	// Send header config with empty branding - should be normalized
	body := `{"header_config": {"branding": {"mode": "none", "label": ""}, "nav": {"links": []}}}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/header-test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	// Check that the saved variant has normalized header config
	saved, _ := mock.GetVariant("header-test")
	// When branding mode is "none" with empty label, normalization should set it to the variant name
	if saved.Variant.HeaderConfig.Branding.Mode != "none" {
		t.Fatalf("Expected branding mode 'none', got '%s'", saved.Variant.HeaderConfig.Branding.Mode)
	}
	if saved.Variant.HeaderConfig.Branding.Label != "" {
		t.Errorf("Expected branding label to remain empty for mode 'none', got '%s'", saved.Variant.HeaderConfig.Branding.Label)
	}
}

func TestHandleVariantUpdate_EmptySlug(t *testing.T) {
	mock := newMockConfigStore()
	handler := handleVariantUpdate(mock)

	body := `{"name": "Test"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
	if mock.VariantCount() != 0 {
		t.Errorf("Expected empty-slug update to leave store unchanged, got %d variants", mock.VariantCount())
	}
}

// --- handleVariantImport Success Tests (using mock) ---

func TestHandleVariantImport_Success(t *testing.T) {
	mock := newMockConfigStore()
	handler := handleVariantImport(mock)

	payload := VariantSnapshotInput{
		Variant: VariantSnapshotMetaInput{
			Slug: "import-test",
			Name: "Imported Variant",
			Axes: map[string]string{
				"persona":         "ops_leader",
				"jtbd":            "launch_bundle",
				"conversionStyle": "demo_led",
			},
		},
		Sections: []VariantSectionInput{
			{SectionType: "hero", Order: 1},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants/import-test/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	// Verify variant was saved
	saved, err := mock.GetVariant("import-test")
	if err != nil {
		t.Fatalf("Expected variant to be saved: %v", err)
	}
	if saved.Variant.Name != "Imported Variant" {
		t.Errorf("Expected name 'Imported Variant', got %s", saved.Variant.Name)
	}
}

func TestHandleVariantImport_SaveError(t *testing.T) {
	mock := newMockConfigStore()
	mock.saveVariantErr = fmt.Errorf("permission denied")

	handler := handleVariantImport(mock)

	payload := VariantSnapshotInput{
		Variant: VariantSnapshotMetaInput{
			Slug: "save-error",
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

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants/save-error/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleVariantImport_EmptySlug(t *testing.T) {
	mock := newMockConfigStore()
	handler := handleVariantImport(mock)

	payload := VariantSnapshotInput{
		Variant: VariantSnapshotMetaInput{
			Slug: "",
			Name: "Test",
			Axes: map[string]string{"persona": "ops_leader"},
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants//import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
	if mock.VariantCount() != 0 {
		t.Errorf("Expected empty-slug import to leave store unchanged, got %d variants", mock.VariantCount())
	}
}

// --- handleVariantDelete Success Tests (using mock) ---

func TestHandleVariantDelete_Success(t *testing.T) {
	mock := newMockConfigStore()
	mock.addTestVariant("delete-me", "To Delete", 50)

	handler := handleVariantDelete(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/variants/delete-me", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	// Verify deletion
	_, err := mock.GetVariant("delete-me")
	if err == nil {
		t.Error("Expected variant to be deleted")
	}
}

func TestHandleVariantDelete_EmptySlug(t *testing.T) {
	mock := newMockConfigStore()
	handler := handleVariantDelete(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/variants/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
	if mock.VariantCount() != 0 {
		t.Errorf("Expected empty-slug delete to leave store unchanged, got %d variants", mock.VariantCount())
	}
}

func TestHandleVariantDelete_DeleteError(t *testing.T) {
	mock := newMockConfigStore()
	mock.addTestVariant("delete-error", "Test", 50)
	mock.deleteVariantErr = fmt.Errorf("file in use")

	handler := handleVariantDelete(mock)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/variants/delete-error", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

// --- handleVariantSnapshotSync Tests (using mock) ---

func TestHandleVariantSnapshotSync_LoadError(t *testing.T) {
	mock := newMockConfigStore()
	mock.loadAllErr = fmt.Errorf("directory not readable")

	handler := handleVariantSnapshotSync(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusInternalServerError)
}
