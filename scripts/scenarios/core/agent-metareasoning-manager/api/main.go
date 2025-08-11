// Alternative Go implementation - Complete API in ~150 lines
// Demonstrates even smaller footprint with single binary deployment

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Config struct {
	Port      string
	N8nBase   string
	WindmillBase string
	WindmillWorkspace string
}

type WorkflowConfig struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Platform       string                 `json:"platform"`
	WebhookPath    string                 `json:"webhook_path,omitempty"`
	JobPath        string                 `json:"job_path,omitempty"`
	Schema         map[string]interface{} `json:"schema"`
}

type ExecutionResponse struct {
	Status      string      `json:"status"`
	ExecutionID string      `json:"executionId"`
	Data        interface{} `json:"data,omitempty"`
	Message     string      `json:"message,omitempty"`
	Timestamp   string      `json:"timestamp"`
}

var (
	config    Config
	workflows map[string]WorkflowConfig
	validTokens = map[string]bool{
		"metareasoning_cli_default_2024": true,
		"metareasoning_admin_2024":       true,
	}
)

func init() {
	// Load configuration from environment variables
	// When run via `manage.sh develop`, these are set by service.json:
	//   - PORT: Dynamically allocated from range 8090-8999 as $SERVICE_PORT
	//   - N8N_BASE_URL: Built using ${RESOURCE_PORTS[n8n]} from port_registry.sh (5678)
	//   - WINDMILL_BASE_URL: Built using ${RESOURCE_PORTS[windmill]} from port_registry.sh (5681)
	// 
	// The hardcoded defaults below are ONLY used as fallbacks when:
	//   - Running the binary directly without manage.sh (development/testing)
	//   - Environment variables fail to be set (error condition)
	// In normal operation via manage.sh, the environment variables are always set.
	config = Config{
		Port:              getEnv("PORT", "8093"),
		N8nBase:          getEnv("N8N_BASE_URL", "http://localhost:5678"),
		WindmillBase:     getEnv("WINDMILL_BASE_URL", "http://localhost:5681"),
		WindmillWorkspace: getEnv("WINDMILL_WORKSPACE", "demo"),
	}

	// Load workflow registry
	file, err := os.Open("workflows.json")
	if err != nil {
		log.Fatal("Failed to open workflows.json:", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&workflows); err != nil {
		log.Fatal("Failed to parse workflows.json:", err)
	}
}

func authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		
		if !validTokens[token] {
			http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Quick health checks
	n8nOk := checkService(config.N8nBase + "/healthz")
	windmillOk := checkService(config.WindmillBase + "/api/version")

	response := map[string]interface{}{
		"status": "healthy",
		"services": map[string]bool{
			"n8n":      n8nOk,
			"windmill": windmillOk,
		},
		"workflows": len(workflows),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func workflowsHandler(w http.ResponseWriter, r *http.Request) {
	workflowList := make([]map[string]interface{}, 0, len(workflows))
	
	for id, workflow := range workflows {
		workflowList = append(workflowList, map[string]interface{}{
			"id":          id,
			"name":        workflow.Name,
			"description": workflow.Description,
			"platform":    workflow.Platform,
			"schema":      workflow.Schema,
		})
	}

	response := map[string]interface{}{
		"workflows": workflowList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	analysisType := vars["type"]
	
	workflow, exists := workflows[analysisType]
	if !exists {
		http.Error(w, fmt.Sprintf(`{"error":"Analysis type '%s' not found"}`, analysisType), http.StatusNotFound)
		return
	}

	executionID := uuid.New().String()

	// Parse request body
	var requestData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	var result interface{}
	var err error

	switch workflow.Platform {
	case "n8n":
		result, err = executeN8nWorkflow(workflow, requestData, executionID)
	case "windmill":
		result, err = executeWindmillWorkflow(workflow, requestData)
	default:
		http.Error(w, `{"error":"Invalid platform"}`, http.StatusBadRequest)
		return
	}

	response := ExecutionResponse{
		ExecutionID: executionID,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if err != nil {
		response.Status = "error"
		response.Message = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response.Status = "success"
		response.Data = result
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func executeN8nWorkflow(workflow WorkflowConfig, data map[string]interface{}, executionID string) (interface{}, error) {
	data["executionId"] = executionID
	
	webhookURL := fmt.Sprintf("%s/webhook/%s", config.N8nBase, workflow.WebhookPath)
	
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func executeWindmillWorkflow(workflow WorkflowConfig, data map[string]interface{}) (interface{}, error) {
	// Start Windmill job
	jobURL := fmt.Sprintf("%s/api/w/%s/jobs/run/f/%s", config.WindmillBase, config.WindmillWorkspace, workflow.JobPath)
	
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(jobURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jobResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&jobResponse); err != nil {
		return nil, err
	}

	jobID := jobResponse["id"].(string)
	return pollWindmillJob(jobID)
}

func pollWindmillJob(jobID string) (interface{}, error) {
	statusURL := fmt.Sprintf("%s/api/w/%s/jobs/%s", config.WindmillBase, config.WindmillWorkspace, jobID)
	
	for i := 0; i < 30; i++ {
		resp, err := http.Get(statusURL)
		if err != nil {
			return nil, err
		}

		var status map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&status)
		resp.Body.Close()

		if status["type"] == "CompletedJob" {
			return status["result"], nil
		} else if status["type"] == "FailedJob" {
			return nil, fmt.Errorf("workflow failed: %v", status["logs"])
		}

		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("workflow timeout")
}

func checkService(url string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(url)
	return err == nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/workflows", workflowsHandler).Methods("GET")
	r.HandleFunc("/analyze/{type}", authenticate(analyzeHandler)).Methods("POST")

	log.Printf("🧠 Metareasoning API (Go) running on port %s", config.Port)
	log.Printf("📊 Loaded %d workflows", len(workflows))
	
	log.Fatal(http.ListenAndServe(":"+config.Port, r))
}