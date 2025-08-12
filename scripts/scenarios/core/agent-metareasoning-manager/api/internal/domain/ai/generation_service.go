package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"metareasoning-api/internal/domain/common"
	"metareasoning-api/internal/domain/workflow"
	"metareasoning-api/internal/pkg/interfaces"
)

// GenerationService handles AI-powered workflow generation
type GenerationService interface {
	// GenerateWorkflow creates a workflow from natural language prompt
	GenerateWorkflow(request *GenerationRequest) (*workflow.CreateRequest, error)
	
	// ListModels returns available AI models
	ListModels() ([]*ModelInfo, error)
	
	// ValidateModel checks if a model is available
	ValidateModel(modelName string) error
}

// GenerationRequest represents a workflow generation request
type GenerationRequest struct {
	Prompt      string           `json:"prompt" validate:"required,min=10"`
	Platform    common.Platform  `json:"platform" validate:"required"`
	Model       string           `json:"model,omitempty"`
	Temperature float64          `json:"temperature" validate:"min=0,max=1"`
}

// ModelInfo represents information about an AI model
type ModelInfo struct {
	Name       string `json:"name"`
	SizeMB     int    `json:"size_mb"`
	ModifiedAt string `json:"modified_at"`
}

// OllamaGenerationService implements GenerationService using Ollama
type OllamaGenerationService struct {
	baseURL         string
	httpClients     interfaces.HTTPClientFactory
	circuitBreakers interfaces.CircuitBreakerManager
}

// NewOllamaGenerationService creates a new Ollama-based generation service
func NewOllamaGenerationService(baseURL string, httpClients interfaces.HTTPClientFactory, circuitBreakers interfaces.CircuitBreakerManager) GenerationService {
	return &OllamaGenerationService{
		baseURL:         baseURL,
		httpClients:     httpClients,
		circuitBreakers: circuitBreakers,
	}
}

// GenerateWorkflow creates a workflow from natural language prompt
func (s *OllamaGenerationService) GenerateWorkflow(request *GenerationRequest) (*workflow.CreateRequest, error) {
	// Set default model if not specified
	model := request.Model
	if model == "" {
		model = "llama3.2"
	}
	
	// Set default temperature
	temperature := request.Temperature
	if temperature == 0 {
		temperature = 0.7
	}
	
	// Create the generation prompt
	prompt := s.buildGenerationPrompt(request.Prompt, request.Platform)
	
	// Call Ollama to generate the workflow
	response, err := s.callOllama(model, prompt, temperature)
	if err != nil {
		return nil, fmt.Errorf("failed to generate workflow: %w", err)
	}
	
	// Parse the generated workflow
	workflowReq, err := s.parseGeneratedWorkflow(response, request.Platform)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated workflow: %w", err)
	}
	
	return workflowReq, nil
}

