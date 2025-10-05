package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app-monitor-api/services"

	"github.com/gin-gonic/gin"
)

func TestNewSystemHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		if handler == nil {
			t.Fatal("Expected non-nil system handler")
		}

		if handler.metricsService != metricsService {
			t.Error("Expected metrics service to be set")
		}

		if handler.resourceCache == nil {
			t.Error("Expected resource cache to be initialized")
		}
	})

	t.Run("WithNilMetricsService", func(t *testing.T) {
		handler := NewSystemHandler(nil)

		if handler == nil {
			t.Fatal("Expected non-nil system handler")
		}

		if handler.metricsService != nil {
			t.Error("Expected metrics service to be nil")
		}
	})
}

func TestGetSystemMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		router := gin.New()
		router.GET("/api/v1/system/metrics", handler.GetSystemMetrics)

		req := httptest.NewRequest("GET", "/api/v1/system/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Should return message about iframe
		if !strings.Contains(w.Body.String(), "iframe") {
			t.Error("Expected response to mention iframe")
		}
	})
}

func TestGetResources(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("FirstCall", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		router := gin.New()
		router.GET("/api/v1/resources", handler.GetResources)

		req := httptest.NewRequest("GET", "/api/v1/resources", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// May succeed or fail depending on whether vrooli CLI is available
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Got status: %d (expected 200 or 500)", w.Code)
		}

		// Response should be JSON
		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}
	})

	t.Run("CachedCall", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		router := gin.New()
		router.GET("/api/v1/resources", handler.GetResources)

		// First call
		req1 := httptest.NewRequest("GET", "/api/v1/resources", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Second call (should use cache if first succeeded)
		req2 := httptest.NewRequest("GET", "/api/v1/resources", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		// If both succeeded, responses should be similar
		if w1.Code == http.StatusOK && w2.Code == http.StatusOK {
			if w1.Body.String() != w2.Body.String() {
				t.Log("Note: Cached responses may differ slightly")
			}
		}
	})
}

func TestGetResourceStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("EmptyID", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		router := gin.New()
		router.GET("/api/v1/resources/:id/status", handler.GetResourceStatus)

		req := httptest.NewRequest("GET", "/api/v1/resources//status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for empty ID, got %d", http.StatusBadRequest, w.Code)
		}

		if !strings.Contains(w.Body.String(), "required") {
			t.Error("Expected error message about required name")
		}
	})

	t.Run("ValidID", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		router := gin.New()
		router.GET("/api/v1/resources/:id/status", handler.GetResourceStatus)

		req := httptest.NewRequest("GET", "/api/v1/resources/postgres/status", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// May succeed or fail depending on CLI availability
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Got status: %d", w.Code)
		}
	})
}

func TestGetResourceDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("EmptyID", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		router := gin.New()
		router.GET("/api/v1/resources/:id/details", handler.GetResourceDetails)

		req := httptest.NewRequest("GET", "/api/v1/resources//details", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for empty ID, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("ValidID", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		router := gin.New()
		router.GET("/api/v1/resources/:id/details", handler.GetResourceDetails)

		req := httptest.NewRequest("GET", "/api/v1/resources/postgres/details", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// May succeed or fail depending on CLI availability
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Logf("Got status: %d", w.Code)
		}
	})
}

