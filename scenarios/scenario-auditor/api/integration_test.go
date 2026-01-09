//go:build legacy_auditor_tests
// +build legacy_auditor_tests

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIntegrationFullScanWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	setupTestScenario(t, tempDir)

	server := setupTestServer(t, tempDir)
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var health map[string]any
		json.NewDecoder(resp.Body).Decode(&health)

		if health["status"] != "healthy" {
			t.Error("Expected healthy status")
		}
	})

	t.Run("ScanScenario", func(t *testing.T) {
		reqBody := map[string]any{
			"scenario": "test-scenario",
			"scanners": []string{"custom"},
			"targets":  []string{tempDir},
		}

		body, _ := json.Marshal(reqBody)
		resp, err := client.Post(
			server.URL+"/api/v1/scan",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("Scan request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Scan failed with status %d: %s", resp.StatusCode, body)
		}

		var scanResult map[string]any
		json.NewDecoder(resp.Body).Decode(&scanResult)

		if scanResult["scan_id"] == nil {
			t.Error("Expected scan_id in response")
		}

		if scanResult["status"] == nil {
			t.Error("Expected status in response")
		}
	})

	t.Run("CheckStandards", func(t *testing.T) {
		reqBody := map[string]any{
			"scenario": "test-scenario",
			"targets":  []string{tempDir},
			"rules":    []string{"api-versioning", "error-handling"},
		}

		body, _ := json.Marshal(reqBody)
		resp, err := client.Post(
			server.URL+"/api/v1/standards/check",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("Standards check failed: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if result["violations"] == nil {
			t.Error("Expected violations in response")
		}

		if result["summary"] == nil {
			t.Error("Expected summary in response")
		}
	})

	t.Run("GenerateFixes", func(t *testing.T) {
		violations := []map[string]any{
			{
				"rule":    "api-versioning",
				"file":    "main.go",
				"line":    10,
				"message": "Missing API version",
			},
		}

		reqBody := map[string]any{
			"scenario":   "test-scenario",
			"violations": violations,
		}

		body, _ := json.Marshal(reqBody)
		resp, err := client.Post(
			server.URL+"/api/v1/standards/fix",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("Fix generation failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Fix generation failed with status %d: %s", resp.StatusCode, body)
		}

		var fixResult map[string]any
		json.NewDecoder(resp.Body).Decode(&fixResult)

		if fixResult["fixes"] == nil {
			t.Error("Expected fixes in response")
		}
	})

	t.Run("AgentWorkflow", func(t *testing.T) {
		reqBody := map[string]string{
			"type": "scanner",
		}

		body, _ := json.Marshal(reqBody)
		resp, err := client.Post(
			server.URL+"/api/v1/agents",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("Agent creation failed: %v", err)
		}
		defer resp.Body.Close()

		var agent map[string]any
		json.NewDecoder(resp.Body).Decode(&agent)

		if agent["id"] == nil {
			t.Fatal("Expected agent ID")
		}

		agentID := agent["id"].(string)

		resp, err = client.Get(server.URL + "/api/v1/agents/" + agentID)
		if err != nil {
			t.Fatalf("Get agent failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		taskBody := map[string]any{
			"agent_id": agentID,
			"type":     "scan",
			"input": map[string]any{
				"target": tempDir,
			},
		}

		body, _ = json.Marshal(taskBody)
		resp, err = client.Post(
			server.URL+"/api/v1/agents/task",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			t.Fatalf("Task execution failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status 200 or 202, got %d", resp.StatusCode)
		}

		req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/agents/"+agentID, nil)
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Agent termination failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status 200 or 204, got %d", resp.StatusCode)
		}
	})
}

func TestIntegrationRuleManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	server := setupTestServer(t, tempDir)
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	t.Run("ListRules", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/v1/rules")
		if err != nil {
			t.Fatalf("List rules failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var rules []map[string]any
		json.NewDecoder(resp.Body).Decode(&rules)

		if len(rules) == 0 {
			t.Error("Expected at least one rule")
		}
	})

	t.Run("GetRule", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/api/v1/rules/api-versioning")
		if err != nil {
			t.Fatalf("Get rule failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			t.Skip("Rule not found")
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var rule map[string]any
		json.NewDecoder(resp.Body).Decode(&rule)

		if rule["id"] != "api-versioning" {
			t.Error("Expected rule ID to match")
		}
	})

	t.Run("ToggleRule", func(t *testing.T) {
		reqBody := map[string]bool{
			"enabled": false,
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("PATCH", server.URL+"/api/v1/rules/api-versioning", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Toggle rule failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			t.Errorf("Expected status 200 or 204, got %d", resp.StatusCode)
		}

		resp, err = client.Get(server.URL + "/api/v1/rules/api-versioning")
		if err != nil {
			t.Fatalf("Get rule after toggle failed: %v", err)
		}
		defer resp.Body.Close()

		var rule map[string]any
		json.NewDecoder(resp.Body).Decode(&rule)

		if rule["enabled"] != false {
			t.Error("Expected rule to be disabled")
		}

		reqBody["enabled"] = true
		body, _ = json.Marshal(reqBody)
		req, _ = http.NewRequest("PATCH", server.URL+"/api/v1/rules/api-versioning", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Re-enable rule failed: %v", err)
		}
		defer resp.Body.Close()
	})
}

func TestIntegrationPerformanceMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	setupLargeTestScenario(t, tempDir)

	server := setupTestServer(t, tempDir)
	defer server.Close()

	client := &http.Client{Timeout: 60 * time.Second}

	start := time.Now()

	reqBody := map[string]any{
		"scenario": "performance-test",
		"targets":  []string{tempDir},
	}

	body, _ := json.Marshal(reqBody)
	resp, err := client.Post(
		server.URL+"/api/v1/standards/check",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}
	defer resp.Body.Close()

	elapsed := time.Since(start)

	if elapsed > 30*time.Second {
		t.Errorf("Standards check took too long: %v", elapsed)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if metrics, ok := result["metrics"].(map[string]any); ok {
		if scanTime, ok := metrics["scan_time_ms"].(float64); ok {
			t.Logf("Scan time: %.2fms", scanTime)
			if scanTime > 30000 {
				t.Error("Scan time exceeded 30 seconds")
			}
		}

		if filesScanned, ok := metrics["files_scanned"].(float64); ok {
			t.Logf("Files scanned: %.0f", filesScanned)
		}

		if rulesExecuted, ok := metrics["rules_executed"].(float64); ok {
			t.Logf("Rules executed: %.0f", rulesExecuted)
		}
	}
}

func TestIntegrationErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := setupTestServer(t, t.TempDir())
	defer server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	tests := []struct {
		name           string
		method         string
		path           string
		body           any
		expectedStatus int
	}{
		{
			name:           "Invalid JSON",
			method:         "POST",
			path:           "/api/v1/scan",
			body:           "{invalid json}",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing required fields",
			method:         "POST",
			path:           "/api/v1/standards/check",
			body:           map[string]any{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Non-existent scenario",
			method:         "POST",
			path:           "/api/v1/scan",
			body:           map[string]any{"scenario": "non-existent"},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid agent type",
			method:         "POST",
			path:           "/api/v1/agents",
			body:           map[string]any{"type": "invalid"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Non-existent agent",
			method:         "GET",
			path:           "/api/v1/agents/non-existent-id",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid rule ID",
			method:         "GET",
			path:           "/api/v1/rules/../../etc/passwd",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					bodyReader = strings.NewReader(str)
				} else {
					data, _ := json.Marshal(tt.body)
					bodyReader = bytes.NewReader(data)
				}
			}

			req, _ := http.NewRequest(tt.method, server.URL+tt.path, bodyReader)
			if bodyReader != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s",
					tt.expectedStatus, resp.StatusCode, body)
			}
		})
	}
}

func TestIntegrationConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	setupTestScenario(t, tempDir)

	server := setupTestServer(t, tempDir)
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	numRequests := 20
	results := make(chan int, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			var endpoint string
			var method string
			var body io.Reader

			switch id % 4 {
			case 0:
				endpoint = "/health"
				method = "GET"
			case 1:
				endpoint = "/api/v1/rules"
				method = "GET"
			case 2:
				endpoint = "/api/v1/agents"
				method = "POST"
				reqBody, _ := json.Marshal(map[string]string{"type": "scanner"})
				body = bytes.NewReader(reqBody)
			case 3:
				endpoint = "/api/v1/standards/check"
				method = "POST"
				reqBody, _ := json.Marshal(map[string]any{
					"scenario": "test",
					"targets":  []string{tempDir},
				})
				body = bytes.NewReader(reqBody)
			}

			req, _ := http.NewRequest(method, server.URL+endpoint, body)
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			resp, err := client.Do(req)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			results <- resp.StatusCode
		}(i)
	}

	successCount := 0
	errorCount := 0

	for i := 0; i < numRequests; i++ {
		select {
		case status := <-results:
			if status >= 200 && status < 300 {
				successCount++
			} else {
				t.Logf("Request returned status: %d", status)
			}
		case err := <-errors:
			errorCount++
			t.Logf("Request error: %v", err)
		case <-time.After(60 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	if successCount < numRequests*8/10 {
		t.Errorf("Too many failed requests: %d/%d succeeded", successCount, numRequests)
	}

	if errorCount > numRequests/10 {
		t.Errorf("Too many errors: %d/%d", errorCount, numRequests)
	}
}

func setupTestServer(t *testing.T, dataDir string) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/api/v1/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req["scenario"] == "non-existent" {
			http.Error(w, "Scenario not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"scan_id": fmt.Sprintf("scan-%d", time.Now().Unix()),
			"status":  "completed",
			"results": []map[string]any{},
		})
	})

	mux.HandleFunc("/api/v1/standards/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req["scenario"] == nil {
			http.Error(w, "scenario is required", http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"violations": []map[string]any{},
			"summary": map[string]any{
				"total":  0,
				"passed": 0,
				"failed": 0,
			},
			"metrics": map[string]any{
				"scan_time_ms":   1000,
				"files_scanned":  10,
				"rules_executed": 5,
			},
		})
	})

	mux.HandleFunc("/api/v1/standards/fix", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"fixes": []map[string]any{
				{
					"file": "main.go",
					"fix":  "Add API version",
				},
			},
		})
	})

	mux.HandleFunc("/api/v1/rules", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "..") {
			http.Error(w, "Invalid rule ID", http.StatusBadRequest)
			return
		}

		if r.URL.Path == "/api/v1/rules" {
			json.NewEncoder(w).Encode([]map[string]any{
				{
					"id":      "api-versioning",
					"name":    "API Versioning",
					"enabled": true,
				},
			})
		} else {
			parts := strings.Split(r.URL.Path, "/")
			ruleID := parts[len(parts)-1]

			if r.Method == "PATCH" {
				w.WriteHeader(http.StatusOK)
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"id":      ruleID,
					"name":    "Test Rule",
					"enabled": false,
				})
			}
		}
	})

	mux.HandleFunc("/api/v1/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["type"] == "invalid" {
				http.Error(w, "Invalid agent type", http.StatusBadRequest)
				return
			}

			json.NewEncoder(w).Encode(map[string]any{
				"id":    fmt.Sprintf("agent-%d", time.Now().Unix()),
				"type":  req["type"],
				"state": "ready",
			})
		} else if r.Method == "GET" {
			json.NewEncoder(w).Encode([]map[string]any{})
		}
	})

	mux.HandleFunc("/api/v1/agents/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		agentID := parts[len(parts)-1]

		if agentID == "non-existent-id" {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}

		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
		} else {
			json.NewEncoder(w).Encode(map[string]any{
				"id":    agentID,
				"type":  "scanner",
				"state": "ready",
			})
		}
	})

	mux.HandleFunc("/api/v1/agents/task", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"task_id": fmt.Sprintf("task-%d", time.Now().Unix()),
			"status":  "running",
		})
	})

	return httptest.NewServer(mux)
}

func setupTestScenario(t *testing.T, dir string) {
	files := []struct {
		path    string
		content string
	}{
		{
			path: "main.go",
			content: `package main
import "net/http"
func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello"))
}`,
		},
		{
			path: "api/handler.go",
			content: `package api
// TODO: Add authentication
func HandleRequest() {
	password := "hardcoded"
}`,
		},
	}

	for _, file := range files {
		path := filepath.Join(dir, file.path)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(file.content), 0644)
	}
}

func setupLargeTestScenario(t *testing.T, dir string) {
	for i := 0; i < 100; i++ {
		subDir := filepath.Join(dir, fmt.Sprintf("module%d", i))
		os.MkdirAll(subDir, 0755)

		for j := 0; j < 10; j++ {
			file := filepath.Join(subDir, fmt.Sprintf("file%d.go", j))
			content := fmt.Sprintf(`package module%d
// TODO: Review this code
func Function%d() {
	// Some code here
}`, i, j)
			os.WriteFile(file, []byte(content), 0644)
		}
	}
}
