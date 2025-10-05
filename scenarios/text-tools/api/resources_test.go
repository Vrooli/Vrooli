package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewResourceManager(t *testing.T) {
	t.Run("Creates_Manager_With_All_Resources", func(t *testing.T) {
		config := &Config{
			MinIOURL:  "http://minio:9000",
			RedisURL:  "http://redis:6379",
			OllamaURL: "http://ollama:11434",
			QdrantURL: "http://qdrant:6333",
		}

		rm := NewResourceManager(config)

		if len(rm.resources) != 4 {
			t.Errorf("Expected 4 resources, got %d", len(rm.resources))
		}

		if rm.resources["minio"] == nil {
			t.Error("Expected minio resource to be initialized")
		}
		if rm.resources["redis"] == nil {
			t.Error("Expected redis resource to be initialized")
		}
		if rm.resources["ollama"] == nil {
			t.Error("Expected ollama resource to be initialized")
		}
		if rm.resources["qdrant"] == nil {
			t.Error("Expected qdrant resource to be initialized")
		}
	})

	t.Run("Creates_Manager_With_Partial_Resources", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://ollama:11434",
			RedisURL:  "http://redis:6379",
		}

		rm := NewResourceManager(config)

		if len(rm.resources) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(rm.resources))
		}

		if rm.resources["ollama"] == nil {
			t.Error("Expected ollama resource to be initialized")
		}
		if rm.resources["redis"] == nil {
			t.Error("Expected redis resource to be initialized")
		}
		if rm.resources["minio"] != nil {
			t.Error("Expected minio to not be initialized")
		}
	})

	t.Run("Creates_Manager_With_No_Resources", func(t *testing.T) {
		config := &Config{}

		rm := NewResourceManager(config)

		if len(rm.resources) != 0 {
			t.Errorf("Expected 0 resources, got %d", len(rm.resources))
		}
	})
}

func TestResourceManagerStart(t *testing.T) {
	t.Run("Starts_Monitoring", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.Start()

		rm.mu.RLock()
		isMonitoring := rm.isMonitoring
		rm.mu.RUnlock()

		if !isMonitoring {
			t.Error("Expected monitoring to be started")
		}

		rm.Stop()
	})

	t.Run("Does_Not_Start_Multiple_Times", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.Start()
		rm.Start() // Should not panic or create duplicate monitors

		rm.mu.RLock()
		isMonitoring := rm.isMonitoring
		rm.mu.RUnlock()

		if !isMonitoring {
			t.Error("Expected monitoring to still be active")
		}

		rm.Stop()
	})
}

func TestResourceManagerStop(t *testing.T) {
	t.Run("Stops_Monitoring", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.Start()

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		rm.Stop()

		rm.mu.RLock()
		isMonitoring := rm.isMonitoring
		rm.mu.RUnlock()

		if isMonitoring {
			t.Error("Expected monitoring to be stopped")
		}
	})

	t.Run("Can_Stop_Without_Starting", func(t *testing.T) {
		config := &Config{}
		rm := NewResourceManager(config)

		// Should not panic
		rm.Stop()

		rm.mu.RLock()
		isMonitoring := rm.isMonitoring
		rm.mu.RUnlock()

		if isMonitoring {
			t.Error("Expected monitoring to not be active")
		}
	})
}

func TestResourceManagerGetStatus(t *testing.T) {
	t.Run("Returns_Resource_Status", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
			RedisURL:  "http://localhost:6379",
		}

		rm := NewResourceManager(config)

		statuses := rm.GetResourceStatus()

		if len(statuses) != 2 {
			t.Errorf("Expected 2 statuses, got %d", len(statuses))
		}

		// Check that statuses are returned
		if statuses["ollama"] == nil {
			t.Error("Expected to find ollama status")
		}
		if statuses["redis"] == nil {
			t.Error("Expected to find redis status")
		}

		if statuses["ollama"] != nil && statuses["ollama"].Name != "ollama" {
			t.Errorf("Expected ollama name, got %s", statuses["ollama"].Name)
		}
	})

	t.Run("Returns_Empty_For_No_Resources", func(t *testing.T) {
		config := &Config{}
		rm := NewResourceManager(config)

		statuses := rm.GetResourceStatus()

		if len(statuses) != 0 {
			t.Errorf("Expected 0 statuses, got %d", len(statuses))
		}
	})
}

func TestResourceManagerIsAvailable(t *testing.T) {
	t.Run("Returns_True_For_Available_Resource", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.resources["ollama"].Available = true

		if !rm.IsResourceAvailable("ollama") {
			t.Error("Expected ollama to be available")
		}
	})

	t.Run("Returns_False_For_Unavailable_Resource", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.resources["ollama"].Available = false

		if rm.IsResourceAvailable("ollama") {
			t.Error("Expected ollama to be unavailable")
		}
	})

	t.Run("Returns_False_For_Unknown_Resource", func(t *testing.T) {
		config := &Config{}
		rm := NewResourceManager(config)

		if rm.IsResourceAvailable("unknown") {
			t.Error("Expected unknown resource to be unavailable")
		}
	})
}