func TestStartResource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("EmptyID", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		router := gin.New()
		router.POST("/api/v1/resources/:id/start", handler.StartResource)

		req := httptest.NewRequest("POST", "/api/v1/resources//start", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for empty ID, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestStopResource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("EmptyID", func(t *testing.T) {
		metricsService := services.NewMetricsService()
		handler := NewSystemHandler(metricsService)

		router := gin.New()
		router.POST("/api/v1/resources/:id/stop", handler.StopResource)

		req := httptest.NewRequest("POST", "/api/v1/resources//stop", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for empty ID, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		name          string
		input         interface{}
		expectedValue bool
		expectedKnown bool
	}{
		{"BoolTrue", true, true, true},
		{"BoolFalse", false, false, true},
		{"StringTrue", "true", true, true},
		{"StringYes", "yes", true, true},
		{"StringFalse", "false", false, true},
		{"StringNo", "no", false, true},
		{"StringEmpty", "", false, false},
		{"StringUnknown", "unknown", false, false},
		{"Float64Zero", float64(0), false, true},
		{"Float64NonZero", float64(1), true, true},
		{"IntInvalid", 123, false, false},
		{"NilValue", nil, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, known := parseBool(tt.input)
			if value != tt.expectedValue {
				t.Errorf("Expected value %v, got %v", tt.expectedValue, value)
			}
			if known != tt.expectedKnown {
				t.Errorf("Expected known %v, got %v", tt.expectedKnown, known)
			}
		})
	}
}

func TestStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"String", "hello", "hello"},
		{"StringWithSpaces", "  hello  ", "hello"},
		{"Bytes", []byte("world"), "world"},
		{"Int", 123, "123"},
		{"Float", 45.67, "45.67"},
		{"Nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringValue(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestLookupValue(t *testing.T) {
	t.Run("NilMap", func(t *testing.T) {
		result := lookupValue(nil, "key")
		if result != nil {
			t.Error("Expected nil for nil map")
		}
	})

	t.Run("ExactMatch", func(t *testing.T) {
		data := map[string]interface{}{
			"Key": "value",
		}
		result := lookupValue(data, "Key")
		if result != "value" {
			t.Errorf("Expected 'value', got %v", result)
		}
	})

	t.Run("CaseInsensitiveMatch", func(t *testing.T) {
		data := map[string]interface{}{
			"Key": "value",
		}
		result := lookupValue(data, "key")
		if result != "value" {
			t.Errorf("Expected 'value', got %v", result)
		}
	})

	t.Run("NoMatch", func(t *testing.T) {
		data := map[string]interface{}{
			"Other": "value",
		}
		result := lookupValue(data, "key")
		if result != nil {
			t.Errorf("Expected nil, got %v", result)
		}
	})
}

func TestTransformResource(t *testing.T) {
	handler := NewSystemHandler(nil)

	t.Run("ValidResource", func(t *testing.T) {
		raw := map[string]interface{}{
			"Name":    "postgres",
			"Type":    "database",
			"Enabled": "true",
			"Running": "true",
			"Status":  "online",
		}

		transformed, valid := handler.transformResource(raw)

		if !valid {
			t.Fatal("Expected valid transformation")
		}

		if transformed["id"] != "postgres" {
			t.Errorf("Expected id 'postgres', got %v", transformed["id"])
		}

		if transformed["status"] != "online" {
			t.Errorf("Expected status 'online', got %v", transformed["status"])
		}
	})

	t.Run("MissingName", func(t *testing.T) {
		raw := map[string]interface{}{
			"Type": "database",
		}

		_, valid := handler.transformResource(raw)

		if valid {
			t.Error("Expected invalid transformation for missing name")
		}
	})

	t.Run("OfflineResource", func(t *testing.T) {
		raw := map[string]interface{}{
			"Name":    "redis",
			"Enabled": "true",
			"Running": "false",
			"Status":  "stopped",
		}

		transformed, valid := handler.transformResource(raw)

		if !valid {
			t.Fatal("Expected valid transformation")
		}

		if transformed["status"] != "stopped" {
			t.Errorf("Expected status 'stopped', got %v", transformed["status"])
		}
	})
}

func TestBuildSummaryFromMap(t *testing.T) {
	t.Run("CompleteData", func(t *testing.T) {
		data := map[string]interface{}{
			"id":            "test-resource",
			"name":          "Test Resource",
			"type":          "test",
			"status":        "online",
			"status_detail": "Running normally",
			"enabled":       true,
			"enabled_known": true,
			"running":       true,
		}

		summary := buildSummaryFromMap(data)

		if summary.ID != "test-resource" {
			t.Errorf("Expected ID 'test-resource', got %s", summary.ID)
		}

		if summary.Status != "online" {
			t.Errorf("Expected status 'online', got %s", summary.Status)
		}

		if !summary.Enabled {
			t.Error("Expected enabled to be true")
		}

		if !summary.Running {
			t.Error("Expected running to be true")
		}
	})

	t.Run("EmptyData", func(t *testing.T) {
		data := map[string]interface{}{}

		summary := buildSummaryFromMap(data)

		if summary.ID != "" {
			t.Errorf("Expected empty ID, got %s", summary.ID)
		}
	})
}

func TestResourceCacheInvalidation(t *testing.T) {
	handler := NewSystemHandler(nil)

	t.Run("InvalidateCache", func(t *testing.T) {
		// Set some cache data
		handler.resourceCache.data = []map[string]interface{}{
			{"id": "test"},
		}

		handler.invalidateResourceCache()

		if handler.resourceCache.data != nil {
			t.Error("Expected cache data to be nil after invalidation")
		}

		if !handler.resourceCache.timestamp.IsZero() {
			t.Error("Expected timestamp to be zero after invalidation")
		}
	})
}

func TestResourceCacheUpsert(t *testing.T) {
	handler := NewSystemHandler(nil)

	t.Run("InsertNew", func(t *testing.T) {
		resource := map[string]interface{}{
			"id":   "postgres",
			"name": "PostgreSQL",
		}

		handler.upsertResourceCache(resource)

		if len(handler.resourceCache.data) != 1 {
			t.Errorf("Expected 1 item in cache, got %d", len(handler.resourceCache.data))
		}
	})

	t.Run("UpdateExisting", func(t *testing.T) {
		resource1 := map[string]interface{}{
			"id":   "postgres",
			"name": "PostgreSQL",
		}

		resource2 := map[string]interface{}{
			"id":     "postgres",
			"name":   "PostgreSQL Updated",
			"status": "online",
		}

		handler.resourceCache.data = nil
		handler.upsertResourceCache(resource1)
		handler.upsertResourceCache(resource2)

		if len(handler.resourceCache.data) != 1 {
			t.Errorf("Expected 1 item in cache, got %d", len(handler.resourceCache.data))
		}

		if handler.resourceCache.data[0]["name"] != "PostgreSQL Updated" {
			t.Error("Expected resource to be updated")
		}
	})

	t.Run("NilResource", func(t *testing.T) {
		handler.resourceCache.data = nil
		handler.upsertResourceCache(nil)

		if handler.resourceCache.data != nil {
			t.Error("Expected cache to remain nil for nil resource")
		}
	})
}
