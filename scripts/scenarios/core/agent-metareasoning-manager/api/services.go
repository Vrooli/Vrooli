package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// WorkflowService handles business logic for workflows
type WorkflowService struct {
	config *Config
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(cfg *Config) *WorkflowService {
	return &WorkflowService{config: cfg}
}

// ExecuteWorkflow executes a workflow through n8n or Windmill
func (s *WorkflowService) ExecuteWorkflow(workflow *Workflow, req ExecutionRequest) (*ExecutionResponse, error) {
	startTime := time.Now()
	
	// Set default model if not specified
	if req.Model == "" {
		req.Model = "llama3.2"
	}
	
	var result interface{}
	var err error
	
	// Execute based on platform
	switch workflow.Platform {
	case "n8n":
		result, err = s.executeN8nWorkflow(workflow, req)
	case "windmill":
		result, err = s.executeWindmillWorkflow(workflow, req)
	default:
		err = fmt.Errorf("unsupported platform: %s", workflow.Platform)
	}
	
	executionTime := int(time.Since(startTime).Milliseconds())
	
	// Prepare response
	resp := &ExecutionResponse{
		ID:          uuid.New(),
		WorkflowID:  workflow.ID,
		ExecutionMS: executionTime,
		Timestamp:   time.Now(),
	}
	
	if err != nil {
		resp.Status = "error"
		resp.Error = err.Error()
	} else {
		resp.Status = "success"
		resp.Data = result
	}
	
	// Record execution
	recordExecution(workflow.ID, req, resp)
	
	return resp, nil
}

func (s *WorkflowService) executeN8nWorkflow(workflow *Workflow, req ExecutionRequest) (interface{}, error) {
	if workflow.WebhookPath == nil {
		return nil, fmt.Errorf("workflow missing webhook path")
	}
	
	url := fmt.Sprintf("%s/webhook/%s", s.config.N8nBase, *workflow.WebhookPath)
	
	// Prepare request body
	body := map[string]interface{}{
		"input":   req.Input,
		"context": req.Context,
		"model":   req.Model,
	}
	
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	
	// Call n8n with circuit breaker protection
	var result interface{}
	
	breaker := circuitBreakerManager.GetBreaker("n8n")
	err = breaker.Call(func() error {
		client := GetHTTPClient(WorkflowClient)
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to call n8n: %w", err)
		}
		defer resp.Body.Close()
		
		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("n8n returned status %d: %s", resp.StatusCode, string(respBody))
		}
		
		// Parse response
		return json.Unmarshal(respBody, &result)
	})
	
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

func (s *WorkflowService) executeWindmillWorkflow(workflow *Workflow, req ExecutionRequest) (interface{}, error) {
	if workflow.JobPath == nil {
		return nil, fmt.Errorf("workflow missing job path")
	}
	
	url := fmt.Sprintf("%s/api/w/%s/jobs/run/p/%s",
		s.config.WindmillBase, s.config.WindmillWorkspace, *workflow.JobPath)
	
	// Prepare request body
	body := map[string]interface{}{
		"args": map[string]interface{}{
			"input":   req.Input,
			"context": req.Context,
			"model":   req.Model,
		},
	}
	
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	
	// Call Windmill with circuit breaker protection
	var result interface{}
	
	breaker := circuitBreakerManager.GetBreaker("windmill")
	err = breaker.Call(func() error {
		httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		
		client := GetHTTPClient(WorkflowClient)
		resp, err := client.Do(httpReq)
		if err != nil {
			return fmt.Errorf("failed to call windmill: %w", err)
		}
		defer resp.Body.Close()
		
		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("windmill returned status %d: %s", resp.StatusCode, string(respBody))
		}
		
		// Parse response
		return json.Unmarshal(respBody, &result)
	})
	
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