// ListModels returns available AI models from Ollama
func (s *OllamaGenerationService) ListModels() ([]*ModelInfo, error) {
	url := s.baseURL + "/api/tags"
	
	var models []*ModelInfo
	breaker := s.circuitBreakers.GetBreaker("ollama")
	
	err := breaker.Call(func() error {
		client := s.httpClients.GetClient(interfaces.AIClient)
		
		resp, err := client.Get(url)
		if err != nil {
			return fmt.Errorf("failed to call ollama: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("ollama returned status %d", resp.StatusCode)
		}
		
		var ollamaResp struct {
			Models []struct {
				Name       string `json:"name"`
				Size       int64  `json:"size"`
				ModifiedAt string `json:"modified_at"`
			} `json:"models"`
		}
		
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			return fmt.Errorf("failed to parse ollama response: %w", err)
		}
		
		// Convert to our model format
		models = make([]*ModelInfo, len(ollamaResp.Models))
		for i, model := range ollamaResp.Models {
			models[i] = &ModelInfo{
				Name:       model.Name,
				SizeMB:     int(model.Size / 1024 / 1024), // Convert bytes to MB
				ModifiedAt: model.ModifiedAt,
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	return models, nil
}

// ValidateModel checks if a model is available
func (s *OllamaGenerationService) ValidateModel(modelName string) error {
	models, err := s.ListModels()
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}
	
	for _, model := range models {
		if model.Name == modelName || strings.HasPrefix(model.Name, modelName+":") {
			return nil
		}
	}
	
	return fmt.Errorf("model %s not available", modelName)
}

// buildGenerationPrompt creates a detailed prompt for workflow generation
func (s *OllamaGenerationService) buildGenerationPrompt(userPrompt string, platform common.Platform) string {
	var platformSpec string
	
	switch platform {
	case common.PlatformN8n:
		platformSpec = `
Generate an n8n workflow with the following JSON structure:
{
  "name": "descriptive name",
  "description": "detailed description",
  "type": "workflow-type",
  "config": {
    "nodes": [
      {"id": "1", "type": "webhook", "name": "Trigger"},
      {"id": "2", "type": "function", "name": "Process"}
    ],
    "connections": {
      "1": {"main": [[{"node": "2", "type": "main", "index": 0}]]}
    }
  },
  "webhook_path": "unique-webhook-path",
  "tags": ["relevant", "tags"]
}`
	case common.PlatformWindmill:
		platformSpec = `
Generate a Windmill workflow with the following JSON structure:
{
  "name": "descriptive name",
  "description": "detailed description", 
  "type": "workflow-type",
  "config": {
    "language": "python",
    "content": "# Python script content here",
    "schema": {"type": "object", "properties": {}}
  },
  "job_path": "unique/job/path",
  "tags": ["relevant", "tags"]
}`
	default:
		platformSpec = "Generate a generic workflow configuration"
	}
	
	return fmt.Sprintf(`You are an expert workflow automation engineer. Create a workflow based on this request:

USER REQUEST: %s

PLATFORM: %s

%s

Requirements:
1. Return ONLY valid JSON, no explanations
2. Make the workflow practical and executable
3. Include appropriate error handling nodes/steps
4. Use descriptive names for all components
5. Estimate execution time in milliseconds
6. Add relevant tags for categorization

Generate the workflow now:`, userPrompt, platform, platformSpec)
}

// callOllama makes a request to Ollama for text generation
func (s *OllamaGenerationService) callOllama(model, prompt string, temperature float64) (string, error) {
	url := s.baseURL + "/api/generate"
	
	reqBody := map[string]interface{}{
		"model":       model,
		"prompt":      prompt,
		"temperature": temperature,
		"stream":      false,
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	var response string
	breaker := s.circuitBreakers.GetBreaker("ollama")
	
	err = breaker.Call(func() error {
		client := s.httpClients.GetClient(interfaces.AIClient)
		
		resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to call ollama: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(respBody))
		}
		
		var ollamaResp struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			return fmt.Errorf("failed to parse ollama response: %w", err)
		}
		
		response = ollamaResp.Response
		return nil
	})
	
	if err != nil {
		return "", err
	}
	
	return response, nil
}

// parseGeneratedWorkflow parses the AI-generated response into a workflow request
func (s *OllamaGenerationService) parseGeneratedWorkflow(response string, platform common.Platform) (*workflow.CreateRequest, error) {
	// Clean up the response (remove markdown, extra text, etc.)
	cleaned := s.cleanGeneratedResponse(response)
	
	// Parse JSON
	var generated map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &generated); err != nil {
		return nil, fmt.Errorf("invalid JSON in generated response: %w", err)
	}
	
	// Extract fields
	name, _ := generated["name"].(string)
	if name == "" {
		name = "AI Generated Workflow"
	}
	
	description, _ := generated["description"].(string)
	if description == "" {
		description = "Generated from AI prompt"
	}
	
	workflowType, _ := generated["type"].(string)
	if workflowType == "" {
		workflowType = "generated"
	}
	
	// Marshal config
	config := generated["config"]
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("invalid config in generated workflow: %w", err)
	}
	
	// Extract optional fields
	var webhookPath, jobPath *string
	if wp, ok := generated["webhook_path"].(string); ok && wp != "" {
		webhookPath = &wp
	}
	if jp, ok := generated["job_path"].(string); ok && jp != "" {
		jobPath = &jp
	}
	
	// Extract tags
	var tags []string
	if tagList, ok := generated["tags"].([]interface{}); ok {
		for _, tag := range tagList {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}
	tags = append(tags, "ai-generated")
	
	// Estimate duration (default to 2 seconds)
	estimatedDuration := 2000
	if duration, ok := generated["estimated_duration"].(float64); ok {
		estimatedDuration = int(duration)
	}
	
	return &workflow.CreateRequest{
		Name:              name,
		Description:       description,
		Type:              workflowType,
		Platform:          platform,
		Config:            configJSON,
		WebhookPath:       webhookPath,
		JobPath:           jobPath,
		EstimatedDuration: estimatedDuration,
		Tags:              tags,
	}, nil
}

// cleanGeneratedResponse cleans up AI-generated response
func (s *OllamaGenerationService) cleanGeneratedResponse(response string) string {
	// Remove markdown code blocks
	cleaned := response
	cleaned = strings.ReplaceAll(cleaned, "```json", "")
	cleaned = strings.ReplaceAll(cleaned, "```", "")
	
	// Find JSON content (between first { and last })
	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	
	if start != -1 && end != -1 && end > start {
		cleaned = cleaned[start : end+1]
	}
	
	return strings.TrimSpace(cleaned)
}