package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Initialize chartProcessor for tests
	chartProcessor = NewChartProcessor(nil)
	os.Exit(m.Run())
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if status, ok := response["status"].(string); !ok || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestGetStylesHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/styles", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getStylesHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if styles, ok := response["styles"].([]interface{}); !ok || len(styles) < 3 {
		t.Errorf("Expected at least 3 style objects, got %v", len(styles))
	}
}

func TestDataValidationHandler(t *testing.T) {
	tests := []struct {
		name       string
		chartType  string
		data       []map[string]interface{}
		wantStatus int
	}{
		{
			name:       "valid bar chart",
			chartType:  "bar",
			data:       []map[string]interface{}{{"x": "A", "y": 10}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid line chart",
			chartType:  "line",
			data:       []map[string]interface{}{{"x": "A", "y": 10}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty data",
			chartType:  "bar",
			data:       []map[string]interface{}{},
			wantStatus: http.StatusBadRequest, // Empty data should fail validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"chart_type": tt.chartType,
				"data":       tt.data,
			}
			body, _ := json.Marshal(reqBody)
			req, err := http.NewRequest("POST", "/validate-data", bytes.NewBuffer(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(dataValidationHandler)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestGenerateChartHandler(t *testing.T) {
	reqBody := map[string]interface{}{
		"chart_type":     "bar",
		"data":           []map[string]interface{}{{"x": "A", "y": 10}, {"x": "B", "y": 20}},
		"export_formats": []string{"png"},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/charts/generate", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(generateChartHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Errorf("Expected success true, got %v", response["success"])
	}

	if _, ok := response["chart_id"].(string); !ok {
		t.Errorf("Expected chart_id in response")
	}

	if _, ok := response["files"].(map[string]interface{}); !ok {
		t.Errorf("Expected files object in response")
	}

	if _, ok := response["metadata"].(map[string]interface{}); !ok {
		t.Errorf("Expected metadata object in response")
	}
}

func TestInvalidChartRequest(t *testing.T) {
	reqBody := map[string]interface{}{
		"chart_type":     "invalid_type",
		"data":           []map[string]interface{}{{"x": "A", "y": 10}},
		"export_formats": []string{"png"},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/charts/generate", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(generateChartHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for invalid request: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestEmptyDataRequest(t *testing.T) {
	reqBody := map[string]interface{}{
		"chart_type":     "bar",
		"data":           []map[string]interface{}{},
		"export_formats": []string{"png"},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/charts/generate", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(generateChartHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for empty data: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestHealthGenerationHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/health/generation", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthGenerationHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	response := assertJSONResponse(t, rr, http.StatusOK)
	assertResponseHasField(t, response, "status")
	assertResponseHasField(t, response, "capabilities")
}

func TestGetTemplatesHandler(t *testing.T) {
	tests := []struct {
		name         string
		category     string
		industry     string
		minTemplates int
	}{
		{
			name:         "AllTemplates",
			minTemplates: 10,
		},
		{
			name:         "BusinessCategory",
			category:     "business",
			minTemplates: 1,
		},
		{
			name:         "FinancialCategory",
			category:     "financial",
			minTemplates: 1,
		},
		{
			name:         "RetailIndustry",
			industry:     "retail",
			minTemplates: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/templates"
			if tt.category != "" {
				url += "?category=" + tt.category
			}
			if tt.industry != "" {
				if tt.category != "" {
					url += "&"
				} else {
					url += "?"
				}
				url += "industry=" + tt.industry
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(getTemplatesHandler)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			response := assertJSONResponse(t, rr, http.StatusOK)

			if templates, ok := response["templates"].([]interface{}); ok {
				if len(templates) < tt.minTemplates {
					t.Errorf("Expected at least %d templates, got %d", tt.minTemplates, len(templates))
				}
			} else {
				t.Error("Expected templates array in response")
			}
		})
	}
}

func TestCreateStyleHandler(t *testing.T) {
	reqBody := map[string]interface{}{
		"name":        "Custom Style",
		"category":    "custom",
		"description": "A custom test style",
		"colors":      []string{"#FF0000", "#00FF00", "#0000FF"},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/styles", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createStyleHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	response := assertJSONResponse(t, rr, http.StatusCreated)
	assertResponseHasField(t, response, "id")
	assertResponseHasField(t, response, "created_at")
}

func TestStyleBuilderPreviewHandler(t *testing.T) {
	reqBody := map[string]interface{}{
		"chart_type": "bar",
		"style": map[string]interface{}{
			"name":         "Preview Style",
			"colors":       []string{"#FF6384", "#36A2EB"},
			"font_family":  "Arial",
			"font_size":    14,
			"background":   "#FFFFFF",
			"grid_lines":   true,
			"animation":    true,
			"border_width": 2,
			"opacity":      0.8,
		},
		"data": generateTestChartData("bar", 5),
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/styles/builder/preview", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(styleBuilderPreviewHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v. Body: %s", status, http.StatusOK, rr.Body.String())
	}

	response := assertJSONResponse(t, rr, http.StatusOK)
	assertResponseField(t, response, "success", true)
	assertResponseHasField(t, response, "preview")
}

func TestStyleBuilderSaveHandler(t *testing.T) {
	reqBody := map[string]interface{}{
		"name":        "Saved Style",
		"colors":      []string{"#FF6384", "#36A2EB"},
		"font_family": "Arial",
		"font_size":   14,
		"background":  "#FFFFFF",
		"grid_lines":  true,
		"animation":   true,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/styles/builder/save", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(styleBuilderSaveHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	response := assertJSONResponse(t, rr, http.StatusCreated)
	assertResponseHasField(t, response, "id")
	assertResponseHasField(t, response, "created_at")
}

func TestGetColorPalettesHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/styles/builder/palettes", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getColorPalettesHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	response := assertJSONResponse(t, rr, http.StatusOK)
	assertResponseHasField(t, response, "palettes")
	assertResponseHasField(t, response, "recommended")

	if palettes, ok := response["palettes"].([]interface{}); ok {
		if len(palettes) < 3 {
			t.Errorf("Expected at least 3 palettes, got %d", len(palettes))
		}
	}
}

func TestTransformDataHandler(t *testing.T) {
	reqBody := map[string]interface{}{
		"data": generateTestChartData("bar", 10),
		"transform": map[string]interface{}{
			"filter": map[string]interface{}{
				"field":    "y",
				"operator": "gt",
				"value":    15.0,
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/data/transform", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(transformDataHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	response := assertJSONResponse(t, rr, http.StatusOK)
	assertResponseField(t, response, "success", true)
	assertResponseHasField(t, response, "data")
}

func TestAggregateDataHandler(t *testing.T) {
	reqBody := map[string]interface{}{
		"data": []map[string]interface{}{
			{"category": "A", "value": 10.0},
			{"category": "A", "value": 20.0},
			{"category": "B", "value": 15.0},
		},
		"method":   "sum",
		"field":    "value",
		"group_by": "category",
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/data/aggregate", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(aggregateDataHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	response := assertJSONResponse(t, rr, http.StatusOK)
	assertResponseField(t, response, "success", true)
	assertResponseHasField(t, response, "result")
}

func TestGenerateInteractiveChartHandler(t *testing.T) {
	reqBody := map[string]interface{}{
		"chart_type":     "line",
		"data":           generateTestChartData("line", 10),
		"title":          "Interactive Chart Test",
		"export_formats": []string{"png"},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/charts/interactive", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(generateInteractiveChartHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v. Body: %s", status, http.StatusCreated, rr.Body.String())
	}

	response := assertJSONResponse(t, rr, http.StatusCreated)
	assertResponseField(t, response, "success", true)
	assertResponseHasField(t, response, "chart_id")

	// Verify interactive metadata
	if metadata, ok := response["metadata"].(map[string]interface{}); ok {
		assertResponseField(t, metadata, "interactive", true)
		assertResponseField(t, metadata, "animation_enabled", true)
		assertResponseHasField(t, metadata, "features")
	}
}

func TestGenerateCompositeChartHandler(t *testing.T) {
	reqBody := map[string]interface{}{
		"chart_type":     "bar",
		"data":           generateTestChartData("bar", 5),
		"export_formats": []string{"png"},
		"config": map[string]interface{}{
			"composition": map[string]interface{}{
				"layout": "grid",
				"charts": []map[string]interface{}{
					{
						"chart_type": "bar",
						"data":       generateTestChartData("bar", 3),
					},
					{
						"chart_type": "line",
						"data":       generateTestChartData("line", 5),
					},
				},
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "/api/v1/charts/composite", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(generateCompositeChartHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v. Body: %s", status, http.StatusCreated, rr.Body.String())
	}

	response := assertJSONResponse(t, rr, http.StatusCreated)
	assertResponseField(t, response, "success", true)
	assertResponseHasField(t, response, "chart_id")
}

func TestInvalidJSONRequest(t *testing.T) {
	req, err := http.NewRequest("POST", "/api/v1/charts/generate", bytes.NewBufferString("{invalid json"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(generateChartHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code for invalid JSON: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestSendErrorResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	sendErrorResponse(rr, "Test error message", "test_error", http.StatusBadRequest)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
	}

	response := assertJSONResponse(t, rr, http.StatusBadRequest)
	assertResponseField(t, response, "success", false)

	if err, ok := response["error"].(map[string]interface{}); ok {
		if message, ok := err["message"].(string); !ok || message != "Test error message" {
			t.Errorf("Expected error message 'Test error message', got %v", message)
		}
	} else {
		t.Error("Expected error object in response")
	}
}

func TestGenerateSampleData(t *testing.T) {
	tests := []struct {
		chartType  string
		expectKeys []string
	}{
		{"bar", []string{"x", "y"}},
		{"line", []string{"x", "y"}},
		{"area", []string{"x", "y"}},
		{"pie", []string{"name", "value"}},
		{"scatter", []string{"x", "y"}},
		{"unknown", []string{"label", "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.chartType, func(t *testing.T) {
			data := generateSampleData(tt.chartType)

			if len(data) == 0 {
				t.Error("Expected non-empty sample data")
			}

			// Check first data point has expected keys
			if len(data) > 0 {
				for _, key := range tt.expectKeys {
					if _, ok := data[0][key]; !ok {
						t.Errorf("Expected key %s in sample data for %s", key, tt.chartType)
					}
				}
			}
		})
	}
}
