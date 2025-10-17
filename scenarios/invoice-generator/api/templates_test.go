package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleListTemplates tests listing all templates
func TestHandleListTemplates(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/api/templates", nil)
	w := httptest.NewRecorder()

	ListTemplatesHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var templates []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
		t.Fatalf("Failed to decode templates response: %v", err)
	}

	// Should return an array (empty or with data)
	if templates == nil {
		t.Error("Expected array response, got nil")
	}
}

// TestHandleGetTemplate tests retrieving a single template
func TestHandleGetTemplate(t *testing.T) {
	setupTestDB(t)

	// Test with non-existent template
	req := httptest.NewRequest("GET", "/api/templates/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()

	GetTemplateHandler(w, req)

	resp := w.Result()
	// Expecting 404 or 500 for non-existent template
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", resp.StatusCode)
	}
}

// TestHandleGetDefaultTemplate tests retrieving the default template
func TestHandleGetDefaultTemplate(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/api/templates/default", nil)
	w := httptest.NewRecorder()

	GetDefaultTemplateHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var template map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&template); err != nil {
		t.Fatalf("Failed to decode template response: %v", err)
	}

	// Verify template has expected fields
	if _, ok := template["id"]; !ok {
		t.Error("Expected template to have 'id' field")
	}
	if _, ok := template["name"]; !ok {
		t.Error("Expected template to have 'name' field")
	}
	if _, ok := template["template_data"]; !ok {
		t.Error("Expected template to have 'template_data' field")
	}
}

// TestHandleCreateTemplate tests template creation
func TestHandleCreateTemplate(t *testing.T) {
	setupTestDB(t)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "valid template",
			requestBody: map[string]interface{}{
				"name":        "Test Template",
				"description": "Test Description",
				"template_data": map[string]interface{}{
					"layout": map[string]interface{}{
						"page_size": "A4",
					},
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/api/templates", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			CreateTemplateHandler(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d for test '%s'", tt.expectedStatus, resp.StatusCode, tt.name)
			}
		})
	}
}

// TestHandleUpdateTemplate tests template updates
func TestHandleUpdateTemplate(t *testing.T) {
	setupTestDB(t)

	// Test invalid JSON
	body := []byte("invalid json")
	req := httptest.NewRequest("PUT", "/api/templates/00000000-0000-0000-0000-000000000000", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	UpdateTemplateHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

// TestHandleDeleteTemplate tests template deletion
func TestHandleDeleteTemplate(t *testing.T) {
	setupTestDB(t)

	// Test deleting non-existent template
	req := httptest.NewRequest("DELETE", "/api/templates/00000000-0000-0000-0000-000000000000", nil)
	w := httptest.NewRecorder()

	DeleteTemplateHandler(w, req)

	resp := w.Result()
	// Expecting 404 or 500 for non-existent template
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 404 or 500, got %d", resp.StatusCode)
	}
}
