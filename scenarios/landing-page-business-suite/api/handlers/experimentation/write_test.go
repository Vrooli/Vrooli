package variant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/experimentation"
)

type writeTestStore struct {
	variants  map[string]*experimentation.VariantSnapshot
	saveErr   error
	deleteErr error
	loadErr   error
}

var _ experimentation.ConfigStorer = (*writeTestStore)(nil)

func newWriteTestStore() *writeTestStore {
	return &writeTestStore{variants: map[string]*experimentation.VariantSnapshot{}}
}

func (store *writeTestStore) GetVariant(slug string) (*experimentation.VariantSnapshot, error) {
	variant, ok := store.variants[slug]
	if !ok {
		return nil, fmt.Errorf("variant %q not found", slug)
	}
	return variant, nil
}
func (store *writeTestStore) ListVariants() []*experimentation.VariantSnapshot { return nil }
func (store *writeTestStore) GetBranding() *experimentation.SiteBranding       { return nil }
func (store *writeTestStore) VariantCount() int                                { return len(store.variants) }
func (store *writeTestStore) GetVariantsDir() string                           { return "" }
func (store *writeTestStore) SaveVariant(slug string, snapshot *experimentation.VariantSnapshot) error {
	if store.saveErr != nil {
		return store.saveErr
	}
	store.variants[slug] = snapshot
	return nil
}

func (store *writeTestStore) DeleteVariant(slug string) error {
	if store.deleteErr != nil {
		return store.deleteErr
	}
	if _, ok := store.variants[slug]; !ok {
		return fmt.Errorf("variant %q not found", slug)
	}
	delete(store.variants, slug)
	return nil
}
func (store *writeTestStore) SaveBranding(*experimentation.SiteBranding) error { return nil }
func (store *writeTestStore) UpdateBranding(*experimentation.BrandingUpdateRequest) (*experimentation.SiteBranding, error) {
	return nil, nil
}
func (store *writeTestStore) ClearBrandingField(string) error { return nil }
func (store *writeTestStore) LoadAll() error                  { return store.loadErr }

func (store *writeTestStore) add(slug, name string) {
	store.variants[slug] = &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{
		Slug: slug, Name: name, Weight: 50,
		Axes: map[string]string{"persona": "silentFounder", "jtbd": "entrepreneurship", "conversionStyle": "emotional"},
	}}
}

func writeTestDependencies(store experimentation.ConfigStorer) WriteDependencies {
	return WriteDependencies{
		Store: store,
		WriteJSON: func(w http.ResponseWriter, payload any) {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(payload); err != nil {
				panic(err)
			}
		},
		WriteError: func(w http.ResponseWriter, status int, message, errorType string) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": message, "error_type": errorType})
		},
		Log: func(string, map[string]any) {}, LogError: func(string, map[string]any) {},
	}
}

func TestUpdate(t *testing.T) {
	t.Run("updates a selected set of fields", func(t *testing.T) {
		store := newWriteTestStore()
		store.add("variant-a", "Original")
		request := httptest.NewRequest(http.MethodPatch, "/api/v1/variants/variant-a", strings.NewReader(`{"name":"Updated","weight":75}`))
		response := httptest.NewRecorder()
		Update(writeTestDependencies(store))(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
		variant, _ := store.GetVariant("variant-a")
		if variant.Variant.Name != "Updated" || variant.Variant.Weight != 75 {
			t.Fatalf("unexpected saved variant: %#v", variant.Variant)
		}
	})

	for name, request := range map[string]*http.Request{
		"invalid JSON": httptest.NewRequest(http.MethodPatch, "/api/v1/variants/variant-a", strings.NewReader("{")),
		"missing slug": httptest.NewRequest(http.MethodPatch, "/api/v1/variants/", strings.NewReader(`{}`)),
		"wrong method": httptest.NewRequest(http.MethodGet, "/api/v1/variants/variant-a", nil),
	} {
		t.Run(name, func(t *testing.T) {
			store := newWriteTestStore()
			store.add("variant-a", "Original")
			response := httptest.NewRecorder()
			Update(writeTestDependencies(store))(response, request)
			if response.Code != http.StatusBadRequest && response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("unexpected status %d", response.Code)
			}
		})
	}

	t.Run("reports absent and failed variants", func(t *testing.T) {
		response := httptest.NewRecorder()
		Update(writeTestDependencies(newWriteTestStore()))(response, httptest.NewRequest(http.MethodPatch, "/api/v1/variants/missing", strings.NewReader(`{}`)))
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d", response.Code)
		}
		store := newWriteTestStore()
		store.add("variant-a", "Original")
		store.saveErr = fmt.Errorf("disk full")
		response = httptest.NewRecorder()
		Update(writeTestDependencies(store))(response, httptest.NewRequest(http.MethodPatch, "/api/v1/variants/variant-a", strings.NewReader(`{}`)))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "disk full") {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})

	t.Run("normalizes header configuration before saving", func(t *testing.T) {
		store := newWriteTestStore()
		store.add("variant-a", "Variant A")
		response := httptest.NewRecorder()
		Update(writeTestDependencies(store))(response, httptest.NewRequest(http.MethodPatch, "/api/v1/variants/variant-a", strings.NewReader(`{"header_config":{"branding":{"mode":"none","label":""},"nav":{"links":[]}}}`)))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
		variant, _ := store.GetVariant("variant-a")
		if variant.Variant.HeaderConfig.Branding.Mode != "none" || variant.Variant.HeaderConfig.Branding.Label != "" {
			t.Fatalf("header configuration was not normalized: %#v", variant.Variant.HeaderConfig)
		}
	})
}

