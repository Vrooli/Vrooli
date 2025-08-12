package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Utility functions

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// checkService checks if a service is available at the given URL
func checkService(url string) bool {
	client := GetHTTPClient(HealthCheckClient)
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// checkServiceErr checks if a service is available and returns an error if not
func checkServiceErr(url string) error {
	client := GetHTTPClient(HealthCheckClient)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("service unavailable: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service returned status %d", resp.StatusCode)
	}
	
	return nil
}

// parseQueryInt parses an integer from query parameters with a default value
func parseQueryInt(r *http.Request, param string, defaultValue int) int {
	valueStr := r.URL.Query().Get(param)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// parseQueryBool parses a boolean from query parameters with a default value
func parseQueryBool(r *http.Request, param string, defaultValue bool) bool {
	valueStr := r.URL.Query().Get(param)
	if valueStr == "" {
		return defaultValue
	}
	
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	
	return value
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// HTTPClientType represents different types of HTTP clients for different use cases
type HTTPClientType int

const (
	HealthCheckClient HTTPClientType = iota  // Quick health checks
	WorkflowClient                           // n8n/Windmill workflow execution  
	AIClient                                 // Ollama AI generation calls
)

// httpClientFactory manages HTTP clients with optimized configurations
var httpClientFactory = struct {
	healthClient   *http.Client
	workflowClient *http.Client
	aiClient       *http.Client
}{
	// Health check client: Short timeout for quick checks
	healthClient: &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     30 * time.Second,
		},
	},
	// Workflow client: Medium timeout for workflow execution
	workflowClient: &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        50,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	},
	// AI client: Long timeout for AI operations
	aiClient: &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     120 * time.Second,
		},
	},
}

// GetHTTPClient returns an optimized HTTP client for the specified use case
func GetHTTPClient(clientType HTTPClientType) *http.Client {
	switch clientType {
	case HealthCheckClient:
		return httpClientFactory.healthClient
	case WorkflowClient:
		return httpClientFactory.workflowClient
	case AIClient:
		return httpClientFactory.aiClient
	default:
		return httpClientFactory.workflowClient // Default fallback
	}
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Port:              getEnv("PORT", "8093"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://vrooli:vrooli@localhost:5432/metareasoning?sslmode=disable"),
		N8nBase:          getEnv("N8N_BASE_URL", "http://localhost:5678"),
		WindmillBase:     getEnv("WINDMILL_BASE_URL", "http://localhost:8000"),
		WindmillWorkspace: getEnv("WINDMILL_WORKSPACE", "demo"),
		OllamaBase:       getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
	}
}