func TestResourceManagerGetHealth(t *testing.T) {
	t.Run("Returns_Health_For_Available_Resource", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.resources["ollama"].Available = true
		rm.resources["ollama"].LastCheck = time.Now()
		rm.resources["ollama"].CheckCount = 10
		rm.resources["ollama"].SuccessCount = 8

		health := rm.GetResourceHealth()

		if health == nil {
			t.Fatal("Expected health to not be nil")
		}

		if health["ollama"] == nil {
			t.Error("Expected ollama in health map")
		}
		if health["ollama"] != "connected" {
			t.Errorf("Expected 'connected', got %v", health["ollama"])
		}
	})

	t.Run("Returns_Detailed_Info_For_Unavailable_Resource", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.resources["ollama"].Available = false
		rm.resources["ollama"].LastError = "connection refused"
		rm.resources["ollama"].LastCheck = time.Now()

		health := rm.GetResourceHealth()

		if health == nil {
			t.Fatal("Expected health to not be nil")
		}

		ollamaHealth, ok := health["ollama"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected ollama health to be a map")
		}

		if ollamaHealth["status"] != "disconnected" {
			t.Error("Expected status to be disconnected")
		}
	})

	t.Run("Returns_Empty_For_No_Resources", func(t *testing.T) {
		config := &Config{}
		rm := NewResourceManager(config)

		health := rm.GetResourceHealth()

		if len(health) != 0 {
			t.Error("Expected empty health map for no resources")
		}
	})
}

func TestCheckResource(t *testing.T) {
	t.Run("Marks_Resource_Available_On_Success", func(t *testing.T) {
		// Create a test server that responds successfully
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := &Config{
			OllamaURL: server.URL,
		}

		rm := NewResourceManager(config)
		resource := rm.resources["ollama"]
		resource.URL = server.URL

		rm.checkResource(resource)

		if !rm.resources["ollama"].Available {
			t.Error("Expected resource to be available after successful check")
		}
		if rm.resources["ollama"].LastError != "" {
			t.Error("Expected no error after successful check")
		}
		if rm.resources["ollama"].SuccessCount != 1 {
			t.Error("Expected success count to increment")
		}
	})

	t.Run("Marks_Resource_Unavailable_On_Failure", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:99999", // Invalid port
		}

		rm := NewResourceManager(config)
		resource := rm.resources["ollama"]
		resource.URL = "http://localhost:99999/invalid"

		rm.checkResource(resource)

		if rm.resources["ollama"].Available {
			t.Error("Expected resource to be unavailable after failed check")
		}
		if rm.resources["ollama"].LastError == "" {
			t.Error("Expected error to be set after failed check")
		}
	})
}

func TestWaitForResource(t *testing.T) {
	t.Run("Returns_True_If_Available", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.resources["ollama"].Available = true

		result := rm.WaitForResource("ollama", 1*time.Second)

		if !result {
			t.Error("Expected true when resource is available")
		}
	})

	t.Run("Returns_False_If_Not_Available", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.resources["ollama"].Available = false

		result := rm.WaitForResource("ollama", 100*time.Millisecond)

		if result {
			t.Error("Expected false when resource times out")
		}
	})

	t.Run("Returns_False_For_Unknown_Resource", func(t *testing.T) {
		config := &Config{}
		rm := NewResourceManager(config)

		result := rm.WaitForResource("unknown", 100*time.Millisecond)

		if result {
			t.Error("Expected false for unknown resource")
		}
	})
}

func TestTryWithResource(t *testing.T) {
	t.Run("Executes_Function_If_Available", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.resources["ollama"].Available = true

		executed := false
		err := rm.TryWithResource("ollama", func() error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !executed {
			t.Error("Expected function to be executed")
		}
	})

	t.Run("Returns_Error_If_Unavailable", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		rm.resources["ollama"].Available = false

		executed := false
		err := rm.TryWithResource("ollama", func() error {
			executed = true
			return nil
		})

		if err == nil {
			t.Error("Expected error for unavailable resource")
		}
		if executed {
			t.Error("Expected function not to be executed")
		}
	})
}

func TestResourceUnavailableError(t *testing.T) {
	t.Run("Error_Message", func(t *testing.T) {
		err := ResourceUnavailableError{Resource: "ollama"}
		expected := "resource ollama is not available"

		if err.Error() != expected {
			t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
		}
	})
}

func TestHandleResourceAvailable(t *testing.T) {
	t.Run("Updates_Status", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
		}

		rm := NewResourceManager(config)
		resource := rm.resources["ollama"]
		resource.Available = false

		// Simulate making resource available
		rm.updateResourceStatus(resource, true, "")

		if !rm.resources["ollama"].Available {
			t.Error("Expected resource to be available")
		}
	})
}

func TestGetResourceMetrics(t *testing.T) {
	t.Run("Returns_Metrics", func(t *testing.T) {
		config := &Config{
			OllamaURL: "http://localhost:11434",
			RedisURL:  "http://localhost:6379",
		}

		rm := NewResourceManager(config)
		rm.resources["ollama"].Available = true
		rm.resources["ollama"].CheckCount = 10
		rm.resources["ollama"].SuccessCount = 8
		rm.resources["redis"].Available = false
		rm.resources["redis"].CheckCount = 10
		rm.resources["redis"].SuccessCount = 2

		metrics := rm.GetResourceMetrics()

		if len(metrics) != 2 {
			t.Errorf("Expected 2 resource metrics, got %d", len(metrics))
		}

		ollamaMetrics, ok := metrics["ollama"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected ollama metrics to be a map")
		}

		if ollamaMetrics["total_checks"] != 10 {
			t.Errorf("Expected total_checks 10, got %v", ollamaMetrics["total_checks"])
		}
		if ollamaMetrics["successful_checks"] != 8 {
			t.Errorf("Expected successful_checks 8, got %v", ollamaMetrics["successful_checks"])
		}
		if ollamaMetrics["currently_available"] != true {
			t.Error("Expected currently_available to be true")
		}
	})

	t.Run("Returns_Empty_For_No_Resources", func(t *testing.T) {
		config := &Config{}
		rm := NewResourceManager(config)

		metrics := rm.GetResourceMetrics()

		if len(metrics) != 0 {
			t.Errorf("Expected 0 metrics, got %d", len(metrics))
		}
	})
}