func TestDelete(t *testing.T) {
	store := newWriteTestStore()
	store.add("remove-me", "Remove")
	response := httptest.NewRecorder()
	Delete(writeTestDependencies(store))(response, httptest.NewRequest(http.MethodDelete, "/api/v1/variants/remove-me", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if _, err := store.GetVariant("remove-me"); err == nil {
		t.Fatal("variant was not deleted")
	}

	for name, request := range map[string]*http.Request{
		"missing":      httptest.NewRequest(http.MethodDelete, "/api/v1/variants/missing", nil),
		"empty":        httptest.NewRequest(http.MethodDelete, "/api/v1/variants/", nil),
		"wrong method": httptest.NewRequest(http.MethodGet, "/api/v1/variants/x", nil),
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			Delete(writeTestDependencies(newWriteTestStore()))(response, request)
			if response.Code != http.StatusBadRequest && response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("unexpected status %d", response.Code)
			}
		})
	}

	t.Run("reports a storage deletion failure", func(t *testing.T) {
		store := newWriteTestStore()
		store.add("remove-me", "Remove")
		store.deleteErr = fmt.Errorf("file in use")
		response := httptest.NewRecorder()
		Delete(writeTestDependencies(store))(response, httptest.NewRequest(http.MethodDelete, "/api/v1/variants/remove-me", nil))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "file in use") {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})
}

func TestExport(t *testing.T) {
	store := newWriteTestStore()
	store.add("export-me", "Export")
	response := httptest.NewRecorder()
	Export(writeTestDependencies(store))(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/variants/export-me/export", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var result experimentation.VariantSnapshot
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil || result.Variant.Slug != "export-me" {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	response = httptest.NewRecorder()
	Export(writeTestDependencies(store))(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/variants/missing/export", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing status = %d", response.Code)
	}
}

func TestImport(t *testing.T) {
	payload := experimentation.VariantSnapshotInput{Variant: experimentation.VariantSnapshotMetaInput{Slug: "import-me", Name: "Imported", Axes: map[string]string{"persona": "silentFounder"}}, Sections: []experimentation.VariantSectionInput{{SectionType: "hero"}}}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	store := newWriteTestStore()
	response := httptest.NewRecorder()
	Import(writeTestDependencies(store))(response, httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants/import-me/import", bytes.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	variant, _ := store.GetVariant("import-me")
	if variant.Sections[0].Order != 1 || !variant.Sections[0].Enabled {
		t.Fatalf("sections = %#v", variant.Sections)
	}

	for name, request := range map[string]*http.Request{
		"invalid JSON":    httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants/import-me/import", strings.NewReader("{")),
		"mismatched slug": httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants/import-me/import", strings.NewReader(`{"variant":{"slug":"other"}}`)),
		"empty slug":      httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants//import", strings.NewReader(`{}`)),
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			Import(writeTestDependencies(newWriteTestStore()))(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d", response.Code)
			}
		})
	}

	t.Run("reports a storage save failure", func(t *testing.T) {
		store := newWriteTestStore()
		store.saveErr = fmt.Errorf("permission denied")
		response := httptest.NewRecorder()
		Import(writeTestDependencies(store))(response, httptest.NewRequest(http.MethodPut, "/api/v1/admin/variants/import-me/import", bytes.NewReader(body)))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "permission denied") {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	})
}

func TestSync(t *testing.T) {
	response := httptest.NewRecorder()
	Sync(writeTestDependencies(newWriteTestStore()))(response, httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	store := newWriteTestStore()
	store.loadErr = fmt.Errorf("unavailable")
	response = httptest.NewRecorder()
	Sync(writeTestDependencies(store))(response, httptest.NewRequest(http.MethodPost, "/api/v1/admin/variants/sync", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", response.Code)
	}
	response = httptest.NewRecorder()
	Sync(writeTestDependencies(newWriteTestStore()))(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/variants/sync", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", response.Code)
	}
}
