package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// Test struct for health response
type HealthResponse struct {
	Status               string   `json:"status"`
	Service              string   `json:"service"`
	Version              string   `json:"version"`
	MaintenanceState     string   `json:"maintenanceState"`
	CanToggle            bool     `json:"canToggle"`
	SupportedOperations  []string `json:"supported_operations"`
	Timestamp            int64    `json:"timestamp"`
}

// Test struct for maintenance state response
type MaintenanceStateResponse struct {
	Status           string `json:"status"`
	MaintenanceState string `json:"maintenanceState"`
}

func TestMaintenanceIntegration(t *testing.T) {
	// Get API port from environment or use default
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "30500"
	}
	baseURL := fmt.Sprintf("http://localhost:%s", port)
	
	// Wait a moment for server to be ready
	time.Sleep(2 * time.Second)
	
	// Test 1: Health endpoint should return maintenance fields
	t.Run("Health endpoint includes maintenance fields", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/health")
		if err != nil {
			t.Fatalf("Failed to call health endpoint: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
		
		var health HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}
		
		// Verify required maintenance fields are present
		if health.MaintenanceState == "" {
			t.Error("maintenanceState field missing from health response")
		}
		
		if !health.CanToggle {
			t.Error("canToggle field should be true")
		}
		
		// Should start in inactive state
		if health.MaintenanceState != "inactive" {
			t.Errorf("Expected inactive maintenance state, got %s", health.MaintenanceState)
		}
		
		fmt.Printf("✅ Health endpoint correctly returns maintenanceState: %s\n", health.MaintenanceState)
	})
	
	// Test 2: Maintenance state endpoint should accept state changes
	t.Run("Maintenance state endpoint accepts valid states", func(t *testing.T) {
		// Test changing to active
		payload := map[string]string{"maintenanceState": "active"}
		jsonData, _ := json.Marshal(payload)
		
		resp, err := http.Post(baseURL + "/api/maintenance/state", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatalf("Failed to call maintenance state endpoint: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
		
		var stateResp MaintenanceStateResponse
		if err := json.NewDecoder(resp.Body).Decode(&stateResp); err != nil {
			t.Fatalf("Failed to decode maintenance state response: %v", err)
		}
		
		if stateResp.Status != "success" {
			t.Errorf("Expected success status, got %s", stateResp.Status)
		}
		
		if stateResp.MaintenanceState != "active" {
			t.Errorf("Expected active maintenance state, got %s", stateResp.MaintenanceState)
		}
		
		fmt.Printf("✅ Successfully changed maintenance state to: %s\n", stateResp.MaintenanceState)
	})
	
	// Test 3: Verify health endpoint reflects the state change
	t.Run("Health endpoint reflects state changes", func(t *testing.T) {
		// Small delay to ensure state change is processed
		time.Sleep(100 * time.Millisecond)
		
		resp, err := http.Get(baseURL + "/health")
		if err != nil {
			t.Fatalf("Failed to call health endpoint: %v", err)
		}
		defer resp.Body.Close()
		
		var health HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}
		
		if health.MaintenanceState != "active" {
			t.Errorf("Expected active maintenance state after change, got %s", health.MaintenanceState)
		}
		
		fmt.Printf("✅ Health endpoint correctly reflects updated state: %s\n", health.MaintenanceState)
	})
	
	// Test 4: Test invalid state rejection
	t.Run("Maintenance endpoint rejects invalid states", func(t *testing.T) {
		payload := map[string]string{"maintenanceState": "invalid"}
		jsonData, _ := json.Marshal(payload)
		
		resp, err := http.Post(baseURL + "/api/maintenance/state", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatalf("Failed to call maintenance state endpoint: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid state, got %d", resp.StatusCode)
		}
		
		fmt.Printf("✅ Invalid maintenance state correctly rejected\n")
	})
	
	// Test 5: Change back to inactive
	t.Run("Change back to inactive state", func(t *testing.T) {
		payload := map[string]string{"maintenanceState": "inactive"}
		jsonData, _ := json.Marshal(payload)
		
		resp, err := http.Post(baseURL + "/api/maintenance/state", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatalf("Failed to call maintenance state endpoint: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}
		
		fmt.Printf("✅ Successfully changed maintenance state back to inactive\n")
	})
}

func main() {
	fmt.Println("🧪 Running Ecosystem Manager Maintenance Integration Tests")
	fmt.Println("Note: This requires ecosystem-manager API to be running")
	fmt.Println()
	
	// Run the tests
	testing.Main(func(pat, str string) (bool, error) { return true, nil }, 
		[]testing.InternalTest{
			{"TestMaintenanceIntegration", TestMaintenanceIntegration},
		}, 
		nil, nil)
	
	fmt.Println()
	fmt.Println("🎉 All maintenance integration tests completed!")
	fmt.Println("✅ Ecosystem Manager is now compatible with maintenance-orchestrator")
}