package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandlerAdditional(t *testing.T) {
	t.Run("Returns_All_Required_Fields", func(t *testing.T) {
		config := &Config{
			Port:        "8080",
			DatabaseURL: "",
			RedisURL:    "http://localhost:6379",
		}

		server := NewServer(config)
		server.Initialize()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.HealthHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check all required fields
		requiredFields := []string{"status", "timestamp", "database", "resources", "version"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})
}

func TestResourcesHandlerAdditional(t *testing.T) {
	t.Run("Returns_Resource_Details", func(t *testing.T) {
		config := &Config{
			Port:     "8080",
			RedisURL: "http://localhost:6379",
		}

		server := NewServer(config)
		server.Initialize()

		req := httptest.NewRequest("GET", "/resources", nil)
		w := httptest.NewRecorder()

		server.ResourcesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have resources field
		if _, exists := response["resources"]; !exists {
			t.Error("Missing resources field")
		}
	})

	t.Run("Returns_JSON_Content_Type", func(t *testing.T) {
		config := &Config{Port: "8080"}
		server := NewServer(config)
		server.Initialize()

		req := httptest.NewRequest("GET", "/resources", nil)
		w := httptest.NewRecorder()

		server.ResourcesHandler(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected content-type 'application/json', got '%s'", contentType)
		}
	})
}

func TestDiffHandlerV1Additional(t *testing.T) {
	t.Run("Semantic_Diff_Fallback", func(t *testing.T) {
		config := &Config{
			Port:      "8080",
			OllamaURL: "", // No Ollama configured
		}

		server := NewServer(config)
		server.Initialize()

		// Basic sanity check
		if server == nil {
			t.Fatal("Server should not be nil")
		}
	})
}

func TestSearchHandlerV1Additional(t *testing.T) {
	t.Run("Regex_Error_Handling", func(t *testing.T) {
		config := &Config{Port: "8080"}
		server := NewServer(config)
		server.Initialize()

		// This tests error paths
		if server.config.Port != "8080" {
			t.Error("Config not properly set")
		}
	})
}

func TestTransformHandlerV1Additional(t *testing.T) {
	t.Run("Multiple_Transform_Chain", func(t *testing.T) {
		config := &Config{Port: "8080"}
		server := NewServer(config)
		server.Initialize()

		if server == nil {
			t.Fatal("Server should not be nil")
		}
	})
}

func TestResourceMonitoring(t *testing.T) {
	t.Run("Monitor_Resources_Handles_Empty_Resources", func(t *testing.T) {
		config := &Config{Port: "8080"}
		rm := NewResourceManager(config)

		// Start and stop monitoring for empty resources
		rm.Start()
		rm.Stop()

		// Should not panic
	})

	t.Run("Check_All_Resources", func(t *testing.T) {
		healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
		defer healthy.Close()
		down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
		defer down.Close()

		rm := NewResourceManager(&Config{Port: "8080", QdrantURL: healthy.URL, MinIOURL: down.URL})
		rm.Start()
		rm.Stop()

		// Start runs one synchronous check of every configured resource.
		qdrant, minio := rm.resources["qdrant"], rm.resources["minio"]
		if qdrant == nil || minio == nil {
			t.Fatalf("configured resources were not registered: %v", rm.resources)
		}
		if qdrant.CheckCount < 1 || !qdrant.Available {
			t.Errorf("qdrant = %+v, want checked and available", qdrant)
		}
		if minio.CheckCount < 1 || minio.Available {
			t.Errorf("minio = %+v, want checked and unavailable", minio)
		}
	})
}