// GenerateWorkflow generates a workflow from a natural language prompt using Ollama
func (s *WorkflowService) GenerateWorkflow(prompt, platform, model string, temperature float64) (*WorkflowCreate, error) {
	if model == "" {
		model = "llama3.2"
	}
	if temperature == 0 {
		temperature = 0.7
	}
	
	// Create prompt for Ollama
	systemPrompt := fmt.Sprintf(`You are a workflow designer. Create a %s workflow definition based on the user's request.
Return ONLY valid JSON in this exact format:
{
  "name": "workflow name",
  "description": "workflow description",
  "type": "workflow-type",
  "webhook_path": "webhook-path-name",
  "tags": ["tag1", "tag2"],
  "estimated_duration_ms": 5000
}`, platform)
	
	fullPrompt := fmt.Sprintf("%s\n\nUser request: %s", systemPrompt, prompt)
	
	// Call Ollama
	ollamaReq := OllamaGenerateRequest{
		Model:       model,
		Prompt:      fullPrompt,
		Temperature: temperature,
		Stream:      false,
	}
	
	jsonBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}
	
	// Call Ollama with circuit breaker protection
	ollamaURL := fmt.Sprintf("%s/api/generate", s.config.OllamaBase)
	var ollamaResp OllamaGenerateResponse
	
	breaker := circuitBreakerManager.GetBreaker("ollama")
	err = breaker.Call(func() error {
		client := GetHTTPClient(AIClient)
		resp, err := client.Post(ollamaURL, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to call Ollama: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
		}
		
		return json.NewDecoder(resp.Body).Decode(&ollamaResp)
	})
	
	if err != nil {
		return nil, err
	}
	
	// Parse the generated workflow
	var workflow WorkflowCreate
	if err := json.Unmarshal([]byte(ollamaResp.Response), &workflow); err != nil {
		// Try to extract JSON from response if it contains extra text
		start := strings.Index(ollamaResp.Response, "{")
		end := strings.LastIndex(ollamaResp.Response, "}")
		if start >= 0 && end > start {
			jsonStr := ollamaResp.Response[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &workflow); err != nil {
				return nil, fmt.Errorf("failed to parse generated workflow: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse generated workflow: %w", err)
		}
	}
	
	// Set platform and default values
	workflow.Platform = platform
	if workflow.EstimatedDuration == 0 {
		workflow.EstimatedDuration = 5000
	}
	
	// Generate minimal config and schema
	workflow.Config = json.RawMessage(`{"generated": true}`)
	workflow.Schema = json.RawMessage(`{"input": {"required": ["input"]}, "output": {"type": "object"}}`)
	
	return &workflow, nil
}

// ImportWorkflow imports a workflow from platform-specific format
func (s *WorkflowService) ImportWorkflow(platform string, data json.RawMessage, name string) (*Workflow, error) {
	var workflow WorkflowCreate
	
	switch platform {
	case "n8n":
		workflow = s.parseN8nWorkflow(data, name)
	case "windmill":
		workflow = s.parseWindmillWorkflow(data, name)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	
	return createWorkflow(workflow, "import")
}

func (s *WorkflowService) parseN8nWorkflow(data json.RawMessage, name string) WorkflowCreate {
	// Parse n8n workflow format
	var n8nData map[string]interface{}
	json.Unmarshal(data, &n8nData)
	
	if name == "" {
		if n, ok := n8nData["name"].(string); ok {
			name = n
		} else {
			name = "Imported n8n Workflow"
		}
	}
	
	webhookPath := fmt.Sprintf("imported-%s", strings.ReplaceAll(strings.ToLower(name), " ", "-"))
	
	return WorkflowCreate{
		Name:              name,
		Description:       "Imported from n8n",
		Type:              "imported",
		Platform:          "n8n",
		Config:            data,
		WebhookPath:       &webhookPath,
		Schema:            json.RawMessage(`{"input": {}, "output": {}}`),
		EstimatedDuration: 5000,
		Tags:              []string{"imported", "n8n"},
	}
}

func (s *WorkflowService) parseWindmillWorkflow(data json.RawMessage, name string) WorkflowCreate {
	// Parse Windmill workflow format
	var wmData map[string]interface{}
	json.Unmarshal(data, &wmData)
	
	if name == "" {
		if n, ok := wmData["summary"].(string); ok {
			name = n
		} else {
			name = "Imported Windmill Workflow"
		}
	}
	
	jobPath := fmt.Sprintf("imported/%s", strings.ReplaceAll(strings.ToLower(name), " ", "_"))
	
	return WorkflowCreate{
		Name:              name,
		Description:       "Imported from Windmill",
		Type:              "imported",
		Platform:          "windmill",
		Config:            data,
		JobPath:           &jobPath,
		Schema:            json.RawMessage(`{"input": {}, "output": {}}`),
		EstimatedDuration: 5000,
		Tags:              []string{"imported", "windmill"},
	}
}

// ExportWorkflow exports a workflow to platform-specific format
func (s *WorkflowService) ExportWorkflow(workflow *Workflow, format string) (interface{}, error) {
	switch format {
	case "n8n":
		return s.exportToN8n(workflow), nil
	case "windmill":
		return s.exportToWindmill(workflow), nil
	case "json":
		return workflow, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *WorkflowService) exportToN8n(w *Workflow) interface{} {
	// Create n8n-compatible format
	return map[string]interface{}{
		"name":        w.Name,
		"nodes":       []interface{}{},
		"connections": map[string]interface{}{},
		"settings":    map[string]interface{}{},
		"staticData":  nil,
		"tags":        w.Tags,
		"createdAt":   w.CreatedAt,
		"updatedAt":   w.UpdatedAt,
	}
}

func (s *WorkflowService) exportToWindmill(w *Workflow) interface{} {
	// Create Windmill-compatible format
	return map[string]interface{}{
		"summary":     w.Name,
		"description": w.Description,
		"kind":        "script",
		"language":    "typescript",
		"content":     "// Exported workflow\n",
		"schema":      w.Schema,
		"is_template": false,
		"lock":        "",
	}
}

// GetAvailableModels fetches available models from Ollama
func (s *WorkflowService) GetAvailableModels() ([]ModelInfo, error) {
	ollamaURL := fmt.Sprintf("%s/api/tags", s.config.OllamaBase)
	var ollamaResp OllamaListResponse
	
	breaker := circuitBreakerManager.GetBreaker("ollama")
	err := breaker.Call(func() error {
		client := GetHTTPClient(AIClient)
		resp, err := client.Get(ollamaURL)
		if err != nil {
			return fmt.Errorf("failed to fetch models from Ollama: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
		}
		
		return json.NewDecoder(resp.Body).Decode(&ollamaResp)
	})
	
	if err != nil {
		return nil, err
	}
	
	models := make([]ModelInfo, len(ollamaResp.Models))
	for i, m := range ollamaResp.Models {
		// Parse modified time
		modTime, _ := time.Parse(time.RFC3339, m.ModifiedAt)
		
		models[i] = ModelInfo{
			Name:       m.Name,
			SizeMB:     int(m.Size / 1024 / 1024),
			ModifiedAt: modTime,
		}
	}
	
	return models, nil
}

// CheckPlatformStatus checks if a platform is available
func (s *WorkflowService) CheckPlatformStatus() []PlatformInfo {
	platforms := []PlatformInfo{
		{
			Name:        "n8n",
			Description: "Visual workflow automation",
			URL:         s.config.N8nBase,
			Status:      s.checkServiceWithCircuitBreaker("n8n", s.config.N8nBase+"/healthz"),
		},
		{
			Name:        "windmill", 
			Description: "Code-based workflows with UI",
			URL:         s.config.WindmillBase,
			Status:      s.checkServiceWithCircuitBreaker("windmill", s.config.WindmillBase+"/api/version"),
		},
		{
			Name:        "ollama",
			Description: "Local AI models",
			URL:         s.config.OllamaBase,
			Status:      s.checkServiceWithCircuitBreaker("ollama", s.config.OllamaBase+"/api/tags"),
		},
	}
	
	return platforms
}

// checkServiceWithCircuitBreaker checks service health with circuit breaker protection
func (s *WorkflowService) checkServiceWithCircuitBreaker(serviceName, url string) bool {
	breaker := circuitBreakerManager.GetBreaker(serviceName)
	
	err := breaker.Call(func() error {
		return checkServiceErr(url)
	})
	
	return err == nil
}